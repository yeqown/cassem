package coordinator

import (
	"context"
	"testing"

	"github.com/casbin/casbin/v2/log"
	"github.com/stretchr/testify/suite"

	"github.com/yeqown/cassem/api/concept"
	apicassemdb "github.com/yeqown/cassem/internal/cassemdb/api"
)

type rbacTestSuite struct {
	suite.Suite

	rbac concept.RBAC
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

	// Initialize with default policy
	_ = s.rbac.(aclImpl)
}

func (s *rbacTestSuite) print() {
	s.T().Log("casbin model:")
	l := &log.DefaultLogger{}
	l.EnableLog(true)
	s.rbac.(aclImpl).e.GetModel().SetLogger(l)
	s.rbac.(aclImpl).e.GetModel().PrintModel()
	s.rbac.(aclImpl).e.GetModel().PrintPolicy()
}

func (s *rbacTestSuite) Test_AddUser() {
	err := s.rbac.AddUser(&concept.User{
		Account:        "yeqown@gmail.com",
		Nickname:       "yeqown",
		HashedPassword: "123456",
		Salt:           "",
		Status:         concept.User_NORMAL,
	})
	s.NoError(err)
}

func (s *rbacTestSuite) Test_DisableUser() {
	err := s.rbac.DisableUser("yeqown@gmail.com")
	s.NoError(err)

	// disable not exists account
	err2 := s.rbac.DisableUser("yeqown@qq.com")
	s.NoError(err2)
}

func (s *rbacTestSuite) Test_AssignRoleToUser() {
	err := s.rbac.AssignRole("yeqown", "admin", concept.Domain_ALL)
	s.NoError(err)
	s.print()
}

func (s *rbacTestSuite) Test_RevokeRoleFromUser() {
	err := s.rbac.RevokeRole("yeqown", "admin", concept.Domain_ALL)
	s.NoError(err)
}

func (s *rbacTestSuite) Test_Enforce() {
	//_ = s.rbac.AssignRole("yeqown", "admin", concept.Domain_ALL)
	s.print()

	allow, err := s.rbac.Enforce("superadmin", concept.Domain_ALL, concept.Object_ELEMENT, concept.Action_READ)
	s.NoError(err)
	s.True(allow)
	// _ = s.rbac.AssignRole("yeqown", "admin", concept.Domain_ALL)

	allow, err = s.rbac.Enforce("admin", concept.Domain_ALL, concept.Object_ELEMENT, concept.Action_READ)
	s.NoError(err)
	s.True(allow)

	allow, err = s.rbac.Enforce("yeqown@gmail.com", concept.Domain_CLUSTER, concept.Object_APP, concept.Action_READ)
	s.NoError(err)
	s.True(allow)

	allow, err = s.rbac.Enforce("yeqown2", concept.Domain_ALL, concept.Object_ELEMENT, concept.Action_READ)
	s.NoError(err)
	s.False(allow)
}

func (s *rbacTestSuite) Test_AutoMigrate() {
	err := s.rbac.(aclImpl).AutoMigrate()
	s.NoError(err)

	err = s.rbac.(aclImpl).a.SavePolicy(s.rbac.(aclImpl).e.GetModel())
	s.NoError(err)
}

func Test_RBAC(t *testing.T) {
	suite.Run(t, new(rbacTestSuite))
}
