package mysql

import (
	"encoding/json"

	"github.com/yeqown/log"

	"github.com/pkg/errors"

	"github.com/yeqown/cassem/pkg/datatypes"
)

var (
	ErrNilPair         = errors.New("nil pair")
	ErrInvalidPairDO   = errors.New("invalid pair DO data")
	ErrUnknownDatatype = errors.New("unknown datatype")
	ErrNilContainer    = errors.New("nil container")
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
	if !ok {
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

type formContainerParsed struct {
	c          *ContainerDO
	fields     []*FieldDO
	kvFields   []*KVFieldToPairDO
	listFields []*ListFieldToPairDO
	dictFields []*DictFieldToPairDO
	//mappingFieldToId map[string]uint
	//mappingPairToId  map[string]uint
}

func (m mysqlConverter) FromContainer(c datatypes.IContainer) (interface{}, error) {
	if c == nil {
		return nil, ErrNilContainer
	}

	parsed := formContainerParsed{
		c: &ContainerDO{
			Key:       c.Key(),
			Namespace: c.NS(),
			CheckSum:  "", // TODO(@yeqown) add check sum
		},
		fields:     nil,
		kvFields:   nil,
		listFields: nil,
		dictFields: nil,
	}

	_fields := c.Fields()
	var (
		fieldDOs = make([]*FieldDO, 0, len(_fields))
		kv       = make([]*KVFieldToPairDO, 0, len(_fields))
		l        = make([]*ListFieldToPairDO, 0, len(_fields))
		d        = make([]*DictFieldToPairDO, 0, len(_fields))
	)

	for _, fld := range _fields {
		fldKey := ""
		pairKey := ""

		fieldDOs = append(fieldDOs, &FieldDO{
			FieldType: fld.Type(),
			Key:       fld.Name(),
			//ContainerID: 0,
		})

		// mapping field to pairs, so repository could query or update
		switch fld.Type() {
		case datatypes.KV_FIELD_:
			fldKey = fld.Name()
			pairKey = fld.Value().(datatypes.IPair).Key()
			kv = append(kv, &KVFieldToPairDO{
				FieldKey: fldKey,
				PairKey:  pairKey,
				//ContainerID: 0,
			})
		case datatypes.LIST_FIELD_:
			for _, v := range fld.Value().([]datatypes.IPair) {
				l = append(l, &ListFieldToPairDO{
					//ContainerID: 0,
					FieldKey: fld.Name(),
					PairKey:  v.Key(),
				})
			}
		case datatypes.DICT_FIELD_:
			for k, v := range fld.Value().(map[string]datatypes.IPair) {
				d = append(d, &DictFieldToPairDO{
					//ContainerID:  0,
					FieldKey:     fld.Name(),
					PairKey:      v.Key(),
					DictFieldKey: k,
				})
			}
		default:
			log.
				WithField("fld", fld).
				Warn("invalid field type: %d", fld.Type())
		}
	}

	(&parsed).fields = fieldDOs
	(&parsed).kvFields = kv
	(&parsed).listFields = l
	(&parsed).dictFields = d

	return &parsed, nil
}

func (m mysqlConverter) ToContainer(v interface{}) (datatypes.IContainer, error) {
	panic("implement me")
}
