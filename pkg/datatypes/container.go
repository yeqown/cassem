package datatypes

import (
	"bytes"
	"encoding/json"
	"sync"

	"github.com/yeqown/cassem/pkg/hash"

	// DONE(@yeqown) use recommended toml library.
	"github.com/BurntSushi/toml"
)

var (
	_ IContainer = &builtinLogicContainer{}
	_ IEncoder   = &builtinLogicContainer{}
)

// IContainer helps logic and repository to operates with Container which contains fields of pair.
type IContainer interface {
	IEncoder

	// Key of IContainer to identify which container in cassem.
	Key() string

	// NS of IContainer indicates to which namespace in cassem the IContainer belongs.
	NS() string

	// GetField get one field in IContainer
	GetField(fieldKey string) (bool, IField)

	// SetField add one pair into IContainer, return (evicted) bool and (err) error,
	// ok means fld is duplicated in IContainer, err means there got into an exception.
	SetField(fld IField) (bool, error)

	// RemoveField delete pair from IContainer
	RemoveField(fieldKey string) (bool, error)

	// Fields list all field in IContainer
	Fields() []IField

	// CheckSum set or calc checksum of IContainer
	CheckSum(sum string) string
}

type builtinLogicContainer struct {
	// uniqueKey identify the builtinLogicContainer in one namespace
	uniqueKey string

	// checksum of builtinLogicContainer
	checksum string

	// namespace indicates to which namespace the builtinLogicContainer belongs.
	namespace string

	// DONE(@yeqown) how to contains list and dictionary?
	// by abstract layer named field(KV, LIST, DICT)
	// FIXME(@yeqown): is there necessary to lock?
	_mu sync.RWMutex
	// fields means map[IField.Name()]IField
	fields map[string]IField
}

// NewContainer to construct a logic container.
func NewContainer(ns, key string) IContainer {
	return &builtinLogicContainer{
		uniqueKey: key,
		namespace: ns,
		_mu:       sync.RWMutex{},
		fields:    make(map[string]IField, 4),
	}
}

func (c *builtinLogicContainer) Key() string {
	return c.uniqueKey
}

func (c *builtinLogicContainer) NS() string {
	return c.namespace
}

func (c *builtinLogicContainer) SetField(fld IField) (bool, error) {
	if fld == nil || fld.Name() == "" {
		return false, ErrInvalidField
	}

	c._mu.Lock()
	defer c._mu.Unlock()

	_, ok := c.fields[fld.Name()]
	c.fields[fld.Name()] = fld
	return ok, nil
}

func (c *builtinLogicContainer) RemoveField(fieldKey string) (bool, error) {
	c._mu.Lock()
	defer c._mu.Unlock()

	_, ok := c.fields[fieldKey]
	if ok {
		delete(c.fields, fieldKey)
	}

	return ok, nil
}

func (c *builtinLogicContainer) Fields() []IField {
	c._mu.RLock()
	defer c._mu.RUnlock()

	fields := make([]IField, 0, len(c.fields))
	for k := range c.fields {
		fields = append(fields, c.fields[k])
	}

	return fields
}

func (c *builtinLogicContainer) GetField(fieldKey string) (bool, IField) {
	c._mu.RLock()
	defer c._mu.RUnlock()

	v, ok := c.fields[fieldKey]
	return ok, v
}

func (c *builtinLogicContainer) MarshalJSON() ([]byte, error) {
	c._mu.RLock()
	defer c._mu.RUnlock()

	return json.Marshal(c.fields)
}

func (c *builtinLogicContainer) ToTOML() ([]byte, error) {
	c._mu.RLock()
	defer c._mu.RUnlock()

	buf := bytes.NewBuffer(nil)
	err := toml.NewEncoder(buf).Encode(c.fields)

	return buf.Bytes(), err
}

// CheckSum set or calculate checksum of builtinLogicContainer.
// DONE(@yeqown): get content of container and calculate checksum
func (c *builtinLogicContainer) CheckSum(sum string) string {
	if len(c.checksum) != 0 {
		return c.checksum
	}

	if len(sum) != 0 {
		c.checksum = sum
		return c.checksum
	}

	// both c.checksum and sum is empty, then need to calculate checksum
	content, _ := json.Marshal(c)
	c.checksum = hash.CheckSum(content)
	return c.checksum
}
