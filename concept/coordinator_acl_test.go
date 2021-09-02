package concept

import (
	"context"
	"testing"

	"github.com/casbin/casbin/v2/log"
	"github.com/stretchr/testify/suite"

	apicassemdb "github.com/yeqown/cassem/internal/cassemdb/api"
)

type rbacTestSuite struct {
	suite.Suite

	rbac RBAC
	ctx  context.Context
}

func (s *rbacTestSuite) SetupSuite() {
	s.ctx = context.TODO()
	endpoints := []string{"127.0.0.1:2021", "127.0.0.1:2022", "127.0.0.1:2023"}
	cc, err := apicassemdb.DialWithMode(endpoints, apicassemdb.Mode_X)
	if err != nil {
		panic(err)
	}

	s.rbac, err = newRBAC(apicassemdb.NewKVClient(cc))
	if err != nil {
		panic(err)
	}
}

func (s rbacTestSuite) print() {
	s.T().Log("casbin model:")
	l := &log.DefaultLogger{}
	l.EnableLog(true)
	s.rbac.(aclImpl).e.GetModel().SetLogger(l)
	s.rbac.(aclImpl).e.GetModel().PrintModel()
	s.rbac.(aclImpl).e.GetModel().PrintPolicy()
}

func (s rbacTestSuite) Test_AddUser() {
	err := s.rbac.AddUser(&User{
		Account:        "yeqown@gmail.com",
		Nickname:       "yeqown",
		HashedPassword: "123456",
		Salt:           "",
		Status:         User_NORMAL,
	})
	s.NoError(err)
}

func (s rbacTestSuite) Test_DisableUser() {
	err := s.rbac.DisableUser("yeqown@gmail.com")
	s.NoError(err)

	// disable not exists account
	err2 := s.rbac.DisableUser("yeqown@qq.com")
	s.NoError(err2)
}

func (s rbacTestSuite) Test_AssignRoleToUser() {
	err := s.rbac.AssignRole("yeqown", "admin", Domain_ALL)
	s.NoError(err)
	s.print()
}

func (s rbacTestSuite) Test_RevokeRoleFromUser() {
	err := s.rbac.RevokeRole("yeqown", "admin", Domain_ALL)
	s.NoError(err)
}

func (s rbacTestSuite) Test_Enforce() {
	//_ = s.rbac.AssignRole("yeqown", "admin", Domain_ALL)
	s.print()

	allow, err := s.rbac.Enforce("superadmin", Domain_ALL, Object_ELEMENT, Action_READ)
	s.NoError(err)
	s.True(allow)
	// _ = s.rbac.AssignRole("yeqown", "admin", Domain_ALL)

	allow, err = s.rbac.Enforce("admin", Domain_ALL, Object_ELEMENT, Action_READ)
	s.NoError(err)
	s.True(allow)

	allow, err = s.rbac.Enforce("yeqown@gmail.com", Domain_CLUSTER, Object_APP, Action_READ)
	s.NoError(err)
	s.True(allow)

	allow, err = s.rbac.Enforce("yeqown2", Domain_ALL, Object_ELEMENT, Action_READ)
	s.NoError(err)
	s.False(allow)
}

func (s rbacTestSuite) Test_AutoMigrate() {
	err := s.rbac.(aclImpl).AutoMigrate()
	s.NoError(err)

	err = s.rbac.(aclImpl).a.SavePolicy(s.rbac.(aclImpl).e.GetModel())
	s.NoError(err)
}

func Test_RBAC(t *testing.T) {
	suite.Run(t, new(rbacTestSuite))
}
