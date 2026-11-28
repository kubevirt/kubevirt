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

package libnet

import (
	"testing"

	k8sv1 "k8s.io/api/core/v1"

	v1 "kubevirt.io/api/core/v1"
)

func TestLookupInterfaceByName(t *testing.T) {
	ifaces := []v1.VirtualMachineInstanceNetworkInterface{
		{Name: "eth0", IP: "10.0.0.1"},
		{Name: "default", IP: "192.168.1.1"},
	}

	t.Run("empty slice", func(t *testing.T) {
		if got := LookupInterfaceByName(nil, "default"); got != nil {
			t.Fatalf("expected nil for nil slice, got %v", got)
		}
		if got := LookupInterfaceByName([]v1.VirtualMachineInstanceNetworkInterface{}, "default"); got != nil {
			t.Fatalf("expected nil for empty slice, got %v", got)
		}
	})

	t.Run("matching interface", func(t *testing.T) {
		got := LookupInterfaceByName(ifaces, "default")
		if got == nil {
			t.Fatal("expected to find interface 'default', got nil")
		}
		if got.IP != "192.168.1.1" {
			t.Fatalf("expected IP 192.168.1.1, got %s", got.IP)
		}
	})

	t.Run("missing interface", func(t *testing.T) {
		if got := LookupInterfaceByName(ifaces, "nonexistent"); got != nil {
			t.Fatalf("expected nil for nonexistent interface, got %v", got)
		}
	})
}

func TestLookupInterfaceStatusByName(t *testing.T) {
	t.Run("nil VMI", func(t *testing.T) {
		if got := LookupInterfaceStatusByName(nil, "default"); got != nil {
			t.Fatalf("expected nil for nil VMI, got %v", got)
		}
	})

	t.Run("found interface on VMI", func(t *testing.T) {
		vmi := &v1.VirtualMachineInstance{
			Status: v1.VirtualMachineInstanceStatus{
				Interfaces: []v1.VirtualMachineInstanceNetworkInterface{
					{Name: "nic1", IP: "10.1.1.1"},
				},
			},
		}
		got := LookupInterfaceStatusByName(vmi, "nic1")
		if got == nil || got.IP != "10.1.1.1" {
			t.Fatalf("expected interface with IP 10.1.1.1, got %v", got)
		}
	})
}

func TestGetVmiPrimaryInterface(t *testing.T) {
	t.Run("nil VMI or empty interfaces", func(t *testing.T) {
		if got := GetVmiPrimaryInterface(nil); got != nil {
			t.Fatalf("expected nil for nil VMI, got %v", got)
		}
		if got := GetVmiPrimaryInterface(&v1.VirtualMachineInstance{}); got != nil {
			t.Fatalf("expected nil for empty VMI, got %v", got)
		}
	})

	t.Run("matches default pod network name", func(t *testing.T) {
		defaultName := v1.DefaultPodNetwork().Name
		vmi := &v1.VirtualMachineInstance{
			Status: v1.VirtualMachineInstanceStatus{
				Interfaces: []v1.VirtualMachineInstanceNetworkInterface{
					{Name: "secondary", IP: "10.0.0.2"},
					{Name: defaultName, IP: "10.0.0.1"},
				},
			},
		}
		got := GetVmiPrimaryInterface(vmi)
		if got == nil {
			t.Fatal("expected primary interface, got nil")
		}
		if got.Name != defaultName || got.IP != "10.0.0.1" {
			t.Fatalf("expected default interface with IP 10.0.0.1, got %v", got)
		}
	})

	t.Run("falls back to first interface when default name not present", func(t *testing.T) {
		vmi := &v1.VirtualMachineInstance{
			Status: v1.VirtualMachineInstanceStatus{
				Interfaces: []v1.VirtualMachineInstanceNetworkInterface{
					{Name: "custom0", IP: "172.16.0.1"},
					{Name: "custom1", IP: "172.16.0.2"},
				},
			},
		}
		got := GetVmiPrimaryInterface(vmi)
		if got == nil {
			t.Fatal("expected fallback interface, got nil")
		}
		if got.Name != "custom0" || got.IP != "172.16.0.1" {
			t.Fatalf("expected first interface 'custom0', got %v", got)
		}
	})
}

func TestGetVmiPrimaryIPByFamily(t *testing.T) {
	t.Run("empty VMI", func(t *testing.T) {
		if got := GetVmiPrimaryIPByFamily(nil, k8sv1.IPv4Protocol); got != "" {
			t.Fatalf("expected empty string, got %s", got)
		}
	})

	t.Run("resolves IPv4 and IPv6", func(t *testing.T) {
		vmi := &v1.VirtualMachineInstance{
			Status: v1.VirtualMachineInstanceStatus{
				Interfaces: []v1.VirtualMachineInstanceNetworkInterface{
					{
						Name: v1.DefaultPodNetwork().Name,
						IPs:  []string{"10.0.2.15", "fd10:0:2::15"},
					},
				},
			},
		}
		if got := GetVmiPrimaryIPByFamily(vmi, k8sv1.IPv4Protocol); got != "10.0.2.15" {
			t.Fatalf("expected 10.0.2.15, got %s", got)
		}
		if got := GetVmiPrimaryIPByFamily(vmi, k8sv1.IPv6Protocol); got != "fd10:0:2::15" {
			t.Fatalf("expected fd10:0:2::15, got %s", got)
		}
	})
}
