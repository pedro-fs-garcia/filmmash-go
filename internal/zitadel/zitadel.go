package zitadel

import "encoding/json"

const ZitadelRolesClaim = "urn:zitadel:iam:org:project:roles"

type ZitadelRoles map[string]map[string]string

// RoleMapper extracts role keys from the Zitadel project-roles claim of a
// raw OIDC claim set. It satisfies auth.ProviderCfg.RoleMapper.
func RoleMapper(claims json.RawMessage) []string {
	var c struct {
		Roles ZitadelRoles `json:"urn:zitadel:iam:org:project:roles"`
	}
	if err := json.Unmarshal(claims, &c); err != nil {
		return nil
	}
	roles := make([]string, 0, len(c.Roles))
	for role := range c.Roles {
		roles = append(roles, role)
	}
	return roles
}

type ListAuthzResp struct {
	Authorizations []struct {
		ID    string `json:"id"`
		State string `json:"state"`
		User  struct {
			ID                 string `json:"id"`
			PreferredLoginName string `json:"preferredLoginName"`
			DisplayName        string `json:"displayName"`
		} `json:"user"`
		Roles []struct {
			Key         string `json:"key"`
			DisplayName string `json:"displayName"`
			Group       string `json:"group"`
		} `json:"roles"`
	} `json:"authorizations"`
}
