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

package translate

import (
	"encoding/xml"
	"fmt"

	libvirtxml "libvirt.org/go/libvirtxml"

	api "kubevirt.io/kubevirt/pkg/virt-launcher/virtwrap/api"
)

// ToLibvirtDomain converts a KubeVirt DomainSpec to a libvirtxml Domain
// by marshaling to XML and unmarshaling into the libvirtxml type.
// Callers must verify the Plugins feature gate is enabled before calling.
func ToLibvirtDomain(spec *api.DomainSpec) (*libvirtxml.Domain, error) {
	if spec == nil {
		return nil, fmt.Errorf("DomainSpec must not be nil")
	}
	xmlBytes, err := xml.Marshal(spec)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal DomainSpec to XML: %w", err)
	}
	domain := &libvirtxml.Domain{}
	if err := domain.Unmarshal(string(xmlBytes)); err != nil {
		return nil, fmt.Errorf("failed to unmarshal XML into libvirtxml.Domain: %w", err)
	}
	return domain, nil
}

// FromLibvirtDomain converts a libvirtxml Domain back to a KubeVirt DomainSpec
// by marshaling to XML and unmarshaling into the KubeVirt type.
// QEMU commandline elements are transferred directly from the libvirtxml structs
// because Go's encoding/xml cannot unmarshal namespace-prefixed struct tags.
// Callers must verify the Plugins feature gate is enabled before calling.
func FromLibvirtDomain(domain *libvirtxml.Domain) (*api.DomainSpec, error) {
	if domain == nil {
		return nil, fmt.Errorf("libvirtxml.Domain must not be nil")
	}
	xmlStr, err := domain.Marshal()
	if err != nil {
		return nil, fmt.Errorf("failed to marshal libvirtxml.Domain to XML: %w", err)
	}
	spec := &api.DomainSpec{}
	if err := xml.Unmarshal([]byte(xmlStr), spec); err != nil {
		return nil, fmt.Errorf("failed to unmarshal XML into DomainSpec: %w", err)
	}
	transferQEMUCommandline(domain, spec)
	return spec, nil
}

const qemuNamespace = "http://libvirt.org/schemas/domain/qemu/1.0"

// transferQEMUCommandline copies QEMU commandline args and envs from the
// libvirtxml Domain to the KubeVirt DomainSpec. Go's encoding/xml cannot
// unmarshal elements with namespace-prefixed struct tags (e.g. xml:"qemu:commandline"),
// so this transfer must be done at the struct level.
func transferQEMUCommandline(domain *libvirtxml.Domain, spec *api.DomainSpec) {
	if domain.QEMUCommandline == nil {
		return
	}
	spec.XmlNS = qemuNamespace
	if len(domain.QEMUCommandline.Args) == 0 && len(domain.QEMUCommandline.Envs) == 0 {
		return
	}
	spec.QEMUCmd = &api.Commandline{}
	for _, arg := range domain.QEMUCommandline.Args {
		spec.QEMUCmd.QEMUArg = append(spec.QEMUCmd.QEMUArg, api.Arg{Value: arg.Value})
	}
	for _, env := range domain.QEMUCommandline.Envs {
		spec.QEMUCmd.QEMUEnv = append(spec.QEMUCmd.QEMUEnv, api.Env{Name: env.Name, Value: env.Value})
	}
}
