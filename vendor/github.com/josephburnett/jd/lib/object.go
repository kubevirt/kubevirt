package jd

import (
	"fmt"
	"sort"
)

type jsonObject map[string]JsonNode

var _ JsonNode = jsonObject{}

func newJsonObject() jsonObject {
	return jsonObject{}
}

func (o jsonObject) Json(_ ...Metadata) string {
	return renderJson(o.raw())
}

func (o jsonObject) MarshalJSON() ([]byte, error) {
	return []byte(o.Json()), nil
}

func (o jsonObject) Yaml(_ ...Metadata) string {
	return renderYaml(o.raw())
}

func (o jsonObject) raw() interface{} {
	j := make(map[string]interface{})
	for k, v := range o {
		j[k] = v.raw()
	}
	return j
}

func (o1 jsonObject) Equals(n JsonNode, metadata ...Metadata) bool {
	o2, ok := n.(jsonObject)
	if !ok {
		return false
	}
	if len(o1) != len(o2) {
		return false
	}

	for key1, val1 := range o1 {
		val2, ok := o2[key1]
		if !ok {
			return false
		}
		ret := val1.Equals(val2, metadata...)
		if !ret {
			return false
		}
	}
	return true
}

func (o jsonObject) hashCode(metadata []Metadata) [8]byte {
	keys := make([]string, 0, len(o))
	for k := range o {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	a := make([]byte, 0, len(o)*16)
	for _, k := range keys {
		keyHash := hash([]byte(k))
		a = append(a, keyHash[:]...)
		valueHash := o[k].hashCode(metadata)
		a = append(a, valueHash[:]...)
	}
	return hash(a)
}

// ident is the identity of the json object based on either the hash of a
// given set of keys or the full object if no keys are present.
func (o jsonObject) ident(metadata []Metadata) [8]byte {
	keys := map[string]bool{}
	if meta := getSetkeysMetadata(metadata); meta != nil {
		keys = meta.keys
	}
	if len(keys) == 0 {
		return o.hashCode(metadata)
	}
	hashes := hashCodes{
		// We start with a constant hash to distinguish between
		// an empty object and an empty array.
		[8]byte{0x4B, 0x08, 0xD2, 0x0F, 0xBD, 0xC8, 0xDE, 0x9A}, // random bytes
	}
	for key := range keys {
		v, ok := o[key]
		if ok {
			hashes = append(hashes, v.hashCode(metadata))
		}
	}
	if len(hashes) == 0 {
		return o.hashCode(metadata)
	}
	return hashes.combine()
}

func (o jsonObject) pathIdent(pathObject jsonObject, metadata []Metadata) [8]byte {
	idKeys := map[string]bool{}
	for k := range pathObject {
		idKeys[k] = true
	}
	keys := getSetkeysMetadata(metadata).mergeKeys(idKeys)
	id := make(map[string]interface{})
	for key := range keys {
		if value, ok := o[key]; ok {
			id[key] = value
		}
	}
	e, _ := NewJsonNode(id)
	return e.hashCode([]Metadata{})
}

func (k1 *setkeysMetadata) mergeKeys(k2 map[string]bool) map[string]bool {
	if k1 == nil {
		// Nothing to merge
		return k2
	}
	k3 := make(map[string]bool)
	for k := range k1.keys {
		k3[k] = true
	}
	for k := range k2 {
		k3[k] = true
	}
	return k3
}

func (o jsonObject) Diff(n JsonNode, metadata ...Metadata) Diff {
	return o.diff(n, make(path, 0), metadata, getPatchStrategy(metadata))
}

func (o1 jsonObject) diff(n JsonNode, path path, metadata []Metadata, strategy patchStrategy) Diff {
	d := make(Diff, 0)
	o2, ok := n.(jsonObject)
	if !ok {
		// Different types
		var e DiffElement
		switch strategy {
		case mergePatchStrategy:
			e = DiffElement{
				Path:      path.clone().prependMetadataMerge(),
				NewValues: []JsonNode{n},
			}
		default:
			e = DiffElement{
				Path:      path.clone(),
				OldValues: []JsonNode{o1},
				NewValues: []JsonNode{n},
			}
		}
		return append(d, e)
	}
	o1Keys := make([]string, 0, len(o1))
	for k := range o1 {
		o1Keys = append(o1Keys, k)
	}
	sort.Strings(o1Keys)
	o2Keys := make([]string, 0, len(o2))
	for k := range o2 {
		o2Keys = append(o2Keys, k)
	}
	sort.Strings(o2Keys)
	for _, k1 := range o1Keys {
		v1 := o1[k1]
		if v2, ok := o2[k1]; ok {
			// Both keys are present
			subDiff := v1.diff(v2, append(path, jsonString(k1)), metadata, strategy)
			d = append(d, subDiff...)
		} else {
			// O2 missing key
			var e DiffElement
			switch strategy {
			case mergePatchStrategy:
				e = DiffElement{
					Path:      append(path, jsonString(k1)).clone().prependMetadataMerge(),
					NewValues: []JsonNode{voidNode{}},
				}
			default:
				e = DiffElement{
					Path:      append(path, jsonString(k1)).clone(),
					OldValues: nodeList(v1),
					NewValues: nodeList(),
				}
			}
			d = append(d, e)
		}
	}
	for _, k2 := range o2Keys {
		v2 := o2[k2]
		if _, ok := o1[k2]; !ok {
			// O1 missing key
			var e DiffElement
			switch strategy {
			case mergePatchStrategy:
				e = DiffElement{
					Path:      append(path, jsonString(k2)).clone().prependMetadataMerge(),
					OldValues: nodeList(),
					NewValues: nodeList(v2),
				}
			default:
				e = DiffElement{
					Path:      append(path, jsonString(k2)).clone(),
					OldValues: nodeList(),
					NewValues: nodeList(v2),
				}
			}
			d = append(d, e)
		}
	}
	return d
}

func (o jsonObject) Patch(d Diff) (JsonNode, error) {
	return patchAll(o, d)
}

func (o jsonObject) patch(pathBehind, pathAhead path, oldValues, newValues []JsonNode, strategy patchStrategy) (JsonNode, error) {
	if (len(pathAhead) == 0) && (len(oldValues) > 1 || len(newValues) > 1) {
		return patchErrNonSetDiff(oldValues, newValues, pathBehind)
	}
	// Base case
	if pathAhead.isLeaf() {
		newValue := singleValue(newValues)
		if strategy == mergePatchStrategy {
			return newValue, nil
		}
		oldValue := singleValue(oldValues)
		if !o.Equals(oldValue) {
			return patchErrExpectValue(oldValue, o, pathBehind)
		}
		return newValue, nil
	}
	// Recursive case
	n, _, rest := pathAhead.next()
	// Special case for jsonStringOrInteger
	sori, ok := n.(jsonStringOrInteger)
	if ok {
		n = jsonString(sori)
	}
	// Path entries for objects must be a string
	pe, ok := n.(jsonString)
	if !ok {
		return nil, fmt.Errorf(
			"found %v at %v: expected JSON object",
			o.Json(), pathBehind)
	}
	nextNode, ok := o[string(pe)]
	if !ok {
		switch strategy {
		case mergePatchStrategy:
			// Create objects
			if rest.isLeaf() {
				nextNode = voidNode{}
			} else {
				nextNode = newJsonObject()
			}
		case strictPatchStrategy:
			nextNode = voidNode{}
		default:
			return patchErrUnsupportedPatchStrategy(pathBehind, strategy)
		}
	}
	patchedNode, err := nextNode.patch(append(pathBehind, pe), rest, oldValues, newValues, strategy)
	if err != nil {
		return nil, err
	}
	if isVoid(patchedNode) {
		// Delete a pair
		delete(o, string(pe))
	} else {
		// Add or replace a pair
		o[string(pe)] = patchedNode
	}
	return o, nil
}
