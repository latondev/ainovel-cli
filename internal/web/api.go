package web

import (
	"net/http"
	"regexp"
)

var slugPattern = regexp.MustCompile(`^[a-z0-9-]+$`)

// Server gom HTTP handler và reader.
type Server struct {
	reader    *Reader
	staticDir string
}

// NewServer tạo server web.
func NewServer(reader *Reader, staticDir string) *Server {
	return &Server{reader: reader, staticDir: staticDir}
}

func (s *Server) handleNovels(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeError(w, "METHOD_NOT_ALLOWED", http.StatusMethodNotAllowed, "chỉ hỗ trợ GET")
		return
	}
	list, err := s.reader.ListNovels()
	if err != nil {
		writeError(w, "INTERNAL_ERROR", http.StatusInternalServerError, err.Error())
		return
	}
	if list == nil {
		list = []NovelSummary{}
	}
	writeJSON(w, http.StatusOK, list)
}

func (s *Server) handleNovelSub(w http.ResponseWriter, r *http.Request, slug, sub string) {
	if !slugPattern.MatchString(slug) {
		writeError(w, "INVALID_SLUG", http.StatusBadRequest, "slug không hợp lệ")
		return
	}
	switch sub {
	case "":
		s.handleNovelDetail(w, r, slug)
	case "premise":
		s.handlePremise(w, r, slug)
	case "outline":
		s.handleOutline(w, r, slug)
	case "characters":
		s.handleCharacters(w, r, slug)
	default:
		writeError(w, "NOT_FOUND", http.StatusNotFound, "endpoint không tồn tại")
	}
}

func (s *Server) handleNovelDetail(w http.ResponseWriter, r *http.Request, slug string) {
	if r.Method != http.MethodGet {
		writeError(w, "METHOD_NOT_ALLOWED", http.StatusMethodNotAllowed, "chỉ hỗ trợ GET")
		return
	}
	detail, err := s.reader.Detail(slug)
	if err != nil {
		if isNotFound(err) {
			writeError(w, "NOVEL_NOT_FOUND", http.StatusNotFound, err.Error())
			return
		}
		writeError(w, "INTERNAL_ERROR", http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, detail)
}

func (s *Server) handlePremise(w http.ResponseWriter, r *http.Request, slug string) {
	if r.Method != http.MethodGet {
		writeError(w, "METHOD_NOT_ALLOWED", http.StatusMethodNotAllowed, "chỉ hỗ trợ GET")
		return
	}
	resp, err := s.reader.Premise(slug)
	if err != nil {
		if isNotFound(err) {
			writeError(w, "NOVEL_NOT_FOUND", http.StatusNotFound, err.Error())
			return
		}
		writeError(w, "INTERNAL_ERROR", http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, resp)
}

func (s *Server) handleOutline(w http.ResponseWriter, r *http.Request, slug string) {
	if r.Method != http.MethodGet {
		writeError(w, "METHOD_NOT_ALLOWED", http.StatusMethodNotAllowed, "chỉ hỗ trợ GET")
		return
	}
	resp, err := s.reader.Outline(slug)
	if err != nil {
		if isNotFound(err) {
			writeError(w, "NOVEL_NOT_FOUND", http.StatusNotFound, err.Error())
			return
		}
		writeError(w, "INTERNAL_ERROR", http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, resp)
}

func (s *Server) handleCharacters(w http.ResponseWriter, r *http.Request, slug string) {
	if r.Method != http.MethodGet {
		writeError(w, "METHOD_NOT_ALLOWED", http.StatusMethodNotAllowed, "chỉ hỗ trợ GET")
		return
	}
	chars, err := s.reader.Characters(slug)
	if err != nil {
		if isNotFound(err) {
			writeError(w, "NOVEL_NOT_FOUND", http.StatusNotFound, err.Error())
			return
		}
		writeError(w, "INTERNAL_ERROR", http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, chars)
}