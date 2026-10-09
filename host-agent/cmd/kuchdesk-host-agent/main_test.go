package main

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestReconcileDNSUsesConfiguredZoneAndRemainsIdempotent(t *testing.T) {
	cfg, err := parseConfig([]string{"-kube-context", "kind-team", "-zone", "dev.example"})
	if err != nil {
		t.Fatal(err)
	}
	ip := net.ParseIP("10.2.3.4")
	for _, recordExists := range []bool{false, true} {
		t.Run(fmt.Sprintf("record_exists_%t", recordExists), func(t *testing.T) {
			adds := 0
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				switch r.URL.Path {
				case "/api/zones/list":
					fmt.Fprint(w, `{"status":"ok","response":{"zones":[{"name":"dev.example"}]}}`)
				case "/api/zones/records/get":
					if r.URL.Query().Get("zone") != "dev.example" || r.URL.Query().Get("domain") != "*.dev.example" {
						t.Errorf("unexpected record query: %s", r.URL.RawQuery)
					}
					if recordExists {
						fmt.Fprint(w, `{"status":"ok","response":{"records":[{"name":"*.dev.example","type":"A","ttl":30,"rData":{"ipAddress":"10.2.3.4"}}]}}`)
					} else {
						fmt.Fprint(w, `{"status":"ok","response":{"records":[]}}`)
					}
				case "/api/zones/records/add":
					adds++
					if r.Method != http.MethodPost || r.FormValue("zone") != "dev.example" || r.FormValue("domain") != "*.dev.example" || r.FormValue("ipAddress") != ip.String() {
						t.Errorf("unexpected record write: method=%s form=%v", r.Method, r.Form)
					}
					fmt.Fprint(w, `{"status":"ok"}`)
				default:
					t.Errorf("unexpected API path: %s", r.URL.Path)
					http.Error(w, "unexpected path", http.StatusNotFound)
				}
			}))
			defer server.Close()

			client := &apiClient{base: server.URL, http: &http.Client{Timeout: time.Second}}
			if err := reconcileDNS(context.Background(), client, cfg, ip); err != nil {
				t.Fatal(err)
			}
			wantAdds := 1
			if recordExists {
				wantAdds = 0
			}
			if adds != wantAdds {
				t.Fatalf("record writes = %d, want %d", adds, wantAdds)
			}
		})
	}
}
