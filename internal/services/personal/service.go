package personal

import (
	"context"
	"database/sql"
	"log/slog"
	"time"

	"github.com/alexbelweb/flibustahub/internal/apperr"
	"github.com/alexbelweb/flibustahub/internal/db"
	"github.com/alexbelweb/flibustahub/internal/repositories"
)

// StampLayout is the lexical UTC form used for personal timestamps.
const StampLayout = "2006-01-02T15:04:05.000Z"

// Service stores ratings, notes, the want-to-read flag, and personal dumps.
type Service struct {
	db  *db.DB
	cat *repositories.Catalog
	log *slog.Logger
	now func() time.Time
}

func New(catalogDB *db.DB, log *slog.Logger) *Service {
	if log == nil {
		log = slog.Default()
	}
	return &Service{
		db:  catalogDB,
		cat: repositories.NewCatalog(catalogDB),
		log: log,
		now: time.Now,
	}
}

func (s *Service) ready() error {
	if s == nil || s.db == nil || s.cat == nil {
		return apperr.New(apperr.CodeDBOpenFailed, nil)
	}
	return nil
}

func Stamp(t time.Time) string {
	return t.UTC().Format(StampLayout)
}

func (s *Service) stamp() string {
	return Stamp(s.now())
}

type Snapshot struct {
	UnsyncedCount int    `json:"unsyncedCount"`
	LastExportAt  string `json:"lastExportAt,omitempty"`
}

func (s *Service) Snapshot(ctx context.Context) (Snapshot, error) {
	if err := s.ready(); err != nil {
		return Snapshot{}, err
	}
	row, err := s.cat.PersonalSnapshot(ctx)
	if err != nil {
		return Snapshot{}, apperr.Wrap(apperr.CodeInternal, err, nil)
	}
	return Snapshot{UnsyncedCount: row.UnsyncedCount, LastExportAt: row.LastExportAt}, nil
}

func (s *Service) SetRating(ctx context.Context, id int64, rating *int) error {
	if err := s.ready(); err != nil {
		return err
	}
	if id <= 0 {
		return apperr.New(apperr.CodeNotFound, map[string]string{"kind": "work"})
	}
	if rating != nil && (*rating < 1 || *rating > 10) {
		return apperr.New(apperr.CodeInvalidRating, nil)
	}
	_, err := s.cat.SetRating(ctx, id, rating, s.stamp())
	if err == sql.ErrNoRows {
		return apperr.New(apperr.CodeNotFound, map[string]string{"kind": "work"})
	}
	if err != nil {
		return apperr.Wrap(apperr.CodeInternal, err, nil)
	}
	return nil
}

func (s *Service) SetComment(ctx context.Context, id int64, comment string) error {
	if err := s.ready(); err != nil {
		return err
	}
	if id <= 0 {
		return apperr.New(apperr.CodeNotFound, map[string]string{"kind": "work"})
	}
	_, err := s.cat.SetComment(ctx, id, comment, s.stamp())
	if err == sql.ErrNoRows {
		return apperr.New(apperr.CodeNotFound, map[string]string{"kind": "work"})
	}
	if err != nil {
		return apperr.Wrap(apperr.CodeInternal, err, nil)
	}
	return nil
}

func (s *Service) SetWantToRead(ctx context.Context, id int64, want bool) error {
	if err := s.ready(); err != nil {
		return err
	}
	if id <= 0 {
		return apperr.New(apperr.CodeNotFound, map[string]string{"kind": "work"})
	}
	_, err := s.cat.SetWantToRead(ctx, id, want, s.stamp())
	if err == sql.ErrNoRows {
		return apperr.New(apperr.CodeNotFound, map[string]string{"kind": "work"})
	}
	if err != nil {
		return apperr.Wrap(apperr.CodeInternal, err, nil)
	}
	return nil
}
