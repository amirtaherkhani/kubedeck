package enrollment

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func loginFixture(t *testing.T, denied, mfa, noCookie bool, suppliedToken ...string) (*Transport, *Store, Policy, string) {
	t.Helper()
	token := testJWT(time.Now().Add(time.Hour))
	if len(suppliedToken) > 0 {
		token = suppliedToken[0]
	}
	server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Infisical/Fastify parses advertised JSON even on a bodyless auth route.
		if r.Header.Get("Content-Type") == "application/json" && r.ContentLength == 0 {
			w.WriteHeader(500)
			fmt.Fprint(w, `{"code":"FST_ERR_CTP_EMPTY_JSON_BODY"}`)
			return
		}

		if r.URL.Path == "/api/v1/auth/token" {
			cookie, e := r.Cookie("jid")
			if e != nil || cookie.Value != token {
				t.Error("wrong refresh cookie")
				w.WriteHeader(401)
				return
			}
			json.NewEncoder(w).Encode(map[string]string{"token": testJWT(time.Now().Add(2 * time.Hour)), "organizationId": "org"})
			return
		}
		if r.Header.Get("Authorization") != "Bearer "+token {
			t.Error("wrong bearer")
			w.WriteHeader(401)
			return
		}
		switch r.URL.Path {
		case "/api/v1/auth/checkAuth":
			if r.Method != "POST" {
				t.Error("method")
			}
			fmt.Fprint(w, `{"message":"Authenticated"}`)
		case "/api/v1/organization-admin/projects":
			if r.Method != "GET" || r.URL.Query().Get("limit") != "1" {
				t.Error("unsafe probe")
			}
			if denied {
				w.WriteHeader(403)
				fmt.Fprint(w, `{"secret":"must-not-leak"}`)
				return
			}
			fmt.Fprint(w, `{"projects":[],"count":0}`)
		case "/api/v3/auth/select-organization":
			var b struct {
				OrganizationID string `json:"organizationId"`
				UserAgent      string `json:"userAgent"`
			}
			if json.NewDecoder(r.Body).Decode(&b) != nil || b.OrganizationID != "org" || b.UserAgent != "cli" {
				t.Error("selection contract")
			}
			if !noCookie {
				http.SetCookie(w, &http.Cookie{Name: "jid", Value: token, Path: "/", HttpOnly: true, Secure: true})
			}
			json.NewEncoder(w).Encode(map[string]any{"token": token, "isMfaEnabled": mfa})
		default:
			t.Errorf("unexpected action %s", r.URL.Path)
			w.WriteHeader(404)
		}
	}))
	t.Cleanup(server.Close)
	h, e := NewTransport(server.URL, server.Client())
	if e != nil {
		t.Fatal(e)
	}
	h.MinInterval = 0
	dir := t.TempDir()
	os.Chmod(dir, 0700)
	s, e := OpenStore(dir)
	if e != nil {
		t.Fatal(e)
	}
	t.Cleanup(func() { s.Close() })
	return h, s, testPolicy(), token
}
func callback(t *testing.T, loginURL, origin, token string) int {
	t.Helper()
	u, e := url.Parse(loginURL)
	if e != nil {
		t.Error(e)
		return 0
	}
	body, _ := json.Marshal(map[string]string{"JTWToken": token, "email": "private@example.invalid", "privateKey": ""})
	req, _ := http.NewRequest("POST", "http://127.0.0.1:"+u.Query().Get("callback_port")+"/", strings.NewReader(string(body)))
	req.Header.Set("Origin", origin)
	req.Header.Set("Content-Type", "application/json")
	res, e := http.DefaultClient.Do(req)
	if e != nil {
		t.Error(e)
		return 0
	}
	defer res.Body.Close()
	b, _ := io.ReadAll(res.Body)
	if strings.Contains(string(b), token) || strings.Contains(string(b), "private@example") {
		t.Error("callback leak")
	}
	return res.StatusCode
}
func TestOfficialBrowserCallbackPersistsValidatedPrivateSession(t *testing.T) {
	h, s, p, token := loginFixture(t, false, false, false)
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	report, e := BrowserLogin(ctx, h, p, s, func(u string) error { go callback(t, u, h.base.String(), token); return nil })
	if e != nil || !report.ServerAuthenticated || !report.AccessAllProjects || report.Status != "human_authority_verified" {
		t.Fatalf("login: %+v %v", report, e)
	}
	var d SessionData
	if e = s.Load("session.json", &d); e != nil {
		t.Fatal(e)
	}
	if d.AccessToken != token || d.RefreshToken != token || d.OrganizationID != p.OrganizationID || d.Origin != h.base.String() {
		t.Fatal("wrong session")
	}
	i, _ := os.Stat(filepath.Join(s.Dir, "session.json"))
	if i.Mode().Perm() != 0600 {
		t.Fatal("unsafe mode")
	}
	b, _ := json.Marshal(report)
	if strings.Contains(string(b), token) || strings.Contains(string(b), "private@example") {
		t.Fatal("report leak")
	}
}
func TestBrowserCallbackRejectsForeignOriginBeforeAcceptingOfficialOrigin(t *testing.T) {
	h, s, p, token := loginFixture(t, false, false, false)
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	_, e := BrowserLogin(ctx, h, p, s, func(u string) error {
		go func() {
			if callback(t, u, "https://evil.invalid", token) != 403 {
				t.Error("foreign origin accepted")
			}
			callback(t, u, h.base.String(), token)
		}()
		return nil
	})
	if e != nil {
		t.Fatal(e)
	}
}
func TestBrowserLoginFailuresPreservePreviousSession(t *testing.T) {
	for _, c := range []struct {
		name                  string
		denied, mfa, noCookie bool
	}{{"denied", true, false, false}, {"MFA", false, true, false}, {"refresh", false, false, true}} {
		t.Run(c.name, func(t *testing.T) {
			h, s, p, token := loginFixture(t, c.denied, c.mfa, c.noCookie)
			old := SessionData{AccessToken: "preserve-old"}
			s.Save("session.json", old)
			ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
			defer cancel()
			_, e := BrowserLogin(ctx, h, p, s, func(u string) error { go callback(t, u, h.base.String(), token); return nil })
			if e == nil {
				t.Fatal("unsafe login")
			}
			var got SessionData
			s.Load("session.json", &got)
			if got.AccessToken != old.AccessToken {
				t.Fatal("overwritten")
			}
		})
	}
}
func TestHumanStatusIsReadOnlyAndSeparatesMissingExpiredAndDenied(t *testing.T) {
	h, s, p, token := loginFixture(t, true, false, false)
	r, e := CheckHumanStatus(context.Background(), h, p, s)
	if e != nil || r.Status != "session_missing" || r.ServerAuthenticated {
		t.Fatalf("missing: %+v %v", r, e)
	}
	d := SessionData{Origin: h.base.String(), OrganizationID: p.OrganizationID, Version: p.Version, AccessToken: token, RefreshToken: "private-refresh"}
	s.Save("session.json", d)
	path := filepath.Join(s.Dir, "session.json")
	before, _ := os.ReadFile(path)
	r, e = CheckHumanStatus(context.Background(), h, p, s)
	if e == nil || r.AccessAllProjects || !r.ServerAuthenticated {
		t.Fatalf("denied: %+v %v", r, e)
	}
	after, _ := os.ReadFile(path)
	if string(before) != string(after) {
		t.Fatal("status mutated session")
	}
	d.AccessToken = testJWT(time.Now().Add(-time.Hour))
	s.Save("session.json", d)
	r, e = CheckHumanStatus(context.Background(), h, p, s)
	if e == nil || r.Status != "session_expired" {
		t.Fatalf("expired: %+v %v", r, e)
	}
}
func TestBrowserLoginCancellationDoesNotPersist(t *testing.T) {
	h, s, p, _ := loginFixture(t, false, false, false)
	ctx, cancel := context.WithCancel(context.Background())
	_, e := BrowserLogin(ctx, h, p, s, func(string) error { cancel(); return nil })
	if e == nil || s.Exists("session.json") {
		t.Fatal("cancel persisted")
	}
}
