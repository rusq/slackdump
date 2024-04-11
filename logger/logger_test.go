package logger

import "testing"

func BenchmarkSlientPrintf(b *testing.B) {
	var l = Silent
	for i := 0; i < b.N; i++ {
		l.Printf("hello world, %s, %d", "foo", i)
	}
	// This benchmark compares the performance of the Silent logger when
	// using io.Discard, and when using a no-op function.
	// io.Discard: BenchmarkSlientPrintf-16    	93075956	        12.92 ns/op	       8 B/op	       0 allocs/op
	// no-op func: BenchmarkSlientPrintf-16    	1000000000	         0.2364 ns/op	       0 B/op	       0 allocs/op
	//
	// Oh, look! We have an WINNER.  The no-op function wins, no surprises.
}
