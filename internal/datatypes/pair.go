package datatypes

type IPair interface {
	// NS
	NS() string

	// Key
	Key() string

	// Value() IData
	Value() IData
}

// Pair include
type Pair struct {
	// namespace indicates pair would only be used in the same namespace file
	// container, and also be unique in one namespace.
	namespace string

	// key is the unique string in one namespace, usually be used to identify the Pair.
	key string

	// value contains basic data type
	value IData
}

func NewPair(ns, key string, value IData) IPair {
	return &Pair{
		namespace: ns,
		key:       key,
		value:     value,
	}
}

func (p Pair) NS() string {
	return p.namespace
}

func (p Pair) Key() string {
	return p.key
}

func (p Pair) Value() IData {
	return p.value
}
