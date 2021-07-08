package repository

import (
	"path"
	"strings"

	"github.com/pkg/errors"
	bolt "go.etcd.io/bbolt"

	"github.com/yeqown/cassem/pkg/conf"
	"github.com/yeqown/cassem/pkg/runtime"
	"github.com/yeqown/cassem/pkg/types"
)

var (
	ErrNotFound     = errors.New("record not found")
	ErrExists       = errors.New("key exists")
	ErrEmptyNode    = errors.New("empty node")
	ErrEmptyLeaf    = errors.New("empty leaf")
	ErrNoSuchBucket = errors.New("no such bucket")
)

type bboltRepoImpl struct {
	db *bolt.DB
}

func (b bboltRepoImpl) locateBucket(
	tx *bolt.Tx, nodes []string, createBucketNotFound bool) (buk *bolt.Bucket, err error) {

	for idx, node := range nodes {
		if strings.Compare(node, "") == 0 {
			return nil, ErrEmptyNode
		}

		name := runtime.ToBytes(node)
		if idx == 0 {
			buk = tx.Bucket(name)
			if buk == nil && createBucketNotFound {
				if buk, err = tx.CreateBucket(name); err != nil {
					break
				}
			}
			continue
		}

		if buk == nil {
			break
		}
		if buk2 := buk.Bucket(runtime.ToBytes(node)); buk2 == nil && createBucketNotFound {
			if buk, err = buk.CreateBucket(name); err != nil {
				break
			}
		} else {
			buk = buk2
		}
	}

	if err != nil {
		return nil, err
	}

	if buk == nil {
		return nil, ErrNoSuchBucket
	}

	return buk, nil
}

func (b bboltRepoImpl) GetKV(key types.StoreKey, isDir bool) (val *types.StoreValue, err error) {
	nodes, leaf := KeySplitter(key)
	if !isDir && IsEmptyLeaf(leaf) {
		return nil, ErrEmptyLeaf
	}

	var d []byte
	err = b.db.View(func(tx *bolt.Tx) error {
		bucket, err := b.locateBucket(tx, nodes, false)
		if err != nil {
			return err
		}

		if isDir {
			return nil
		}

		if d = bucket.Get(runtime.ToBytes(leaf)); d == nil {
			return ErrNotFound
		}

		return nil
	})
	if err != nil {
		return
	}

	if isDir {
		return &types.StoreValue{Key: key}, nil
	}

	val = new(types.StoreValue)
	err = val.Unmarshal(d)

	return val, err
}

func (b bboltRepoImpl) SetKV(key types.StoreKey, val types.StoreValue, isDir bool) (err error) {
	nodes, leaf := KeySplitter(key)
	if IsEmptyLeaf(leaf) {
		return ErrEmptyLeaf
	}

	d, err := val.Marshal()
	if err != nil {
		return err
	}

	err = b.db.Update(func(tx *bolt.Tx) error {
		bucket, err := b.locateBucket(tx, nodes, true)
		if err != nil {
			return err
		}

		if isDir {
			return nil
		}

		return bucket.Put(runtime.ToBytes(leaf), d)
	})

	return
}

func (b bboltRepoImpl) UnsetKV(key types.StoreKey, isDir bool) (err error) {
	nodes, leaf := KeySplitter(key)
	if IsEmptyLeaf(leaf) {
		return ErrEmptyLeaf
	}

	err = b.db.Update(func(tx *bolt.Tx) error {
		bucket, err := b.locateBucket(tx, nodes, true)
		if err != nil {
			return err
		}

		if isDir {
			return bucket.DeleteBucket(runtime.ToBytes(leaf))
		}

		return bucket.Delete(runtime.ToBytes(leaf))
	})

	return
}

func NewRepository(c *conf.Bolt) (Repository, error) {
	db, err := bolt.Open(path.Join(c.Dir, c.DB), 0600, nil)
	if err != nil {
		return nil, errors.Wrap(err, "open bolt.DB failed")
	}

	return newRepositoryWithDB(db), nil
}

func newRepositoryWithDB(db *bolt.DB) Repository {
	return bboltRepoImpl{
		db: db,
	}
}
