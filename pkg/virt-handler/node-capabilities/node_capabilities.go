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
 * Copyright the KubeVirt Authors.
 *
 */

package nodecapabilities

import (
	"os"
	"path/filepath"

	"libvirt.org/go/libvirtxml"
)

const (
	CapabilitiesVolumePath   = "/var/lib/kubevirt-node-labeller/"
	HostCapabilitiesFilename = "capabilities.xml"
)

func ParseHostCapabilities(hostCapabilities string) (*libvirtxml.Caps, error) {
	var capabilities libvirtxml.Caps
	if err := capabilities.Unmarshal(hostCapabilities); err != nil {
		return nil, err
	}
	return &capabilities, nil
}

func HostCapabilities() (*libvirtxml.Caps, error) {
	hostCapabilitiesXML, err := os.ReadFile(filepath.Join(CapabilitiesVolumePath, HostCapabilitiesFilename))
	if err != nil {
		return nil, err
	}
	return ParseHostCapabilities(string(hostCapabilitiesXML))
}
