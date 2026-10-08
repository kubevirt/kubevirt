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

package admitters_test

import (
	"context"
	"encoding/json"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"go.uber.org/mock/gomock"

	admissionv1 "k8s.io/api/admission/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"

	v1 "kubevirt.io/api/core/v1"
	snapshotv1 "kubevirt.io/api/snapshot/v1beta1"
	"kubevirt.io/client-go/kubecli"

	"kubevirt.io/kubevirt/pkg/pointer"
	"kubevirt.io/kubevirt/pkg/storage/admitters"
	"kubevirt.io/kubevirt/pkg/virt-api/webhooks"
)

var _ = Describe("Validating VirtualMachineSnapshot Admitter", func() {
	vmName := "vm"
	apiGroup := "kubevirt.io"

	Context("With a disabled feature gate", func() {
		It("should reject anything", func() {
			snapshot := &snapshotv1.VirtualMachineSnapshot{
				Spec: snapshotv1.VirtualMachineSnapshotSpec{},
			}

			ar := createSnapshotAdmissionReview(snapshot)
			resp := createTestVMSnapshotAdmitter(stubVMSnapshotConfigChecker{}, nil).Admit(context.Background(), ar)
			Expect(resp.Allowed).To(BeFalse())
			Expect(resp.Result.Message).Should(Equal("snapshot feature gate not enabled"))
		})
	})

	Context("With feature gate enabled", func() {
		It("should reject invalid request resource", func() {
			ar := &admissionv1.AdmissionReview{
				Request: &admissionv1.AdmissionRequest{
					Resource: webhooks.VirtualMachineGroupVersionResource,
				},
			}

			resp := createTestVMSnapshotAdmitter(stubVMSnapshotConfigChecker{snapshotEnabled: true}, nil).Admit(context.Background(), ar)
			Expect(resp.Allowed).To(BeFalse())
			Expect(resp.Result.Message).Should(ContainSubstring("unexpected resource"))
		})

		It("should reject missing apigroup", func() {
			snapshot := &snapshotv1.VirtualMachineSnapshot{
				Spec: snapshotv1.VirtualMachineSnapshotSpec{},
			}

			ar := createSnapshotAdmissionReview(snapshot)
			resp := createTestVMSnapshotAdmitter(stubVMSnapshotConfigChecker{snapshotEnabled: true}, nil).Admit(context.Background(), ar)
			Expect(resp.Allowed).To(BeFalse())
			Expect(resp.Result.Details.Causes).To(HaveLen(1))
			Expect(resp.Result.Details.Causes[0].Field).To(Equal("spec.source.apiGroup"))
		})

		It("should allow when VM does not exist", func() {
			snapshot := &snapshotv1.VirtualMachineSnapshot{
				Spec: snapshotv1.VirtualMachineSnapshotSpec{
					Source: corev1.TypedLocalObjectReference{
						APIGroup: &apiGroup,
						Kind:     "VirtualMachine",
						Name:     vmName,
					},
				},
			}

			ar := createSnapshotAdmissionReview(snapshot)
			resp := createTestVMSnapshotAdmitter(stubVMSnapshotConfigChecker{snapshotEnabled: true}, nil).Admit(context.Background(), ar)
			Expect(resp.Allowed).To(BeTrue())
		})

		DescribeTable("should admit External mode only against a running VM", func(vmi *v1.VirtualMachineInstance, allowed bool) {
			snapshot := &snapshotv1.VirtualMachineSnapshot{
				Spec: snapshotv1.VirtualMachineSnapshotSpec{
					Source: corev1.TypedLocalObjectReference{
						APIGroup: &apiGroup,
						Kind:     "VirtualMachine",
						Name:     vmName,
					},
					SnapshotMode: pointer.P(snapshotv1.SnapshotModeExternal),
				},
			}

			config := stubVMSnapshotConfigChecker{snapshotEnabled: true, externalVMSnapshotEnabled: true, utilityVolumesEnabled: true}
			ar := createSnapshotAdmissionReview(snapshot)
			resp := createTestVMSnapshotAdmitter(config, nil, vmi).Admit(context.Background(), ar)

			Expect(resp.Allowed).To(Equal(allowed))
			if !allowed {
				Expect(resp.Result.Details.Causes[0].Message).To(ContainSubstring("requires a running VirtualMachine"))
			}
		},
			Entry("running", runningVMI(), true),
			Entry("stopped", stoppedVMI(), false),
			Entry("no VMI at all", nil, false),
		)

		DescribeTable("should reject External mode for the disks libvirt cannot overlay", func(rejections []string, disks ...v1.Disk) {
			snapshot := &snapshotv1.VirtualMachineSnapshot{
				Spec: snapshotv1.VirtualMachineSnapshotSpec{
					Source: corev1.TypedLocalObjectReference{
						APIGroup: &apiGroup,
						Kind:     "VirtualMachine",
						Name:     vmName,
					},
					SnapshotMode: pointer.P(snapshotv1.SnapshotModeExternal),
				},
			}
			vmi := runningVMI()
			vmi.Spec.Domain.Devices.Disks = append([]v1.Disk{
				{Name: "rootdisk", DiskDevice: v1.DiskDevice{Disk: &v1.DiskTarget{}}},
			}, disks...)

			config := stubVMSnapshotConfigChecker{snapshotEnabled: true, externalVMSnapshotEnabled: true, utilityVolumesEnabled: true}
			ar := createSnapshotAdmissionReview(snapshot)
			resp := createTestVMSnapshotAdmitter(config, nil, vmi).Admit(context.Background(), ar)

			Expect(resp.Allowed).To(Equal(len(rejections) == 0))
			if len(rejections) > 0 {
				var messages []string
				for _, cause := range resp.Result.Details.Causes {
					messages = append(messages, cause.Message)
				}
				Expect(messages).To(Equal(rejections))
			}
		},
			Entry("a LUN disk", []string{"External mode does not support the LUN disk lundisk"},
				v1.Disk{Name: "lundisk", DiskDevice: v1.DiskDevice{LUN: &v1.LunTarget{}}}),
			Entry("a shareable disk", []string{"External mode does not support the shareable disk shared"},
				v1.Disk{Name: "shared", Shareable: pointer.P(true), DiskDevice: v1.DiskDevice{Disk: &v1.DiskTarget{}}}),
			Entry("every such disk", []string{
				"External mode does not support the LUN disk lundisk",
				"External mode does not support the shareable disk shared",
			},
				v1.Disk{Name: "lundisk", DiskDevice: v1.DiskDevice{LUN: &v1.LunTarget{}}},
				v1.Disk{Name: "shared", Shareable: pointer.P(true), DiskDevice: v1.DiskDevice{Disk: &v1.DiskTarget{}}}),
			Entry("not a read-only shareable disk", nil,
				v1.Disk{Name: "shared", Shareable: pointer.P(true), DiskDevice: v1.DiskDevice{Disk: &v1.DiskTarget{ReadOnly: true}}}),
		)

		It("should admit Direct mode without a running VM", func() {
			snapshot := &snapshotv1.VirtualMachineSnapshot{
				Spec: snapshotv1.VirtualMachineSnapshotSpec{
					Source: corev1.TypedLocalObjectReference{
						APIGroup: &apiGroup,
						Kind:     "VirtualMachine",
						Name:     vmName,
					},
					SnapshotMode: pointer.P(snapshotv1.SnapshotModeDirect),
				},
			}

			ar := createSnapshotAdmissionReview(snapshot)
			resp := createTestVMSnapshotAdmitter(stubVMSnapshotConfigChecker{snapshotEnabled: true}, nil, nil).
				Admit(context.Background(), ar)
			Expect(resp.Allowed).To(BeTrue())
		})

		DescribeTable("should admit snapshotMode according to the ExternalVMSnapshot feature gate", func(mode snapshotv1.SnapshotMode, externalEnabled, utilityEnabled bool, rejection string) {
			snapshot := &snapshotv1.VirtualMachineSnapshot{
				Spec: snapshotv1.VirtualMachineSnapshotSpec{
					Source: corev1.TypedLocalObjectReference{
						APIGroup: &apiGroup,
						Kind:     "VirtualMachine",
						Name:     vmName,
					},
					SnapshotMode: pointer.P(mode),
				},
			}

			config := stubVMSnapshotConfigChecker{snapshotEnabled: true, externalVMSnapshotEnabled: externalEnabled, utilityVolumesEnabled: utilityEnabled}
			ar := createSnapshotAdmissionReview(snapshot)
			resp := createTestVMSnapshotAdmitter(config, nil, runningVMI()).Admit(context.Background(), ar)
			Expect(resp.Allowed).To(Equal(rejection == ""))
			if rejection != "" {
				Expect(resp.Result.Message).Should(Equal(rejection))
			}
		},
			Entry("reject External when the gate is disabled", snapshotv1.SnapshotModeExternal, false, true, "ExternalVMSnapshot feature gate not enabled"),
			Entry("reject External without the UtilityVolumes gate", snapshotv1.SnapshotModeExternal, true, false, "ExternalVMSnapshot requires the UtilityVolumes feature gate"),
			Entry("allow External when both gates are enabled", snapshotv1.SnapshotModeExternal, true, true, ""),
			Entry("allow Direct when the gates are disabled", snapshotv1.SnapshotModeDirect, false, false, ""),
		)

		It("should reject spec update", func() {
			snapshot := &snapshotv1.VirtualMachineSnapshot{
				Spec: snapshotv1.VirtualMachineSnapshotSpec{
					Source: corev1.TypedLocalObjectReference{
						APIGroup: &apiGroup,
						Kind:     "VirtualMachine",
						Name:     vmName,
					},
				},
			}

			oldSnapshot := &snapshotv1.VirtualMachineSnapshot{
				Spec: snapshotv1.VirtualMachineSnapshotSpec{
					Source: corev1.TypedLocalObjectReference{
						APIGroup: &apiGroup,
						Kind:     "VirtualMachine",
						Name:     "baz",
					},
				},
			}

			ar := createSnapshotUpdateAdmissionReview(oldSnapshot, snapshot)
			resp := createTestVMSnapshotAdmitter(stubVMSnapshotConfigChecker{snapshotEnabled: true}, nil).Admit(context.Background(), ar)
			Expect(resp.Allowed).To(BeFalse())
			Expect(resp.Result.Details.Causes).To(HaveLen(1))
			Expect(resp.Result.Details.Causes[0].Field).To(Equal("spec"))
		})

		It("should allow metadata update", func() {
			oldSnapshot := &snapshotv1.VirtualMachineSnapshot{
				Spec: snapshotv1.VirtualMachineSnapshotSpec{
					Source: corev1.TypedLocalObjectReference{
						APIGroup: &apiGroup,
						Kind:     "VirtualMachine",
						Name:     vmName,
					},
				},
			}

			snapshot := &snapshotv1.VirtualMachineSnapshot{
				ObjectMeta: metav1.ObjectMeta{
					Finalizers: []string{"finalizer"},
				},
				Spec: snapshotv1.VirtualMachineSnapshotSpec{
					Source: corev1.TypedLocalObjectReference{
						APIGroup: &apiGroup,
						Kind:     "VirtualMachine",
						Name:     vmName,
					},
				},
			}

			ar := createSnapshotUpdateAdmissionReview(oldSnapshot, snapshot)
			resp := createTestVMSnapshotAdmitter(stubVMSnapshotConfigChecker{snapshotEnabled: true}, nil).Admit(context.Background(), ar)
			Expect(resp.Allowed).To(BeTrue())
		})

		Context("when VirtualMachine exists", func() {
			var vm *v1.VirtualMachine

			BeforeEach(func() {
				vm = &v1.VirtualMachine{
					ObjectMeta: metav1.ObjectMeta{
						Name: vmName,
					},
				}
			})

			It("should accept when VM is running", func() {
				snapshot := &snapshotv1.VirtualMachineSnapshot{
					Spec: snapshotv1.VirtualMachineSnapshotSpec{
						Source: corev1.TypedLocalObjectReference{
							APIGroup: &apiGroup,
							Kind:     "VirtualMachine",
							Name:     vmName,
						},
					},
				}

				vm.Spec.RunStrategy = pointer.P(v1.RunStrategyAlways)

				ar := createSnapshotAdmissionReview(snapshot)
				resp := createTestVMSnapshotAdmitter(stubVMSnapshotConfigChecker{snapshotEnabled: true}, vm).Admit(context.Background(), ar)
				Expect(resp.Allowed).To(BeTrue())
			})

			It("should reject invalid kind", func() {
				snapshot := &snapshotv1.VirtualMachineSnapshot{
					Spec: snapshotv1.VirtualMachineSnapshotSpec{
						Source: corev1.TypedLocalObjectReference{
							APIGroup: &apiGroup,
							Kind:     "VirtualMachineInstance",
							Name:     vmName,
						},
					},
				}

				vm.Spec.RunStrategy = pointer.P(v1.RunStrategyAlways)

				ar := createSnapshotAdmissionReview(snapshot)
				resp := createTestVMSnapshotAdmitter(stubVMSnapshotConfigChecker{snapshotEnabled: true}, vm).Admit(context.Background(), ar)
				Expect(resp.Allowed).To(BeFalse())
				Expect(resp.Result.Details.Causes).To(HaveLen(1))
				Expect(resp.Result.Details.Causes[0].Field).To(Equal("spec.source.kind"))
			})

			It("should reject invalid apiGroup", func() {
				g := "foo.bar"
				snapshot := &snapshotv1.VirtualMachineSnapshot{
					Spec: snapshotv1.VirtualMachineSnapshotSpec{
						Source: corev1.TypedLocalObjectReference{
							APIGroup: &g,
							Kind:     "VirtualMachine",
							Name:     vmName,
						},
					},
				}

				vm.Spec.RunStrategy = pointer.P(v1.RunStrategyAlways)

				ar := createSnapshotAdmissionReview(snapshot)
				resp := createTestVMSnapshotAdmitter(stubVMSnapshotConfigChecker{snapshotEnabled: true}, vm).Admit(context.Background(), ar)
				Expect(resp.Allowed).To(BeFalse())
				Expect(resp.Result.Details.Causes).To(HaveLen(1))
				Expect(resp.Result.Details.Causes[0].Field).To(Equal("spec.source.apiGroup"))
			})

			DescribeTable("should accept persistent storage with both offline and online snapshot", func(runStrategy v1.VirtualMachineRunStrategy) {
				vm.Spec.Template = &v1.VirtualMachineInstanceTemplateSpec{
					Spec: v1.VirtualMachineInstanceSpec{
						Domain: v1.DomainSpec{
							Devices: v1.Devices{
								TPM: &v1.TPMDevice{
									Persistent: pointer.P(true),
								},
							},
						},
					},
				}
				vm.Spec.RunStrategy = pointer.P(runStrategy)
				snapshot := &snapshotv1.VirtualMachineSnapshot{
					Spec: snapshotv1.VirtualMachineSnapshotSpec{
						Source: corev1.TypedLocalObjectReference{
							APIGroup: &apiGroup,
							Kind:     "VirtualMachine",
							Name:     vmName,
						},
					},
				}

				ar := createSnapshotAdmissionReview(snapshot)
				resp := createTestVMSnapshotAdmitter(stubVMSnapshotConfigChecker{snapshotEnabled: true}, vm).Admit(context.Background(), ar)
				Expect(resp.Allowed).To(BeTrue())
			},
				Entry("when VM is running", v1.RunStrategyAlways),
				Entry("when VM is halted", v1.RunStrategyHalted),
			)

			It("should accept when VM is not running", func() {
				snapshot := &snapshotv1.VirtualMachineSnapshot{
					Spec: snapshotv1.VirtualMachineSnapshotSpec{
						Source: corev1.TypedLocalObjectReference{
							APIGroup: &apiGroup,
							Kind:     "VirtualMachine",
							Name:     vmName,
						},
					},
				}

				vm.Spec.RunStrategy = pointer.P(v1.RunStrategyHalted)

				ar := createSnapshotAdmissionReview(snapshot)
				resp := createTestVMSnapshotAdmitter(stubVMSnapshotConfigChecker{snapshotEnabled: true}, vm).Admit(context.Background(), ar)
				Expect(resp.Allowed).To(BeTrue())
			})
		})
	})
})

func createSnapshotAdmissionReview(snapshot *snapshotv1.VirtualMachineSnapshot) *admissionv1.AdmissionReview {
	bytes, _ := json.Marshal(snapshot)

	ar := &admissionv1.AdmissionReview{
		Request: &admissionv1.AdmissionRequest{
			Operation: admissionv1.Create,
			Namespace: "foo",
			Resource: metav1.GroupVersionResource{
				Group:    "snapshot.kubevirt.io",
				Resource: "virtualmachinesnapshots",
			},
			Object: runtime.RawExtension{
				Raw: bytes,
			},
		},
	}

	return ar
}

func createSnapshotUpdateAdmissionReview(old, current *snapshotv1.VirtualMachineSnapshot) *admissionv1.AdmissionReview {
	oldBytes, _ := json.Marshal(old)
	currentBytes, _ := json.Marshal(current)

	ar := &admissionv1.AdmissionReview{
		Request: &admissionv1.AdmissionRequest{
			Operation: admissionv1.Update,
			Namespace: "foo",
			Resource: metav1.GroupVersionResource{
				Group:    "snapshot.kubevirt.io",
				Resource: "virtualmachinesnapshots",
			},
			Object: runtime.RawExtension{
				Raw: currentBytes,
			},
			OldObject: runtime.RawExtension{
				Raw: oldBytes,
			},
		},
	}

	return ar
}

func runningVMI() *v1.VirtualMachineInstance {
	vmi := &v1.VirtualMachineInstance{ObjectMeta: metav1.ObjectMeta{Name: "vm"}}
	vmi.Status.Phase = v1.Running
	return vmi
}

func stoppedVMI() *v1.VirtualMachineInstance {
	vmi := &v1.VirtualMachineInstance{ObjectMeta: metav1.ObjectMeta{Name: "vm"}}
	vmi.Status.Phase = v1.Succeeded
	return vmi
}

func createTestVMSnapshotAdmitter(config stubVMSnapshotConfigChecker, vm *v1.VirtualMachine, vmi ...*v1.VirtualMachineInstance) *admitters.VMSnapshotAdmitter {
	ctrl := gomock.NewController(GinkgoT())
	virtClient := kubecli.NewMockKubevirtClient(ctrl)
	vmInterface := kubecli.NewMockVirtualMachineInterface(ctrl)
	virtClient.EXPECT().VirtualMachine(gomock.Any()).Return(vmInterface).AnyTimes()

	vmiInterface := kubecli.NewMockVirtualMachineInstanceInterface(ctrl)
	virtClient.EXPECT().VirtualMachineInstance(gomock.Any()).Return(vmiInterface).AnyTimes()
	if len(vmi) == 0 || vmi[0] == nil {
		notFound := errors.NewNotFound(schema.GroupResource{Group: "kubevirt.io", Resource: "virtualmachineinstances"}, "foo")
		vmiInterface.EXPECT().Get(gomock.Any(), gomock.Any(), gomock.Any()).Return(nil, notFound).AnyTimes()
	} else {
		vmiInterface.EXPECT().Get(gomock.Any(), gomock.Any(), gomock.Any()).Return(vmi[0], nil).AnyTimes()
	}
	if vm == nil {
		err := errors.NewNotFound(schema.GroupResource{Group: "kubevirt.io", Resource: "virtualmachines"}, "foo")
		vmInterface.EXPECT().Get(gomock.Any(), gomock.Any(), gomock.Any()).Return(nil, err).AnyTimes()
	} else {
		vmInterface.EXPECT().Get(gomock.Any(), vm.Name, gomock.Any()).Return(vm, nil).AnyTimes()
	}
	return admitters.NewVMSnapshotAdmitter(config, virtClient)
}

type stubVMSnapshotConfigChecker struct {
	snapshotEnabled           bool
	externalVMSnapshotEnabled bool
	utilityVolumesEnabled     bool
}

func (s stubVMSnapshotConfigChecker) SnapshotEnabled() bool { return s.snapshotEnabled }

func (s stubVMSnapshotConfigChecker) ExternalVMSnapshotEnabled() bool {
	return s.externalVMSnapshotEnabled
}

func (s stubVMSnapshotConfigChecker) UtilityVolumesEnabled() bool {
	return s.utilityVolumesEnabled
}
