package personal

import (
	"bytes"
	"database/sql"
	"encoding/csv"
	"encoding/json"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/alexbelweb/flibustahub/internal/repositories"
)

var csvHeader = []string{
	"workKey", "workId", "authors", "title", "rating", "comment", "wantToRead",
	"ratingUpdatedAt", "commentUpdatedAt", "wantToReadUpdatedAt",
}

type dumpRow struct {
	WorkKey             string   `json:"workKey"`
	WorkID              int64    `json:"workId"`
	Authors             string   `json:"authors"`
	Title               string   `json:"title"`
	Rating              *float64 `json:"rating"`
	Comment             *string  `json:"comment"`
	WantToRead          *bool    `json:"wantToRead"`
	RatingUpdatedAt     string   `json:"ratingUpdatedAt"`
	CommentUpdatedAt    string   `json:"commentUpdatedAt"`
	WantToReadUpdatedAt string   `json:"wantToReadUpdatedAt"`
	parseErr            string   `json:"-"`
}

func rowFromPersonal(r repositories.PersonalRow) dumpRow {
	out := dumpRow{
		WorkKey:             r.WorkKey,
		WorkID:              r.ID,
		Authors:             r.AuthorsText,
		Title:               r.Title,
		RatingUpdatedAt:     nullString(r.RatingUpdatedAt),
		CommentUpdatedAt:    nullString(r.CommentUpdatedAt),
		WantToReadUpdatedAt: nullString(r.WantToReadUpdatedAt),
	}
	if r.RatingUpdatedAt.Valid {
		if r.Rating.Valid {
			stars := float64(r.Rating.Int64) / 2
			out.Rating = &stars
		}
	}
	if r.CommentUpdatedAt.Valid {
		c := ""
		if r.Comment.Valid {
			c = r.Comment.String
		}
		out.Comment = &c
	}
	if r.WantToReadUpdatedAt.Valid {
		v := r.WantToRead == 1
		out.WantToRead = &v
	}
	return out
}

func nullString(v sql.NullString) string {
	if v.Valid {
		return v.String
	}
	return ""
}

func encodeJSON(rows []dumpRow) ([]byte, error) {
	if rows == nil {
		rows = []dumpRow{}
	}
	return json.MarshalIndent(rows, "", "  ")
}

func encodeCSV(rows []dumpRow) ([]byte, error) {
	var buf bytes.Buffer
	buf.Write([]byte{0xEF, 0xBB, 0xBF})
	w := csv.NewWriter(&buf)
	w.UseCRLF = true
	if err := w.Write(csvHeader); err != nil {
		return nil, err
	}
	for _, r := range rows {
		if err := w.Write(csvRecord(r)); err != nil {
			return nil, err
		}
	}
	w.Flush()
	return buf.Bytes(), w.Error()
}

func csvRecord(r dumpRow) []string {
	rating := ""
	if r.RatingUpdatedAt != "" {
		if r.Rating != nil {
			rating = StarsFromInternal(int(*r.Rating*2 + 0.0000001))
		}
	}
	comment := ""
	if r.Comment != nil {
		comment = *r.Comment
	}
	return []string{
		r.WorkKey,
		strconv.FormatInt(r.WorkID, 10),
		r.Authors,
		r.Title,
		rating,
		comment,
		formatWantCSV(r.WantToRead),
		r.RatingUpdatedAt,
		r.CommentUpdatedAt,
		r.WantToReadUpdatedAt,
	}
}

func writeAtomicFile(dest string, data []byte) error {
	tmp := dest + ".part"
	_ = os.Remove(tmp)
	if err := os.MkdirAll(filepath.Dir(dest), 0o755); err != nil {
		return err
	}
	if err := os.WriteFile(tmp, data, 0o644); err != nil {
		return err
	}
	if err := os.Rename(tmp, dest); err != nil {
		_ = os.Remove(tmp)
		return err
	}
	return nil
}

func dumpFormatOf(path string) string {
	switch strings.ToLower(filepath.Ext(path)) {
	case ".json":
		return "json"
	default:
		return "csv"
	}
}
