package mysql

import (
	"database/sql/driver"
	"encoding/json"
	"fmt"

	"github.com/yeqown/cassem/pkg/datatypes"
	"github.com/yeqown/cassem/pkg/set"

	"github.com/pkg/errors"
	"gorm.io/gorm"
)

// pairDO contains the basic datatype, it represent the minimum cell in cassem.
type pairDO struct {
	gorm.Model

	Key       string             `gorm:"column:key;type:varchar(64);uniqueIndex:idx_unique_pair,priority:1;index:idx_ns_to_key,priority:2"`
	Namespace string             `gorm:"column:namespace;type:varchar(32);uniqueIndex:idx_unique_pair,priority:2;index:idx_ns_to_key,priority:1"`
	Datatype  datatypes.Datatype `gorm:"column:datatype;type:tinyint(3)"`
	Value     []byte             `gorm:"column:value;type:blob"`
}

func (m pairDO) TableName() string { return "cassem_pairs" }

// fieldPairs contains all pairs of fieldDO.
// KV_FIELD_ contains like: {"KV": "pairKey"}, "KV" is a const mark of KV field.
// LIST_FIELD_ contains like: {"0": "pairKey", "1": "pairKey"}, the `key` of fieldPairs is index of pairKey.
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

func (f fieldPairs) Value() (driver.Value, error) {
	return json.Marshal(f)
}

func (f *fieldPairs) Scan(src interface{}) error {
	byts, ok := src.([]byte)
	if !ok {
		return errors.New(fmt.Sprint("Failed to unmarshal(JSON) from src:", src))
	}

	err := json.Unmarshal(byts, f)
	return err
}

type fieldDO struct {
	gorm.Model

	FieldType   datatypes.FieldTyp `gorm:"column:field_type;type:tinyint(8)"`
	Key         string             `gorm:"column:key;type:varchar(64);uniqueIndex:idx_unique_field,priority:2"`
	ContainerID uint               `gorm:"column:container_id;type:bigint;uniqueIndex:idx_unique_field,priority:1"`

	// Pairs contains all pair keys of fieldDO.
	//
	// KV_FIELD_ contains like: {"KV": "pairKey"}, "KV" is a const mark of KV field.
	// LIST_FIELD_ contains like: {"0": "pairKey", "1": "pairKey"}, the `key` of fieldPairs is index of pairKey.
	// DICT_FIELD_ contains like: {"dictKey": "pairKey"}
	Pairs fieldPairs `gorm:"column:field_pairs;type:blob"`
}

func (m fieldDO) TableName() string { return "cassem_field" }

type containerDO struct {
	gorm.Model

	Key       string `gorm:"column:key;type:varchar(64);uniqueIndex:idx_unique_field,priority:1"`
	Namespace string `gorm:"column:namespace;type:varchar(32);uniqueIndex:idx_unique_field,priority:2"`
	CheckSum  string `gorm:"column:checksum;type:varchar(128);"`

	Fields []*fieldDO `gorm:"foreignKey:ContainerID"`
}

func (m containerDO) TableName() string { return "cassem_container" }

type NamespaceDO struct {
	gorm.Model

	Namespace string `gorm:"column:namespace;type:varchar(32);uniqueIndex:idx_unique_ns"`
}

func (m NamespaceDO) TableName() string { return "cassem_ns" }

type formContainerParsed struct {
	c               *containerDO
	fields          []*fieldDO
	uniqueFieldKeys set.StringSet
	uniquePairKeys  set.StringSet
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
	gorm.Model

	Account          string `gorm:"column:account;type:varchar(32);uniqueIndex:idx_unique_account"`
	PasswordWithSalt string `gorm:"column:password;type:varchar(64)"`
	Name             string `gorm:"column:name;type:varchar(16)"`
}

func (m userDO) TableName() string {
	return "cassem_user"
}
