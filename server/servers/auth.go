package servers

import (
	"crypto/tls"
	"log"
	"net/http"

	"github.com/Doridian/wsvpn/server/authenticators"
)

const rootRoutePreauthorize = "/preauthorize"

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

func (s *Server) handlePreauthorize(w http.ResponseWriter, r *http.Request, tlsState *tls.ConnectionState) {
	if r.Method == http.MethodOptions {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "POST")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Content-Length, Authorization")
		w.Header().Set("Access-Control-Allow-Credentials", "true")
		w.Write([]byte("OK"))
		return
	}

	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	if !s.PreauthorizeEnabled {
		http.Error(w, "Preauthorization is not enabled", http.StatusForbidden)
		return
	}

	authOk, authUsername := s.handleSocketAuth(s.log, w, r, tlsState)
	if !authOk {
		return
	}

}
