package utils

// AndThen applies the given function if the passed value is non-nil and returns `nil` otherwise.
func AndThen[T any, V any](value *T, f func(T) V) *V {
	if value == nil {
		return nil
	}
	result := f(*value)
	return &result
}

// Map is a functional operator to map all values of a slice to a different type.
func Map[T any, V any](values []T, f func(T) V) []V {
	result := make([]V, 0, len(values))
	for _, val := range values {
		result = append(result, f(val))
	}
	return result
}
