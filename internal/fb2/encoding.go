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

type charset struct {
	name string
	enc  encoding.Encoding
}

// DecodeBody converts an FB2 payload to UTF-8.
// BOM is honoured first; otherwise encoding= is read from the first ~800 bytes.
// Candidates are tried in order and the variant with the fewest U+FFFD runes wins.
func DecodeBody(raw []byte) []byte {
	if len(raw) == 0 {
		return raw
	}
	declared, bomSkip := declaredCharset(raw)
	cands := make([]charset, 0, 4)
	if declared.name != "" {
		cands = append(cands, declared)
	}
	for _, c := range []charset{
		{name: "utf-8", enc: encoding.Nop},
		{name: "windows-1251", enc: charmap.Windows1251},
		{name: "koi8-r", enc: charmap.KOI8R},
	} {
		if !hasCharset(cands, c.name) {
			cands = append(cands, c)
		}
	}

	best := cands[0]
	bestN := int(^uint(0) >> 1)
	for _, c := range cands {
		in := payloadFor(c, declared, raw, bomSkip)
		sample := in
		if len(sample) > encScore {
			sample = sample[:encScore]
		}
		_, n := decodeCount(c, sample)
		if n < bestN {
			bestN = n
			best = c
		}
	}
	out, err := decodeAll(best, payloadFor(best, declared, raw, bomSkip))
	if err != nil {
		return bytes.ToValidUTF8(raw, []byte("\ufffd"))
	}
	return out
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
		return charset{name: "utf-16le", enc: unicode.UTF16(unicode.LittleEndian, unicode.IgnoreBOM)}
	case "utf-16be", "utf16be", "utf-16", "utf16":
		return charset{name: "utf-16be", enc: unicode.UTF16(unicode.BigEndian, unicode.IgnoreBOM)}
	case "windows-1251", "cp1251", "windows1251":
		return charset{name: "windows-1251", enc: charmap.Windows1251}
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

func decodeCount(c charset, sample []byte) ([]byte, int) {
	out, err := decodeAll(c, sample)
	if err != nil {
		return nil, len(sample) + 1
	}
	return out, bytes.Count(out, []byte("\ufffd"))
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
