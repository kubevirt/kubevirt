package diff

import (
	"bytes"
	"fmt"
	"io"
	"reflect"
)

// Strings returns a PairWriterTo that can diff and write a and b.
func Strings(a, b []string) PairWriterTo {
	return &diffStrings{a: a, b: b}
}

type diffStrings struct {
	a, b []string
}

func (ab *diffStrings) LenA() int                                { return len(ab.a) }
func (ab *diffStrings) LenB() int                                { return len(ab.b) }
func (ab *diffStrings) Equal(ai, bi int) bool                    { return ab.a[ai] == ab.b[bi] }
func (ab *diffStrings) WriteATo(w io.Writer, i int) (int, error) { return io.WriteString(w, ab.a[i]) }
func (ab *diffStrings) WriteBTo(w io.Writer, i int) (int, error) { return io.WriteString(w, ab.b[i]) }

// Bytes returns a PairWriterTo that can diff and write a and b.
func Bytes(a, b [][]byte) PairWriterTo {
	return &diffBytes{a: a, b: b}
}

type diffBytes struct {
	a, b [][]byte
}

func (ab *diffBytes) LenA() int                                { return len(ab.a) }
func (ab *diffBytes) LenB() int                                { return len(ab.b) }
func (ab *diffBytes) Equal(ai, bi int) bool                    { return bytes.Equal(ab.a[ai], ab.b[bi]) }
func (ab *diffBytes) WriteATo(w io.Writer, i int) (int, error) { return w.Write(ab.a[i]) }
func (ab *diffBytes) WriteBTo(w io.Writer, i int) (int, error) { return w.Write(ab.b[i]) }

// Slices returns a PairWriterTo that diffs a and b.
// It uses fmt.Print to print the elements of a and b.
// It uses equal to compare elements of a and b;
// if equal is nil, Slices uses reflect.DeepEqual.
func Slices(a, b interface{}, equal func(x, y interface{}) bool) PairWriterTo {
	if equal == nil {
		equal = reflect.DeepEqual
	}
	ab := &diffSlices{a: reflect.ValueOf(a), b: reflect.ValueOf(b), eq: equal}
	if ab.a.Type().Kind() != reflect.Slice || ab.b.Type().Kind() != reflect.Slice {
		panic(fmt.Errorf("diff.Slices called with a non-slice argument: %T, %T", a, b))
	}
	return ab
}

type diffSlices struct {
	a, b reflect.Value
	eq   func(x, y interface{}) bool
}

func (ab *diffSlices) LenA() int                                { return ab.a.Len() }
func (ab *diffSlices) LenB() int                                { return ab.b.Len() }
func (ab *diffSlices) atA(i int) interface{}                    { return ab.a.Index(i).Interface() }
func (ab *diffSlices) atB(i int) interface{}                    { return ab.b.Index(i).Interface() }
func (ab *diffSlices) Equal(ai, bi int) bool                    { return ab.eq(ab.atA(ai), ab.atB(bi)) }
func (ab *diffSlices) WriteATo(w io.Writer, i int) (int, error) { return fmt.Fprint(w, ab.atA(i)) }
func (ab *diffSlices) WriteBTo(w io.Writer, i int) (int, error) { return fmt.Fprint(w, ab.atB(i)) }

// TODO: consider adding a LargeFile wrapper.
// It should read each file once, storing the location of all newlines in each file,
// probably using a compact, delta-based encoding.
// Then Seek/ReadAt to read each line lazily as needed, relying on the OS page cache for performance.
// This will allow diffing giant files with low memory use, at a significant time cost.
// An alternative is to mmap the files, although this is OS-specific and can be fiddly.
