package bbolt

import (
	"encoding/json"

	"github.com/yeqown/cassem/pkg/datatypes"
	"github.com/yeqown/cassem/pkg/runtime"
	"github.com/yeqown/cassem/pkg/set"
)

var (
	_ boltValuer = namespaceDO{}
	_ boltValuer = pairDO{}
	_ boltValuer = containerDO{}
	_ boltValuer = userDO{}
)

type namespaceDO struct {
	Key         string
	Description string
}

func (m namespaceDO) value() []byte {
	return runtime.ToBytes(m.Key)
}

func (m namespaceDO) key() []byte {
	v, _ := json.Marshal(m)
	return v
}

type boltValuer interface {
	key() []byte
	value() []byte
}

type pairDO struct {
	Key         string
	Description string
	Namespace   string
	Datatype    datatypes.Datatype
	Value       []byte
}

func (p pairDO) key() []byte {
	return runtime.ToBytes(p.Key)
}

func (p pairDO) value() []byte {
	v, _ := json.Marshal(p)
	return v
}

type containerDO struct {
	Key         string
	Description string
	Namespace   string
	CheckSum    string
	Fields      []field
}

func (c containerDO) key() []byte {
	return runtime.ToBytes(c.Key)
}

func (c containerDO) value() []byte {
	v, _ := json.Marshal(c)
	return v
}

type field struct {
	FieldType datatypes.FieldTyp
	Key       string
	Pairs     fieldPairs
}

// fieldPairs contains all pairs of fieldDO.
// KV_FIELD_ contains like: {"KV": "pairKey"}, "KV" is a const mark of KV field.
// LIST_FIELD_ contains like: {"0": "pairKey", "1": "pairKey"}, the `bucketKey` of fieldPairs is index of pairKey.
// DICT_FIELD_ contains like: {"dictKey": "pairKey"}
type fieldPairs map[string]string

// PairKeys returns all pairKey in fieldPairs.
//
// Notice that all pairKey should save into fieldPairs.Value, of course, you can change fieldPairs' definition, so
// you choose how to parse fieldPairs in customized way which is saved in it's definition.
func (f fieldPairs) PairKeys() []string {
	keys := make([]string, 0, len(f))
	for _, pairKey := range f {
		keys = append(keys, pairKey)
	}

	return keys
}

type formContainerParsed struct {
	c              *containerDO
	uniquePairKeys set.StringSet
}

type toOrigin uint32

const (
	toOriginDetail toOrigin = iota + 1 // detail
	toOriginPaging
)

type toContainerWithPairs struct {
	// origin indicates toContainerWithPairs.paris has value or not.
	// toOriginDetail means no data in pairs, otherwise pairs includes all pairs related to c.
	origin toOrigin

	// c contains containerDO
	c *containerDO

	// pairs means map[pairKey]*pairDO dictionary.
	pairs map[string]*pairDO
}

type userDO struct {
	Account  string
	Password string
	Name     string
}

func (u userDO) key() []byte {
	return runtime.ToBytes(u.Account)
}

func (u userDO) value() []byte {
	v, _ := json.Marshal(u)
	return v
}
