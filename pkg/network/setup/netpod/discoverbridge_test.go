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

package netpod

import (
	"testing"

	v1 "kubevirt.io/api/core/v1"
)

func TestResolveMacAddress(t *testing.T) {
	const (
		podMAC       = "aa:bb:cc:dd:ee:01"
		vmiStatusMAC = "aa:bb:cc:dd:ee:02"
		vmiSpecMAC   = "aa:bb:cc:dd:ee:03"
	)

	tests := []struct {
		name        string
		podMAC      string
		statusMAC   string
		specMAC     string
		expectedMAC string
	}{
		{
			name:        "uses pod MAC when no other source",
			podMAC:      podMAC,
			statusMAC:   "",
			specMAC:     "",
			expectedMAC: podMAC,
		},
		{
			name:        "prefers VMI status MAC over pod MAC",
			podMAC:      podMAC,
			statusMAC:   vmiStatusMAC,
			specMAC:     "",
			expectedMAC: vmiStatusMAC,
		},
		{
			name:        "prefers VMI spec MAC over status MAC",
			podMAC:      podMAC,
			statusMAC:   vmiStatusMAC,
			specMAC:     vmiSpecMAC,
			expectedMAC: vmiSpecMAC,
		},
		{
			name:        "uses VMI spec MAC over pod MAC when status is empty",
			podMAC:      podMAC,
			statusMAC:   "",
			specMAC:     vmiSpecMAC,
			expectedMAC: vmiSpecMAC,
		},
		{
			name:        "migration scenario: status MAC preserved when spec is empty",
			podMAC:      podMAC,
			statusMAC:   vmiStatusMAC,
			specMAC:     "",
			expectedMAC: vmiStatusMAC,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mac, err := resolveMacAddress(tt.podMAC, tt.statusMAC, tt.specMAC)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if mac.String() != tt.expectedMAC {
				t.Errorf("expected %s, got %s", tt.expectedMAC, mac.String())
			}
		})
	}
}

func TestResolveMacAddressInvalidMAC(t *testing.T) {
	_, err := resolveMacAddress("invalid-mac", "", "")
	if err == nil {
		t.Error("expected error for invalid MAC address")
	}
}

func TestGetVMIStatusMAC(t *testing.T) {
	const (
		ifaceName = "default"
		ifaceMAC  = "aa:bb:cc:dd:ee:ff"
	)

	tests := []struct {
		name        string
		statuses    []v1.VirtualMachineInstanceNetworkInterface
		ifaceName   string
		expectedMAC string
	}{
		{
			name: "returns MAC address for matching interface name",
			statuses: []v1.VirtualMachineInstanceNetworkInterface{
				{Name: ifaceName, MAC: ifaceMAC},
			},
			ifaceName:   ifaceName,
			expectedMAC: ifaceMAC,
		},
		{
			name: "returns empty string when interface not found",
			statuses: []v1.VirtualMachineInstanceNetworkInterface{
				{Name: "other", MAC: ifaceMAC},
			},
			ifaceName:   ifaceName,
			expectedMAC: "",
		},
		{
			name:        "returns empty string when statuses are empty",
			statuses:    nil,
			ifaceName:   ifaceName,
			expectedMAC: "",
		},
		{
			name: "finds correct interface among multiple interfaces",
			statuses: []v1.VirtualMachineInstanceNetworkInterface{
				{Name: "first", MAC: "11:22:33:44:55:66"},
				{Name: ifaceName, MAC: ifaceMAC},
				{Name: "third", MAC: "77:88:99:aa:bb:cc"},
			},
			ifaceName:   ifaceName,
			expectedMAC: ifaceMAC,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			n := NetPod{vmiIfaceStatuses: tt.statuses}
			mac := n.getVMIStatusMAC(tt.ifaceName)
			if mac != tt.expectedMAC {
				t.Errorf("expected %q, got %q", tt.expectedMAC, mac)
			}
		})
	}
}
