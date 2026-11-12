package utils

import (
	"bytes"
	"encoding/json"
)

/*
Optional is a JSON field that can be left out, set to null or set to a value. A plain pointer
can't tell "left out" from null, and null is how a field is cleared
*/
type Optional[T any] struct {
	Set   bool
	Value *T
}

func (o *Optional[T]) UnmarshalJSON(data []byte) error {
	o.Set = true
	if bytes.Equal(data, []byte("null")) {
		o.Value = nil
		return nil
	}
	var v T
	if err := json.Unmarshal(data, &v); err != nil {
		return err
	}
	o.Value = &v
	return nil
}
