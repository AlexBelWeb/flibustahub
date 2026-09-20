package personal

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/csv"
	"encoding/json"
	"io"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/alexbelweb/flibustahub/internal/apperr"
	"github.com/alexbelweb/flibustahub/internal/db"
	"github.com/alexbelweb/flibustahub/internal/repositories"
)

const (
	NoteStale       = "stale"
	NoteNotFound    = "not_found"
	NoteInvalid     = "invalid"
	NoteNoTimestamp = "no_timestamp"
)

type ImportNote struct {
	WorkKey string `json:"workKey,omitempty"`
	Title   string `json:"title,omitempty"`
	Field   string `json:"field,omitempty"`
	Reason  string `json:"reason"`
}

type ImportReport struct {
	Applied      int          `json:"applied"`
	Skipped      int          `json:"skipped"`
	NotFound     int          `json:"notFound"`
	Invalid      int          `json:"invalid"`
	Notes        []ImportNote `json:"notes,omitempty"`
	NotFoundPath string       `json:"notFoundPath,omitempty"`
}

type parsedRow struct {
	raw       dumpRow
	workID    int64
	rating    *int
	comment   *string
	want      *bool
	ratingTS  string
	commentTS string
	wantTS    string
	invalid   string
}

func (s *Service) PreviewImport(ctx context.Context, path string) (ImportReport, error) {
	return s.runImport(ctx, path, false)
}

func (s *Service) Import(ctx context.Context, path string) (ImportReport, error) {
	return s.runImport(ctx, path, true)
}

func (s *Service) runImport(ctx context.Context, path string, apply bool) (ImportReport, error) {
	if err := s.ready(); err != nil {
		return ImportReport{}, err
	}
	path = strings.TrimSpace(path)
	if path == "" {
		return ImportReport{}, apperr.New(apperr.CodePersonalImportFailed, nil)
	}
	rows, err := readDumpFile(path)
	if err != nil {
		return ImportReport{}, apperr.Wrap(apperr.CodePersonalImportFailed, err, nil)
	}
	now := s.stamp()
	rep := ImportReport{}
	var missing []dumpRow
	type pending struct {
		id        int64
		rating    *int
		ratingTS  string
		comment   *string
		commentTS string
		want      *bool
		wantTS    string
	}
	var writes []pending
	for _, raw := range rows {
		p := parseDumpRow(raw)
		if p.invalid != "" {
			rep.Invalid++
			rep.Notes = append(rep.Notes, ImportNote{WorkKey: raw.WorkKey, Title: raw.Title, Reason: NoteInvalid, Field: p.invalid})
			continue
		}
		local, err := s.matchWork(ctx, p)
		if err == sql.ErrNoRows {
			rep.NotFound++
			rep.Notes = append(rep.Notes, ImportNote{WorkKey: p.raw.WorkKey, Title: p.raw.Title, Reason: NoteNotFound})
			missing = append(missing, p.raw)
			continue
		}
		if err != nil {
			return ImportReport{}, apperr.Wrap(apperr.CodePersonalImportFailed, err, nil)
		}
		dec := decideRow(local, p, now)
		rep.Notes = append(rep.Notes, dec.notes...)
		if dec.applied {
			rep.Applied++
			writes = append(writes, pending{
				id: local.ID, rating: dec.rating, ratingTS: dec.ratingTS,
				comment: dec.comment, commentTS: dec.commentTS, want: dec.want, wantTS: dec.wantTS,
			})
		} else {
			rep.Skipped++
		}
	}
	if apply {
		if len(writes) > 0 {
			if _, err := s.db.Backup(ctx, db.BackupReasonPersonal); err != nil {
				return ImportReport{}, err
			}
			err := s.cat.WithWriteTx(ctx, func(tx *sql.Tx) error {
				for _, w := range writes {
					if err := repositories.ApplyPersonalTx(ctx, tx, w.id, w.rating, w.ratingTS, w.comment, w.commentTS, w.want, w.wantTS); err != nil {
						return err
					}
				}
				return nil
			})
			if err != nil {
				return ImportReport{}, apperr.Wrap(apperr.CodePersonalImportFailed, err, nil)
			}
		}
		if len(missing) > 0 {
			out := notFoundPath(path)
			var data []byte
			var encErr error
			if dumpFormatOf(path) == "json" {
				data, encErr = encodeJSON(missing)
			} else {
				data, encErr = encodeCSV(missing)
			}
			if encErr != nil {
				s.log.Warn("personal import not-found file skipped", "err", encErr)
			} else if err := writeAtomicFile(out, data); err != nil {
				s.log.Warn("personal import not-found file skipped", "err", err)
			} else {
				rep.NotFoundPath = out
			}
		}
	}
	return rep, nil
}

func (s *Service) matchWork(ctx context.Context, p parsedRow) (repositories.PersonalRow, error) {
	if p.workID > 0 {
		row, err := s.cat.Personal(ctx, p.workID)
		if err == nil && (p.raw.WorkKey == "" || row.WorkKey == p.raw.WorkKey) {
			return row, nil
		}
		if err != nil && err != sql.ErrNoRows {
			return repositories.PersonalRow{}, err
		}
	}
	if p.raw.WorkKey == "" {
		return repositories.PersonalRow{}, sql.ErrNoRows
	}
	return s.cat.WorkByKey(ctx, p.raw.WorkKey)
}

type rowDecision struct {
	applied   bool
	rating    *int
	ratingTS  string
	comment   *string
	commentTS string
	want      *bool
	wantTS    string
	notes     []ImportNote
}

func decideRow(local repositories.PersonalRow, p parsedRow, now string) rowDecision {
	var d rowDecision
	base := ImportNote{WorkKey: p.raw.WorkKey, Title: p.raw.Title}
	if n := decideField(nullString(local.RatingUpdatedAt), p.ratingTS, ratingEmpty(local), p.rating != nil || p.ratingTS != "", now); n.kind == applyField {
		d.applied = true
		d.rating = p.rating
		d.ratingTS = n.ts
	} else if n.kind != skipQuiet {
		note := base
		note.Field = "rating"
		note.Reason = n.kind
		d.notes = append(d.notes, note)
	}
	commentPresent := p.comment != nil || p.commentTS != ""
	if n := decideField(nullString(local.CommentUpdatedAt), p.commentTS, commentEmpty(local), commentPresent, now); n.kind == applyField {
		d.applied = true
		d.comment = p.comment
		d.commentTS = n.ts
	} else if n.kind != skipQuiet {
		note := base
		note.Field = "comment"
		note.Reason = n.kind
		d.notes = append(d.notes, note)
	}
	wantPresent := p.want != nil || p.wantTS != ""
	if n := decideField(nullString(local.WantToReadUpdatedAt), p.wantTS, wantEmpty(local), wantPresent, now); n.kind == applyField {
		d.applied = true
		d.want = p.want
		if d.want == nil {
			v := false
			d.want = &v
		}
		d.wantTS = n.ts
	} else if n.kind != skipQuiet {
		note := base
		note.Field = "wantToRead"
		note.Reason = n.kind
		d.notes = append(d.notes, note)
	}
	return d
}

const (
	applyField = "apply"
	skipQuiet  = ""
)

type fieldOut struct {
	kind string
	ts   string
}

func decideField(localTS, fileTS string, localEmpty, filePresent bool, now string) fieldOut {
	if !filePresent {
		return fieldOut{kind: skipQuiet}
	}
	if fileTS != "" {
		if localTS == "" || fileTS > localTS {
			return fieldOut{kind: applyField, ts: fileTS}
		}
		return fieldOut{kind: NoteStale}
	}
	if localEmpty {
		return fieldOut{kind: applyField, ts: now}
	}
	return fieldOut{kind: NoteNoTimestamp}
}

func ratingEmpty(r repositories.PersonalRow) bool { return !r.Rating.Valid }

func commentEmpty(r repositories.PersonalRow) bool {
	return !r.Comment.Valid || strings.TrimSpace(r.Comment.String) == ""
}

func wantEmpty(r repositories.PersonalRow) bool {
	return !r.WantToReadUpdatedAt.Valid && r.WantToRead == 0
}

func parseDumpRow(raw dumpRow) parsedRow {
	p := parsedRow{raw: raw, workID: raw.WorkID, ratingTS: strings.TrimSpace(raw.RatingUpdatedAt), commentTS: strings.TrimSpace(raw.CommentUpdatedAt), wantTS: strings.TrimSpace(raw.WantToReadUpdatedAt)}
	if raw.parseErr != "" {
		p.invalid = raw.parseErr
		return p
	}
	if raw.WorkKey == "" && raw.WorkID == 0 {
		p.invalid = "workKey"
		return p
	}
	if raw.Rating != nil {
		n, ok := ParseStars(strconv.FormatFloat(*raw.Rating, 'f', -1, 64))
		if !ok {
			p.invalid = "rating"
			return p
		}
		p.rating = &n
	}
	if raw.Comment != nil {
		c := *raw.Comment
		p.comment = &c
	}
	p.want = raw.WantToRead
	return p
}

func readDumpFile(path string) ([]dumpRow, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	if dumpFormatOf(path) == "json" {
		raw = bytes.TrimPrefix(raw, []byte{0xEF, 0xBB, 0xBF})
		var rows []dumpRow
		if err := json.Unmarshal(raw, &rows); err != nil {
			return nil, err
		}
		return rows, nil
	}
	return parseCSV(raw)
}

func parseCSV(raw []byte) ([]dumpRow, error) {
	raw = bytes.TrimPrefix(raw, []byte{0xEF, 0xBB, 0xBF})
	r := csv.NewReader(bytes.NewReader(raw))
	r.ReuseRecord = true
	header, err := r.Read()
	if err != nil {
		return nil, err
	}
	idx := map[string]int{}
	for i, h := range header {
		idx[strings.TrimSpace(h)] = i
	}
	var out []dumpRow
	for {
		rec, err := r.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, err
		}
		cell := func(name string) string {
			i, ok := idx[name]
			if !ok || i >= len(rec) {
				return ""
			}
			return rec[i]
		}
		row := dumpRow{
			WorkKey:             cell("workKey"),
			Authors:             cell("authors"),
			Title:               cell("title"),
			RatingUpdatedAt:     cell("ratingUpdatedAt"),
			CommentUpdatedAt:    cell("commentUpdatedAt"),
			WantToReadUpdatedAt: cell("wantToReadUpdatedAt"),
		}
		if id := cell("workId"); id != "" {
			n, convErr := strconv.ParseInt(id, 10, 64)
			if convErr != nil {
				return nil, convErr
			}
			row.WorkID = n
		}
		if s := cell("rating"); s != "" {
			n, ok := ParseStars(s)
			if !ok {
				row.parseErr = "rating"
			} else {
				stars := float64(n) / 2
				row.Rating = &stars
			}
		}
		if _, has := idx["comment"]; has {
			c := cell("comment")
			if c != "" || cell("commentUpdatedAt") != "" {
				row.Comment = &c
			}
		}
		if s := cell("wantToRead"); s != "" {
			v, ok := ParseWantToRead(s)
			if !ok {
				row.parseErr = "wantToRead"
			} else {
				row.WantToRead = v
			}
		}
		out = append(out, row)
	}
	return out, nil
}

func notFoundPath(src string) string {
	ext := filepath.Ext(src)
	base := strings.TrimSuffix(src, ext)
	if ext == "" {
		ext = ".csv"
	}
	return base + ".notfound" + ext
}
