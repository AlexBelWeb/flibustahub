package httpapi

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"net"
	"net/http"
	"os"
	"runtime"
	"runtime/debug"
	"strconv"
	"strings"
	"sync/atomic"
	"time"

	"github.com/alexbelweb/flibustahub/internal/apperr"
	"github.com/alexbelweb/flibustahub/internal/services/covers"
)

// CoverPriorityHeader carries queue priority. It is not part of the URL: one
// file must have one address, or the WebView caches two copies.
const CoverPriorityHeader = "X-Cover-Priority"

// CoverSource extracts and serves a cached cover. Implemented by app.Service.
type CoverSource interface {
	ServeCover(ctx context.Context, workID int64, prio string) (covers.Hit, error)
}

// Server is the loopback HTTP listener.
type Server struct {
	log    *slog.Logger
	http   *http.Server
	addr   string
	covers atomic.Value // CoverSource
}

func New(log *slog.Logger) *Server {
	if log == nil {
		log = slog.Default()
	}
	s := &Server{log: log}
	mux := http.NewServeMux()
	mux.HandleFunc("GET /healthz", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "text/plain; charset=utf-8")
		_, _ = w.Write([]byte("ok\n"))
	})
	mux.HandleFunc("OPTIONS /media/cover/{workId}", s.handleCoverOptions)
	mux.HandleFunc("GET /media/cover/{workId}", s.handleCover)
	if os.Getenv("FLIBUSTAHUB_MEMSTATS") == "1" {
		mux.HandleFunc("GET /debug/memstats", handleMemStats)
	}
	s.http = &http.Server{
		Handler:           mux,
		ReadHeaderTimeout: 5 * time.Second,
	}
	return s
}

func (s *Server) SetCovers(src CoverSource) {
	s.covers.Store(src)
}

func (s *Server) handleCoverOptions(w http.ResponseWriter, _ *http.Request) {
	setCoverCORS(w)
	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) handleCover(w http.ResponseWriter, r *http.Request) {
	setCoverCORS(w)
	raw, _ := s.covers.Load().(CoverSource)
	if raw == nil {
		http.Error(w, "unavailable", http.StatusServiceUnavailable)
		return
	}
	id, err := strconv.ParseInt(r.PathValue("workId"), 10, 64)
	if err != nil || id <= 0 {
		http.NotFound(w, r)
		return
	}
	hit, err := raw.ServeCover(r.Context(), id, r.Header.Get(CoverPriorityHeader))
	if err != nil && (errors.Is(err, context.Canceled) || errors.Is(r.Context().Err(), context.Canceled)) {
		return
	}
	if code := coverStatus(hit, err); code != http.StatusOK {
		http.Error(w, http.StatusText(code), code)
		return
	}
	w.Header().Set("Cache-Control", "no-cache")
	if hit.MIME != "" {
		w.Header().Set("Content-Type", hit.MIME)
	}
	http.ServeFile(w, r, hit.Path)
}

func handleMemStats(w http.ResponseWriter, r *http.Request) {
	setCoverCORS(w)
	freed := r.URL.Query().Get("free") == "1"
	if freed {
		debug.FreeOSMemory()
	}
	var m runtime.MemStats
	runtime.ReadMemStats(&m)
	out := map[string]any{
		"alloc":        m.Alloc,
		"heapAlloc":    m.HeapAlloc,
		"heapInuse":    m.HeapInuse,
		"heapIdle":     m.HeapIdle,
		"heapReleased": m.HeapReleased,
		"heapSys":      m.HeapSys,
		"sys":          m.Sys,
		"stackSys":     m.StackSys,
		"numGC":        m.NumGC,
		"free":         freed,
	}
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	_ = json.NewEncoder(w).Encode(out)
}

func setCoverCORS(w http.ResponseWriter) {
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Headers", CoverPriorityHeader)
	w.Header().Set("Access-Control-Allow-Methods", "GET, OPTIONS")
}

func coverStatus(hit covers.Hit, err error) int {
	if err == nil {
		if hit.Found && !hit.None && !hit.Missing && hit.Path != "" {
			return http.StatusOK
		}
		if hit.None {
			return http.StatusNotFound
		}
		if hit.Missing {
			return http.StatusServiceUnavailable
		}
		return http.StatusNotFound
	}
	switch apperr.As(err).Code {
	case apperr.CodeNotFound:
		return http.StatusNotFound
	case apperr.CodeLibraryOffline, apperr.CodeArchiveMissing, apperr.CodeFB2Unreadable:
		return http.StatusServiceUnavailable
	default:
		return http.StatusServiceUnavailable
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
	err := s.http.Shutdown(ctx)
	if errors.Is(err, http.ErrServerClosed) {
		return nil
	}
	return err
}

func isAddrInUse(err error) bool {
	msg := strings.ToLower(err.Error())
	return strings.Contains(msg, "address already in use") ||
		strings.Contains(msg, "only one usage of each socket address")
}
