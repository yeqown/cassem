package datatypes

import "encoding/json"

type FieldTyp uint8

const (
	KV_FIELD_ FieldTyp = iota + 1
	LIST_FIELD_
	DICT_FIELD_
)

type IField interface {
	json.Marshaler

	Name() string

	Type() FieldTyp

	Value() interface{}
}

var (
	_ IField = kvField{}
)

type kvField struct {
	name string

	kv IPair
}

func (k kvField) MarshalJSON() ([]byte, error) {
	return json.Marshal(k.kv)
}

func NewKVField(fieldKey string, p IPair) IField {
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

var (
	_ IField = listField{}
)

type listField struct {
	name string

	pairs []IPair
}

func NewListField(fieldKey string, pairs []IPair) IField {
	if pairs == nil {
		pairs = make([]IPair, 4)
	}

	return listField{
		name:  fieldKey,
		pairs: pairs,
	}
}

func (k listField) MarshalJSON() ([]byte, error) {
	return json.Marshal(k.pairs)
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

var (
	_ IField = dictField{}
)

type dictField struct {
	name string

	pairs map[string]IPair
}

func NewDictField(fieldKey string, pairs map[string]IPair) IField {
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

func (k dictField) Name() string {
	return k.name
}

func (k dictField) Type() FieldTyp {
	return DICT_FIELD_
}

func (k dictField) Value() interface{} {
	return k.pairs
}
