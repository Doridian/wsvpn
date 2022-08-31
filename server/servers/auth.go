package servers

import (
	"crypto/tls"
	"log"
	"net/http"

	"github.com/Doridian/wsvpn/server/authenticators"
)

func (s *Server) handleSocketAuth(logger *log.Logger, w http.ResponseWriter, r *http.Request, tlsState *tls.ConnectionState) (bool, string) {
	tlsUsername := ""
	if tlsState != nil && len(tlsState.PeerCertificates) > 0 {
		tlsUsername = tlsState.PeerCertificates[0].Subject.CommonName
	}

	if s.TLSConfig != nil && s.TLSConfig.ClientAuth == tls.RequireAndVerifyClientCert && tlsUsername == "" {
		http.Error(w, "Mutual TLS required but no certificate given", http.StatusUnauthorized)
		logger.Printf("Mutual TLS required but no certificate given")
		return false, ""
	}

	authResult, authUsername := s.Authenticator.Authenticate(r, w)
	if authResult != authenticators.AuthOk {
		if authResult == authenticators.AuthFailedDefault {
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
		}
		logger.Printf("Client failed authenticator challenge")
		return false, ""
	}

	if authUsername != "" && tlsUsername != "" && authUsername != tlsUsername {
		http.Error(w, "Mismatch between MTLS CN and authenticator username", http.StatusUnauthorized)
		logger.Printf("Mismatch between MTLS CN and authenticator username")
		return false, ""
	}

	if authUsername == "" {
		return true, tlsUsername
	}

	return true, authUsername
}
