package storage

import (
	"path"
	"strings"

	"github.com/pkg/errors"
	"github.com/yeqown/log"
	bolt "go.etcd.io/bbolt"

	apicassemdb "github.com/yeqown/cassem/internal/cassemdb/api"
	"github.com/yeqown/cassem/pkg/conf"
	"github.com/yeqown/cassem/pkg/errorx"
	"github.com/yeqown/cassem/pkg/runtime"
)

var (
	ErrNotFound       = errors.Wrap(errorx.Err_NOT_FOUND, "record not found")
	ErrExists         = errors.Wrap(errorx.Err_ALREADY_EXISTS, "key/bucket exists")
	ErrEmptyNode      = errors.Wrap(errorx.Err_INVALID_ARGUMENT, "empty node")
	ErrEmptyLeaf      = errors.Wrap(errorx.Err_INVALID_ARGUMENT, "empty leaf")
	ErrNoSuchBucket   = errors.Wrap(errorx.Err_NOT_FOUND, "no such bucket")
	ErrNoParentBucket = errors.Wrap(errorx.Err_INVALID_ARGUMENT, "no parent bucket")
)

type boltRepoImpl struct {
	db *bolt.DB

	// preWriteC chan *preWriteLog
}

func NewRepository(c *conf.Bolt) (KV, error) {
	db, err := bolt.Open(path.Join(c.Dir, c.DB), 0600, &bolt.Options{
		Timeout:        0,
		NoGrowSync:     false,
		FreelistType:   bolt.FreelistArrayType,
		NoFreelistSync: true,
	})
	if err != nil {
		return nil, errors.Wrap(err, "open bolt.DB failed")
	}

	return newRepositoryWithDB(db), nil
}

func newRepositoryWithDB(db *bolt.DB) KV {
	b := boltRepoImpl{
		db: db,
		// preWriteC: make(chan *preWriteLog, _PRE_WRITE_BUF_SIZE),
	}

	// run forever until the process quit.
	// runtime.GoFunc("boltRepoImpl.preWriteDispatcher", b.preWriteDispatcher)

	return b
}

// locateBucket locate bucket which parameters specified.
// createBucketNotFound means create bucket if bucket on key path does not exist.
//
// NOTE, such keys are invalid:
//
// 1: p
// 2: p/
// 3: p/p/
//
// and locateBucket only return the parent bucket of key, for example (p1/p2/leaf)
// returns buk: p1/p2, leaf: leaf, err: nil.
func (b boltRepoImpl) locateBucket(
	tx *bolt.Tx, key string, createBucketNotFound bool) (buk *bolt.Bucket, leaf string, err error) {
	nodes, leaf := KeySplitter(key)
	if len(nodes) == 0 {
		return nil, leaf, ErrNoParentBucket
	}

	if isEmptyLeaf(leaf) {
		return nil, leaf, ErrEmptyLeaf
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

func (b boltRepoImpl) GetKV(key string, dir bool) (val *apicassemdb.Entity, err error) {
	var d []byte
	err = b.db.View(func(tx *bolt.Tx) error {
		buk, leaf, err2 := b.locateBucket(tx, key, false)
		if err2 != nil {
			return err2
		}

		// locate leaf bucket while dir is true
		if dir {
			if buk = buk.Bucket(runtime.ToBytes(leaf)); buk != nil {
				return nil
			}
			return ErrNoSuchBucket
		}

		if d = buk.Get(runtime.ToBytes(leaf)); d == nil {
			return ErrNotFound
		}

		return nil
	})
	if err != nil {
		return
	}

	if dir {
		return &apicassemdb.Entity{Key: key}, nil
	}

	val = new(apicassemdb.Entity)
	err = apicassemdb.Unmarshal(d, val)

	return val, err
}

func (b boltRepoImpl) SetKV(key string, val *apicassemdb.Entity, dir bool) (err error) {
	log.
		WithFields(log.Fields{
			"key": key,
			"ttl": val.GetTtl(),
			"val": runtime.ToString(val.GetVal()),
			"dir": dir,
		}).
		Debug("boltRepoImpl.SetKV called")

	err = b.db.Batch(func(tx *bolt.Tx) error {
		bucket, leaf, err2 := b.locateBucket(tx, key, true)
		if err2 != nil {
			return err2
		}
		if dir {
			_, err2 = bucket.CreateBucketIfNotExists(runtime.ToBytes(leaf))
			return err2
		}
		d := apicassemdb.Must(apicassemdb.Marshal(val))

		return bucket.Put(runtime.ToBytes(leaf), d)
	})

	return
}

func (b boltRepoImpl) UnsetKV(key string, dir bool) (err error) {
	err = b.db.Batch(func(tx *bolt.Tx) error {
		bucket, leaf, err2 := b.locateBucket(tx, key, false)
		if err2 != nil {
			return err2
		}

		if dir {
			return bucket.DeleteBucket(runtime.ToBytes(leaf))
		}

		return bucket.Delete(runtime.ToBytes(leaf))
	})

	if errors.Is(err, bolt.ErrBucketNotFound) || errors.Is(err, ErrNoSuchBucket) {
		return nil
	}

	return
}

// Range key must be directory key.
func (b boltRepoImpl) Range(key string, seek string, limit int) (*RangeResult, error) {
	var (
		err    error
		result *RangeResult
	)
	err = b.db.View(func(tx *bolt.Tx) error {
		bucket, leaf, err2 := b.locateBucket(tx, key, false)
		if err2 != nil {
			return errors.Wrap(err2, "range.locateBucket")
		}
		bucket = bucket.Bucket(runtime.ToBytes(leaf))
		if bucket == nil {
			return errors.Wrap(ErrNoSuchBucket, "range.locateLeafBuck")
		}

		var (
			cur   = bucket.Cursor()
			count = 0
		)

		k, v := cur.First()
		result = &RangeResult{
			Items:       make([]*apicassemdb.Entity, 0, limit),
			HasMore:     false,
			NextSeekKey: "",
		}
		if len(seek) != 0 {
			k, v = cur.Seek(runtime.ToBytes(seek))
		}

		for ; k != nil && count < limit; k, v = cur.Next() {
			entity := &apicassemdb.Entity{
				Key: runtime.ToString(k),
			}
			if v != nil {
				apicassemdb.MustUnmarshal(v, entity)
				// FIXED: shielding expired data in range
				if err2 == nil && entity.Expired() {
					result.ExpiredKeys = append(result.ExpiredKeys, entity.Key)
					continue
				}
			}

			result.Items = append(result.Items, entity)
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

	return result, nil
}
