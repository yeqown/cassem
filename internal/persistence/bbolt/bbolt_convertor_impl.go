package bbolt

import (
	"encoding/json"
	"strconv"

	"github.com/yeqown/cassem/pkg/set"
	"github.com/yeqown/log"

	"github.com/yeqown/cassem/internal/persistence"
	"github.com/yeqown/cassem/pkg/datatypes"

	"github.com/pkg/errors"
)

var _ persistence.Converter = bboltConvertorImpl{}

var (
	ErrNilPair                = errors.New("nil pair")
	ErrInvalidPairDO          = errors.New("invalid pair DO data")
	ErrInvalidPairWithNonData = errors.New("non data couldn't be used to save")
	ErrUnknownDatatype        = errors.New("unknown datatype")
	ErrNilContainer           = errors.New("nil container")
	ErrInvalidContainerDO     = errors.New("invalid container DO data")

	ErrPairKeyNotExist = errors.New("some pair bucketKey is not exists")
)

type bboltConvertorImpl struct{}

func NewConverter() persistence.Converter {
	return bboltConvertorImpl{}
}

func (b bboltConvertorImpl) FromPair(p datatypes.IPair) (interface{}, error) {
	if p == nil {
		return nil, ErrNilPair
	}

	if p.Value() == nil || p.Value().Datatype() == datatypes.EMPTY_DATATYPE {
		return nil, ErrInvalidPairWithNonData
	}

	v, err := p.MarshalJSON()
	if err != nil {
		return nil, errors.Wrap(err, "bboltConvertorImpl.FromPair failed")
	}

	return &pairDO{
		Key:       p.Key(),
		Namespace: p.NS(),
		Datatype:  p.Value().Datatype(),
		Value:     v,
	}, nil
}

func (b bboltConvertorImpl) ToPair(v interface{}) (p datatypes.IPair, err error) {
	pd, ok := v.(*pairDO)
	if !ok || pd == nil {
		return nil, ErrInvalidPairDO
	}

	var (
		d datatypes.IData
	)
	switch pd.Datatype {
	case datatypes.BOOL_DATATYPE_:
		var b bool
		if err = json.Unmarshal(pd.Value, &b); err == nil {
			d = datatypes.WithBool(b)
		}
	case datatypes.INT_DATATYPE_:
		var b int
		if err = json.Unmarshal(pd.Value, &b); err == nil {
			d = datatypes.WithInt(b)
		}
	case datatypes.STRING_DATATYPE_:
		var b string
		if err = json.Unmarshal(pd.Value, &b); err == nil {
			d = datatypes.WithString(b)
		}
	case datatypes.FLOAT_DATATYPE_:
		var b float64
		if err = json.Unmarshal(pd.Value, &b); err == nil {
			d = datatypes.WithFloat(b)
		}
	case datatypes.LIST_DATATYPE_:
		var ll []interface{}
		d = datatypes.WithList()
		if err = json.Unmarshal(pd.Value, &ll); err == nil {
			d = datatypes.FromSliceInterfaceToList(ll)
		}
	case datatypes.DICT_DATATYPE_:
		d = datatypes.WithDict()
		dd := make(map[string]interface{})
		if err = json.Unmarshal(pd.Value, &dd); err == nil {
			d = datatypes.FromMapInterfaceToDict(dd)
		}
	default:
		err = ErrUnknownDatatype
	}

	if err != nil {
		return
	}
	p = datatypes.NewPair(pd.Namespace, pd.Key, d)

	return
}

func (b bboltConvertorImpl) FromContainer(c datatypes.IContainer) (interface{}, error) {
	if c == nil {
		return nil, ErrNilContainer
	}

	_fields := c.Fields()
	parsed := formContainerParsed{
		c: &containerDO{
			Key:       c.Key(),
			Namespace: c.NS(),
			// CheckSum:  "", NOTICE: checksum would not be calculate and updated, until it's requested.
		},
		uniquePairKeys: set.NewStringSet(len(_fields) * 4),
	}

	for _, fld := range _fields {
		pairs := make(fieldPairs, 16)
		// mapping field to pairs, so repository could query or update
		switch fld.Type() {
		case datatypes.KV_FIELD_:
			pairKey := fld.Value().(datatypes.IPair).Key()
			pairs["KV"] = pairKey
			_ = parsed.uniquePairKeys.Add(pairKey)
		case datatypes.LIST_FIELD_:
			// FIXED(@yeqown) list may have duplicated paris. so how to keep the origin detail.
			for idx, v := range fld.Value().([]datatypes.IPair) {
				pairs[strconv.Itoa(idx)] = v.Key()
				_ = parsed.uniquePairKeys.Add(v.Key())
			}
		case datatypes.DICT_FIELD_:
			for k, v := range fld.Value().(map[string]datatypes.IPair) {
				pairs[k] = v.Key()
				_ = parsed.uniquePairKeys.Add(v.Key())
			}
		default:
			log.
				WithField("fld", fld).
				Warnf("invalid field type: %d", fld.Type())
		}

		//_ = parsed.uniqueFieldKeys.Add(fld.Name())
		parsed.c.Fields = append(parsed.c.Fields, field{
			FieldType: fld.Type(),
			Key:       fld.Name(),
			Pairs:     pairs,
		})
	}

	return &parsed, nil
}

func (b bboltConvertorImpl) ToContainer(v interface{}) (datatypes.IContainer, error) {
	toc, ok := v.(*toContainerWithPairs)
	if !ok || toc == nil || toc.c == nil {
		log.WithFields(log.Fields{
			"v":   v,
			"ok":  ok,
			"toc": toc,
		}).Warn("bboltConvertorImpl.ToContainer invalid containerDO")

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
		for pairKey, pairVO := range toc.pairs {
			// parse pair and save into mapping
			if parsedPairMapping[pairKey], err = b.ToPair(pairVO); err != nil {
				log.WithFields(log.Fields{
					"pairKey": pairKey,
					"pairDO":  v,
					"pairs":   toc.pairs,
				}).Warnf("bboltConvertorImpl.ToContainer failed to convert pair: %v", err)
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
			for _, pairKey := range fld.Pairs {
				pair = parsedPairMapping[pairKey]
			}
			f = datatypes.NewKVField(fld.Key, pair)
		case datatypes.LIST_FIELD_:
			var pairs = make([]datatypes.IPair, len(fld.Pairs))
			for idxKey, pairKey := range fld.Pairs {
				idx, _ := strconv.Atoi(idxKey)
				pairs[idx] = parsedPairMapping[pairKey]
			}
			f = datatypes.NewListField(fld.Key, pairs)
		case datatypes.DICT_FIELD_:
			var pairs = make(map[string]datatypes.IPair, len(fld.Pairs))
			for dictKey, pairKey := range fld.Pairs {
				pairs[dictKey] = parsedPairMapping[pairKey]
			}
			f = datatypes.NewDictField(fld.Key, pairs)
		default:
			log.
				WithFields(log.Fields{
					"fieldType": fld.FieldType,
					"field":     fld,
				}).
				Warnf("bboltConvertorImpl.ToContainer invalid fieldType=%d", fld.FieldType)
			continue
		}

		if _, err = c.SetField(f); err != nil {
			log.
				WithFields(log.Fields{
					"field": f,
					"error": err,
				}).
				Warnf("bboltConvertorImpl.ToContainer failed to SetField: %v", err)
		}
	}

	// set containerDO's checksum to container
	c.CheckSum(toc.c.CheckSum)

	return c, nil
}
