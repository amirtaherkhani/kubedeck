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
	r.Header.Set("Content-Type", "application/json")
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
		return &StatusError{Code: res.StatusCode, RetryAfter: delay}
	}
	b, e := io.ReadAll(io.LimitReader(res.Body, maxState+1))
	if e != nil || len(b) > maxState {
		return ErrUnavailable
	}
	if out != nil && json.Unmarshal(b, out) != nil {
		return ErrUnavailable
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
