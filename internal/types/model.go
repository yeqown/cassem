package types

type App struct {
	AppId string

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

type Item struct {
	Key     string
	Desc    string
	Version string
	Format  ItemFormat
}
