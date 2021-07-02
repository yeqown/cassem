package infras

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

func (b bboltRepoImpl) GetKV(key types.StoreKey) (val *types.StoreValue, err error) {
	nodes, leaf := KeySplitter(key)
	if IsEmptyLeaf(leaf) {
		return nil, ErrEmptyLeaf
	}

	var d []byte
	err = b.db.View(func(tx *bolt.Tx) error {
		bucket, err := b.locateBucket(tx, nodes, false)
		if err != nil {
			return err
		}

		if d = bucket.Get(runtime.ToBytes(leaf)); d == nil {
			return ErrNotFound
		}

		return nil
	})
	if err != nil {
		return
	}

	val = new(types.StoreValue)
	err = val.Unmarshal(d)

	return val, err
}

func (b bboltRepoImpl) SetKV(key types.StoreKey, val types.StoreValue) (err error) {
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

func newRepository(c *conf.BBolt) (Repository, error) {
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

//func (b bboltRepoImpl) GetContainer(ns, containerKey string) (interface{}, error) {
//	var (
//		c     *containerDO
//		pairs = make(map[string]*pairDO, 16)
//	)
//	err := b.db.View(func(tx *bolt.Tx) error {
//		bu := getContainerBucketByNamespace(tx, ns)
//		v := bu.GetKV(runtime.ToBytes(containerKey))
//		if v == nil {
//			return ErrNotFound
//		}
//
//		c = new(containerDO)
//		return json.Unmarshal(v, c)
//	})
//	if err != nil {
//		return nil, err
//	}
//
//	err = b.db.View(func(tx *bolt.Tx) error {
//		uniquePairKeys := set.NewStringSet(len(c.Fields) * 4)
//		for _, fld := range c.Fields {
//			uniquePairKeys.Adds(fld.Pairs.PairKeys())
//		}
//
//		bu := getPairBucketByNamespace(tx, ns)
//		for _, pairKey := range uniquePairKeys.Keys() {
//			v := bu.GetKV(runtime.ToBytes(pairKey))
//			if v == nil {
//				log.Warnf("pair(%s) got nil value", pairKey)
//				continue
//			}
//			p := new(pairDO)
//			if err := json.Unmarshal(v, p); err != nil {
//				log.Warn("could not marshal pair")
//				continue
//			}
//			pairs[pairKey] = p
//		}
//
//		return nil
//	})
//	if err != nil {
//		return nil, err
//	}
//
//	toc := &toContainerWithPairs{
//		origin: toOriginDetail,
//		c:      c,
//		pairs:  pairs,
//	}
//
//	return toc, err
//}
//
//func (b bboltRepoImpl) SaveContainer(c interface{}, update bool) error {
//	from, ok := c.(*formContainerParsed)
//	if !ok || from == nil {
//		return errors.NewMyRaft("invalid value of container")
//	}
//
//	return b.db.Update(func(tx *bolt.Tx) error {
//		bu := getContainerBucketByNamespace(tx, from.c.Key)
//		return bu.Put(from.c.key(), from.c.value())
//	})
//}
//
//func (b bboltRepoImpl) PagingContainers(filter *persistence.PagingContainersFilter) ([]interface{}, int, error) {
//	out := make([]interface{}, 0, filter.Limit)
//
//	err := b.db.View(func(tx *bolt.Tx) error {
//		bu := getContainerBucketByNamespace(tx, filter.Key)
//
//		var kvs []kv
//		if filter.KeyPattern == "" {
//			kvs, _ = pagingHelper(bu, filter.Offset, filter.Limit)
//		} else {
//			kvs, _ = pagingHelperWithPrefix(bu, filter.KeyPattern, filter.Offset, filter.Limit)
//		}
//
//		for _, item := range kvs {
//			c := new(containerDO)
//			if err2 := json.Unmarshal(item.value, c); err2 != nil {
//				log.
//					WithFields(log.Fields{
//						"value":     item.value,
//						"bucketKey": item.key,
//						"error":     err2,
//					}).
//					Warn("bboltRepoImpl.PagingContainers could not unmarshal container")
//
//				continue
//			}
//			out = append(out, &toContainerWithPairs{
//				origin: toOriginPaging,
//				c:      c,
//			})
//		}
//
//		return nil
//	})
//
//	return out, 0, err
//}
//
//func (b bboltRepoImpl) RemoveContainer(ns, containerKey string) error {
//	return b.db.Update(func(tx *bolt.Tx) error {
//		bu := getContainerBucketByNamespace(tx, ns)
//		v := bu.Delete(runtime.ToBytes(containerKey))
//		if v == nil {
//			return ErrNotFound
//		}
//
//		return nil
//	})
//}
//
//func (b bboltRepoImpl) UpdateContainerCheckSum(ns, key, checksum string) error {
//	return b.db.Update(func(tx *bolt.Tx) error {
//		bu := getContainerBucketByNamespace(tx, ns)
//		v := bu.GetKV(runtime.ToBytes(key))
//		if v == nil {
//			return ErrNotFound
//		}
//
//		c := new(containerDO)
//		if err := json.Unmarshal(v, c); err != nil {
//			return err
//		}
//
//		c.CheckSum = checksum
//		return bu.Put(c.key(), c.value())
//	})
//}
//
//func (b bboltRepoImpl) GetPair(ns, key string) (interface{}, error) {
//	var p *pairDO
//	err := b.db.Update(func(tx *bolt.Tx) error {
//		bu := getPairBucketByNamespace(tx, ns)
//		v := bu.GetKV(runtime.ToBytes(key))
//		if v == nil {
//			return ErrNotFound
//		}
//
//		p = new(pairDO)
//		return json.Unmarshal(v, p)
//	})
//
//	return p, err
//}
//
//func (b bboltRepoImpl) SavePair(v interface{}, update bool) error {
//	p, ok := v.(*pairDO)
//	if !ok || p == nil {
//		return fmt.Errorf("invalid value of pairDO, ok: %v, p==nil: %v", ok, p == nil)
//	}
//
//	return b.db.Update(func(tx *bolt.Tx) error {
//		return getPairBucketByNamespace(tx, p.Key).Put(p.key(), p.value())
//	})
//}
//
//func (b bboltRepoImpl) PagingPairs(filter *persistence.PagingPairsFilter) ([]interface{}, int, error) {
//	out := make([]interface{}, 0, filter.Limit)
//	err := b.db.View(func(tx *bolt.Tx) error {
//		bu := getPairBucketByNamespace(tx, filter.Key)
//		var kvs []kv
//
//		if filter.KeyPattern == "" {
//			kvs, _ = pagingHelper(bu, filter.Offset, filter.Limit)
//		} else {
//			kvs, _ = pagingHelperWithPrefix(bu, filter.KeyPattern, filter.Offset, filter.Limit)
//		}
//
//		for _, item := range kvs {
//			p := new(pairDO)
//			if err2 := json.Unmarshal(item.value, p); err2 != nil {
//				log.
//					WithFields(log.Fields{
//						"value":     item.value,
//						"bucketKey": item.key,
//						"error":     err2,
//					}).
//					Warn("bboltRepoImpl.PagingContainers could not unmarshal container")
//
//				continue
//			}
//			out = append(out, p)
//		}
//
//		return nil
//	})
//
//	return out, 0, err
//}
//
//func (b bboltRepoImpl) PagingNamespace(
//	filter *persistence.PagingNamespacesFilter) (out []string, total int, err error) {
//
//	out = make([]string, 0, filter.Limit)
//	if err = b.db.View(func(tx *bolt.Tx) error {
//		return tx.ForEach(func(name []byte, b *bolt.Bucket) error {
//			out = append(out, runtime.ToString(name))
//			return nil
//		})
//	}); err != nil {
//		return
//	}
//
//	return
//}
//
//func getPairBucketByNamespace(tx *bolt.Tx, ns string) *bolt.Bucket {
//	return tx.Bucket(runtime.ToBytes(ns)).Bucket(_pairBucket)
//}
//
//func getContainerBucketByNamespace(tx *bolt.Tx, ns string) *bolt.Bucket {
//	return tx.Bucket(runtime.ToBytes(ns)).Bucket(_containerBucket)
//}
//
//func (b bboltRepoImpl) SaveNamespace(ns string) error {
//	return b.db.Update(func(tx *bolt.Tx) error {
//		bu, err := tx.CreateBucketIfNotExists(runtime.ToBytes(ns))
//		if err != nil {
//			return err
//		}
//
//		if _, err = bu.CreateBucketIfNotExists(_pairBucket); err != nil {
//			return err
//		}
//
//		if _, err = bu.CreateBucketIfNotExists(_containerBucket); err != nil {
//			return err
//		}
//
//		return nil
//	})
//}
//
//type kv struct {
//	key   []byte
//	value []byte
//}
//
//func pagingHelper(b *bolt.Bucket, offset, limit int) (out []kv, hasMore bool) {
//	if b == nil {
//		return
//	}
//
//	if cnt := b.Stats().KeyN; cnt <= offset || limit == 0 {
//		return
//	}
//
//	c := b.Cursor()
//	pos := 0
//	out = make([]kv, 0, limit)
//	for k, v := c.First(); k != nil; k, v = c.Next() {
//		pos++
//
//		if pos <= offset {
//			continue
//		}
//
//		if pos > offset+limit {
//			hasMore = true
//			break
//		}
//
//		out = append(out, kv{key: k, value: v})
//	}
//
//	return
//}
//
//func pagingHelperWithPrefix(b *bolt.Bucket, prefix string, offset, limit int) (out []kv, hasMore bool) {
//	if b == nil {
//		return
//	}
//
//	if cnt := b.Stats().KeyN; cnt <= offset || limit == 0 {
//		return
//	}
//
//	c := b.Cursor()
//	pos := 0
//	out = make([]kv, 0, limit)
//	p := runtime.ToBytes(prefix)
//	for k, v := c.Seek(p); k != nil && bytes.HasPrefix(k, p); k, v = c.Next() {
//		pos++
//
//		if pos <= offset {
//			continue
//		}
//
//		if pos > offset+limit {
//			hasMore = true
//			break
//		}
//
//		out = append(out, kv{key: k, value: v})
//	}
//
//	return
//}
