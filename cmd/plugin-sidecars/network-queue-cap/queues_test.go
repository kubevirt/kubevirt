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
	"fmt"
	"testing"

	libvirtxml "libvirt.org/go/libvirtxml"
)

func TestMutateDomainCapsVirtioQueues(t *testing.T) {
	input := domainXML(t, iface("virtio", 8), iface("e1000", 4), iface("virtio-non-transitional", 6))

	got, err := mutateDomain(input, vmiJSON("2"))
	if err != nil {
		t.Fatal(err)
	}

	queues := interfaceQueues(t, got)
	if queues[0] != 2 || queues[1] != 4 || queues[2] != 2 {
		t.Fatalf("queues = %v, want [2 4 2]", queues)
	}
}

func TestMutateDomainLeavesSmallerVirtioCount(t *testing.T) {
	input := domainXML(t, iface("virtio", 2))

	got, err := mutateDomain(input, vmiJSON("4"))
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != string(input) {
		t.Fatalf("domain XML changed")
	}
}

func TestMutateDomainIgnoresMissingOrInvalidAnnotation(t *testing.T) {
	input := domainXML(t, iface("virtio", 8))
	for _, annotation := range []string{"", "0", "257", "nope"} {
		vmi := []byte(`{"metadata":{}}`)
		if annotation != "" {
			vmi = vmiJSON(annotation)
		}
		got, err := mutateDomain(input, vmi)
		if err != nil {
			t.Fatal(err)
		}
		if string(got) != string(input) {
			t.Fatalf("annotation %q changed the domain XML", annotation)
		}
	}
}

func TestMutateDomainLeavesVirtioWithoutQueues(t *testing.T) {
	domain := &libvirtxml.Domain{
		Type: "kvm",
		Name: "test",
		Devices: &libvirtxml.DomainDeviceList{
			Interfaces: []libvirtxml.DomainInterface{{
				Source: &libvirtxml.DomainInterfaceSource{Ethernet: &libvirtxml.DomainInterfaceSourceEthernet{}},
				Model:  &libvirtxml.DomainInterfaceModel{Type: "virtio"},
			}},
		},
	}
	input, err := domain.Marshal()
	if err != nil {
		t.Fatal(err)
	}

	got, err := mutateDomain([]byte(input), vmiJSON("2"))
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != input {
		t.Fatalf("domain XML changed")
	}
}

func vmiJSON(maxQueues string) []byte {
	return []byte(fmt.Sprintf(`{"metadata":{"annotations":{"%s":"%s"}}}`, maxQueuesAnnotation, maxQueues))
}

func iface(model string, queues uint) libvirtxml.DomainInterface {
	return libvirtxml.DomainInterface{
		Source: &libvirtxml.DomainInterfaceSource{Ethernet: &libvirtxml.DomainInterfaceSourceEthernet{}},
		Model:  &libvirtxml.DomainInterfaceModel{Type: model},
		Driver: &libvirtxml.DomainInterfaceDriver{Name: "vhost", Queues: queues},
	}
}

func domainXML(t *testing.T, ifaces ...libvirtxml.DomainInterface) []byte {
	t.Helper()
	domain := &libvirtxml.Domain{
		Type: "kvm",
		Name: "test",
		Devices: &libvirtxml.DomainDeviceList{
			Interfaces: ifaces,
		},
	}
	raw, err := domain.Marshal()
	if err != nil {
		t.Fatal(err)
	}
	return []byte(raw)
}

func interfaceQueues(t *testing.T, raw []byte) []uint {
	t.Helper()
	domain := &libvirtxml.Domain{}
	if err := domain.Unmarshal(string(raw)); err != nil {
		t.Fatal(err)
	}
	if domain.Devices == nil {
		t.Fatal("domain has no devices")
	}
	queues := make([]uint, len(domain.Devices.Interfaces))
	for i, iface := range domain.Devices.Interfaces {
		if iface.Driver == nil {
			t.Fatalf("interface %d has no driver", i)
		}
		queues[i] = iface.Driver.Queues
	}
	return queues
}
