/*
 * This file is part of the KubeVirt project
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
 * Copyright The KubeVirt Authors.
 *
 */

package env

import (
	"fmt"
	"os"
	"strconv"
	"strings"

	"k8s.io/apimachinery/pkg/api/resource"

	"kubevirt.io/client-go/log"
)

// Lookup returns the trimmed value of key and whether it is set to a non-empty string.
func Lookup(key string) (string, bool) {
	value := strings.TrimSpace(os.Getenv(key))
	if value == "" {
		return "", false
	}
	return value, true
}

// Parse converts value to T. Supported types are bool, int64, uint64, and resource.Quantity.
func Parse[T any](value string) (T, error) {
	var zero T
	switch any(zero).(type) {
	case bool:
		parsed, err := strconv.ParseBool(value)
		return any(parsed).(T), err
	case int64:
		parsed, err := strconv.ParseInt(value, 10, 64)
		return any(parsed).(T), err
	case uint64:
		parsed, err := strconv.ParseUint(value, 10, 64)
		return any(parsed).(T), err
	case resource.Quantity:
		parsed, err := resource.ParseQuantity(value)
		return any(parsed).(T), err
	default:
		return zero, fmt.Errorf("unsupported env type %T", zero)
	}
}

// Var is a named environment variable of type T.
type Var[T any] struct {
	Name string
}

// LoadAndParse reads the process environment and converts it to T.
// Unset, empty, and invalid values return an error; invalid values are also logged as a warning.
func (v Var[T]) LoadAndParse() (T, error) {
	var zero T
	raw, ok := Lookup(v.Name)
	if !ok {
		return zero, fmt.Errorf("%s is not set", v.Name)
	}
	parsed, err := Parse[T](raw)
	if err != nil {
		log.Log.Reason(err).Warningf("ignoring invalid env %s=%q", v.Name, raw)
		return zero, err
	}
	return parsed, nil
}

func (v Var[T]) Binding(value string) Binding {
	return Binding{
		name:  v.Name,
		value: value,
		parse: func(raw string) (any, error) {
			return Parse[T](raw)
		},
	}
}

// Binding is an Environment Variable paired with a raw string value for ConfigMap injection.
type Binding struct {
	name  string
	value string
	parse func(string) (any, error)
}

func (b Binding) Key() string { return b.name }
func (b Binding) Raw() string { return b.value }

// Parse converts the bound raw string using the same parser as LoadAndParse.
func (b Binding) Parse() (any, error) {
	value := strings.TrimSpace(b.value)
	if value == "" {
		return nil, fmt.Errorf("%s is empty or whitespace-only", b.name)
	}
	parsed, err := b.parse(value)
	if err != nil {
		return nil, fmt.Errorf("%s=%q: %w", b.name, b.value, err)
	}
	return parsed, nil
}
