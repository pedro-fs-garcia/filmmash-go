package zitadel

const ZitadelRolesClaim = "urn:zitadel:iam:org:project:roles"

type ZitadelRoles map[string]map[string]string

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
