package jd

// DiffElement (hunk) is a way in which two JsonNodes differ at a given
// Path. OldValues can be removed and NewValues can be added. The exact
// Path and how to interpret the intervening structure is determined by a
// list of JsonNodes (path elements).
type DiffElement struct {

	// Path elements can be strings to index Objects, numbers to
	// index Lists and objects to index Sets and Multisets. Path
	// elements can be preceeded by an Array of Metadata strings to
	// change how the structure is interpretted. Metadata in lower
	// case applies to the following path element. Metadata in upper
	// case applies to the rest of the path.
	//
	// For example:
	//   ["foo","bar"]               // indexes to 1 in {"foo":{"bar":1}}
	//   ["foo",0]                   // indexes to 1 in {"foo":[1]}
	//   ["foo",{}]                  // indexes a set under "foo" in {"foo":[1]}
	//   ["foo",["multiset"],{}]     // indexes a multiset under "foo" in {"foo":[1,1]}
	//   ["foo",{"id":"bar"},"baz"]  // indexes to 1 in {"foo":[{"id":"bar","baz":1}]}
	//   [["MERGE"],"foo","bar"]     // indexes to 1 in {"foo":{"bar":1}} with merge semantics
	Path []JsonNode

	// OldValues are removed from the JsonNode at the Path. Usually
	// only one old value is provided unless removing entries from a
	// Set or Multiset. When using merge semantics no old values are
	// provided (new values stomp old ones).
	OldValues []JsonNode

	// NewValues are added to the JsonNode at the Path. Usually only
	// one new value is provided unless adding entries to a Set or
	// Multiset.
	NewValues []JsonNode
}

// Diff describes how two JsonNodes differ from each other. A Diff is
// composed of DiffElements (hunks) which describe a difference at a
// given Path. Each hunk stands alone with all necessary Metadata
// embedded in the Path, so a Diff rendered in native jd format can
// easily be edited by hand. The elements of a Diff can be applied to
// a JsonNode by the Patch method.
type Diff []DiffElement

// JSON Patch (RFC 6902)
type patchElement struct {
	Op    string      `json:"op"`   // "add", "test" or "remove"
	Path  string      `json:"path"` // JSON Pointer (RFC 6901)
	Value interface{} `json:"value"`
}
