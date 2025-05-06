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
	"kubevirt.io/kubevirt/pkg/apimachinery/patch"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"k8s.io/client-go/testing"
	snapshotv1 "kubevirt.io/api/snapshot/v1beta1"
	"kubevirt.io/client-go/kubevirt/fake"

	"github.com/golang/mock/gomock"
	admissionv1 "k8s.io/api/admission/v1"
	k8sv1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/rand"
	"k8s.io/client-go/tools/cache"

	clonebase "kubevirt.io/api/clone"
	clone "kubevirt.io/api/clone/v1beta1"
	"kubevirt.io/api/core"
	v1 "kubevirt.io/api/core/v1"
	"kubevirt.io/client-go/kubecli"

	"kubevirt.io/kubevirt/pkg/pointer"
	"kubevirt.io/kubevirt/pkg/testutils"
	virtconfig "kubevirt.io/kubevirt/pkg/virt-config"
)

var _ = Describe("Validating VirtualMachineClone Admitter", func() {
	var ctrl *gomock.Controller
	var virtClient *kubecli.MockKubevirtClient
	var kubevirtClient *fake.Clientset
	var admitter *VirtualMachineCloneAdmitter
	var vmClone *clone.VirtualMachineClone
	var config *virtconfig.ClusterConfig
	var kvStore cache.Store
	var vmInterface *kubecli.MockVirtualMachineInterface
	var vm *v1.VirtualMachine

	enableFeatureGate := func(featureGate string) {
		testutils.UpdateFakeKubeVirtClusterConfig(kvStore, &v1.KubeVirt{
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
		testutils.UpdateFakeKubeVirtClusterConfig(kvStore, &v1.KubeVirt{
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
							{
								Name: "containerDiskVol",
								VolumeSource: v1.VolumeSource{
									ContainerDisk: &v1.ContainerDiskSource{},
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
					{
						Name:    "containerDiskVol",
						Enabled: false,
					},
				},
			},
		}
	}

	BeforeEach(func() {
		ctrl = gomock.NewController(GinkgoT())
		virtClient = kubecli.NewMockKubevirtClient(ctrl)
		config, _, kvStore = testutils.NewFakeClusterConfigUsingKVConfig(&v1.KubeVirtConfiguration{})
		vmInterface = kubecli.NewMockVirtualMachineInterface(ctrl)
		kubevirtClient = fake.NewSimpleClientset()
		virtClient.
			EXPECT().
			VirtualMachine(metav1.NamespaceDefault).
			Return(vmInterface).
			AnyTimes()
		virtClient.
			EXPECT().
			VirtualMachineSnapshot(metav1.NamespaceDefault).
			Return(kubevirtClient.SnapshotV1beta1().VirtualMachineSnapshots(metav1.NamespaceDefault)).
			AnyTimes()
		virtClient.
			EXPECT().
			VirtualMachineSnapshotContent(metav1.NamespaceDefault).
			Return(kubevirtClient.SnapshotV1beta1().VirtualMachineSnapshotContents(metav1.NamespaceDefault)).
			AnyTimes()

		admitter = &VirtualMachineCloneAdmitter{Config: config, Client: virtClient}
		vmClone = newValidClone()
		vm = newValidVM(vmClone.Namespace, vmClone.Spec.Source.Name)
		vmInterface.EXPECT().Get(gomock.Any(), vmClone.Spec.Source.Name, gomock.Any()).Return(vm, nil).AnyTimes()
		vmInterface.EXPECT().Get(gomock.Any(), gomock.Any(), gomock.Any()).Return(nil, fmt.Errorf("does-not-exist")).AnyTimes()

		kubevirtClient.Fake.PrependReactor("*", "*", func(action testing.Action) (handled bool, obj runtime.Object, err error) {
			Expect(action).To(BeNil())
			return true, nil, nil
		})
		kubevirtClient.Fake.PrependReactor("get", "virtualmachinesnapshots", func(action testing.Action) (handled bool, obj runtime.Object, err error) {
			snapshot := &snapshotv1.VirtualMachineSnapshot{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-snapshot",
					Namespace: metav1.NamespaceDefault,
				},
				Status: &snapshotv1.VirtualMachineSnapshotStatus{
					VirtualMachineSnapshotContentName: pointer.P("snapshot-contents"),
				},
			}
			return true, snapshot, nil
		})
		kubevirtClient.Fake.PrependReactor("get", "virtualmachinesnapshotcontents", func(action testing.Action) (handled bool, obj runtime.Object, err error) {
			var volumeBackups []snapshotv1.VolumeBackup
			for _, volume := range vm.Spec.Template.Spec.Volumes {
				volumeBackups = append(volumeBackups, snapshotv1.VolumeBackup{
					VolumeName: volume.Name,
				})
			}

			contents := &snapshotv1.VirtualMachineSnapshotContent{
				Spec: snapshotv1.VirtualMachineSnapshotContentSpec{
					VirtualMachineSnapshotName: pointer.P("test-vm"),
					Source: snapshotv1.SourceSpec{
						VirtualMachine: &snapshotv1.VirtualMachine{
							Spec: vm.Spec,
						},
					},
					VolumeBackups: volumeBackups,
				},
			}
			return true, contents, nil
		})

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
			source.APIGroup = pointer.P("")
			return source
		}),
		Entry("Source with bad kind", func() *k8sv1.TypedLocalObjectReference {
			source := newValidObjReference()
			source.Kind = "Foobar"
			return source
		}),
	)

	Context("source types", func() {

		DescribeTable("should allow legal types", func(kind string) {
			vmClone.Spec.Source.Kind = kind
			admitter.admitAndExpect(vmClone, true)
		},
			Entry("VM", virtualMachineKind),
			Entry("Snapshot", virtualMachineSnapshotKind),
		)

		It("Should reject unknown source type", func() {
			vmClone.Spec.Source.Kind = rand.String(5)
			admitter.admitAndExpect(vmClone, false)
		})
	})

	It("Should reject unknown target type", func() {
		vmClone.Spec.Target.Kind = rand.String(5)
		admitter.admitAndExpect(vmClone, false)
	})

	It("Should allow source VM that does not exist", func() {
		vmClone.Spec.Source.Name = "vm-that-doesnt-exist"
		admitter.admitAndExpect(vmClone, true)
	})

	When("Both source and target kinds are VirtualMachine", func() {
		It("Should reject a target with the same name as the source", func() {
			vmClone.Spec.Source.Kind = virtualMachineKind
			vmClone.Spec.Target.Kind = virtualMachineKind

			vmClone.Spec.Target.Name = vmClone.Spec.Source.Name
			admitter.admitAndExpect(vmClone, false)
		})
	})

	When("Source kind is a VirtualMachineSnapshot and target kind is VirtualMachine", func() {
		It("Should allow the target to have the same name as the source", func() {
			vmClone.Spec.Source.Kind = virtualMachineSnapshotKind
			vmClone.Spec.Target.Kind = virtualMachineKind

			vmClone.Spec.Target.Name = vmClone.Spec.Source.Name
			admitter.admitAndExpect(vmClone, true)
		})
	})

	It("Should reject if snapshot feature gate is not enabled", func() {
		disableFeatureGates()
		admitter.admitAndExpect(vmClone, false)
	})

	DescribeTable("Should allow a source volume not Snapshot-able", func(index int) {
		vm.Status.VolumeSnapshotStatuses[index].Enabled = false
		admitter.admitAndExpect(vmClone, true)
	},
		Entry("DataVolume", 0),
		Entry("PersistentVolumeClaim", 1),
	)

	Context("volume snapshots", func() {
		It("should allow non-PVC/DV volumes that have disabled volume snapshot status", func() {
			volumeName := "ephemeral-volume"
			vm.Spec.Template.Spec.Volumes = []v1.Volume{
				{
					Name:         volumeName,
					VolumeSource: v1.VolumeSource{ContainerDisk: &v1.ContainerDiskSource{}},
				},
			}
			vm.Status.VolumeSnapshotStatuses = []v1.VolumeSnapshotStatus{
				{
					Name:    volumeName,
					Enabled: false,
				},
			}

			admitter.admitAndExpect(vmClone, true)
		})

		It("should allow PVC/DV volumes with disabled volume snapshot status", func() {
			for i := range vm.Status.VolumeSnapshotStatuses {
				vm.Status.VolumeSnapshotStatuses[i].Enabled = false
			}
			admitter.admitAndExpect(vmClone, true)
		})

		It("should allow if vmsnapshot contents don't include a volume's backup", func() {
			vmClone.Spec.Source.Kind = virtualMachineSnapshotKind

			kubevirtClient.Fake.PrependReactor("get", "virtualmachinesnapshotcontents", func(action testing.Action) (handled bool, obj runtime.Object, err error) {
				contents := &snapshotv1.VirtualMachineSnapshotContent{
					Spec: snapshotv1.VirtualMachineSnapshotContentSpec{
						VirtualMachineSnapshotName: pointer.P("test-vm"),
						Source: snapshotv1.SourceSpec{
							VirtualMachine: &snapshotv1.VirtualMachine{
								Spec: vm.Spec,
							},
						},
						VolumeBackups: nil,
					},
				}
				return true, contents, nil
			})

			admitter.admitAndExpect(vmClone, true)
		})
	})

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

	Context("Template Annotations and labels filters", func() {
		testFilter := func(filter string, expectAllowed bool) {
			vmClone.Spec.Template.LabelFilters = []string{filter}
			vmClone.Spec.Template.AnnotationFilters = []string{filter}
			admitter.admitAndExpect(vmClone, expectAllowed)
		}

		DescribeTable("Should reject", func(filter string) {
			testFilter(filter, false)
		},
			Entry("templateFilter negation character alone", "!"),
			Entry("templateFilter negation in the middle", "mykey/!something"),
			Entry("templateFilter negation in the end", "mykey/something!"),
			Entry("templateFilter wildcard in the beginning", "*mykey/something"),
			Entry("templateFilter wildcard in the middle", "mykey/*something"),
		)

		DescribeTable("Should allow", func(filter string) {
			testFilter(filter, true)
		},
			Entry("templateFilter regular filter", "mykey/something"),
			Entry("templateFilter wildcard only", "*"),
			Entry("templateFilter wildcard in the end", "mykey/something*"),
			Entry("templateFilter negation in the beginning", "!mykey/something"),
		)
	})

	DescribeTable("newMacAddresses", func(mac string, expectAllowed bool) {
		vmClone.Spec.NewMacAddresses = map[string]string{
			"default": mac,
		}
		admitter.admitAndExpect(vmClone, expectAllowed)
	},
		Entry("valid mac address", "00:00:00:00:00:00", true),
		Entry("invalid mac address", "00:00:00:00:00", false),
	)

	Context("Custom patches", func() {
		It("Should accept valid JSON patches", func() {
			validPatch := patch.New(patch.WithReplace("/spec/template/spec/domain/devices/interfaces/0/macAddress", "DE-AD-00-FF-FF-FF"))
			p, err := validPatch.GeneratePayload()
			Expect(err).NotTo(HaveOccurred())

			vmClone.Spec.Patches = []string{string(p)}

			admitter.admitAndExpect(vmClone, true)
		})

		It("Should reject invalid JSON patches", func() {
			vmClone.Spec.Patches = []string{ // Missing comma after "replace"
				`{"op": "replace" "path": "/spec/template/spec/domain/devices/interfaces/0/macAddress", "value": "DE-AD-00-FF-FF-FF"}`,
			}

			admitter.admitAndExpect(vmClone, false)
		})
	})

})

func createCloneAdmissionReview(vmClone *clone.VirtualMachineClone) *admissionv1.AdmissionReview {
	policyBytes, _ := json.Marshal(vmClone)

	ar := &admissionv1.AdmissionReview{
		Request: &admissionv1.AdmissionRequest{
			Operation: admissionv1.Create,
			Resource: metav1.GroupVersionResource{
				Group:    clone.VirtualMachineCloneKind.Group,
				Resource: clonebase.ResourceVMClonePlural,
			},
			Object: runtime.RawExtension{
				Raw: policyBytes,
			},
		},
	}

	return ar
}

func (admitter *VirtualMachineCloneAdmitter) admitAndExpect(clone *clone.VirtualMachineClone, expectAllowed bool) {
	ar := createCloneAdmissionReview(clone)
	resp := admitter.Admit(context.Background(), ar)
	Expect(resp.Allowed).To(Equal(expectAllowed))
}

func newValidClone() *clone.VirtualMachineClone {
	vmClone := kubecli.NewMinimalCloneWithNS("testclone", metav1.NamespaceDefault)
	vmClone.Spec.Source = newValidObjReference()
	vmClone.Spec.Target = newValidObjReference()
	vmClone.Spec.Target.Name = "clone-target-vm"

	return vmClone
}

func newValidObjReference() *k8sv1.TypedLocalObjectReference {
	return &k8sv1.TypedLocalObjectReference{
		APIGroup: pointer.P(core.GroupName),
		Kind:     virtualMachineKind,
		Name:     "clone-source-vm",
	}
}
