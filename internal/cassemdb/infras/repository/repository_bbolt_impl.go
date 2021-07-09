package repository

import (
	"path"
	"strings"

	"github.com/yeqown/log"

	"github.com/pkg/errors"
	bolt "go.etcd.io/bbolt"

	"github.com/yeqown/cassem/pkg/conf"
	"github.com/yeqown/cassem/pkg/runtime"
)

var (
	ErrNotFound     = errors.New("record not found")
	ErrExists       = errors.New("key/bucket exists")
	ErrEmptyNode    = errors.New("empty node")
	ErrEmptyLeaf    = errors.New("empty leaf")
	ErrNoSuchBucket = errors.New("no such bucket")
)

type boltRepoImpl struct {
	db *bolt.DB
}

func NewRepository(c *conf.Bolt) (KV, error) {
	db, err := bolt.Open(path.Join(c.Dir, c.DB), 0600, nil)
	if err != nil {
		return nil, errors.Wrap(err, "open bolt.DB failed")
	}

	return newRepositoryWithDB(db), nil
}

func newRepositoryWithDB(db *bolt.DB) KV {
	return boltRepoImpl{
		db: db,
	}
}

// locateBucket locate bucket which parameters specified.
// key is the path to bucket or key which can distinguish by isDir,
// createBucketNotFound means create bucket if bucket on key path does not exist.
func (b boltRepoImpl) locateBucket(
	tx *bolt.Tx, key StoreKey, isDir, createBucketNotFound bool) (buk *bolt.Bucket, leaf string, err error) {
	nodes, leaf := keySplitter(key)
	if isEmptyLeaf(leaf) {
		return nil, leaf, ErrEmptyLeaf
	}

	if isDir {
		nodes = append(nodes, leaf)
	}

	for idx, node := range nodes {
		if strings.Compare(node, "") == 0 {
			return nil, leaf, ErrEmptyNode
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
		return nil, leaf, err
	}

	if buk == nil {
		return nil, leaf, ErrNoSuchBucket
	}

	return buk, leaf, nil
}

func (b boltRepoImpl) GetKV(key StoreKey, isDir bool) (val *StoreValue, err error) {
	var d []byte
	err = b.db.View(func(tx *bolt.Tx) error {
		buk, leaf, err := b.locateBucket(tx, key, isDir, false)
		if err != nil {
			return err
		}

		if isDir {
			if buk = buk.Bucket(runtime.ToBytes(leaf)); buk != nil {
				return nil
			}

			return ErrNotFound
		}

		if d = buk.Get(runtime.ToBytes(leaf)); d == nil {
			return ErrNotFound
		}

		return nil
	})
	if err != nil {
		return
	}

	if isDir {
		return &StoreValue{Key: key}, nil
	}

	val = new(StoreValue)
	err = val.Unmarshal(d)

	return val, err
}

func (b boltRepoImpl) SetKV(key StoreKey, val *StoreValue, isDir bool) (err error) {
	err = b.db.Update(func(tx *bolt.Tx) error {
		bucket, leaf, err := b.locateBucket(tx, key, isDir, true)
		if err != nil {
			return err
		}

		if isDir {
			return nil
		}

		d, err := val.Marshal()
		if err != nil {
			return err
		}

		return bucket.Put(runtime.ToBytes(leaf), d)
	})

	return
}

func (b boltRepoImpl) UnsetKV(key StoreKey, isDir bool) (err error) {
	err = b.db.Update(func(tx *bolt.Tx) error {
		bucket, leaf, err := b.locateBucket(tx, key, isDir, true)
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

func (b boltRepoImpl) Range(key StoreKey, seek string, limit int) (result *RangeResult, err error) {
	err = b.db.View(func(tx *bolt.Tx) error {
		bucket, _, err := b.locateBucket(tx, key, true, false)
		if err != nil {
			return err
		}

		var (
			cur   = bucket.Cursor()
			count = 0
		)

		k, v := cur.First()
		result = &RangeResult{
			Items:       make([]StoreValue, 0, limit),
			HasMore:     false,
			NextSeekKey: "",
		}
		if len(seek) != 0 {
			k, v = cur.Seek(runtime.ToBytes(seek))
		}

		for ; k != nil && count < limit; k, v = cur.Next() {
			//typ := ItemType_KV
			//if v == nil {
			//	typ = ItemType_DIR
			//}
			sv := StoreValue{
				Key:  StoreKey(k),
				Val:  nil,
				Size: 0,
			}
			if v != nil {
				if err = sv.Unmarshal(v); err != nil {
					log.
						WithFields(log.Fields{"error": err, "raw": string(v)}).
						Error("could not be unmarshaled")
				}
			}

			result.Items = append(result.Items, sv)
			count++
		}

		// k, v = cur.Next()
		if k != nil {
			result.HasMore = true
			result.NextSeekKey = runtime.ToString(k)
		}

		return nil
	})

	if err != nil {
		return nil, err
	}

	return
}
