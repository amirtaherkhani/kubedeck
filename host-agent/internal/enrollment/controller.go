package enrollment

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/amirtaherkhani/kuchdesk/host-agent/internal/infisical"
)

type Backend interface {
	Projects(context.Context, int) ([]Project, error)
	Identities(context.Context) ([]Identity, error)
	CreateIdentity(context.Context, string) (Identity, error)
	Memberships(context.Context, string, bool) ([]Membership, error)
	EnrollHuman(context.Context, string) error
	SetMembership(context.Context, string, string, string, bool, bool) error
	ReadOnlyIdentity(context.Context, Project, string) error
}
type Entry struct {
	HostSeen bool   `json:"hostSeen"`
	K8Seen   bool   `json:"k8Seen"`
	Pending  string `json:"pending,omitempty"`
	Revoked  bool   `json:"revoked"`
}
type Ledger struct {
	OrganizationID  string           `json:"organizationId"`
	HostIdentityID  string           `json:"hostIdentityId"`
	K8IdentityID    string           `json:"k8IdentityId,omitempty"`
	IdentityPending bool             `json:"identityPending"`
	Projects        map[string]Entry `json:"projects"`
}
type Result struct {
	ProjectID string `json:"projectId,omitempty"`
	Status    string `json:"status"`
	Action    string `json:"action,omitempty"`
}
type Report struct {
	HumanAuthority string        `json:"humanAuthority"`
	Status         string        `json:"status"`
	Results        []Result      `json:"results"`
	ObservedAt     time.Time     `json:"observedAt"`
	RetryAfter     time.Duration `json:"-"`
}
type Controller struct {
	Backend    Backend
	Store      *Store
	Access     *Access
	PolicyPath string
	Initial    Policy
	ledger     Ledger
	mu         sync.Mutex
	last       Report
}

func NewController(api Backend, store *Store, access *Access, path string, p Policy) (*Controller, error) {
	if e := p.Validate(); e != nil {
		return nil, e
	}
	c := &Controller{Backend: api, Store: store, Access: access, PolicyPath: path, Initial: p, ledger: Ledger{OrganizationID: p.OrganizationID, HostIdentityID: p.HostIdentityID, Projects: map[string]Entry{}}}
	if store.Exists("ledger.json") {
		if e := store.Load("ledger.json", &c.ledger); e != nil {
			return nil, e
		}
	}
	if c.ledger.OrganizationID != p.OrganizationID || c.ledger.HostIdentityID != p.HostIdentityID || (p.K8IdentityID != "" && c.ledger.K8IdentityID != "" && p.K8IdentityID != c.ledger.K8IdentityID) {
		return nil, errors.New("state_scope_mismatch")
	}
	if c.ledger.Projects == nil {
		c.ledger.Projects = map[string]Entry{}
	}
	return c, nil
}
func sameBinding(a, b Policy) bool {
	return a.Version == b.Version && a.OrganizationID == b.OrganizationID && a.HostIdentityID == b.HostIdentityID && a.K8IdentityID == b.K8IdentityID && a.K8IdentityName == b.K8IdentityName
}
func classify(e error) string {
	var s *infisical.APIError
	var h *StatusError
	switch {
	case errors.Is(e, ErrExpired):
		return "session_expired"
	case errors.Is(e, ErrEnrollment):
		return "enrollment_required"
	case errors.Is(e, ErrUnsupported):
		return "unsupported_version"
	case errors.Is(e, ErrDenied):
		return "denied"
	case errors.As(e, &s) && s.StatusCode == 403:
		return "denied"
	case errors.As(e, &s) && s.StatusCode == 401:
		return "machine_auth_required"
	case errors.As(e, &h) && h.Code == 429:
		return "rate_limited"
	case errors.As(e, &s) && s.StatusCode == 429:
		return "rate_limited"
	default:
		return "temporarily_unavailable"
	}
}
func forbidden(e error) bool {
	var s *infisical.APIError
	return errors.Is(e, ErrDenied) || (errors.As(e, &s) && s.StatusCode == 403)
}
func (c *Controller) save() error { return c.Store.Save("ledger.json", c.ledger) }
func (c *Controller) Snapshot() Report {
	c.mu.Lock()
	defer c.mu.Unlock()
	r := c.last
	r.Results = append([]Result(nil), r.Results...)
	return r
}
func (c *Controller) Cycle(ctx context.Context) Report {
	c.mu.Lock()
	defer c.mu.Unlock()
	report := Report{HumanAuthority: "not_checked", Status: "ready", ObservedAt: time.Now(), Results: []Result{}}
	verified := map[string]bool{}
	defer func() { c.Access.Publish(verified); c.last = report }()
	fail := func(e error) Report {
		report.Status = classify(e)
		var h *StatusError
		if errors.As(e, &h) {
			report.RetryAfter = h.RetryAfter
		}
		var upstream *infisical.APIError
		if errors.As(e, &upstream) {
			report.RetryAfter = upstream.RetryAfter
		}
		return report
	}
	p, e := LoadPolicy(c.PolicyPath)
	if e != nil {
		report.Status = "invalid_policy"
		return report
	}
	if !sameBinding(p, c.Initial) {
		report.Status = "restart_required"
		return report
	}
	projects, e := c.Backend.Projects(ctx, p.MaxProjects)
	if e != nil {
		return fail(e)
	}
	identities, e := c.Backend.Identities(ctx)
	if e != nil {
		return fail(e)
	}
	hostFound := false
	k8ID := p.K8IdentityID
	if k8ID == "" {
		k8ID = c.ledger.K8IdentityID
	}
	matches := []Identity{}
	for _, i := range identities {
		if i.ID == p.HostIdentityID {
			hostFound = true
		}
		if (k8ID != "" && i.ID == k8ID) || (k8ID == "" && i.Name == p.K8IdentityName) {
			matches = append(matches, i)
		}
	}
	if !hostFound {
		return fail(ErrEnrollment)
	}
	if len(matches) > 1 {
		return fail(ErrDenied)
	}
	if len(matches) == 0 {
		if k8ID != "" || !p.CreateK8Identity || c.ledger.IdentityPending {
			return fail(ErrEnrollment)
		}
		if !p.Apply {
			report.Status = "dry_run"
			report.Results = append(report.Results, Result{Status: "planned", Action: "create_k8_identity"})
			return report
		}
		c.ledger.IdentityPending = true
		if e = c.save(); e != nil {
			return fail(e)
		}
		if e = c.authorizeWrite(p, ""); e != nil {
			return fail(e)
		}
		i, err := c.Backend.CreateIdentity(ctx, p.K8IdentityName)
		if err != nil {
			return fail(err)
		}
		k8ID = i.ID
	} else {
		k8ID = matches[0].ID
	}
	if k8ID == p.HostIdentityID || !identifier.MatchString(k8ID) {
		return fail(ErrDenied)
	}
	c.ledger.K8IdentityID = k8ID
	c.ledger.IdentityPending = false
	if e = c.save(); e != nil {
		return fail(e)
	}
	for _, project := range projects {
		if !p.Allows(project.ID) {
			continue
		}
		if ctx.Err() != nil {
			return fail(ErrUnavailable)
		}
		// Re-read before each project; an invalid edit or exclude denies new work.
		latest, err := LoadPolicy(c.PolicyPath)
		if err != nil || !sameBinding(latest, p) {
			report.Status = "invalid_policy"
			return report
		}
		p = latest
		if !p.Allows(project.ID) {
			continue
		}
		c.Access.Withdraw(project.ID)
		result, err := c.project(ctx, p, project, k8ID)
		report.Results = append(report.Results, result)
		if err != nil {
			report.Status = "partial_failure"
			var h *StatusError
			if errors.As(err, &h) && h.RetryAfter > report.RetryAfter {
				report.RetryAfter = h.RetryAfter
			}
			var upstream *infisical.APIError
			if errors.As(err, &upstream) && upstream.RetryAfter > report.RetryAfter {
				report.RetryAfter = upstream.RetryAfter
			}
			if classify(err) == "rate_limited" {
				return report
			}
			continue
		}
		if result.Status == "ready" {
			verified[project.ID] = true
		} else if report.Status == "ready" {
			report.Status = result.Status
		}
	}
	if !p.Apply && report.Status == "ready" {
		report.Status = "dry_run"
	}
	return report
}
func findMembership(items []Membership, id string) (Membership, bool) {
	for _, m := range items {
		if m.IdentityID == id {
			return m, true
		}
	}
	return Membership{}, false
}
func (c *Controller) project(ctx context.Context, p Policy, project Project, k8 string) (Result, error) {
	r := Result{ProjectID: project.ID, Status: "ready"}
	entry := c.ledger.Projects[project.ID]
	finish := func(e error) (Result, error) { r.Status = classify(e); return r, e }
	if entry.Revoked && !p.RepairRevocations {
		r.Status = "revoked_preserved"
		return r, nil
	}
	items, e := c.Backend.Memberships(ctx, project.ID, false)
	human := false
	if e != nil {
		if !forbidden(e) {
			return finish(e)
		}
		if entry.HostSeen && !p.RepairRevocations {
			entry.Revoked = true
			c.ledger.Projects[project.ID] = entry
			if e = c.save(); e != nil {
				return finish(e)
			}
			r.Status = "revoked_preserved"
			return r, nil
		}
		if !p.Apply {
			r.Status = "enrollment_required"
			return r, nil
		}
		// An ambiguous previous enrollment is observed before any retry.
		items, e = c.Backend.Memberships(ctx, project.ID, true)
		if forbidden(e) {
			if entry.Pending != "" && !p.RepairRevocations {
				r.Status = "enrollment_required"
				return r, nil
			}
			entry.Pending = "human_enrollment"
			c.ledger.Projects[project.ID] = entry
			if e = c.save(); e != nil {
				return finish(e)
			}
			if e = c.authorizeWrite(p, project.ID); e != nil {
				return finish(e)
			}
			if e = c.Backend.EnrollHuman(ctx, project.ID); e != nil {
				return finish(e)
			}
			items, e = c.Backend.Memberships(ctx, project.ID, true)
		}
		if e != nil {
			return finish(e)
		}
		human = true
	}
	for _, target := range []struct {
		id, role string
		seen     *bool
	}{{p.HostIdentityID, "admin", &entry.HostSeen}, {k8, "viewer", &entry.K8Seen}} {
		m, exists := findMembership(items, target.id)
		if exists && m.Exact(target.role) {
			// Conclusive read-back completes this write, including a lost response.
			if entry.Pending == target.id {
				entry.Pending = ""
			}
			*target.seen = true
			c.ledger.Projects[project.ID] = entry
			if e = c.save(); e != nil {
				return finish(e)
			}
			if target.id == p.HostIdentityID && human {
				// Existing membership is not evidence that machine auth works.
				human = false
				items, e = c.Backend.Memberships(ctx, project.ID, false)
				if e != nil {
					return finish(e)
				}
			}
			continue
		}
		if (*target.seen || exists) && !p.RepairRevocations {
			entry.Revoked = true
			c.ledger.Projects[project.ID] = entry
			if e = c.save(); e != nil {
				return finish(e)
			}
			r.Status = "revoked_preserved"
			return r, nil
		}
		if entry.Pending != "" && entry.Pending != "human_enrollment" && !p.RepairRevocations {
			r.Status = "enrollment_required"
			return r, nil
		}
		if !p.Apply {
			r.Status = "dry_run"
			r.Action = "enroll_machine_memberships"
			return r, nil
		}
		entry.Pending = target.id
		c.ledger.Projects[project.ID] = entry
		if e = c.save(); e != nil {
			return finish(e)
		}
		if e = c.authorizeWrite(p, project.ID); e != nil {
			return finish(e)
		}
		if e = c.Backend.SetMembership(ctx, project.ID, target.id, target.role, exists, human); e != nil {
			return finish(e)
		}
		// Read back exact roles before granting bridge access or moving to the next write.
		items, e = c.Backend.Memberships(ctx, project.ID, human)
		if e != nil {
			return finish(e)
		}
		actual, ok := findMembership(items, target.id)
		if !ok || !actual.Exact(target.role) {
			return finish(ErrDenied)
		}
		*target.seen = true
		entry.Pending = ""
		c.ledger.Projects[project.ID] = entry
		if e = c.save(); e != nil {
			return finish(e)
		}
		if target.id == p.HostIdentityID {
			human = false
			items, e = c.Backend.Memberships(ctx, project.ID, false)
			if e != nil {
				return finish(e)
			}
		}
	}
	if e = c.Backend.ReadOnlyIdentity(ctx, project, k8); e != nil {
		return finish(e)
	}
	entry.Pending = ""
	entry.Revoked = false
	c.ledger.Projects[project.ID] = entry
	if e = c.save(); e != nil {
		return finish(e)
	}
	return r, nil
}
func (c *Controller) Run(ctx context.Context, emit func(Report)) {
	failures := 0
	for ctx.Err() == nil {
		cycle, cancel := context.WithTimeout(ctx, 2*time.Minute)
		r := c.Cycle(cycle)
		cancel()
		if emit != nil {
			emit(r)
		}
		if r.Status == "ready" || r.Status == "dry_run" {
			failures = 0
		} else {
			failures++
		}
		if !waitForNextCycle(ctx, func() (Policy, error) { return LoadPolicy(c.PolicyPath) }, c.Initial, failures, r.RetryAfter) {
			return
		}
	}
}

func (r Report) String() string {
	return fmt.Sprintf("enrollment status=%s projects=%d", r.Status, len(r.Results))
}

// Recheck local authority immediately before each external mutation.
func (c *Controller) authorizeWrite(observed Policy, project string) error {
	latest, e := LoadPolicy(c.PolicyPath)
	if e != nil || !sameBinding(latest, observed) || !latest.Apply {
		return ErrDenied
	}
	if observed.RepairRevocations && !latest.RepairRevocations {
		return ErrDenied
	}
	if project == "" {
		if !latest.CreateK8Identity {
			return ErrDenied
		}
	} else if !latest.Allows(project) {
		return ErrDenied
	}
	return nil
}
