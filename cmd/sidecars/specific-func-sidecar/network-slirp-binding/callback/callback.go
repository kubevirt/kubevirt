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
 * Copyright 2023 Red Hat, Inc.
 *
 */

package callback

import (
	"encoding/xml"
	"fmt"

	domainschema "kubevirt.io/kubevirt/pkg/virt-launcher/virtwrap/api"
)

// TODO: move to Kubevirt domain API package
const libvirtDomainQemuSchema = "http://libvirt.org/schemas/domain/qemu/1.0"

type DomainSpecMutator interface {
	Mutate(*domainschema.DomainSpec) (*domainschema.DomainSpec, error)
}

func OnDefineDomain(domainXML []byte, domSpecMutator DomainSpecMutator) ([]byte, error) {
	domainSpec := &domainschema.DomainSpec{
		// Unmarshalling domain spec makes the XML namespace attribute empty.
		// Some domain parameters requires namespace to be defined.
		// e.g: https://libvirt.org/drvqemu.html#pass-through-of-arbitrary-qemu-commands
		XmlNS: libvirtDomainQemuSchema,
	}
	if err := xml.Unmarshal(domainXML, domainSpec); err != nil {
		return nil, fmt.Errorf("failed to unmarshal given domain spec: %v", err)
	}

	updatedDomainSpec, err := domSpecMutator.Mutate(domainSpec)
	if err != nil {
		return nil, err
	}

	updatedDomainSpecXML, err := xml.Marshal(updatedDomainSpec)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal updated domain spec: %v", err)
	}

	return updatedDomainSpecXML, nil
}
