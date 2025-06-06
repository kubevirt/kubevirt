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

package passt

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"kubevirt.io/client-go/log"
)

const passtRepairSocketSuffix = ".socket.repair"

// CreateShortenedSymlink: The full path of the socket is longet than the linux limit (108)
// create a symlink to provide passt-repair a shortened path.
// Input: /pods/289c1d09-9552-46ba-ba87-11c5c00692f3/volumes/kubernetes.io~empty-dir/libvirt-runtime/qemu/run/passt
// Output: /pods/289c1d09-9552-46ba-ba87-11c5c00692f3/p
func CreateShortenedSymlink(inputPath, baseDir string) (string, error) {
	cleanedPath := filepath.Clean(inputPath)
	relPath := strings.TrimPrefix(cleanedPath, baseDir)
	segments := strings.Split(relPath, string(filepath.Separator))

	if len(segments) < 2 {
		return "", fmt.Errorf("input path '%s' too short for symlink", inputPath)
	}

	symlinkPath := filepath.Join(baseDir, segments[0], segments[1], "p")

	if err := os.MkdirAll(filepath.Dir(symlinkPath), 0755); err != nil {
		return "", fmt.Errorf("failed to create symlink parent directories: %w", err)
	}

	if err := os.Symlink(inputPath, symlinkPath); err != nil {
		if os.IsExist(err) {
			if existingTarget, linkErr := os.Readlink(symlinkPath); linkErr == nil && existingTarget == inputPath {
				return symlinkPath, nil
			}
		}
		return "", fmt.Errorf("failed to create symbolic link from '%s' to '%s': %w", inputPath, symlinkPath, err)
	}

	return symlinkPath, nil
}

func FindRepairSocketInDir(dirPath string) (string, error) {
	entries, err := os.ReadDir(dirPath)
	if err != nil {
		return "", fmt.Errorf("failed to read directory %s: %w", dirPath, err)
	}

	log.Log.V(4).Infof("listing directory %s", dirPath)
	for _, entry := range entries {
		log.Log.V(4).Info(getFileDetails(entry))
		fileName := entry.Name()
		if strings.HasSuffix(fileName, passtRepairSocketSuffix) {
			fullPath := filepath.Join(dirPath, fileName)
			return fullPath, nil
		}
	}
	return "", nil
}

func getFileDetails(fileName os.DirEntry) string {
	info, err := fileName.Info()
	if err != nil {
		return err.Error()
	}
	modTime := info.ModTime().Format("Jan _2 15:04")
	mode := info.Mode()
	size := info.Size()
	name := fileName.Name()
	return fmt.Sprintf("%s %10d %s %s", mode, size, modTime, name)
}
