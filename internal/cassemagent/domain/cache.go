package domain

import (
	"strings"
	"sync"
	"time"

	"github.com/yeqown/log"

	"github.com/yeqown/cassem/concept"
	"github.com/yeqown/cassem/internal/cassemagent/infras/lru"
)

var (
	meaningless = struct{}{}
)

type Cache interface {
	Query(app, env, key string) (*concept.Element, bool)
	Set(app, env, key string, elem *concept.Element)
}

// appPool is the root of cache object, managing apps and their envPool objects.
type appPool struct {
	pool sync.Map // map[app]*envPool
	cp   lru.CacheReplacing
}

// NewCache construct a cache instance to help app manages caches.
// FIXED(@yeqown): how to clear caches? or cache items will be released at time.
// Solution: limit cache size with cache replacing algorithm so that no need to worry about OOM.
//
// DONE(@yeqown): using LRU-2 cache replacing algorithm to limit cache.
func NewCache(size uint) Cache {
	c := &appPool{
		pool: sync.Map{},
	}

	c.cp, _ = lru.NewLRUK(2, size, size-1, c.evictCallback)

	return c
}

func (p *appPool) Query(app, env, key string) (*concept.Element, bool) {
	_, ok := p.cp.Get(p.genKey(app, env, key))
	if !ok {
		return nil, false
	}
	return p.query(app, env, key)
}

// Set a newest element of app-env-key
func (p *appPool) Set(app, env, key string, elem *concept.Element) {
	// cp(lru-k) only set cache value while 2+ more hit.
	set, _ := p.cp.Put(p.genKey(app, env, key), meaningless)
	if set {
		p.set(app, env, key, elem)
	}
}

func (p *appPool) genKey(app, env, key string) string {
	return app + "@" + env + "@" + key
}

func (p *appPool) parseKey(k string) (app string, env string, key string, ok bool) {
	arr := strings.Split(k, "@")
	if len(arr) != 3 {
		return
	}

	ok = true
	app = arr[0]
	env = arr[1]
	key = arr[2]

	return
}

// evictCallback will be called while an eviction happened which means one old
// cache is replaced by a new one, so here need to remove the evicted element
// from appPool.pool.
func (p *appPool) evictCallback(k interface{}, v interface{}) {
	key, ok := k.(string)
	if !ok {
		log.
			WithFields(log.Fields{
				"k": k,
			}).
			Warn("appPool.evictCallback received invalid key (not string type)")
		return
	}

	app, env, key, ok := p.parseKey(key)
	if !ok {
		log.
			WithFields(log.Fields{
				"k": k,
			}).
			Warn("appPool.evictCallback received invalid key (parse failed)")
		return
	}

	p.unset(app, env, key)
}

func (p *appPool) query(app, env, key string) (*concept.Element, bool) {
	b, ok := p.get(app, false)
	if !ok {
		return nil, false
	}

	return b.query(env, key)
}

// set create or update item in appPool.
func (p *appPool) set(app, env, key string, elem *concept.Element) {
	a, _ := p.get(app, true)
	e, _ := a.get(env, true)
	e.set(key, elem)
}

func (p *appPool) unset(app, env, key string) {
	a, ok := p.get(app, false)
	if !ok {
		return
	}
	e, ok := a.get(env, false)
	if !ok {
		return
	}
	e.unset(key)
}

func (p *appPool) get(app string, createIfNotExists bool) (b *envPool, ok bool) {
	var v interface{}
	if createIfNotExists {
		v, _ = p.pool.LoadOrStore(app, newEnvPool())
		ok = true
	} else {
		v, ok = p.pool.Load(app)
	}

	if !ok {
		return
	}

	b = v.(*envPool)

	return
}

type envPool struct {
	pool sync.Map // map[string]*elemPool
}

func newEnvPool() *envPool {
	return &envPool{pool: sync.Map{}}
}

func (b *envPool) get(env string, createIfNotExists bool) (e *elemPool, ok bool) {
	var v interface{}
	if createIfNotExists {
		v, _ = b.pool.LoadOrStore(env, newElemPool())
		ok = true
	} else {
		v, ok = b.pool.Load(env)
	}

	if !ok {
		return
	}

	e = v.(*elemPool)
	return
}

func (b *envPool) query(env, key string) (elem *concept.Element, ok bool) {
	e, ok := b.get(env, false)
	if !ok {
		return nil, false
	}

	return e.query(key)
}

type item struct {
	val *concept.Element
	// dirtyTime is last time val has been refreshed. it helps to judge should
	// request to cassemdb or not.
	dirtyTime time.Time
}

type elemPool struct {
	pool sync.Map // map[key]*item
}

func newElemPool() *elemPool {
	return &elemPool{pool: sync.Map{}}
}

func (e *elemPool) query(key string) (elem *concept.Element, ok bool) {
	v, ok := e.pool.Load(key)
	if !ok {
		return nil, false
	}

	i, ok := v.(*item)
	if !ok {
		return nil, ok
	}

	// ignore cache if now - dirtyTime > 10s
	elem = i.val
	ok = true
	if time.Now().Sub(i.dirtyTime) > 10*time.Second {
		ok = false
		elem = nil
	}

	return
}

func (e *elemPool) set(key string, elem *concept.Element) {
	e.pool.Store(key, &item{
		val:       elem,
		dirtyTime: time.Now(),
	})
}

func (e *elemPool) unset(key string) {
	e.pool.Delete(key)
}
