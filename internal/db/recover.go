package db

import (
	"context"
	"fmt"
	"time"
)

// RecoverSearchIndex rebuilds FTS when fts_dirty is set or works_fts triggers are missing.
func (d *DB) RecoverSearchIndex(ctx context.Context) error {
	if d == nil {
		return nil
	}
	write := d.Write
	if write == nil {
		return nil
	}
	var hasWorks int
	if err := write.QueryRowContext(ctx, `SELECT count(*) FROM sqlite_master WHERE type='table' AND name='works'`).Scan(&hasWorks); err != nil {
		return err
	}
	if hasWorks == 0 {
		return nil
	}
	var works int
	if err := write.QueryRowContext(ctx, `SELECT EXISTS (SELECT 1 FROM works)`).Scan(&works); err != nil {
		return err
	}
	dirty, err := ftsDirty(ctx, write)
	if err != nil {
		return err
	}
	nTrig, err := worksFTSTriggerCount(ctx, write)
	if err != nil {
		return err
	}
	var genres int
	if err := write.QueryRowContext(ctx, `SELECT EXISTS (SELECT 1 FROM work_genres)`).Scan(&genres); err != nil {
		return err
	}
	needFTS := dirty || nTrig != 2
	needWarm := works > 0 && (genres == 0 || needFTS)
	if works == 0 {
		if nTrig != 2 {
			if err := EnsureWorksFTSTriggers(ctx, write); err != nil {
				return err
			}
		}
		if dirty {
			return SetFTSDirty(ctx, write, false)
		}
		return nil
	}
	if !needFTS && !needWarm {
		return nil
	}
	if err := Analyze(ctx, write); err != nil {
		return err
	}
	if needFTS {
		conn, err := write.Conn(ctx)
		if err != nil {
			return err
		}
		err = RebuildWorksFTS(ctx, conn)
		_ = conn.Close()
		if err != nil {
			return err
		}
	}
	if needWarm {
		if err := WarmUpCatalog(ctx, write); err != nil {
			return err
		}
		if err := Optimize(ctx, write); err != nil {
			return err
		}
		_ = WarmCache(ctx, write)
	}
	return SetFTSDirty(ctx, write, false)
}

func (d *DB) indexRecoveryNeeded(ctx context.Context) (bool, error) {
	var hasWorks int
	if err := d.Write.QueryRowContext(ctx, `SELECT count(*) FROM sqlite_master WHERE type='table' AND name='works'`).Scan(&hasWorks); err != nil {
		return false, err
	}
	if hasWorks == 0 {
		return false, nil
	}
	dirty, err := ftsDirty(ctx, d.Write)
	if err != nil {
		return false, err
	}
	nTrig, err := worksFTSTriggerCount(ctx, d.Write)
	if err != nil {
		return false, err
	}
	if dirty || nTrig != 2 {
		return true, nil
	}
	var works, genres int
	if err := d.Write.QueryRowContext(ctx, `SELECT EXISTS (SELECT 1 FROM works)`).Scan(&works); err != nil {
		return false, err
	}
	if works == 0 {
		return false, nil
	}
	if err := d.Write.QueryRowContext(ctx, `SELECT EXISTS (SELECT 1 FROM work_genres)`).Scan(&genres); err != nil {
		return false, err
	}
	return genres == 0, nil
}

func (d *DB) startIndexRecovery(ctx context.Context) error {
	d.recoverDone = make(chan struct{})
	runCtx, cancel := context.WithCancel(context.Background())
	d.recoverCancel = cancel

	need, err := d.indexRecoveryNeeded(ctx)
	if err != nil {
		close(d.recoverDone)
		return err
	}
	if !need {
		close(d.recoverDone)
		return nil
	}
	d.mu.Lock()
	d.recovering = true
	d.mu.Unlock()
	go func() {
		defer close(d.recoverDone)
		defer func() {
			d.mu.Lock()
			d.recovering = false
			d.mu.Unlock()
		}()
		if err := d.RecoverSearchIndex(runCtx); err != nil {
			d.mu.Lock()
			d.recoverErr = err
			d.mu.Unlock()
			if runCtx.Err() == nil {
				d.log.Error("search index recovery failed", "err", err)
			}
		}
	}()
	return nil
}

func (d *DB) stopIndexRecoveryUntil(deadline time.Time) {
	d.mu.Lock()
	if d.recoverWaited {
		d.mu.Unlock()
		return
	}
	d.recoverWaited = true
	cancel := d.recoverCancel
	done := d.recoverDone
	d.recoverCancel = nil
	d.mu.Unlock()
	if cancel != nil {
		cancel()
	}
	if done == nil {
		return
	}
	timer := time.NewTimer(remaining(deadline))
	defer timer.Stop()
	select {
	case <-done:
	case <-timer.C:
		select {
		case <-done:
		default:
			if d.log != nil {
				d.log.Warn("shutdown timed out", "task", "search-index-recovery")
			}
		}
	}
}

// SearchIndexReady reports whether FTS may be used. While false, search must
// use the LIKE fallback: the index may be missing, dirty, or still rebuilding.
func (d *DB) SearchIndexReady() bool {
	if d == nil {
		return false
	}
	d.mu.Lock()
	recovering := d.recovering
	d.mu.Unlock()
	if recovering {
		return false
	}
	ctx := context.Background()
	var exec Execer
	switch {
	case d.Read != nil:
		exec = d.Read
	case d.Write != nil:
		exec = d.Write
	default:
		return false
	}
	dirty, err := ftsDirty(ctx, exec)
	if err != nil || dirty {
		return false
	}
	n, err := worksFTSTriggerCount(ctx, exec)
	return err == nil && n == 2
}

// WaitSearchIndex blocks until background recovery finishes or ctx is done.
func (d *DB) WaitSearchIndex(ctx context.Context) error {
	if d == nil || d.recoverDone == nil {
		return nil
	}
	select {
	case <-d.recoverDone:
		d.mu.Lock()
		err := d.recoverErr
		d.mu.Unlock()
		if err != nil {
			return fmt.Errorf("search index recovery: %w", err)
		}
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}
