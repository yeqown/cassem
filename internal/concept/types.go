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

// Instance describes a client connect to agent, this information is saved into cassemdb and displayed on cassemadm
// dashboard, helps cassemadm to achieve config push ability.
type Instance struct {
	// ClientID was a unique ID in cassem which can be set by client SDK. A random string merges client IP will be
	// used while client SDK doesn't set it.
	ClientID          string
	Ip                string
	AppId             string
	Env               string
	WatchKeys         []string
	LastJoinTimestamp int64
	LastGetTimestamp  int64
}

func (i Instance) Id() string {
	if i.ClientID == "" {
		return "cassem" + "@" + i.Ip
	}

	return i.ClientID + "@" + i.Ip
}

func (i *Instance) Unmarshal(bytes []byte) error {
	return json.Unmarshal(bytes, i)
}

func (i *Instance) Marshal() ([]byte, error) {
	return json.Marshal(i)
}
