package jd

import (
	"fmt"
)

func patchAll(n JsonNode, d Diff) (JsonNode, error) {
	var err error
	for _, de := range d {
		strategy := path(de.Path).getPatchStrategy()
		n, err = n.patch(make(path, 0), de.Path, de.OldValues, de.NewValues, strategy)
		if err != nil {
			return nil, err
		}
	}
	return n, nil
}

func patch(
	node JsonNode,
	pathBehind, pathAhead path,
	oldValues, newValues []JsonNode,
	strategy patchStrategy,
) (JsonNode, error) {
	if !pathAhead.isLeaf() {
		if strategy != mergePatchStrategy {
			return patchErrExpectColl(node, pathAhead[0])
		}
		next, _, rest := pathAhead.next()
		key, ok := next.(jsonString)
		if !ok {
			return nil, fmt.Errorf("merge patch path must be composed of only strings: found %v", next)
		}
		o := newJsonObject()
		value, err := node.patch(append(pathBehind.clone(), key), rest, oldValues, newValues, strategy)
		if err != nil {
			return nil, err
		}
		if !isVoid(value) || !rest.isLeaf() {
			o[string(key)] = value
		}
		return o, nil
	}
	if len(oldValues) > 1 || len(newValues) > 1 {
		return patchErrNonSetDiff(oldValues, newValues, pathBehind)
	}
	oldValue := singleValue(oldValues)
	newValue := singleValue(newValues)
	switch strategy {
	case mergePatchStrategy:
		if !isVoid(oldValue) {
			return patchErrMergeWithOldValue(pathBehind, oldValue)
		}
	case strictPatchStrategy:
		if !node.Equals(oldValue) {
			return patchErrExpectValue(oldValue, node, pathBehind)
		}
	default:
		return patchErrUnsupportedPatchStrategy(pathBehind, strategy)
	}
	return newValue, nil

}

func singleValue(nodes []JsonNode) JsonNode {
	if len(nodes) == 0 {
		return voidNode{}
	}
	return nodes[0]
}

func patchErrExpectColl(n JsonNode, pe interface{}) (JsonNode, error) {
	switch pe := pe.(type) {
	case string:
		return nil, fmt.Errorf(
			"found %v at %v: expected JSON object",
			// TODO: plumb through metadata.
			n.Json(), pe)
	case float64:
		return nil, fmt.Errorf(
			"found %v at %v: expected JSON array",
			n.Json(), pe)
	default:
		return nil, fmt.Errorf("invalid path element %v", pe)
	}

}

func patchErrNonSetDiff(oldValues, newValues []JsonNode, path path) (JsonNode, error) {
	if len(oldValues) > 1 {
		return nil, fmt.Errorf(
			"invalid diff: multiple removals from non-set at %v",
			path)
	} else {
		return nil, fmt.Errorf(
			"invalid diff: multiple additions to a non-set at %v",
			path)
	}
}

func patchErrExpectValue(want, found JsonNode, path path) (JsonNode, error) {
	return nil, fmt.Errorf(
		"found %v at %v: expected %v",
		found.Json(), path, want.Json())
}

func patchErrMergeWithOldValue(path path, oldValue JsonNode) (JsonNode, error) {
	return nil, fmt.Errorf(
		"patch with merge strategy at %v has unnecessary old value %v",
		path, oldValue)
}

func patchErrUnsupportedPatchStrategy(path path, strategy patchStrategy) (JsonNode, error) {
	return nil, fmt.Errorf(
		"unsupported patch strategy %v at %v",
		strategy, path)
}
