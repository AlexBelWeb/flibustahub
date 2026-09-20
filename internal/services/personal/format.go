package personal

import (
	"fmt"
	"strconv"
	"strings"
)

const (
	starsMin = 0.5
	starsMax = 5.0
	step     = 0.5
)

// StarsFromInternal formats 1..10 as 0.5..5 for the dump file.
func StarsFromInternal(n int) string {
	if n%2 == 0 {
		return strconv.Itoa(n / 2)
	}
	return fmt.Sprintf("%d.5", n/2)
}

// ParseStars accepts 0.5..5 with a point or comma, or an integer 1..5.
// Values outside the range or finer than a half star are rejected.
func ParseStars(raw string) (int, bool) {
	s := strings.TrimSpace(raw)
	if s == "" {
		return 0, true
	}
	s = strings.ReplaceAll(s, ",", ".")
	f, err := strconv.ParseFloat(s, 64)
	if err != nil {
		return 0, false
	}
	if f < starsMin || f > starsMax {
		return 0, false
	}
	halves := f / step
	rounded := float64(int(halves + 0.0000001))
	if halves-rounded > 0.0000001 || rounded-halves > 0.0000001 {
		return 0, false
	}
	n := int(f/step + 0.0000001)
	if n < 1 || n > 10 {
		return 0, false
	}
	return n, true
}

// ParseWantToRead accepts 1/0/true/false, case-insensitive.
// Empty is "not set". Anything else is invalid.
func ParseWantToRead(raw string) (value *bool, ok bool) {
	s := strings.TrimSpace(raw)
	if s == "" {
		return nil, true
	}
	switch strings.ToLower(s) {
	case "1", "true":
		v := true
		return &v, true
	case "0", "false":
		v := false
		return &v, true
	default:
		return nil, false
	}
}

func formatWantCSV(v *bool) string {
	if v == nil {
		return ""
	}
	if *v {
		return "1"
	}
	return "0"
}
