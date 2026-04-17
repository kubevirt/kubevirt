package jd

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/go-openapi/jsonpointer"
)

func readPointer(s string) ([]JsonNode, error) {
	pointer, err := jsonpointer.New(s)
	if err != nil {
		return nil, err
	}
	tokens := pointer.DecodedTokens()
	path := make([]JsonNode, len(tokens))
	for i, t := range tokens {
		var element JsonNode
		var err error
		if _, err := strconv.Atoi(t); err == nil {
			// Wait to decide if we use this token as a string or integer.
			element = jsonStringOrInteger(t)
		} else {
			element, err = NewJsonNode(t)
		}
		if err != nil {
			return nil, err
		}
		if s, ok := element.(jsonString); ok && s == "-" {
			element, _ = NewJsonNode(-1)
		}
		path[i] = element
	}
	return path, nil
}

func writePointer(path []JsonNode) (string, error) {
	var b strings.Builder
	for _, element := range path {
		b.WriteString("/")
		switch e := element.(type) {
		case jsonNumber:
			if int(e) == -1 {
				b.WriteString("-")
			} else {
				b.WriteString(jsonpointer.Escape(strconv.Itoa(int(e))))
			}
		case jsonString:
			if string(e) == "-" {
				return "", fmt.Errorf("JSON Pointer does not support object key '-'")
			}
			s := jsonpointer.Escape(string(e))
			b.WriteString(s)
		case jsonStringOrInteger:
			b.WriteString(string(e))
		case jsonArray:
			return "", fmt.Errorf("JSON Pointer does not support jd metadata")
		default:
			return "", fmt.Errorf("unsupported type: %T", e)
		}
	}
	return b.String(), nil
}
