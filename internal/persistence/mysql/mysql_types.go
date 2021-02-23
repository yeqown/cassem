package mysql

import (
	"database/sql/driver"
	"encoding/json"
	"fmt"

	"github.com/pkg/errors"
	"github.com/yeqown/cassem/pkg/datatypes"
	"gorm.io/gorm"
)

// PairDO contains the basic datatype, it represent the minimum cell in cassem.
type PairDO struct {
	gorm.Model

	Key       string             `gorm:"column:key;type:varchar(64);uniqueIndex:idx_unique_pair,priority:1"`
	Namespace string             `gorm:"column:namespace;type:varchar(64);uniqueIndex:idx_unique_pair,priority:2"`
	Datatype  datatypes.Datatype `gorm:"column:datatype;type:tinyint(3)"`
	Value     []byte             `gorm:"column:value;type:blob"`
}

func (m PairDO) TableName() string { return "cassem_pairs" }

// XXXFieldToPairDO 1 field to 1+ pairs related to to field type
//
//// KVFieldToPairDO (kv: 1 <> 1)
//type KVFieldToPairDO struct {
//	gorm.Model
//
//	ContainerID uint   `gorm:"column:container_id;type:bigint;uniqueIndex:idx_unique_kv_field,priority:1"`
//	FieldKey    string `gorm:"column:field_key;type:varchar(64);uniqueIndex:idx_unique_kv_field,priority:2"`
//	PairKey     string `gorm:"column:pair_key;type:varchar(64)"`
//
//	Pair  PairDO  `gorm:""` // TODO(@yeqown)
//	Field FieldDO `gorm:""` // TODO(@yeqown)
//}
//
//func (m KVFieldToPairDO) TableName() string { return "cassem_field_kv" }
//
//// ListFieldToPairDO (list: 1 <> 1+)
//type ListFieldToPairDO struct {
//	gorm.Model
//
//	ContainerID uint   `gorm:"column:container_id;type:bigint;uniqueIndex:idx_unique_list_field,priority:1"`
//	FieldKey    string `gorm:"column:field_key;type:varchar(64);uniqueIndex:idx_unique_list_field,priority:2"`
//	PairKey     string `gorm:"column:pair_key;type:varchar(64)"`
//
//	Pair  PairDO  `gorm:""` // TODO(@yeqown)
//	Field FieldDO `gorm:""` // TODO(@yeqown)
//}
//
//func (m ListFieldToPairDO) TableName() string { return "cassem_field_list" }
//
//// DictFieldToPairDO (dict: 1 <> 1+)
//// DONE(@yeqown) think about the unique index of DictFieldToPairDO.
//type DictFieldToPairDO struct {
//	gorm.Model
//
//	ContainerID  uint   `gorm:"column:container_id;type:bigint;uniqueIndex:idx_unique_dict_field,priority:1"`
//	FieldKey     string `gorm:"column:field_key;type:varchar(64);uniqueIndex:idx_unique_dict_field,priority:2"`
//	DictFieldKey string `gorm:"column:dict_field_key;type:varchar(64);uniqueIndex:idx_unique_dict_field,priority:3"`
//	PairKey      string `gorm:"column:pair_key;type:varchar(64)"`
//
//	Pair  PairDO  `gorm:""` // TODO(@yeqown)
//	Field FieldDO `gorm:""` // TODO(@yeqown)
//}
//
//func (m DictFieldToPairDO) TableName() string { return "cassem_field_dict" }

type FieldPairs map[string]string

func (f FieldPairs) Keys() []string {
	keys := make([]string, 0, len(f))
	for k := range f {
		keys = append(keys, k)
	}

	return keys
}

func (f FieldPairs) Value() (driver.Value, error) {
	return json.Marshal(f)
}

func (f *FieldPairs) Scan(src interface{}) error {
	byts, ok := src.([]byte)
	if !ok {
		return errors.New(fmt.Sprint("Failed to unmarshal JSONB src:", src))
	}

	err := json.Unmarshal(byts, f)
	return err
}

type FieldDO struct {
	gorm.Model

	FieldType   datatypes.FieldTyp `gorm:"column:field_type;type:tinyint(8)"`
	Key         string             `gorm:"column:key;type:varchar(64);uniqueIndex:idx_unique_field,priority:2"`
	ContainerID uint               `gorm:"column:container_id;type:bigint;uniqueIndex:idx_unique_field,priority:1"`
	Pairs       FieldPairs         `gorm:"column:field_pairs;type:blob"`
}

func (m FieldDO) TableName() string { return "cassem_field" }

type ContainerDO struct {
	gorm.Model

	Key       string `gorm:"column:key;type:varchar(64);uniqueIndex:idx_unique_field,priority:1"`
	Namespace string `gorm:"column:namespace;type:varchar(64);uniqueIndex:idx_unique_field,priority:2"`
	CheckSum  string `gorm:"column:checksum;type:varchar(128);"`

	Fields []*FieldDO `gorm:"foreignKey:ContainerID"`
}

func (m ContainerDO) TableName() string { return "cassem_container" }

type NamespaceDO struct {
	gorm.Model

	Namespace string `gorm:"column:namespace;type:varchar(64);uniqueIndex:idx_unique_ns"`
}

func (m NamespaceDO) TableName() string { return "cassem_ns" }

type formContainerParsed struct {
	c               *ContainerDO
	fields          []*FieldDO
	uniqueFieldKeys []string
}

type toOrigin uint32

const (
	toOriginDetail toOrigin = iota + 1 // detail
	toOriginPaging
)

type toContainerWithPairs struct {
	// origin indicates toContainerWithPairs.paris has value or not.
	// toOriginDetail means no data in pairs, otherwise pairs includes all pairs related to c
	origin toOrigin
	c      *ContainerDO
	pairs  map[string]*PairDO
}
