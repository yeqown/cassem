package repository

import (
	"path"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"
	"github.com/yeqown/log"
	bolt "go.etcd.io/bbolt"

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

func (s testRepositoryBBoltSuite) TearDownSuite() {
	// clear testdata
}

func (s testRepositoryBBoltSuite) Test_locateBucket() {
	impl := s.repo.(boltRepoImpl)
	_ = impl
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

func (s testRepositoryBBoltSuite) Test_Set_Get_Unset_DIR() {
	var dirVal *StoreValue
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

func (s testRepositoryBBoltSuite) Test_Set_Get_Unset_KV() {
	err := s.repo.SetKV("kv/b", _setkv, false)
	s.NoError(err)

	val, err := s.repo.GetKV("kv/b", false)
	s.NoError(err)
	s.Equal(_setkv, val)

	err = s.repo.UnsetKV("kv/b", false)
	s.NoError(err)

	val, err = s.repo.GetKV("kv/b", false)
	s.Error(err)
	s.Equal(ErrNotFound, err)
}

func (s testRepositoryBBoltSuite) Test_Range() {
	err := s.repo.UnsetKV("range/dir", true)
	s.Require().NoError(err)

	// write kv under range/dir bucket
	for i := 0; i < 10; i++ {
		k, v := NewKVWithCreatedAt("range/dir/"+strconv.Itoa(i), []byte("range value"), 0, time.Now().Unix())
		err := s.repo.SetKV(k, &v, false)
		s.NoError(err)
	}

	// write dir under range/dir
	for i := 0; i < 2; i++ {
		k := StoreKey("range/dir/d" + strconv.Itoa(i))
		err := s.repo.SetKV(k, nil, true)
		s.NoError(err)
	}

	result, err := s.repo.Range("range/dir", "", 6)
	s.Require().NoError(err)
	s.T().Logf("%+v", result)
	s.Require().Equal(6, len(result.Items))
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
		Dir: "./debugdata",
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

func Benchmark_bolt_write_32B(b *testing.B) {
	db, err := bolt.Open(path.Join("./debugdata", "cassem.db"), 0600, &bolt.Options{
		Timeout:        0,
		NoGrowSync:     false,
		FreelistType:   bolt.FreelistArrayType,
		NoFreelistSync: true,
	})
	if err != nil {
		b.Fatal(err)
	}

	// 32B
	bytes := []byte(strings.Repeat("a", 32))
	//val := &StoreValue{
	//	Fingerprint: "fingerprint",
	//	Key:         "benchmark/write_32B",
	//	Val:         bytes,
	//	Size:        int64(len(bytes)),
	//	CreatedAt:   time.Now().Unix(),
	//	UpdatedAt:   time.Now().Unix(),
	//	TTL:         30,
	//}

	b.ResetTimer()
	for i := 0; i < 1000; i++ {
		err = db.Batch(func(tx *bolt.Tx) error {
			buc, err2 := tx.CreateBucketIfNotExists([]byte("Benchmark_bolt_write_32B"))
			if err2 != nil {
				return err2
			}
			return buc.Put([]byte("benchmark/write_32B"), bytes)
		})
		if err != nil {
			b.Error(err)
		}
	}
}

func Benchmark_repo_write_32B(b *testing.B) {
	log.SetLogLevel(log.LevelError)
	cfg := conf.Bolt{
		Dir: "./debugdata",
		DB:  "cassem.db",
	}

	repo, err := NewRepository(&cfg)
	if err != nil {
		b.Fatalf("Test_Repo_BBolt_mysql failed to open DB: %_setkv", err)
	}

	// 32B
	bytes := []byte(strings.Repeat("a", 32))
	println("size:", len(bytes))
	val := &StoreValue{
		Fingerprint: "fingerprint",
		Key:         "benchmark/write_32B",
		Val:         bytes,
		Size:        int64(len(bytes)),
		CreatedAt:   time.Now().Unix(),
		UpdatedAt:   time.Now().Unix(),
		TTL:         30,
	}

	b.ResetTimer()
	for i := 0; i < 1000; i++ {
		err = repo.SetKV(val.Key, val, false)
		if err != nil {
			b.Error(err)
		}
	}
}

func Benchmark_repo_write_1KB(b *testing.B) {
	log.SetLogLevel(log.LevelError)
	cfg := conf.Bolt{
		Dir: "./debugdata",
		DB:  "cassem.db",
	}

	repo, err := NewRepository(&cfg)
	if err != nil {
		b.Fatalf("Test_Repo_BBolt_mysql failed to open DB: %_setkv", err)
	}

	// 1024 * 1 byte = 1KB
	bytes := []byte(strings.Repeat("a", 1024))
	println("size:", len(bytes))
	val := &StoreValue{
		Fingerprint: "fingerprint",
		Key:         "benchmark/write_1KB",
		Val:         bytes,
		Size:        int64(len(bytes)),
		CreatedAt:   time.Now().Unix(),
		UpdatedAt:   time.Now().Unix(),
		TTL:         30,
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		err = repo.SetKV(val.Key, val, false)
		if err != nil {
			b.Error(err)
		}
	}
}

func Benchmark_repo_write_10KB(b *testing.B) {
	log.SetLogLevel(log.LevelError)
	cfg := conf.Bolt{
		Dir: "./debugdata",
		DB:  "cassem.db",
	}

	repo, err := NewRepository(&cfg)
	if err != nil {
		b.Fatalf("Test_Repo_BBolt_mysql failed to open DB: %_setkv", err)
	}

	// // 1024 * 10 byte = 10KB
	bytes := []byte(strings.Repeat("1234567890", 1024))
	print("size:", len(bytes))
	val := &StoreValue{
		Fingerprint: "fingerprint",
		Key:         "benchmark/write_10KB",
		Val:         bytes,
		Size:        int64(len(bytes)),
		CreatedAt:   time.Now().Unix(),
		UpdatedAt:   time.Now().Unix(),
		TTL:         30,
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		err = repo.SetKV(val.Key, val, false)
		if err != nil {
			b.Error(err)
		}
	}
}
