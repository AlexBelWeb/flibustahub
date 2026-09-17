// Package inpximport imports an INPX dump into the catalog database.
package inpximport

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"log/slog"
	"os"
	"time"

	"github.com/alexbelweb/flibustahub/internal/apperr"
	"github.com/alexbelweb/flibustahub/internal/db"
	"github.com/alexbelweb/flibustahub/internal/inpx"
	"github.com/alexbelweb/flibustahub/internal/repositories"
)

const (
	PhaseReading = "reading"
	PhaseRecords = "records"
	PhaseFTS     = "fts"
	PhaseWarmup  = "warmup"

	StatusRunning   = "running"
	StatusDone      = "done"
	StatusFailed    = "failed"
	StatusCancelled = "cancelled"
)

// Progress is the backend contract the UI will bind to Wails events.
type Progress struct {
	Phase        string `json:"phase"`
	RecordsSeen  int    `json:"recordsSeen"`
	RecordsTotal int    `json:"recordsTotal"`
}

// Notes is stored in import_batches.notes as JSON.
type Notes struct {
	MissingArchives      []string       `json:"missing_archives"`
	MissingArchivesTotal int            `json:"missing_archives_total"`
	UnnamedGenres        []string       `json:"unnamed_genres"`
	UnnamedGenresTotal   int            `json:"unnamed_genres_total"`
	SkippedMalformed     int            `json:"skipped_malformed"`
	SkippedNoLibID       int            `json:"skipped_no_libid"`
	PhasesMS             map[string]int `json:"phases_ms,omitempty"`
}

// Report is the finished import_batches row plus parsed notes.
type Report struct {
	ID                   int64
	Status               string
	INPXPath             string
	INPXVersion          string
	RecordsSeen          int
	WorksAdded           int
	EditionsAdded        int
	EditionsUpdated      int
	EditionsDeactivated  int
	LibIDCollisions      int
	Notes                Notes
}

// Options configure one import run.
type Options struct {
	LibraryRoot  string
	INPXPath     string
	Now          func() time.Time
	Progress     func(Progress)
	RecordsProbe func(tx *sql.Tx) error // tests: inspect FTS during records
}

// Service imports dumps using the catalog write pool.
type Service struct {
	catalog *db.DB
	log     *slog.Logger
}

func New(catalog *db.DB, log *slog.Logger) *Service {
	if log == nil {
		log = slog.Default()
	}
	return &Service{catalog: catalog, log: log}
}

func (s *Service) Import(ctx context.Context, opt Options) (Report, error) {
	if opt.Now == nil {
		opt.Now = time.Now
	}
	path, err := inpx.FindINPX(opt.LibraryRoot, opt.INPXPath)
	if err != nil {
		return Report{}, apperr.Wrap(apperr.CodeINPXNotFound, err, nil)
	}

	phases := map[string]int{}
	mark := func(name string, start time.Time) {
		phases[name] = int(time.Since(start).Milliseconds())
	}

	tBackup := time.Now()
	if _, err := s.catalog.Backup(ctx); err != nil {
		return Report{}, err
	}
	mark("backup", tBackup)

	conn, err := s.catalog.Write.Conn(ctx)
	if err != nil {
		return Report{}, apperr.Wrap(apperr.CodeImportFailed, err, nil)
	}
	defer conn.Close()

	if err := db.ApplyImportPragmas(ctx, conn); err != nil {
		return Report{}, apperr.Wrap(apperr.CodeImportFailed, err, nil)
	}
	defer func() { _ = db.RestoreWorkPragmas(context.Background(), conn) }()

	started := opt.Now().UTC().Format(time.RFC3339)
	var batchID int64
	if err := conn.QueryRowContext(ctx, `INSERT INTO import_batches (started_at, status, inpx_path) VALUES (?, ?, ?) RETURNING id`,
		started, StatusRunning, path).Scan(&batchID); err != nil {
		return Report{}, apperr.Wrap(apperr.CodeImportFailed, err, nil)
	}

	emit := throttleProgress(opt.Progress)
	emit(Progress{Phase: PhaseReading})

	tRead := time.Now()
	st, err := os.Stat(path)
	if err != nil {
		_ = finishBatch(ctx, conn, batchID, StatusFailed, Report{INPXPath: path}, Notes{}, opt.Now)
		return Report{}, apperr.Wrap(apperr.CodeINPXNotFound, err, nil)
	}
	f, err := os.Open(path)
	if err != nil {
		_ = finishBatch(ctx, conn, batchID, StatusFailed, Report{INPXPath: path}, Notes{}, opt.Now)
		return Report{}, apperr.Wrap(apperr.CodeImportFailed, err, nil)
	}
	metaPeek, err := inpx.PeekMeta(f, st.Size())
	_ = f.Close()
	if err != nil {
		_ = finishBatch(ctx, conn, batchID, StatusFailed, Report{INPXPath: path}, Notes{}, opt.Now)
		return Report{}, apperr.Wrap(apperr.CodeImportFailed, err, nil)
	}
	missing := inpx.MissingArchives(opt.LibraryRoot, metaPeek.Archives)
	notes := Notes{
		MissingArchives:      clip(missing, 80),
		MissingArchivesTotal: len(missing),
	}
	mark("reading", tRead)
	if _, err := conn.ExecContext(ctx, `UPDATE import_batches SET inpx_version = ? WHERE id = ?`, metaPeek.Version, batchID); err != nil {
		return Report{}, apperr.Wrap(apperr.CodeImportFailed, err, nil)
	}

	tx, err := conn.BeginTx(ctx, nil)
	if err != nil {
		return Report{}, apperr.Wrap(apperr.CodeImportFailed, err, nil)
	}
	rollback := func() { _ = tx.Rollback() }

	if err := db.DropWorksFTSTriggers(ctx, tx); err != nil {
		rollback()
		_ = finishBatch(ctx, conn, batchID, StatusFailed, Report{INPXPath: path}, notes, opt.Now)
		return Report{}, apperr.Wrap(apperr.CodeImportFailed, err, nil)
	}
	if err := db.SetFTSDirty(ctx, tx, true); err != nil {
		rollback()
		return Report{}, apperr.Wrap(apperr.CodeImportFailed, err, nil)
	}

	prep, err := repositories.PrepareImport(ctx, tx, opt.Now())
	if err != nil {
		rollback()
		return Report{}, apperr.Wrap(apperr.CodeImportFailed, err, nil)
	}
	defer prep.Close()

	rep := Report{ID: batchID, INPXPath: path, INPXVersion: metaPeek.Version, Status: StatusDone}
	tRec := time.Now()
	seen := 0
	probed := false
	f2, err := os.Open(path)
	if err != nil {
		rollback()
		return Report{}, apperr.Wrap(apperr.CodeImportFailed, err, nil)
	}
	meta, err := inpx.WalkRecords(f2, st.Size(), func(rec inpx.Record) error {
		if err := ctx.Err(); err != nil {
			return err
		}
		res, err := prep.Apply(ctx, rec)
		if err != nil {
			return err
		}
		seen++
		if res.WorkAdded {
			rep.WorksAdded++
		}
		if res.EditionAdded {
			rep.EditionsAdded++
		}
		if res.EditionUpdated {
			rep.EditionsUpdated++
		}
		if res.Collision {
			rep.LibIDCollisions++
		}
		if !probed && opt.RecordsProbe != nil {
			probed = true
			if err := opt.RecordsProbe(tx); err != nil {
				return err
			}
		}
		emit(Progress{Phase: PhaseRecords, RecordsSeen: seen})
		return nil
	})
	_ = f2.Close()
	if err != nil {
		rollback()
		status := StatusFailed
		if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
			status = StatusCancelled
			err = apperr.Wrap(apperr.CodeImportCancelled, err, nil)
		} else {
			err = apperr.Wrap(apperr.CodeImportFailed, err, nil)
		}
		_ = finishBatch(ctx, conn, batchID, status, rep, notes, opt.Now)
		return Report{}, err
	}
	deact, err := repositories.DeactivateMissing(ctx, tx)
	if err != nil {
		rollback()
		_ = finishBatch(ctx, conn, batchID, StatusFailed, rep, notes, opt.Now)
		return Report{}, apperr.Wrap(apperr.CodeImportFailed, err, nil)
	}
	rep.EditionsDeactivated = int(deact)
	rep.RecordsSeen = seen
	notes.SkippedMalformed = meta.SkippedMalformed
	notes.SkippedNoLibID = meta.SkippedNoLibID
	notes.UnnamedGenres = prep.UnnamedGenres()
	notes.UnnamedGenresTotal = prep.UnnamedTotal
	if err := tx.Commit(); err != nil {
		_ = finishBatch(ctx, conn, batchID, StatusFailed, rep, notes, opt.Now)
		return Report{}, apperr.Wrap(apperr.CodeImportFailed, err, nil)
	}
	mark("records", tRec)

	emit(Progress{Phase: PhaseFTS, RecordsSeen: seen})
	tFTS := time.Now()
	if err := db.RebuildWorksFTS(ctx, conn); err != nil {
		_ = finishBatch(ctx, conn, batchID, StatusFailed, rep, notes, opt.Now)
		return Report{}, apperr.Wrap(apperr.CodeImportFailed, err, nil)
	}
	mark("fts", tFTS)

	emit(Progress{Phase: PhaseWarmup, RecordsSeen: seen})
	tWarm := time.Now()
	if err := db.WarmUpCatalog(ctx, conn); err != nil {
		_ = finishBatch(ctx, conn, batchID, StatusFailed, rep, notes, opt.Now)
		return Report{}, apperr.Wrap(apperr.CodeImportFailed, err, nil)
	}
	mark("warmup", tWarm)
	tAn := time.Now()
	if err := db.Analyze(ctx, conn); err != nil {
		_ = finishBatch(ctx, conn, batchID, StatusFailed, rep, notes, opt.Now)
		return Report{}, apperr.Wrap(apperr.CodeImportFailed, err, nil)
	}
	mark("analyze", tAn)
	_ = db.WarmCache(ctx, conn)
	if err := db.SetFTSDirty(ctx, conn, false); err != nil {
		_ = finishBatch(ctx, conn, batchID, StatusFailed, rep, notes, opt.Now)
		return Report{}, apperr.Wrap(apperr.CodeImportFailed, err, nil)
	}

	notes.PhasesMS = phases
	rep.Notes = notes
	rep.Status = StatusDone
	if err := finishBatch(ctx, conn, batchID, StatusDone, rep, notes, opt.Now); err != nil {
		return Report{}, apperr.Wrap(apperr.CodeImportFailed, err, nil)
	}
	return rep, nil
}

func finishBatch(_ context.Context, conn *sql.Conn, id int64, status string, rep Report, notes Notes, now func() time.Time) error {
	raw, err := json.Marshal(notes)
	if err != nil {
		return err
	}
	_, err = conn.ExecContext(context.Background(), `UPDATE import_batches SET
		finished_at = ?, status = ?, inpx_version = ?, records_seen = ?, works_added = ?,
		editions_added = ?, editions_updated = ?, editions_deactivated = ?, libid_collisions = ?, notes = ?
		WHERE id = ?`,
		now().UTC().Format(time.RFC3339), status, rep.INPXVersion, rep.RecordsSeen, rep.WorksAdded,
		rep.EditionsAdded, rep.EditionsUpdated, rep.EditionsDeactivated, rep.LibIDCollisions, string(raw), id)
	return err
}

func clip(v []string, n int) []string {
	if len(v) <= n {
		return v
	}
	return append([]string(nil), v[:n]...)
}

func throttleProgress(fn func(Progress)) func(Progress) {
	if fn == nil {
		return func(Progress) {}
	}
	var last time.Time
	var lastSeen int
	return func(p Progress) {
		now := time.Now()
		if p.Phase == PhaseRecords && p.RecordsSeen-lastSeen < 2000 && now.Sub(last) < 100*time.Millisecond && p.RecordsSeen != 0 {
			return
		}
		last = now
		lastSeen = p.RecordsSeen
		fn(p)
	}
}
