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

package driver

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path/filepath"

	"github.com/opencontainers/selinux/go-selinux"
	specs "tags.cncf.io/container-device-interface/specs-go"
)

const (
	baseDir       = "/var/run/kubevirt/dra"
	cdiDir        = "/var/run/cdi"
	cdiVendor     = "kubevirt.io"
	cdiClass      = "hostpath"
	containerPath = "/var/run/kubevirt/dra/hostpath"
	qemuUID       = 107
	qemuGID       = 107
)

func prepareHostpath(claimName string) (string, error) {
	path := filepath.Join(baseDir, claimName)
	if err := os.MkdirAll(path, 0o775); err != nil {
		return "", fmt.Errorf("failed to create directory %s: %w", path, err)
	}
	if err := os.Chown(path, qemuUID, qemuGID); err != nil {
		return "", fmt.Errorf("failed to chown %s: %w", path, err)
	}
	if err := selinux.SetFileLabel(path, "system_u:object_r:container_file_t:s0"); err != nil {
		return "", fmt.Errorf("failed to set SELinux label on %s: %w", path, err)
	}
	log.Printf("Created directory: %s", path)

	return createCDISpec(claimName, path)
}

func createCDISpec(claimName, path string) (string, error) {
	spec := specs.Spec{
		Version: "0.5.0",
		Kind:    cdiVendor + "/" + cdiClass,
		Devices: []specs.Device{{
			Name: claimName,
			ContainerEdits: specs.ContainerEdits{
				Env: []string{fmt.Sprintf("KUBEVIRT_HOSTPATH_MOUNTPOINT=%s", containerPath)},
				Mounts: []*specs.Mount{{
					HostPath:      path,
					ContainerPath: containerPath,
					Options:       []string{"rbind"},
				}},
			},
		}},
	}

	if err := os.MkdirAll(cdiDir, 0o755); err != nil {
		return "", fmt.Errorf("failed to create CDI dir %s: %w", cdiDir, err)
	}
	specPath := fmt.Sprintf("%s/%s-%s.json", cdiDir, cdiVendor, cdiClass)
	data, err := json.MarshalIndent(spec, "", "  ")
	if err != nil {
		return "", fmt.Errorf("failed to marshal CDI spec: %w", err)
	}
	if err := os.WriteFile(specPath, data, 0o644); err != nil {
		return "", fmt.Errorf("failed to write CDI spec %s: %w", specPath, err)
	}
	log.Printf("Created CDI spec: %s", specPath)

	return cdiVendor + "/" + cdiClass + "=" + claimName, nil
}

func unprepareHostpath(claimName string) {
	path := filepath.Join(baseDir, claimName)
	os.RemoveAll(path)
	log.Printf("Removed directory: %s", path)

	specPath := fmt.Sprintf("%s/%s-%s.json", cdiDir, cdiVendor, cdiClass)
	os.Remove(specPath)
	log.Printf("Removed CDI spec: %s", specPath)
}
