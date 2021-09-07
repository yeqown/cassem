package app

import (
	"strings"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/yeqown/cassem/concept"
	"github.com/yeqown/cassem/internal/cassemadm/infras"
	"github.com/yeqown/cassem/pkg/errorx"
	"github.com/yeqown/cassem/pkg/hash"
	"github.com/yeqown/cassem/pkg/httpx"
)

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

	// compare password with salt
	if strings.Compare(hash.WithSalt(req.Password, u.Salt), u.GetHashedPassword()) != 0 {
		httpx.ResponseError(c, errorx.New(errorx.Code_NOT_FOUND, "user not found"))
		return
	}

	sess, err := infras.EncodeSession(&infras.Session{
		Account:   u.GetAccount(),
		Salt:      u.GetSalt(),
		ExpiredAt: time.Now().AddDate(0, 0, 1).Unix(), // after 1 day to expire
	})

	httpx.ResponseJSON(c, userLoginResp{User: u, Session: sess})
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
	// TODO(@yeqown):
	panic("implement me")
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

	err := d.aggregate.AssignRole(req.Account, req.Role, req.Domains...)
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

	err := d.aggregate.RevokeRole(req.Account, req.Role, req.Domains...)
	if err != nil {
		httpx.ResponseError(c, err)
		return
	}

	httpx.ResponseJSON(c, nil)
}
