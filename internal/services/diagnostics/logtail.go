package diagnostics

import (
	"bufio"
	"encoding/json"
	"io"
	"os"
	"strings"
	"unicode/utf8"
)

const (
	logReadBufSize  = 256 * 1024
	logLineMaxRunes = 16 * 1024
)

func lastWarnErrors(path string, limit int) []string {
	if path == "" || limit <= 0 {
		return []string{}
	}
	f, err := os.Open(path)
	if err != nil {
		return []string{}
	}
	defer func() { _ = f.Close() }()
	r := bufio.NewReaderSize(f, logReadBufSize)
	var lines []string
	for {
		line, err := readLogLine(r)
		line = strings.TrimSpace(line)
		if line != "" && isWarnOrError(line) {
			lines = append(lines, line)
		}
		if err == io.EOF {
			break
		}
		if err != nil {
			// Keep lines already collected; the remainder is unreadable.
			break
		}
	}
	if len(lines) > limit {
		lines = lines[len(lines)-limit:]
	}
	if lines == nil {
		return []string{}
	}
	return lines
}

func readLogLine(r *bufio.Reader) (string, error) {
	var b strings.Builder
	runes := 0
	skipping := false
	for {
		chunk, err := r.ReadSlice('\n')
		hasNL := len(chunk) > 0 && chunk[len(chunk)-1] == '\n'
		body := chunk
		if hasNL {
			body = chunk[:len(chunk)-1]
			if len(body) > 0 && body[len(body)-1] == '\r' {
				body = body[:len(body)-1]
			}
		}
		if !skipping {
			n, rest := appendRunes(&b, body, logLineMaxRunes-runes)
			runes += n
			if rest {
				skipping = true
			}
		}
		if err == bufio.ErrBufferFull {
			continue
		}
		if err == io.EOF {
			return b.String(), io.EOF
		}
		if err != nil {
			if b.Len() > 0 {
				return b.String(), err
			}
			return "", err
		}
		return b.String(), nil
	}
}

func appendRunes(b *strings.Builder, p []byte, budget int) (written int, overflow bool) {
	if budget <= 0 {
		return 0, utf8.RuneCount(p) > 0
	}
	s := string(p)
	n := 0
	for _, r := range s {
		if n >= budget {
			return n, true
		}
		b.WriteRune(r)
		n++
	}
	return n, false
}

func isWarnOrError(line string) bool {
	var rec struct {
		Level string `json:"level"`
	}
	if json.Unmarshal([]byte(line), &rec) == nil {
		lvl := strings.ToUpper(rec.Level)
		return lvl == "WARN" || lvl == "WARNING" || lvl == "ERROR"
	}
	lower := strings.ToLower(line)
	return strings.Contains(lower, `"level":"error"`) ||
		strings.Contains(lower, `"level":"warn"`) ||
		strings.Contains(lower, `"level":"warning"`)
}
