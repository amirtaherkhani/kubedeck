package main

import (
	"bytes"
	"context"
	"errors"
	"net"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/amirtaherkhani/kuchdesk/host-agent/internal/doctor"
)

type fakeCommands struct{}

func (fakeCommands) Output(_ context.Context, name string, args ...string) ([]byte, error) {
	switch name {
	case "route":
		return []byte("interface: en0\n"), nil
	case "docker":
		return []byte("28.0.0\n"), nil
	case "kubectl":
		joined := strings.Join(args, " ")
		if strings.Contains(joined, "get nodes") {
			return []byte(`{"items":[{"status":{"conditions":[{"type":"Ready","status":"True"}]}}]}`), nil
		}
		if strings.Contains(joined, "top nodes") {
			return []byte("node 100m 10% 1024Mi 20%\n"), nil
		}
		return []byte(`{"spec":{"replicas":1},"status":{"readyReplicas":1}}`), nil
	}
	return nil, errors.New("unexpected command")
}

type fakeResolver struct{}

func (fakeResolver) LookupIPAddr(context.Context, string) ([]net.IPAddr, error) {
	return []net.IPAddr{{IP: net.ParseIP("192.0.2.1")}}, nil
}

func TestCLIReportAndOfflineAI(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(200) }))
	defer server.Close()
	service := doctor.Service{Commands: fakeCommands{}, Resolver: fakeResolver{}, HTTP: server.Client(), DiskFree: func(string) (uint64, error) { return 5 << 30, nil }}
	args := []string{"-domain", "infisical.local.dev", "-kube-context", "docker-desktop", "-registry-url", server.URL, "-disk-path", t.TempDir(), "-service", "development-tools/kuchdesk-agent", "-ai"}
	var out, errOut bytes.Buffer
	if code := run(args, &out, &errOut, service); code != 0 || !strings.Contains(out.String(), `"healthy":true`) || !strings.Contains(out.String(), `"provider_not_configured"`) {
		t.Fatalf("doctor CLI failed: code=%d out=%s err=%s", code, out.String(), errOut.String())
	}
	out.Reset()
	errOut.Reset()
	bad := append(append([]string(nil), args...), "-service", "../other")
	if code := run(bad, &out, &errOut, service); code != 2 || out.Len() != 0 {
		t.Fatalf("unsafe CLI profile accepted: code=%d out=%s", code, out.String())
	}
}
