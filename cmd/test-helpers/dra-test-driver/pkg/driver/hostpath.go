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
	"k8s.io/apimachinery/pkg/types"
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

// cdiSpecPath returns the CDI spec file for a single claim. The claim UID is part
// of the file name so that concurrent claims on the same node do not overwrite each
// other's spec, and so that unpreparing one claim only removes its own file.
func cdiSpecPath(claimUID types.UID) string {
	return filepath.Join(cdiDir, fmt.Sprintf("%s-%s-%s.json", cdiVendor, cdiClass, claimUID))
}

// The claim UID, not the claim name, identifies the host directory and the CDI
// device: claim names are unique only within a namespace, and are reused across
// runs, which would let a stale spec satisfy a new claim.
func (d *Driver) prepareHostpath(claimUID types.UID) (string, error) {

	d.hostPath = filepath.Join(baseDir, string(claimUID))
	if err := os.MkdirAll(d.hostPath, 0o755); err != nil {
		return "", fmt.Errorf("failed to create directory %s: %w", d.hostPath, err)
	}
	if err := os.Chown(d.hostPath, qemuUID, qemuGID); err != nil {
		return "", fmt.Errorf("failed to chown %s: %w", d.hostPath, err)
	}
	if err := selinux.SetFileLabel(d.hostPath, "system_u:object_r:container_file_t:s0"); err != nil {
		return "", fmt.Errorf("failed to set SELinux label on %s: %w", d.hostPath, err)
	}
	log.Printf("Created directory: %s", d.hostPath)

	return createCDISpec(claimUID, d.hostPath)
}

func createCDISpec(claimUID types.UID, path string) (string, error) {
	deviceName := string(claimUID)
	spec := specs.Spec{
		Version: "0.5.0",
		Kind:    cdiVendor + "/" + cdiClass,
		Devices: []specs.Device{{
			Name: deviceName,
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
	specPath := cdiSpecPath(claimUID)
	data, err := json.MarshalIndent(spec, "", "  ")
	if err != nil {
		return "", fmt.Errorf("failed to marshal CDI spec: %w", err)
	}
	if err := os.WriteFile(specPath, data, 0o644); err != nil {
		return "", fmt.Errorf("failed to write CDI spec %s: %w", specPath, err)
	}
	log.Printf("Created CDI spec: %s", specPath)

	return cdiVendor + "/" + cdiClass + "=" + deviceName, nil
}

func unprepareHostpath(claimUID types.UID) {
	path := filepath.Join(baseDir, string(claimUID))
	os.RemoveAll(path)
	log.Printf("Removed directory: %s", path)

	specPath := cdiSpecPath(claimUID)
	os.Remove(specPath)
	log.Printf("Removed CDI spec: %s", specPath)
}
