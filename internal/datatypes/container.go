package datatypes

import (
	"encoding/json"
	"sync"

	// TODO(@yeqown) use recommended toml library.
	"github.com/pelletier/go-toml"
)

var (
	_ IContainer = &fileContainer{}
	_ IExporter  = &fileContainer{}
)

type IContainer interface {
	Key() string

	NS() string

	// SetField add one pair into IContainer
	SetField(fieldKey string, value IField) (bool, error)

	// RemoveField delete pair from IContainer
	RemoveField(fieldKey string) (bool, error)
}

type fileContainer struct {
	// uniqueKey identify the fileContainer in one namespace
	uniqueKey string

	// namespace indicates to which namespace the fileContainer belongs.
	namespace string

	_mu sync.RWMutex

	// TODO(@yeqown) how to contains list and dictionary?
	fields map[string]IField
}

func (f *fileContainer) Key() string {
	return f.uniqueKey
}

func (f *fileContainer) NS() string {
	return f.namespace
}

func (f *fileContainer) SetField(fieldKey string, pair IField) (bool, error) {
	f._mu.Lock()
	defer f._mu.Unlock()

	_, ok := f.fields[fieldKey]
	f.fields[fieldKey] = pair
	return ok, nil
}

func (f *fileContainer) RemoveField(fieldKey string) (bool, error) {
	f._mu.Lock()
	defer f._mu.Unlock()

	_, ok := f.fields[fieldKey]
	if ok {
		delete(f.fields, fieldKey)
	}

	return ok, nil
}

func (f *fileContainer) ToJSON() ([]byte, error) {
	f._mu.RLock()
	defer f._mu.RUnlock()

	return json.Marshal(f.fields)
}

func (f *fileContainer) ToTOML() ([]byte, error) {
	f._mu.RLock()
	defer f._mu.RUnlock()

	return toml.Marshal(f.fields)
}
