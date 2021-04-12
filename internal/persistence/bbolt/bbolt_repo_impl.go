// Package bbolt aims to implement persistence.Repository based on boltDB which use buckets rather than tables, here is
// some concepts design:
// 1. each ns was kept in root DB. root's buckets contains all namespace buckets.
// 2. all pairs in one namespace are saved in one bucket named 'pair', so as containers with named 'container', which
// 	means namespace bucket contains 2 buckets (pair and container).
package bbolt

import (
	"bytes"
	"encoding/json"
	"fmt"
	"path"

	"github.com/yeqown/cassem/internal/conf"
	"github.com/yeqown/cassem/internal/persistence"
	"github.com/yeqown/cassem/pkg/runtime"
	"github.com/yeqown/cassem/pkg/set"

	"github.com/pkg/errors"
	"github.com/yeqown/log"
	bolt "go.etcd.io/bbolt"
)

var (
	_pairBucket      = []byte("pair")
	_containerBucket = []byte("container")

	ErrNotFound = errors.New("record not found")
)

type bboltRepoImpl struct {
	db *bolt.DB
}

func New(c *conf.BBolt) (persistence.Repository, error) {
	db, err := bolt.Open(path.Join(c.Dir, c.DB), 0600, nil)
	if err != nil {
		return nil, errors.Wrap(err, "open bolt.DB failed")
	}

	return NewWithDB(db), nil
}

func NewWithDB(db *bolt.DB) persistence.Repository {
	return bboltRepoImpl{
		db: db,
	}
}

func (b bboltRepoImpl) Migrate() (err error) {
	// there is no migration to do.
	return nil
}

func (b bboltRepoImpl) CannReplicated() bool {
	return true
}

func (b bboltRepoImpl) GetContainer(ns, containerKey string) (interface{}, error) {
	var (
		c     *containerDO
		pairs = make(map[string]*pairDO, 16)
	)
	err := b.db.View(func(tx *bolt.Tx) error {
		bu := getContainerBucketByNamespace(tx, ns)
		v := bu.Get(runtime.ToBytes(containerKey))
		if v == nil {
			return ErrNotFound
		}

		c = new(containerDO)
		return json.Unmarshal(v, c)
	})
	if err != nil {
		return nil, err
	}

	err = b.db.View(func(tx *bolt.Tx) error {
		uniquePairKeys := set.NewStringSet(len(c.Fields) * 4)
		for _, fld := range c.Fields {
			uniquePairKeys.Adds(fld.Pairs.PairKeys())
		}

		bu := getPairBucketByNamespace(tx, ns)
		for _, pairKey := range uniquePairKeys.Keys() {
			v := bu.Get(runtime.ToBytes(pairKey))
			if v == nil {
				log.Warnf("pair(%s) got nil value", pairKey)
				continue
			}
			p := new(pairDO)
			if err := json.Unmarshal(v, p); err != nil {
				log.Warn("could not marshal pair")
				continue
			}
			pairs[pairKey] = p
		}

		return nil
	})
	if err != nil {
		return nil, err
	}

	toc := &toContainerWithPairs{
		origin: toOriginDetail,
		c:      c,
		pairs:  pairs,
	}

	return toc, err
}

func (b bboltRepoImpl) SaveContainer(c interface{}, update bool) error {
	from, ok := c.(*formContainerParsed)
	if !ok || from == nil {
		return errors.New("invalid value of container")
	}

	return b.db.Update(func(tx *bolt.Tx) error {
		bu := getContainerBucketByNamespace(tx, from.c.Namespace)
		return bu.Put(from.c.key(), from.c.value())
	})
}

func (b bboltRepoImpl) PagingContainers(filter *persistence.PagingContainersFilter) ([]interface{}, int, error) {
	out := make([]interface{}, 0, filter.Limit)

	err := b.db.View(func(tx *bolt.Tx) error {
		bu := getContainerBucketByNamespace(tx, filter.Namespace)

		var kvs []kv
		if filter.KeyPattern == "" {
			kvs, _ = pagingHelper(bu, filter.Offset, filter.Limit)
		} else {
			kvs, _ = pagingHelperWithPrefix(bu, filter.KeyPattern, filter.Offset, filter.Limit)
		}

		for _, item := range kvs {
			c := new(containerDO)
			if err2 := json.Unmarshal(item.value, c); err2 != nil {
				log.
					WithFields(log.Fields{
						"value":     item.value,
						"bucketKey": item.key,
						"error":     err2,
					}).
					Warn("bboltRepoImpl.PagingContainers could not unmarshal container")

				continue
			}
			out = append(out, &toContainerWithPairs{
				origin: toOriginPaging,
				c:      c,
			})
		}

		return nil
	})

	return out, 0, err
}

func (b bboltRepoImpl) RemoveContainer(ns, containerKey string) error {
	return b.db.Update(func(tx *bolt.Tx) error {
		bu := getContainerBucketByNamespace(tx, ns)
		v := bu.Delete(runtime.ToBytes(containerKey))
		if v == nil {
			return ErrNotFound
		}

		return nil
	})
}

func (b bboltRepoImpl) UpdateContainerCheckSum(ns, key, checksum string) error {
	return b.db.Update(func(tx *bolt.Tx) error {
		bu := getContainerBucketByNamespace(tx, ns)
		v := bu.Get(runtime.ToBytes(key))
		if v == nil {
			return ErrNotFound
		}

		c := new(containerDO)
		if err := json.Unmarshal(v, c); err != nil {
			return err
		}

		c.CheckSum = checksum
		return bu.Put(c.key(), c.value())
	})
}

func (b bboltRepoImpl) GetPair(ns, key string) (interface{}, error) {
	var p *pairDO
	err := b.db.Update(func(tx *bolt.Tx) error {
		bu := getPairBucketByNamespace(tx, ns)
		v := bu.Get(runtime.ToBytes(key))
		if v == nil {
			return ErrNotFound
		}

		p = new(pairDO)
		return json.Unmarshal(v, p)
	})

	return p, err
}

func (b bboltRepoImpl) SavePair(v interface{}, update bool) error {
	p, ok := v.(*pairDO)
	if !ok || p == nil {
		return fmt.Errorf("invalid value of pairDO, ok: %v, p==nil: %v", ok, p == nil)
	}

	return b.db.Update(func(tx *bolt.Tx) error {
		return getPairBucketByNamespace(tx, p.Namespace).Put(p.key(), p.value())
	})
}

func (b bboltRepoImpl) PagingPairs(filter *persistence.PagingPairsFilter) ([]interface{}, int, error) {
	out := make([]interface{}, 0, filter.Limit)
	err := b.db.View(func(tx *bolt.Tx) error {
		bu := getPairBucketByNamespace(tx, filter.Namespace)
		var kvs []kv

		if filter.KeyPattern == "" {
			kvs, _ = pagingHelper(bu, filter.Offset, filter.Limit)
		} else {
			kvs, _ = pagingHelperWithPrefix(bu, filter.KeyPattern, filter.Offset, filter.Limit)
		}

		for _, item := range kvs {
			p := new(pairDO)
			if err2 := json.Unmarshal(item.value, p); err2 != nil {
				log.
					WithFields(log.Fields{
						"value":     item.value,
						"bucketKey": item.key,
						"error":     err2,
					}).
					Warn("bboltRepoImpl.PagingContainers could not unmarshal container")

				continue
			}
			out = append(out, p)
		}

		return nil
	})

	return out, 0, err
}

func (b bboltRepoImpl) PagingNamespace(
	filter *persistence.PagingNamespacesFilter) (out []string, total int, err error) {

	out = make([]string, 0, filter.Limit)
	if err = b.db.View(func(tx *bolt.Tx) error {
		return tx.ForEach(func(name []byte, b *bolt.Bucket) error {
			out = append(out, runtime.ToString(name))
			return nil
		})
	}); err != nil {
		return
	}

	return
}

func getPairBucketByNamespace(tx *bolt.Tx, ns string) *bolt.Bucket {
	return tx.Bucket(runtime.ToBytes(ns)).Bucket(_pairBucket)
}

func getContainerBucketByNamespace(tx *bolt.Tx, ns string) *bolt.Bucket {
	return tx.Bucket(runtime.ToBytes(ns)).Bucket(_containerBucket)
}

func (b bboltRepoImpl) SaveNamespace(ns string) error {
	return b.db.Update(func(tx *bolt.Tx) error {
		bu, err := tx.CreateBucketIfNotExists(runtime.ToBytes(ns))
		if err != nil {
			return err
		}

		if _, err = bu.CreateBucketIfNotExists(_pairBucket); err != nil {
			return err
		}

		if _, err = bu.CreateBucketIfNotExists(_containerBucket); err != nil {
			return err
		}

		return nil
	})
}

type kv struct {
	key   []byte
	value []byte
}

func pagingHelper(b *bolt.Bucket, offset, limit int) (out []kv, hasMore bool) {
	if b == nil {
		return
	}

	if cnt := b.Stats().KeyN; cnt <= offset || limit == 0 {
		return
	}

	c := b.Cursor()
	pos := 0
	out = make([]kv, 0, limit)
	for k, v := c.First(); k != nil; k, v = c.Next() {
		pos++

		if pos <= offset {
			continue
		}

		if pos > offset+limit {
			hasMore = true
			break
		}

		out = append(out, kv{key: k, value: v})
	}

	return
}

func pagingHelperWithPrefix(b *bolt.Bucket, prefix string, offset, limit int) (out []kv, hasMore bool) {
	if b == nil {
		return
	}

	if cnt := b.Stats().KeyN; cnt <= offset || limit == 0 {
		return
	}

	c := b.Cursor()
	pos := 0
	out = make([]kv, 0, limit)
	p := runtime.ToBytes(prefix)
	for k, v := c.Seek(p); k != nil && bytes.HasPrefix(k, p); k, v = c.Next() {
		pos++

		if pos <= offset {
			continue
		}

		if pos > offset+limit {
			hasMore = true
			break
		}

		out = append(out, kv{key: k, value: v})
	}

	return
}
