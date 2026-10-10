package enrollment

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"sync/atomic"
	"testing"
	"time"
)

func scopedTestJWT(org, method string) string {
	b, _ := json.Marshal(map[string]any{"exp": time.Now().Add(time.Hour).Unix(), "organizationId": org, "userId": "human", "authMethod": method, "isMfaVerified": true, "mfaMethod": "totp"})
	return "test." + base64.RawURLEncoding.EncodeToString(b) + ".fixture"
}
func callbackAddress(t *testing.T, loginURL string) string {
	t.Helper()
	u, e := url.Parse(loginURL)
	if e != nil {
		t.Fatal(e)
	}
	return "http://127.0.0.1:" + u.Query().Get("callback_port") + "/"
}
func sendCallbackRequest(t *testing.T, req *http.Request) int {
	t.Helper()
	res, e := http.DefaultClient.Do(req)
	if e != nil {
		t.Error(e)
		return 0
	}
	defer res.Body.Close()
	io.Copy(io.Discard, res.Body)
	return res.StatusCode
}
func TestBrowserCallbackPreflightAndRejectionBoundaries(t *testing.T) {
	h, s, p, token := loginFixture(t, false, false, false)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	_, e := BrowserLogin(ctx, h, p, s, func(loginURL string) error {
		addr := callbackAddress(t, loginURL)
		req, _ := http.NewRequest("OPTIONS", addr, nil)
		req.Header.Set("Origin", h.base.String())
		req.Header.Set("Access-Control-Request-Method", "POST")
		req.Header.Set("Access-Control-Request-Private-Network", "true")
		res, e := http.DefaultClient.Do(req)
		if e != nil {
			return e
		}
		res.Body.Close()
		if res.StatusCode != 204 || res.Header.Get("Access-Control-Allow-Origin") != h.base.String() || res.Header.Get("Access-Control-Allow-Credentials") != "true" || res.Header.Get("Access-Control-Allow-Private-Network") != "true" {
			t.Error("official credentialed/PNA preflight unsupported")
		}
		good, _ := json.Marshal(map[string]string{"JTWToken": token, "email": "test@example.invalid", "privateKey": ""})
		cases := []struct {
			name, method, path, origin, host, media, body string
			want                                          int
		}{
			{"missing-origin", "POST", "", "", "", "application/json", string(good), 403},
			{"null-origin", "POST", "", "null", "", "application/json", string(good), 403},
			{"host-rebinding", "POST", "", h.base.String(), "attacker.invalid", "application/json", string(good), 403},
			{"query", "POST", "?state=unexpected", h.base.String(), "", "application/json", string(good), 403},
			{"path", "POST", "elsewhere", h.base.String(), "", "application/json", string(good), 403},
			{"get", "GET", "", h.base.String(), "", "application/json", string(good), 400},
			{"form", "POST", "", h.base.String(), "", "application/x-www-form-urlencoded", string(good), 400},
			{"oversize", "POST", "", h.base.String(), "", "application/json", strings.Repeat("x", 33000), 400},
			{"trailing-json", "POST", "", h.base.String(), "", "application/json", string(good) + "{}", 400},
			{"unknown-key", "POST", "", h.base.String(), "", "application/json", `{"JTWToken":"x","unexpected":"value"}`, 400},
			{"legacy-private-key", "POST", "", h.base.String(), "", "application/json", `{"JTWToken":"x","privateKey":"do-not-save"}`, 400},
			{"wrong-org", "POST", "", h.base.String(), "", "application/json", fmt.Sprintf(`{"JTWToken":%q}`, scopedTestJWT("other", "oidc")), 400},
			{"expired", "POST", "", h.base.String(), "", "application/json", fmt.Sprintf(`{"JTWToken":%q}`, testJWT(time.Now().Add(-time.Hour))), 400},
		}
		for _, c := range cases {
			req, _ := http.NewRequest(c.method, addr+c.path, strings.NewReader(c.body))
			req.Header.Set("Origin", c.origin)
			req.Header.Set("Content-Type", c.media)
			if c.host != "" {
				req.Host = c.host
			}
			if got := sendCallbackRequest(t, req); got != c.want {
				t.Errorf("%s got %d want %d", c.name, got, c.want)
			}
		}
		if s.Exists("session.json") {
			t.Error("malicious callback persisted")
		}
		go callback(t, loginURL, h.base.String(), token)
		return nil
	})
	if e != nil {
		t.Fatal(e)
	}
}
func TestBrowserLoginRejectsForgedTokenAndDoesNotFollowServerRedirect(t *testing.T) {
	for _, code := range []int{401, 302} {
		t.Run(fmt.Sprint(code), func(t *testing.T) {
			var leaked atomic.Bool
			other := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { leaked.Store(true) }))
			defer other.Close()
			remote := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Location", other.URL)
				w.WriteHeader(code)
				fmt.Fprint(w, "private-error-token")
			}))
			defer remote.Close()
			h, _ := NewTransport(remote.URL, remote.Client())
			h.MinInterval = 0
			dir := t.TempDir()
			os.Chmod(dir, 0700)
			s, _ := OpenStore(dir)
			defer s.Close()
			ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
			defer cancel()
			report, e := BrowserLogin(ctx, h, testPolicy(), s, func(u string) error {
				go callback(t, u, h.base.String(), testJWT(time.Now().Add(time.Hour)))
				return nil
			})
			if e == nil || s.Exists("session.json") || report.ServerAuthenticated || leaked.Load() {
				t.Fatal("untrusted response accepted or token redirected")
			}
			b, _ := json.Marshal(report)
			if strings.Contains(string(b)+e.Error(), "private-error-token") {
				t.Fatal("upstream error leaked")
			}
		})
	}
}
func TestBrowserCallbackSingleUseAndCancelInFlight(t *testing.T) {
	started := make(chan struct{})
	release := make(chan struct{})
	var selections atomic.Int32
	token := testJWT(time.Now().Add(time.Hour))
	remote := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v1/auth/checkAuth" {
			selections.Add(1)
			w.WriteHeader(500)
			return
		}
		close(started)
		select {
		case <-release:
			fmt.Fprint(w, `{"message":"Authenticated"}`)
		case <-r.Context().Done():
		}
	}))
	defer remote.Close()
	h, _ := NewTransport(remote.URL, remote.Client())
	h.MinInterval = 0
	dir := t.TempDir()
	os.Chmod(dir, 0700)
	s, _ := OpenStore(dir)
	defer s.Close()
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	var addr string
	firstDone := make(chan struct{})
	_, e := BrowserLogin(ctx, h, testPolicy(), s, func(u string) error {
		addr = callbackAddress(t, u)
		go func() { callback(t, u, h.base.String(), token); close(firstDone) }()
		select {
		case <-started:
		case <-ctx.Done():
			return ctx.Err()
		}
		if callback(t, u, h.base.String(), token) != 409 {
			t.Error("callback replay accepted")
		}
		cancel()
		return nil
	})
	close(release)
	<-firstDone
	if e == nil || s.Exists("session.json") || selections.Load() != 0 {
		t.Fatal("cancellation created state or continued exchange")
	}
	c := http.Client{Timeout: time.Second}
	if res, e := c.Get(addr); e == nil {
		res.Body.Close()
		t.Fatal("listener survived cancellation")
	}
}
func TestBrowserLoginTTLAndUnsupportedVersion(t *testing.T) {
	h, s, p, _ := loginFixture(t, false, false, false)
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Millisecond)
	defer cancel()
	var addr string
	_, e := BrowserLogin(ctx, h, p, s, func(u string) error { addr = callbackAddress(t, u); return nil })
	if e == nil || s.Exists("session.json") {
		t.Fatal("TTL did not close login")
	}
	c := http.Client{Timeout: time.Second}
	if res, e := c.Get(addr); e == nil {
		res.Body.Close()
		t.Fatal("expired listener remained open")
	}
	p.Version = "v0.166.3"
	_, e = BrowserLogin(context.Background(), h, p, s, func(string) error { t.Fatal("unsupported version opened login"); return nil })
	if e == nil {
		t.Fatal("unsupported version accepted")
	}
}
func TestHumanStatusDoesNotRequireWriterLockAndRejectsUnsafeState(t *testing.T) {
	h, s, p, token := loginFixture(t, false, false, false)
	d := SessionData{Origin: h.base.String(), OrganizationID: p.OrganizationID, Version: p.Version, AccessToken: token}
	s.Save("session.json", d)
	r, e := HumanStatusAt(context.Background(), h, p, s.Dir)
	if e != nil || !r.AccessAllProjects {
		t.Fatalf("status while writer holds lock %+v %v", r, e)
	}
	d.OrganizationID = "other"
	s.Save("session.json", d)
	r, e = HumanStatusAt(context.Background(), h, p, s.Dir)
	if e == nil || r.ServerAuthenticated {
		t.Fatal("cross-org status accepted")
	}
	os.Chmod(filepath.Join(s.Dir, "session.json"), 0644)
	r, e = HumanStatusAt(context.Background(), h, p, s.Dir)
	if e == nil || r.ServerAuthenticated {
		t.Fatal("unsafe file accepted")
	}
}

func TestCallbackRejectsDuplicateTokenFields(t *testing.T) {
	h, s, p, token := loginFixture(t, false, false, false)
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	_, e := BrowserLogin(ctx, h, p, s, func(u string) error {
		req, _ := http.NewRequest("POST", callbackAddress(t, u), strings.NewReader(fmt.Sprintf(`{"JTWToken":%q,"JTWToken":%q}`, token, token)))
		req.Header.Set("Origin", h.base.String())
		req.Header.Set("Content-Type", "application/json")
		if code := sendCallbackRequest(t, req); code != 400 {
			t.Errorf("duplicate token fields accepted: %d", code)
		}
		cancel()
		return nil
	})
	if e == nil || s.Exists("session.json") {
		t.Fatal("ambiguous token persisted")
	}
}

func TestOfficialBrowserMFAAndSSOTokensUseSameProtocol(t *testing.T) {
	for _, method := range []string{"email", "saml", "oidc", "google"} {
		t.Run(method, func(t *testing.T) {
			token := scopedTestJWT("org", method)
			h, s, p, _ := loginFixture(t, false, false, false, token)
			ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
			defer cancel()
			r, e := BrowserLogin(ctx, h, p, s, func(u string) error { go callback(t, u, h.base.String(), token); return nil })
			if e != nil || !r.AccessAllProjects || !r.RefreshAvailable {
				t.Fatalf("official authenticated callback failed %+v %v", r, e)
			}
		})
	}
}
func TestBrowserSessionRefreshPersistsAfterRestart(t *testing.T) {
	h, s, p, token := loginFixture(t, false, false, false)
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	_, e := BrowserLogin(ctx, h, p, s, func(u string) error { go callback(t, u, h.base.String(), token); return nil })
	if e != nil {
		t.Fatal(e)
	}
	var d SessionData
	if e = s.Load("session.json", &d); e != nil {
		t.Fatal(e)
	}
	d.AccessToken = testJWT(time.Now().Add(-time.Minute))
	if e = s.Save("session.json", d); e != nil {
		t.Fatal(e)
	}
	session, e := NewSession(d, s, refreshSession(h))
	if e != nil {
		t.Fatal(e)
	}
	got, e := session.Token(ctx)
	if e != nil || got == d.AccessToken {
		t.Fatalf("refresh failed: %v", e)
	}
	var saved SessionData
	s.Load("session.json", &saved)
	if saved.AccessToken != got || saved.RefreshToken != token || saved.Pending || saved.Expired {
		t.Fatal("refresh state not durably preserved")
	}
}
func TestHumanStatusExpiredWithRefreshIsNotAuthenticatedProof(t *testing.T) {
	h, s, p, _ := loginFixture(t, false, false, false)
	d := SessionData{Origin: h.base.String(), OrganizationID: p.OrganizationID, Version: p.Version, AccessToken: testJWT(time.Now().Add(-time.Hour)), RefreshToken: "must-not-use"}
	s.Save("session.json", d)
	before, _ := os.ReadFile(filepath.Join(s.Dir, "session.json"))
	r, e := HumanStatusAt(context.Background(), h, p, s.Dir)
	after, _ := os.ReadFile(filepath.Join(s.Dir, "session.json"))
	if e == nil || r.ServerAuthenticated || r.AccessAllProjects || !r.RefreshAvailable || r.Status != "session_expired" || string(before) != string(after) {
		t.Fatalf("read-only status conflated refresh availability with authority %+v %v", r, e)
	}
}

func TestBrowserListenerBindsOnlyIPv4Loopback(t *testing.T) {
	listener, e := listenForBrowser()
	if e != nil {
		t.Fatal(e)
	}
	defer listener.Close()
	addr, ok := listener.Addr().(*net.TCPAddr)
	if !ok || !addr.IP.Equal(net.ParseIP("127.0.0.1")) || addr.Port == 0 {
		t.Fatal("callback listener is not exclusive loopback")
	}
}

func TestBrowserLoginEmitsOnlySafeCallbackStages(t *testing.T) {
	h, s, p, token := loginFixture(t, false, false, false)
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	events := make(chan string, 20)
	_, e := BrowserLogin(ctx, h, p, s, func(u string) error {
		go func() { callback(t, u, "https://foreign.invalid", token); callback(t, u, h.base.String(), token) }()
		return nil
	}, func(event string) { events <- event })
	if e != nil {
		t.Fatal(e)
	}
	close(events)
	seen := map[string]bool{}
	for event := range events {
		seen[event] = true
		if strings.Contains(event, token) || strings.Contains(event, "foreign.invalid") || strings.Contains(event, "private@example") {
			t.Fatal("unsafe event")
		}
	}
	for _, event := range []string{"callback_waiting", "callback_rejected_origin", "callback_received", "callback_session_saved"} {
		if !seen[event] {
			t.Errorf("missing safe stage %s", event)
		}
	}
}

func TestOfficialManualBrowserFallbackValidatesAndPersists(t *testing.T) {
	h, s, p, token := loginFixture(t, false, false, false)
	body, _ := json.Marshal(map[string]string{"JTWToken": token, "email": "private@example.invalid", "privateKey": ""})
	encoded := base64.StdEncoding.EncodeToString(body)
	r, e := LoginFromBrowserToken(context.Background(), h, p, s, encoded)
	if e != nil || !r.AccessAllProjects || !r.RefreshAvailable || !s.Exists("session.json") {
		t.Fatalf("fallback failed %+v %v", r, e)
	}
	before, _ := os.ReadFile(filepath.Join(s.Dir, "session.json"))
	for _, bad := range []string{"not-base64", strings.Repeat("x", 65537), base64.StdEncoding.EncodeToString([]byte(`{"JTWToken":"forged"}`))} {
		if _, e := LoginFromBrowserToken(context.Background(), h, p, s, bad); e == nil {
			t.Fatal("bad fallback accepted")
		}
	}
	after, _ := os.ReadFile(filepath.Join(s.Dir, "session.json"))
	if string(before) != string(after) {
		t.Fatal("invalid fallback replaced state")
	}
}

func TestCombinedBrowserFallbackUsesFreshURLAndValidates(t *testing.T) {
	h, s, p, token := loginFixture(t, false, false, false)
	body, _ := json.Marshal(map[string]string{"JTWToken": token, "email": "test@example.invalid", "privateKey": ""})
	announced := false
	r, e := BrowserLoginWithFallback(context.Background(), h, p, s, func(u string) error {
		address := callbackAddress(t, u)
		conn, e := net.DialTimeout("tcp", strings.TrimPrefix(address, "http://")[:len(strings.TrimPrefix(address, "http://"))-1], time.Second)
		if e != nil {
			return e
		}
		conn.Close()
		announced = true
		return nil
	}, func(context.Context) (string, error) {
		if !announced {
			t.Error("input started before fresh login URL")
		}
		return base64.StdEncoding.EncodeToString(body), nil
	})
	if e != nil || !r.AccessAllProjects || !r.RefreshAvailable {
		t.Fatalf("combined login failed: %+v %v", r, e)
	}
}

func TestCombinedAutomaticCallbackCancelsHiddenReader(t *testing.T) {
	h, s, p, token := loginFixture(t, false, false, false)
	stopped := make(chan struct{})
	r, e := BrowserLoginWithFallback(context.Background(), h, p, s, func(u string) error { go callback(t, u, h.base.String(), token); return nil }, func(ctx context.Context) (string, error) { defer close(stopped); <-ctx.Done(); return "", ctx.Err() })
	if e != nil || !r.AccessAllProjects {
		t.Fatalf("callback failed: %+v %v", r, e)
	}
	select {
	case <-stopped:
	default:
		t.Fatal("hidden reader survived automatic callback")
	}
}
