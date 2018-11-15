package main

import (
	"net/http"
)

type Authenticator interface {
	Authenticate(r *http.Request, w http.ResponseWriter) bool
}

type AllowAllAuthenticator struct {
}

func (a *AllowAllAuthenticator) Authenticate(r *http.Request, w http.ResponseWriter) bool {
	return true
}
