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

package cgroup

import (
	"os"

	"kubevirt.io/kubevirt/pkg/safepath"
)

type safePath interface {
	JoinNoFollow(path string) (safePath, error)
	StatAtNoFollow() (os.FileInfo, error)
	ExecuteNoFollow(callback func(safePath string) error) error
}

type realSafePath struct {
	path *safepath.Path
}

func (r *realSafePath) JoinNoFollow(path string) (safePath, error) {
	p, err := safepath.JoinNoFollow(r.path, path)
	if err != nil {
		return nil, err
	}
	return &realSafePath{path: p}, nil
}

func (r *realSafePath) StatAtNoFollow() (os.FileInfo, error) {
	return safepath.StatAtNoFollow(r.path)
}

func (r *realSafePath) ExecuteNoFollow(callback func(safePath string) error) error {
	return r.path.ExecuteNoFollow(callback)
}
