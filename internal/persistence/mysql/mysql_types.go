package mysql

import (
	"github.com/yeqown/cassem/pkg/datatypes"

	"gorm.io/gorm"
)

// PairDO contains the basic datatype, it represent the minimum cell in cassem.
type PairDO struct {
	gorm.Model

	Key       string             `gorm:"column:key;type:varchar(64);uniqueIndex:idx_unique_pair"`
	Namespace string             `gorm:"column:namespace;type:varchar(64);uniqueIndex:idx_unique_pair"`
	Datatype  datatypes.Datatype `gorm:"column:datatype;type:tinyint(3)"`
	Value     []byte             `gorm:"column:value;type:blob"`
}

func (m PairDO) TableName() string { return "cassem_pairs" }

// XXXFieldToPairDO 1 field to 1+ pairs related to to field type

// KVFieldToPairDO (kv: 1 <> 1)
type KVFieldToPairDO struct {
	gorm.Model

	ContainerID uint   `gorm:"column:container_id;type:bigint;uniqueIndex:idx_unique_kv_field"`
	FieldKey    string `gorm:"column:field_key;type:varchar(64);uniqueIndex:idx_unique_kv_field"`
	PairKey     string `gorm:"column:pair_key;type:varchar(64)"`

	Pair  PairDO  `gorm:""` // TODO(@yeqown)
	Field FieldDO `gorm:""` // TODO(@yeqown)
}

func (m KVFieldToPairDO) TableName() string { return "cassem_field_kv" }

// ListFieldToPairDO (list: 1 <> 1+)
type ListFieldToPairDO struct {
	gorm.Model

	ContainerID uint   `gorm:"column:container_id;type:bigint;uniqueIndex:idx_unique_list_field"`
	FieldKey    string `gorm:"column:field_key;type:varchar(64);uniqueIndex:idx_unique_list_field"`
	PairKey     string `gorm:"column:pair_key;type:varchar(64)"`

	Pair  PairDO  `gorm:""` // TODO(@yeqown)
	Field FieldDO `gorm:""` // TODO(@yeqown)
}

func (m ListFieldToPairDO) TableName() string { return "cassem_field_list" }

// DictFieldToPairDO (dict: 1 <> 1+)
// DONE(@yeqown) think about the unique index of DictFieldToPairDO.
type DictFieldToPairDO struct {
	gorm.Model

	ContainerID  uint   `gorm:"column:container_id;type:bigint;uniqueIndex:idx_unique_dict_field"`
	FieldKey     string `gorm:"column:field_key;type:varchar(64);uniqueIndex:idx_unique_dict_field"`
	DictFieldKey string `gorm:"column:dict_field_key;type:varchar(64);uniqueIndex:idx_unique_dict_field"`
	PairKey      string `gorm:"column:pair_key;type:varchar(64)"`

	Pair  PairDO  `gorm:""` // TODO(@yeqown)
	Field FieldDO `gorm:""` // TODO(@yeqown)
}

func (m DictFieldToPairDO) TableName() string { return "cassem_field_dict" }

type FieldDO struct {
	gorm.Model

	FieldType   datatypes.FieldTyp `gorm:"column:field_type;type:tinyint(8)"`
	Key         string             `gorm:"column:key;type:varchar(64);uniqueIndex:idx_unique_field"`
	ContainerID uint               `gorm:"column:container_id;type:bigint;uniqueIndex:idx_unique_field"`
	//ContainerKey string             `gorm:"column:container_key;type:varchar(64);uniqueIndex:idx_unique_field"`
	//Namespace    string             `gorm:"column:namespace;type:varchar(64);uniqueIndex:idx_unique_field"`
}

func (m FieldDO) TableName() string { return "cassem_field" }

type ContainerDO struct {
	gorm.Model

	Key       string `gorm:"column:key;type:varchar(64);uniqueIndex:idx_unique_field"`
	Namespace string `gorm:"column:namespace;type:varchar(64);uniqueIndex:idx_unique_field"`
	CheckSum  string `gorm:"column:checksum;type:varchar(128);"`

	Fields []*FieldDO
}

func (m ContainerDO) TableName() string { return "cassem_container" }

type NamespaceDO struct {
	gorm.Model

	Namespace string `gorm:"column:namespace;type:varchar(64);uniqueIndex:idx_unique_ns"`
}

func (m NamespaceDO) TableName() string { return "cassem_ns" }
