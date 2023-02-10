package authenticators

import (
	"context"
	"log"
	"net/http"
	"os"

	"gopkg.in/yaml.v3"

	"layeh.com/radius"
	"layeh.com/radius/rfc2865"
)

type RadiusAuthenticator struct {
	Server string `yaml:"server"`
	Secret string `yaml:"secret"`
}

var _ Authenticator = &RadiusAuthenticator{}

func (a *RadiusAuthenticator) Load(configFile string) error {
	fh, err := os.Open(configFile)
	if err != nil {
		return err
	}
	defer fh.Close()
	return yaml.NewDecoder(fh).Decode(a)
}

func (a *RadiusAuthenticator) Authenticate(r *http.Request, w http.ResponseWriter) (AuthResult, string) {
	username, password, ok := r.BasicAuth()
	if !ok {
		respondWWWAuthenticateBasic(w)
		return AuthFailedDefault, ""
	}

	packet := radius.New(radius.CodeAccessRequest, []byte(a.Secret))
	err := rfc2865.UserName_SetString(packet, username)
	if err != nil {
		respondWWWAuthenticateBasic(w)
		return AuthFailedDefault, ""
	}
	err = rfc2865.UserPassword_SetString(packet, password)
	if err != nil {
		respondWWWAuthenticateBasic(w)
		return AuthFailedDefault, ""
	}
	response, err := radius.Exchange(context.Background(), packet, a.Server)
	if err != nil {
		log.Printf("radius exchange error: %v", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return AuthFailedCustom, ""
	}

	if response.Code == radius.CodeAccessChallenge {
		http.Error(w, "Access challenge not implemented", http.StatusUnauthorized)
		return AuthFailedCustom, ""
	}

	if response.Code != radius.CodeAccessAccept {
		respondWWWAuthenticateBasic(w)
		return AuthFailedDefault, ""
	}

	return AuthOk, username
}
