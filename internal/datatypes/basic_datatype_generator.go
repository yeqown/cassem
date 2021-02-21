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

func WithList() ListDT {
	return ListDT{}
}

func WithDict() DictDT {
	d := make(DictDT, 4)
	return d
}
