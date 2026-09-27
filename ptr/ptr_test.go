package ptr

import "testing"

func TestOptionalValue(t *testing.T) {
	pointer := To("value")
	if pointer == nil || Value(pointer) != "value" || Value[string](nil) != "" {
		t.Fatal("pointer conversion did not preserve value and nil semantics")
	}
}
