package coord

import (
	"github.com/yeqown/cassem/internal/persistence"
	"github.com/yeqown/cassem/pkg/datatypes"
)

type coordinator struct {
	repo persistence.Repository
}

func New(repo persistence.Repository) ICoordinator {
	return coordinator{
		repo: repo,
	}
}

func (c coordinator) GetContainer(key, ns string) (datatypes.IContainer, error) {
	panic("implement me")
}

func (c coordinator) AllContainers(filter *FilterContainersOption) ([]*datatypes.IContainer, int, error) {
	panic("implement me")
}

func (c coordinator) SaveContainer(container datatypes.IContainer) error {
	panic("implement me")
}

func (c coordinator) AllNamespaces(filter *FilterNamespacesOption) ([]string, int, error) {
	panic("implement me")
}

func (c coordinator) SaveNamespace(ns string) error {
	panic("implement me")
}

func (c coordinator) GetPair(key, ns string) (datatypes.IPair, error) {
	panic("implement me")
}

func (c coordinator) AllPairs(filter *FilterPairsOption) ([]datatypes.IPair, int, error) {
	panic("implement me")
}

func (c coordinator) SavePair(p datatypes.IPair) error {
	panic("implement me")
}
