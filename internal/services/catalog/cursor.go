package catalog

import (
	"encoding/base64"
	"encoding/json"
	"strings"
)

type cursorKind string

const (
	curTitle    cursorKind = "title"
	curAdded    cursorKind = "added"
	curName     cursorKind = "name"
	curSeriesNo cursorKind = "seriesno"
	curRating   cursorKind = "rating"
	curRatedAt  cursorKind = "ratedat"
	curWantAt   cursorKind = "wantat"
)

type pageCursor struct {
	K  cursorKind `json:"k"`
	V  string     `json:"v"`
	ID int64      `json:"id"`
	B  int        `json:"b,omitempty"`
	N  *float64   `json:"n,omitempty"`
	R  int        `json:"r,omitempty"`
}

func encodeCursor(c pageCursor) string {
	raw, err := json.Marshal(c)
	if err != nil {
		return ""
	}
	return base64.RawURLEncoding.EncodeToString(raw)
}

func decodeCursor(s string, want cursorKind) (pageCursor, bool) {
	s = strings.TrimSpace(s)
	if s == "" {
		return pageCursor{}, false
	}
	raw, err := base64.RawURLEncoding.DecodeString(s)
	if err != nil {
		return pageCursor{}, false
	}
	var c pageCursor
	if err := json.Unmarshal(raw, &c); err != nil {
		return pageCursor{}, false
	}
	if c.K != want || c.ID <= 0 {
		return pageCursor{}, false
	}
	return c, true
}
