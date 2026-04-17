package jd

import (
	"bytes"
	"encoding/binary"
	"math"
)

type jsonNumber float64

var _ JsonNode = jsonNumber(0)

func (n jsonNumber) Json(_ ...Metadata) string {
	return renderJson(n.raw())
}

func (n jsonNumber) Yaml(_ ...Metadata) string {
	return renderYaml(n.raw())
}

func (n jsonNumber) raw() interface{} {
	return float64(n)
}

func (n1 jsonNumber) Equals(node JsonNode, metadata ...Metadata) bool {

	precision := getPrecision(metadata)

	n2, ok := node.(jsonNumber)
	if !ok {
		return false
	}
	return math.Abs(float64(n1)-float64(n2)) <= precision
}

func (n jsonNumber) hashCode(metadata []Metadata) [8]byte {
	a := make([]byte, 0, 8)
	b := bytes.NewBuffer(a)
	binary.Write(b, binary.LittleEndian, n)
	return hash(b.Bytes())
}

func (n jsonNumber) Diff(node JsonNode, metadata ...Metadata) Diff {
	return n.diff(node, make(path, 0), metadata, getPatchStrategy(metadata))
}

func (n jsonNumber) diff(
	node JsonNode,
	path path,
	metadata []Metadata,
	strategy patchStrategy,
) Diff {
	return diff(n, node, path, metadata, strategy)
}

func (n jsonNumber) Patch(d Diff) (JsonNode, error) {
	return patchAll(n, d)
}

func (n jsonNumber) patch(
	pathBehind, pathAhead path,
	oldValues, newValues []JsonNode,
	strategy patchStrategy,
) (JsonNode, error) {
	return patch(n, pathBehind, pathAhead, oldValues, newValues, strategy)
}
