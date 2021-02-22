package mysql

import (
	"fmt"

	"github.com/yeqown/cassem/internal/persistence"

	"github.com/pkg/errors"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

var (
	_pairTbl      = new(PairDO)
	_containerTbl = new(ContainerDO)
	_nsTbl        = new(NamespaceDO)
)

type mysqlRepo struct {
	db *gorm.DB

	converter *mysqlConverter
}

func New(db *gorm.DB) persistence.Repository {
	return mysqlRepo{
		db: db,
	}
}

func (m mysqlRepo) GetContainer(ns, containerKey string) (interface{}, error) {
	panic("implement me")
}

func (m mysqlRepo) SaveContainer(c interface{}, isUpdate bool) error {
	panic("implement me")
}

func (m mysqlRepo) PagingContainers(filter *persistence.PagingContainersFilter) ([]interface{}, int, error) {
	panic("implement me")
}

func (m mysqlRepo) GetPair(ns, key string) (interface{}, error) {
	pairDO := PairDO{
		Namespace: ns,
		Key:       key,
	}

	if err := m.db.
		Model(pairDO).
		First(&pairDO, "`key` = ? AND namespace = ?", key, ns).
		Error; err != nil {

		return nil, err
	}

	return &pairDO, nil
}

func (m mysqlRepo) SavePair(v interface{}, isUpdate bool) (err error) {
	pairDO, ok := v.(*PairDO)
	if !ok || pairDO == nil {
		return errors.New("invalid value of pair")
	}

	if !isUpdate {
		err = m.db.Model(pairDO).Create(pairDO).Error
		return
	}

	// update
	err = m.db.Model(pairDO).
		Clauses(clause.OnConflict{
			Columns: []clause.Column{
				{Name: "key"},
				{Name: "namespace"},
			},
			DoUpdates: clause.Set{
				clause.Assignment{
					Column: clause.Column{Name: "value"},
					Value:  pairDO.Value,
				},
				clause.Assignment{
					Column: clause.Column{Name: "datatype"},
					Value:  pairDO.Datatype,
				},
			},
		}).
		Create(pairDO).Error
	return
}

func (m mysqlRepo) PagingPairs(filter *persistence.PagingPairsFilter) ([]interface{}, int, error) {
	if filter == nil {
		filter = &persistence.PagingPairsFilter{
			Limit:      10,
			Offset:     0,
			KeyPattern: "",
			Namespace:  "",
		}
	}

	tx := m.db.Model(_pairTbl)
	if filter.KeyPattern != "" {
		tx = tx.Where("key LIKE ?", fmt.Sprintf("%%%s%%", filter.KeyPattern))
	}
	if filter.Namespace != "" {
		tx = tx.Where("namespace = ?", filter.Namespace)
	}

	count := int64(0)
	pairs := make([]*PairDO, 0, filter.Limit)
	err := tx.
		Order("created_at DESC").
		Count(&count).
		Offset(filter.Offset).
		Limit(filter.Limit).
		Find(&pairs).
		Error

	out := make([]interface{}, len(pairs))
	if len(pairs) != 0 {
		for idx := range pairs {
			out[idx] = pairs[idx]
		}
	}

	return out, int(count), err
}

func (m mysqlRepo) PagingNamespace(filter *persistence.PagingNamespacesFilter) ([]string, error) {
	if filter == nil || filter.Limit <= 0 || filter.Offset < 0 {
		filter = &persistence.PagingNamespacesFilter{
			Limit:            999,
			Offset:           0,
			NamespacePattern: "",
		}
	}

	tx := m.db.Model(_nsTbl)
	if filter.NamespacePattern != "" {
		tx = tx.Where("namespace LIKE ?", fmt.Sprintf("%%%s%%", filter.NamespacePattern))
	}

	out := make([]string, 0, 10)
	err := tx.
		Order("namespace ASC").
		Offset(filter.Offset).
		Limit(filter.Limit).
		Pluck("namespace", &out).
		Error

	return out, err
}

func (m mysqlRepo) SaveNamespace(ns string) error {
	if ns == "" {
		return errors.New("namespace could not be empty")
	}

	nsDO := NamespaceDO{
		Namespace: ns,
	}

	return m.db.Model(nsDO).
		Create(&nsDO).
		Error
}

func (m mysqlRepo) Converter() persistence.Converter {
	if m.converter == nil {
		(&m).converter = newConverter()
	}

	return m.converter
}
