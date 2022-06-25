package servers

import (
	"log"
	"net/http"

	"github.com/Doridian/wsvpn/server/authenticators"
)

func (s *Server) handleSocketAuth(logger *log.Logger, w http.ResponseWriter, r *http.Request) bool {
	tlsUsername := ""
	if r.TLS != nil && len(r.TLS.PeerCertificates) > 0 {
		tlsUsername = r.TLS.PeerCertificates[0].Subject.CommonName
	}
	authResult, authUsername := s.Authenticator.Authenticate(r, w)
	if authResult != authenticators.AUTH_OK {
		if authResult == authenticators.AUTH_FAILED_DEFAULT {
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
		}
		logger.Printf("Client failed authenticator challenge")
		return false
	}

	if authUsername != "" && tlsUsername != "" && authUsername != tlsUsername {
		http.Error(w, "Mutual TLS CN is not equal authenticator username", http.StatusUnauthorized)
		logger.Printf("Client mismatch between MTLS CN and authenticator username")
		return false
	}

	return true
}
