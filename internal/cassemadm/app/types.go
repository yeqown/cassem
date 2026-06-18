package app

import (
	"github.com/yeqown/cassem/api/concept"
)

type commonAppEnvRequest struct {
	AppId string `uri:"appId" binding:"required,identifier"`
	Env   string `uri:"env" binding:"required,identifier"`
}

type commonAppEnvEltRequest struct {
	AppId      string `uri:"appId" form:"app" binding:"required,identifier"`
	Env        string `uri:"env" form:"env" binding:"required,identifier"`
	ElementKey string `uri:"key" form:"key" binding:"required,identifier"`
}

type commonPagingRequest struct {
	Limit int    `form:"limit,default=100"`
	Seek  string `form:"seek"`
}

type getAppEnvElementsReq struct {
	commonAppEnvRequest
	commonPagingRequest

	ElementKeys []string `form:"key" binding:"omitempty,dive,identifier"`
	Query       string   `form:"query"`
}

type createAppEnvElementReq struct {
	commonAppEnvEltRequest

	Raw         string           `json:"raw" binding:"required"`
	ContentType contentTypeParam `json:"contentType" binding:"required,oneof=1 2 3 4"`
}

type updateAppEnvElementReq struct {
	commonAppEnvEltRequest

	Raw string `json:"raw" binding:"required"`
}

type getAppEnvElementReq struct {
	commonAppEnvEltRequest

	Version uint `form:"version"`
}

type getAppEnvElementVersionsReq struct {
	commonAppEnvEltRequest
	commonPagingRequest

	Versions []uint `form:"version"`
}

type getAppEnvElementOperationsReq struct {
	commonAppEnvEltRequest
	commonPagingRequest
}

type deleteAppEnvElementsReq struct {
	commonAppEnvEltRequest
}

type pagingAppsReq struct {
	commonPagingRequest

	Query string `form:"query"`
}

type createAppUriReq struct {
	App string `uri:"appId" binding:"required,identifier"`
}

type createAppReq struct {
	Name        string `json:"name" binding:"required"`
	Description string `json:"description" binding:"required"`
}

type deleteAppReq struct {
	App string `uri:"appId" binding:"required,identifier"`
}

type getAppReq struct {
	App string `uri:"appId" binding:"required,identifier"`
}

type getAppEnvsReq struct {
	commonPagingRequest

	App string `uri:"appId" binding:"required,identifier"`
}

type createAppEnvReq struct {
	commonAppEnvRequest
}

type deleteAppEnvReq struct {
	commonAppEnvRequest
}

type getInstanceReq struct {
	InsId string `uri:"insId" binding:"required"`
}

type getInstancesReq struct {
	commonPagingRequest
}

type getInstancesByElementReq struct {
	commonAppEnvEltRequest
}

type rollbackAppEnvElementReq struct {
	commonAppEnvEltRequest

	RollbackTo uint32 `json:"version" form:"version" binding:"required"`
}

type publishAppEnvElementReq struct {
	commonAppEnvEltRequest

	Publish     uint32                 `json:"version" form:"version" binding:"required"`
	AgentIds    []string               `json:"agentId" form:"agentId"`
	InstanceIds []string               `json:"instanceId" form:"instanceId"`
	PublishMode concept.PublishingMode `json:"publishMode" form:"publishMode,default=2" binding:"required,oneof=1 2"`
}

type pagingAgentInstanceReq struct {
	commonPagingRequest
}

type addUserReq struct {
	Account  string `json:"account" binding:"email,required"`
	Password string `json:"password" binding:"required"`
	Nickname string `json:"nickname" binding:"required"`
}

type getUsersReq struct {
	commonPagingRequest
}

type accountUserBinding struct {
	Role   string `json:"role"`
	Domain string `json:"domain"`
}

type accountUserView struct {
	Account       string               `json:"account"`
	Nickname      string               `json:"nickname,omitempty"`
	Status        int32                `json:"status,omitempty"`
	Roles         []string             `json:"roles,omitempty"`
	BindingCount  int                  `json:"bindingCount,omitempty"`
	AccessSummary []accountUserBinding `json:"accessSummary,omitempty"`
}

type getUsersResp struct {
	Users []accountUserView `json:"users"`
}

type getUserAclReq struct {
	Account string `uri:"account" binding:"required,email"`
}

type getUserAclResp struct {
	Bindings []accountUserBinding `json:"bindings"`
}

type getAclDomainsResp struct {
	Domains []string `json:"domains"`
}

type disableUserReq struct {
	Account string `form:"account" binding:"email,required"`
}

type resetUserReq struct {
	Account  string `json:"account" form:"account" binding:"email,required"`
	Password string `json:"password" form:"password" binding:"required"`
}

type userLoginReq struct {
	Account  string `json:"account" binding:"email,required"`
	Password string `json:"password" binding:"required"`
}

type userLoginUser struct {
	Account  string   `json:"account"`
	Nickname string   `json:"nickname,omitempty"`
	Status   int32    `json:"status,omitempty"`
	Roles    []string `json:"roles,omitempty"`
}

type userLoginResp struct {
	User    userLoginUser `json:"user"`
	Session string        `json:"session"`
}

type assignOrRevokeRoleReq struct {
	Account string   `form:"account" binding:"required,email"`
	Role    string   `form:"role" binding:"required,oneof=superadmin admin appowner appdeveloper developer visitor"`
	Domains []string `form:"domain"`
}
