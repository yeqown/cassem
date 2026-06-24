package storage

import (
	"bytes"
	"errors"
	"fmt"
	apikv "github.com/yeqown/cassem/api/kv"
	"os"
	"path"
	"sync"

	"github.com/yeqown/log"
	bolt "go.etcd.io/bbolt"
	bolterrors "go.etcd.io/bbolt/errors"

	"github.com/yeqown/cassem/pkg/conf"
	errorx "github.com/yeqown/cassem/api/concept"
	"github.com/yeqown/cassem/pkg/runtime"
)

var (
	ErrNotFound       = fmt.Errorf("record not found: %w", errorx.Err_NOT_FOUND)
	ErrExists         = fmt.Errorf("key/bucket exists: %w", errorx.Err_ALREADY_EXISTS)
	ErrEmptyNode      = fmt.Errorf("empty node: %w", errorx.Err_INVALID_ARGUMENT)
	ErrEmptyLeaf      = fmt.Errorf("empty leaf: %w", errorx.Err_INVALID_ARGUMENT)
	ErrNoSuchBucket   = fmt.Errorf("no such bucket: %w", errorx.Err_NOT_FOUND)
	ErrNoParentBucket = fmt.Errorf("no parent bucket: %w", errorx.Err_INVALID_ARGUMENT)
)

type boltRepoImpl struct {
	mu sync.RWMutex
	db *bolt.DB

	// preWriteC chan *preWriteLog
}

func NewRepository(c *conf.Bolt) (KV, error) {
	db, err := openBoltDB(path.Join(c.Dir, c.DB))
	if err != nil {
		return nil, fmt.Errorf("open bolt.DB failed: %w", err)
	}

	return newRepositoryWithDB(db), nil
}

func openBoltDB(dbPath string) (*bolt.DB, error) {
	return bolt.Open(dbPath, 0600, &bolt.Options{
		Timeout:        0,
		NoGrowSync:     false,
		FreelistType:   bolt.FreelistArrayType,
		NoFreelistSync: true,
	})
}

func newRepositoryWithDB(db *bolt.DB) KV {
	b := &boltRepoImpl{
		mu: sync.RWMutex{},
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
func (b *boltRepoImpl) locateBucket(
	tx *bolt.Tx, key string, createBucketNotFound bool) (buk *bolt.Bucket, leaf string, err error) {
	nodes, leaf := KeySplitter(key)
	if len(nodes) == 0 {
		return nil, leaf, ErrNoParentBucket
	}

	if isEmptyLeaf(leaf) {
		return nil, leaf, ErrEmptyLeaf
	}

	for idx, node := range nodes {
		if node == "" {
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

func (b *boltRepoImpl) GetKV(key string, dir bool) (val *apikv.Entity, err error) {
	b.mu.RLock()
	defer b.mu.RUnlock()

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
		return &apikv.Entity{Key: key}, nil
	}

	val = new(apikv.Entity)
	err = apikv.Unmarshal(d, val)

	return val, err
}

func (b *boltRepoImpl) SetKV(key string, val *apikv.Entity, dir bool) (err error) {
	b.mu.RLock()
	defer b.mu.RUnlock()

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
		d := apikv.Must(apikv.Marshal(val))

		return bucket.Put(runtime.ToBytes(leaf), d)
	})

	return
}

func (b *boltRepoImpl) UnsetKV(key string, dir bool) (err error) {
	b.mu.RLock()
	defer b.mu.RUnlock()

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

	if errors.Is(err, bolterrors.ErrBucketNotFound) || errors.Is(err, ErrNoSuchBucket) {
		return nil
	}

	return
}

// Range key must be directory key.
func (b *boltRepoImpl) Range(key string, seek string, limit int) (*RangeResult, error) {
	b.mu.RLock()
	defer b.mu.RUnlock()

	var (
		err    error
		result *RangeResult
	)
	err = b.db.View(func(tx *bolt.Tx) error {
		bucket, leaf, err2 := b.locateBucket(tx, key, false)
		if err2 != nil {
			if !errors.Is(err2, ErrNoParentBucket) {
				return fmt.Errorf("range.locateBucket: %w", err2)
			}
			bucket = tx.Bucket(runtime.ToBytes(key))
		} else {
			bucket = bucket.Bucket(runtime.ToBytes(leaf))
		}
		if bucket == nil {
			return fmt.Errorf("range.locateLeafBuck: %w", ErrNoSuchBucket)
		}

		var (
			cur   = bucket.Cursor()
			count = 0
		)

		k, v := cur.First()
		result = &RangeResult{
			Items:       make([]*apikv.Entity, 0, limit),
			HasMore:     false,
			NextSeekKey: "",
		}
		if len(seek) != 0 {
			k, v = cur.Seek(runtime.ToBytes(seek))
		}

		for ; k != nil && count < limit; k, v = cur.Next() {
			entryKey := path.Join(key, runtime.ToString(k))
			entity := &apikv.Entity{
				Key: entryKey,
			}
			if v != nil {
				if err2 = apikv.Unmarshal(v, entity); err2 != nil {
					return fmt.Errorf("range.decode %s: %w", entryKey, err2)
				}
				// FIXED: shielding expired data in range
				if entity.Expired() {
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

func (b *boltRepoImpl) Snapshot() ([]byte, error) {
	b.mu.RLock()
	defer b.mu.RUnlock()

	buf := new(bytes.Buffer)
	if err := b.db.View(func(tx *bolt.Tx) error {
		_, err := tx.WriteTo(buf)
		return err
	}); err != nil {
		return nil, fmt.Errorf("boltRepoImpl.Snapshot: %w", err)
	}
	return buf.Bytes(), nil
}

func (b *boltRepoImpl) RecoverSnapshot(snapshot []byte) error {
	if len(snapshot) == 0 {
		return fmt.Errorf("empty snapshot: %w", errorx.Err_INVALID_ARGUMENT)
	}

	b.mu.Lock()
	defer b.mu.Unlock()

	dbPath := b.db.Path()
	tmpPath := dbPath + ".snapshot.tmp"
	if err := os.WriteFile(tmpPath, snapshot, 0600); err != nil {
		return fmt.Errorf("boltRepoImpl.RecoverSnapshot.write: %w", err)
	}
	tmpDB, err := openBoltDB(tmpPath)
	if err != nil {
		_ = os.Remove(tmpPath)
		return fmt.Errorf("boltRepoImpl.RecoverSnapshot.validate: %w", err)
	}
	if err = tmpDB.Close(); err != nil {
		_ = os.Remove(tmpPath)
		return fmt.Errorf("boltRepoImpl.RecoverSnapshot.validate.close: %w", err)
	}
	if err := b.db.Close(); err != nil {
		_ = os.Remove(tmpPath)
		return fmt.Errorf("boltRepoImpl.RecoverSnapshot.close: %w", err)
	}
	if err := os.Rename(tmpPath, dbPath); err != nil {
		_ = os.Remove(tmpPath)
		reopened, openErr := openBoltDB(dbPath)
		if openErr == nil {
			b.db = reopened
		}
		return fmt.Errorf("boltRepoImpl.RecoverSnapshot.rename: %w", err)
	}

	db, err := openBoltDB(dbPath)
	if err != nil {
		return fmt.Errorf("boltRepoImpl.RecoverSnapshot.open: %w", err)
	}
	b.db = db
	return nil
}
