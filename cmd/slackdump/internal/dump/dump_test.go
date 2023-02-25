package dump

import (
	"context"
	_ "embed"
	"testing"
)

func Test_reconstruct(t *testing.T) {
	if err := reconstruct(context.Background(), nil, "../../../../tmp", namer{}); err != nil {
		t.Fatal(err)
	}
	t.Fatal("x")
}
