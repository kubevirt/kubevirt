package jd

import (
	"fmt"
	"sort"
	"strings"
)

// Metadata is a closed set of values which modify Diff and Equals
// semantics.
type Metadata interface {
	is_metadata()
	string() string
}

type setMetadata struct{}
type multisetMetadata struct{}
type setkeysMetadata struct {
	keys map[string]bool
}
type mergeMetadata struct{}
type precisionMetadata struct {
	precision float64
}

func (setMetadata) is_metadata()       {}
func (multisetMetadata) is_metadata()  {}
func (setkeysMetadata) is_metadata()   {}
func (mergeMetadata) is_metadata()     {}
func (precisionMetadata) is_metadata() {}

func (m setMetadata) string() string {
	return "set"
}

func (m multisetMetadata) string() string {
	return "multiset"
}

func (m setkeysMetadata) string() string {
	ks := make([]string, 0)
	for k := range m.keys {
		ks = append(ks, k)
	}
	sort.Strings(ks)
	// TODO: escape commas.
	return "setkeys=" + strings.Join(ks, ",")
}

func (m mergeMetadata) string() string {
	// Merge apply to the whole path not just the next
	// element. Therefore it is all caps.
	return "MERGE"
}

func (m precisionMetadata) string() string {
	return "precision=" + fmt.Sprintf("%f", m.precision)
}

var (
	// MULTISET interprets all Arrays as Multisets (bags) during Diff
	// and Equals operations.
	MULTISET Metadata = multisetMetadata{}
	// SET interprets all Arrays as Sets during Diff and Equals
	// operations.
	SET Metadata = setMetadata{}
	// MERGE produces a Diff with merge semantics (RFC 7386).
	MERGE Metadata = mergeMetadata{}
)

// SetKeys constructs Metadata to identify unique objects in an Array for
// deeper Diff and Patch operations.
func Setkeys(keys ...string) Metadata {
	m := setkeysMetadata{
		keys: make(map[string]bool),
	}
	for _, key := range keys {
		m.keys[key] = true
	}
	return m
}

func SetPrecision(precision float64) Metadata {
	return precisionMetadata{precision}
}

type patchStrategy string

const (
	mergePatchStrategy  patchStrategy = "merge"
	strictPatchStrategy patchStrategy = "strict"
)

func getPatchStrategy(metadata []Metadata) patchStrategy {
	if checkMetadata(MERGE, metadata) {
		return mergePatchStrategy
	}
	return strictPatchStrategy
}

func getPrecision(metadata []Metadata) float64 {
	for _, o := range metadata {
		if s, ok := o.(precisionMetadata); ok {
			return s.precision
		}
	}
	return 0
}

func dispatch(n JsonNode, metadata []Metadata) JsonNode {
	switch n := n.(type) {
	case jsonArray:
		if checkMetadata(SET, metadata) {
			return jsonSet(n)
		}
		if checkMetadata(MULTISET, metadata) {
			return jsonMultiset(n)
		}
		return jsonList(n)
	}
	return n
}

func dispatchRenderOptions(n JsonNode, opts []RenderOption) JsonNode {
	metadata := []Metadata{}
	for _, o := range opts {
		if m, ok := o.(Metadata); ok {
			metadata = append(metadata, m)
		}
	}
	return dispatch(n, metadata)
}

func checkMetadata(want Metadata, metadata []Metadata) bool {
	for _, o := range metadata {
		if o == want {
			return true
		}
	}
	return false
}

func getSetkeysMetadata(metadata []Metadata) *setkeysMetadata {
	for _, o := range metadata {
		if s, ok := o.(setkeysMetadata); ok {
			return &s
		}
	}
	return nil
}
