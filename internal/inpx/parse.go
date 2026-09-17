// Package inpx parses INPX dumps and computes catalog keys.
package inpx

import (
	"bytes"
	"strconv"
	"strings"
	"unicode/utf8"

	"golang.org/x/text/encoding/charmap"
)

// Encoding is the detected character set of one dump member.
type Encoding string

const (
	EncodingUTF8   Encoding = "utf-8"
	EncodingCP1251 Encoding = "cp1251"
)

const fieldSep = 0x04

// SkipReason explains why a line produced no record.
type SkipReason int

const (
	SkipNone SkipReason = iota
	SkipEmpty
	SkipMalformed
	SkipNoLibID
)

func (s SkipReason) String() string {
	switch s {
	case SkipMalformed:
		return "malformed"
	case SkipNoLibID:
		return "no_libid"
	case SkipEmpty:
		return "empty"
	default:
		return ""
	}
}

// Author is one person from the AUTHOR field.
type Author struct {
	Last   string
	First  string
	Middle string
}

// Record is one catalog row from an .inp file.
type Record struct {
	Authors     []Author
	Genres      []string
	Title       string
	Series      string
	SeriesNo    string
	File        string
	Size        *int64
	LibID       string
	IsDeleted   bool
	Ext         string
	Date        string
	Lang        string
	LibRate     *int
	Keywords    string
	ArchiveName string
}

// DecodeFile detects encoding for one dump member (an .inp, version.info, or
// collection.info): strict UTF-8, else CP1251. Detection is per file, not per line.
func DecodeFile(raw []byte) ([]byte, Encoding) {
	if utf8.Valid(raw) {
		return stripBOM(raw), EncodingUTF8
	}
	out, err := charmap.Windows1251.NewDecoder().Bytes(raw)
	if err != nil {
		return raw, EncodingCP1251
	}
	return out, EncodingCP1251
}

func splitList(s string) []string {
	if s == "" {
		return nil
	}
	raw := strings.Split(s, ":")
	out := make([]string, 0, len(raw))
	for _, p := range raw {
		p = strings.TrimSpace(p)
		if p == "" {
			continue
		}
		out = append(out, p)
	}
	return out
}

func parseAuthor(seg string) Author {
	parts := strings.SplitN(seg, ",", 3)
	a := Author{}
	if len(parts) > 0 {
		a.Last = parts[0]
	}
	if len(parts) > 1 {
		a.First = parts[1]
	}
	if len(parts) > 2 {
		a.Middle = parts[2]
	}
	return a
}

func parseAuthors(field string) []Author {
	segs := splitList(field)
	if len(segs) == 0 {
		return []Author{{Last: "Неизвестен", First: "Автор"}}
	}
	out := make([]Author, 0, len(segs))
	for _, s := range segs {
		out = append(out, parseAuthor(s))
	}
	return out
}

func parseInt64(s string) *int64 {
	if s == "" {
		return nil
	}
	n, err := strconv.ParseInt(s, 10, 64)
	if err != nil {
		return nil
	}
	return &n
}

func parseInt(s string) *int {
	if s == "" {
		return nil
	}
	n, err := strconv.Atoi(s)
	if err != nil {
		return nil
	}
	return &n
}

// ParseLine parses one already-decoded (UTF-8) INP record. archiveName is the
// .zip derived from the .inp file name. Call DecodeFile on the member first.
func ParseLine(raw []byte, archiveName string) (Record, SkipReason) {
	raw = bytes.TrimRight(raw, "\r")
	if len(bytes.TrimSpace(raw)) == 0 {
		return Record{}, SkipEmpty
	}
	parts := bytes.Split(raw, []byte{fieldSep})
	nFields := len(parts)
	if nFields > 0 && len(parts[nFields-1]) == 0 {
		nFields--
	}
	if nFields < 14 {
		return Record{}, SkipMalformed
	}
	field := func(i int) string {
		return strings.TrimSpace(string(parts[i]))
	}
	libid := field(7)
	if libid == "" {
		return Record{}, SkipNoLibID
	}
	ext := field(9)
	if ext == "" {
		ext = "fb2"
	}
	del := field(8)
	rec := Record{
		Authors:     parseAuthors(field(0)),
		Genres:      splitList(field(1)),
		Title:       field(2),
		Series:      field(3),
		SeriesNo:    field(4),
		File:        field(5),
		Size:        parseInt64(field(6)),
		LibID:       libid,
		IsDeleted:   del != "" && del != "0",
		Ext:         ext,
		Date:        field(10),
		Lang:        field(11),
		LibRate:     parseInt(field(12)),
		Keywords:    field(13),
		ArchiveName: archiveName,
	}
	return rec, SkipNone
}

// ArchiveNameFromInp maps d.fb2-009373-367300.inp → d.fb2-009373-367300.zip.
func ArchiveNameFromInp(inpName string) string {
	name := inpName
	if i := strings.LastIndexAny(name, `/\`); i >= 0 {
		name = name[i+1:]
	}
	if strings.HasSuffix(strings.ToLower(name), ".inp") {
		return name[:len(name)-4] + ".zip"
	}
	return name + ".zip"
}
