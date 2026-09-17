package db

import "context"

// RecoverSearchIndex rebuilds FTS when fts_dirty is set or works_fts triggers are missing.
func (d *DB) RecoverSearchIndex(ctx context.Context) error {
	if d == nil || d.Write == nil {
		return nil
	}
	var hasWorks int
	if err := d.Write.QueryRowContext(ctx, `SELECT count(*) FROM sqlite_master WHERE type='table' AND name='works'`).Scan(&hasWorks); err != nil {
		return err
	}
	if hasWorks == 0 {
		return nil
	}
	var works int
	if err := d.Write.QueryRowContext(ctx, `SELECT count(*) FROM works`).Scan(&works); err != nil {
		return err
	}
	dirty, err := ftsDirty(ctx, d.Write)
	if err != nil {
		return err
	}
	nTrig, err := worksFTSTriggerCount(ctx, d.Write)
	if err != nil {
		return err
	}
	var genres int
	if err := d.Write.QueryRowContext(ctx, `SELECT count(*) FROM work_genres`).Scan(&genres); err != nil {
		return err
	}
	needFTS := dirty || nTrig != 2
	needWarm := works > 0 && (genres == 0 || needFTS)
	if works == 0 {
		if nTrig != 2 {
			if err := EnsureWorksFTSTriggers(ctx, d.Write); err != nil {
				return err
			}
		}
		if dirty {
			return SetFTSDirty(ctx, d.Write, false)
		}
		return nil
	}
	if !needFTS && !needWarm {
		return nil
	}
	if needFTS {
		if err := RebuildWorksFTS(ctx, d.Write); err != nil {
			return err
		}
	}
	if needWarm {
		if err := WarmUpCatalog(ctx, d.Write); err != nil {
			return err
		}
		if err := Analyze(ctx, d.Write); err != nil {
			return err
		}
		_ = WarmCache(ctx, d.Write)
	}
	return SetFTSDirty(ctx, d.Write, false)
}
