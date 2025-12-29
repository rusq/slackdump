// Package primitive contains some primitives and helper functions.
package primitive

// IfTrue returns second argument if the condition is true, otherwise, returns the third one.
// Same as C's ternary condition operator:
//
//	cond ? t : f;
func IfTrue[T any](cond bool, t T, f T) T {
	if cond {
		return t
	}
	return f
}
