package servers

import (
	"net/http"
	"path"
	"strings"
)

func (s *Server) serveHTTP(w http.ResponseWriter, r *http.Request, username string) {
	if strings.HasPrefix(r.URL.Path, "/api") {
		s.serveAPI(w, r, username)
		return
	}

	if s.WebsiteDirectory == "" {
		http.Error(w, "Website not enabled", http.StatusNotFound)
		return
	}

	method := strings.ToUpper(r.Method)
	if method != "GET" && method != "HEAD" {
		http.Error(w, "Only GET and HEAD are allowed", http.StatusBadRequest)
		return
	}

	http.ServeFile(w, r, path.Join(s.WebsiteDirectory, path.Clean(r.URL.Path)))
}
