package fb2

import (
	"bytes"
	"encoding/base64"
	"errors"
	"testing"
	"unicode/utf16"

	"golang.org/x/text/encoding/charmap"
)

var jpegBytes = []byte{0xff, 0xd8, 0xff, 0xe0, 0x00, 0x10, 'J', 'F', 'I', 'F'}
var pngBytes = []byte{0x89, 0x50, 0x4e, 0x47, 0x0d, 0x0a, 0x1a, 0x0a, 0x00}
var gifBytes = []byte("GIF89a\x01\x00\x01\x00")
var webpBytes = []byte{'R', 'I', 'F', 'F', 0x10, 0, 0, 0, 'W', 'E', 'B', 'P'}

func TestDetectImage(t *testing.T) {
	cases := []struct {
		b    []byte
		kind ImageKind
		ext  string
	}{
		{jpegBytes, ImageJPEG, ".jpg"},
		{pngBytes, ImagePNG, ".png"},
		{gifBytes, ImageGIF, ".gif"},
		{webpBytes, ImageWebP, ".webp"},
		{[]byte("not-an-image"), ImageNone, ""},
		{[]byte{}, ImageNone, ""},
	}
	for _, tc := range cases {
		got := DetectImage(tc.b)
		if got != tc.kind || got.Ext() != tc.ext {
			t.Fatalf("kind=%v ext=%q want %v %q", got, got.Ext(), tc.kind, tc.ext)
		}
	}
}

func fb2Doc(encoding, hrefAttr, contentType string, cover []byte, annotation string) []byte {
	b64 := base64.StdEncoding.EncodeToString(cover)
	var href string
	switch hrefAttr {
	case "l":
		href = `l:href="#cover.jpg"`
	case "xlink":
		href = `xlink:href="#cover.jpg"`
	default:
		href = `href="#cover.jpg"`
	}
	body := `<?xml version="1.0" encoding="` + encoding + `"?>` +
		`<FictionBook xmlns="http://www.gribuser.ru/xml/fictionbook/2.0" xmlns:l="http://www.w3.org/1999/xlink">` +
		`<description><title-info>` +
		`<annotation><p>` + annotation + `</p></annotation>` +
		`<coverpage><image ` + href + `/></coverpage>` +
		`</title-info></description>` +
		`<binary id="cover.jpg" content-type="` + contentType + `">` + b64 + `</binary>` +
		`</FictionBook>`
	return []byte(body)
}

func TestParseCoverAndAnnotation(t *testing.T) {
	raw := fb2Doc("utf-8", "l", "image/png", jpegBytes, "Hello &amp; world")
	got, err := Parse(raw)
	if err != nil {
		t.Fatal(err)
	}
	if !got.HasCover || got.CoverKind != ImageJPEG {
		t.Fatalf("cover=%v kind=%v (content-type lied)", got.HasCover, got.CoverKind)
	}
	if !bytes.Equal(got.Cover, jpegBytes) {
		t.Fatalf("cover bytes changed")
	}
	if got.Annotation != "Hello & world" {
		t.Fatalf("annotation = %q", got.Annotation)
	}
}

func TestParseXLinkAndPlainHref(t *testing.T) {
	for _, attr := range []string{"xlink", "href"} {
		got, err := Parse(fb2Doc("utf-8", attr, "image/jpeg", pngBytes, "A"))
		if err != nil {
			t.Fatal(err)
		}
		if got.CoverKind != ImagePNG {
			t.Fatalf("%s: kind=%v", attr, got.CoverKind)
		}
	}
}

func TestParseGIFWebPKept(t *testing.T) {
	gif, err := Parse(fb2Doc("utf-8", "l", "image/jpeg", gifBytes, "g"))
	if err != nil {
		t.Fatal(err)
	}
	if gif.CoverKind != ImageGIF {
		t.Fatalf("gif kind=%v", gif.CoverKind)
	}
	webp, err := Parse(fb2Doc("utf-8", "l", "image/jpeg", webpBytes, "w"))
	if err != nil {
		t.Fatal(err)
	}
	if webp.CoverKind != ImageWebP {
		t.Fatalf("webp kind=%v", webp.CoverKind)
	}
}

func TestUnknownMagicIsNoCover(t *testing.T) {
	got, err := Parse(fb2Doc("utf-8", "l", "image/jpeg", []byte("XXXX-not-image"), "a"))
	if err != nil {
		t.Fatal(err)
	}
	if got.HasCover {
		t.Fatal("garbage must not become a cover")
	}
	if !got.UnrecognizedCover {
		t.Fatal("unrecognized magic must be reported")
	}
}

func TestWindows1251Annotation(t *testing.T) {
	xmlUTF := `<?xml version="1.0" encoding="windows-1251"?>` +
		`<FictionBook><description><title-info>` +
		`<annotation><p>Ёлка</p></annotation>` +
		`</title-info></description></FictionBook>`
	raw, err := charmap.Windows1251.NewEncoder().Bytes([]byte(xmlUTF))
	if err != nil {
		t.Fatal(err)
	}
	got, err := Parse(raw)
	if err != nil {
		t.Fatal(err)
	}
	if got.Annotation != "Ёлка" {
		t.Fatalf("annotation = %q", got.Annotation)
	}
}

func TestUTF16LEBOM(t *testing.T) {
	xmlUTF := `<?xml version="1.0"?><FictionBook><description><title-info>` +
		`<annotation><p>Ёлка</p></annotation></title-info></description></FictionBook>`
	u := utf16.Encode([]rune(xmlUTF))
	raw := []byte{0xff, 0xfe}
	for _, r := range u {
		raw = append(raw, byte(r), byte(r>>8))
	}
	got, err := Parse(raw)
	if err != nil {
		t.Fatal(err)
	}
	if got.Annotation != "Ёлка" {
		t.Fatalf("annotation = %q", got.Annotation)
	}
}

func TestParseParagraphBreaks(t *testing.T) {
	raw := []byte(`<?xml version="1.0" encoding="utf-8"?><FictionBook><description><title-info>` +
		`<annotation><p>First</p><empty-line/><p>Second</p></annotation>` +
		`</title-info></description></FictionBook>`)
	got, err := Parse(raw)
	if err != nil {
		t.Fatal(err)
	}
	if got.Annotation != "First\n\nSecond" {
		t.Fatalf("annotation = %q", got.Annotation)
	}
}

func TestCP866Annotation(t *testing.T) {
	xmlUTF := `<?xml version="1.0" encoding="cp866"?>` +
		`<FictionBook><description><title-info>` +
		`<annotation><p>Ёлка</p></annotation>` +
		`</title-info></description></FictionBook>`
	raw, err := charmap.CodePage866.NewEncoder().Bytes([]byte(xmlUTF))
	if err != nil {
		t.Fatal(err)
	}
	got, err := Parse(raw)
	if err != nil {
		t.Fatal(err)
	}
	if got.Annotation != "Ёлка" {
		t.Fatalf("annotation = %q encoding=%s", got.Annotation, got.Encoding)
	}
}

func TestUTF16LENoBOM(t *testing.T) {
	xmlUTF := `<?xml version="1.0"?><FictionBook><description><title-info>` +
		`<annotation><p>Ёлка</p></annotation></title-info></description></FictionBook>`
	u := utf16.Encode([]rune(xmlUTF))
	raw := make([]byte, 0, len(u)*2)
	for _, r := range u {
		raw = append(raw, byte(r), byte(r>>8))
	}
	got, err := Parse(raw)
	if err != nil {
		t.Fatal(err)
	}
	if got.Annotation != "Ёлка" {
		t.Fatalf("annotation = %q encoding=%s", got.Annotation, got.Encoding)
	}
}

func TestImplausibleAnnotationDropped(t *testing.T) {
	raw := []byte(`<?xml version="1.0" encoding="utf-8"?><FictionBook><description><title-info>` +
		`<annotation><p>╔══╗ ░▒▓│┤ © ¤</p></annotation></title-info></description></FictionBook>`)
	got, err := Parse(raw)
	if err != nil {
		t.Fatal(err)
	}
	if got.Annotation != "" {
		t.Fatalf("wanted empty, got %q", got.Annotation)
	}
	if !got.AnnotationRejected {
		t.Fatal("gate must reject the text")
	}
}

func TestUTF8DeclaredNotStolenByCP866(t *testing.T) {
	raw := []byte(`<?xml version="1.0" encoding="utf-8"?><FictionBook><description><title-info>` +
		`<annotation><p>Он говорил, что я его пара, его Истинная.</p></annotation>` +
		`</title-info></description></FictionBook>`)
	got, err := Parse(raw)
	if err != nil {
		t.Fatal(err)
	}
	if got.Encoding != "utf-8" {
		t.Fatalf("encoding = %s", got.Encoding)
	}
	if got.Annotation != "Он говорил, что я его пара, его Истинная." {
		t.Fatalf("annotation = %q", got.Annotation)
	}
}

func TestCP866MojibakeOfUTF8Rejected(t *testing.T) {
	utf := []byte("Он говорил, что я его пара, его Истинная. Повесть, основанная на реальных событиях.")
	mojibake, err := charmap.CodePage866.NewDecoder().Bytes(utf)
	if err != nil {
		t.Fatal(err)
	}
	s := string(mojibake)
	if PlausibleText(s) {
		t.Fatalf("cp866-of-utf8 must fail the gate, score=%.4f text=%q", TextScore(s), s)
	}
}

func TestUTF8BeatsDeclared1251(t *testing.T) {
	raw := []byte(`<?xml version="1.0" encoding="windows-1251"?><FictionBook><description><title-info>` +
		`<annotation><p>Привет, мир. Это настоящий текст.</p></annotation>` +
		`</title-info></description></FictionBook>`)
	got, err := Parse(raw)
	if err != nil {
		t.Fatal(err)
	}
	if got.Annotation != "Привет, мир. Это настоящий текст." {
		t.Fatalf("annotation = %q encoding=%s", got.Annotation, got.Encoding)
	}
	if got.Encoding != "utf-8" {
		t.Fatalf("encoding = %s", got.Encoding)
	}
}

func TestReplacementAnnotationDropped(t *testing.T) {
	raw := []byte(`<?xml version="1.0" encoding="utf-8"?><FictionBook><description><title-info>` +
		`<annotation><p>bad &#xfffd; text</p></annotation></title-info></description></FictionBook>`)
	got, err := Parse(raw)
	if err != nil {
		t.Fatal(err)
	}
	if got.Annotation != "" {
		t.Fatalf("wanted empty, got %q", got.Annotation)
	}
}

func TestTruncatedFB2(t *testing.T) {
	_, err := Parse([]byte("<FictionBook><description>"))
	if !errors.Is(err, ErrTruncated) {
		t.Fatalf("err=%v", err)
	}
}

func TestCp1251AliasDeclared(t *testing.T) {
	xmlUTF := `<?xml version="1.0" encoding="cp1251"?>` +
		`<FictionBook><description><title-info>` +
		`<annotation><p>Сад</p></annotation>` +
		`</title-info></description></FictionBook>`
	raw, err := charmap.Windows1251.NewEncoder().Bytes([]byte(xmlUTF))
	if err != nil {
		t.Fatal(err)
	}
	got, err := Parse(raw)
	if err != nil {
		t.Fatal(err)
	}
	if got.Annotation != "Сад" {
		t.Fatalf("annotation = %q", got.Annotation)
	}
}
