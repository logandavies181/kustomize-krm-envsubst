package fieldtype

type hashset map[string]struct{}

func newHashset() hashset {
	return make(hashset)
}

func (h hashset) add(s ...string) {
	for _, v := range s {
		h[v] = struct{}{}
	}
}

func (h hashset) has(s string) bool {
	_, has := h[s]
	return has
}
