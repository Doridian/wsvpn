package authenticators

import (
	"net/http"

	"github.com/tg123/go-htpasswd"
)

type HtpasswdAuthenticator struct {
	authFile *htpasswd.File
}

var _ Authenticator = &HtpasswdAuthenticator{}

func (a *HtpasswdAuthenticator) Load(configFile string) (err error) {
	if configFile == "" {
		configFile = "htpasswd"
	}
	a.authFile, err = htpasswd.New(configFile, htpasswd.DefaultSystems, nil)
	return
}

func respondWWWAuthenticateBasic(w http.ResponseWriter) {
	w.Header().Add("WWW-Authenticate", "Basic")
}

func (a *HtpasswdAuthenticator) Authenticate(r *http.Request, w http.ResponseWriter) AuthResult {
	username, password, ok := r.BasicAuth()
	if !ok {
		respondWWWAuthenticateBasic(w)
		return AUTH_FAILED_DEFAULT
	}

	authOk := a.authFile.Match(username, password)
	if !authOk {
		respondWWWAuthenticateBasic(w)
		return AUTH_FAILED_DEFAULT
	}

	return AUTH_OK
}
