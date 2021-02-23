package mysql

import (
	"fmt"

	"github.com/yeqown/log"

	"github.com/yeqown/cassem/internal/persistence"

	"github.com/pkg/errors"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

var (
	_pairTbl      = new(PairDO)
	_containerTbl = new(ContainerDO)
	_fieldTbl     = new(FieldDO)
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
	containerDO := new(ContainerDO)

	err := m.db.Model(_containerTbl).
		Preload("Fields").
		Where("`key` = ? AND namespace = ?", containerKey, ns).
		First(containerDO).Error
	if err != nil {
		return nil, err
	}

	pairsKeys := make([]string, 0, 64)
	for _, fld := range containerDO.Fields {
		pairsKeys = append(pairsKeys, fld.Pairs.Keys()...)
	}

	pairsDOs := make([]*PairDO, 0, len(pairsKeys))
	err = m.db.Model(_pairTbl).
		Where("namespace = ? AND `key` in ?", ns, pairsKeys).
		Find(&pairsDOs).Error
	if err != nil {
		return nil, err
	}

	pairsMapping := make(map[string]*PairDO, len(pairsDOs))
	for idx, pair := range pairsDOs {
		pairsMapping[pair.Key] = pairsDOs[idx]
	}

	return &toContainerWithPairs{
		origin: toOriginDetail,
		c:      containerDO,
		pairs:  pairsMapping,
	}, nil
}

func (m mysqlRepo) SaveContainer(c interface{}, update bool) (err error) {
	from, ok := c.(*formContainerParsed)
	if !ok || from == nil {
		return errors.New("invalid value of container")
	}

	// start a transaction
	tx := m.db.Begin()
	defer func() {
		if err == nil {
			tx.Commit()
			return
		}

		log.
			WithFields(log.Fields{
				"error":  err,
				"update": update,
				"input":  c,
			}).
			Debugf("mysqlRepo.SaveContainer failed, now rollback: err=%v", tx.Rollback())
	}()

	if update {
		err = m.updateContainer(tx, from)
	} else {
		err = m.createContainer(tx, from)
	}

	return
}

// updateContainer update or create
func (m mysqlRepo) updateContainer(tx *gorm.DB, from *formContainerParsed) (err error) {
	if from.c == nil {
		return
	}

	if from.c.Namespace == "" || from.c.Key == "" {
		return errors.New("empty container to update")
	}

	// firstOrCreate container
	if err = tx.Model(_containerTbl).
		Omit(clause.Associations).
		Where(from.c).
		FirstOrCreate(from.c).Error; err != nil {
		return
	}

	log.
		WithFields(log.Fields{
			"c":  from.c,
			"id": from.c.ID, // this could not be empty.
		}).
		Debug("mysqlRepo.updateContainer")

	containerId := from.c.ID

	// DONE(@yeqown): drop deleted-fields before
	if err = tx.Model(_fieldTbl).
		Unscoped().
		Where("container_id = ? AND `key` NOT IN (?)", containerId, from.uniqueFieldKeys).
		Delete(nil).Error; err != nil {
		return
	}

	// update fields
	for idx := range from.fields {
		from.fields[idx].ContainerID = containerId
	}
	if err = tx.Model(_fieldTbl).
		Clauses(clause.OnConflict{
			Columns: []clause.Column{
				{Name: "key"},
				{Name: "container_id"},
			},
			// DONE(@yeqown): only update field_type
			DoUpdates: clause.AssignmentColumns([]string{
				"field_type",
				"field_pairs",
			}),
		}).
		CreateInBatches(from.fields, len(from.fields)).Error; err != nil {
		return
	}

	return nil
}

func (m mysqlRepo) createContainer(tx *gorm.DB, from *formContainerParsed) (err error) {
	if from.c == nil {
		return
	}

	if from.c.Namespace == "" || from.c.Key == "" {
		return errors.New("empty container to create")
	}

	// create container
	if err = tx.Model(_containerTbl).
		Omit(clause.Associations).
		Create(from.c).Error; err != nil {
		return
	}

	containerId := from.c.ID
	// create fields
	for idx := range from.fields {
		from.fields[idx].ContainerID = containerId
	}
	if err = tx.Model(_fieldTbl).
		CreateInBatches(from.fields, len(from.fields)).Error; err != nil {
		return
	}

	return nil
}

// PagingContainers do not resolve all data in container, but overview of container to display.
func (m mysqlRepo) PagingContainers(filter *persistence.PagingContainersFilter) ([]interface{}, int, error) {
	if filter == nil || filter.Limit <= 0 || filter.Offset < 0 {
		filter = &persistence.PagingContainersFilter{
			Limit:      10,
			Offset:     0,
			Namespace:  "",
			KeyPattern: "",
		}
	}

	containerDOs := make([]*ContainerDO, 0, filter.Limit)
	tx := m.db.Model(_containerTbl).Preload("Fields")
	if filter.KeyPattern != "" {
		tx = tx.Where("")
	}
	if filter.Namespace != "" {
		tx = tx.Where("")
	}

	count := int64(0)
	err := tx.Order("created_at DESC").
		Count(&count).
		Offset(filter.Offset).
		Limit(filter.Limit).
		Find(&containerDOs).Error

	out := make([]interface{}, len(containerDOs))
	for idx := range containerDOs {
		out[idx] = &toContainerWithPairs{
			origin: toOriginPaging,
			c:      containerDOs[idx],
		}
	}

	return out, int(count), err
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

func (m mysqlRepo) SavePair(v interface{}, update bool) (err error) {
	pairDO, ok := v.(*PairDO)
	if !ok || pairDO == nil {
		return errors.New("invalid value of pair")
	}

	if !update {
		return m.db.Model(pairDO).Create(pairDO).Error
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
