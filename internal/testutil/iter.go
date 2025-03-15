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

func IterVal2Iter[T, U any](s []IterVal[T, U]) iter.Seq2[T, U] {
	return func(yield func(T, U) bool) {
		for _, v := range s {
			if !yield(v.T, v.U) {
				return
			}
		}
	}
}

func Slice2Seq2[T any](s []T) iter.Seq2[T, error] {
	return func(yield func(T, error) bool) {
		for _, v := range s {
			if !yield(v, nil) {
				return
			}
		}
	}
}
