package concept

import "context"

// AdmAggregate describes all methods should storage component should support.
type AdmAggregate interface {
	KVReadOnly
	KVWriteOnly
	InstanceHybrid
}

type AgentAggregate interface {
	KVReadOnly
	InstanceHybrid
}

type KVReadOnly interface {
	GetElementWithVersion(ctx context.Context, app, env, key string, version int) (*Element, error)
	GetElementVersions(ctx context.Context, app, env, key string, seek string, limit int) (*getElementsResult, error)
	GetElements(ctx context.Context, app, env string, seek string, limit int) (*getElementsResult, error)
	GetElementsByKeys(ctx context.Context, app, env string, keys []string) (*getElementsResult, error)
	GetElementOperations(
		ctx context.Context, app, env, key string, start int) (ops []*ElementOperation, next int, err error)

	GetApp(ctx context.Context, app string) (*AppMetadata, error)
	GetApps(ctx context.Context, seek string, limit int) (*getAppsResult, error)

	GetEnvironments(ctx context.Context, app, seek string, limit int) (*getAppEnvsResult, error)
}

type KVWriteOnly interface {
	CreateElement(ctx context.Context, app, env, key string, raw []byte, contentTyp ContentType) error
	UpdateElement(ctx context.Context, app, env, key string, raw []byte) error
	DeleteElement(ctx context.Context, app, env, key string) error

	CreateApp(ctx context.Context, md *AppMetadata) error
	DeleteApp(ctx context.Context, appId string) error
}

// InstanceHybrid describes all methods to manages instance information.
type InstanceHybrid interface {
	// GetElementInstances get all instance those watching this app/env/key.
	GetElementInstances(ctx context.Context, app, env, key string) ([]*Instance, error)
	// GetInstance describes instance detail by insId.
	GetInstance(ctx context.Context, insId string) (*Instance, error)

	RegisterInstance(ctx context.Context, ins *Instance) error
	RenewInstance(ctx context.Context, ins *Instance) error
	UnregisterInstance(ctx context.Context, insId string) error
}

type commonPager struct {
	HasMore  bool   `json:"hasMore"`
	NextSeek string `json:"nextSeek"`
}

type getAppsResult struct {
	commonPager

	Apps []*AppMetadata `json:"apps"`
}

type getAppEnvsResult struct {
	commonPager

	Environments []string `json:"envs"`
}

type getElementsResult struct {
	commonPager

	Elements []*Element `json:"elements"`
}
