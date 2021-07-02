package types

import "encoding/json"

//go:generate stringer -type=EltContentType
type EltContentType uint8

const (
	EltContentType_JSON = iota + 1
	EltContentType_TOML
	EltContentType_PLAINTEXT
)

type EltMetadataDO struct {
	LatestVersion     int
	LatestFingerprint string
	Key               string
	ContentType       EltContentType
	App               string
	Env               string
}

func (e *EltMetadataDO) Unmarshal(bytes []byte) error {
	return json.Unmarshal(bytes, e)
}

func (e EltMetadataDO) Marshal() ([]byte, error) {
	return json.Marshal(e)
}

type VersionedEltDO struct {
	Metadata *EltMetadataDO `json:"-"`
	Version  int
	Raw      []byte
}

func (e *VersionedEltDO) Unmarshal(bytes []byte) error {
	return json.Unmarshal(bytes, e)
}

func (e VersionedEltDO) Marshal() ([]byte, error) {
	return json.Marshal(e)
}

//type Elt struct {
//	Metadata EltMetadataDO
//}

type EnvMetadataDO struct {
	Id          string
	Name        string
	Description string
}

type AppMetadataDO struct {
	Id          string
	Name        string
	Description string
}

//type Env struct {
//	Metadata EnvMetadataDO
//	Elements []Elt
//}
//
//type App struct {
//	Id  string
//	Env []Env
//}

// Operation indicates the element operation in cassemadm.
//go:generate stringer -type=Operation
type Operation uint8

const (
	Operation_SET Operation = iota + 1
	Operation_UNSET
	Operation_PUBLISH
)

// EltOperateLog element operation log entry.
type EltOperateLog struct {
	Operator      string
	OperatedAt    int64
	OperatedKey   string
	Op            Operation
	BeforeVersion int
	AfterVersion  int
	Remark        string
}
