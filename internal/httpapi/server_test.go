package httpapi

import (
	"context"
	"io"
	"log/slog"
	"net"
	"net/http"
	"testing"
	"time"

	"github.com/alexbelweb/flibustahub/internal/apperr"
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
