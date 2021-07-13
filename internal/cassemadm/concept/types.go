package concept

import "encoding/json"

//go:generate stringer -type=EltContentType
type RawContentType string

const (
	RawContentType_JSON      = "application/json"
	RawContentType_TOML      = "application/toml"
	RawContentType_INI       = "application/ini"
	RawContentType_PLAINTEXT = "application/plaintext"
)

type EltMetadataDO struct {
	LatestVersion     int            `json:"latest_version"`
	LatestFingerprint string         `json:"latest_fingerprint"`
	Key               string         `json:"key"`
	ContentType       RawContentType `json:"content_type"`
	App               string         `json:"app"`
	Env               string         `json:"env"`
}

func (e *EltMetadataDO) Unmarshal(bytes []byte) error {
	return json.Unmarshal(bytes, e)
}

func (e EltMetadataDO) Marshal() ([]byte, error) {
	return json.Marshal(e)
}

type VersionedEltDO struct {
	Metadata *EltMetadataDO `json:"metadata"`
	Version  int            `json:"version"`
	Raw      []byte         `json:"raw"`
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

//type EnvMetadataDO struct {
//	Id          string `json:"id"`
//	Name        string `json:"name"`
//	Description string `json:"description"`
//}

type AppMetadataDO struct {
	Id          string `json:"id"`
	Name        string `json:"name"`
	Description string `json:"description"`
}

func (e *AppMetadataDO) Unmarshal(bytes []byte) error {
	return json.Unmarshal(bytes, e)
}

func (e AppMetadataDO) Marshal() ([]byte, error) {
	return json.Marshal(e)
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
