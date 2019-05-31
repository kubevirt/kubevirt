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
 * Copyright 2018 Red Hat, Inc.
 *
 */

package admitters

import (
	"encoding/json"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"k8s.io/api/admission/v1beta1"

	"k8s.io/apimachinery/pkg/runtime"

	v1 "kubevirt.io/kubevirt/pkg/api/v1"
	"kubevirt.io/kubevirt/pkg/virt-api/webhooks"
)

var _ = Describe("Validating VMIRS Admitter", func() {
	vmssAdmitter := &VMSsAdmitter{}

    Context("with VirtualMachineSnapshot", func() {
		It("should reject invalid VirtualMachineInstance spec", func() {
			vms := v1.VirtualMachineSnapshot{
				Spec: v1.VirtualMachineSnapshotSpec{
					VirtualMachine: "",
				},
			}

			vmsBytes, _ := json.Marshal(vms)

			ar := &v1beta1.AdmissionReview{
				Request: &v1beta1.AdmissionRequest{
					Resource: webhooks.VirtualMachineSnapshotGroupVersionResource,
					Object: runtime.RawExtension{
						Raw: vmsBytes,
					},
				},
			}

			resp := vmssAdmitter.Admit(ar)
			Expect(resp.Allowed).To(Equal(false))
			Expect(len(resp.Result.Details.Causes)).To(Equal(1))
			Expect(resp.Result.Details.Causes[0].Field).To(Equal("spec.virtualMachine"))
		})

		It("should accept valid VirtualMachineInstance spec", func() {
			vms := v1.VirtualMachineSnapshot{
				Spec: v1.VirtualMachineSnapshotSpec{
					VirtualMachine: "completelyvalidvmtrustme",
				},
			}

			vmsBytes, _ := json.Marshal(vms)

			ar := &v1beta1.AdmissionReview{
				Request: &v1beta1.AdmissionRequest{
					Resource: webhooks.VirtualMachineSnapshotGroupVersionResource,
					Object: runtime.RawExtension{
						Raw: vmsBytes,
					},
				},
			}

			resp := vmssAdmitter.Admit(ar)
			Expect(resp.Allowed).To(Equal(true))
		})
    })
})

