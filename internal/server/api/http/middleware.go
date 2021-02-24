package http

import "github.com/gin-gonic/gin"

func authorize() gin.HandlerFunc {

	return func(c *gin.Context) {
		c.Next()
	}
}
