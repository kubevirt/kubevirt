package jd

import "strconv"

// jsonStringOrInteger is a string unless it needs to be a number. It's
// created only when encountering an integer token in a JSON
// Pointer. Because the JSON Pointer spec doesn't quote strings, it's
// impossible to tell if an integer is supposed to be an array index or a
// map key. Rather than forbidding map keys that look like integers, we
// defer determining the concrete type until the token is actually
// used. When indexing into an object it's a string. When indexing into a
// list it's an integer.
type jsonStringOrInteger string

var _ JsonNode = jsonStringOrInteger("")

func (sori jsonStringOrInteger) Json(_ ...Metadata) string {
	return renderJson(sori.raw())
}

func (sori jsonStringOrInteger) Yaml(_ ...Metadata) string {
	return renderYaml(sori.raw())
}

func (sori jsonStringOrInteger) raw() interface{} {
	return string(sori)
}

func (sori jsonStringOrInteger) Equals(node JsonNode, metadata ...Metadata) bool {
	switch node.(type) {
	case jsonString:
		return jsonString(sori).Equals(node, metadata...)
	case jsonNumber:
		i, err := strconv.Atoi(string(sori))
		if err != nil {
			return false
		}
		return jsonNumber(i).Equals(node, metadata...)
	default:
		return false
	}
}

func (sori jsonStringOrInteger) hashCode(metadata []Metadata) [8]byte {
	return jsonString(sori).hashCode(metadata)
}

func (sori jsonStringOrInteger) Diff(node JsonNode, metadata ...Metadata) Diff {
	return sori.diff(node, make(path, 0), metadata, getPatchStrategy(metadata))
}

func (sori jsonStringOrInteger) diff(
	node JsonNode,
	path path,
	metadata []Metadata,
	strategy patchStrategy,
) Diff {
	switch node.(type) {
	case jsonString:
		return jsonString(sori).Diff(node, metadata...)
	case jsonNumber:
		i, err := strconv.Atoi(string(sori))
		if err != nil {
			return jsonString(sori).Diff(node, metadata...)
		}
		return jsonNumber(i).Diff(node, metadata...)
	default:
		return jsonString(sori).Diff(node, metadata...)
	}
}

func (sori jsonStringOrInteger) Patch(d Diff) (JsonNode, error) {
	return patchAll(jsonString(sori), d)
}

func (sori jsonStringOrInteger) patch(
	pathBehind, pathAhead path,
	oldValue, newValue []JsonNode,
	strategy patchStrategy,
) (JsonNode, error) {
	return patch(jsonString(sori), pathBehind, pathAhead, oldValue, newValue, strategy)
}
