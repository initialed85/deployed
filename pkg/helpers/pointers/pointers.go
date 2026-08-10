package pointers

//go:fix inline
func Ptr[T any](v T) *T {
	return new(v)
}

func Deref[T any](v *T, d ...T) T {
	if v == nil {
		if len(d) > 0 {
			return d[0]
		}

		return *new(T)
	}

	return *v
}
