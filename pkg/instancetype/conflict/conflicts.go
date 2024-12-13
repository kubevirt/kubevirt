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
	"strings"

	k8sfield "k8s.io/apimachinery/pkg/util/validation/field"
)

type Conflict struct {
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

func (c Conflict) NewChild(name string, moreNames ...string) *Conflict {
	return &Conflict{
		Path: *c.Child(name, moreNames...),
	}
}

type Conflicts []*Conflict

func (c Conflicts) String() string {
	pathStrings := make([]string, 0, len(c))
	for _, path := range c {
		pathStrings = append(pathStrings, path.String())
	}
	return strings.Join(pathStrings, ", ")
}
