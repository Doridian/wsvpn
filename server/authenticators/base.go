package authenticators

import (
	"net/http"
)

type AuthResult int

const (
	AuthOk            AuthResult = iota // Authentication succeeded
	AuthFailedDefault                   // Authentication failed, provide default error
	AuthFailedCustom                    // Authentication failed, custom response from authenticator
)

type Authenticator interface {
	Load(configFile string) error
	Authenticate(r *http.Request, w http.ResponseWriter) (AuthResult, string)
}
