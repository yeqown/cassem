package adm

import (
	"github.com/gin-gonic/gin"
	"github.com/gin-gonic/gin/binding"
)

func bindURIParams(c *gin.Context, obj any) error {
	uriParams := make(map[string][]string, len(c.Params))
	for _, param := range c.Params {
		uriParams[param.Key] = []string{param.Value}
	}

	return binding.MapFormWithTag(obj, uriParams, "uri")
}
