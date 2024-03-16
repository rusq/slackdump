package view

import (
	"path"
	"testing"
)

func TestGlob(t *testing.T) {
	pattern := "[CD]*.json"
	match, err := path.Match(pattern, "D04AJ95SQ5.json")
	if err != nil {
		t.Fatal(err)
	}
	if !match {
		t.Errorf("expected match, got %v", match)
	}
}
