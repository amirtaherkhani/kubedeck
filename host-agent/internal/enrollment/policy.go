// Package enrollment implements the local-only Infisical enrollment control plane.
package enrollment

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"regexp"
	"sync"
	"time"
)

var identifier = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9._-]{0,127}$`)
var ErrDenied = errors.New("denied")
var ErrEnrollment = errors.New("enrollment_required")
var ErrExpired = errors.New("session_expired")
var ErrUnsupported = errors.New("unsupported_version")
var ErrUnavailable = errors.New("temporarily_unavailable")

// Policy is local administrator intent, not authority inferred from discovery.
// Changing organization, identity or version requires restarting the controller.
type Policy struct {
	Version           string   `json:"version"`
	OrganizationID    string   `json:"organizationId"`
	HostIdentityID    string   `json:"hostIdentityId"`
	K8IdentityID      string   `json:"k8IdentityId,omitempty"`
	K8IdentityName    string   `json:"k8IdentityName,omitempty"`
	CreateK8Identity  bool     `json:"createK8Identity"`
	Include           []string `json:"include"`
	Exclude           []string `json:"exclude"`
	AllProjects       bool     `json:"allProjects"`
	Apply             bool     `json:"apply"`
	RepairRevocations bool     `json:"repairRevocations"`
	IntervalSeconds   int      `json:"intervalSeconds"`
	MaxProjects       int      `json:"maxProjects"`
}

func (p Policy) Validate() error {
	if p.Version != "v0.151.0" && p.Version != "v0.166.3" {
		return ErrUnsupported
	}
	if !identifier.MatchString(p.OrganizationID) || !identifier.MatchString(p.HostIdentityID) || (p.K8IdentityID != "" && !identifier.MatchString(p.K8IdentityID)) || p.K8IdentityID == p.HostIdentityID {
		return errors.New("invalid_identity_policy")
	}
	if p.K8IdentityID == "" && !identifier.MatchString(p.K8IdentityName) {
		return errors.New("k8_identity_required")
	}
	if p.IntervalSeconds < 30 || p.IntervalSeconds > 3600 || p.MaxProjects < 1 || p.MaxProjects > 1000 {
		return errors.New("invalid_reconciliation_bounds")
	}
	for _, ids := range [][]string{p.Include, p.Exclude} {
		for _, id := range ids {
			if !identifier.MatchString(id) {
				return errors.New("invalid_project_policy")
			}
		}
	}
	return nil
}
func (p Policy) Allows(id string) bool {
	for _, v := range p.Exclude {
		if v == id {
			return false
		}
	}
	if p.AllProjects {
		return true
	}
	for _, v := range p.Include {
		if v == id {
			return true
		}
	}
	return false
}
func decode(data []byte, out any) error {
	d := json.NewDecoder(bytes.NewReader(data))
	d.DisallowUnknownFields()
	if d.Decode(out) != nil || d.Decode(new(any)) != io.EOF {
		return errors.New("invalid_local_state")
	}
	return nil
}
func LoadPolicy(path string) (Policy, error) {
	var p Policy
	b, err := readPrivate(path)
	if err != nil {
		return p, err
	}
	if err = decode(b, &p); err != nil {
		return p, err
	}
	return p, p.Validate()
}

// Access publishes only currently verified projects. Invalid policy clears access.
// Requests re-read policy, so excludes/invalid edits take effect without a restart.
type Access struct {
	Binding  Policy
	Path     string
	mu       sync.RWMutex
	verified map[string]bool
}

func (a *Access) Publish(ids map[string]bool) { a.mu.Lock(); defer a.mu.Unlock(); a.verified = ids }

// Withdraw immediately removes eligibility while this project's authority is checked.
func (a *Access) Withdraw(id string) {
	a.mu.Lock()
	defer a.mu.Unlock()
	delete(a.verified, id)
}

func (a *Access) Projects() map[string]bool {
	p, e := LoadPolicy(a.Path)
	if e != nil || !sameBinding(p, a.Binding) {
		return map[string]bool{}
	}
	a.mu.RLock()
	defer a.mu.RUnlock()
	out := map[string]bool{}
	for id, ok := range a.verified {
		if ok && p.Allows(id) {
			out[id] = true
		}
	}
	return out
}
func delay(p Policy, failures int) time.Duration {
	d := time.Duration(p.IntervalSeconds) * time.Second
	if d < 30*time.Second {
		d = 30 * time.Second
	}
	for i := 0; i < failures && d < 15*time.Minute; i++ {
		d *= 2
	}
	if d > 15*time.Minute {
		d = 15 * time.Minute
	}
	return d
}
