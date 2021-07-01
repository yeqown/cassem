package types

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

type VersionedEltDO struct {
	Metadata EltMetadataDO
	Version  int
	Raw      []byte
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
