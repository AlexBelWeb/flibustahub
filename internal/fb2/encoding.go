package fb2

import (
	"bytes"
	"strings"
	"unicode/utf8"

	"golang.org/x/text/encoding"
	"golang.org/x/text/encoding/charmap"
	"golang.org/x/text/encoding/unicode"
	"golang.org/x/text/transform"
)

const (
	encSample = 800
	encScore  = 32 * 1024
)

var (
	unicodeUTF16LE = unicode.UTF16(unicode.LittleEndian, unicode.IgnoreBOM)
	unicodeUTF16BE = unicode.UTF16(unicode.BigEndian, unicode.IgnoreBOM)
)

type charset struct {
	name string
	enc  encoding.Encoding
}

// DecodeBody converts an FB2 payload to UTF-8 and names the encoding used.
// BOM is honoured first; otherwise UTF-16 without BOM is recognised by the
// share of NUL bytes; otherwise encoding= is read from the first ~800 bytes.
// Candidates are scored by text plausibility, not by U+FFFD count.
func DecodeBody(raw []byte) ([]byte, string) {
	if len(raw) == 0 {
		return raw, ""
	}
	declared, bomSkip := declaredCharset(raw)
	if declared.name != "" && bomSkip > 0 {
		out, err := decodeAll(declared, raw[bomSkip:])
		if err == nil {
			return out, declared.name
		}
	}
	if u16, ok := utf16WithoutBOM(raw); ok {
		out, err := decodeAll(u16, raw)
		if err == nil {
			return out, u16.name
		}
	}

	cands := make([]charset, 0, 5)
	if declared.name != "" && bomSkip == 0 {
		cands = append(cands, declared)
	}
	for _, c := range []charset{
		{name: "utf-8", enc: encoding.Nop},
		{name: "windows-1251", enc: charmap.Windows1251},
		{name: "cp866", enc: charmap.CodePage866},
		{name: "koi8-r", enc: charmap.KOI8R},
	} {
		if !hasCharset(cands, c.name) {
			cands = append(cands, c)
		}
	}

	validUTF8 := likelyUTF8(raw)
	var (
		best     charset
		bestBody []byte
		bestN    = -1.0
	)
	for i, c := range cands {
		in := payloadFor(c, declared, raw, bomSkip)
		out, err := decodeAll(c, in)
		if err != nil {
			continue
		}
		sample := out
		if len(sample) > encScore {
			sample = sample[:encScore]
		}
		score := TextScore(string(sample))
		if validUTF8 && c.name == "utf-8" && score >= minPlausible {
			score += 0.15
		}
		if score > bestN || (bestBody == nil && i == 0) {
			bestN = score
			best = c
			bestBody = out
		}
	}
	if bestBody == nil {
		return bytes.ToValidUTF8(raw, []byte("\ufffd")), "utf-8"
	}
	return bestBody, best.name
}

func likelyUTF8(raw []byte) bool {
	if !utf8.Valid(raw) {
		return false
	}
	n := len(raw)
	if n > encScore {
		n = encScore
	}
	cyr, high := 0, 0
	for _, r := range string(raw[:n]) {
		if r < 0x80 {
			continue
		}
		high++
		if r >= 0x0400 && r <= 0x04FF {
			cyr++
		}
	}
	if high == 0 {
		return true
	}
	return cyr*2 >= high
}

func payloadFor(c, declared charset, raw []byte, bomSkip int) []byte {
	if declared.name != "" && c.name == declared.name && bomSkip > 0 && bomSkip <= len(raw) {
		return raw[bomSkip:]
	}
	return raw
}

func declaredCharset(raw []byte) (charset, int) {
	if len(raw) >= 3 && raw[0] == 0xef && raw[1] == 0xbb && raw[2] == 0xbf {
		return charset{name: "utf-8", enc: encoding.Nop}, 3
	}
	if len(raw) >= 2 && raw[0] == 0xff && raw[1] == 0xfe {
		return charset{name: "utf-16le", enc: unicode.UTF16(unicode.LittleEndian, unicode.IgnoreBOM)}, 2
	}
	if len(raw) >= 2 && raw[0] == 0xfe && raw[1] == 0xff {
		return charset{name: "utf-16be", enc: unicode.UTF16(unicode.BigEndian, unicode.IgnoreBOM)}, 2
	}
	head := raw
	if len(head) > encSample {
		head = head[:encSample]
	}
	return charsetFromName(encodingAttr(head)), 0
}

func encodingAttr(head []byte) string {
	lower := bytes.ToLower(head)
	i := bytes.Index(lower, []byte("encoding="))
	if i < 0 {
		return ""
	}
	rest := bytes.TrimSpace(lower[i+len("encoding="):])
	if len(rest) == 0 {
		return ""
	}
	quote := rest[0]
	if quote != '"' && quote != '\'' {
		return ""
	}
	rest = rest[1:]
	j := bytes.IndexByte(rest, quote)
	if j < 0 {
		return ""
	}
	return string(rest[:j])
}

func charsetFromName(name string) charset {
	switch alias(name) {
	case "utf-8", "utf8":
		return charset{name: "utf-8", enc: encoding.Nop}
	case "utf-16le", "utf16le":
		return charset{name: "utf-16le", enc: unicodeUTF16LE}
	case "utf-16be", "utf16be", "utf-16", "utf16":
		return charset{name: "utf-16be", enc: unicodeUTF16BE}
	case "windows-1251", "cp1251", "windows1251":
		return charset{name: "windows-1251", enc: charmap.Windows1251}
	case "cp866", "ibm866", "866", "dos":
		return charset{name: "cp866", enc: charmap.CodePage866}
	case "koi8-r", "koi8r", "koi8-u", "koi8u":
		return charset{name: "koi8-r", enc: charmap.KOI8R}
	case "iso-8859-5", "iso8859-5", "iso88595":
		return charset{name: "iso-8859-5", enc: charmap.ISO8859_5}
	default:
		return charset{}
	}
}

func alias(s string) string {
	s = strings.ToLower(strings.TrimSpace(s))
	s = strings.ReplaceAll(s, "_", "-")
	return s
}

func hasCharset(cands []charset, name string) bool {
	for _, c := range cands {
		if c.name == name {
			return true
		}
	}
	return false
}

func decodeAll(c charset, in []byte) ([]byte, error) {
	if c.enc == nil || c.enc == encoding.Nop {
		if utf8.Valid(in) {
			return in, nil
		}
		return bytes.ToValidUTF8(in, []byte("\ufffd")), nil
	}
	out, _, err := transform.Bytes(c.enc.NewDecoder(), in)
	return out, err
}
