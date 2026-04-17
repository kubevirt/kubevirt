package jd

type voidNode struct{}

var _ JsonNode = voidNode{}

func isVoid(n JsonNode) bool {
	if n == nil {
		return false
	}
	if _, ok := n.(voidNode); ok {
		return true
	}
	return false
}

func isNull(n JsonNode) bool {
	if n == nil {
		return false
	}
	if _, ok := n.(jsonNull); ok {
		return true
	}
	return false
}

func (v voidNode) Json(_ ...Metadata) string {
	return ""
}

func (v voidNode) Yaml(_ ...Metadata) string {
	return ""
}

func (v voidNode) raw() interface{} {
	return ""
}

func (v voidNode) Equals(n JsonNode, metadata ...Metadata) bool {
	switch n.(type) {
	case voidNode:
		return true
	default:
		return false
	}
}

func (v voidNode) hashCode(_ []Metadata) [8]byte {
	return hash([]byte{0xF3, 0x97, 0x6B, 0x21, 0x91, 0x26, 0x8D, 0x96}) // Random bytes
}

func (v voidNode) Diff(n JsonNode, metadata ...Metadata) Diff {
	return v.diff(n, make(path, 0), metadata, getPatchStrategy(metadata))
}

func (v voidNode) diff(
	n JsonNode,
	p path,
	metadata []Metadata,
	strategy patchStrategy,
) Diff {
	return diff(v, n, p, metadata, strategy)
}

func (v voidNode) Patch(d Diff) (JsonNode, error) {
	return patchAll(v, d)
}

func (v voidNode) patch(
	pathBehind, pathAhead path,
	oldValues, newValues []JsonNode,
	strategy patchStrategy,
) (JsonNode, error) {
	return patch(v, pathBehind, pathAhead, oldValues, newValues, strategy)
}
