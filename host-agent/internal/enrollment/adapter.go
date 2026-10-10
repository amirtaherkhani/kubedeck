package enrollment

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/url"
	"strconv"

	"github.com/amirtaherkhani/kuchdesk/host-agent/internal/infisical"
)

type Project struct {
	ID             string `json:"id"`
	Name           string `json:"name"`
	Slug           string `json:"slug"`
	OrganizationID string `json:"orgId"`
}
type Identity struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}
type Role struct {
	Role         string  `json:"role"`
	IsTemporary  bool    `json:"isTemporary"`
	CustomRoleID *string `json:"customRoleId"`
}
type Membership struct {
	IdentityID string `json:"identityId"`
	Roles      []Role `json:"roles"`
}

func (m Membership) Exact(role string) bool {
	return len(m.Roles) == 1 && m.Roles[0].Role == role && !m.Roles[0].IsTemporary && m.Roles[0].CustomRoleID == nil
}

// API is a typed, version-pinned REST adapter for routes absent from the Go SDK.
// Routine reads/writes use the machine client. Only enrollment uses a human JWT.
type API struct {
	HTTP           *Transport
	Machine        *infisical.Client
	Human          *Session
	OrganizationID string
	Version        string
}

func (a *API) call(ctx context.Context, human bool, method, path string, q url.Values, body, out any) error {
	if a.Version != "v0.151.0" {
		return ErrUnsupported
	}
	if !human {
		if e := a.HTTP.Wait(ctx); e != nil {
			return e
		}
		if a.Machine == nil {
			return ErrEnrollment
		}
		return a.Machine.EnrollmentRequest(ctx, method, path, q, body, out)
	}
	if a.Human == nil {
		return ErrEnrollment
	}
	token, e := a.Human.Token(ctx)
	if e != nil {
		return e
	}
	headers := http.Header{}
	headers.Set("Authorization", "Bearer "+token)
	e = a.HTTP.Call(ctx, method, path, q, headers, body, out)
	if errors.Is(e, ErrExpired) {
		a.Human.Expire(token)
	}
	return e
}
func (a *API) Projects(ctx context.Context, max int) ([]Project, error) {
	result := []Project{}
	seen := map[string]bool{}
	for offset := 0; offset < max; {
		n := 100
		if max-offset < n {
			n = max - offset
		}
		var res struct {
			Projects []Project `json:"projects"`
			Count    *int      `json:"count"`
		}
		if e := a.call(ctx, false, "GET", "/api/v1/organization-admin/projects", url.Values{"offset": {strconv.Itoa(offset)}, "limit": {strconv.Itoa(n)}}, nil, &res); e != nil {
			return nil, e
		}
		if res.Count == nil || *res.Count > max || *res.Count < offset+len(res.Projects) || len(res.Projects) > n {
			return nil, errors.New("discovery_limit_or_contract")
		}
		for _, p := range res.Projects {
			if !identifier.MatchString(p.ID) || seen[p.ID] || p.OrganizationID != a.OrganizationID {
				return nil, errors.New("discovery_scope_or_pagination")
			}
			seen[p.ID] = true
			result = append(result, p)
		}
		offset += len(res.Projects)
		if offset >= *res.Count {
			return result, nil
		}
		if len(res.Projects) == 0 {
			return nil, ErrUnavailable
		}
	}
	return nil, errors.New("discovery_limit")
}
func (a *API) Identities(ctx context.Context) ([]Identity, error) {
	var res struct {
		Identities []struct {
			Identity Identity `json:"identity"`
		} `json:"identities"`
		TotalCount *int `json:"totalCount"`
	}
	if e := a.call(ctx, false, "GET", "/api/v1/identities", url.Values{"orgId": {a.OrganizationID}}, nil, &res); e != nil {
		return nil, e
	}
	if res.TotalCount == nil || len(res.Identities) != *res.TotalCount || len(res.Identities) > 1000 {
		return nil, ErrUnavailable
	}
	out := []Identity{}
	seen := map[string]bool{}
	for _, i := range res.Identities {
		if !identifier.MatchString(i.Identity.ID) || seen[i.Identity.ID] {
			return nil, ErrUnavailable
		}
		seen[i.Identity.ID] = true
		out = append(out, i.Identity)
	}
	return out, nil
}
func (a *API) CreateIdentity(ctx context.Context, name string) (Identity, error) {
	var res struct {
		Identity Identity `json:"identity"`
	}
	e := a.call(ctx, false, "POST", "/api/v1/identities", nil, map[string]any{"name": name, "organizationId": a.OrganizationID, "role": "no-access", "hasDeleteProtection": true}, &res)
	if e == nil && !identifier.MatchString(res.Identity.ID) {
		e = ErrUnavailable
	}
	return res.Identity, e
}
func (a *API) Memberships(ctx context.Context, project string, human bool) ([]Membership, error) {
	if !identifier.MatchString(project) {
		return nil, ErrDenied
	}
	out := []Membership{}
	seen := map[string]bool{}
	for offset := 0; offset < 2000; {
		var res struct {
			Items []Membership `json:"identityMemberships"`
			Total *int         `json:"totalCount"`
		}
		e := a.call(ctx, human, "GET", "/api/v1/projects/"+project+"/identity-memberships", url.Values{"offset": {strconv.Itoa(offset)}, "limit": {"100"}}, nil, &res)
		if e != nil {
			return nil, e
		}
		if res.Total == nil || *res.Total > 2000 || *res.Total < offset+len(res.Items) || len(res.Items) > 100 {
			return nil, ErrUnavailable
		}
		for _, m := range res.Items {
			if !identifier.MatchString(m.IdentityID) || seen[m.IdentityID] {
				return nil, ErrUnavailable
			}
			seen[m.IdentityID] = true
			out = append(out, m)
		}
		offset += len(res.Items)
		if offset >= *res.Total {
			return out, nil
		}
		if len(res.Items) == 0 {
			return nil, ErrUnavailable
		}
	}
	return nil, ErrUnavailable
}
func (a *API) EnrollHuman(ctx context.Context, project string) error {
	if !identifier.MatchString(project) {
		return ErrDenied
	}
	return a.call(ctx, true, "POST", "/api/v1/organization-admin/projects/"+project+"/grant-admin-access", nil, nil, nil)
}
func (a *API) SetMembership(ctx context.Context, project, id, role string, exists, human bool) error {
	if !identifier.MatchString(project) || !identifier.MatchString(id) || (role != "admin" && role != "viewer") {
		return ErrDenied
	}
	method := "POST"
	if exists {
		method = "PATCH"
	}
	return a.call(ctx, human, method, "/api/v1/projects/"+project+"/identity-memberships/"+id, nil, map[string]any{"roles": []map[string]any{{"role": role, "isTemporary": false}}}, nil)
}

// ReadOnlyIdentity refuses extra project privileges and organization privileges.
// Unsupported verification endpoints fail closed instead of claiming read-only.
func (a *API) ReadOnlyIdentity(ctx context.Context, p Project, id string) error {
	var org struct {
		Identity struct {
			Role  string `json:"role"`
			OrgID string `json:"orgId"`
		} `json:"identity"`
	}
	if e := a.call(ctx, false, "GET", "/api/v1/identities/"+id, nil, nil, &org); e != nil {
		return e
	}
	if org.Identity.OrgID != a.OrganizationID || org.Identity.Role != "no-access" {
		return ErrDenied
	}
	var extra struct {
		Privileges *[]json.RawMessage `json:"privileges"`
	}
	if e := a.call(ctx, false, "GET", "/api/v1/additional-privilege/identity", url.Values{"identityId": {id}, "projectSlug": {p.Slug}}, nil, &extra); e != nil {
		return e
	}
	if extra.Privileges == nil || len(*extra.Privileges) > 0 {
		return ErrDenied
	}
	return nil
}
