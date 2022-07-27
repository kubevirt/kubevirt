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
	"fmt"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/golang/mock/gomock"
	admissionv1 "k8s.io/api/admission/v1"
	k8sv1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/rand"
	"k8s.io/client-go/tools/cache"
	"k8s.io/utils/pointer"

	"kubevirt.io/api/clone"
	clonev1lpha1 "kubevirt.io/api/clone/v1alpha1"
	"kubevirt.io/api/core"
	v1 "kubevirt.io/api/core/v1"
	"kubevirt.io/client-go/kubecli"

	"kubevirt.io/kubevirt/pkg/testutils"
	virtconfig "kubevirt.io/kubevirt/pkg/virt-config"
	"kubevirt.io/kubevirt/tests/util"
)

var _ = Describe("Validating VirtualMachineClone Admitter", func() {
	var ctrl *gomock.Controller
	var virtClient *kubecli.MockKubevirtClient
	var admitter *VirtualMachineCloneAdmitter
	var vmClone *clonev1lpha1.VirtualMachineClone
	var config *virtconfig.ClusterConfig
	var kvInformer cache.SharedIndexInformer
	var vmInterface *kubecli.MockVirtualMachineInterface
	var vm *v1.VirtualMachine

	enableFeatureGate := func(featureGate string) {
		testutils.UpdateFakeKubeVirtClusterConfig(kvInformer, &v1.KubeVirt{
			Spec: v1.KubeVirtSpec{
				Configuration: v1.KubeVirtConfiguration{
					DeveloperConfiguration: &v1.DeveloperConfiguration{
						FeatureGates: []string{featureGate},
					},
				},
			},
		})
	}

	disableFeatureGates := func() {
		testutils.UpdateFakeKubeVirtClusterConfig(kvInformer, &v1.KubeVirt{
			Spec: v1.KubeVirtSpec{
				Configuration: v1.KubeVirtConfiguration{
					DeveloperConfiguration: &v1.DeveloperConfiguration{
						FeatureGates: make([]string, 0),
					},
				},
			},
		})
	}

	newValidVM := func(namespace, name string) *v1.VirtualMachine {
		return &v1.VirtualMachine{
			ObjectMeta: metav1.ObjectMeta{
				Namespace: namespace,
				Name:      name,
			},
			Spec: v1.VirtualMachineSpec{
				Template: &v1.VirtualMachineInstanceTemplateSpec{
					Spec: v1.VirtualMachineInstanceSpec{
						Volumes: []v1.Volume{
							{
								Name: "dvVol",
								VolumeSource: v1.VolumeSource{
									DataVolume: &v1.DataVolumeSource{},
								},
							},
							{
								Name: "pvcVol",
								VolumeSource: v1.VolumeSource{
									PersistentVolumeClaim: &v1.PersistentVolumeClaimVolumeSource{},
								},
							},
						},
					},
				},
			},
			Status: v1.VirtualMachineStatus{
				VolumeSnapshotStatuses: []v1.VolumeSnapshotStatus{
					{
						Name:    "dvVol",
						Enabled: true,
					},
					{
						Name:    "pvcVol",
						Enabled: true,
					},
				},
			},
		}
	}

	BeforeEach(func() {
		ctrl = gomock.NewController(GinkgoT())
		virtClient = kubecli.NewMockKubevirtClient(ctrl)
		config, _, kvInformer = testutils.NewFakeClusterConfigUsingKVConfig(&v1.KubeVirtConfiguration{})
		vmInterface = kubecli.NewMockVirtualMachineInterface(ctrl)
		virtClient.EXPECT().VirtualMachine(gomock.Any()).Return(vmInterface).AnyTimes()

		admitter = &VirtualMachineCloneAdmitter{Config: config, Client: virtClient}
		vmClone = newValidClone()
		vm = newValidVM(vmClone.Namespace, vmClone.Spec.Source.Name)
		vmInterface.EXPECT().Get(vmClone.Spec.Source.Name, gomock.Any()).Return(vm, nil).AnyTimes()
		vmInterface.EXPECT().Get(gomock.Any(), gomock.Any()).Return(nil, fmt.Errorf("does-not-exist")).AnyTimes()
		enableFeatureGate("Snapshot")
	})

	AfterEach(func() {
		disableFeatureGates()
	})

	It("should allow legal clone", func() {
		admitter.admitAndExpect(vmClone, true)
	})

	DescribeTable("should reject clone with source that lacks information", func(getSource func() *k8sv1.TypedLocalObjectReference) {
		vmClone.Spec.Source = getSource()
		admitter.admitAndExpect(vmClone, false)
	},
		Entry("Source without Name", func() *k8sv1.TypedLocalObjectReference {
			source := newValidObjReference()
			source.Name = ""
			return source
		}),
		Entry("Source without Kind", func() *k8sv1.TypedLocalObjectReference {
			source := newValidObjReference()
			source.Kind = ""
			return source
		}),
		Entry("Source with nil APIGroup", func() *k8sv1.TypedLocalObjectReference {
			source := newValidObjReference()
			source.APIGroup = nil
			return source
		}),
		Entry("Source with empty APIGroup", func() *k8sv1.TypedLocalObjectReference {
			source := newValidObjReference()
			source.APIGroup = pointer.String("")
			return source
		}),
		Entry("Source with bad kind", func() *k8sv1.TypedLocalObjectReference {
			source := newValidObjReference()
			source.Kind = "Foobar"
			return source
		}),
	)

	It("Should reject unknown source type", func() {
		vmClone.Spec.Source.Kind = rand.String(5)
		admitter.admitAndExpect(vmClone, false)
	})

	It("Should reject unknown target type", func() {
		vmClone.Spec.Target.Kind = rand.String(5)
		admitter.admitAndExpect(vmClone, false)
	})

	It("Should reject a source VM that does not exist", func() {
		vmClone.Spec.Source.Name = "vm-that-doesnt-exist"
		admitter.admitAndExpect(vmClone, false)
	})

	It("Should reject if snapshot feature gate is not enabled", func() {
		disableFeatureGates()
		admitter.admitAndExpect(vmClone, false)
	})

	DescribeTable("Should reject a source volume not Snapshot-able", func(index int) {
		vm.Status.VolumeSnapshotStatuses[index].Enabled = false
		admitter.admitAndExpect(vmClone, false)
	},
		Entry("DataVolume", 0),
		Entry("PersistentVolumeClaim", 1),
	)

	Context("Annotations and labels filters", func() {
		testFilter := func(filter string, expectAllowed bool) {
			vmClone.Spec.LabelFilters = []string{filter}
			vmClone.Spec.AnnotationFilters = []string{filter}
			admitter.admitAndExpect(vmClone, expectAllowed)
		}

		DescribeTable("Should reject", func(filter string) {
			testFilter(filter, false)

		},
			Entry("negation character alone", "!"),
			Entry("negation in the middle", "mykey/!something"),
			Entry("negation in the end", "mykey/something!"),
			Entry("wildcard in the beginning", "*mykey/something"),
			Entry("wildcard in the middle", "mykey/*something"),
		)

		DescribeTable("Should allow", func(filter string) {
			testFilter(filter, true)
		},
			Entry("regular filter", "mykey/something"),
			Entry("wildcard only", "*"),
			Entry("wildcard in the end", "mykey/something*"),
			Entry("negation in the beginning", "!mykey/something"),
		)
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

func newValidClone() *clonev1lpha1.VirtualMachineClone {
	vmClone := kubecli.NewMinimalCloneWithNS("testclone", util.NamespaceTestDefault)
	vmClone.Spec.Source = newValidObjReference()
	vmClone.Spec.Target = newValidObjReference()
	vmClone.Spec.Target.Name = "clone-target-vm"

	return vmClone
}

func newValidObjReference() *k8sv1.TypedLocalObjectReference {
	return &k8sv1.TypedLocalObjectReference{
		APIGroup: pointer.String(core.GroupName),
		Kind:     "VirtualMachine",
		Name:     "clone-source-vm",
	}
}
