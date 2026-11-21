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

package diskpaths_test

import (
	"os"
	"path/filepath"
	"testing"

	"kubevirt.io/kubevirt/pkg/storage/container-disk/diskpaths"
)

func TestSetLocalDirectoryDoesNotCreateDirectory(t *testing.T) {
	originalPath := diskpaths.GetMountBaseDir()
	t.Cleanup(func() {
		diskpaths.SetLocalDirectory(originalPath)
	})

	path := filepath.Join(t.TempDir(), "container-disks")
	diskpaths.SetLocalDirectory(path)

	if got := diskpaths.GetMountBaseDir(); got != path {
		t.Fatalf("expected mount base directory %q, got %q", path, got)
	}
	if _, err := os.Stat(path); !os.IsNotExist(err) {
		t.Fatalf("expected %q to remain absent, got error %v", path, err)
	}
}

func TestSetKubeletPodsDirectoryDoesNotCreateDirectory(t *testing.T) {
	originalPath := diskpaths.GetPodsBaseDir()
	t.Cleanup(func() {
		diskpaths.SetKubeletPodsDirectory(originalPath)
	})

	path := filepath.Join(t.TempDir(), "pods")
	diskpaths.SetKubeletPodsDirectory(path)

	if got := diskpaths.GetPodsBaseDir(); got != path {
		t.Fatalf("expected kubelet pods base directory %q, got %q", path, got)
	}
	if _, err := os.Stat(path); !os.IsNotExist(err) {
		t.Fatalf("expected %q to remain absent, got error %v", path, err)
	}
}
