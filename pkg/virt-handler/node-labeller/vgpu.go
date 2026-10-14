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

package nodelabeller

import (
	"os"
	"regexp"
	"strings"
)

// nvidiaDriverVersionPath is the host sysfs file exposed by the NVIDIA vGPU kernel module.
// virt-handler has the host PID namespace, so /proc/1/root is the host root.
const nvidiaDriverVersionPath = "/proc/1/root/sys/module/nvidia_vgpu_vfio/version"

var nvidiaDriverVersionPattern = regexp.MustCompile(`^[0-9]+(?:\.[0-9]+)+$`)

func (n *NodeLabeller) vgpuHostDriverVersion() (string, bool) {
	read := n.vgpuQuery
	if read == nil {
		read = readNVIDIADriverVersion
	}

	output, err := read()
	if err != nil {
		n.logger.V(3).Infof("omitting vGPU host driver version label: %v", err)
		return "", false
	}

	version, ok := parseVGPUHostDriverVersion(string(output))
	if !ok {
		n.logger.V(3).Info("omitting vGPU host driver version label: driver version was not reported")
		return "", false
	}

	return version, true
}

func parseVGPUHostDriverVersion(output string) (string, bool) {
	version := strings.TrimSpace(output)
	if !nvidiaDriverVersionPattern.MatchString(version) {
		return "", false
	}
	return version, true
}

func readNVIDIADriverVersion() ([]byte, error) {
	return os.ReadFile(nvidiaDriverVersionPath)
}
