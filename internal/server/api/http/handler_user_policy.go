package http

import (
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/pkg/errors"
	"github.com/yeqown/cassem/internal/authorizer"
)

type loginReq struct {
	Account  string `json:"account" binding:"required"`
	Password string `json:"password" binding:"required"`
}

func (srv Server) Login(c *gin.Context) {
	req := new(loginReq)

	if err := c.ShouldBind(req); err != nil {
		responseError(c, err)
		return
	}

	token, err := srv.auth.Login(req.Account, req.Password)
	if err != nil {
		responseError(c, err)
		return
	}

	responseJSON(c, token)
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

	err := srv.auth.AddUser(req.Account, req.Password, req.Name)
	if err != nil {
		responseError(c, err)
		return
	}

	responseJSON(c, nil)
}

func (srv Server) PagingUsers(c *gin.Context) {
	// TODO(@yeqown): fill paging logic
	panic("implement me")
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

	err := srv.auth.ResetPassword(req.Account, req.Password)
	if err != nil {
		responseError(c, err)
		return
	}

	responseJSON(c, nil)
}

type policyVO struct {
	Object  string `json:"object" binding:"required,oneof=namespace container pair user"`
	Action  string `json:"action" binding:"required, oneof=read write any"`
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
	token := authorizer.Token{UserId: uid}

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

	out := srv.auth.ListSubjectPolicies(token.Subject())
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
	token := authorizer.Token{UserId: uid}

	req := new(updateUserPoliciesReq)
	if err = c.ShouldBind(req); err != nil {
		responseError(c, err)
		return
	}

	subject := token.Subject()
	if err = srv.auth.UpdateSubjectPolicies(subject, req.policies(subject)); err != nil {
		responseError(c, err)
		return
	}

	responseJSON(c, nil)
}
