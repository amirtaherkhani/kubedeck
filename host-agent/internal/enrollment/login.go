package enrollment

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"io"
	"mime"
	"net"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

// HumanStatus never contains tokens, user details, or upstream error bodies.
// AccessAllProjects is the server capability used for self-enrollment, not a role name.
type HumanStatus struct {
	HTTPStatus                int       `json:"httpStatus,omitempty"`
	RequestMethod             string    `json:"requestMethod,omitempty"`
	RequestEndpoint           string    `json:"requestEndpoint,omitempty"`
	Status                    string    `json:"status"`
	ServerAuthenticated       bool      `json:"serverAuthenticated"`
	OrganizationScopeVerified bool      `json:"organizationScopeVerified"`
	AccessAllProjects         bool      `json:"accessAllProjectsVerified"`
	RefreshAvailable          bool      `json:"refreshAvailable"`
	ObservedAt                time.Time `json:"observedAt"`
}

func verifyHuman(ctx context.Context, h *Transport, p Policy, token string) (HumanStatus, error) {
	r := HumanStatus{Status: "not_verified", ObservedAt: time.Now()}
	fail := func(e error) (HumanStatus, error) { r.Status = classify(e); return requestFailure(r, e), e }
	if p.Version != "v0.151.0" {
		return fail(ErrUnsupported)
	}
	expiry, e := tokenExpiry(token, p.OrganizationID)
	if len(token) > 16<<10 || e != nil || !time.Now().Before(expiry) {
		return fail(ErrExpired)
	}
	headers := http.Header{"Authorization": []string{"Bearer " + token}}
	var auth struct {
		Message string `json:"message"`
	}
	if e = h.Call(ctx, "POST", "/api/v1/auth/checkAuth", nil, headers, nil, &auth); e != nil {
		return fail(e)
	}
	if auth.Message != "Authenticated" {
		return fail(ErrUnavailable)
	}
	r.ServerAuthenticated = true
	// JWT claims are trusted only after the server accepts the same human token.
	r.OrganizationScopeVerified = true
	var projects struct {
		Projects []Project `json:"projects"`
		Count    *int      `json:"count"`
	}
	if e = h.Call(ctx, "GET", "/api/v1/organization-admin/projects", url.Values{"limit": {"1"}, "offset": {"0"}}, headers, nil, &projects); e != nil {
		return fail(e)
	}
	if projects.Count == nil || *projects.Count < 0 || len(projects.Projects) > 1 || *projects.Count < len(projects.Projects) {
		return fail(ErrUnavailable)
	}
	for _, project := range projects.Projects {
		if project.OrganizationID != p.OrganizationID {
			return fail(ErrDenied)
		}
	}
	r.AccessAllProjects = true
	r.Status = "human_authority_verified"
	return r, nil
}

// CheckHumanStatus performs read-only server validation. It does not refresh,
// rewrite private session state, enroll users, or create identities/memberships.
func CheckHumanStatus(ctx context.Context, h *Transport, p Policy, store *Store) (HumanStatus, error) {
	r := HumanStatus{Status: "session_missing", ObservedAt: time.Now()}
	if p.Version != "v0.151.0" {
		r.Status = "unsupported_version"
		return r, ErrUnsupported
	}
	if !store.Exists("session.json") {
		return r, nil
	}
	var d SessionData
	if e := store.Load("session.json", &d); e != nil {
		r.Status = "invalid_private_session"
		return r, e
	}
	if d.Origin != h.base.String() || d.OrganizationID != p.OrganizationID || d.Version != p.Version {
		r.Status = "session_scope_mismatch"
		return r, errors.New("session_scope_mismatch")
	}
	if d.Pending || d.Expired {
		r.Status = "session_expired"
		return r, ErrExpired
	}
	r, e := verifyHuman(ctx, h, p, d.AccessToken)
	r.RefreshAvailable = d.RefreshToken != ""
	return r, e
}

// BrowserLogin uses Infisical v0.151.0's official callback_port contract.
// Passwords, SSO and MFA remain in the official browser UI. Only the user's
// explicit login callback is received; no browser storage is inspected.
func BrowserLogin(ctx context.Context, h *Transport, p Policy, store *Store, announce func(string) error, events ...func(string)) (HumanStatus, error) {
	return browserLogin(ctx, h, p, store, announce, nil, events...)
}

// BrowserLoginWithFallback keeps one official callback attempt and one hidden
// user-input path under the same writer lock and single-use validation gate.
func BrowserLoginWithFallback(ctx context.Context, h *Transport, p Policy, store *Store, announce func(string) error, read func(context.Context) (string, error), events ...func(string)) (HumanStatus, error) {
	return browserLogin(ctx, h, p, store, announce, read, events...)
}
func browserLogin(ctx context.Context, h *Transport, p Policy, store *Store, announce func(string) error, read func(context.Context) (string, error), events ...func(string)) (HumanStatus, error) {
	initial := HumanStatus{Status: "login_not_completed", ObservedAt: time.Now()}
	emit := func(stage string) {
		for _, event := range events {
			if event != nil {
				event(stage)
			}
		}
	}
	if p.Version != "v0.151.0" {
		return initial, ErrUnsupported
	}
	listener, e := listenForBrowser()
	if e != nil {
		return initial, errors.New("login_listener_unavailable")
	}
	defer listener.Close()
	ctx, cancel := context.WithTimeout(ctx, 10*time.Minute)
	defer cancel()
	type outcome struct {
		report HumanStatus
		err    error
	}
	result := make(chan outcome, 2)
	var mu sync.Mutex
	used := false
	origin := strings.TrimSuffix(h.base.String(), "/")
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Cache-Control", "no-store")
		ip, _, err := net.SplitHostPort(r.RemoteAddr)
		rejected := ""
		switch {
		case err != nil || !net.ParseIP(ip).IsLoopback():
			rejected = "callback_rejected_peer"
		case r.Host != listener.Addr().String():
			rejected = "callback_rejected_host"
		case r.Header.Get("Origin") != origin:
			rejected = "callback_rejected_origin"
		case r.URL.Path != "/" || r.URL.RawQuery != "":
			rejected = "callback_rejected_route"
		}
		if rejected != "" {
			emit(rejected)
			http.Error(w, "callback_rejected", 403)
			return
		}
		w.Header().Set("Access-Control-Allow-Origin", origin)
		w.Header().Set("Access-Control-Allow-Credentials", "true")
		w.Header().Set("Vary", "Origin")
		if r.Method == "OPTIONS" {
			emit("callback_preflight_received")
			if r.Header.Get("Access-Control-Request-Method") != "POST" {
				http.Error(w, "callback_rejected", 403)
				return
			}
			w.Header().Set("Access-Control-Allow-Methods", "POST")
			w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
			if r.Header.Get("Access-Control-Request-Private-Network") == "true" {
				w.Header().Set("Access-Control-Allow-Private-Network", "true")
			}
			w.WriteHeader(204)
			return
		}
		media, _, err := mime.ParseMediaType(r.Header.Get("Content-Type"))
		if r.Method != "POST" || err != nil || media != "application/json" {
			emit("callback_rejected_request")
			http.Error(w, "callback_rejected", 400)
			return
		}
		token, err := decodeBrowserCallback(http.MaxBytesReader(w, r.Body, 32<<10))
		if err != nil {
			emit("callback_rejected_payload")
			http.Error(w, "callback_rejected", 400)
			return
		}
		expiry, err := tokenExpiry(token, p.OrganizationID)
		if err != nil || len(token) > 16<<10 || !time.Now().Before(expiry) {
			emit("callback_rejected_scope")
			http.Error(w, "callback_scope_rejected", 400)
			return
		}
		mu.Lock()
		if used {
			mu.Unlock()
			emit("callback_rejected_replay")
			http.Error(w, "callback_already_used", 409)
			return
		}
		used = true
		mu.Unlock()
		emit("callback_received")
		report, err := completeBrowserLogin(ctx, h, p, store, token)
		if err != nil {
			emit("callback_server_validation_failed")
			http.Error(w, "login_not_completed_check_terminal", 400)
		} else {
			emit("callback_session_saved")
			w.WriteHeader(200)
			_, _ = io.WriteString(w, "login_completed")
		}
		result <- outcome{report, err}
	})
	server := &http.Server{Handler: handler, ReadHeaderTimeout: 5 * time.Second, ReadTimeout: 10 * time.Second, WriteTimeout: time.Minute, IdleTimeout: 5 * time.Second, MaxHeaderBytes: 8 << 10}
	go func() { _ = server.Serve(listener) }()
	defer func() {
		stop, c := context.WithTimeout(context.Background(), time.Second)
		defer c()
		_ = server.Shutdown(stop)
		_ = server.Close()
	}()
	loginURL := origin + "/login?" + url.Values{"callback_port": {strings.Split(listener.Addr().String(), ":")[1]}}.Encode()
	emit("callback_waiting")
	if e = announce(loginURL); e != nil {
		return initial, errors.New("login_instructions_unavailable")
	}
	if read != nil {
		finished := make(chan struct{})
		go func() {
			defer close(finished)
			encoded, err := read(ctx)
			if err != nil {
				result <- outcome{initial, err}
				return
			}
			token, err := decodeBrowserToken(encoded)
			if err != nil {
				result <- outcome{initial, err}
				return
			}
			mu.Lock()
			if used {
				mu.Unlock()
				return
			}
			used = true
			mu.Unlock()
			emit("fallback_received")
			report, err := completeBrowserLogin(ctx, h, p, store, token)
			result <- outcome{report, err}
		}()
		defer func() { cancel(); <-finished }()
	}
	select {
	case o := <-result:
		return o.report, o.err
	case <-ctx.Done():
		emit("callback_timeout_or_cancel")
		return initial, errors.New("login_cancelled_or_timed_out")
	}
}

func completeBrowserLogin(ctx context.Context, h *Transport, p Policy, store *Store, token string) (HumanStatus, error) {
	report, e := verifyHuman(ctx, h, p, token)
	if e != nil {
		return report, e
	}
	var selected struct {
		Token string `json:"token"`
		MFA   *bool  `json:"isMfaEnabled"`
	}
	var cookies []*http.Cookie
	headers := http.Header{"Authorization": []string{"Bearer " + token}}
	e = h.call(ctx, "POST", "/api/v3/auth/select-organization", nil, headers, map[string]string{"organizationId": p.OrganizationID, "userAgent": "cli"}, &selected, &cookies)
	if e != nil {
		return requestFailure(HumanStatus{Status: classify(e), ObservedAt: time.Now()}, e), e
	}
	if selected.MFA == nil || *selected.MFA {
		return HumanStatus{Status: "browser_mfa_required", ObservedAt: time.Now()}, errors.New("browser_mfa_required")
	}
	refresh := ""
	cookieCount := 0
	for _, c := range cookies {
		if c.Name == "jid" {
			cookieCount++
			if cookieCount > 1 {
				return HumanStatus{Status: "invalid_refresh_response"}, ErrUnavailable
			}
			refresh = c.Value
		}
	}
	expiry, e := tokenExpiry(refresh, p.OrganizationID)
	if len(refresh) > 16<<10 || e != nil || !time.Now().Before(expiry) {
		return HumanStatus{Status: "refresh_unavailable", ObservedAt: time.Now()}, errors.New("refresh_unavailable")
	}
	report, e = verifyHuman(ctx, h, p, selected.Token)
	if e != nil {
		return report, e
	}
	if ctx.Err() != nil {
		return HumanStatus{Status: "login_cancelled_or_timed_out"}, errors.New("login_cancelled_or_timed_out")
	}
	d := SessionData{Origin: h.base.String(), OrganizationID: p.OrganizationID, Version: p.Version, AccessToken: selected.Token, RefreshToken: refresh}
	if e = store.Save("session.json", d); e != nil {
		return HumanStatus{Status: "session_persistence_failed", ObservedAt: time.Now()}, e
	}
	report.RefreshAvailable = true
	return report, nil
}

// HumanStatusAt reads atomically persisted state without taking the writer lock.
// It can run alongside the controller and never creates a lock or session file.
func HumanStatusAt(ctx context.Context, h *Transport, p Policy, dir string) (HumanStatus, error) {
	info, e := os.Lstat(dir)
	if !filepath.IsAbs(dir) || e != nil || !info.IsDir() || info.Mode().Perm() != 0700 || !owner(info) {
		return HumanStatus{Status: "invalid_private_state", ObservedAt: time.Now()}, errors.New("private_state_directory_required")
	}
	return CheckHumanStatus(ctx, h, p, &Store{Dir: dir})
}

// The callback is an untrusted network message, including when it originates on
// loopback. Reject duplicate/case-variant keys rather than taking the last token.
func decodeBrowserCallback(body io.Reader) (string, error) {
	d := json.NewDecoder(body)
	start, e := d.Token()
	if e != nil || start != json.Delim('{') {
		return "", ErrDenied
	}
	fields := map[string]string{}
	for d.More() {
		key, e := d.Token()
		if e != nil {
			return "", ErrDenied
		}
		name, ok := key.(string)
		if !ok || (name != "JTWToken" && name != "email" && name != "privateKey") {
			return "", ErrDenied
		}
		if _, exists := fields[name]; exists {
			return "", ErrDenied
		}
		var value string
		if d.Decode(&value) != nil {
			return "", ErrDenied
		}
		fields[name] = value
	}
	end, e := d.Token()
	if e != nil || end != json.Delim('}') || fields["JTWToken"] == "" || fields["privateKey"] != "" {
		return "", ErrDenied
	}
	var extra any
	if d.Decode(&extra) != io.EOF {
		return "", ErrDenied
	}
	return fields["JTWToken"], nil
}

func listenForBrowser() (net.Listener, error) { return net.Listen("tcp4", "127.0.0.1:0") }

// LoginFromBrowserToken accepts only the official Copy-to-clipboard payload,
// explicitly entered by the user in a hidden local terminal. It does not read
// clipboard/browser storage and uses the same server checks as the callback.
func LoginFromBrowserToken(ctx context.Context, h *Transport, p Policy, store *Store, encoded string) (HumanStatus, error) {
	rejected := HumanStatus{Status: "invalid_browser_fallback", ObservedAt: time.Now()}
	if p.Version != "v0.151.0" {
		return rejected, ErrUnsupported
	}
	token, e := decodeBrowserToken(encoded)
	if e != nil {
		return rejected, e
	}
	return completeBrowserLogin(ctx, h, p, store, token)
}
func decodeBrowserToken(encoded string) (string, error) {
	if len(encoded) > 64<<10 {
		return "", errors.New("invalid_browser_fallback")
	}
	data, e := base64.StdEncoding.Strict().DecodeString(strings.TrimSpace(encoded))
	if e != nil || len(data) > 32<<10 {
		return "", errors.New("invalid_browser_fallback")
	}
	token, e := decodeBrowserCallback(bytes.NewReader(data))
	if e != nil {
		return "", errors.New("invalid_browser_fallback")
	}
	return token, nil
}

// requestFailure exposes only fixed route metadata and numeric HTTP status.
func requestFailure(report HumanStatus, err error) HumanStatus {
	var failure *StatusError
	if errors.As(err, &failure) {
		report.HTTPStatus = failure.Code
		report.RequestMethod = failure.Method
		report.RequestEndpoint = failure.Endpoint
	}
	return report
}
