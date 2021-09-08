package app

import (
	"github.com/yeqown/cassem/concept"
)

type commonAppEnvRequest struct {
	AppId string `uri:"appId" binding:"required"`
	Env   string `uri:"env" binding:"required"`
}

type commonAppEnvEltRequest struct {
	AppId      string `uri:"appId" form:"app" binding:"required"`
	Env        string `uri:"env" form:"env" binding:"required"`
	ElementKey string `uri:"key" form:"key" binding:"required"`
}

type commonPagingRequest struct {
	Limit int    `form:"limit,default=100"`
	Seek  string `form:"seek"`
}

type getAppEnvElementsReq struct {
	commonAppEnvRequest
	commonPagingRequest

	ElementKeys []string `form:"key"`
}

type createAppEnvElementReq struct {
	commonAppEnvEltRequest

	Raw         string              `json:"raw" binding:"required"`
	ContentType concept.ContentType `json:"contentType" binding:"required,oneof=1 2 3 4"`
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

type deleteAppEnvElementsReq struct {
	commonAppEnvEltRequest
}

type diffAppEnvElementsReq struct {
	commonAppEnvEltRequest

	Base    uint `form:"base"`
	Compare uint `form:"compare"`
}

type diffAppEnvElementsResp struct {
	Base    *concept.Element `json:"base"`
	Compare *concept.Element `json:"compare"`
	Diff    string           `json:"diff"`
}

type pagingAppsReq struct {
	commonPagingRequest
}

type createAppReq struct {
	App         string `uri:"appId" binding:"required"`
	Name        string `json:"name" binding:"required"`
	Description string `json:"description" binding:"required"`
}

type deleteAppReq struct {
	App string `uri:"appId" binding:"required"`
}

type getAppReq struct {
	App string `uri:"appId" binding:"required"`
}

type getAppEnvsReq struct {
	commonPagingRequest

	App string `uri:"appId" binding:"required"`
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

type getEleInstancesReq struct {
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

type disableUserReq struct {
	Account string `form:"account" binding:"email,required"`
}

type userLoginReq struct {
	Account  string `json:"account" binding:"email,required"`
	Password string `json:"password" binding:"required"`
}

type userLoginResp struct {
	User    *concept.User `json:"user"`
	Session string        `json:"session"`
}

type assignOrRevokeRoleReq struct {
	Account string   `form:"account" binding:"required,email"`
	Role    string   `form:"role" binding:"required,oneof=superadmin admin appowner developer"`
	Domains []string `form:"domain"`
}
