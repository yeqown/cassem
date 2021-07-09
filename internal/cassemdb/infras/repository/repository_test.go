package repository

import (
	"strconv"
	"testing"
	"time"

	"github.com/stretchr/testify/suite"

	"github.com/stretchr/testify/assert"

	"github.com/yeqown/cassem/pkg/conf"
)

var (
	_emptyNodes []string
	_emptyLeaf  string
)

func TestKeySplitter(t *testing.T) {
	type args struct {
		s StoreKey
	}
	tests := []struct {
		name      string
		args      args
		wantNodes []string
		wantLeaf  string
	}{
		{
			name:      "case 0",
			args:      args{s: "/a"},
			wantNodes: []string{""},
			wantLeaf:  "a",
		},
		{
			name:      "case 1",
			args:      args{s: "a/"},
			wantNodes: []string{"a"},
			wantLeaf:  _emptyLeaf,
		},
		{
			name:      "case 2",
			args:      args{s: "a/b/c/d"},
			wantNodes: []string{"a", "b", "c"},
			wantLeaf:  "d",
		},
		{
			name:      "case 3",
			args:      args{s: "/"},
			wantNodes: []string{""},
			wantLeaf:  _emptyLeaf,
		},
		{
			name:      "case 4",
			args:      args{s: "a"},
			wantNodes: _emptyNodes,
			wantLeaf:  "a",
		},
		{
			name:      "case 5",
			args:      args{s: ""},
			wantNodes: _emptyNodes,
			wantLeaf:  _emptyLeaf,
		},
		{
			name:      "case 6",
			args:      args{s: "a/b"},
			wantNodes: []string{"a"},
			wantLeaf:  "b",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotNodes, gotLeaf := keySplitter(tt.args.s)
			assert.Equal(t, tt.wantNodes, gotNodes)
			assert.Equal(t, tt.wantLeaf, gotLeaf)
		})
	}
}

type testRepositoryBBoltSuite struct {
	suite.Suite

	repo KV
}

func (s testRepositoryBBoltSuite) TearDownSuite() {
	// clear testdata
}

var _setkv = &StoreValue{
	Fingerprint: "1231231",
	Key:         "a/b",
	Val:         []byte("hello"),
	Size:        5,
	CreatedAt:   123,
	UpdatedAt:   123,
	TTL:         123,
}

func (s testRepositoryBBoltSuite) Test_SetKV() {
	err := s.repo.SetKV("a/b", _setkv, false)
	s.NoError(err)
}

func (s testRepositoryBBoltSuite) Test_GetKV() {
	val, err := s.repo.GetKV("a/b", false)
	s.NoError(err)
	s.Equal(_setkv, val)
}

func (s testRepositoryBBoltSuite) Test_Range() {
	for i := 0; i < 10; i++ {
		k, v := NewKVWithCreatedAt("range/"+strconv.Itoa(i), []byte("range value"), 0, time.Now().Unix())
		err := s.repo.SetKV(k, &v, false)
		s.NoError(err)
	}

	for i := 0; i < 2; i++ {
		k := StoreKey("range/dir" + strconv.Itoa(i))
		err := s.repo.SetKV(k, nil, true)
		s.NoError(err)
	}

	result, err := s.repo.Range("range", "", 6)
	s.Require().NoError(err)
	s.T().Logf("%+v", result)
	s.Require().Equal(6, len(result.Items))
	s.Require().True(result.HasMore)
	s.Require().NotEmpty(result.NextSeekKey)
	s.Require().Equal("6", result.NextSeekKey)

	result, err = s.repo.Range("range", result.NextSeekKey, 6)
	s.Require().NoError(err)
	s.T().Logf("%+v", result)
	s.Require().Equal(6, len(result.Items))
	s.Require().False(result.HasMore)
	s.Require().Empty(result.NextSeekKey)

	// Range empty dir
	result2, err2 := s.repo.Range("range/dir0", "", 100)
	s.Require().NoError(err2)
	s.Require().Equal(0, len(result2.Items))
	s.Require().False(result2.HasMore)
	s.Require().Empty(result2.NextSeekKey)
	s.T().Logf("%+v", result2)
}

func Test_Repo_BBolt_mysql(t *testing.T) {
	cfg := conf.Bolt{
		Dir: "./debugdata",
		DB:  "cassem.db",
	}

	repo, err := NewRepository(&cfg)
	if err != nil {
		t.Fatalf("Test_Repo_BBolt_mysql failed to open DB: %_setkv", err)
	}

	s := testRepositoryBBoltSuite{
		repo: repo,
	}

	suite.Run(t, &s)
}
