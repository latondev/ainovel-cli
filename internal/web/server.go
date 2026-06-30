package web

import (
	"net/http"
	"os"
	"path/filepath"
	"strings"
)

// Handler trả về http.Handler gốc.
func (s *Server) Handler() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /api/novels", s.handleNovels)
	mux.HandleFunc("GET /api/novels/{slug}", func(w http.ResponseWriter, r *http.Request) {
		slug := r.PathValue("slug")
		s.handleNovelSub(w, r, slug, "")
	})
	mux.HandleFunc("GET /api/novels/{slug}/{sub}", func(w http.ResponseWriter, r *http.Request) {
		slug := r.PathValue("slug")
		sub := r.PathValue("sub")
		s.handleNovelSub(w, r, slug, sub)
	})
	mux.HandleFunc("GET /api/health", func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
	})

	if dirExists(s.staticDir) {
		mux.Handle("/", s.spaHandler())
	} else {
		mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
			if strings.HasPrefix(r.URL.Path, "/api/") {
				writeError(w, "NOT_FOUND", http.StatusNotFound, "endpoint không tồn tại")
				return
			}
			w.Header().Set("Content-Type", "text/plain; charset=utf-8")
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte("ainovel-web API đang chạy. Chạy npm run build trong web/frontend để có UI.\n"))
		})
	}
	return mux
}

func (s *Server) spaHandler() http.Handler {
	fileServer := http.FileServer(http.Dir(s.staticDir))
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.HasPrefix(r.URL.Path, "/api/") {
			writeError(w, "NOT_FOUND", http.StatusNotFound, "endpoint không tồn tại")
			return
		}
		path := filepath.Join(s.staticDir, filepath.Clean("/"+r.URL.Path))
		if r.URL.Path != "/" && fileExists(path) {
			fileServer.ServeHTTP(w, r)
			return
		}
		http.ServeFile(w, r, filepath.Join(s.staticDir, "index.html"))
	})
}

func dirExists(path string) bool {
	info, err := os.Stat(path)
	return err == nil && info.IsDir()
}

func fileExists(path string) bool {
	info, err := os.Stat(path)
	return err == nil && !info.IsDir()
}