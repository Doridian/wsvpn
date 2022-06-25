package servers

import (
	"net/http"

	"github.com/Doridian/wsvpn/server/authenticators"
)

func (s *Server) handleSocketAuth(w http.ResponseWriter, r *http.Request) (bool, string) {
	tlsUsername := ""
	if r.TLS != nil && len(r.TLS.PeerCertificates) > 0 {
		tlsUsername = r.TLS.PeerCertificates[0].Subject.CommonName
	}
	authResult, authUsername := s.Authenticator.Authenticate(r, w)
	if authResult != authenticators.AUTH_OK {
		if authResult == authenticators.AUTH_FAILED_DEFAULT {
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
		}
		return false, ""
	}

	if authUsername != "" && tlsUsername != "" && authUsername != tlsUsername {
		http.Error(w, "Mutual TLS CN is not equal authenticator username", http.StatusUnauthorized)
		return false, ""
	}

	if authUsername == "" {
		authUsername = tlsUsername
	}

	return true, authUsername
}
