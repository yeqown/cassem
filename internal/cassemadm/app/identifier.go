package app

import (
	"regexp"

	"github.com/gin-gonic/gin/binding"
	"github.com/go-playground/validator/v10"
)

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
