package enrollment

import (
	"context"
	"encoding/base64"
	"errors"
	"fmt"
	"sync"
	"testing"
	"time"
)

type memoryStore struct {
	mu   sync.Mutex
	data SessionData
	fail bool
}

func (m *memoryStore) Save(_ string, v any) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.fail {
		return errors.New("disk_unavailable")
	}
	m.data = v.(SessionData)
	return nil
}
func testJWT(exp time.Time) string {
	return "test." + base64.RawURLEncoding.EncodeToString([]byte(fmt.Sprintf(`{"exp":%d,"organizationId":"org","userId":"human"}`, exp.Unix()))) + ".fixture"
}
func TestSessionSingleWriterAndRotation(t *testing.T) {
	for _, version := range []string{"v0.151.0", "v0.166.3"} {
		t.Run(version, func(t *testing.T) {
			m := &memoryStore{}
			calls := 0
			s, _ := NewSession(SessionData{Version: version, OrganizationID: "org", RefreshToken: "test-refresh"}, m, func(context.Context, string) (RefreshResult, error) {
				calls++
				return RefreshResult{Token: testJWT(time.Now().Add(time.Hour)), RefreshToken: "test-rotated", OrganizationID: "org"}, nil
			})
			var wg sync.WaitGroup
			for i := 0; i < 20; i++ {
				wg.Add(1)
				go func() {
					defer wg.Done()
					if _, e := s.Token(context.Background()); e != nil {
						t.Error(e)
					}
				}()
			}
			wg.Wait()
			if calls != 1 || m.data.RefreshToken != "test-rotated" || m.data.Pending {
				t.Fatal("rotation was not single-writer durable")
			}
			restarted, _ := NewSession(m.data, m, s.refresh)
			if _, e := restarted.Token(context.Background()); e != nil || calls != 1 {
				t.Fatal("restart refreshed valid session")
			}
		})
	}
}
func TestSessionCrashOfflineAndDiskFailure(t *testing.T) {
	for _, version := range []string{"v0.151.0", "v0.166.3"} {
		m := &memoryStore{}
		calls := 0
		s, _ := NewSession(SessionData{Version: version, OrganizationID: "org", Pending: true, RefreshToken: "test-refresh"}, m, func(context.Context, string) (RefreshResult, error) { calls++; return RefreshResult{}, ErrUnavailable })
		_, e := s.Token(context.Background())
		if version == "v0.166.3" && (!errors.Is(e, ErrExpired) || calls != 0) {
			t.Fatal("reused uncertain rotated token")
		}
		if version == "v0.151.0" && calls != 1 {
			t.Fatal("legacy recovery not attempted")
		}
	}
	m := &memoryStore{fail: true}
	s, _ := NewSession(SessionData{Version: "v0.151.0", RefreshToken: "test-refresh"}, m, func(context.Context, string) (RefreshResult, error) {
		t.Fatal("refresh before durable marker")
		return RefreshResult{}, nil
	})
	if _, e := s.Token(context.Background()); e == nil {
		t.Fatal("disk failure accepted")
	}
}
func TestSessionRevokedAndWrongOrganization(t *testing.T) {
	for _, result := range []RefreshResult{{Token: testJWT(time.Now().Add(time.Hour)), OrganizationID: "different"}, {}} {
		m := &memoryStore{}
		s, _ := NewSession(SessionData{Version: "v0.151.0", OrganizationID: "org", RefreshToken: "test-refresh"}, m, func(context.Context, string) (RefreshResult, error) { return result, nil })
		if _, e := s.Token(context.Background()); !errors.Is(e, ErrExpired) {
			t.Fatal("invalid session accepted")
		}
	}
}

func TestSessionOfflineBackoffAndLostRotationResponse(t *testing.T) {
	for _, v := range []string{"v0.151.0", "v0.166.3"} {
		m := &memoryStore{}
		calls := 0
		s, _ := NewSession(SessionData{Version: v, OrganizationID: "org", RefreshToken: "test-refresh"}, m, func(context.Context, string) (RefreshResult, error) { calls++; return RefreshResult{}, ErrUnavailable })
		for i := 0; i < 10; i++ {
			_, _ = s.Token(context.Background())
		}
		if calls != 1 {
			t.Fatal("offline refresh storm")
		}
		if v == "v0.166.3" && s.Status() != "session_expired" {
			t.Fatal("lost rotating response was reusable")
		}
	}
}
func TestSessionRevocationRequiresRelogin(t *testing.T) {
	m := &memoryStore{}
	s, _ := NewSession(SessionData{Version: "v0.151.0", OrganizationID: "org", RefreshToken: "test-refresh"}, m, func(context.Context, string) (RefreshResult, error) { return RefreshResult{}, ErrExpired })
	_, _ = s.Token(context.Background())
	if s.Status() != "session_expired" {
		t.Fatal("revocation not terminal")
	}
	if _, e := s.Token(context.Background()); !errors.Is(e, ErrExpired) {
		t.Fatal("revoked session retried")
	}
}
