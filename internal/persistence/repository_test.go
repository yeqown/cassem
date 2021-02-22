package persistence_test

import (
	"testing"

	"github.com/yeqown/cassem/internal/persistence"
	"github.com/yeqown/cassem/internal/persistence/mysql"
	"github.com/yeqown/cassem/pkg/datatypes"

	"github.com/stretchr/testify/suite"
	mysqld "gorm.io/driver/mysql"
	"gorm.io/gorm"
)

type testRepositorySuite struct {
	suite.Suite

	repo persistence.Repository
}

func (s testRepositorySuite) TearDownSuite() {
	// clear testdata
}

func (s testRepositorySuite) Test_Converter() {
	cv := s.repo.Converter()
	s.NotNil(cv)
}

func (s testRepositorySuite) Test_Pair() {
	// from pair
	p := datatypes.NewPair("ns", "kv1", datatypes.WithBool(true))
	v, err := s.repo.Converter().FromPair(p)
	s.Require().Nil(err)
	s.Require().NotNil(v)
	err = s.repo.SavePair(v, false)
	s.Require().Nil(err)

	// read
	v, err = s.repo.GetPair("ns", "kv1")
	s.Require().Nil(err)
	s.Require().NotNil(v)
	// to pair
	pout, err := s.repo.Converter().ToPair(v)
	s.Require().Nil(err)
	s.Require().NotNil(v)
	s.Equal(p, pout)

	// test update
	p2 := datatypes.NewPair("ns", "kv1", datatypes.WithInt(32222))
	v2, _ := s.repo.Converter().FromPair(p2)
	err = s.repo.SavePair(v2, true)
	s.Require().Nil(err)
	// get and check
	v2, err = s.repo.GetPair("ns", "kv1")
	s.Require().Nil(err)
	s.Require().NotNil(v2)
	pout2, err := s.repo.Converter().ToPair(v2)
	s.Require().Nil(err)
	s.Require().NotNil(v)
	s.Equal(p2, pout2)
}

func (s testRepositorySuite) Test_Pair_Paging() {
	out, count, err := s.repo.PagingPairs(nil)
	s.Require().Nil(err)
	s.NotEmpty(count)
	s.NotNil(out)
}

func (s testRepositorySuite) Test_Namespace() {
	err := s.repo.SaveNamespace("")
	s.NotNil(err)
	err = s.repo.SaveNamespace("ns-1")
	s.Nil(err)
	err = s.repo.SaveNamespace("ns-1")
	s.NotNil(err)
	err = s.repo.SaveNamespace("ns-2")
	s.Nil(err)

	out, err := s.repo.PagingNamespace(nil)
	s.Nil(err)
	s.GreaterOrEqual(len(out), 2)
}

func Test_Repo_mysql(t *testing.T) {
	db, err := gorm.Open(mysqld.Open("root:@tcp(127.0.0.1:3306)/cassem?charset=utf8mb4&parseTime=true&loc=Local"), nil)
	if err != nil {
		t.Fatalf("Test_Repo_mysql failed to open mysql DB: %v", err)
	}

	if err = db.AutoMigrate(
		mysql.PairDO{},
		mysql.NamespaceDO{},
	); err != nil {
		t.Fatalf("Test_Repo_mysql failed to AutoMigrate mysql DB: %v", err)
	}

	s := testRepositorySuite{
		repo: mysql.New(db),
	}

	suite.Run(t, &s)
}
