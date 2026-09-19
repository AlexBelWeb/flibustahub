package downloads

import (
	"archive/zip"
	"io"
	"os"
	"strconv"
	"strings"

	"github.com/alexbelweb/flibustahub/internal/apperr"
	"github.com/alexbelweb/flibustahub/internal/fb2"
	"github.com/alexbelweb/flibustahub/internal/repositories"
)

// Stream is one ZIP member. The archive handle lives only as long as the stream.
type Stream struct {
	io.ReadCloser
	FileName string
	Size     int64
	Ext      string
}

type zipStream struct {
	rc io.ReadCloser
	zr *zip.ReadCloser
}

func (s *zipStream) Read(p []byte) (int, error) { return s.rc.Read(p) }

func (s *zipStream) Close() error {
	err := s.rc.Close()
	if closeErr := s.zr.Close(); err == nil {
		err = closeErr
	}
	return err
}

func openStream(root string, ed repositories.EditionFile) (*Stream, error) {
	path, err := zipPath(root, ed.ArchiveName)
	if err != nil {
		return nil, err
	}
	if _, err := os.Stat(path); err != nil {
		return nil, apperr.New(apperr.CodeArchiveMissing, map[string]string{"archive": ed.ArchiveName})
	}
	zr, err := zip.OpenReader(path)
	if err != nil {
		return nil, apperr.Wrap(apperr.CodeFB2Unreadable, err, map[string]string{
			"workId": strconv.FormatInt(ed.WorkID, 10),
		})
	}
	entry := fb2.FindEntry(zr.File, ed.FileName, ed.FileExt)
	if entry == nil {
		_ = zr.Close()
		return nil, apperr.Wrap(apperr.CodeFB2Unreadable, fb2.ErrMissingEntry, map[string]string{
			"workId": strconv.FormatInt(ed.WorkID, 10),
		})
	}
	rc, err := entry.Open()
	if err != nil {
		_ = zr.Close()
		return nil, apperr.Wrap(apperr.CodeFB2Unreadable, err, map[string]string{
			"workId": strconv.FormatInt(ed.WorkID, 10),
		})
	}
	ext := strings.TrimPrefix(strings.TrimSpace(ed.FileExt), ".")
	if ext == "" {
		ext = "fb2"
	}
	return &Stream{
		ReadCloser: &zipStream{rc: rc, zr: zr},
		FileName:   ed.FileName,
		Size:       int64(entry.UncompressedSize64),
		Ext:        ext,
	}, nil
}
