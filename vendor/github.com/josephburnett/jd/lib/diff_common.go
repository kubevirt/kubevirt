package jd

func diff(
	a, b JsonNode,
	p path,
	metadata []Metadata,
	strategy patchStrategy,
) Diff {
	d := make(Diff, 0)
	if a.Equals(b, metadata...) {
		return d
	}
	var de DiffElement
	switch strategy {
	case mergePatchStrategy:
		de = DiffElement{
			Path:      p.prependMetadataMerge(),
			NewValues: jsonArray{b},
		}
	default:
		de = DiffElement{
			Path:      p.clone(),
			OldValues: nodeList(a),
			NewValues: nodeList(b),
		}
	}
	return append(d, de)
}
