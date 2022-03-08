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
* Copyright 2022 Red Hat, Inc.
*
 */

package admitters

import (
	"encoding/json"

	"github.com/golang/mock/gomock"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	admissionv1 "k8s.io/api/admission/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"

	"kubevirt.io/api/clone"
	clonev1lpha1 "kubevirt.io/api/clone/v1alpha1"
	"kubevirt.io/client-go/kubecli"
)

var _ = Describe("Validating VirtualMachineClone Admitter", func() {
	var ctrl *gomock.Controller
	var virtClient *kubecli.MockKubevirtClient
	var admitter *VirtualMachineCloneAdmitter

	BeforeEach(func() {
		ctrl = gomock.NewController(GinkgoT())
		virtClient = kubecli.NewMockKubevirtClient(ctrl)
		admitter = &VirtualMachineCloneAdmitter{Client: virtClient}
	})

	It("should allow legal clone", func() {
		vmClone := &clonev1lpha1.VirtualMachineClone{}
		admitter.admitAndExpect(vmClone, true)
	})

})

func createCloneAdmissionReview(vmClone *clonev1lpha1.VirtualMachineClone) *admissionv1.AdmissionReview {
	policyBytes, _ := json.Marshal(vmClone)

	ar := &admissionv1.AdmissionReview{
		Request: &admissionv1.AdmissionRequest{
			Operation: admissionv1.Create,
			Resource: metav1.GroupVersionResource{
				Group:    clonev1lpha1.VirtualMachineCloneKind.Group,
				Resource: clone.ResourceVMClonePlural,
			},
			Object: runtime.RawExtension{
				Raw: policyBytes,
			},
		},
	}

	return ar
}

func (admitter *VirtualMachineCloneAdmitter) admitAndExpect(clone *clonev1lpha1.VirtualMachineClone, expectAllowed bool) {
	ar := createCloneAdmissionReview(clone)
	resp := admitter.Admit(ar)
	Expect(resp.Allowed).To(Equal(expectAllowed))
}
