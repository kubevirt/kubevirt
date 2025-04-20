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
 */

package unsafepath

import "path/filepath"

type Path struct {
	rootBase     string
	relativePath string
}

func New(rootBase string, relativePath string) *Path {
	return &Path{
		rootBase:     rootBase,
		relativePath: relativePath,
	}
}

func UnsafeAbsolute(path *Path) string {
	return filepath.Join(path.rootBase, path.relativePath)
}

func UnsafeRelative(path *Path) string {
	return path.relativePath
}

func UnsafeRoot(path *Path) string {
	return path.rootBase
}
