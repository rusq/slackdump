package testutil

import (
	"encoding/json"
	"testing"
)

// MarshalJSON marshals data to JSON and returns the byte slice.
func MarshalJSON(t *testing.T, v any) []byte {
	t.Helper()
	b, err := json.Marshal(v)
	if err != nil {
		t.Fatal(err)
	}
	return b
}
