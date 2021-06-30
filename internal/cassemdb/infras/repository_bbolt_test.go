package infras

import (
	"testing"

	"github.com/stretchr/testify/suite"

	"github.com/yeqown/cassem/pkg/conf"
	"github.com/yeqown/cassem/pkg/types"
)

type testRepositoryBBoltSuite struct {
	suite.Suite

	repo Repository
}

func (s testRepositoryBBoltSuite) TearDownSuite() {
	// clear testdata
}

var v = types.StoreValue{
	Fingerprint: "1231231",
	Key:         "a/b",
	Val:         []byte("hello"),
	Size:        5,
	CreatedAt:   123,
	UpdatedAt:   123,
}

func (s testRepositoryBBoltSuite) TestSet() {
	err := s.repo.SetKV("a/b", v)
	s.NoError(err)
}

func (s testRepositoryBBoltSuite) TestGet() {
	val, err := s.repo.GetKV("a/b")
	s.NoError(err)
	s.Equal(v, val)
}

func Test_Repo_BBolt_mysql(t *testing.T) {
	cfg := conf.BBolt{
		Dir: "./debugdata",
		DB:  "cassem.db",
	}

	repo, err := newRepository(&cfg)
	if err != nil {
		t.Fatalf("Test_Repo_BBolt_mysql failed to open DB: %v", err)
	}

	s := testRepositoryBBoltSuite{
		repo: repo,
	}

	suite.Run(t, &s)
}
