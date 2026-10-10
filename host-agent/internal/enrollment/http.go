package enrollment

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"net"
	"net/http"
	"net/url"
	"strconv"
	"sync"
	"time"
)

// StatusError deliberately discards response bodies and URLs.
type StatusError struct {
	Code       int
	Method     string
	Endpoint   string
	RetryAfter time.Duration
}

func (e *StatusError) Error() string { return "infisical_request_rejected" }
func (e *StatusError) Is(target error) bool {
	return (target == ErrDenied && e.Code == 403) || (target == ErrExpired && e.Code == 401)
}

type Transport struct {
	base        *url.URL
	client      *http.Client
	mu          sync.Mutex
	next        time.Time
	MinInterval time.Duration
}

func NewTransport(origin string, client *http.Client) (*Transport, error) {
	u, e := url.Parse(origin)
	if e != nil || u.Host == "" || u.User != nil || u.RawQuery != "" || u.Fragment != "" || (u.Path != "" && u.Path != "/") {
		return nil, errors.New("invalid_infisical_origin")
	}
	ip := net.ParseIP(u.Hostname())
	if u.Scheme != "https" && (u.Scheme != "http" || (u.Hostname() != "localhost" && (ip == nil || !ip.IsLoopback()))) {
		return nil, errors.New("https_required")
	}
	if client == nil {
		client = &http.Client{Timeout: 10 * time.Second}
	}
	c := *client
	if c.Timeout == 0 {
		c.Timeout = 10 * time.Second
	}
	c.Jar = nil
	c.CheckRedirect = func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }
	return &Transport{base: u, client: &c, MinInterval: 200 * time.Millisecond}, nil
}
func (h *Transport) Call(ctx context.Context, method, path string, q url.Values, headers http.Header, body, out any) error {
	return h.call(ctx, method, path, q, headers, body, out, nil)
}

// call can capture only this response's cookies for an explicit user login.
func (h *Transport) call(ctx context.Context, method, path string, q url.Values, headers http.Header, body, out any, cookies *[]*http.Cookie) error {
	if e := h.Wait(ctx); e != nil {
		return e
	}
	var data []byte
	var e error
	if body != nil {
		data, e = json.Marshal(body)
		if e != nil || len(data) > 64<<10 {
			return ErrUnavailable
		}
	}
	u := h.base.ResolveReference(&url.URL{Path: path, RawQuery: q.Encode()})
	r, e := http.NewRequestWithContext(ctx, method, u.String(), bytes.NewReader(data))
	if e != nil {
		return ErrUnavailable
	}
	r.Header = headers.Clone()
	if r.Header == nil {
		r.Header = make(http.Header)
	}
	if body != nil {
		r.Header.Set("Content-Type", "application/json")
	} else {
		r.Header.Del("Content-Type")
	}
	r.Header.Set("User-Agent", "kuchdesk-enrollment")
	res, e := h.client.Do(r)
	if e != nil {
		return ErrUnavailable
	}
	defer res.Body.Close()
	if res.StatusCode < 200 || res.StatusCode >= 300 {
		delay := time.Duration(0)
		if n, e := strconv.Atoi(res.Header.Get("Retry-After")); e == nil && n > 0 {
			if n > 3600 {
				n = 3600
			}
			delay = time.Duration(n) * time.Second
		} else if at, e := http.ParseTime(res.Header.Get("Retry-After")); e == nil {
			delay = time.Until(at)
		}
		if delay > time.Hour {
			delay = time.Hour
		}
		failure := &StatusError{Code: res.StatusCode, RetryAfter: delay}
		// Fixed auth routes only; never emit dynamic paths, query values or bodies.
		switch path {
		case "/api/v1/auth/checkAuth", "/api/v1/auth/token", "/api/v1/organization-admin/projects", "/api/v3/auth/select-organization":
			failure.Endpoint = path
			if method == "GET" || method == "POST" {
				failure.Method = method
			}
		}
		return failure
	}
	b, e := io.ReadAll(io.LimitReader(res.Body, maxState+1))
	if e != nil || len(b) > maxState {
		return ErrUnavailable
	}
	if out != nil && json.Unmarshal(b, out) != nil {
		return ErrUnavailable
	}
	if cookies != nil {
		*cookies = res.Cookies()
	}
	return nil
}

func (h *Transport) Wait(ctx context.Context) error {
	h.mu.Lock()
	wait := time.Until(h.next)
	if wait < 0 {
		wait = 0
	}
	h.next = time.Now().Add(wait + h.MinInterval)
	h.mu.Unlock()
	timer := time.NewTimer(wait)
	defer timer.Stop()
	select {
	case <-ctx.Done():
		return ErrUnavailable
	case <-timer.C:
	}
	return nil
}
