/*
 * This file is part of the kubevirt project
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 *
 * Copyright 2017 Red Hat, Inc.
 *
 */

package convert

import (
	"bufio"
	"bytes"
	"io"
	"unicode"

	"k8s.io/apimachinery/pkg/util/yaml"
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
