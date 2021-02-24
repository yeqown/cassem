package datatypes

import (
	"bytes"
	"encoding/json"

	"github.com/yeqown/cassem/pkg/hash"

	"github.com/BurntSushi/toml"
)

type FieldTyp uint8

const (
	KV_FIELD_ FieldTyp = iota + 1
	LIST_FIELD_
	DICT_FIELD_
)

type IField interface {
	IEncoder

	Name() string

	Type() FieldTyp

	Value() interface{}
}

var (
	_ IField = kvField{}
	_ IField = listField{}
	_ IField = dictField{}
)

type kvField struct {
	name string

	kv IPair
}

func NewKVField(fieldKey string, p IPair) IField {
	if fieldKey == "" {
		fieldKey = hashFieldKey()
	}

	return kvField{
		name: fieldKey,
		kv:   p,
	}
}

func (k kvField) Name() string {
	return k.name
}

func (k kvField) Type() FieldTyp {
	return KV_FIELD_
}

func (k kvField) Value() interface{} {
	return k.kv
}

func (k kvField) MarshalJSON() ([]byte, error) {
	return json.Marshal(k.kv)
}

func (k kvField) MarshalText() (text []byte, err error) {
	buf := bytes.NewBuffer(nil)
	err = toml.NewEncoder(buf).Encode(k.kv)

	return buf.Bytes(), err
}

type listField struct {
	name string

	pairs []IPair
}

func hashFieldKey() string {
	return "field" + hash.RandKey(6)
}

func NewListField(fieldKey string, pairs []IPair) IField {
	if fieldKey == "" {
		// DONE(@yeqown): use hashed string to name this fieldKey
		fieldKey = hashFieldKey()
	}

	if pairs == nil {
		pairs = make([]IPair, 0, 4)
	}

	return listField{
		name:  fieldKey,
		pairs: pairs,
	}
}

func (k listField) MarshalJSON() ([]byte, error) {
	return json.Marshal(k.pairs)
}

func (k listField) MarshalText() (text []byte, err error) {
	buf := bytes.NewBuffer(nil)
	err = toml.NewEncoder(buf).Encode(k.pairs)

	return buf.Bytes(), err
}

func (k listField) Name() string {
	return k.name
}

func (k listField) Type() FieldTyp {
	return LIST_FIELD_
}

func (k listField) Value() interface{} {
	return k.pairs
}

type dictField struct {
	name string

	pairs map[string]IPair
}

func NewDictField(fieldKey string, pairs map[string]IPair) IField {
	if fieldKey == "" {
		fieldKey = hashFieldKey()
	}

	if pairs == nil {
		pairs = make(map[string]IPair, 4)
	}

	return dictField{
		name:  fieldKey,
		pairs: pairs,
	}
}

func (k dictField) MarshalJSON() ([]byte, error) {
	return json.Marshal(k.pairs)
}

func (k dictField) MarshalText() (text []byte, err error) {
	buf := bytes.NewBuffer(nil)
	err = toml.NewEncoder(buf).Encode(k.pairs)

	return buf.Bytes(), err
}

func (k dictField) Name() string {
	return k.name
}

func (k dictField) Type() FieldTyp {
	return DICT_FIELD_
}

func (k dictField) Value() interface{} {
	return k.pairs
}
