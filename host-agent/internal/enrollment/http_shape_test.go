package enrollment

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestBodylessAuthRequestsDoNotAdvertiseEmptyJSON(t *testing.T) {
	remote := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		if r.Header.Get("Content-Type") == "application/json" && len(body) == 0 {
			w.WriteHeader(500)
			fmt.Fprint(w, `{"code":"FST_ERR_CTP_EMPTY_JSON_BODY"}`)
			return
		}
		fmt.Fprint(w, `{"message":"Authenticated"}`)
	}))
	defer remote.Close()
	h, _ := NewTransport(remote.URL, remote.Client())
	h.MinInterval = 0
	for _, method := range []string{"POST", "GET"} {
		var result struct {
			Message string `json:"message"`
		}
		if e := h.Call(context.Background(), method, "/api/v1/auth/checkAuth", nil, nil, nil, &result); e != nil {
			t.Fatalf("%s empty-body protocol rejected: %v", method, e)
		}
	}
}

func TestAuthFailureDiagnosticsContainOnlySafeRequestMetadata(t *testing.T) {
	remote := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(500)
		fmt.Fprint(w, `private-token-and-body`)
	}))
	defer remote.Close()
	h, _ := NewTransport(remote.URL, remote.Client())
	h.MinInterval = 0
	report, e := verifyHuman(context.Background(), h, testPolicy(), testJWT(time.Now().Add(time.Hour)))
	if e == nil || report.HTTPStatus != 500 || report.RequestMethod != "POST" || report.RequestEndpoint != "/api/v1/auth/checkAuth" {
		t.Fatalf("missing safe diagnostics: %+v %v", report, e)
	}
	encoded, _ := json.Marshal(report)
	if strings.Contains(string(encoded)+e.Error(), "private-token") {
		t.Fatal("upstream payload exposed")
	}
}
