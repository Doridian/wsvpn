package servers

import (
	"net/http"
	"path"
	"strings"
)

func (s *Server) serveStatic(w http.ResponseWriter, r *http.Request) {
	if strings.HasPrefix(r.URL.Path, "/api") {
		http.Error(w, "API not implemented, yet", http.StatusNotFound)
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
