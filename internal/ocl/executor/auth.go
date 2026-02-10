package executor

import (
	"encoding/base64"
	"net/http"
)

// AuthType represents authentication method
type AuthType string

const (
	AuthNone   AuthType = "none"
	AuthBasic  AuthType = "basic"
	AuthBearer AuthType = "bearer"
	AuthAPIKey AuthType = "apikey"
)

// ServiceAuth holds authentication config for a service
type ServiceAuth struct {
	Type     AuthType
	Username string // for basic auth
	Password string // for basic auth
	Token    string // for bearer/apikey
	Header   string // header name for apikey (default: X-API-Key)
}

// serviceAuths maps service names to their auth config
var defaultAuths = map[string]*ServiceAuth{}

// SetServiceAuth configures authentication for a service
func (e *Executor) SetServiceAuth(serviceName string, auth *ServiceAuth) {
	if e.auths == nil {
		e.auths = make(map[string]*ServiceAuth)
	}
	e.auths[serviceName] = auth
}

// ApplyAuth applies authentication to a request
func (e *Executor) ApplyAuth(req *http.Request, serviceName string) {
	if e.auths == nil {
		return
	}

	auth, ok := e.auths[serviceName]
	if !ok {
		return
	}

	switch auth.Type {
	case AuthBasic:
		credentials := base64.StdEncoding.EncodeToString(
			[]byte(auth.Username + ":" + auth.Password))
		req.Header.Set("Authorization", "Basic "+credentials)

	case AuthBearer:
		req.Header.Set("Authorization", "Bearer "+auth.Token)

	case AuthAPIKey:
		header := auth.Header
		if header == "" {
			header = "X-API-Key"
		}
		req.Header.Set(header, auth.Token)
	}
}
