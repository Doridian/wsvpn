package servers

import (
	"crypto/tls"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/Doridian/wsvpn/server/authenticators"
	"github.com/golang-jwt/jwt/v4"
)

const rootRoutePreauthorize = "/preauthorize"
const prefixRoutePreauthorize = "/preauthorize/"

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

type preauthorizeResponse struct {
	Success bool   `json:"success"`
	Token   string `json:"token"`
}

func sendPreauthorizedResponse(w http.ResponseWriter, r *http.Request, resp *preauthorizeResponse) {
	d, err := json.Marshal(resp)
	if err != nil {
		http.Error(w, "Failed to marshal response", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(d)
}

func (s *Server) handlePreauthorizeToken(logger *log.Logger, w http.ResponseWriter, r *http.Request, token string) (bool, string) {
	jwtToken, err := jwt.Parse(token, func(token *jwt.Token) (interface{}, error) {
		if token.Method.Alg() != jwt.SigningMethodHS256.Alg() {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}

		return s.PreauthorizeSecret, nil
	})

	if err != nil {
		logger.Printf("JWT parsing failed: %v", err)
		http.Error(w, "Failed to parse JWT", http.StatusBadRequest)
		return false, ""
	}

	claims, ok := jwtToken.Claims.(jwt.MapClaims)
	if !ok || !jwtToken.Valid {
		logger.Printf("JWT reading failed: %v", err)
		http.Error(w, "Failed to reading JWT", http.StatusBadRequest)
	}
	return true, claims["sub"].(string)
}

func (s *Server) handlePreauthorize(logger *log.Logger, w http.ResponseWriter, r *http.Request, tlsState *tls.ConnectionState) {
	if r.Method == http.MethodOptions {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "POST")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Content-Length, Authorization")
		w.Header().Set("Access-Control-Allow-Credentials", "true")
		_, _ = w.Write([]byte("OK"))
		return
	}

	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	if len(s.PreauthorizeSecret) == 0 {
		http.Error(w, "Preauthorization is not enabled", http.StatusForbidden)
		return
	}

	authOk, authUsername := s.handleSocketAuth(s.log, w, r, tlsState)
	if !authOk {
		return
	}

	if authUsername == "" {
		sendPreauthorizedResponse(w, r, &preauthorizeResponse{
			Success: true,
			Token:   "",
		})
		return
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"sub": authUsername,
		"exp": time.Now().UTC().Add(time.Minute * 1).Unix(),
	})

	signedToken, err := token.SignedString(s.PreauthorizeSecret)
	if err != nil {
		logger.Printf("JWT signing failed: %v", err)
		http.Error(w, "Failed to sign JWT", http.StatusInternalServerError)
		return
	}

	sendPreauthorizedResponse(w, r, &preauthorizeResponse{
		Success: true,
		Token:   signedToken,
	})
}
