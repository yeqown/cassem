package adm

import (
	"encoding/json"
	"fmt"
	"github.com/gin-gonic/gin"
	"github.com/gin-gonic/gin/binding"
	"github.com/go-playground/validator/v10"
	"github.com/yeqown/cassem/api/concept"
	"regexp"
	"strings"
)

func bindURIParams(c *gin.Context, obj any) error {
	uriParams := make(map[string][]string, len(c.Params))
	for _, param := range c.Params {
		uriParams[param.Key] = []string{param.Value}
	}

	return binding.MapFormWithTag(obj, uriParams, "uri")
}

type contentTypeParam concept.ContentType

func (p *contentTypeParam) UnmarshalJSON(data []byte) error {
	var numeric int32
	if err := json.Unmarshal(data, &numeric); err == nil {
		*p = contentTypeParam(numeric)
		return nil
	}

	var name string
	if err := json.Unmarshal(data, &name); err != nil {
		return err
	}
	value, ok := concept.ContentType_value[strings.ToUpper(name)]
	if !ok {
		return fmt.Errorf("unknown contentType %q", name)
	}

	*p = contentTypeParam(value)
	return nil
}

func (p contentTypeParam) concept() concept.ContentType {
	return concept.ContentType(p)
}

var identifierPattern = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9_-]*$`)

func isIdentifier(value string) bool {
	return identifierPattern.MatchString(value)
}

func registerIdentifierValidator(validate *validator.Validate) error {
	return validate.RegisterValidation("identifier", func(fl validator.FieldLevel) bool {
		return isIdentifier(fl.Field().String())
	})
}

func init() {
	if validate, ok := binding.Validator.Engine().(*validator.Validate); ok {
		if err := registerIdentifierValidator(validate); err != nil {
			panic(err)
		}
	}
}
