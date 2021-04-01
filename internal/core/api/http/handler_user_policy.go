package http

import (
	"strconv"

	"github.com/yeqown/cassem/internal/authorizer"
	"github.com/yeqown/cassem/internal/persistence"

	"github.com/gin-gonic/gin"
	"github.com/pkg/errors"
)

type loginReq struct {
	Account  string `json:"account" binding:"required"`
	Password string `json:"password" binding:"required"`
}

type loginResp struct {
	Token string `json:"token"`
	User  userVO `json:"user"`
}

func (srv Server) Login(c *gin.Context) {
	req := new(loginReq)

	if err := c.ShouldBind(req); err != nil {
		responseError(c, err)
		return
	}

	u, token, err := srv.coordinator.Login(req.Account, req.Password)
	if err != nil {
		responseError(c, err)
		return
	}
	r := new(loginResp)
	r.Token = token
	r.User = toUserVO(u)

	responseJSON(c, r)
}

type createUserReq struct {
	Account  string `json:"account" binding:"required"`
	Password string `json:"password" binding:"required"`
	Name     string `json:"name"`
}

func (srv Server) CreateUser(c *gin.Context) {
	req := new(createUserReq)

	if err := c.ShouldBind(req); err != nil {
		responseError(c, err)
		return
	}

	_, err := srv.coordinator.AddUser(req.Account, req.Password, req.Name)
	if err != nil {
		responseError(c, err)
		return
	}

	responseJSON(c, nil)
}

type pagingUsersRequest struct {
	Limit          int    `form:"limit,default=100"`
	Offset         int    `form:"offset,default=0"`
	AccountPattern string `form:"account"`
}

type userVO struct {
	UserId    uint   `json:"userId"`
	Account   string `json:"account"`
	Name      string `json:"name"`
	CreatedAt int64  `json:"createdAt"`
}

func toUserVO(u *persistence.User) userVO {
	return userVO{
		UserId:    u.ID,
		Account:   u.Account,
		Name:      u.Name,
		CreatedAt: u.CreatedAt.Unix(),
	}
}

type pagingUsersResp struct {
	Users []userVO `json:"users"`
	Total int      `json:"total"`
}

func (srv Server) PagingUsers(c *gin.Context) {
	req := new(pagingUsersRequest)

	if err := c.ShouldBind(req); err != nil {
		responseError(c, err)
		return
	}

	out, count, err := srv.coordinator.PagingUsers(req.Limit, req.Offset, req.AccountPattern)
	if err != nil {
		responseError(c, err)
		return
	}

	r := new(pagingUsersResp)
	r.Total = count
	r.Users = make([]userVO, 0, len(out))
	for _, u := range out {
		r.Users = append(r.Users, toUserVO(u))
	}

	responseJSON(c, r)
}

type resetPasswordReq struct {
	Account  string `json:"account" binding:"required"`
	Password string `json:"password" binding:"required"`
}

func (srv Server) ResetPassword(c *gin.Context) {
	req := new(resetPasswordReq)

	if err := c.ShouldBind(req); err != nil {
		responseError(c, err)
		return
	}

	err := srv.coordinator.ResetPassword(req.Account, req.Password)
	if err != nil {
		responseError(c, err)
		return
	}

	responseJSON(c, nil)
}

type policyVO struct {
	Object  string `json:"object" binding:"required,oneof=namespace container pair user policy"`
	Action  string `json:"action" binding:"required,oneof=read write any"`
	Subject string `json:"subject"`
}

func toPolicyVO(p authorizer.Policy) policyVO {
	return policyVO{
		Subject: p.Subject,
		Object:  p.Object,
		Action:  p.Action,
	}
}

func (srv Server) GetUserPolicies(c *gin.Context) {
	s := c.Params.ByName("userid")
	uid, err := strconv.Atoi(s)
	if err != nil {
		responseError(c, errors.New("invalid token: not set"))
		return
	}
	token := authorizer.NewToken(uid)

	//v, ok := c.Get(_authorizationKey)
	//if !ok {
	//	responseError(c, errors.New("invalid token: not set"))
	//	return
	//}
	//
	//token, ok := v.(*authorizer.Token)
	//if !ok || token == nil {
	//	responseError(c, errors.New("invalid token: empty or invalid type"))
	//	return
	//}

	out := srv.coordinator.ListSubjectPolicies(token.Subject())
	policies := make([]policyVO, 0, len(out))
	for _, p := range out {
		policies = append(policies, toPolicyVO(p))
	}

	responseJSON(c, policies)
}

type updateUserPoliciesReq struct {
	Metas []policyVO `json:"metas" binding:"required"`
}

func (req updateUserPoliciesReq) policies(subject string) []authorizer.Policy {
	out := make([]authorizer.Policy, 0, len(req.Metas))
	for _, v := range req.Metas {
		out = append(out, authorizer.Policy{
			Subject: subject,
			Object:  v.Object,
			Action:  v.Action,
		})
	}

	return out
}

func (srv Server) UpdateUserPolicies(c *gin.Context) {
	s := c.Params.ByName("userid")
	uid, err := strconv.Atoi(s)
	if err != nil {
		responseError(c, errors.Wrap(err, "invalid uid"))
		return
	}
	token := authorizer.NewToken(uid)

	req := new(updateUserPoliciesReq)
	if err = c.ShouldBind(req); err != nil {
		responseError(c, err)
		return
	}

	subject := token.Subject()
	if err = srv.coordinator.UpdateSubjectPolicies(subject, req.policies(subject)); err != nil {
		responseError(c, err)
		return
	}

	responseJSON(c, nil)
}
