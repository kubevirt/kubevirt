package jd

type jsonNull []byte

var _ JsonNode = jsonNull{}

func (n jsonNull) Json(_ ...Metadata) string {
	return renderJson(n.raw())
}

func (n jsonNull) Yaml(_ ...Metadata) string {
	return renderJson(n.raw())
}

func (n jsonNull) raw() interface{} {
	return nil
}

func (n jsonNull) Equals(node JsonNode, metadata ...Metadata) bool {
	switch node.(type) {
	case jsonNull:
		return true
	default:
		return false
	}
}

func (n jsonNull) hashCode(_ []Metadata) [8]byte {
	return hash([]byte{0xFE, 0x73, 0xAB, 0xCC, 0xE6, 0x32, 0xE0, 0x88}) // random bytes
}

func (n jsonNull) Diff(node JsonNode, metadata ...Metadata) Diff {
	return n.diff(node, make(path, 0), metadata, getPatchStrategy(metadata))
}

func (n jsonNull) diff(
	node JsonNode,
	path path,
	metadata []Metadata,
	strategy patchStrategy,
) Diff {
	return diff(n, node, path, metadata, strategy)
}

func (n jsonNull) Patch(d Diff) (JsonNode, error) {
	return patchAll(n, d)
}

func (n jsonNull) patch(
	pathBehind, pathAhead path,
	oldValues, newValues []JsonNode,
	strategy patchStrategy,
) (JsonNode, error) {
	return patch(n, pathBehind, pathAhead, oldValues, newValues, strategy)
}
