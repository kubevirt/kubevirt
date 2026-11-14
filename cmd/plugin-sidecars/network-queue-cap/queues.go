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

package main

import (
	"encoding/json"
	"fmt"
	"strconv"
	"strings"

	libvirtxml "libvirt.org/go/libvirtxml"

	v1 "kubevirt.io/api/core/v1"
)

const (
	maxQueuesAnnotation = "network.kubevirt.io/max-queues"
	minQueues           = 1
	maxQueues           = 256
)

func mutateDomain(domainXML, vmiJSON []byte) ([]byte, error) {
	vmi := v1.VirtualMachineInstance{}
	if err := json.Unmarshal(vmiJSON, &vmi); err != nil {
		return nil, fmt.Errorf("unmarshal VMI: %w", err)
	}

	limit, ok := parseMaxQueues(vmi.GetAnnotations())
	if !ok {
		return domainXML, nil
	}

	domain := &libvirtxml.Domain{}
	if err := domain.Unmarshal(string(domainXML)); err != nil {
		return nil, fmt.Errorf("unmarshal domain: %w", err)
	}
	if domain.Devices == nil || !capVirtioQueues(domain, limit) {
		return domainXML, nil
	}

	updated, err := domain.Marshal()
	if err != nil {
		return nil, fmt.Errorf("marshal domain: %w", err)
	}
	return []byte(updated), nil
}

func parseMaxQueues(annotations map[string]string) (uint, bool) {
	raw, found := annotations[maxQueuesAnnotation]
	if !found {
		return 0, false
	}
	value, err := strconv.ParseUint(raw, 10, 32)
	if err != nil || value < minQueues || value > maxQueues {
		return 0, false
	}
	return uint(value), true
}

func capVirtioQueues(domain *libvirtxml.Domain, limit uint) bool {
	changed := false
	for i := range domain.Devices.Interfaces {
		iface := &domain.Devices.Interfaces[i]
		if !isVirtio(iface.Model) || iface.Driver == nil || iface.Driver.Queues == 0 || iface.Driver.Queues <= limit {
			continue
		}
		iface.Driver.Queues = limit
		changed = true
	}
	return changed
}

func isVirtio(model *libvirtxml.DomainInterfaceModel) bool {
	if model == nil {
		return false
	}
	return model.Type == v1.VirtIO || strings.HasPrefix(model.Type, v1.VirtIO+"-")
}
