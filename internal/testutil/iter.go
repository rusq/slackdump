package testutil

import "iter"

type IterVal[T, U any] struct {
	T T
	U U
}

func Seq2Collect[T, U any](it iter.Seq2[T, U]) (ret []IterVal[T, U]) {
	for t, u := range it {
		ret = append(ret, IterVal[T, U]{T: t, U: u})
	}
	return
}
