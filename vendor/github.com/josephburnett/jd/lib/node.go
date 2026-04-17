package jd

import (
	"fmt"
)

// JsonNode is a JSON value, collection of values, or a void representing
// the absense of a value. JSON values can be a Number, String, Boolean
// or Null. Collections can be an Object, native JSON array, ordered
// List, unordered Set or Multiset. JsonNodes are created with the
// NewJsonNode function or ReadJson* and ReadYaml* functions.
type JsonNode interface {

	// Json renders a JsonNode as a JSON string.
	Json(renderOptions ...Metadata) string

	// Yaml renders a JsonNode as a YAML string in block format.
	Yaml(renderOptions ...Metadata) string

	// Equals returns true if the JsonNodes are equal according to
	// the provided Metadata. The default behavior (no Metadata) is
	// to compare the entire structure down to scalar values treating
	// Arrays as orders Lists. The SET and MULTISET Metadata will
	// treat Arrays as sets or multisets (bags) respectively. To deep
	// compare objects in an array irrespective of order, the SetKeys
	// function will construct Metadata to compare objects by a set
	// of keys. If two JsonNodes are equal, then Diff with the same
	// Metadata will produce an empty Diff. And vice versa.
	Equals(n JsonNode, metadata ...Metadata) bool

	// Diff produces a list of differences (Diff) between two
	// JsonNodes such that if the output Diff were applied to the
	// first JsonNode (Patch) then the two JsonNodes would be
	// Equal. The necessary Metadata is embeded in the Diff itself so
	// only the Diff is required to Patch a JsonNode.
	Diff(n JsonNode, metadata ...Metadata) Diff

	// Patch applies a Diff to a JsonNode. No Metadata is provided
	// because the original interpretation of the structure is
	// embedded in the Diff itself.
	Patch(d Diff) (JsonNode, error)

	jsonNodeInternals
}

type jsonNodeInternals interface {
	raw() interface{}
	hashCode(metadata []Metadata) [8]byte
	diff(n JsonNode, p path, metadata []Metadata, strategy patchStrategy) Diff
	patch(pathBehind, pathAhead path, oldValues, newValues []JsonNode, strategy patchStrategy) (JsonNode, error)
}

// NewJsonNode constructs a JsonNode from native Golang objects. See the
// function source for supported types and conversions. Slices are always
// placed into native JSON Arrays and interpretated as Lists, Sets or
// Multisets based on Metadata provided during Equals and Diff
// operations.
func NewJsonNode(n interface{}) (JsonNode, error) {
	switch t := n.(type) {
	case map[string]interface{}:
		m := newJsonObject()
		for k, v := range t {
			n, ok := v.(JsonNode)
			if !ok {
				e, err := NewJsonNode(v)
				if err != nil {
					return nil, err
				}
				n = e
			}
			m[k] = n
		}
		return m, nil
	case map[interface{}]interface{}:
		m := newJsonObject()
		for k, v := range t {
			s, ok := k.(string)
			if !ok {
				return nil, fmt.Errorf("unsupported key type %T", k)
			}
			if _, ok := v.(JsonNode); !ok {
				e, err := NewJsonNode(v)
				if err != nil {
					return nil, err
				}
				m[s] = e
			}
		}
		return m, nil
	case []interface{}:
		l := make(jsonArray, len(t))
		for i, v := range t {
			if _, ok := v.(JsonNode); !ok {
				e, err := NewJsonNode(v)
				if err != nil {
					return nil, err
				}
				l[i] = e
			}
		}
		return l, nil
	case float64:
		return jsonNumber(t), nil
	case int:
		return jsonNumber(t), nil
	case string:
		return jsonString(t), nil
	case bool:
		return jsonBool(t), nil
	case nil:
		return jsonNull(nil), nil
	default:
		return nil, fmt.Errorf("unsupported type %T", t)
	}
}

func nodeList(n ...JsonNode) []JsonNode {
	l := []JsonNode{}
	if len(n) == 0 {
		return l
	}
	if n[0].Equals(voidNode{}) {
		return l
	}
	return append(l, n...)
}
