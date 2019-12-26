package auth

import (
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"
)

// JWTToken represents jwt token.
type JWTToken struct {
	token     string
	expiredAt int64
}

// NewJWTToken create a jwt token with token string.
func NewJWTToken(token string) (*JWTToken, error) {
	t := &JWTToken{token: token}
	if err := t.SetExpireTime(); err != nil {
		return nil, fmt.Errorf("new jwt token error: %w", err)
	}
	return t, nil
}

// Valid return whehter token valid.
func (t *JWTToken) Valid() bool {
	return t != nil && t.token != "" && !t.almostExpired()
}

// almostExpired reports whether the token is expired after 10s.
// t must be non-nil.
func (t *JWTToken) almostExpired() bool {
	if t.expiredAt == 0 {
		return true
	}
	return (time.Now().Unix() + 10) > t.expiredAt
}

// SetAuthorization set http request Authorization header.
func (t *JWTToken) SetAuthorization(req *http.Request) {
	req.Header.Set("Authorization", "Token "+t.token)
}

// SetExpireTime set expire time.
func (t *JWTToken) SetExpireTime() error {
	parts := strings.Split(t.token, ".")
	if len(parts) != 3 {
		return errors.New("need jwt token")
	}
	payload := parts[1]
	if l := len(payload) % 4; l > 0 {
		payload += strings.Repeat("=", 4-l)
	}
	b, err := base64.URLEncoding.DecodeString(payload)
	if err != nil {
		return fmt.Errorf("decode jwt token error: %w", err)
	}
	var m map[string]interface{}
	if err := json.Unmarshal(b, &m); err != nil {
		return fmt.Errorf("unmarshal decode jwt token error: %w", err)
	}
	if ts, ok := m["exp"]; ok {
		t.expiredAt = int64(ts.(float64))
	}
	return nil
}
