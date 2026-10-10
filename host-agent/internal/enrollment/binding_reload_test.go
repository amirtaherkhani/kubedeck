package enrollment

import (
	"context"
	"testing"
)

func TestBindingChangesImmediatelyDenyAccessAndRequireRestart(t *testing.T) {
	changes := map[string]func(*Policy){
		"version":       func(p *Policy) { p.Version = "v0.166.3" },
		"organization":  func(p *Policy) { p.OrganizationID = "other-org" },
		"host-identity": func(p *Policy) { p.HostIdentityID = "other-host" },
		"k8-identity":   func(p *Policy) { p.K8IdentityID = "other-k8" },
		"k8-name":       func(p *Policy) { p.K8IdentityName = "other-name" },
	}
	for name, change := range changes {
		t.Run(name, func(t *testing.T) {
			p := testPolicy()
			c, f := newTestController(t, p)
			if got := c.Cycle(context.Background()); got.Status != "ready" || len(c.Access.Projects()) == 0 {
				t.Fatal("initial reconciliation did not establish access")
			}
			writes, enrolls := f.writes, f.enrolls
			change(&p)
			putPolicy(t, c.PolicyPath, p)
			if len(c.Access.Projects()) != 0 {
				t.Fatal("changed binding retained request access")
			}
			if got := c.Cycle(context.Background()); got.Status != "restart_required" {
				t.Fatalf("expected restart_required, got %s", got.Status)
			}
			if f.writes != writes || f.enrolls != enrolls {
				t.Fatal("changed binding performed mutations")
			}
		})
	}
}
