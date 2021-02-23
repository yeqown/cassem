package mysql

import (
	"encoding/json"
	"strconv"

	"github.com/yeqown/log"

	"github.com/pkg/errors"

	"github.com/yeqown/cassem/pkg/datatypes"
)

var (
	ErrNilPair            = errors.New("nil pair")
	ErrInvalidPairDO      = errors.New("invalid pair DO data")
	ErrUnknownDatatype    = errors.New("unknown datatype")
	ErrNilContainer       = errors.New("nil container")
	ErrInvalidContainerDO = errors.New("invalid container DO data")
)

type mysqlConverter struct{}

func newConverter() *mysqlConverter {
	return &mysqlConverter{}
}

// FromPair from pair to PairDO
func (m mysqlConverter) FromPair(p datatypes.IPair) (interface{}, error) {
	if p == nil {
		return nil, ErrNilPair
	}

	v, err := p.MarshalJSON()
	if err != nil {
		return nil, errors.Wrap(err, "mysqlConverter.FromPair failed")
	}

	pairDO := PairDO{
		Key:       p.Key(),
		Namespace: p.NS(),
		Datatype:  p.Value().Datatype(),
		Value:     v,
	}

	return &pairDO, nil
}

func (m mysqlConverter) ToPair(v interface{}) (p datatypes.IPair, err error) {
	pairDO, ok := v.(*PairDO)
	if !ok || pairDO == nil {
		return nil, ErrInvalidPairDO
	}

	var (
		d datatypes.IData
	)
	switch pairDO.Datatype {
	case datatypes.BOOL_DATATYPE_:
		var b bool
		if err = json.Unmarshal(pairDO.Value, &b); err == nil {
			d = datatypes.WithBool(b)
		}
	case datatypes.INT_DATATYPE_:
		var b int
		if err = json.Unmarshal(pairDO.Value, &b); err == nil {
			d = datatypes.WithInt(b)
		}
	case datatypes.STRING_DATATYPE_:
		var b string
		if err = json.Unmarshal(pairDO.Value, &b); err == nil {
			d = datatypes.WithString(b)
		}
	case datatypes.FLOAT_DATATYPE_:
		var b float64
		if err = json.Unmarshal(pairDO.Value, &b); err == nil {
			d = datatypes.WithFloat(b)
		}
	case datatypes.LIST_DATATYPE_:
		var ll []interface{}
		d = datatypes.WithList()
		if err = json.Unmarshal(pairDO.Value, &ll); err == nil {
			datatypes.ConstructListRecursive(ll)
		}
	case datatypes.DICT_DATATYPE_:
		d = datatypes.WithDict()
		dd := make(map[string]interface{})
		if err = json.Unmarshal(pairDO.Value, &dd); err == nil {
			d = datatypes.ConstructDictRecursive(dd)
		}
	default:
		err = ErrUnknownDatatype
	}

	if err != nil {
		return
	}
	p = datatypes.NewPair(pairDO.Namespace, pairDO.Key, d)

	return
}

func (m mysqlConverter) FromContainer(c datatypes.IContainer) (interface{}, error) {
	if c == nil {
		return nil, ErrNilContainer
	}

	_fields := c.Fields()
	parsed := formContainerParsed{
		c: &ContainerDO{
			Key:       c.Key(),
			Namespace: c.NS(),
			CheckSum:  "", // TODO(@yeqown) add check sum
		},
		fields:          make([]*FieldDO, 0, len(_fields)),
		uniqueFieldKeys: make([]string, 0, len(_fields)),
	}

	for _, fld := range _fields {
		fieldPairs := make(FieldPairs, 16)
		// mapping field to pairs, so repository could query or update
		switch fld.Type() {
		case datatypes.KV_FIELD_:
			pairKey := fld.Value().(datatypes.IPair).Key()
			fieldPairs[pairKey] = "kv"
		case datatypes.LIST_FIELD_:
			// FIXED(@yeqown) list may have duplicated paris. so how to keep the origin detail.
			for idx, v := range fld.Value().([]datatypes.IPair) {
				fieldPairs[strconv.Itoa(idx)] = v.Key()
			}
		case datatypes.DICT_FIELD_:
			for k, v := range fld.Value().(map[string]datatypes.IPair) {
				fieldPairs[k] = v.Key()
			}
		default:
			log.
				WithField("fld", fld).
				Warn("invalid field type: %d", fld.Type())
		}

		parsed.uniqueFieldKeys = append(parsed.uniqueFieldKeys, fld.Name())
		parsed.fields = append(parsed.fields, &FieldDO{
			FieldType: fld.Type(),
			Key:       fld.Name(),
			Pairs:     fieldPairs,
		})
	}

	return &parsed, nil
}

func (m mysqlConverter) ToContainer(v interface{}) (datatypes.IContainer, error) {
	toc, ok := v.(*toContainerWithPairs)
	if !ok || toc == nil || toc.c == nil {
		return nil, ErrInvalidContainerDO
	}

	var (
		//shouldParsePair   = false
		parsedPairMapping map[string]datatypes.IPair
		err               error
	)
	if toc.origin == toOriginDetail {
		// toc.paris is not empty, so need to parse and mapping
		//shouldParsePair = true
		parsedPairMapping = make(map[string]datatypes.IPair, len(toc.pairs))
		for k, v := range toc.pairs {
			// parse pair and save into mapping
			if parsedPairMapping[k], err = m.ToPair(v); err != nil {
				log.WithFields(log.Fields{
					"k":         k,
					"container": toc.c,
				}).Warnf("mysqlConverter.ToContainer failed to convert pair: %v", err)
			}
		}
	}

	// DONE(@yeqown): add pair to field
	c := datatypes.NewContainer(toc.c.Namespace, toc.c.Key)
	for _, fld := range toc.c.Fields {
		var f datatypes.IField
		switch fld.FieldType {
		case datatypes.KV_FIELD_:
			var pair datatypes.IPair
			for k := range fld.Pairs {
				pair = parsedPairMapping[k]
			}
			f = datatypes.NewKVField(fld.Key, pair)
		case datatypes.LIST_FIELD_:
			var pairs = make([]datatypes.IPair, len(fld.Pairs))
			for idxKey, k := range fld.Pairs {
				idx, _ := strconv.Atoi(idxKey)
				pairs[idx] = parsedPairMapping[k]
			}
			f = datatypes.NewListField(fld.Key, pairs)
		case datatypes.DICT_FIELD_:
			var pairs = make(map[string]datatypes.IPair, len(fld.Pairs))
			for k, alias := range fld.Pairs {
				pairs[alias] = parsedPairMapping[k]
			}
			f = datatypes.NewDictField(fld.Key, pairs)
		default:
			log.
				WithFields(log.Fields{
					"fieldType": fld.FieldType,
					"field":     fld,
				}).
				Warnf("mysqlConverter.ToContainer invalid fieldType=%d", fld.FieldType)
			continue
		}

		if _, err = c.SetField(f); err != nil {
			log.
				WithFields(log.Fields{
					"field": f,
					"error": err,
				}).
				Warnf("mysqlConverter.ToContainer failed to SetField: %v", err)
		}
	}

	return c, nil
}
