package authenticators

import (
	"net/http"
)

type AuthResult int

const (
	AUTH_OK             AuthResult = iota // Authentication succeeded
	AUTH_FAILED_DEFAULT                   // Authentication failed, provide default error
	AUTH_FAILED_CUSTOM                    // Authentication failed, custom response from authenticator
)

type Authenticator interface {
	Load(configFile string) error
	Authenticate(r *http.Request, w http.ResponseWriter) (AuthResult, string)
}
