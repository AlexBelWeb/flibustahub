package testdata

import (
	"archive/zip"
	"bytes"
	"fmt"
	"time"

	"golang.org/x/text/encoding/charmap"
)

const eot = 0x04

func cp1251(s string) []byte {
	out, err := charmap.Windows1251.NewEncoder().Bytes([]byte(s))
	if err != nil {
		panic("testdata: cp1251 encode " + s + ": " + err.Error())
	}
	return out
}

// Record builds one INP line: 14 fields, trailing 0x04, optional CRLF.
func Record(fields [14]string, crlf bool) []byte {
	var b bytes.Buffer
	for i, f := range fields {
		if i > 0 {
			b.WriteByte(eot)
		}
		b.Write(cp1251(f))
	}
	b.WriteByte(eot)
	if crlf {
		b.WriteString("\r\n")
	}
	return b.Bytes()
}

func field(authors, genres, title, series, serno, file, size, libid, del, ext, date, lang, librate, keywords string) [14]string {
	return [14]string{authors, genres, title, series, serno, file, size, libid, del, ext, date, lang, librate, keywords}
}

// BrokenRecord is a truncated line with fewer than 14 fields and a trailing 0x04.
func BrokenRecord(parts ...string) []byte {
	var b bytes.Buffer
	for i, p := range parts {
		if i > 0 {
			b.WriteByte(eot)
		}
		b.Write(cp1251(p))
	}
	b.WriteByte(eot)
	b.WriteString("\r\n")
	return b.Bytes()
}

func addZip(w *zip.Writer, name string, body []byte, t time.Time) error {
	h := &zip.FileHeader{Name: name, Method: zip.Deflate, Modified: t}
	f, err := w.CreateHeader(h)
	if err != nil {
		return err
	}
	_, err = f.Write(body)
	return err
}

func writeZip(files []zipEntry, t time.Time) []byte {
	var buf bytes.Buffer
	w := zip.NewWriter(&buf)
	for _, e := range files {
		if err := addZip(w, e.name, e.body, t); err != nil {
			panic(err)
		}
	}
	if err := w.Close(); err != nil {
		panic(err)
	}
	return buf.Bytes()
}

type zipEntry struct {
	name string
	body []byte
}

func emptyZip() []byte {
	var buf bytes.Buffer
	w := zip.NewWriter(&buf)
	if err := w.Close(); err != nil {
		panic(err)
	}
	return buf.Bytes()
}

func nAuthors(n int) string {
	var b bytes.Buffer
	for i := 0; i < n; i++ {
		fmt.Fprintf(&b, "Автор%02d,Имя,Отчество:", i+1)
	}
	return b.String()
}
