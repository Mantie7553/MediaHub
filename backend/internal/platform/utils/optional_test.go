package utils

import (
	"encoding/json"
	"testing"
)

func TestOptionalTellsNullFromMissing(t *testing.T) {
	var req struct {
		Missing Optional[int]    `json:"missing"`
		Null    Optional[int]    `json:"null"`
		Value   Optional[string] `json:"value"`
	}
	if err := json.Unmarshal([]byte(`{"null":null,"value":"x"}`), &req); err != nil {
		t.Fatal(err)
	}
	if req.Missing.Set {
		t.Error("a missing field must not count as set")
	}
	if !req.Null.Set || req.Null.Value != nil {
		t.Errorf("null = %+v, want set and nil", req.Null)
	}
	if !req.Value.Set || req.Value.Value == nil || *req.Value.Value != "x" {
		t.Errorf("value = %+v", req.Value)
	}
}

func TestOptionalRejectsWrongType(t *testing.T) {
	var req struct {
		N Optional[int] `json:"n"`
	}
	if err := json.Unmarshal([]byte(`{"n":"ten"}`), &req); err == nil {
		t.Error("a string for an int field must fail")
	}
}
