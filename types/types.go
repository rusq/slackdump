package types

// toslice creates a slice of IN type containing the keys from m.
func toslice[IN comparable](m map[IN]bool) []IN {
	var out = make([]IN, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	return out
}
