package diff

import (
	"bytes"
	"fmt"
	"io"
)

// A Pair is two things that can be diffed using the Myers diff algorithm.
// A is the initial state; B is the final state.
type Pair interface {
	// LenA returns the number of initial elements.
	LenA() int
	// LenA returns the number of final elements.
	LenB() int
	// Equal reports whether the ai'th element of A is equal to the bi'th element of B.
	Equal(ai, bi int) bool
}

// A WriterTo type supports writing a diff, element by element.
// A is the initial state; B is the final state.
type WriterTo interface {
	// WriteATo writes the element a[ai] to w.
	WriteATo(w io.Writer, ai int) (int, error)
	// WriteBTo writes the element b[bi] to w.
	WriteBTo(w io.Writer, bi int) (int, error)
}

// PairWriterTo is the union of Pair and WriterTo.
type PairWriterTo interface {
	Pair
	WriterTo
}

// TODO: consider adding a StringIntern type, something like:
//
// type StringIntern struct {
// 	s map[string]*string
// }
//
// func (i *StringIntern) Bytes(b []byte) *string
// func (i *StringIntern) String(s string) *string
//
// And document what it is and why to use it.
// And consider adding helper functions to Strings and Bytes to use it.
// The reason to use it is that a lot of the execution time in diffing
// (which is an expensive operation) is taken up doing string comparisons.
// If you have paid the O(n) cost to intern all strings involved in both A and B,
// then string comparisons are reduced to cheap pointer comparisons.

// An op is a edit operation used to transform A into B.
type op int8

//go:generate stringer -type op

const (
	del op = -1
	eq  op = 0
	ins op = 1
)

// IndexRanges represents a pair of clopen index ranges.
// They represent elements A[LowA:HighA] and B[LowB:HighB].
type IndexRanges struct {
	LowA, HighA int
	LowB, HighB int
}

// IsInsert reports whether r represents an insertion in an EditScript.
// If so, the inserted elements are B[LowB:HighB].
func (r *IndexRanges) IsInsert() bool {
	return r.LowA == r.HighA
}

// IsDelete reports whether r represents a deletion in an EditScript.
// If so, the deleted elements are A[LowA:HighA].
func (r *IndexRanges) IsDelete() bool {
	return r.LowB == r.HighB
}

// IsEqual reports whether r represents a series of equal elements in an EditScript.
// If so, the elements A[LowA:HighA] are equal to the elements B[LowB:HighB].
func (r *IndexRanges) IsEqual() bool {
	return r.HighB-r.LowB == r.HighA-r.LowA
}

func (r *IndexRanges) op() op {
	if r.IsInsert() {
		return ins
	}
	if r.IsDelete() {
		return del
	}
	if r.IsEqual() {
		return eq
	}
	panic("malformed IndexRanges")
}

func (s IndexRanges) debugString() string {
	// This output is helpful when hacking on a Myers diff.
	// In other contexts it is usually more natural to group LowA, HighA and LowB, HighB.
	return fmt.Sprintf("(%d, %d) -- %s %d --> (%d, %d)", s.LowA, s.LowB, s.op(), s.len(), s.HighA, s.HighB)
}

func (s IndexRanges) len() int {
	if s.LowA == s.HighA {
		return s.HighB - s.LowB
	}
	return s.HighA - s.LowA
}

// An EditScript is an edit script to alter A into B.
type EditScript struct {
	IndexRanges []IndexRanges
}

// IsIdentity reports whether e is the identity edit script, that is, whether A and B are identical.
// See the TestHelper example.
func (e EditScript) IsIdentity() bool {
	for _, seg := range e.IndexRanges {
		if !seg.IsEqual() {
			return false
		}
	}
	return true
}

// Stat reports the number of insertions and deletions in e.
func (e EditScript) Stat() (ins, del int) {
	for _, r := range e.IndexRanges {
		switch {
		case r.IsDelete():
			del += r.HighA - r.LowA
		case r.IsInsert():
			ins += r.HighB - r.LowB
		}
	}
	return ins, del
}

// TODO: consider adding an "it just works" test helper that accepts two slices (via interface{}),
// diffs them using Strings or Bytes or Slices (using reflect.DeepEqual) as appropriate,
// and calls t.Errorf with a generated diff if they're not equal.

// scriptWithIndexRanges returns an EditScript containing s.
// It is used to reduce line noise.
func scriptWithIndexRanges(s ...IndexRanges) EditScript {
	return EditScript{IndexRanges: s}
}

// dump formats s for debugging.
func (e EditScript) dump() string {
	buf := new(bytes.Buffer)
	for _, seg := range e.IndexRanges {
		fmt.Fprintln(buf, seg)
	}
	return buf.String()
}
