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

package hooks

import (
	"encoding/json"
	"fmt"
	"strings"

	k8sv1 "k8s.io/api/core/v1"

	v1 "kubevirt.io/api/core/v1"
)

const HookSidecarListAnnotationName = "hooks.kubevirt.io/hookSidecars"
const HookSocketsSharedDirectory = "/var/run/kubevirt-hooks"

const ContainerNameEnvVar = "CONTAINER_NAME"

const (
	PodInfoVolumeName      = "podinfo"
	PodInfoMountPath       = "/var/run/kubevirt-private/downwardapi/podinfo"
	PodInfoLabelsFile      = "labels"
	PodInfoAnnotationsFile = "annotations"
)

type DownwardAPI []v1.NetworkBindingDownwardAPIType

func NewDownwardAPI(api v1.NetworkBindingDownwardAPIType) DownwardAPI {
	if api == "" {
		return nil
	}
	return DownwardAPI{api}
}

func (d DownwardAPI) Has(api v1.NetworkBindingDownwardAPIType) bool {
	for _, requestedAPI := range d {
		if requestedAPI == api {
			return true
		}
	}
	return false
}

func (d DownwardAPI) Validate() error {
	for _, api := range d {
		switch api {
		case v1.DeviceInfo, v1.PodInfo:
		default:
			return fmt.Errorf("unsupported downwardAPI value %q", api)
		}
	}
	return nil
}

func (d *DownwardAPI) UnmarshalJSON(data []byte) error {
	raw := strings.TrimSpace(string(data))
	if raw == "" || raw == "null" {
		*d = nil
		return nil
	}

	if strings.HasPrefix(raw, "[") {
		type downwardAPIList []v1.NetworkBindingDownwardAPIType
		var apis downwardAPIList
		if err := json.Unmarshal(data, &apis); err != nil {
			return err
		}

		downwardAPIs := DownwardAPI(apis)
		if err := downwardAPIs.Validate(); err != nil {
			return err
		}

		*d = downwardAPIs
		return nil
	}

	var api v1.NetworkBindingDownwardAPIType
	if err := json.Unmarshal(data, &api); err != nil {
		return err
	}

	downwardAPIs := DownwardAPI{api}
	if err := downwardAPIs.Validate(); err != nil {
		return err
	}

	*d = downwardAPIs
	return nil
}

type HookSidecarList []HookSidecar

func (h HookSidecarList) HasDownwardAPI(api v1.NetworkBindingDownwardAPIType) bool {
	for _, sidecar := range h {
		if sidecar.DownwardAPI.Has(api) {
			return true
		}
	}
	return false
}

type ConfigMap struct {
	Name     string `json:"name"`
	Key      string `json:"key"`
	HookPath string `json:"hookPath"`
}

type PVC struct {
	Name              string `json:"name"`
	VolumePath        string `json:"volumePath"`
	SharedComputePath string `json:"sharedComputePath"`
}

type HookSidecar struct {
	Image           string           `json:"image,omitempty"`
	ImagePullPolicy k8sv1.PullPolicy `json:"imagePullPolicy"`
	Command         []string         `json:"command,omitempty"`
	Args            []string         `json:"args,omitempty"`
	ConfigMap       *ConfigMap       `json:"configMap,omitempty"`
	PVC             *PVC             `json:"pvc,omitempty"`
	DownwardAPI     DownwardAPI      `json:"downwardAPI,omitempty"`
}

func UnmarshalHookSidecarList(vmiObject *v1.VirtualMachineInstance) (HookSidecarList, error) {
	hookSidecarList := make(HookSidecarList, 0)

	if rawRequestedHookSidecarList, requestedHookSidecarListDefined := vmiObject.GetAnnotations()[HookSidecarListAnnotationName]; requestedHookSidecarListDefined {
		if err := json.Unmarshal([]byte(rawRequestedHookSidecarList), &hookSidecarList); err != nil {
			return nil, err
		}
	}

	return hookSidecarList, nil
}
