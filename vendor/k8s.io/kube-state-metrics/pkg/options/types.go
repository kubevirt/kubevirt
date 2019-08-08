/*
Copyright 2018 The Kubernetes Authors All rights reserved.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package options

import (
	"sort"
	"strings"

	"github.com/pkg/errors"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// MetricSet represents a collection which has a unique set of metrics.
type MetricSet map[string]struct{}

func (ms *MetricSet) String() string {
	s := *ms
	ss := s.asSlice()
	sort.Strings(ss)
	return strings.Join(ss, ",")
}

// Set converts a comma-separated string of metrics into a slice and appends it to the MetricSet.
func (ms *MetricSet) Set(value string) error {
	s := *ms
	metrics := strings.Split(value, ",")
	for _, metric := range metrics {
		metric = strings.TrimSpace(metric)
		if len(metric) != 0 {
			s[metric] = struct{}{}
		}
	}
	return nil
}

// asSlice returns the MetricSet in the form of plain string slice.
func (ms MetricSet) asSlice() []string {
	metrics := []string{}
	for metric := range ms {
		metrics = append(metrics, metric)
	}
	return metrics
}

// IsEmpty returns true if the length of the MetricSet is zero.
func (ms MetricSet) IsEmpty() bool {
	return len(ms.asSlice()) == 0
}

// Type returns a descriptive string about the MetricSet type.
func (ms *MetricSet) Type() string {
	return "string"
}

// CollectorSet represents a collection which has a unique set of collectors.
type CollectorSet map[string]struct{}

func (c *CollectorSet) String() string {
	s := *c
	ss := s.AsSlice()
	sort.Strings(ss)
	return strings.Join(ss, ",")
}

// Set converts a comma-separated string of collectors into a slice and appends it to the CollectorSet.
func (c *CollectorSet) Set(value string) error {
	s := *c
	cols := strings.Split(value, ",")
	for _, col := range cols {
		col = strings.TrimSpace(col)
		if len(col) != 0 {
			_, ok := DefaultCollectors[col]
			if !ok {
				return errors.Errorf("collector \"%s\" does not exist", col)
			}
			s[col] = struct{}{}
		}
	}
	return nil
}

// AsSlice returns the Collector in the form of a plain string slice.
func (c CollectorSet) AsSlice() []string {
	cols := []string{}
	for col := range c {
		cols = append(cols, col)
	}
	return cols
}

// isEmpty() returns true if the length of the CollectorSet is zero.
func (c CollectorSet) isEmpty() bool {
	return len(c.AsSlice()) == 0
}

// Type returns a descriptive string about the CollectorSet type.
func (c *CollectorSet) Type() string {
	return "string"
}

// NamespaceList represents a list of namespaces to query forom.
type NamespaceList []string

func (n *NamespaceList) String() string {
	return strings.Join(*n, ",")
}

// IsAllNamespaces checks if the Namespace selector is that of `NamespaceAll` which is used for
// selecting or filtering across all namespaces.
func (n *NamespaceList) IsAllNamespaces() bool {
	return len(*n) == 1 && (*n)[0] == metav1.NamespaceAll
}

// Set converts a comma-separated string of namespaces into a slice and appends it to the NamespaceList
func (n *NamespaceList) Set(value string) error {
	splitNamespaces := strings.Split(value, ",")
	for _, ns := range splitNamespaces {
		ns = strings.TrimSpace(ns)
		if len(ns) != 0 {
			*n = append(*n, ns)
		}
	}
	return nil
}

// Type returns a descriptive string about the NamespaceList type.
func (n *NamespaceList) Type() string {
	return "string"
}
