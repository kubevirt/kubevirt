package jd

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"strings"
)

// ReadDiffFile reads a file in native jd format.
func ReadDiffFile(filename string) (Diff, error) {
	bytes, err := ioutil.ReadFile(filename)
	if err != nil {
		return nil, err
	}
	return readDiff(string(bytes))
}

// ReadDiffString reads a string in native jd format.
func ReadDiffString(s string) (Diff, error) {
	return readDiff(s)
}

func readDiff(s string) (Diff, error) {
	diff := Diff{}
	diffLines := strings.Split(s, "\n")
	const (
		INIT = iota
		AT   = iota
		OLD  = iota
		NEW  = iota
	)
	var de DiffElement
	var state = INIT
	for i, dl := range diffLines {
		if len(dl) == 0 {
			continue
		}
		header := dl[:1]
		// Validate state transition.
		switch state {
		case INIT:
			if header != "@" {
				return errorAt(i, "Unexpected %c. Expecteding @.", dl[0])
			}
		case AT:
			if header != "-" && header != "+" {
				return errorAt(i, "Unexpected %c. Expecting - or +.", dl[0])
			}
		case OLD:
			if header != "@" && header != "-" && header != "+" {
				return errorAt(i, "Unexpected %c. Expecting + or @.", dl[0])
			}
		case NEW:
			if header != "+" && header != "@" {
				return errorAt(i, "Unexpected %c. Expecteding + or @.", dl[0])
			}
		}
		// Process line.
		switch header {
		case "@":
			if state != INIT {
				// Save the previous diff element.
				errString := checkDiffElement(de)
				if errString != "" {
					return errorAt(i, errString)
				}
				diff = append(diff, de)
			}
			p, err := ReadJsonString(dl[1:])
			if err != nil {
				return errorAt(i, "Invalid path. %v", err.Error())
			}
			pa, ok := p.(jsonArray)
			if !ok {
				return errorAt(i, "Invalid path. Want JSON list. Got %T.", p)
			}
			de = DiffElement{
				Path:      path(pa).clone(),
				OldValues: []JsonNode{},
				NewValues: []JsonNode{},
			}
			state = AT
		case "-":
			v, err := ReadJsonString(dl[1:])
			if err != nil {
				return errorAt(i, "Invalid value. %v", err.Error())
			}
			de.OldValues = append(de.OldValues, v)
			state = OLD
		case "+":
			v, err := ReadJsonString(dl[1:])
			if err != nil {
				return errorAt(i, "Invalid value. %v", err.Error())
			}
			de.NewValues = append(de.NewValues, v)
			state = NEW
		default:
			errorAt(i, "Unexpected %v.", dl[0])
		}
	}
	if state == AT {
		// @ is not a valid terminal state.
		return errorAt(len(diffLines), "Unexpected end of diff. Expecting - or +.")
	}
	if state != INIT {
		// Save the last diff element.
		// Empty string diff is valid so state could be INIT
		errString := checkDiffElement(de)
		if errString != "" {
			return errorAt(len(diffLines), errString)
		}
		diff = append(diff, de)
	}
	return diff, nil
}

func checkDiffElement(de DiffElement) string {
	if len(de.NewValues) > 1 || len(de.OldValues) > 1 {
		// Must be a set.
		emptyObject, _ := NewJsonNode(map[string]interface{}{})
		if len(de.Path) == 0 || !de.Path[len(de.Path)-1].Equals(emptyObject) {
			return "expected path to end with {} for sets."
		}
	}
	return ""
}

func errorAt(lineZeroIndex int, err string, i ...interface{}) (Diff, error) {
	line := lineZeroIndex + 1
	e := fmt.Sprintf(err, i...)
	return nil, fmt.Errorf("invalid diff at line %v. %v", line, e)
}

// ReadPatchFile reads a JSON Patch (RFC 6902) from a file. It is subject
// to the same restrictions as ReadPatchString.
func ReadPatchFile(filename string) (Diff, error) {
	bytes, err := ioutil.ReadFile(filename)
	if err != nil {
		return nil, err
	}
	return ReadPatchString(string(bytes))
}

// ReadPatchString reads a JSON Patch (RFC 6902) from a
// string. ReadPatchString supports a subset of the specification and
// requires a sequence of "test", "remove", "add" operations which mimics
// the strict patching strategy of a native jd patch.
//
// For example:
//
//	[
//	  {"op":"test","path":"/foo","value":"bar"},
//	  {"op":"remove","path":"/foo","value":"bar"},
//	  {"op":"add","path":"/foo","value":"baz"}
//	]
func ReadPatchString(s string) (Diff, error) {
	var patch []patchElement
	err := json.Unmarshal([]byte(s), &patch)
	if err != nil {
		return nil, err
	}
	var diff Diff
	if len(patch) == 0 {
		return diff, nil
	}
	var element DiffElement
	for {
		if len(patch) == 0 {
			return diff, nil
		}
		element, patch, err = readPatchDiffElement(patch)
		if err != nil {
			return nil, err
		}
		diff = append(diff, element)
	}
}

func readPatchDiffElement(patch []patchElement) (DiffElement, []patchElement, error) {
	d := DiffElement{}
	if len(patch) == 0 {
		return d, nil, fmt.Errorf("unexpected end of JSON Patch")
	}
	p := patch[0]
	var err error
	switch p.Op {
	case "test":
		// Read path.
		d.Path, err = readPointer(p.Path)
		if err != nil {
			return d, nil, err
		}
		// Read value to test and remove.
		testValue, err := NewJsonNode(p.Value)
		if err != nil {
			return d, nil, err
		}
		d.OldValues = []JsonNode{testValue}
		patch = patch[1:]
		// Validate test and remove are paired because jd remove is strict.
		if len(patch) == 0 || patch[0].Op != "remove" {
			return d, nil, fmt.Errorf("JSON Patch test op must be followed by a remove op")
		}
		if patch[0].Path != p.Path {
			return d, nil, fmt.Errorf("JSON Patch remove op must have the same path as test op")
		}
		removeValue, err := NewJsonNode(patch[0].Value)
		if err != nil {
			return d, nil, err
		}
		if !testValue.Equals(removeValue) {
			return d, nil, fmt.Errorf("JSON Patch remove op must have the same value as test op")
		}
		return d, patch[1:], nil
	case "add":
		d.Path, err = readPointer(p.Path)
		if err != nil {
			return d, nil, err
		}
		addValue, err := NewJsonNode(p.Value)
		if err != nil {
			return d, nil, err
		}
		d.NewValues = []JsonNode{addValue}
		return d, patch[1:], nil
	default:
		return d, nil, fmt.Errorf("invalid JSON Patch: must be test/remove or add ops")
	}
}

// ReadMergeFile reads a JSON Merge Patch (RFC 7386) from a file.
func ReadMergeFile(filename string) (Diff, error) {
	bytes, err := ioutil.ReadFile(filename)
	if err != nil {
		return nil, err
	}
	return ReadMergeString(string(bytes))
}

// ReadMergeString reads a JSON Merge Patch (RFC 7386) from a string.
func ReadMergeString(s string) (Diff, error) {
	n, err := ReadJsonString(s)
	if err != nil {
		return nil, err
	}
	d := Diff{}
	if n.Equals(jsonObject{}) {
		return d, nil
	}
	p := []JsonNode{jsonArray{jsonString(MERGE.string())}}
	return readMergeInto(d, p, n), nil
}

func readMergeInto(d Diff, p path, n JsonNode) Diff {
	switch n := n.(type) {
	case jsonObject:
		for k, v := range n {
			d = readMergeInto(d, append(p.clone(), jsonString(k)), v)
		}
		if len(n) == 0 {
			return append(d, DiffElement{
				Path:      p.clone(),
				NewValues: []JsonNode{newJsonObject()},
			})
		}
	case voidNode:
		return d
	default:
		if isNull(n) {
			n = voidNode{}
		}
		return append(d, DiffElement{
			Path:      p.clone(),
			NewValues: []JsonNode{n},
		})
	}
	return d
}
