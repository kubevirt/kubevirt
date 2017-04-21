package convert

import (
	"bufio"
	"bytes"
	"io"
	"unicode"

	"k8s.io/client-go/pkg/util/yaml"
)

// Guess if the provided reader represents a XML or JSON stream.
// If neither is true, YAML is assumed.
func GuessStreamType(source io.Reader, size int) (io.Reader, Type) {
	if r, _, isXML := guessXMLStream(source, size); isXML {
		return r, XML
	} else if r, _, isJSON := yaml.GuessJSONStream(r, size); isJSON {
		return r, JSON
	} else {
		return r, YAML
	}
}

func guessXMLStream(r io.Reader, size int) (io.Reader, []byte, bool) {
	buffer := bufio.NewReaderSize(r, size)
	b, _ := buffer.Peek(size)
	return buffer, b, hasXMLPrefix(b)
}

var xmlPrefix = []byte("<")

func hasXMLPrefix(buf []byte) bool {
	trim := bytes.TrimLeftFunc(buf, unicode.IsSpace)
	return bytes.HasPrefix(trim, xmlPrefix)
}
