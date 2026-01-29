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

package virtiofs

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	k8sv1 "k8s.io/api/core/v1"

	v1 "kubevirt.io/api/core/v1"
)

// ContainerPathVirtiofsContainerName returns the expected container name for a containerPath volume
func ContainerPathVirtiofsContainerName(volumeName string) string {
	return fmt.Sprintf("virtiofs-%s", volumeName)
}

// GetContainerPathVolumesWithFilesystems returns containerPath volumes that have a matching filesystem defined
func GetContainerPathVolumesWithFilesystems(vmi *v1.VirtualMachineInstance) []v1.Volume {
	if vmi == nil {
		return nil
	}

	// Build a set of filesystem names
	filesystemNames := make(map[string]struct{})
	for _, fs := range vmi.Spec.Domain.Devices.Filesystems {
		if fs.Virtiofs != nil {
			filesystemNames[fs.Name] = struct{}{}
		}
	}

	if len(filesystemNames) == 0 {
		return nil
	}

	// Find containerPath volumes with matching filesystems
	var containerPathVolumes []v1.Volume
	for _, volume := range vmi.Spec.Volumes {
		if volume.ContainerPath != nil {
			if _, hasFilesystem := filesystemNames[volume.Name]; hasFilesystem {
				containerPathVolumes = append(containerPathVolumes, volume)
			}
		}
	}

	return containerPathVolumes
}

// ExpectedContainerPathContainerNames returns the virtiofs container names expected for a VMI's containerPath volumes
func ExpectedContainerPathContainerNames(vmi *v1.VirtualMachineInstance) []string {
	volumes := GetContainerPathVolumesWithFilesystems(vmi)
	if len(volumes) == 0 {
		return nil
	}

	names := make([]string, 0, len(volumes))
	for _, volume := range volumes {
		names = append(names, ContainerPathVirtiofsContainerName(volume.Name))
	}
	return names
}

// MissingContainerPathContainers returns which expected virtiofs containers are missing from the pod
func MissingContainerPathContainers(vmi *v1.VirtualMachineInstance, pod *k8sv1.Pod) []string {
	expectedNames := ExpectedContainerPathContainerNames(vmi)
	if len(expectedNames) == 0 {
		return nil
	}

	// Build set of existing container names
	existingContainers := make(map[string]struct{})
	for _, container := range pod.Spec.Containers {
		existingContainers[container.Name] = struct{}{}
	}

	// Find missing containers
	var missing []string
	for _, name := range expectedNames {
		if _, exists := existingContainers[name]; !exists {
			missing = append(missing, name)
		}
	}

	return missing
}

// FindVolumeMountForPath finds the volumeMount in the container that matches the given path.
// It returns the volumeMount and the subPath within that mount, or nil if not found.
func FindVolumeMountForPath(container *k8sv1.Container, path string) (*k8sv1.VolumeMount, string) {
	var bestMatch *k8sv1.VolumeMount
	var bestMatchLen int

	for i := range container.VolumeMounts {
		mount := &container.VolumeMounts[i]
		mountPath := mount.MountPath

		// Check if the path is exactly the mount path or is under it
		if path == mountPath {
			return mount, ""
		}

		// Check if path is under this mount point
		if strings.HasPrefix(path, mountPath+"/") {
			// This mount is a candidate; prefer longer (more specific) matches
			if len(mountPath) > bestMatchLen {
				bestMatch = mount
				bestMatchLen = len(mountPath)
			}
		}
	}

	if bestMatch != nil {
		// Calculate the subpath within the mount
		subPath := strings.TrimPrefix(path, bestMatch.MountPath+"/")
		return bestMatch, subPath
	}

	return nil, ""
}

// ValidateContainerPath validates that a container path doesn't escape its mount point
// via symlinks. It resolves the path and verifies the resolved path stays within the
// mount point that contains the original path.
//
// This validation is performed at runtime in virt-launcher before starting the VM,
// as it requires access to the actual filesystem to resolve symlinks and check mounts.
func ValidateContainerPath(containerPath string) error {
	// First check the path exists
	info, err := os.Lstat(containerPath)
	if err != nil {
		return fmt.Errorf("containerPath %q does not exist: %w", containerPath, err)
	}
	if !info.IsDir() {
		return fmt.Errorf("containerPath %q is not a directory", containerPath)
	}

	// Find the mount point for this path
	mountPoint, err := findMountPoint(containerPath)
	if err != nil {
		return fmt.Errorf("failed to find mount point for %q: %w", containerPath, err)
	}

	// Resolve symlinks in the path
	resolvedPath, err := filepath.EvalSymlinks(containerPath)
	if err != nil {
		return fmt.Errorf("failed to resolve symlinks in %q: %w", containerPath, err)
	}

	// Verify the resolved path starts with the mount point
	// Use filepath.Clean to normalize paths before comparison
	cleanMountPoint := filepath.Clean(mountPoint)
	cleanResolvedPath := filepath.Clean(resolvedPath)

	if cleanResolvedPath != cleanMountPoint && !strings.HasPrefix(cleanResolvedPath, cleanMountPoint+"/") {
		return fmt.Errorf("containerPath %q resolves to %q which escapes mount point %q",
			containerPath, resolvedPath, mountPoint)
	}

	return nil
}

// findMountPoint finds the mount point for a given path by reading /proc/self/mountinfo
// and finding the longest matching mount path.
func findMountPoint(path string) (string, error) {
	// Normalize the path
	absPath, err := filepath.Abs(path)
	if err != nil {
		return "", err
	}

	file, err := os.Open("/proc/self/mountinfo")
	if err != nil {
		return "", fmt.Errorf("failed to open /proc/self/mountinfo: %w", err)
	}
	defer file.Close()

	var bestMatch string
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := scanner.Text()
		fields := strings.Fields(line)
		if len(fields) < 5 {
			continue
		}
		// mountinfo format: ID PARENT_ID MAJOR:MINOR ROOT MOUNT_POINT ...
		mountPath := fields[4]

		// Check if this mount path is a prefix of our path
		if absPath == mountPath || strings.HasPrefix(absPath, mountPath+"/") {
			// Keep the longest (most specific) match
			if len(mountPath) > len(bestMatch) {
				bestMatch = mountPath
			}
		}
	}

	if err := scanner.Err(); err != nil {
		return "", fmt.Errorf("error reading /proc/self/mountinfo: %w", err)
	}

	if bestMatch == "" {
		return "", fmt.Errorf("no mount point found for path %q", path)
	}

	return bestMatch, nil
}
