package auth

// things specific to v2 (to ease backporting)

func (p RodAuth) Type() Type {
	return TypeRod
}
