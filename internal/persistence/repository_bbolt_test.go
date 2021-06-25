package persistence_test

import (
	"testing"

	"github.com/yeqown/cassem/internal/conf"
	"github.com/yeqown/cassem/internal/persistence"
	"github.com/yeqown/cassem/internal/persistence/bbolt"

	"github.com/stretchr/testify/suite"
)

type testRepositoryBBoltSuite struct {
	suite.Suite

	repo persistence.Repository
}

func (s testRepositoryBBoltSuite) TearDownSuite() {
	// clear testdata
}

func (s testRepositoryBBoltSuite) TestSet() {
	err := s.repo.Set("a/b", []byte("a1231231"))
	s.NoError(err)
}

func (s testRepositoryBBoltSuite) TestGet() {
	val, err := s.repo.Get("a/b")
	s.NoError(err)
	s.Equal([]byte("a1231231"), val)
}

func Test_Repo_BBolt_mysql(t *testing.T) {
	cfg := conf.BBolt{
		Dir: "./debugdata",
		DB:  "cassem.db",
	}

	repo, err := bbolt.New(&cfg)
	if err != nil {
		t.Fatalf("Test_Repo_BBolt_mysql failed to open DB: %v", err)
	}

	s := testRepositoryBBoltSuite{
		repo: repo,
	}

	suite.Run(t, &s)
}
