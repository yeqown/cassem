package infras_test

import (
	"testing"

	"github.com/stretchr/testify/suite"

	"github.com/yeqown/cassem/internal/cassemadm/infras"
)

type cassemadmRepoTestSuite struct {
	*suite.Suite

	repo infras.Repository
}

func (c cassemadmRepoTestSuite) SetupSuite() {
	panic("implement me")
}

func Test_cassemadm_Repository(t *testing.T) {
	suite.Run(t, new(cassemadmRepoTestSuite))
}
