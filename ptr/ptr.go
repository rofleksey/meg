// Package ptr provides small helpers for optional Go values.
package ptr

// To returns a pointer to value.
func To[T any](value T) *T { return &value }

// Value returns the pointed-to value or T's zero value when pointer is nil.
func Value[T any](pointer *T) T {
	if pointer == nil {
		var zero T
		return zero
	}

	return *pointer
}
