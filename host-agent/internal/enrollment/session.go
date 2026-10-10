package enrollment

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"net/http"
	"strings"
	"sync"
	"time"
)

// SessionData is private on-disk state, never a command response or log object.
// A live import requires a separately approved browser/SSO handoff; no password login.
type SessionData struct {
	Origin         string `json:"origin"`
	OrganizationID string `json:"organizationId"`
	Version        string `json:"version"`
	AccessToken    string `json:"accessToken"`
	RefreshToken   string `json:"refreshToken"`
	Pending        bool   `json:"refreshPending"`
	Expired        bool   `json:"expired"`
}
type sessionStore interface{ Save(string, any) error }
type RefreshResult struct {
	Token          string `json:"token"`
	RefreshToken   string `json:"refreshToken"`
	OrganizationID string `json:"organizationId"`
}
type Refresher func(context.Context, string) (RefreshResult, error)
type Session struct {
	mu      sync.Mutex
	data    SessionData
	store   sessionStore
	refresh Refresher
	now     func() time.Time
	retryAt time.Time
}

func tokenExpiry(token, org string) (time.Time, error) {
	parts := strings.Split(token, ".")
	if len(parts) != 3 {
		return time.Time{}, ErrExpired
	}
	b, e := base64.RawURLEncoding.DecodeString(parts[1])
	if e != nil {
		return time.Time{}, ErrExpired
	}
	var c struct {
		Exp            int64  `json:"exp"`
		OrganizationID string `json:"organizationId"`
		UserID         string `json:"userId"`
	}
	if json.Unmarshal(b, &c) != nil || c.Exp <= 0 || c.OrganizationID != org || c.UserID == "" {
		return time.Time{}, ErrExpired
	}
	// Claims are only a local expiry/scope hint. Infisical verifies the signature.
	return time.Unix(c.Exp, 0), nil
}
func NewSession(d SessionData, store sessionStore, refresh Refresher) (*Session, error) {
	if d.Version != "v0.151.0" && d.Version != "v0.166.3" {
		return nil, ErrUnsupported
	}
	if d.Pending && d.Version == "v0.166.3" {
		d.Expired = true
	}
	if d.Pending && d.Version == "v0.151.0" {
		d.Pending = false
	}
	if store == nil || refresh == nil {
		return nil, ErrEnrollment
	}
	return &Session{data: d, store: store, refresh: refresh, now: time.Now}, nil
}
func (s *Session) Status() string {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.data.Expired {
		return "session_expired"
	}
	if s.data.RefreshToken == "" {
		exp, e := tokenExpiry(s.data.AccessToken, s.data.OrganizationID)
		if e != nil || !s.now().Before(exp) {
			return "enrollment_required"
		}
	}
	return "ready"
}
func (s *Session) Expire(observed string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.data.AccessToken == observed {
		s.data.Expired = true
		s.data.AccessToken = ""
		_ = s.store.Save("session.json", s.data)
	}
}
func (s *Session) Token(ctx context.Context) (string, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.data.Expired {
		return "", ErrExpired
	}
	exp, e := tokenExpiry(s.data.AccessToken, s.data.OrganizationID)
	if e == nil && s.now().Add(30*time.Second).Before(exp) {
		return s.data.AccessToken, nil
	}
	if s.now().Before(s.retryAt) {
		return "", ErrUnavailable
	}
	if s.data.RefreshToken == "" {
		return "", ErrEnrollment
	}
	// Write-ahead marker: a lost rotating response must never reuse the old token.
	s.data.Pending = true
	if s.store.Save("session.json", s.data) != nil {
		s.data.Expired = true
		return "", ErrExpired
	}
	result, e := s.refresh(ctx, s.data.RefreshToken)
	if e != nil {
		s.retryAt = s.now().Add(30 * time.Second)
	}
	if e != nil {
		if s.data.Version == "v0.166.3" || errors.Is(e, ErrExpired) {
			s.data.Expired = true
			s.data.AccessToken = ""
		} else {
			s.data.Pending = false
		}
		if s.store.Save("session.json", s.data) != nil {
			s.data.Expired = true
		}
		return "", e
	}
	if result.OrganizationID != s.data.OrganizationID {
		s.data.Expired = true
		_ = s.store.Save("session.json", s.data)
		return "", ErrExpired
	}
	exp, e = tokenExpiry(result.Token, s.data.OrganizationID)
	if e != nil || !s.now().Add(30*time.Second).Before(exp) {
		s.data.Expired = true
		_ = s.store.Save("session.json", s.data)
		return "", ErrExpired
	}
	if result.RefreshToken != "" {
		s.data.RefreshToken = result.RefreshToken
	}
	// v0.151.0 returns no refresh token. v0.166.3 grace hits may also omit it.
	s.data.AccessToken = result.Token
	s.data.Pending = false
	if s.store.Save("session.json", s.data) != nil {
		s.data.Expired = true
		s.data.AccessToken = ""
		return "", ErrExpired
	}
	return s.data.AccessToken, nil
}
func refreshSession(h *Transport) Refresher {
	return func(ctx context.Context, refresh string) (RefreshResult, error) {
		var out RefreshResult
		headers := http.Header{}
		headers.Set("Cookie", (&http.Cookie{Name: "jid", Value: refresh}).String())
		e := h.Call(ctx, http.MethodPost, "/api/v1/auth/token", nil, headers, nil, &out)
		if e != nil {
			return out, e
		}
		return out, nil
	}
}
