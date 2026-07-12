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
)

const (
	baseDir       = "/var/run/kubevirt/dra"
	cdiDir        = "/var/run/cdi"
	cdiVendor     = "kubevirt.io"
	cdiClass      = "hostpath"
	containerPath = "/var/run/kubevirt/dra/hostpath"
)

type cdiSpec struct {
	Version string      `json:"cdiVersion"`
	Kind    string      `json:"kind"`
	Devices []cdiDevice `json:"devices"`
}

type cdiDevice struct {
	Name           string            `json:"name"`
	ContainerEdits cdiContainerEdits `json:"containerEdits"`
}

type cdiContainerEdits struct {
	Env    []string   `json:"env,omitempty"`
	Mounts []cdiMount `json:"mounts,omitempty"`
}

type cdiMount struct {
	HostPath      string   `json:"hostPath"`
	ContainerPath string   `json:"containerPath"`
	Options       []string `json:"options,omitempty"`
}

func prepareHostpath(claimName string) (string, error) {
	path := baseDir + "/" + claimName
	if err := os.MkdirAll(path, 0755); err != nil {
		return "", fmt.Errorf("failed to create directory %s: %w", path, err)
	}
	log.Printf("Created directory: %s", path)

	return createCDISpec(claimName, path)
}

func createCDISpec(claimName, path string) (string, error) {
	spec := cdiSpec{
		Version: "0.5.0",
		Kind:    cdiVendor + "/" + cdiClass,
		Devices: []cdiDevice{{
			Name: claimName,
			ContainerEdits: cdiContainerEdits{
				Env: []string{fmt.Sprintf("KUBEVIRT_HOSTPATH_MOUNTPOINT=%s", containerPath)},
				Mounts: []cdiMount{{
					HostPath:      path,
					ContainerPath: containerPath,
					Options:       []string{"rbind"},
				}},
			},
		}},
	}

	if err := os.MkdirAll(cdiDir, 0755); err != nil {
		return "", fmt.Errorf("failed to create CDI dir %s: %w", cdiDir, err)
	}
	specPath := fmt.Sprintf("%s/%s-%s.json", cdiDir, cdiVendor, cdiClass)
	data, err := json.MarshalIndent(spec, "", "  ")
	if err != nil {
		return "", fmt.Errorf("failed to marshal CDI spec: %w", err)
	}
	if err := os.WriteFile(specPath, data, 0644); err != nil {
		return "", fmt.Errorf("failed to write CDI spec %s: %w", specPath, err)
	}
	log.Printf("Created CDI spec: %s", specPath)

	return cdiVendor + "/" + cdiClass + "=" + claimName, nil
}

func unprepareHostpath(claimName string) {
	path := baseDir + "/" + claimName
	os.RemoveAll(path)
	log.Printf("Removed directory: %s", path)

	specPath := fmt.Sprintf("%s/%s-%s.json", cdiDir, cdiVendor, cdiClass)
	os.Remove(specPath)
	log.Printf("Removed CDI spec: %s", specPath)
}
