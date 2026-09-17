// Package httpapi hosts the local HTTP server (media + OPDS).
// Slice 1 only binds loopback and exposes a health check; routes are added later.
package httpapi

import (
	"context"
	"errors"
	"log/slog"
	"net"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/alexbelweb/flibustahub/internal/apperr"
)

// Server is the loopback HTTP listener.
type Server struct {
	log  *slog.Logger
	http *http.Server
	addr string
}

func New(log *slog.Logger) *Server {
	if log == nil {
		log = slog.Default()
	}
	mux := http.NewServeMux()
	mux.HandleFunc("GET /healthz", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "text/plain; charset=utf-8")
		_, _ = w.Write([]byte("ok\n"))
	})
	return &Server{
		log: log,
		http: &http.Server{
			Handler:           mux,
			ReadHeaderTimeout: 5 * time.Second,
		},
	}
}

// Start binds host:port. A busy port is a typed error; the rest of the app keeps running.
func (s *Server) Start(host string, port int) error {
	if host == "" {
		host = "127.0.0.1"
	}
	ln, err := net.Listen("tcp", net.JoinHostPort(host, strconv.Itoa(port)))
	if err != nil {
		if isAddrInUse(err) {
			return apperr.New(apperr.CodeHTTPPortInUse, map[string]string{
				"port": strconv.Itoa(port),
			})
		}
		return apperr.Wrap(apperr.CodeInternal, err, nil)
	}
	s.addr = ln.Addr().String()
	s.log.Info("http: listening", "addr", s.addr)
	go func() {
		if serveErr := s.http.Serve(ln); serveErr != nil && !errors.Is(serveErr, http.ErrServerClosed) {
			s.log.Error("http: server stopped", "err", serveErr)
		}
	}()
	return nil
}

func (s *Server) Addr() string {
	return s.addr
}

func (s *Server) Shutdown(ctx context.Context) error {
	if s.http == nil {
		return nil
	}
	return s.http.Shutdown(ctx)
}

func isAddrInUse(err error) bool {
	msg := strings.ToLower(err.Error())
	return strings.Contains(msg, "address already in use") ||
		strings.Contains(msg, "only one usage of each socket address")
}
