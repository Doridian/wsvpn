package authenticators

import "net/http"

type AllowAllAuthenticator struct {
}

var _ Authenticator = &AllowAllAuthenticator{}

func (a *AllowAllAuthenticator) Load(configFile string) error {
	return nil
}

func (a *AllowAllAuthenticator) Authenticate(r *http.Request, w http.ResponseWriter) AuthResult {
	return AUTH_OK
}
