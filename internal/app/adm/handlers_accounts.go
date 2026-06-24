package adm

import (
	"net/http"
	"slices"
	"time"

	"github.com/yeqown/cassem/api/concept"
	errorx "github.com/yeqown/cassem/api/concept"
	"github.com/yeqown/cassem/pkg/hash"
	"github.com/yeqown/cassem/pkg/httpx"
)

func normalizeAccountRole(role string) string {
	if role == "developer" {
		return concept.Role_DEVELOPER
	}
	return role
}

func (d app) UserLoginHTTP(w http.ResponseWriter, r *http.Request) {
	req := new(userLoginReq)
	if err := bindRequest(r, req); err != nil {
		httpx.WriteError(w, err)
		return
	}

	u, err := d.aggregate.GetUser(req.Account)
	if err != nil {
		if ex, ok := errorx.FromError(err); ok && ex.Code == errorx.Code_NOT_FOUND {
			httpx.WriteError(w, errorx.New(errorx.Code_NOT_FOUND, "login failed"))
			return
		}
		httpx.WriteError(w, err)
		return
	}
	if u.GetStatus() != concept.User_NORMAL {
		httpx.WriteError(w, errorx.Err_PERMISSION_DENIED)
		return
	}
	if hash.WithSalt(req.Password, u.Salt) != u.GetHashedPassword() {
		httpx.WriteError(w, errorx.New(errorx.Code_NOT_FOUND, "login failed"))
		return
	}
	roles, err := d.aggregate.GetUserRoles(req.Account)
	if err != nil {
		httpx.WriteError(w, err)
		return
	}
	sess, err := EncodeSession(&Session{
		Account:   u.GetAccount(),
		Salt:      u.GetSalt(),
		ExpiredAt: time.Now().AddDate(0, 0, 1).Unix(),
	}, d.conf.Auth.SessionSecret)
	if err != nil {
		httpx.WriteError(w, err)
		return
	}

	httpx.WriteJSON(w, userLoginResp{User: userLoginUser{
		Account:  u.GetAccount(),
		Nickname: u.GetNickname(),
		Status:   int32(u.GetStatus()),
		Roles:    roles,
	}, Session: sess})
}

func (d app) GetUsersHTTP(w http.ResponseWriter, r *http.Request) {
	req := new(getUsersReq)
	if err := bindRequest(r, req); err != nil {
		httpx.WriteError(w, err)
		return
	}
	out, err := d.aggregate.GetUsers(req.Seek, req.Limit)
	if err != nil {
		httpx.WriteError(w, err)
		return
	}
	users := make([]accountUserView, 0, len(out.Users))
	for _, item := range out.Users {
		bindings, err := d.aggregate.GetUserRoleBindings(item.GetAccount())
		if err != nil {
			httpx.WriteError(w, err)
			return
		}
		roles := make([]string, 0, len(bindings))
		summary := make([]accountUserBinding, 0, len(bindings))
		for _, binding := range bindings {
			if !slices.Contains(roles, binding.Role) {
				roles = append(roles, binding.Role)
			}
			summary = append(summary, accountUserBinding{Role: binding.Role, Domain: binding.Domain})
		}
		users = append(users, accountUserView{
			Account:       item.GetAccount(),
			Nickname:      item.GetNickname(),
			Status:        int32(item.GetStatus()),
			Roles:         roles,
			BindingCount:  len(summary),
			AccessSummary: summary,
		})
	}
	httpx.WriteJSON(w, getUsersResp{Users: users})
}

func (d app) GetUserACLHTTP(w http.ResponseWriter, r *http.Request) {
	req := new(getUserAclReq)
	if err := bindRequest(r, req); err != nil {
		httpx.WriteError(w, err)
		return
	}
	bindings, err := d.aggregate.GetUserRoleBindings(req.Account)
	if err != nil {
		httpx.WriteError(w, err)
		return
	}
	out := make([]accountUserBinding, 0, len(bindings))
	for _, binding := range bindings {
		out = append(out, accountUserBinding{Role: binding.Role, Domain: binding.Domain})
	}
	httpx.WriteJSON(w, getUserAclResp{Bindings: out})
}

func (d app) GetACLDomainOptionsHTTP(w http.ResponseWriter, r *http.Request) {
	domains, err := d.aggregate.ListDomainOptions()
	if err != nil {
		httpx.WriteError(w, err)
		return
	}
	httpx.WriteJSON(w, getAclDomainsResp{Domains: domains})
}

func (d app) AddUserHTTP(w http.ResponseWriter, r *http.Request) {
	req := new(addUserReq)
	if err := bindRequest(r, req); err != nil {
		httpx.WriteError(w, err)
		return
	}
	u := &concept.User{Account: req.Account, Nickname: req.Nickname, HashedPassword: req.Password, Status: concept.User_NORMAL}
	if err := d.aggregate.AddUser(u); err != nil {
		httpx.WriteError(w, err)
		return
	}
	httpx.WriteJSON(w, nil)
}

func (d app) DisableUserHTTP(w http.ResponseWriter, r *http.Request) {
	req := new(disableUserReq)
	if err := bindRequest(r, req); err != nil {
		httpx.WriteError(w, err)
		return
	}
	if err := d.aggregate.DisableUser(req.Account); err != nil {
		httpx.WriteError(w, err)
		return
	}
	httpx.WriteJSON(w, nil)
}

func (d app) ResetUserHTTP(w http.ResponseWriter, r *http.Request) {
	req := new(resetUserReq)
	if err := bindRequest(r, req); err != nil {
		httpx.WriteError(w, err)
		return
	}
	if err := d.aggregate.ResetUser(req.Account, req.Password); err != nil {
		httpx.WriteError(w, err)
		return
	}
	httpx.WriteJSON(w, nil)
}

func (d app) AssignRoleHTTP(w http.ResponseWriter, r *http.Request) {
	req := new(assignOrRevokeRoleReq)
	if err := bindRequest(r, req); err != nil {
		httpx.WriteError(w, err)
		return
	}
	if len(req.Domains) == 0 {
		req.Domains = []string{concept.Domain_CLUSTER}
	}
	if err := d.aggregate.AssignRole(req.Account, normalizeAccountRole(req.Role), req.Domains...); err != nil {
		httpx.WriteError(w, err)
		return
	}
	httpx.WriteJSON(w, nil)
}

func (d app) RevokeRoleHTTP(w http.ResponseWriter, r *http.Request) {
	req := new(assignOrRevokeRoleReq)
	if err := bindRequest(r, req); err != nil {
		httpx.WriteError(w, err)
		return
	}
	if len(req.Domains) == 0 {
		req.Domains = []string{concept.Domain_CLUSTER}
	}
	if err := d.aggregate.RevokeRole(req.Account, normalizeAccountRole(req.Role), req.Domains...); err != nil {
		httpx.WriteError(w, err)
		return
	}
	httpx.WriteJSON(w, nil)
}
