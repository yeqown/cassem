package infras

import (
	"github.com/gin-gonic/gin"
	"github.com/pkg/errors"
	"github.com/yeqown/log"

	"github.com/yeqown/cassem/concept"
	"github.com/yeqown/cassem/pkg/errorx"
	"github.com/yeqown/cassem/pkg/httpx"
)

type req struct {
	Domain string `uri:"env"`
}

func Authentication(rbac concept.RBAC) gin.HandlerFunc {
	return func(c *gin.Context) {
		sess, ok := GetSessionFromContext(c)
		if !ok {
			log.Debug("Authentication session not found")
			httpx.ResponseErrorAndAbort(c, errors.Wrap(errorx.Err_PERMISSION_DENIED, "session not found"))
			return
		}

		fp := c.FullPath()
		def, ok := defMapping[c.Request.Method+fp]
		if !ok {
			log.
				WithFields(log.Fields{
					"fullPath": fp,
					"method":   c.Request.Method,
				}).
				Debug("Authentication objectDef not found")
			c.Next()
			return
		}

		// parse domain(env) from uri
		r := new(req)
		_ = c.ShouldBindUri(r)
		if r.Domain == "" {
			r.Domain = concept.Domain_CLUSTER
		}
		allow, err := rbac.Enforce(sess.Account, r.Domain, def.object, def.act)
		if err != nil {
			httpx.ResponseErrorAndAbort(c, err)
			return
		}

		if !allow {
			httpx.ResponseErrorAndAbort(c, errors.Wrap(errorx.Err_PERMISSION_DENIED, "not allowed"))
			return
		}

		c.Next()
	}
}

type objectDef struct {
	object string
	act    string
}

var defMapping = map[string]objectDef{
	// app and it's sub-objects
	"GET/api/apps":           {object: concept.Object_APP, act: concept.Action_READ},
	"GET/api/apps/:appId":    {object: concept.Object_APP, act: concept.Action_READ},
	"POST/api/apps/:appId":   {object: concept.Object_APP, act: concept.Action_WRITE},
	"DELETE/api/apps/:appId": {object: concept.Object_APP, act: concept.Action_DELETE},

	"GET/api/apps/:appId/envs":                       {object: concept.Object_APP, act: concept.Action_READ},
	"POST/api/apps/:appId/envs/:env":                 {object: concept.Object_APP, act: concept.Action_WRITE},
	"DELETE/api/apps/:appId/envs/:env":               {object: concept.Object_APP, act: concept.Action_WRITE},
	"GET/api/apps/:appId/envs/:env/elements":         {object: concept.Object_APP, act: concept.Action_READ},
	"GET/api/apps/:appId/envs/:env/elements/:key":    {object: concept.Object_ELEMENT, act: concept.Action_READ},
	"POST/api/apps/:appId/envs/:env/elements/:key":   {object: concept.Object_ELEMENT, act: concept.Action_WRITE},
	"PUT/api/apps/:appId/envs/:env/elements/:key":    {object: concept.Object_ELEMENT, act: concept.Action_WRITE},
	"DELETE/api/apps/:appId/envs/:env/elements/:key": {object: concept.Object_ELEMENT, act: concept.Action_DELETE},

	"GET/api/apps/:appId/envs/:env/elements/:key/versions": {object: concept.Object_ELEMENT, act: concept.Action_READ},
	"GET/api/apps/:appId/envs/:env/elements/:key/diff":     {object: concept.Object_ELEMENT, act: concept.Action_READ},
	"GET/api/apps/:appId/envs/:env/elements/:key/rollback": {object: concept.Object_ELEMENT, act: concept.Action_PUBLISH},
	"GET/api/apps/:appId/envs/:env/elements/:key/publish":  {object: concept.Object_ELEMENT, act: concept.Action_PUBLISH},

	// acl
	"POST/api/account/add":       {object: concept.Object_USER, act: concept.Action_WRITE},
	"GET/api/account/disable":    {object: concept.Object_USER, act: concept.Action_WRITE},
	"GET/api/account/reset":      {object: concept.Object_USER, act: concept.Action_WRITE},
	"GET/api/account/acl/assign": {object: concept.Object_ACL, act: concept.Action_WRITE},
	"GET/api/account/acl/revoke": {object: concept.Object_ACL, act: concept.Action_WRITE},

	// cluster
	"GET/api/agents":           {object: concept.Object_CLUSTER, act: concept.Action_READ},
	"GET/api/instances":        {object: concept.Object_CLUSTER, act: concept.Action_READ},
	"GET/api/instances/:insId": {object: concept.Object_CLUSTER, act: concept.Action_READ},
}
