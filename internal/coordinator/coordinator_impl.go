package coord

import (
	"github.com/yeqown/cassem/internal/persistence"
	"github.com/yeqown/cassem/pkg/datatypes"

	"github.com/pkg/errors"
	"github.com/yeqown/log"
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
	v, err := c.repo.GetContainer(ns, key)
	if err != nil {
		return nil, errors.Wrap(err, "coordinator.GetContainer failed to get container")
	}

	return c.repo.Converter().ToContainer(v)
}

func (c coordinator) PagingContainers(filter *FilterContainersOption) ([]datatypes.IContainer, int, error) {
	outs, count, err := c.repo.PagingContainers(&persistence.PagingContainersFilter{
		Limit:      filter.Limit,
		Offset:     filter.Offset,
		KeyPattern: filter.KeyPattern,
		Namespace:  filter.Namespace,
	})
	if err != nil {
		return nil, 0, errors.Wrap(err, "coordinator.PagingContainers failed to paging pairs")
	}

	containers := make([]datatypes.IContainer, 0, len(outs))
	for _, v := range outs {
		p, err := c.repo.Converter().ToContainer(v)
		if err != nil {
			log.
				WithFields(log.Fields{
					"containerSource": v,
				}).
				Warnf("coordinator.PagingContainers could not convert pair: %v", err)
			continue
		}
		containers = append(containers, p)
	}

	return containers, count, nil
}

func (c coordinator) SaveContainer(container datatypes.IContainer) error {
	v, err := c.repo.Converter().FromContainer(container)
	if err != nil {
		return errors.Wrap(err, "coordinator.SaveContainer failed to convert container")
	}

	return c.repo.SaveContainer(v, true)
}

func (c coordinator) RemoveContainer(key string, ns string) error {
	return c.repo.RemoveContainer(ns, key)
}

// PagingNamespaces list namespaces those conform to the filter(FilterNamespacesOption).
//
// DONE(@yeqown): handle count return value
func (c coordinator) PagingNamespaces(filter *FilterNamespacesOption) ([]string, int, error) {
	ns, count, err := c.repo.PagingNamespace(&persistence.PagingNamespacesFilter{
		Limit:            filter.Limit,
		Offset:           filter.Offset,
		NamespacePattern: filter.NamespacePattern,
	})

	return ns, count, err
}

func (c coordinator) SaveNamespace(ns string) error {
	return c.repo.SaveNamespace(ns)
}

func (c coordinator) GetPair(key, ns string) (datatypes.IPair, error) {
	v, err := c.repo.GetPair(ns, key)
	if err != nil {
		return nil, errors.Wrap(err, "coordinator.GetPair failed to get pair")
	}

	return c.repo.Converter().ToPair(v)
}

func (c coordinator) PagingPairs(filter *FilterPairsOption) ([]datatypes.IPair, int, error) {
	outs, count, err := c.repo.PagingPairs(&persistence.PagingPairsFilter{
		Limit:      filter.Limit,
		Offset:     filter.Offset,
		KeyPattern: filter.KeyPattern,
		Namespace:  filter.Namespace,
	})
	if err != nil {
		return nil, 0, errors.Wrap(err, "coordinator.PagingPairs failed to paging pairs")
	}

	pairs := make([]datatypes.IPair, 0, len(outs))
	for _, v := range outs {
		p, err := c.repo.Converter().ToPair(v)
		if err != nil {
			log.
				WithFields(log.Fields{
					"pairSource": v,
				}).
				Warnf("coordinator.PagingPairs could not convert pair: %v", err)
			continue
		}
		pairs = append(pairs, p)
	}

	return pairs, count, nil
}

func (c coordinator) SavePair(p datatypes.IPair) error {
	v, err := c.repo.Converter().FromPair(p)
	if err != nil {
		return errors.Wrap(err, "coordinator.SavePair failed to convert pair")
	}

	return c.repo.SavePair(v, true)
}
