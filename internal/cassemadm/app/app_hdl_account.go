package app

import (
	"slices"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/yeqown/cassem/api/concept"
	"github.com/yeqown/cassem/internal/cassemadm/infras"
	"github.com/yeqown/cassem/pkg/errorx"
	"github.com/yeqown/cassem/pkg/hash"
	"github.com/yeqown/cassem/pkg/httpx"
)

func normalizeAccountRole(role string) string {
	if role == "developer" {
		return concept.Role_DEVELOPER
	}
	return role
}

func (d app) UserLogin(c *gin.Context) {
	req := new(userLoginReq)
	if err := c.ShouldBind(req); err != nil {
		httpx.ResponseError(c, err)
		return
	}

	u, err := d.aggregate.GetUser(req.Account)
	if err != nil {
		httpx.ResponseError(c, err)
		return
	}

	if u.GetStatus() != concept.User_NORMAL {
		httpx.ResponseError(c, errorx.Err_PERMISSION_DENIED)
		return
	}

	if hash.WithSalt(req.Password, u.Salt) != u.GetHashedPassword() {
		httpx.ResponseError(c, errorx.New(errorx.Code_NOT_FOUND, "login failed"))
		return
	}

	roles, err := d.aggregate.GetUserRoles(req.Account)
	if err != nil {
		httpx.ResponseError(c, err)
		return
	}

	sess, err := infras.EncodeSession(&infras.Session{
		Account:   u.GetAccount(),
		Salt:      u.GetSalt(),
		ExpiredAt: time.Now().AddDate(0, 0, 1).Unix(),
	}, d.conf.Auth.SessionSecret)
	if err != nil {
		httpx.ResponseError(c, err)
		return
	}

	httpx.ResponseJSON(c, userLoginResp{User: userLoginUser{
		Account:  u.GetAccount(),
		Nickname: u.GetNickname(),
		Status:   int32(u.GetStatus()),
		Roles:    roles,
	}, Session: sess})
}

func (d app) GetUsers(c *gin.Context) {
	req := new(getUsersReq)
	if err := c.ShouldBind(req); err != nil {
		httpx.ResponseError(c, err)
		return
	}

	out, err := d.aggregate.GetUsers(req.Seek, req.Limit)
	if err != nil {
		httpx.ResponseError(c, err)
		return
	}

	users := make([]accountUserView, 0, len(out.Users))
	for _, item := range out.Users {
		bindings, err := d.aggregate.GetUserRoleBindings(item.GetAccount())
		if err != nil {
			httpx.ResponseError(c, err)
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

	httpx.ResponseJSON(c, getUsersResp{Users: users})
}

func (d app) GetUserACL(c *gin.Context) {
	req := new(getUserAclReq)
	_ = c.ShouldBindUri(req)
	if err := c.ShouldBind(req); err != nil {
		httpx.ResponseError(c, err)
		return
	}

	bindings, err := d.aggregate.GetUserRoleBindings(req.Account)
	if err != nil {
		httpx.ResponseError(c, err)
		return
	}

	out := make([]accountUserBinding, 0, len(bindings))
	for _, binding := range bindings {
		out = append(out, accountUserBinding{Role: binding.Role, Domain: binding.Domain})
	}

	httpx.ResponseJSON(c, getUserAclResp{Bindings: out})
}

func (d app) GetACLDomainOptions(c *gin.Context) {
	domains, err := d.aggregate.ListDomainOptions()
	if err != nil {
		httpx.ResponseError(c, err)
		return
	}

	httpx.ResponseJSON(c, getAclDomainsResp{Domains: domains})
}

func (d app) AddUser(c *gin.Context) {
	req := new(addUserReq)
	if err := c.ShouldBind(req); err != nil {
		httpx.ResponseError(c, err)
		return
	}

	u := &concept.User{
		Account:        req.Account,
		Nickname:       req.Nickname,
		HashedPassword: req.Password,
		Status:         concept.User_NORMAL,
	}
	err := d.aggregate.AddUser(u)
	if err != nil {
		httpx.ResponseError(c, err)
		return
	}

	httpx.ResponseJSON(c, nil)
}

func (d app) DisableUser(c *gin.Context) {
	req := new(disableUserReq)
	if err := c.ShouldBind(req); err != nil {
		httpx.ResponseError(c, err)
		return
	}

	err := d.aggregate.DisableUser(req.Account)
	if err != nil {
		httpx.ResponseError(c, err)
		return
	}

	httpx.ResponseJSON(c, nil)
}

func (d app) ResetUser(c *gin.Context) {
	req := new(resetUserReq)
	if err := c.ShouldBind(req); err != nil {
		httpx.ResponseError(c, err)
		return
	}

	if err := d.aggregate.ResetUser(req.Account, req.Password); err != nil {
		httpx.ResponseError(c, err)
		return
	}

	httpx.ResponseJSON(c, nil)
}

func (d app) AssignRole(c *gin.Context) {
	req := new(assignOrRevokeRoleReq)
	if err := c.ShouldBind(req); err != nil {
		httpx.ResponseError(c, err)
		return
	}

	if len(req.Domains) == 0 {
		req.Domains = []string{concept.Domain_CLUSTER}
	}

	err := d.aggregate.AssignRole(req.Account, normalizeAccountRole(req.Role), req.Domains...)
	if err != nil {
		httpx.ResponseError(c, err)
		return
	}

	httpx.ResponseJSON(c, nil)
}

func (d app) RevokeRole(c *gin.Context) {
	req := new(assignOrRevokeRoleReq)
	if err := c.ShouldBind(req); err != nil {
		httpx.ResponseError(c, err)
		return
	}

	if len(req.Domains) == 0 {
		req.Domains = []string{concept.Domain_CLUSTER}
	}

	err := d.aggregate.RevokeRole(req.Account, normalizeAccountRole(req.Role), req.Domains...)
	if err != nil {
		httpx.ResponseError(c, err)
		return
	}

	httpx.ResponseJSON(c, nil)
}
