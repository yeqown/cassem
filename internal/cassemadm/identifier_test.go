package cassemadm

import (
	"testing"

	"github.com/go-playground/validator/v10"
	"github.com/stretchr/testify/require"
)

func TestIdentifierValidator(t *testing.T) {
	validate := validator.New()
	require.NoError(t, registerIdentifierValidator(validate))

	type req struct {
		Value string `validate:"identifier"`
	}

	for _, value := range []string{"app1", "APP", "app-prod", "app_prod", "a1_B-2"} {
		require.NoError(t, validate.Struct(req{Value: value}), value)
	}

	for _, value := range []string{"", "-app", "_app", "app.prod", "app prod", "app,prod", "app/prod"} {
		require.Error(t, validate.Struct(req{Value: value}), value)
	}
}
