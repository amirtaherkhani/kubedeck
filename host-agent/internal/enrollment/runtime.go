package enrollment

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"time"

	"github.com/amirtaherkhani/kuchdesk/host-agent/internal/doctor"
	"github.com/amirtaherkhani/kuchdesk/host-agent/internal/infisical"
)

// OpenRuntime never creates a state directory or imports credentials implicitly.
func OpenRuntime(origin, policyPath, stateDir string, machine *infisical.Client) (*Controller, *Store, error) {
	p, e := LoadPolicy(policyPath)
	if e != nil {
		return nil, nil, e
	}
	h, e := NewTransport(origin, nil)
	if e != nil {
		return nil, nil, e
	}
	store, e := OpenStore(stateDir)
	if e != nil {
		return nil, nil, e
	}
	fail := func(e error) (*Controller, *Store, error) { store.Close(); return nil, nil, e }
	var session *Session
	if store.Exists("session.json") {
		var d SessionData
		if e = store.Load("session.json", &d); e != nil {
			return fail(e)
		}
		if d.Origin != origin || d.OrganizationID != p.OrganizationID || d.Version != p.Version {
			return fail(errors.New("session_scope_mismatch"))
		}
		session, e = NewSession(d, store, refreshSession(h))
		if e != nil {
			return fail(e)
		}
	}
	a := &Access{Path: policyPath, Binding: p}
	api := &API{HTTP: h, Machine: machine, Human: session, OrganizationID: p.OrganizationID, Version: p.Version}
	c, e := NewController(api, store, a, policyPath, p)
	if e != nil {
		return fail(e)
	}
	return c, store, nil
}

// ImportSession is a local explicit handoff, not a browser credential extractor.
// Caller must stop the controller first; OpenStore enforces the single writer.
func ImportSession(source, origin string, p Policy, store *Store) error {
	b, e := readPrivate(source)
	if e != nil {
		return e
	}
	var d SessionData
	if e = decode(b, &d); e != nil {
		return e
	}
	if d.Origin != origin || d.OrganizationID != p.OrganizationID || d.Version != p.Version || d.Pending || d.Expired {
		return errors.New("session_scope_mismatch")
	}
	expiry, e := tokenExpiry(d.AccessToken, p.OrganizationID)
	if e != nil || !time.Now().Before(expiry) {
		return ErrExpired
	}
	if len(d.AccessToken) > 16<<10 || len(d.RefreshToken) > 16<<10 {
		return ErrEnrollment
	}
	return store.Save("session.json", d)
}

// Diagnostics adds value-free control-plane health to the existing Doctor report.
type Diagnostics struct {
	Controller *Controller
	Base       doctor.Service
}

func (d Diagnostics) Run(ctx context.Context, cfg doctor.Config) (doctor.Report, error) {
	r, e := d.Base.Run(ctx, cfg)
	if e != nil {
		return r, e
	}
	check := DiagnosticCheck(d.Controller.Snapshot())
	r.Summary.Total++
	if check.Status != "ok" {
		r.Summary.Warnings++
		r.Healthy = false
		r.Findings = append(r.Findings, doctor.Finding{CheckID: check.ID, Severity: "warn", Problem: check.Message, Recommendation: check.Recommendation})
	} else {
		r.Summary.Passed++
	}
	r.Checks = append(r.Checks, check)
	return r, nil
}

// EmitReport serializes only public status types, never SessionData or upstream bodies.
func EmitReport(r Report) { _ = json.NewEncoder(os.Stdout).Encode(r) }

func DiagnosticCheck(s Report) doctor.Check {
	state := s.Status
	if state == "" {
		state = "starting"
	}
	check := doctor.Check{ID: "infisical-enrollment", Component: "infisical", Status: "ok", Message: state, Source: "host-controller", Context: "local", ObservedAt: time.Now(), Evidence: map[string]any{"projects": len(s.Results)}, Recommendation: "Review local enrollment policy and session status; do not send credentials to Doctor"}
	if state != "ready" && state != "dry_run" {
		check.Status = "warn"
	}
	return check
}
