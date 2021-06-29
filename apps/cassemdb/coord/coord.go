package coord

type ICoordinator interface {
	GetKV(key string) ([]byte, error)
	SetKV(key string, val []byte) error
	UnsetKV(key string) error

	ShouldForwardToLeader() (bool, string)

	RemoveNode(serveId string) error    // RemoveNode
	AddNode(serveId, addr string) error // AddNode
	Apply(data []byte) error            // Apply
}
