package primitive

import (
	"fmt"
	"iter"
)

// Collect collects all Ks from iterator it, returning any encountered error.
func Collect[K any](it iter.Seq2[K, error]) ([]K, error) {
	kk := make([]K, 0)
	for k, err := range it {
		if err != nil {
			return kk, fmt.Errorf("iterator error: %w", err)
		}
		kk = append(kk, k)
	}
	return kk, nil
}
