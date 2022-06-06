package fixtures

import "math/rand"

func RandString(sz int) string {
	const (
		charset = "ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz"
		chrstSz = len(charset)
	)
	var ret = make([]byte, sz)
	for i := 0; i < sz; i++ {
		ret[i] = charset[rand.Int()%chrstSz]
	}
	return string(ret)
}
