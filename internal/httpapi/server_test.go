package httpapi

import (
	"context"
	"io"
	"log/slog"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/alexbelweb/flibustahub/internal/apperr"
	"github.com/alexbelweb/flibustahub/internal/services/covers"
)

func TestHealthz(t *testing.T) {
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	port := ln.Addr().(*net.TCPAddr).Port
	_ = ln.Close()

	srv := New(slog.New(slog.DiscardHandler))
	if err := srv.Start("127.0.0.1", port); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		ctx, cancel := context.WithTimeout(context.Background(), time.Second)
		defer cancel()
		_ = srv.Shutdown(ctx)
	})

	var last error
	for i := 0; i < 20; i++ {
		resp, err := http.Get("http://" + srv.Addr() + "/healthz")
		if err == nil {
			body, _ := io.ReadAll(resp.Body)
			_ = resp.Body.Close()
			if resp.StatusCode != http.StatusOK || string(body) != "ok\n" {
				t.Fatalf("status=%d body=%q", resp.StatusCode, body)
			}
			return
		}
		last = err
		time.Sleep(20 * time.Millisecond)
	}
	t.Fatalf("healthz never answered: %v", last)
}

func TestPortInUse(t *testing.T) {
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = ln.Close() }()
	port := ln.Addr().(*net.TCPAddr).Port
	srv := New(slog.New(slog.DiscardHandler))
	err = srv.Start("127.0.0.1", port)
	if err == nil {
		t.Fatal("expected port-in-use error")
	}
	if apperr.As(err).Code != apperr.CodeHTTPPortInUse {
		t.Fatalf("unexpected error %v", err)
	}
}

type stubCover struct {
	hit covers.Hit
	err error
}

func (s stubCover) ServeCover(context.Context, int64, string) (covers.Hit, error) {
	return s.hit, s.err
}

func TestCoverEndpoint(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "1.jpg")
	body := []byte{0xff, 0xd8, 0xff, 0xe0, 0x00, 0x10, 'J', 'F', 'I', 'F'}
	if err := os.WriteFile(path, body, 0o644); err != nil {
		t.Fatal(err)
	}
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	port := ln.Addr().(*net.TCPAddr).Port
	_ = ln.Close()
	srv := New(slog.New(slog.DiscardHandler))
	srv.SetCovers(stubCover{hit: covers.Hit{Found: true, Path: path, MIME: "image/jpeg"}})
	if err := srv.Start("127.0.0.1", port); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		ctx, cancel := context.WithTimeout(context.Background(), time.Second)
		defer cancel()
		_ = srv.Shutdown(ctx)
	})

	resp, err := http.Get("http://" + srv.Addr() + "/media/cover/1")
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status %d", resp.StatusCode)
	}
	if resp.Header.Get("Cache-Control") != "no-cache" {
		t.Fatalf("Cache-Control=%q", resp.Header.Get("Cache-Control"))
	}
	if resp.Header.Get("Last-Modified") == "" {
		t.Fatal("missing Last-Modified")
	}
	got, _ := io.ReadAll(resp.Body)
	if string(got) != string(body) {
		t.Fatalf("body %q", got)
	}

	req, err := http.NewRequest(http.MethodGet, "http://"+srv.Addr()+"/media/cover/1", nil)
	if err != nil {
		t.Fatal(err)
	}
	req.Header.Set(CoverPriorityHeader, "open")
	prioResp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	_ = prioResp.Body.Close()
	if prioResp.StatusCode != http.StatusOK {
		t.Fatalf("header prio status %d", prioResp.StatusCode)
	}

	srv.SetCovers(stubCover{hit: covers.Hit{Found: true, None: true}})
	resp2, err := http.Get("http://" + srv.Addr() + "/media/cover/2")
	if err != nil {
		t.Fatal(err)
	}
	_ = resp2.Body.Close()
	if resp2.StatusCode != http.StatusNotFound {
		t.Fatalf("none status %d", resp2.StatusCode)
	}

	srv.SetCovers(stubCover{err: apperr.New(apperr.CodeLibraryOffline, nil)})
	resp3, err := http.Get("http://" + srv.Addr() + "/media/cover/3")
	if err != nil {
		t.Fatal(err)
	}
	_ = resp3.Body.Close()
	if resp3.StatusCode != http.StatusServiceUnavailable {
		t.Fatalf("offline status %d", resp3.StatusCode)
	}

	srv.SetCovers(stubCover{err: apperr.New(apperr.CodeArchiveMissing, map[string]string{"archive": "a.zip"})})
	resp4, err := http.Get("http://" + srv.Addr() + "/media/cover/4")
	if err != nil {
		t.Fatal(err)
	}
	_ = resp4.Body.Close()
	if resp4.StatusCode != http.StatusServiceUnavailable {
		t.Fatalf("archive status %d", resp4.StatusCode)
	}
}
