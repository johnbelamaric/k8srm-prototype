package schedule

func (r *Resource) MeetsConstraints(constraints string, poolAttrs []Attribute) (bool, error) {
	return true, nil
}
