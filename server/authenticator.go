package main

import (
	"net/http"
	"github.com/tg123/go-htpasswd"
)

type AuthResult int

const (
	AUTH_OK             AuthResult = iota // Authentication succeeded
	AUTH_FAILED_DEFAULT                   // Authentication failed, provide default error
	AUTH_FAILED_CUSTOM                    // Authentication failed, custom response from authenticator
)

type Authenticator interface {
	Load() error
	Authenticate(r *http.Request, w http.ResponseWriter) AuthResult
}

type AllowAllAuthenticator struct {
}

type HtpasswdAuthenticator struct {
	authFile *htpasswd.File
}

func (a *AllowAllAuthenticator) Load() error {
	return nil
}

func (a *AllowAllAuthenticator) Authenticate(r *http.Request, w http.ResponseWriter) AuthResult {
	return AUTH_OK
}

func (a *HtpasswdAuthenticator) Load() (err error) {
	a.authFile, err = htpasswd.New("./htpasswd", htpasswd.DefaultSystems, nil)
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
