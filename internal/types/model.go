package types

// App is a concept of biz application, who's id must indicate the unique one in config center.
type App struct {
	AppId string
	Name  string

	// Containers map[namespace]Containers
	Containers map[string]Container
}

type Container struct {
	Key       string
	Desc      string
	Namespace string

	Items map[string]Item
}

type ItemFormat uint8

const (
	JSON ItemFormat = iota + 1
	TOML
	TXT
	PROPERTIES
)

type Scope uint8

const (
	PUBLIC Scope = iota + 1
	PRIVATE
)

// Pair describes the key value pair under one App-Env-NS
type Pair struct {
	Key         string
	Value       []byte
	Desc        string
	Scope       Scope
	Version     string
	Format      ItemFormat
	IsPublished bool
}

type PairChangeLog struct{}
