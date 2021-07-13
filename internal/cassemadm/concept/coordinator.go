package concept

import "context"

// Hybrid describes all methods should storage component should support.
type Hybrid interface {
	ReadOnly
	WriteOnly
}

type ReadOnly interface {
	GetElementWithVersion(ctx context.Context, app, env, eltKey string, version int) (*VersionedEltDO, error)
	GetElements(ctx context.Context, app, env string, seek string, limit int) (*getElementsResult, error)
	GetElementsByKeys(ctx context.Context, app, env string, eltKeys []string) (*getElementsResult, error)
	GetElementOperations(
		ctx context.Context, app, env, eltKey string, start int) (ops []*EltOperateLog, next int, err error)

	GetApp(ctx context.Context, app string) (*AppMetadataDO, error)
	GetApps(ctx context.Context, seek string, limit int) (*getAppsResult, error)

	GetEnvironments(ctx context.Context, app, seek string, limit int) (*getAppEnvsResult, error)
}

type WriteOnly interface {
	CreateElement(ctx context.Context, app, env, eltKey string, raw []byte, contentTyp RawContentType) error
	UpdateElement(ctx context.Context, app, env, eltKey string, raw []byte) error
	DeleteElement(ctx context.Context, app, env, eltKey string) error

	CreateApp(ctx context.Context, md *AppMetadataDO) error
	DeleteApp(ctx context.Context, appId string) error

	CreateEnvironment(ctx context.Context, md *AppMetadataDO) error
	DeleteEnvironment(ctx context.Context, envId string) error
}

type commonPager struct {
	HasMore  bool   `json:"has_more"`
	NextSeek string `json:"next_seek"`
}

type getAppsResult struct {
	commonPager

	Apps []*AppMetadataDO `json:"apps"`
}

type getAppEnvsResult struct {
	commonPager

	Environments []string `json:"envs"`
}

type getElementsResult struct {
	commonPager

	Elements []*VersionedEltDO `json:"elements"`
}
