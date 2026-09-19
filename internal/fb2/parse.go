package fb2

import (
	"bytes"
	"encoding/base64"
	"encoding/xml"
	"errors"
	"io"
	"strings"
	"unicode"
)

// ErrTruncated is a broken or truncated FB2 payload. It must not produce a .none marker.
var ErrTruncated = errors.New("fb2 truncated")

// Book is the cover image and annotation taken from one FB2 document.
type Book struct {
	Cover              []byte
	CoverKind          ImageKind
	HasCover           bool
	UnrecognizedCover  bool
	CoverID            string
	Annotation         string
	Encoding           string
	AnnotationRejected bool
}

// Parse reads a complete FB2 body. The payload is decoded first, then scanned
// with the XML tokenizer so tags are stripped and entities unescaped.
func Parse(raw []byte) (Book, error) {
	if len(bytes.TrimSpace(raw)) == 0 {
		return Book{}, ErrTruncated
	}
	utf, enc := DecodeBody(raw)
	if !bytes.Contains(bytes.ToLower(utf), []byte("</fictionbook>")) {
		return Book{}, ErrTruncated
	}
	dec := xml.NewDecoder(bytes.NewReader(utf))
	dec.Strict = false
	dec.CharsetReader = func(_ string, input io.Reader) (io.Reader, error) {
		return input, nil
	}

	var (
		out          Book
		inTitleInfo  bool
		inAnnotation bool
		inCoverpage  bool
		binaryID     string
		ann          strings.Builder
		bin          strings.Builder
		binaries     = map[string]string{}
	)
	out.Encoding = enc

	for {
		tok, err := dec.Token()
		if err == io.EOF {
			break
		}
		if err != nil {
			return Book{}, ErrTruncated
		}
		switch t := tok.(type) {
		case xml.StartElement:
			switch t.Name.Local {
			case "title-info":
				inTitleInfo = true
			case "annotation":
				if inTitleInfo {
					inAnnotation = true
				}
			case "coverpage":
				if inTitleInfo {
					inCoverpage = true
				}
			case "image":
				if inCoverpage {
					if href := attrLocal(t, "href"); href != "" {
						out.CoverID = strings.TrimPrefix(href, "#")
					}
				}
			case "binary":
				binaryID = attrLocal(t, "id")
				bin.Reset()
			}
		case xml.EndElement:
			switch t.Name.Local {
			case "title-info":
				inTitleInfo = false
				inAnnotation = false
				inCoverpage = false
			case "annotation":
				inAnnotation = false
			case "p", "empty-line":
				if inAnnotation {
					ann.WriteByte('\n')
				}
			case "coverpage":
				inCoverpage = false
			case "binary":
				if binaryID != "" {
					binaries[binaryID] = bin.String()
				}
				binaryID = ""
			}
		case xml.CharData:
			if inAnnotation {
				ann.Write(t)
			}
			if binaryID != "" {
				bin.Write(t)
			}
		}
	}

	rawAnn := normalizeAnnotation(ann.String())
	if rawAnn != "" && !PlausibleText(rawAnn) {
		out.AnnotationRejected = true
		out.Annotation = ""
	} else {
		out.Annotation = rawAnn
	}
	if out.CoverID == "" {
		return out, nil
	}
	payload, ok := binaries[out.CoverID]
	if !ok {
		return out, nil
	}
	img, err := decodeBase64(payload)
	if err != nil {
		return Book{}, ErrTruncated
	}
	kind := DetectImage(img)
	if kind == ImageNone {
		out.UnrecognizedCover = true
		return out, nil
	}
	out.Cover = img
	out.CoverKind = kind
	out.HasCover = true
	return out, nil
}

func attrLocal(el xml.StartElement, local string) string {
	for _, a := range el.Attr {
		if a.Name.Local == local {
			return strings.TrimSpace(a.Value)
		}
	}
	return ""
}

func decodeBase64(s string) ([]byte, error) {
	compact := strings.Map(func(r rune) rune {
		if unicode.IsSpace(r) {
			return -1
		}
		return r
	}, s)
	if compact == "" {
		return nil, ErrTruncated
	}
	out, err := base64.StdEncoding.DecodeString(compact)
	if err != nil {
		return nil, ErrTruncated
	}
	return out, nil
}

func normalizeAnnotation(s string) string {
	lines := strings.Split(s, "\n")
	out := make([]string, 0, len(lines))
	for _, line := range lines {
		line = strings.Join(strings.Fields(line), " ")
		out = append(out, line)
	}
	return strings.TrimSpace(strings.Join(out, "\n"))
}
