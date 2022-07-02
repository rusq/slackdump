package structures

type EntityListIdx map[string]bool

func (el EntityListIdx) Merge(val string, include bool) error {
	if _, ok := el[val]; ok && !IsValidSlackURL(val) {
		el[val] = include
		return nil
	}

	ci, err := ParseURL(val)
	if err != nil {
		return err
	}
	if _, ok := el[ci.String()]; ok {
		el[ci.String()] = include
	}
	return nil
}
