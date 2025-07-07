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
 * Copyright The KubeVirt Authors
 *
 */
package conflict

import (
	"fmt"
	"strings"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	k8sfield "k8s.io/apimachinery/pkg/util/validation/field"
)

const conflictsErrorFmt = "VM field(s) %s conflicts with selected instance type"

type Conflict struct {
	Message string
	k8sfield.Path
}

func New(name string, moreNames ...string) *Conflict {
	return &Conflict{
		Path: *k8sfield.NewPath(name, moreNames...),
	}
}

func NewFromPath(path *k8sfield.Path) *Conflict {
	return &Conflict{
		Path: *path,
	}
}

func NewWithMessage(message, name string, moreNames ...string) *Conflict {
	return &Conflict{
		Path:    *k8sfield.NewPath(name, moreNames...),
		Message: message,
	}
}

func (c Conflict) NewChild(name string, moreNames ...string) *Conflict {
	return &Conflict{
		Path: *c.Child(name, moreNames...),
	}
}

func (c Conflict) Error() string {
	if c.Message != "" {
		return c.Message
	}
	return fmt.Sprintf(conflictsErrorFmt, c.String())
}

func (c Conflict) StatusCause() metav1.StatusCause {
	return metav1.StatusCause{
		Type:    metav1.CauseTypeFieldValueInvalid,
		Message: c.Error(),
		Field:   c.String(),
	}
}

func (c Conflict) StatusCauses() []metav1.StatusCause {
	return []metav1.StatusCause{c.StatusCause()}
}

type Conflicts []*Conflict

func (c Conflicts) String() string {
	pathStrings := make([]string, 0, len(c))
	for _, path := range c {
		pathStrings = append(pathStrings, path.String())
	}
	return strings.Join(pathStrings, ", ")
}

func (c Conflicts) Error() string {
	return fmt.Sprintf(conflictsErrorFmt, c.String())
}

func (c Conflicts) StatusCauses() []metav1.StatusCause {
	causes := make([]metav1.StatusCause, 0, len(c))
	for _, conflict := range c {
		causes = append(causes, conflict.StatusCause())
	}
	return causes
}
