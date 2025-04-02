package testutil

func Ptr[T any](v T) *T {
	return &v
}
