package datatypes

func WithInt(i int) IntDT {
	return IntDT(i)
}

func WithFloat(f float64) FloatDT {
	return FloatDT(f)
}

func WithString(s string) StringDT {
	return StringDT(s)
}

func WithBool(b bool) BoolDT {
	return BoolDT(b)
}

// WithList returns an empty list contains nothing.
func WithList() ListDT {
	return ListDT{}
}

func ConstructListRecursive(v []interface{}) ListDT {
	if v == nil {
		return nil
	}

	l := WithList()
	if len(v) == 0 {
		return l
	}

	for _, value := range v {
		l.Append(constructIDataRecursive(value))
	}

	return l
}

// WithDict returns an empty dict contains nothing.
func WithDict() DictDT {
	d := make(DictDT, 4)
	return d
}

func ConstructDictRecursive(v map[string]interface{}) DictDT {
	if v == nil {
		return nil
	}

	d := WithDict()
	if len(v) == 0 {
		return d
	}

	for k, value := range v {
		d.Add(k, constructIDataRecursive(value))
	}
	return d
}

func ConstructIData(v interface{}) IData {
	return constructIDataRecursive(v)
}

func constructIDataRecursive(v interface{}) (d IData) {
	switch v.(type) {
	case int, int8, int16, int32, int64, uint, uint8, uint16, uint32, uint64:
		// NOTE(@yeqown) maybe unsafe can helps this, if could not convert to int from (uint or int_x)
		d = WithInt(v.(int))
	case float64, float32:
		d = WithFloat(v.(float64))
	case string:
		d = WithString(v.(string))
	case bool:
		d = WithBool(v.(bool))
	case []interface{}:
		l := WithList()
		for _, value := range v.([]interface{}) {
			l.Append(constructIDataRecursive(value))
		}
		d = l
	case map[string]interface{}:
		l := WithList()
		for _, value := range v.(map[string]interface{}) {
			l.Append(constructIDataRecursive(value))
		}
		d = l
	}

	return d
}
