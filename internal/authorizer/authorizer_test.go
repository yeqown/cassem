package authorizer_test

import (
	"testing"

	"github.com/yeqown/cassem/internal/authorizer"
	"github.com/yeqown/cassem/internal/conf"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var _conf = &conf.MySQL{
	DSN:         "root:@tcp(127.0.0.1:3306)/cassem?charset=utf8mb4&parseTime=true&loc=Local",
	MaxIdle:     10,
	MaxOpen:     100,
	MaxLifeTime: 30,
	Debug:       true,
}

func Test_IAuthorizer_Enforce(t *testing.T) {
	a, err := authorizer.New(_conf)
	require.Nil(t, err)

	// I have prepared data into DB
	allowed := a.Enforce(&authorizer.EnforceRequest{Subject: "alice", Object: "data1", Action: "w"})
	assert.Equal(t, true, allowed)

	allowed = a.Enforce(&authorizer.EnforceRequest{Subject: "alice", Object: "container", Action: "r"})
	assert.Equal(t, true, allowed)
}

func Test_IAuthorizer_AddPolicy(t *testing.T) {
	a, err := authorizer.New(_conf)
	require.Nil(t, err)

	err = a.UpdateSubjectPolicies("uid:1", []authorizer.Policy{
		{
			Subject: "uid:1",
			Object:  authorizer.OBJ_ANY,
			Action:  authorizer.ACTION_ANY,
		},
	})
	require.Nil(t, err)

	allow := a.Enforce(&authorizer.EnforceRequest{
		Subject: "root",
		Object:  authorizer.OBJ_CONTAINER,
		Action:  authorizer.ACTION_READ,
	})
	assert.Equal(t, true, allow)

	allow = a.Enforce(&authorizer.EnforceRequest{
		Subject: "root",
		Object:  authorizer.OBJ_NAMESPACE,
		Action:  authorizer.ACTION_WRITE,
	})
	assert.Equal(t, true, allow)

	allow = a.Enforce(&authorizer.EnforceRequest{
		Subject: "root",
		Object:  authorizer.OBJ_ANY,
		Action:  authorizer.ACTION_ANY,
	})
	assert.Equal(t, true, allow)

	allow = a.Enforce(&authorizer.EnforceRequest{
		Subject: "root",
		Object:  authorizer.OBJ_ANY,
		Action:  authorizer.ACTION_READ,
	})
	assert.Equal(t, true, allow)

	allow = a.Enforce(&authorizer.EnforceRequest{
		Subject: "root",
		Object:  authorizer.OBJ_PAIR,
		Action:  authorizer.ACTION_ANY,
	})
	assert.Equal(t, true, allow)

	allow = a.Enforce(&authorizer.EnforceRequest{
		Subject: "alice",
		Object:  authorizer.OBJ_PAIR,
		Action:  authorizer.ACTION_ANY,
	})
	assert.Equal(t, false, allow)
}

func Test_IAuthorizer_LoginAndSession(t *testing.T) {
	a, err := authorizer.New(_conf)
	require.Nil(t, err)

	tokenString, err := a.Login("root", "123456")
	require.Nil(t, err)
	t.Log(tokenString)

	token, err := a.Session(tokenString)
	require.Nil(t, err)

	assert.NotEmpty(t, token.UserId)
}

func Test_IAuthorizer_ListPolicy(t *testing.T) {
	a, err := authorizer.New(_conf)
	require.Nil(t, err)

	policies := a.ListSubjectPolicies("root")
	t.Logf("%+v", policies)
}
