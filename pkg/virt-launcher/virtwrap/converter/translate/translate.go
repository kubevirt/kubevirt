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
	"io"
	"strings"

	libvirtxml "libvirt.org/go/libvirtxml"

	"kubevirt.io/client-go/log"

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
	TransferQEMUCommandline(xmlStr, spec)
	return spec, nil
}

const qemuNamespace = "http://libvirt.org/schemas/domain/qemu/1.0"

// TransferQEMUCommandline extracts qemu:commandline args and envs from raw
// domain XML and applies them to the KubeVirt DomainSpec. Go's encoding/xml
// cannot unmarshal namespace-prefixed struct tags (e.g. xml:"qemu:commandline"),
// so this function uses token-level parsing via xml.Decoder, which resolves
// namespace URIs correctly. Only elements in the QEMU namespace are inspected;
// the rest of the domain XML is skipped, so this works with any well-formed
// XML regardless of schema strictness.
func TransferQEMUCommandline(domainXML string, spec *api.DomainSpec) {
	args, envs, hasCommandline := parseQEMUCommandline(domainXML)

	if hasCommandline {
		spec.XmlNS = qemuNamespace
	}
	if len(args) == 0 && len(envs) == 0 {
		return
	}
	spec.QEMUCmd = &api.Commandline{
		QEMUArg: args,
		QEMUEnv: envs,
	}
}

func parseQEMUCommandline(domainXML string) ([]api.Arg, []api.Env, bool) {
	decoder := xml.NewDecoder(strings.NewReader(domainXML))

	var (
		args           []api.Arg
		envs           []api.Env
		inCommandline  bool
		hasCommandline bool
	)

	for {
		token, err := decoder.Token()
		if err != nil {
			if err != io.EOF {
				log.Log.Warningf("unexpected error parsing domain XML for QEMUCmd: %v", err)
			}
			break
		}

		switch t := token.(type) {
		case xml.StartElement:
			if t.Name.Space != qemuNamespace {
				continue
			}
			switch t.Name.Local {
			case "commandline":
				inCommandline = true
				hasCommandline = true
			case "arg":
				if inCommandline {
					args = append(args, parseArgAttrs(t))
				}
			case "env":
				if inCommandline {
					if env, ok := parseEnvAttrs(t); ok {
						envs = append(envs, env)
					}
				}
			}
		case xml.EndElement:
			if t.Name.Space == qemuNamespace && t.Name.Local == "commandline" {
				inCommandline = false
			}
		}
	}

	return args, envs, hasCommandline
}

func parseArgAttrs(t xml.StartElement) api.Arg {
	for _, attr := range t.Attr {
		if attr.Name.Local == "value" {
			return api.Arg{Value: attr.Value}
		}
	}
	return api.Arg{}
}

func parseEnvAttrs(t xml.StartElement) (api.Env, bool) {
	var env api.Env
	for _, attr := range t.Attr {
		switch attr.Name.Local {
		case "name":
			env.Name = attr.Value
		case "value":
			env.Value = attr.Value
		}
	}
	if env.Name == "" {
		return api.Env{}, false
	}
	return env, true
}
