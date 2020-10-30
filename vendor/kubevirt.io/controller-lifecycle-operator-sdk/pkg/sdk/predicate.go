/*
Copyright 2020 The CDI Authors.

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

package sdk

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
)

// LeaderElectionAnnotation is the annotatation used on resources for leader election
const LeaderElectionAnnotation = "control-plane.alpha.kubernetes.io/leader"

// NewIgnoreLeaderElectionPredicate returns a predicate used for ignoring leader election resources
func NewIgnoreLeaderElectionPredicate() predicate.Predicate {
	return &IgnoreWithMeta{AnnotationKeys: []string{LeaderElectionAnnotation}}
}

// IgnoreWithMeta ignores resources with specified labels/annotations
type IgnoreWithMeta struct {
	LabelKeys      []string
	AnnotationKeys []string
}

// Create implements Predicate
func (p *IgnoreWithMeta) Create(e event.CreateEvent) bool {
	return p.check(e.Meta)
}

// Delete implements Predicate
func (p *IgnoreWithMeta) Delete(e event.DeleteEvent) bool {
	return p.check(e.Meta)
}

// Update implements Predicate
func (p *IgnoreWithMeta) Update(e event.UpdateEvent) bool {
	return p.check(e.MetaNew)
}

// Generic implements Predicate
func (p *IgnoreWithMeta) Generic(e event.GenericEvent) bool {
	return p.check(e.Meta)
}

func (p *IgnoreWithMeta) check(o metav1.Object) bool {
	if o != nil {
		if checkKeys(o.GetLabels(), p.LabelKeys) {
			return false
		}
		if checkKeys(o.GetAnnotations(), p.AnnotationKeys) {
			return false
		}
	}
	return true
}

func checkKeys(m map[string]string, keys []string) bool {
	for _, k := range keys {
		_, ok := m[k]
		if ok {
			return true
		}
	}
	return false
}
