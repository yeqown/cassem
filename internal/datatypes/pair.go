package datatypes

import "encoding/json"

type IPair interface {
	// NS
	NS() string

	// Key
	Key() string

	// Value() IData
	Value() IData
}

// builtinPair include
type builtinPair struct {
	// namespace indicates pair would only be used in the same namespace file
	// container, and also be unique in one namespace.
	namespace string

	// key is the unique string in one namespace, usually be used to identify the builtinPair.
	key string

	// value contains basic data type
	value IData
}

func (p builtinPair) UnmarshalJSON(data []byte) error {
	return json.Unmarshal(data, p.value)
}

func (p builtinPair) MarshalJSON() ([]byte, error) {
	return json.Marshal(p.value)
}

func NewPair(ns, key string, value IData) IPair {
	return &builtinPair{
		namespace: ns,
		key:       key,
		value:     value,
	}
}

func (p builtinPair) NS() string {
	return p.namespace
}

func (p builtinPair) Key() string {
	return p.key
}

func (p builtinPair) Value() IData {
	return p.value
}
