package enrollment

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

type fakeAPI struct {
	projects                 []Project
	identities               []Identity
	members                  map[string][]Membership
	human                    map[string]bool
	writes, enrolls, creates int
	fail                     string
	extra                    bool
}

func (f *fakeAPI) Projects(context.Context, int) ([]Project, error) { return f.projects, nil }
func (f *fakeAPI) Identities(context.Context) ([]Identity, error)   { return f.identities, nil }
func (f *fakeAPI) CreateIdentity(_ context.Context, name string) (Identity, error) {
	f.creates++
	i := Identity{ID: "created-k8", Name: name}
	f.identities = append(f.identities, i)
	return i, nil
}
func (f *fakeAPI) Memberships(_ context.Context, p string, human bool) ([]Membership, error) {
	if human && f.human[p] {
		return f.members[p], nil
	}
	for _, m := range f.members[p] {
		if m.IdentityID == "host" && m.Exact("admin") {
			return f.members[p], nil
		}
	}
	return nil, ErrDenied
}
func (f *fakeAPI) EnrollHuman(_ context.Context, p string) error {
	f.enrolls++
	f.human[p] = true
	return nil
}
func (f *fakeAPI) SetMembership(_ context.Context, p, id, role string, exists, human bool) error {
	f.writes++
	if f.fail == p {
		return ErrUnavailable
	}
	m := Membership{IdentityID: id, Roles: []Role{{Role: role}}}
	for i, x := range f.members[p] {
		if x.IdentityID == id {
			f.members[p][i] = m
			return nil
		}
	}
	f.members[p] = append(f.members[p], m)
	return nil
}
func (f *fakeAPI) ReadOnlyIdentity(context.Context, Project, string) error {
	if f.extra {
		return ErrDenied
	}
	return nil
}
func testPolicy() Policy {
	return Policy{Version: "v0.151.0", OrganizationID: "org", HostIdentityID: "host", K8IdentityID: "k8", Include: []string{"one", "two"}, Apply: true, IntervalSeconds: 30, MaxProjects: 20}
}
func putPolicy(t *testing.T, path string, p Policy) {
	t.Helper()
	b, _ := json.Marshal(p)
	if e := os.WriteFile(path, b, 0600); e != nil {
		t.Fatal(e)
	}
}
func newTestController(t *testing.T, p Policy) (*Controller, *fakeAPI) {
	t.Helper()
	dir := t.TempDir()
	_ = os.Chmod(dir, 0700)
	s, e := OpenStore(dir)
	if e != nil {
		t.Fatal(e)
	}
	t.Cleanup(func() { s.Close() })
	path := filepath.Join(dir, "policy.json")
	putPolicy(t, path, p)
	f := &fakeAPI{projects: []Project{{ID: "one", Name: "Unrelated display", Slug: "one"}, {ID: "two", Name: "第二", Slug: "two"}}, identities: []Identity{{ID: "host"}, {ID: "k8"}}, members: map[string][]Membership{}, human: map[string]bool{}}
	a := &Access{Path: path, Binding: p}
	c, e := NewController(f, s, a, path, p)
	if e != nil {
		t.Fatal(e)
	}
	return c, f
}
func TestControllerStartupRepeatedRenameAndRevocation(t *testing.T) {
	c, f := newTestController(t, testPolicy())
	r := c.Cycle(context.Background())
	if r.Status != "ready" || f.enrolls != 2 || f.writes != 4 || len(c.Access.Projects()) != 2 {
		t.Fatalf("first cycle %v", r)
	}
	f.projects[0].Name = "renamed"
	f.projects[0].Slug = "renamed-slug"
	r = c.Cycle(context.Background())
	if r.Status != "ready" || f.writes != 4 || f.enrolls != 2 {
		t.Fatal("rename or repeat duplicated grants")
	}
	f.members["one"] = f.members["one"][:1]
	r = c.Cycle(context.Background())
	if len(c.Access.Projects()) != 1 || f.writes != 4 {
		t.Fatal("revocation repaired without policy")
	}
	c2, e := NewController(f, c.Store, c.Access, c.PolicyPath, c.Initial)
	if e != nil {
		t.Fatal(e)
	}
	c2.Cycle(context.Background())
	if f.writes != 4 {
		t.Fatal("restart forgot revocation")
	}
}
func TestPolicyReloadExcludesAndMalformedFailClosed(t *testing.T) {
	c, f := newTestController(t, testPolicy())
	c.Cycle(context.Background())
	p := testPolicy()
	p.Exclude = []string{"one"}
	putPolicy(t, c.PolicyPath, p)
	if c.Access.Projects()["one"] {
		t.Fatal("exclude did not apply on request")
	}
	c.Cycle(context.Background())
	if f.writes != 4 {
		t.Fatal("exclude changed grants")
	}
	_ = os.WriteFile(c.PolicyPath, []byte(`{`), 0600)
	if len(c.Access.Projects()) != 0 || c.Cycle(context.Background()).Status != "invalid_policy" {
		t.Fatal("malformed policy allowed access")
	}
}
func TestDryRunAndExistingIdentityAndPartialFailure(t *testing.T) {
	p := testPolicy()
	p.Apply = false
	c, f := newTestController(t, p)
	c.Cycle(context.Background())
	if f.writes+f.enrolls+f.creates != 0 {
		t.Fatal("dry run mutated upstream")
	}
	p.Apply = true
	putPolicy(t, c.PolicyPath, p)
	f.fail = "one"
	r := c.Cycle(context.Background())
	if r.Status != "partial_failure" || !c.Access.Projects()["two"] || c.Access.Projects()["one"] {
		t.Fatal("partial failure not isolated")
	}
	writes := f.writes
	f.fail = ""
	c.Cycle(context.Background())
	if f.writes != writes {
		t.Fatal("ambiguous write replayed")
	}
}
func TestIdentityReuseAndAmbiguousCreation(t *testing.T) {
	p := testPolicy()
	p.K8IdentityID = ""
	p.K8IdentityName = "shared-reader"
	p.CreateK8Identity = true
	c, f := newTestController(t, p)
	f.identities[1].Name = p.K8IdentityName
	c.Cycle(context.Background())
	if f.creates != 0 {
		t.Fatal("existing identity duplicated")
	}
	c2, f2 := newTestController(t, p)
	f2.identities = f2.identities[:1]
	c2.ledger.IdentityPending = true
	r := c2.Cycle(context.Background())
	if r.Status != "enrollment_required" || f2.creates != 0 {
		t.Fatal("uncertain create replayed")
	}
}
func TestStoreLockPermissionsAndRecovery(t *testing.T) {
	dir := t.TempDir()
	_ = os.Chmod(dir, 0700)
	s, e := OpenStore(dir)
	if e != nil {
		t.Fatal(e)
	}
	defer s.Close()
	if _, e = OpenStore(dir); e == nil {
		t.Fatal("second process writer allowed")
	}
	if e = s.Save("ledger.json", Ledger{OrganizationID: "org"}); e != nil {
		t.Fatal(e)
	}
	var l Ledger
	if e = s.Load("ledger.json", &l); e != nil || l.OrganizationID != "org" {
		t.Fatal("atomic state not readable")
	}
	if e = os.Chmod(filepath.Join(dir, "ledger.json"), 0644); e != nil {
		t.Fatal(e)
	}
	if e = s.Load("ledger.json", &l); e == nil {
		t.Fatal("unsafe file accepted")
	}
}

func TestExplicitRepairAndAdditionalPrivileges(t *testing.T) {
	c, f := newTestController(t, testPolicy())
	c.Cycle(context.Background())
	f.members["one"] = f.members["one"][:1]
	c.Cycle(context.Background())
	p := testPolicy()
	p.RepairRevocations = true
	putPolicy(t, c.PolicyPath, p)
	c.Cycle(context.Background())
	if !c.Access.Projects()["one"] {
		t.Fatal("explicit repair not applied")
	}
	f.extra = true
	c.Cycle(context.Background())
	if len(c.Access.Projects()) != 0 {
		t.Fatal("additional privileges accepted as readonly")
	}
}

func TestControllerCreatesExactlyOneK8Identity(t *testing.T) {
	p := testPolicy()
	p.K8IdentityID = ""
	p.K8IdentityName = "configured-reader"
	p.CreateK8Identity = true
	c, f := newTestController(t, p)
	f.identities = f.identities[:1]
	for i := 0; i < 3; i++ {
		if r := c.Cycle(context.Background()); r.Status != "ready" {
			t.Fatal(r)
		}
	}
	if f.creates != 1 || f.writes != 4 {
		t.Fatal("identity or membership duplication")
	}
}

func TestImmediateWriteAuthorizationSeesPolicyRevocation(t *testing.T) {
	p := testPolicy()
	c, _ := newTestController(t, p)
	changed := p
	changed.Exclude = []string{"one"}
	putPolicy(t, c.PolicyPath, changed)
	if e := c.authorizeWrite(p, "one"); e == nil {
		t.Fatal("revoked scope permitted a write")
	}
	changed.Apply = false
	putPolicy(t, c.PolicyPath, changed)
	if e := c.authorizeWrite(p, "two"); e == nil {
		t.Fatal("disabled apply permitted a write")
	}
}
