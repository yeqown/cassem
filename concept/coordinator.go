package concept

import "context"

// AdmAggregate describes all methods should storage component should support.
type AdmAggregate interface {
	KVReadOnly
	KVWriteOnly
	InstanceHybrid
	AgentHybrid
	RBAC
}

type AgentAggregate interface {
	KVReadOnly
	InstanceHybrid
	AgentHybrid
}

type KVReadOnly interface {
	GetElementWithVersion(ctx context.Context, app, env, key string, version int) (*Element, error)
	GetElementVersions(ctx context.Context, app, env, key string,
		seek string, limit int) (*getElementsResult, error)
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

	RollbackElementVersion(ctx context.Context, app string, env string, key string,
		rollbackVersion uint32) error
	PublishElementVersion(ctx context.Context, app string, env string, key string,
		publishVersion uint32) (*Element, error)

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

// AgentHybrid describes all methods to manage agent nodes in cassemdb.
type AgentHybrid interface {
	// Watch would block util any error happened, otherwise any change of agents will be
	// pushed into ch.
	Watch(ctx context.Context, ch chan<- *AgentInstanceChange) error
	// Register helps agent registers itself.
	Register(ctx context.Context, ins *AgentInstance, ttl int32) error
	// Renew helps agents keep online.
	Renew(ctx context.Context, ins *AgentInstance, ttl int32) error
	// Unregister helps agent unregister itself.
	Unregister(ctx context.Context, agentId string) error
	GetAgents(ctx context.Context, seek string, limit int) (*getAgentsResult, error)
}

// RBAC is an ACL model to implement authentication permission management.
type RBAC interface {
	GetUser(account string) (*User, error)
	AddUser(u *User) error
	DisableUser(account string) error
	AssignRole(account, role string, domain ...string) error
	RevokeRole(account, role string, domain ...string) error
	Enforce(subject, domain, object, act string) (bool, error)
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

// PublishingMode indicates how to publish the element's update.
type PublishingMode uint8

const (
	// PublishMode_GRAY gray publish mode only push to specified instance.
	PublishMode_GRAY PublishingMode = iota + 1
	// PublishMode_FULL full publish mode, push to all instances.
	PublishMode_FULL
)

type getAgentsResult struct {
	commonPager

	Agents []*AgentInstance `json:"agents"`
}
