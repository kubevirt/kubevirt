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
	"os"
	"strconv"
	"strings"

	"k8s.io/apimachinery/pkg/api/resource"
)

// Lookup returns the trimmed value of key and whether it is set to a non-empty string.
func Lookup(key string) (string, bool) {
	value := strings.TrimSpace(os.Getenv(key))
	if value == "" {
		return "", false
	}
	return value, true
}

// Uint64 parses key as a uint64 when set.
func Uint64(key string) (uint64, bool) {
	value, ok := Lookup(key)
	if !ok {
		return 0, false
	}
	parsed, err := strconv.ParseUint(value, 10, 64)
	if err != nil {
		return 0, false
	}
	return parsed, true
}

// Int64 parses key as an int64 when set.
func Int64(key string) (int64, bool) {
	value, ok := Lookup(key)
	if !ok {
		return 0, false
	}
	parsed, err := strconv.ParseInt(value, 10, 64)
	if err != nil {
		return 0, false
	}
	return parsed, true
}

// Bool parses key as a bool when set.
func Bool(key string) (bool, bool) {
	value, ok := Lookup(key)
	if !ok {
		return false, false
	}
	parsed, err := strconv.ParseBool(value)
	if err != nil {
		return false, false
	}
	return parsed, true
}

// Quantity parses key as a resource.Quantity when set.
func Quantity(key string) (resource.Quantity, bool) {
	value, ok := Lookup(key)
	if !ok {
		return resource.Quantity{}, false
	}
	parsed, err := resource.ParseQuantity(value)
	if err != nil {
		return resource.Quantity{}, false
	}
	return parsed, true
}
