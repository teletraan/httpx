package auth

import (
	"errors"
	"fmt"
	"net/http"
)

// Token represents the credentials used to authorize
// the requests to access protected resources on server backend.
type Token interface {
	// Valid return whehter token valid.
	Valid() bool
	// SetAuthorization set http request authorization, may be header or query.
	SetAuthorization(*http.Request)
}

// A TokenSource is anything that can return a token.
type TokenSource interface {
	// Token returns a token or an error.
	// Token must be safe for concurrent use by multiple goroutines.
	// The returned Token must not be modified.
	Token() (Token, error)
}

// TokenAuthTransport is an http.RoundTripper that authenticates all requests
// using HTTP Token Authentication with the provided token.
type TokenAuthTransport struct {
	// Source supplies the token to add to outgoing requests'
	// Authorization headers.
	Source TokenSource

	// Transport is the underlying HTTP transport to use when making requests.
	// It will default to http.DefaultTransport if nil.
	Transport http.RoundTripper
}

// cloneRequest returns a clone of the provided *http.Request.
// The clone is a shallow copy of the struct and its Header map.
func cloneRequest(r *http.Request) *http.Request {
	// shallow copy of the struct
	r2 := new(http.Request)
	*r2 = *r
	// deep copy of the Header
	r2.Header = make(http.Header, len(r.Header))
	for k, s := range r.Header {
		r2.Header[k] = append([]string(nil), s...)
	}
	return r2
}

// RoundTrip implements the RoundTripper interface.
func (t *TokenAuthTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	// To set extra headers, we must make a copy of the Request so
	// that we don't modify the Request we were given. This is required by the
	// specification of http.RoundTripper.
	req2 := cloneRequest(req)
	if t.Source == nil {
		return nil, errors.New("auth: Transport's Source is nil")
	}
	token, err := t.Source.Token()
	if err != nil {
		return nil, fmt.Errorf("roundtrip error: %w", err)
	}
	token.SetAuthorization(req2)
	return t.transport().RoundTrip(req2)
}

// Client returns an *http.Client that makes requests that are authenticated
// using HTTP Basic Authentication.
func (t *TokenAuthTransport) Client() *http.Client {
	return &http.Client{Transport: t}
}

func (t *TokenAuthTransport) transport() http.RoundTripper {
	if t.Transport != nil {
		return t.Transport
	}
	return http.DefaultTransport
}
