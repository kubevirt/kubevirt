package util

import (
	"errors"
	"fmt"
	"strings"
)

// ExtractEmbeddedMap can extract in a string embedded maps which have the form `val :="key1=val1;key2=val2"`
func ExtractEmbeddedMap(header string) (map[string]string, error) {
	if header == "" {
		return nil, nil
	}
	headerMap := map[string]string{}
	pairs := strings.Split(header, ";")
	for _, p := range pairs {
		p = strings.TrimSpace(p)
		kvs := strings.Split(p, "=")
		if len(kvs) != 2 || kvs[1] == "" {
			return nil, errors.New(fmt.Sprintf("The pair %s of the supplied string %s is not valid", p, header))
		}
		key := strings.TrimSpace(kvs[0])
		value := strings.TrimSpace(kvs[1])
		headerMap[key] = value
	}
	return headerMap, nil
}
