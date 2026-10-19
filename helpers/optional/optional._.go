package optional

import "encoding/json"

type Value[T any] struct {
	Set  bool
	Null bool
	V    T
}

func Of[T any](v T) Value[T] {
	return Value[T]{Set: true, V: v}
}

func (v *Value[T]) UnmarshalJSON(b []byte) error {
	v.Set = true
	if string(b) == "null" {
		v.Null = true
		return nil
	}

	return json.Unmarshal(b, &v.V)
}

func (v Value[T]) MarshalJSON() ([]byte, error) {
	if !v.Set || v.Null {
		return []byte("null"), nil
	}

	return json.Marshal(v.V)
}

func (v Value[T]) Valid() bool {
	return v.Set && !v.Null
}
