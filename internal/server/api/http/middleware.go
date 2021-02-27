package http

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

func authorize() gin.HandlerFunc {

	return func(c *gin.Context) {
		c.Next()
	}
}

func clusterAuthorizeSimple() gin.HandlerFunc {
	return func(c *gin.Context) {
		if c.Query("clusterSecret") == "9520059dd167" {
			c.Next()
			return
		}

		// then forbidden the invalid request
		c.AbortWithStatus(http.StatusForbidden)
	}
}
