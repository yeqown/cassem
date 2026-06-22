package storage

import (
	apikv "github.com/yeqown/cassem/api/kv"
	"strconv"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"
	bolt "go.etcd.io/bbolt"
	"google.golang.org/protobuf/proto"

	"github.com/yeqown/cassem/pkg/conf"
	"github.com/yeqown/cassem/pkg/runtime"
)

var (
	_emptyNodes []string
	_emptyLeaf  string
)

func TestKeySplitter(t *testing.T) {
	type args struct {
		s string
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
			gotNodes, gotLeaf := KeySplitter(tt.args.s)
			assert.Equal(t, tt.wantNodes, gotNodes)
			assert.Equal(t, tt.wantLeaf, gotLeaf)
		})
	}
}

type testRepositoryBBoltSuite struct {
	suite.Suite

	repo KV
}

func (s *testRepositoryBBoltSuite) TearDownSuite() {
	// clear testdata
}

func (s *testRepositoryBBoltSuite) Test_locateBucket() {
	impl := s.repo.(*boltRepoImpl)
	_ = impl
}

var _setkv = &apikv.Entity{
	Fingerprint: "1231231",
	Key:         "a/b",
	Val:         []byte("hello"),
	Size:        5,
	CreatedAt:   123,
	UpdatedAt:   123,
	Ttl:         123,
}

func (s *testRepositoryBBoltSuite) Test_Set_Get_Unset_DIR() {
	var dirVal *apikv.Entity
	err := s.repo.SetKV("dir/b", dirVal, true)
	s.NoError(err)

	val, err := s.repo.GetKV("dir/b", true)
	s.Require().NoError(err)
	s.NotNil(val)
	s.Equal("dir/b", val.Key)

	err = s.repo.UnsetKV("dir/b", true)
	s.Require().NoError(err)

	val, err = s.repo.GetKV("dir/b", true)
	s.T().Logf("%+v", val)
	s.Error(err)
	s.Equal(ErrNoSuchBucket, err)
}

func (s *testRepositoryBBoltSuite) Test_Set_Get_Unset_KV() {
	err := s.repo.SetKV("kv/b", _setkv, false)
	s.NoError(err)

	val, err := s.repo.GetKV("kv/b", false)
	s.NoError(err)
	s.True(proto.Equal(_setkv, val))

	err = s.repo.UnsetKV("kv/b", false)
	s.NoError(err)

	val, err = s.repo.GetKV("kv/b", false)
	s.Error(err)
	s.Equal(ErrNotFound, err)
}

func (s *testRepositoryBBoltSuite) Test_Snapshot_Recover() {
	err := s.repo.SetKV("snapshot/dir", nil, true)
	s.Require().NoError(err)

	before := apikv.NewEntityWithCreated("snapshot/dir/key", []byte("before"), 0, time.Now().Unix())
	err = s.repo.SetKV("snapshot/dir/key", before, false)
	s.Require().NoError(err)

	snap, err := s.repo.Snapshot()
	s.Require().NoError(err)
	s.Require().NotEmpty(snap)

	after := apikv.NewEntityWithCreated("snapshot/dir/after", []byte("after"), 0, time.Now().Unix())
	err = s.repo.SetKV("snapshot/dir/after", after, false)
	s.Require().NoError(err)
	err = s.repo.UnsetKV("snapshot/dir/key", false)
	s.Require().NoError(err)

	err = s.repo.RecoverSnapshot(snap)
	s.Require().NoError(err)

	got, err := s.repo.GetKV("snapshot/dir/key", false)
	s.Require().NoError(err)
	s.True(proto.Equal(before, got))

	_, err = s.repo.GetKV("snapshot/dir/after", false)
	s.Error(err)
	s.Equal(ErrNotFound, err)
}

func (s *testRepositoryBBoltSuite) Test_RangeTopLevelBucket() {
	err := s.repo.SetKV("top/child", nil, true)
	s.Require().NoError(err)

	result, err := s.repo.Range("top", "", 10)
	s.Require().NoError(err)
	s.Require().Len(result.Items, 1)
	s.Equal("top/child", result.Items[0].GetKey())
}

func (s *testRepositoryBBoltSuite) Test_RangeReturnsDecodeErrors() {
	impl := s.repo.(*boltRepoImpl)
	err := impl.db.Update(func(tx *bolt.Tx) error {
		bucket, err := tx.CreateBucketIfNotExists(runtime.ToBytes("bad-range"))
		if err != nil {
			return err
		}
		return bucket.Put(runtime.ToBytes("broken"), []byte("not protobuf"))
	})
	s.Require().NoError(err)

	var result *RangeResult
	s.NotPanics(func() {
		result, err = s.repo.Range("bad-range", "", 10)
	})
	s.Require().Error(err)
	s.Nil(result)
	s.Contains(err.Error(), "bad-range/broken")
}

func (s *testRepositoryBBoltSuite) Test_Range() {
	err := s.repo.UnsetKV("range/dir", true)
	s.Require().NoError(err)

	// write kv under range/dir bucket
	for i := 0; i < 10; i++ {
		k := "range/dir/" + strconv.Itoa(i)
		v := apikv.NewEntityWithCreated(k, []byte("range value"), 0, time.Now().Unix())
		err := s.repo.SetKV(k, v, false)
		s.NoError(err)
	}

	// write dir under range/dir
	for i := 0; i < 2; i++ {
		k := string("range/dir/d" + strconv.Itoa(i))
		err := s.repo.SetKV(k, nil, true)
		s.NoError(err)
	}

	result, err := s.repo.Range("range/dir", "", 6)
	s.Require().NoError(err)
	s.T().Logf("%+v", result)
	s.Require().Equal(6, len(result.Items))
	s.Require().Equal("range/dir/0", result.Items[0].GetKey())
	s.Require().True(result.HasMore)
	s.Require().NotEmpty(result.NextSeekKey)
	s.Require().Equal("6", result.NextSeekKey)

	result, err = s.repo.Range("range/dir", result.NextSeekKey, 6)
	s.Require().NoError(err)
	s.T().Logf("%+v", result)
	s.Require().Equal(6, len(result.Items))
	s.Require().False(result.HasMore)
	s.Require().Empty(result.NextSeekKey)

	// Range empty dir
	result2, err2 := s.repo.Range("range/dir/d0", "", 100)
	s.Require().NoError(err2)
	s.Require().Equal(0, len(result2.Items))
	s.Require().False(result2.HasMore)
	s.Require().Empty(result2.NextSeekKey)
	s.T().Logf("%+v", result2)
}

func Test_Repo_BBolt_mysql(t *testing.T) {
	cfg := conf.Bolt{
		Dir: t.TempDir(),
		DB:  "cassem.db",
	}

	repo, err := NewRepository(&cfg)
	if err != nil {
		t.Fatalf("Test_Repo_BBolt_mysql failed to open DB: %v", err)
	}

	s := testRepositoryBBoltSuite{
		repo: repo,
	}

	suite.Run(t, &s)
}
