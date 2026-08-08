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

package network

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"

	v1 "kubevirt.io/api/core/v1"

	"kubevirt.io/kubevirt/pkg/network/setup/netpod"
)

var _ = Describe("networkConfigStateKey", func() {
	const (
		vmiUID   = "vmi-uid"
		nodeName = "node1"
	)

	It("includes the active pod UID when one pod matches the node", func() {
		vmi := &v1.VirtualMachineInstance{
			ObjectMeta: metav1.ObjectMeta{UID: types.UID(vmiUID)},
			Status: v1.VirtualMachineInstanceStatus{
				NodeName: nodeName,
				ActivePods: map[types.UID]string{
					types.UID("pod-uid"): nodeName,
				},
			},
		}

		Expect(networkConfigStateKey(vmi)).To(Equal(vmiUID + "/pod-uid"))
	})

	It("falls back to the VMI UID when no active pod matches the node", func() {
		vmi := &v1.VirtualMachineInstance{
			ObjectMeta: metav1.ObjectMeta{UID: types.UID(vmiUID)},
			Status: v1.VirtualMachineInstanceStatus{
				NodeName: nodeName,
				ActivePods: map[types.UID]string{
					types.UID("pod-uid"): "other-node",
				},
			},
		}

		Expect(networkConfigStateKey(vmi)).To(Equal(vmiUID))
	})

	It("uses the lexicographically smallest pod UID when multiple pods match the node", func() {
		vmi := &v1.VirtualMachineInstance{
			ObjectMeta: metav1.ObjectMeta{UID: types.UID(vmiUID)},
			Status: v1.VirtualMachineInstanceStatus{
				NodeName: nodeName,
				ActivePods: map[types.UID]string{
					types.UID("pod-z"): nodeName,
					types.UID("pod-a"): nodeName,
				},
			},
		}

		Expect(networkConfigStateKey(vmi)).To(Equal(vmiUID + "/pod-a"))
	})
})

var _ = Describe("deleteNetworkConfigStatesForVMI", func() {
	It("removes all state entries for the VMI", func() {
		stateMap := map[string]*netpod.State{
			"vmi-uid":           nil,
			"vmi-uid/old-pod":   nil,
			"vmi-uid/new-pod":   nil,
			"other-vmi/pod-uid": nil,
		}

		deleteNetworkConfigStatesForVMI(stateMap, "vmi-uid")

		Expect(stateMap).To(Equal(map[string]*netpod.State{
			"other-vmi/pod-uid": nil,
		}))
	})
})
