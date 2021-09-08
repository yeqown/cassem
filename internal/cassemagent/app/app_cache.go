package app

import (
	"github.com/yeqown/log"

	"github.com/yeqown/cassem/api/concept"
)

// queryCacheResult result of queryFromCache call, it indicates how many elements
// can be found from cache, how many keys are missed.
type queryCacheResult struct {
	elems []*concept.Element
	miss  []string
}

// queryFromCache only hit and valid cache element will be returned.
func (d app) queryFromCache(app, env string, keys ...string) *queryCacheResult {
	r := &queryCacheResult{
		elems: make([]*concept.Element, 0, len(keys)),
		miss:  make([]string, 0, len(keys)),
	}

	for _, k := range keys {
		elem, ok := d.cache.Query(app, env, k)
		if !ok {
			r.miss = append(r.miss, k)
			continue
		}

		r.elems = append(r.elems, elem)
	}

	log.
		WithFields(log.Fields{
			"app":    app,
			"env":    env,
			"result": r,
		}).
		Debug("agent.app.queryFromCache called")

	return r
}

func (d app) updateCache(app, env string, elems ...*concept.Element) {
	log.
		WithFields(log.Fields{
			"app":   app,
			"env":   env,
			"elems": elems,
		}).
		Debug("agent.app.updateCache called")

	for idx := range elems {
		d.cache.Set(app, env, elems[idx].GetMetadata().GetKey(), elems[idx])
	}
}
