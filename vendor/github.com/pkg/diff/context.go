package diff

import "fmt"

// WithContextSize returns an edit script preserving only n common elements of context for changes.
// The returned edit script may alias the input.
// If n is negative, WithContextSize panics.
// To generate a "unified diff", use WithContextSize and then WriteUnified the resulting edit script.
func (e EditScript) WithContextSize(n int) EditScript {
	if n < 0 {
		panic(fmt.Sprintf("EditScript.WithContextSize called with negative n: %d", n))
	}

	// Handle small scripts.
	switch len(e.IndexRanges) {
	case 0:
		return EditScript{}
	case 1:
		if e.IndexRanges[0].IsEqual() {
			// Entirely identical contents.
			// Unclear what to do here. For now, just bail.
			// TODO: something else? what does command line diff do?
			return EditScript{}
		}
		return scriptWithIndexRanges(e.IndexRanges[0])
	}

	out := make([]IndexRanges, 0, len(e.IndexRanges))
	for i, seg := range e.IndexRanges {
		if !seg.IsEqual() {
			out = append(out, seg)
			continue
		}
		if i == 0 {
			// Leading IndexRanges. Keep only the final n entries.
			if seg.len() > n {
				seg = indexRangesLastN(seg, n)
			}
			out = append(out, seg)
			continue
		}
		if i == len(e.IndexRanges)-1 {
			// Trailing IndexRanges. Keep only the first n entries.
			if seg.len() > n {
				seg = indexRangesFirstN(seg, n)
			}
			out = append(out, seg)
			continue
		}
		if seg.len() <= n*2 {
			// Small middle IndexRanges. Keep unchanged.
			out = append(out, seg)
			continue
		}
		// Large middle IndexRanges. Break into two disjoint parts.
		out = append(out, indexRangesFirstN(seg, n), indexRangesLastN(seg, n))
	}

	// TODO: Stock macOS diff also trims common blank lines
	// from the beginning/end of eq IndexRangess.
	// Perhaps we should do that here too.
	// Or perhaps that should be a separate, composable EditScript method?
	return EditScript{IndexRanges: out}
}

func indexRangesFirstN(seg IndexRanges, n int) IndexRanges {
	if !seg.IsEqual() {
		panic("indexRangesFirstN bad op")
	}
	if seg.len() < n {
		panic("indexRangesFirstN bad Len")
	}
	return IndexRanges{
		LowA: seg.LowA, HighA: seg.LowA + n,
		LowB: seg.LowB, HighB: seg.LowB + n,
	}
}

func indexRangesLastN(seg IndexRanges, n int) IndexRanges {
	if !seg.IsEqual() {
		panic("indexRangesLastN bad op")
	}
	if seg.len() < n {
		panic("indexRangesLastN bad Len")
	}
	return IndexRanges{
		LowA: seg.HighA - n, HighA: seg.HighA,
		LowB: seg.HighB - n, HighB: seg.HighB,
	}
}
