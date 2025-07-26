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

package admitters

import (
	"context"
	"encoding/json"
	"fmt"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"go.uber.org/mock/gomock"
	admissionv1 "k8s.io/api/admission/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/testing"

	v1 "kubevirt.io/api/core/v1"
	"kubevirt.io/client-go/kubecli"
	"kubevirt.io/client-go/kubevirt/fake"

	"kubevirt.io/kubevirt/pkg/libvmi"
	"kubevirt.io/kubevirt/pkg/pointer"
	"kubevirt.io/kubevirt/pkg/virt-api/webhooks"
)

const defaultTerminationGracePeriod = 1600

var _ = Describe("Validating VMs Delete Admitter", func() {
	var (
		vmsDeleteAdmitter *VMsDeleteAdmitter
		virtClient        *kubecli.MockKubevirtClient
		virtFakeClient    *fake.Clientset
		vm                *v1.VirtualMachine
		vmi               *v1.VirtualMachineInstance
	)

	BeforeEach(func() {
		virtClient = kubecli.NewMockKubevirtClient(gomock.NewController(GinkgoT()))
		virtFakeClient = fake.NewSimpleClientset()

		vmsDeleteAdmitter = &VMsDeleteAdmitter{
			VirtClient: virtClient,
		}

		virtClient.EXPECT().VirtualMachineInstance(metav1.NamespaceDefault).Return(
			virtFakeClient.KubevirtV1().VirtualMachineInstances(metav1.NamespaceDefault)).AnyTimes()

		vmi = libvmi.New(
			libvmi.WithNamespace(metav1.NamespaceDefault),
			libvmi.WithTerminationGracePeriod(defaultTerminationGracePeriod),
		)
		_, err := virtFakeClient.KubevirtV1().VirtualMachineInstances(metav1.NamespaceDefault).Create(context.Background(), vmi, metav1.CreateOptions{})
		Expect(err).NotTo(HaveOccurred())

		vm = libvmi.NewVirtualMachine(vmi)
		_, err = virtFakeClient.KubevirtV1().VirtualMachines(metav1.NamespaceDefault).Create(context.Background(), vm, metav1.CreateOptions{})
		Expect(err).NotTo(HaveOccurred())
	})

	Context("when handling VM deletion", func() {
		Context("with dry-run enabled", func() {
			It("should not patch VMI when dryRun is true", func() {
				grace := int64(10)
				deleteOpts := &metav1.DeleteOptions{
					GracePeriodSeconds: &grace,
					DryRun:             []string{"All"},
				}
				resp := admitVmForDeletion(vmsDeleteAdmitter, vm, deleteOpts)
				Expect(resp.Allowed).To(BeTrue())
				verifyPatch(virtFakeClient, vmi, false, nil)
			})
		})

		Context("with no DeleteOptions provided", func() {
			It("should not patch VMI when DeleteOptions is nil", func() {
				resp := admitVmForDeletion(vmsDeleteAdmitter, vm, nil)
				Expect(resp.Allowed).To(BeTrue())
				verifyPatch(virtFakeClient, vmi, false, nil)
			})
		})

		Context("with grace period settings", func() {
			It("should not patch VMI when GracePeriodSeconds is nil", func() {
				deleteOpts := &metav1.DeleteOptions{}
				resp := admitVmForDeletion(vmsDeleteAdmitter, vm, deleteOpts)
				Expect(resp.Allowed).To(BeTrue())
				verifyPatch(virtFakeClient, vmi, false, nil)
			})

			It("should not patch VMI when GracePeriodSeconds matches VM's current value", func() {
				grace := int64(defaultTerminationGracePeriod)
				deleteOpts := &metav1.DeleteOptions{
					GracePeriodSeconds: &grace,
				}
				resp := admitVmForDeletion(vmsDeleteAdmitter, vm, deleteOpts)
				Expect(resp.Allowed).To(BeTrue())
				verifyPatch(virtFakeClient, vmi, false, nil)
			})

			It("should patch VMI when GracePeriodSeconds differs from VM's current value", func() {
				grace := int64(10)
				deleteOpts := &metav1.DeleteOptions{
					GracePeriodSeconds: &grace,
				}
				resp := admitVmForDeletion(vmsDeleteAdmitter, vm, deleteOpts)
				Expect(resp.Allowed).To(BeTrue())
				verifyPatch(virtFakeClient, vmi, true, &grace)
			})
		})

		Context("with invalid input", func() {
			It("should not patch VMI when VM object unmarshalling fails", func() {
				grace := int64(10)
				deleteOpts := &metav1.DeleteOptions{GracePeriodSeconds: &grace}
				optsBytes, _ := json.Marshal(deleteOpts)

				ar := &admissionv1.AdmissionReview{
					Request: &admissionv1.AdmissionRequest{
						Resource:  webhooks.VirtualMachineGroupVersionResource,
						Name:      vm.Name,
						Namespace: vm.Namespace,
						Object: runtime.RawExtension{
							Raw: []byte("invalid-json"),
						},
						Operation: admissionv1.Delete,
						Options: runtime.RawExtension{
							Raw: optsBytes,
						},
					},
				}

				resp := vmsDeleteAdmitter.Admit(context.Background(), ar)
				Expect(resp.Allowed).To(BeTrue())
				verifyPatch(virtFakeClient, vmi, false, nil)
			})

			It("should not patch VMI when DeleteOptions unmarshalling fails", func() {
				vmBytes, _ := json.Marshal(vm)
				ar := &admissionv1.AdmissionReview{
					Request: &admissionv1.AdmissionRequest{
						Resource:  webhooks.VirtualMachineGroupVersionResource,
						Name:      vm.Name,
						Namespace: vm.Namespace,
						Object: runtime.RawExtension{
							Raw: vmBytes,
						},
						Operation: admissionv1.Delete,
						Options: runtime.RawExtension{
							Raw: []byte("invalid-json"),
						},
					},
				}

				resp := vmsDeleteAdmitter.Admit(context.Background(), ar)
				Expect(resp.Allowed).To(BeTrue())
				verifyPatch(virtFakeClient, vmi, false, nil)
			})
		})

		Context("with patch errors", func() {
			It("should attempt patch but proceed when patch returns NotFound error", func() {
				grace := int64(10)
				deleteOpts := &metav1.DeleteOptions{GracePeriodSeconds: &grace}

				virtFakeClient.PrependReactor("patch", "virtualmachineinstances", func(action testing.Action) (handled bool, ret runtime.Object, err error) {
					return true, nil, errors.NewNotFound(v1.Resource("virtualmachineinstances"), vmi.Name)
				})

				resp := admitVmForDeletion(vmsDeleteAdmitter, vm, deleteOpts)
				Expect(resp.Allowed).To(BeTrue())
			})

			It("should deny deletion when patch fails with non-NotFound error", func() {
				grace := int64(10)
				deleteOpts := &metav1.DeleteOptions{GracePeriodSeconds: &grace}

				virtFakeClient.PrependReactor("patch", "virtualmachineinstances", func(action testing.Action) (handled bool, ret runtime.Object, err error) {
					return true, nil, fmt.Errorf("patch error")
				})

				resp := admitVmForDeletion(vmsDeleteAdmitter, vm, deleteOpts)
				Expect(resp.Allowed).To(BeFalse())
				Expect(resp.Result.Message).To(ContainSubstring("Failed to update the VMI's terminationGracePeriodSeconds"))
			})
		})
	})
})

func admitVmForDeletion(admitter *VMsDeleteAdmitter, vm *v1.VirtualMachine, deleteOpts *metav1.DeleteOptions) *admissionv1.AdmissionResponse {
	vmBytes, _ := json.Marshal(vm)

	var optsBytes []byte
	var dryRun bool
	if deleteOpts != nil {
		var err error
		optsBytes, err = json.Marshal(deleteOpts)
		if err != nil {
			optsBytes = []byte{}
		}
		if len(deleteOpts.DryRun) > 0 {
			dryRun = true
		}
	}

	ar := &admissionv1.AdmissionReview{
		Request: &admissionv1.AdmissionRequest{
			Resource:  webhooks.VirtualMachineGroupVersionResource,
			Name:      vm.Name,
			Namespace: vm.Namespace,
			OldObject: runtime.RawExtension{
				Raw: vmBytes,
			},
			Operation: admissionv1.Delete,
			DryRun:    pointer.P(dryRun),
			Options: runtime.RawExtension{
				Raw: optsBytes,
			},
		},
	}

	return admitter.Admit(context.Background(), ar)
}

func verifyPatch(virtFakeClient *fake.Clientset, vmi *v1.VirtualMachineInstance, expectPatch bool, expectedGrace *int64) {
	updatedVMI, err := virtFakeClient.KubevirtV1().VirtualMachineInstances(metav1.NamespaceDefault).Get(context.Background(), vmi.Name, metav1.GetOptions{})
	Expect(err).NotTo(HaveOccurred())
	if expectPatch {
		Expect(updatedVMI.Spec.TerminationGracePeriodSeconds).ToNot(BeNil(), "TerminationGracePeriodSeconds should be set")
		Expect(*updatedVMI.Spec.TerminationGracePeriodSeconds).To(Equal(*expectedGrace), "TerminationGracePeriodSeconds should match expected value")
	} else {
		Expect(updatedVMI.Spec.TerminationGracePeriodSeconds).ToNot(BeNil(), "TerminationGracePeriodSeconds should be set")
		Expect(*updatedVMI.Spec.TerminationGracePeriodSeconds).To(Equal(int64(defaultTerminationGracePeriod)), "TerminationGracePeriodSeconds should remain at default value")
	}

	actions := virtFakeClient.Actions()
	patchFound := false
	for _, action := range actions {
		if action.GetVerb() == "patch" && action.GetResource().Resource == "virtualmachineinstances" {
			patchFound = true
			Expect(action.GetNamespace()).To(Equal(metav1.NamespaceDefault), "Patch action should target default namespace")
			Expect(action.(testing.PatchAction).GetName()).To(Equal(vmi.Name), "Patch action should target correct VMI")
		}
	}
	Expect(patchFound).To(Equal(expectPatch))
}
