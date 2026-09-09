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

package lifecycle

import (
	"context"
	"fmt"
	"net/http"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"go.uber.org/mock/gomock"

	k8sv1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	k8smetav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/uuid"
	"k8s.io/client-go/kubernetes/fake"
	"k8s.io/client-go/testing"

	v1 "kubevirt.io/api/core/v1"
	"kubevirt.io/client-go/kubecli"

	"kubevirt.io/kubevirt/pkg/apimachinery/patch"
	"kubevirt.io/kubevirt/pkg/controller"
	"kubevirt.io/kubevirt/pkg/libvmi"
	"kubevirt.io/kubevirt/pkg/pointer"
)

const (
	Running    = true
	Paused     = true
	NotRunning = false
	testVMName = "testvm"
)

func withDryRun() []string {
	return []string{k8smetav1.DryRunAll}
}

var _ = Describe("VirtualMachine lifecycle", func() {
	var (
		ctrl          *gomock.Controller
		kubeClient    *fake.Clientset
		virtClient    *kubecli.MockKubevirtClient
		vmClient      *kubecli.MockVirtualMachineInterface
		vmiClient     *kubecli.MockVirtualMachineInstanceInterface
		migrateClient *kubecli.MockVirtualMachineInstanceMigrationInterface
		handler       *Handler
	)

	gracePeriodZero := pointer.P(int64(0))

	BeforeEach(func() {
		ctrl = gomock.NewController(GinkgoT())
		kubeClient = fake.NewSimpleClientset()
		virtClient = kubecli.NewMockKubevirtClient(ctrl)
		vmClient = kubecli.NewMockVirtualMachineInterface(ctrl)
		vmiClient = kubecli.NewMockVirtualMachineInstanceInterface(ctrl)
		migrateClient = kubecli.NewMockVirtualMachineInstanceMigrationInterface(ctrl)

		virtClient.EXPECT().CoreV1().Return(kubeClient.CoreV1()).AnyTimes()
		virtClient.EXPECT().VirtualMachine(k8smetav1.NamespaceDefault).Return(vmClient).AnyTimes()
		virtClient.EXPECT().VirtualMachine("").Return(vmClient).AnyTimes()
		virtClient.EXPECT().VirtualMachineInstance(k8smetav1.NamespaceDefault).Return(vmiClient).AnyTimes()
		virtClient.EXPECT().VirtualMachineInstance("").Return(vmiClient).AnyTimes()
		virtClient.EXPECT().VirtualMachineInstanceMigration(k8smetav1.NamespaceDefault).Return(migrateClient).AnyTimes()

		handler = NewHandler(virtClient)

		kubeClient.Fake.PrependReactor("*", "*", func(action testing.Action) (handled bool, obj runtime.Object, err error) {
			Expect(action).To(BeNil())
			return true, nil, nil
		})
	})

	expectStatusError := func(statusErr *errors.StatusError, code int) {
		Expect(statusErr).To(HaveOccurred())
		Expect(statusErr.Status().Code).To(BeNumerically("==", code))
	}

	Context("restart", func() {
		It("should fail if VirtualMachine not exists", func() {
			vmClient.EXPECT().Get(context.Background(), testVMName, k8smetav1.GetOptions{}).Return(nil, errors.NewNotFound(v1.Resource("virtualmachine"), testVMName))

			statusErr := handler.RestartVM(context.Background(), k8smetav1.NamespaceDefault, testVMName, &v1.RestartOptions{})
			expectStatusError(statusErr, http.StatusNotFound)
		})

		DescribeTable("should return an error when VM is not running", func(errMsg string, running *bool, runStrategy *v1.VirtualMachineRunStrategy) {
			vm := v1.VirtualMachine{
				Spec: v1.VirtualMachineSpec{
					Running:     running,
					RunStrategy: runStrategy,
				},
			}

			vmClient.EXPECT().Get(context.Background(), testVMName, k8smetav1.GetOptions{}).Return(&vm, nil)

			if runStrategy != nil && *runStrategy == v1.RunStrategyManual {
				vmiClient.EXPECT().Get(context.Background(), testVMName, k8smetav1.GetOptions{}).Return(nil, errors.NewNotFound(v1.Resource("virtualmachineinstance"), vm.Name))
			}
			statusErr := handler.RestartVM(context.Background(), k8smetav1.NamespaceDefault, testVMName, &v1.RestartOptions{})

			expectStatusError(statusErr, http.StatusConflict)
			Expect(statusErr.Error()).To(ContainSubstring(errMsg))
		},
			Entry("with Running field", "RunStategy Halted does not support manual restart requests", pointer.P(NotRunning), nil),
			Entry("with RunStrategyHalted", "RunStategy Halted does not support manual restart requests", nil, pointer.P(v1.RunStrategyHalted)),
			Entry("with RunStrategyManual", "VM is not running: Halted", nil, pointer.P(v1.RunStrategyManual)),
		)

		DescribeTable("should ForceRestart VirtualMachine according to options", func(terminationGracePeriod *int64, restartOptions *v1.RestartOptions) {
			vm := newVirtualMachineWithRunning(pointer.P(Running))
			vmi := v1.VirtualMachineInstance{
				ObjectMeta: k8smetav1.ObjectMeta{
					Name:      testVMName,
					Namespace: k8smetav1.NamespaceDefault,
				},
				Spec: v1.VirtualMachineInstanceSpec{
					TerminationGracePeriodSeconds: terminationGracePeriod,
				},
			}
			vmi.ObjectMeta.SetUID(uuid.NewUUID())

			vmClient.EXPECT().Get(context.Background(), vm.Name, k8smetav1.GetOptions{}).Return(vm, nil)
			vmiClient.EXPECT().Get(context.Background(), vm.Name, k8smetav1.GetOptions{}).Return(&vmi, nil)
			gomock.InOrder(
				vmiClient.EXPECT().Patch(context.Background(), vmi.Name, types.JSONPatchType, gomock.Any(), gomock.Any()).DoAndReturn(
					func(ctx context.Context, name string, patchType types.PatchType, data []byte, opts k8smetav1.PatchOptions, _ ...string) (interface{}, interface{}) {
						Expect(opts.DryRun).To(BeEquivalentTo(restartOptions.DryRun))

						patchSet := patch.New()
						if terminationGracePeriod != nil {
							patchSet.AddOption(patch.WithTest("/spec/terminationGracePeriodSeconds", *terminationGracePeriod))
						} else {
							patchSet.AddOption(patch.WithTest("/spec/terminationGracePeriodSeconds", nil))
						}
						patchSet.AddOption(patch.WithReplace("/spec/terminationGracePeriodSeconds", int64(1)))
						patchBytes, err := patchSet.GeneratePayload()
						Expect(err).ToNot(HaveOccurred())
						Expect(string(data)).To(Equal(string(patchBytes)))
						return &vmi, nil
					}),
				vmClient.EXPECT().PatchStatus(context.Background(), vm.Name, types.JSONPatchType, gomock.Any(), gomock.Any()).DoAndReturn(
					func(ctx context.Context, name string, patchType types.PatchType, body interface{}, opts k8smetav1.PatchOptions) (interface{}, interface{}) {
						Expect(opts.DryRun).To(BeEquivalentTo(restartOptions.DryRun))
						return vm, nil
					}),
			)

			statusErr := handler.RestartVM(context.Background(), k8smetav1.NamespaceDefault, testVMName, restartOptions)
			Expect(statusErr).ToNot(HaveOccurred())
		},
			Entry("with non-nil terminationGracePeriod", pointer.P(int64(600)), &v1.RestartOptions{GracePeriodSeconds: gracePeriodZero}),
			Entry("with nil terminationGracePeriod", nil, &v1.RestartOptions{GracePeriodSeconds: gracePeriodZero}),
			Entry("with dry-run option", pointer.P(int64(600)), &v1.RestartOptions{GracePeriodSeconds: gracePeriodZero, DryRun: withDryRun()}),
		)

		It("should return error when VMI terminationGracePeriod patch fails during ForceRestart", func() {
			restartOptions := &v1.RestartOptions{GracePeriodSeconds: gracePeriodZero}

			vm := newVirtualMachineWithRunning(pointer.P(Running))
			var terminationGracePeriodSeconds int64 = 600
			vmi := v1.VirtualMachineInstance{
				ObjectMeta: k8smetav1.ObjectMeta{
					Name:      testVMName,
					Namespace: k8smetav1.NamespaceDefault,
				},
				Spec: v1.VirtualMachineInstanceSpec{
					TerminationGracePeriodSeconds: &terminationGracePeriodSeconds,
				},
			}
			vmi.ObjectMeta.SetUID(uuid.NewUUID())

			vmClient.EXPECT().Get(context.Background(), vm.Name, k8smetav1.GetOptions{}).Return(vm, nil)
			vmiClient.EXPECT().Get(context.Background(), vm.Name, k8smetav1.GetOptions{}).Return(&vmi, nil)
			vmiClient.EXPECT().Patch(context.Background(), vmi.Name, types.JSONPatchType, gomock.Any(), gomock.Any()).Return(nil, fmt.Errorf("optimistic locking conflict"))

			statusErr := handler.RestartVM(context.Background(), k8smetav1.NamespaceDefault, testVMName, restartOptions)
			expectStatusError(statusErr, http.StatusInternalServerError)
		})

		It("should rollback VMI terminationGracePeriod when VM PatchStatus fails after successful VMI patch", func() {
			restartOptions := &v1.RestartOptions{GracePeriodSeconds: gracePeriodZero}

			vm := newVirtualMachineWithRunning(pointer.P(Running))
			var terminationGracePeriodSeconds int64 = 600
			vmi := v1.VirtualMachineInstance{
				ObjectMeta: k8smetav1.ObjectMeta{
					Name:      testVMName,
					Namespace: k8smetav1.NamespaceDefault,
				},
				Spec: v1.VirtualMachineInstanceSpec{
					TerminationGracePeriodSeconds: &terminationGracePeriodSeconds,
				},
			}
			vmi.ObjectMeta.SetUID(uuid.NewUUID())

			vmClient.EXPECT().Get(context.Background(), vm.Name, k8smetav1.GetOptions{}).Return(vm, nil)
			vmiClient.EXPECT().Get(context.Background(), vm.Name, k8smetav1.GetOptions{}).Return(&vmi, nil)
			gomock.InOrder(
				vmiClient.EXPECT().Patch(context.Background(), vmi.Name, types.JSONPatchType, gomock.Any(), gomock.Any()).Return(&vmi, nil),
				vmClient.EXPECT().PatchStatus(context.Background(), vm.Name, types.JSONPatchType, gomock.Any(), gomock.Any()).Return(nil, fmt.Errorf("status patch conflict")),
				vmiClient.EXPECT().Patch(context.Background(), vmi.Name, types.JSONPatchType, gomock.Any(), gomock.Any()).DoAndReturn(
					func(ctx context.Context, name string, patchType types.PatchType, data []byte, opts k8smetav1.PatchOptions, _ ...string) (interface{}, interface{}) {
						patchSet := patch.New()
						patchSet.AddOption(patch.WithTest("/spec/terminationGracePeriodSeconds", int64(1)))
						patchSet.AddOption(patch.WithReplace("/spec/terminationGracePeriodSeconds", terminationGracePeriodSeconds))
						patchBytes, err := patchSet.GeneratePayload()
						Expect(err).ToNot(HaveOccurred())
						Expect(string(data)).To(Equal(string(patchBytes)))
						return &vmi, nil
					}),
			)

			statusErr := handler.RestartVM(context.Background(), k8smetav1.NamespaceDefault, testVMName, restartOptions)
			expectStatusError(statusErr, http.StatusInternalServerError)
		})

		It("should restart VirtualMachine", func() {
			vm := newVirtualMachineWithRunning(pointer.P(Running))

			vmi := v1.VirtualMachineInstance{
				Spec: v1.VirtualMachineInstanceSpec{},
			}

			vmi.ObjectMeta.SetUID(uuid.NewUUID())
			vmClient.EXPECT().Get(context.Background(), vm.Name, k8smetav1.GetOptions{}).Return(vm, nil)
			vmiClient.EXPECT().Get(context.Background(), vm.Name, k8smetav1.GetOptions{}).Return(&vmi, nil)
			vmClient.EXPECT().PatchStatus(context.Background(), vm.Name, types.JSONPatchType, gomock.Any(), k8smetav1.PatchOptions{}).Return(vm, nil)

			statusErr := handler.RestartVM(context.Background(), k8smetav1.NamespaceDefault, testVMName, &v1.RestartOptions{})
			Expect(statusErr).ToNot(HaveOccurred())
		})

		It("should start VirtualMachine if VMI doesn't exist", func() {
			vm := newVirtualMachineWithRunning(pointer.P(Running))
			vmi := newVirtualMachineInstanceInPhase(v1.Running)

			vmClient.EXPECT().Get(context.Background(), vm.Name, k8smetav1.GetOptions{}).Return(vm, nil)
			vmiClient.EXPECT().Get(context.Background(), vm.Name, k8smetav1.GetOptions{}).Return(vmi, nil)
			vmClient.EXPECT().PatchStatus(context.Background(), vm.Name, types.JSONPatchType, gomock.Any(), k8smetav1.PatchOptions{}).Return(vm, nil)

			statusErr := handler.RestartVM(context.Background(), k8smetav1.NamespaceDefault, testVMName, &v1.RestartOptions{})
			Expect(statusErr).ToNot(HaveOccurred())
		})

		It("should fail when the volume migration in ongoing", func() {
			vmi := libvmi.New()
			vm := libvmi.NewVirtualMachine(vmi)
			controller.NewVirtualMachineConditionManager().UpdateCondition(vm, &v1.VirtualMachineCondition{
				Type:   v1.VirtualMachineConditionType(v1.VirtualMachineInstanceVolumesChange),
				Status: k8sv1.ConditionTrue,
			})
			vmClient.EXPECT().Get(context.Background(), vm.Name, k8smetav1.GetOptions{}).Return(vm, nil)

			statusErr := handler.RestartVM(context.Background(), k8smetav1.NamespaceDefault, vm.Name, &v1.RestartOptions{})
			expectStatusError(statusErr, http.StatusConflict)
			Expect(statusErr.Error()).To(ContainSubstring("VM recovery required"))
		})
	})

	Context("stop", func() {
		DescribeTable("should ForceStop VirtualMachine according to options", func(statusPhase v1.VirtualMachineInstancePhase, stopOptions *v1.StopOptions) {
			var terminationGracePeriodSeconds int64 = 1800

			vm := newVirtualMachineWithRunning(pointer.P(Running))

			vmi := v1.VirtualMachineInstance{
				ObjectMeta: k8smetav1.ObjectMeta{
					Name:      testVMName,
					Namespace: k8smetav1.NamespaceDefault,
				},
				Spec: v1.VirtualMachineInstanceSpec{
					TerminationGracePeriodSeconds: &terminationGracePeriodSeconds,
				},
				Status: v1.VirtualMachineInstanceStatus{
					Phase: statusPhase,
				},
			}
			vmi.ObjectMeta.SetUID(uuid.NewUUID())

			vmClient.EXPECT().Get(context.Background(), vm.Name, k8smetav1.GetOptions{}).Return(vm, nil)
			vmiClient.EXPECT().Get(context.Background(), vmi.Name, k8smetav1.GetOptions{}).Return(&vmi, nil)
			vmiClient.EXPECT().Patch(context.Background(), vmi.Name, types.JSONPatchType, gomock.Any(), gomock.Any()).DoAndReturn(
				func(ctx context.Context, name string, patchType types.PatchType, body interface{}, opts k8smetav1.PatchOptions, _ ...string) (interface{}, interface{}) {
					Expect(opts.DryRun).To(BeEquivalentTo(stopOptions.DryRun))
					return &vmi, nil
				}).AnyTimes()
			vmClient.EXPECT().Patch(context.Background(), vm.Name, types.JSONPatchType, gomock.Any(), gomock.Any()).DoAndReturn(
				func(ctx context.Context, name string, patchType types.PatchType, body interface{}, opts k8smetav1.PatchOptions, _ ...string) (interface{}, interface{}) {
					Expect(opts.DryRun).To(BeEquivalentTo(stopOptions.DryRun))
					return vm, nil
				})

			statusErr := handler.StopVM(context.Background(), k8smetav1.NamespaceDefault, testVMName, stopOptions)
			Expect(statusErr).ToNot(HaveOccurred())
		},
			Entry("in status Running with default", v1.Running, &v1.StopOptions{GracePeriod: gracePeriodZero}),
			Entry("in status Failed with default", v1.Failed, &v1.StopOptions{GracePeriod: gracePeriodZero}),
			Entry("in status Running with dry-run", v1.Running, &v1.StopOptions{GracePeriod: gracePeriodZero, DryRun: withDryRun()}),
			Entry("in status Failed with dry-run", v1.Failed, &v1.StopOptions{GracePeriod: gracePeriodZero, DryRun: withDryRun()}),
		)
	})

	Context("error handling for RestartVM", func() {
		It("should fail on VM with RunStrategyHalted", func() {
			vm := newVirtualMachineWithRunStrategy(v1.RunStrategyHalted)

			vmClient.EXPECT().Get(context.Background(), vm.Name, k8smetav1.GetOptions{}).Return(vm, nil)

			statusErr := handler.RestartVM(context.Background(), k8smetav1.NamespaceDefault, testVMName, &v1.RestartOptions{})
			expectStatusError(statusErr, http.StatusConflict)
			Expect(statusErr.Error()).To(ContainSubstring("Halted does not support manual restart requests"))
		})

		DescribeTable("should not fail with VMI and RunStrategy", func(runStrategy v1.VirtualMachineRunStrategy) {
			vm := newVirtualMachineWithRunStrategy(runStrategy)
			vmi := newVirtualMachineInstanceInPhase(v1.Failed)

			vmClient.EXPECT().Get(context.Background(), vm.Name, k8smetav1.GetOptions{}).Return(vm, nil)
			vmiClient.EXPECT().Get(context.Background(), vm.Name, k8smetav1.GetOptions{}).Return(vmi, nil)
			vmClient.EXPECT().PatchStatus(context.Background(), vm.Name, types.JSONPatchType, gomock.Any(), k8smetav1.PatchOptions{}).Return(vm, nil)

			statusErr := handler.RestartVM(context.Background(), k8smetav1.NamespaceDefault, testVMName, &v1.RestartOptions{})
			Expect(statusErr).ToNot(HaveOccurred())
		},
			Entry("Always", v1.RunStrategyAlways),
			Entry("Manual", v1.RunStrategyManual),
			Entry("RerunOnFailure", v1.RunStrategyRerunOnFailure),
		)

		DescribeTable("should fail anytime without VMI and RunStrategy", func(runStrategy v1.VirtualMachineRunStrategy, msg string, restartOptions *v1.RestartOptions) {
			vm := newVirtualMachineWithRunStrategy(runStrategy)

			vmClient.EXPECT().Get(context.Background(), vm.Name, k8smetav1.GetOptions{}).Return(vm, nil)
			vmiClient.EXPECT().Get(context.Background(), vm.Name, k8smetav1.GetOptions{}).Return(nil, errors.NewNotFound(v1.Resource("virtualmachineinstance"), vm.Name)).AnyTimes()

			statusErr := handler.RestartVM(context.Background(), k8smetav1.NamespaceDefault, testVMName, restartOptions)
			expectStatusError(statusErr, http.StatusConflict)
			Expect(statusErr.Error()).To(ContainSubstring(msg))
		},
			Entry("Always", v1.RunStrategyAlways, "VM is not running", &v1.RestartOptions{}),
			Entry("Manual", v1.RunStrategyManual, "VM is not running", &v1.RestartOptions{}),
			Entry("RerunOnFailure", v1.RunStrategyRerunOnFailure, "VM is not running", &v1.RestartOptions{}),
			Entry("Once", v1.RunStrategyOnce, "Once does not support manual restart requests", &v1.RestartOptions{}),
			Entry("Halted", v1.RunStrategyHalted, "Halted does not support manual restart requests", &v1.RestartOptions{}),

			Entry("Always with dry-run option", v1.RunStrategyAlways, "VM is not running", &v1.RestartOptions{DryRun: withDryRun()}),
			Entry("Manual with dry-run option", v1.RunStrategyManual, "VM is not running", &v1.RestartOptions{DryRun: withDryRun()}),
			Entry("RerunOnFailure with dry-run option", v1.RunStrategyRerunOnFailure, "VM is not running", &v1.RestartOptions{DryRun: withDryRun()}),
			Entry("Once with dry-run option", v1.RunStrategyOnce, "Once does not support manual restart requests", &v1.RestartOptions{DryRun: withDryRun()}),
			Entry("Halted with dry-run option", v1.RunStrategyHalted, "Halted does not support manual restart requests", &v1.RestartOptions{DryRun: withDryRun()}),
		)
	})

	Context("error handling for StartVM", func() {
		DescribeTable("should fail on VM with RunStrategy",
			func(runStrategy v1.VirtualMachineRunStrategy, phase v1.VirtualMachineInstancePhase, status int, msg string, startOptions *v1.StartOptions) {
				vm := newVirtualMachineWithRunStrategy(runStrategy)
				var vmi *v1.VirtualMachineInstance
				if phase != v1.VmPhaseUnset {
					vmi = newVirtualMachineInstanceInPhase(phase)
				}

				vmClient.EXPECT().Get(context.Background(), vm.Name, k8smetav1.GetOptions{}).Return(vm, nil)
				vmiClient.EXPECT().Get(context.Background(), testVMName, k8smetav1.GetOptions{}).DoAndReturn(func(ctx context.Context, name string, opts k8smetav1.GetOptions) (interface{}, interface{}) {
					if status == http.StatusNotFound {
						return vmi, errors.NewNotFound(v1.Resource("virtualmachineinstance"), testVMName)
					}
					return vmi, nil
				})

				statusErr := handler.StartVM(context.Background(), k8smetav1.NamespaceDefault, testVMName, startOptions)
				expectStatusError(statusErr, http.StatusConflict)
				Expect(statusErr.Error()).To(ContainSubstring(msg))
			},
			Entry("Always without VMI", v1.RunStrategyAlways, v1.VmPhaseUnset, http.StatusNotFound, "Always does not support manual start requests", &v1.StartOptions{}),
			Entry("Always with VMI in phase Running", v1.RunStrategyAlways, v1.Running, http.StatusOK, "VM is already running", &v1.StartOptions{}),
			Entry("Once", v1.RunStrategyOnce, v1.VmPhaseUnset, http.StatusNotFound, "Once does not support manual start requests", &v1.StartOptions{}),
			Entry("RerunOnFailure with VMI in phase Failed", v1.RunStrategyRerunOnFailure, v1.Failed, http.StatusOK, "RerunOnFailure does not support starting VM from failed state", &v1.StartOptions{}),

			Entry("Always without VMI and with dry-run option", v1.RunStrategyAlways, v1.VmPhaseUnset, http.StatusNotFound, "Always does not support manual start requests", &v1.StartOptions{DryRun: withDryRun()}),
			Entry("Always with VMI in phase Running and with dry-run option", v1.RunStrategyAlways, v1.Running, http.StatusOK, "VM is already running", &v1.StartOptions{DryRun: withDryRun()}),
			Entry("Once with dry-run option", v1.RunStrategyOnce, v1.VmPhaseUnset, http.StatusNotFound, "Once does not support manual start requests", &v1.StartOptions{DryRun: withDryRun()}),
			Entry("RerunOnFailure with VMI in phase Failed and with dry-run option", v1.RunStrategyRerunOnFailure, v1.Failed, http.StatusOK, "RerunOnFailure does not support starting VM from failed state", &v1.StartOptions{DryRun: withDryRun()}),
		)

		DescribeTable("should not fail on VM with RunStrategy ",
			func(runStrategy v1.VirtualMachineRunStrategy, phase v1.VirtualMachineInstancePhase, status int) {
				vm := newVirtualMachineWithRunStrategy(runStrategy)
				var vmi *v1.VirtualMachineInstance
				if phase != v1.VmPhaseUnset {
					vmi = newVirtualMachineInstanceInPhase(phase)
				}

				vmClient.EXPECT().Get(context.Background(), vm.Name, k8smetav1.GetOptions{}).Return(vm, nil)
				vmiClient.EXPECT().Get(context.Background(), vm.Name, k8smetav1.GetOptions{}).Return(vmi, nil)
				vmClient.EXPECT().PatchStatus(context.Background(), vm.Name, types.JSONPatchType, gomock.Any(), k8smetav1.PatchOptions{}).DoAndReturn(
					func(ctx context.Context, name string, patchType types.PatchType, body interface{}, opts k8smetav1.PatchOptions) (interface{}, interface{}) {
						Expect(opts.DryRun).To(BeNil())
						return vm, nil
					})

				statusErr := handler.StartVM(context.Background(), k8smetav1.NamespaceDefault, testVMName, &v1.StartOptions{})
				Expect(statusErr).ToNot(HaveOccurred())
			},
			Entry("RerunOnFailure with VMI in state Succeeded", v1.RunStrategyRerunOnFailure, v1.Succeeded, http.StatusOK),
			Entry("Manual with VMI in state Succeeded", v1.RunStrategyManual, v1.Succeeded, http.StatusOK),
			Entry("Manual with VMI in state Failed", v1.RunStrategyManual, v1.Failed, http.StatusOK),
		)

		It("should fail when the volume migration in ongoing", func() {
			vmi := libvmi.New()
			vm := libvmi.NewVirtualMachine(vmi)
			controller.NewVirtualMachineConditionManager().UpdateCondition(vm, &v1.VirtualMachineCondition{
				Type:   v1.VirtualMachineManualRecoveryRequired,
				Status: k8sv1.ConditionTrue,
			})
			vmClient.EXPECT().Get(context.Background(), vm.Name, k8smetav1.GetOptions{}).Return(vm, nil)
			vmiClient.EXPECT().Get(context.Background(), vm.Name, k8smetav1.GetOptions{}).Return(vmi, nil)

			statusErr := handler.StartVM(context.Background(), k8smetav1.NamespaceDefault, vm.Name, &v1.StartOptions{})
			expectStatusError(statusErr, http.StatusConflict)
			Expect(statusErr.Error()).To(ContainSubstring("VM recovery required"))
		})
	})

	Context("error handling for StopVM", func() {
		DescribeTable("should handle VMI does not exist per run strategy", func(runStrategy v1.VirtualMachineRunStrategy, msg string, expectError bool, stopOptions *v1.StopOptions) {
			vm := newVirtualMachineWithRunStrategy(runStrategy)

			vmClient.EXPECT().Get(context.Background(), vm.Name, k8smetav1.GetOptions{}).Return(vm, nil)
			vmiClient.EXPECT().Get(context.Background(), vm.Name, k8smetav1.GetOptions{}).Return(nil, errors.NewNotFound(v1.Resource("virtualmachineinstance"), testVMName))
			if !expectError {
				vmClient.EXPECT().Patch(context.Background(), vm.Name, types.JSONPatchType, gomock.Any(), gomock.Any()).DoAndReturn(
					func(ctx context.Context, name string, patchType types.PatchType, body interface{}, opts k8smetav1.PatchOptions, _ ...string) (interface{}, interface{}) {
						Expect(opts.DryRun).To(BeEquivalentTo(stopOptions.DryRun))
						return vm, nil
					})
			}

			statusErr := handler.StopVM(context.Background(), k8smetav1.NamespaceDefault, testVMName, stopOptions)

			if expectError {
				expectStatusError(statusErr, http.StatusConflict)
				Expect(statusErr.Error()).To(ContainSubstring(msg))
			} else {
				Expect(statusErr).ToNot(HaveOccurred())
			}
		},
			Entry("RunStrategyAlways", v1.RunStrategyAlways, "", false, &v1.StopOptions{}),
			Entry("RunStrategyOnce", v1.RunStrategyOnce, "", false, &v1.StopOptions{}),
			Entry("RunStrategyRerunOnFailure", v1.RunStrategyRerunOnFailure, "", true, &v1.StopOptions{}),
			Entry("RunStrategyManual", v1.RunStrategyManual, "VM is not running", true, &v1.StopOptions{}),
			Entry("RunStrategyHalted", v1.RunStrategyHalted, "VM is not running", true, &v1.StopOptions{}),

			Entry("RunStrategyAlways with dry-run option", v1.RunStrategyAlways, "", false, &v1.StopOptions{DryRun: withDryRun()}),
			Entry("RunStrategyOnce with dry-run option", v1.RunStrategyOnce, "", false, &v1.StopOptions{DryRun: withDryRun()}),
			Entry("RunStrategyRerunOnFailure with dry-run option", v1.RunStrategyRerunOnFailure, "", true, &v1.StopOptions{DryRun: withDryRun()}),
			Entry("RunStrategyManual with dry-run option", v1.RunStrategyManual, "VM is not running", true, &v1.StopOptions{DryRun: withDryRun()}),
			Entry("RunStrategyHalted with dry-run option", v1.RunStrategyHalted, "VM is not running", true, &v1.StopOptions{DryRun: withDryRun()}),
		)

		It("should fail on VM with VMI in Unknown Phase", func() {
			vm := newVirtualMachineWithRunStrategy(v1.RunStrategyHalted)
			vmi := newVirtualMachineInstanceInPhase(v1.Unknown)

			vmClient.EXPECT().Get(context.Background(), vm.Name, k8smetav1.GetOptions{}).Return(vm, nil)
			vmiClient.EXPECT().Get(context.Background(), vm.Name, k8smetav1.GetOptions{}).Return(vmi, nil)

			statusErr := handler.StopVM(context.Background(), k8smetav1.NamespaceDefault, testVMName, &v1.StopOptions{})
			expectStatusError(statusErr, http.StatusConflict)
			Expect(statusErr.Error()).To(ContainSubstring("Halted only supports manual stop requests with a shorter graceperiod"))
		})

		DescribeTable("for VM with RunStrategyHalted, should", func(terminationGracePeriod *int64, graceperiod *int64, shouldFail bool) {
			vm := newVirtualMachineWithRunStrategy(v1.RunStrategyHalted)
			vmi := newVirtualMachineInstanceInPhase(v1.Running)

			vmi.Spec.TerminationGracePeriodSeconds = terminationGracePeriod

			stopOptions := &v1.StopOptions{GracePeriod: graceperiod}

			vmClient.EXPECT().Get(context.Background(), vm.Name, k8smetav1.GetOptions{}).Return(vm, nil)
			vmiClient.EXPECT().Get(context.Background(), vm.Name, k8smetav1.GetOptions{}).Return(vmi, nil)

			if graceperiod != nil {
				vmiClient.EXPECT().Patch(context.Background(), vmi.Name, types.JSONPatchType, gomock.Any(), gomock.Any()).DoAndReturn(
					func(ctx context.Context, name string, patchType types.PatchType, data []byte, opts k8smetav1.PatchOptions, _ ...string) (interface{}, interface{}) {
						Expect(opts.DryRun).To(BeEquivalentTo(stopOptions.DryRun))

						patchSet := patch.New()
						if vmi.Spec.TerminationGracePeriodSeconds != nil {
							patchSet.AddOption(patch.WithTest("/spec/terminationGracePeriodSeconds", *vmi.Spec.TerminationGracePeriodSeconds))
						} else {
							patchSet.AddOption(patch.WithTest("/spec/terminationGracePeriodSeconds", nil))
						}
						patchSet.AddOption(patch.WithReplace("/spec/terminationGracePeriodSeconds", *graceperiod))
						patchBytes, err := patchSet.GeneratePayload()
						Expect(err).ToNot(HaveOccurred())
						Expect(string(data)).To(Equal(string(patchBytes)))
						return vm, nil
					})
			}
			if !shouldFail {
				vmClient.EXPECT().PatchStatus(context.Background(), vm.Name, types.JSONPatchType, gomock.Any(), k8smetav1.PatchOptions{}).Return(vm, nil)
			}
			statusErr := handler.StopVM(context.Background(), k8smetav1.NamespaceDefault, testVMName, stopOptions)

			if shouldFail {
				expectStatusError(statusErr, http.StatusConflict)
				Expect(statusErr.Error()).To(ContainSubstring("Halted only supports manual stop requests with a shorter graceperiod"))
			} else {
				Expect(statusErr).ToNot(HaveOccurred())
			}
		},
			Entry("fail with nil graceperiod", pointer.P(int64(1800)), nil, true),
			Entry("fail with equal graceperiod", pointer.P(int64(1800)), pointer.P(int64(1800)), true),
			Entry("fail with greater graceperiod", pointer.P(int64(1800)), pointer.P(int64(2400)), true),
			Entry("not fail with non-nil graceperiod and nil termination graceperiod", nil, pointer.P(int64(1800)), false),
			Entry("not fail with shorter graceperiod and non-nil termination graceperiod", pointer.P(int64(1800)), pointer.P(int64(800)), false),
		)

		DescribeTable("should not fail on VM with RunStrategy", func(runStrategy v1.VirtualMachineRunStrategy) {
			vm := newVirtualMachineWithRunStrategy(runStrategy)
			vmi := newVirtualMachineInstanceInPhase(v1.Running)

			vmClient.EXPECT().Get(context.Background(), vm.Name, k8smetav1.GetOptions{}).Return(vm, nil)
			vmiClient.EXPECT().Get(context.Background(), vm.Name, k8smetav1.GetOptions{}).Return(vmi, nil)

			if runStrategy == v1.RunStrategyManual || runStrategy == v1.RunStrategyRerunOnFailure {
				vmClient.EXPECT().PatchStatus(context.Background(), vm.Name, types.JSONPatchType, gomock.Any(), k8smetav1.PatchOptions{}).Return(vm, nil)
			} else {
				vmClient.EXPECT().Patch(context.Background(), vm.Name, types.JSONPatchType, gomock.Any(), k8smetav1.PatchOptions{}).Return(vm, nil)
			}

			statusErr := handler.StopVM(context.Background(), k8smetav1.NamespaceDefault, testVMName, &v1.StopOptions{})
			Expect(statusErr).ToNot(HaveOccurred())
		},
			Entry("Always", v1.RunStrategyAlways),
			Entry("RerunOnFailure", v1.RunStrategyRerunOnFailure),
			Entry("Once", v1.RunStrategyOnce),
			Entry("Manual", v1.RunStrategyManual),
		)
	})

	Context("MigrateVM", func() {
		DescribeTable("should fail if VirtualMachine not exists according to options", func(migrateOptions *v1.MigrateOptions) {
			vmClient.EXPECT().Get(context.Background(), testVMName, k8smetav1.GetOptions{}).Return(nil, errors.NewNotFound(v1.Resource("virtualmachine"), testVMName))
			statusErr := handler.MigrateVM(context.Background(), k8smetav1.NamespaceDefault, testVMName, migrateOptions)
			expectStatusError(statusErr, http.StatusNotFound)
		},
			Entry("with default", &v1.MigrateOptions{}),
			Entry("with dry-run option", &v1.MigrateOptions{DryRun: withDryRun()}),
		)

		DescribeTable("should fail if VirtualMachine is not running according to options", func(migrateOptions *v1.MigrateOptions) {
			vm := v1.VirtualMachine{}
			vmi := v1.VirtualMachineInstance{}

			vmClient.EXPECT().Get(context.Background(), testVMName, k8smetav1.GetOptions{}).Return(&vm, nil)
			vmiClient.EXPECT().Get(context.Background(), testVMName, k8smetav1.GetOptions{}).Return(&vmi, nil)

			statusErr := handler.MigrateVM(context.Background(), k8smetav1.NamespaceDefault, testVMName, migrateOptions)
			expectStatusError(statusErr, http.StatusConflict)
			Expect(statusErr.Error()).To(ContainSubstring("VM is not running"))
		},
			Entry("with default", &v1.MigrateOptions{}),
			Entry("with dry-run option", &v1.MigrateOptions{DryRun: withDryRun()}),
		)

		DescribeTable("should fail if migration is not posted according to options", func(migrateOptions *v1.MigrateOptions) {
			vm := v1.VirtualMachine{}

			vmi := v1.VirtualMachineInstance{
				Status: v1.VirtualMachineInstanceStatus{
					Phase: v1.Running,
				},
			}

			vmClient.EXPECT().Get(context.Background(), testVMName, k8smetav1.GetOptions{}).Return(&vm, nil)
			vmiClient.EXPECT().Get(context.Background(), testVMName, k8smetav1.GetOptions{}).Return(&vmi, nil)
			migrateClient.EXPECT().Create(context.Background(), gomock.Any(), gomock.Any()).Return(nil, errors.NewInternalError(fmt.Errorf("error creating object")))
			statusErr := handler.MigrateVM(context.Background(), k8smetav1.NamespaceDefault, testVMName, migrateOptions)
			expectStatusError(statusErr, http.StatusInternalServerError)
		},
			Entry("with default", &v1.MigrateOptions{}),
			Entry("with dry-run option", &v1.MigrateOptions{DryRun: withDryRun()}),
		)

		DescribeTable("should migrate VirtualMachine according to options", func(migrateOptions *v1.MigrateOptions) {
			vm := v1.VirtualMachine{}

			vmi := v1.VirtualMachineInstance{
				Status: v1.VirtualMachineInstanceStatus{
					Phase: v1.Running,
				},
			}

			migration := v1.VirtualMachineInstanceMigration{}

			vmClient.EXPECT().Get(context.Background(), testVMName, k8smetav1.GetOptions{}).Return(&vm, nil)
			vmiClient.EXPECT().Get(context.Background(), testVMName, k8smetav1.GetOptions{}).Return(&vmi, nil)

			migrateClient.EXPECT().Create(context.Background(), gomock.Any(), gomock.Any()).Do(
				func(ctx context.Context, obj interface{}, opts k8smetav1.CreateOptions) {
					Expect(opts.DryRun).To(BeEquivalentTo(migrateOptions.DryRun))
				}).Return(&migration, nil)
			statusErr := handler.MigrateVM(context.Background(), k8smetav1.NamespaceDefault, testVMName, migrateOptions)
			Expect(statusErr).ToNot(HaveOccurred())
		},
			Entry("with default", &v1.MigrateOptions{}),
			Entry("with dry-run option", &v1.MigrateOptions{DryRun: withDryRun()}),
		)
	})

	Context("StateChange JSON", func() {
		It("should create a stop request if status exists", func() {
			uid := uuid.NewUUID()
			vm := newMinimalVM(testVMName)
			vm.Status.Created = true
			stopRequest := v1.VirtualMachineStateChangeRequest{
				Action: v1.StopRequest,
				UID:    &uid,
			}

			res, err := getChangeRequestJSON(vm, stopRequest)
			Expect(err).ToNot(HaveOccurred())

			ref, err := patch.New(
				patch.WithTest("/status/stateChangeRequests", nil),
				patch.WithAdd("/status/stateChangeRequests", []v1.VirtualMachineStateChangeRequest{stopRequest}),
			).GeneratePayload()
			Expect(err).ToNot(HaveOccurred())
			Expect(res).To(Equal(ref))
		})

		It("should create a stop request if status doesn't exist", func() {
			uid := uuid.NewUUID()
			vm := newMinimalVM(testVMName)
			stopRequest := v1.VirtualMachineStateChangeRequest{
				Action: v1.StopRequest,
				UID:    &uid,
			}

			res, err := getChangeRequestJSON(vm, stopRequest)
			Expect(err).ToNot(HaveOccurred())

			ref, err := patch.New(
				patch.WithAdd("/status", v1.VirtualMachineStatus{StateChangeRequests: []v1.VirtualMachineStateChangeRequest{stopRequest}}),
			).GeneratePayload()
			Expect(err).ToNot(HaveOccurred())
			Expect(res).To(Equal(ref))
		})

		It("should create a restart request if status exists", func() {
			uid := uuid.NewUUID()
			vm := newMinimalVM(testVMName)
			vm.Status.Created = true
			stopRequest := v1.VirtualMachineStateChangeRequest{
				Action: v1.StopRequest,
				UID:    &uid,
			}
			startRequest := v1.VirtualMachineStateChangeRequest{
				Action: v1.StartRequest,
			}

			res, err := getChangeRequestJSON(vm, stopRequest, startRequest)
			Expect(err).ToNot(HaveOccurred())

			ref, err := patch.New(
				patch.WithTest("/status/stateChangeRequests", nil),
				patch.WithAdd("/status/stateChangeRequests", []v1.VirtualMachineStateChangeRequest{stopRequest, startRequest}),
			).GeneratePayload()
			Expect(err).ToNot(HaveOccurred())
			Expect(res).To(Equal(ref))
		})

		It("should create a restart request if status doesn't exist", func() {
			uid := uuid.NewUUID()
			vm := newMinimalVM(testVMName)
			stopRequest := v1.VirtualMachineStateChangeRequest{
				Action: v1.StopRequest,
				UID:    &uid,
			}
			startRequest := v1.VirtualMachineStateChangeRequest{
				Action: v1.StartRequest,
			}

			res, err := getChangeRequestJSON(vm, stopRequest, startRequest)
			Expect(err).ToNot(HaveOccurred())

			ref, err := patch.New(
				patch.WithAdd("/status", v1.VirtualMachineStatus{StateChangeRequests: []v1.VirtualMachineStateChangeRequest{stopRequest, startRequest}}),
			).GeneratePayload()
			Expect(err).ToNot(HaveOccurred())
			Expect(res).To(Equal(ref))
		})

		It("should create a start request if status exists", func() {
			vm := newMinimalVM(testVMName)
			vm.Status.Created = true

			startRequest := v1.VirtualMachineStateChangeRequest{
				Action: v1.StartRequest,
			}

			res, err := getChangeRequestJSON(vm, startRequest)
			Expect(err).ToNot(HaveOccurred())

			ref, err := patch.New(
				patch.WithTest("/status/stateChangeRequests", nil),
				patch.WithAdd("/status/stateChangeRequests", []v1.VirtualMachineStateChangeRequest{startRequest}),
			).GeneratePayload()
			Expect(err).ToNot(HaveOccurred())
			Expect(res).To(Equal(ref))
		})

		It("should create a start request if status doesn't exist", func() {
			vm := newMinimalVM(testVMName)

			startRequest := v1.VirtualMachineStateChangeRequest{
				Action: v1.StartRequest,
			}

			res, err := getChangeRequestJSON(vm, startRequest)
			Expect(err).ToNot(HaveOccurred())

			ref, err := patch.New(
				patch.WithAdd("/status", v1.VirtualMachineStatus{StateChangeRequests: []v1.VirtualMachineStateChangeRequest{startRequest}}),
			).GeneratePayload()
			Expect(err).ToNot(HaveOccurred())
			Expect(res).To(Equal(ref))
		})

		It("should force a stop request to override", func() {
			uid := uuid.NewUUID()
			vm := newMinimalVM(testVMName)
			stopRequest := v1.VirtualMachineStateChangeRequest{
				Action: v1.StopRequest,
				UID:    &uid,
			}
			startRequest := v1.VirtualMachineStateChangeRequest{
				Action: v1.StartRequest,
			}
			vm.Status.StateChangeRequests = append(vm.Status.StateChangeRequests, startRequest)

			res, err := getChangeRequestJSON(vm, stopRequest)
			Expect(err).ToNot(HaveOccurred())

			ref, err := patch.New(
				patch.WithTest("/status/stateChangeRequests", []v1.VirtualMachineStateChangeRequest{startRequest}),
				patch.WithReplace("/status/stateChangeRequests", []v1.VirtualMachineStateChangeRequest{stopRequest}),
			).GeneratePayload()
			Expect(err).ToNot(HaveOccurred())
			Expect(res).To(Equal(ref))
		})

		It("should error on start request if other requests exist", func() {
			uid := uuid.NewUUID()
			vm := newMinimalVM(testVMName)
			stopRequest := v1.VirtualMachineStateChangeRequest{
				Action: v1.StopRequest,
				UID:    &uid,
			}
			startRequest := v1.VirtualMachineStateChangeRequest{
				Action: v1.StartRequest,
			}
			vm.Status.StateChangeRequests = append(vm.Status.StateChangeRequests, stopRequest)

			_, err := getChangeRequestJSON(vm, startRequest)
			Expect(err).To(HaveOccurred())
		})

		It("should error on restart request if other requests exist", func() {
			uid := uuid.NewUUID()
			vm := newMinimalVM(testVMName)
			stopRequest := v1.VirtualMachineStateChangeRequest{
				Action: v1.StopRequest,
				UID:    &uid,
			}
			startRequest := v1.VirtualMachineStateChangeRequest{
				Action: v1.StartRequest,
			}
			vm.Status.StateChangeRequests = append(vm.Status.StateChangeRequests, startRequest)

			_, err := getChangeRequestJSON(vm, stopRequest, startRequest)
			Expect(err).To(HaveOccurred())
		})
	})

	Context("start paused", func() {
		DescribeTable("should patch status on start according to options", func(startOptions *v1.StartOptions) {
			vm := v1.VirtualMachine{
				ObjectMeta: k8smetav1.ObjectMeta{
					Name:      testVMName,
					Namespace: k8smetav1.NamespaceDefault,
				},
				Spec: v1.VirtualMachineSpec{
					Running:  pointer.P(NotRunning),
					Template: &v1.VirtualMachineInstanceTemplateSpec{},
				},
			}
			vmi := v1.VirtualMachineInstance{
				Spec: v1.VirtualMachineInstanceSpec{},
			}
			vmi.ObjectMeta.SetUID(uuid.NewUUID())

			vmClient.EXPECT().Get(context.Background(), vm.Name, k8smetav1.GetOptions{}).Return(&vm, nil)
			vmiClient.EXPECT().Get(context.Background(), vm.Name, k8smetav1.GetOptions{}).Return(&vmi, nil)
			vmClient.EXPECT().PatchStatus(context.Background(), vm.Name, types.JSONPatchType, gomock.Any(), gomock.Any()).DoAndReturn(
				func(ctx context.Context, name string, patchType types.PatchType, body interface{}, opts k8smetav1.PatchOptions) (interface{}, interface{}) {
					Expect(opts.DryRun).To(BeEquivalentTo(startOptions.DryRun))
					return &vm, nil
				})

			statusErr := handler.StartVM(context.Background(), k8smetav1.NamespaceDefault, testVMName, startOptions)
			Expect(statusErr).ToNot(HaveOccurred())
		},
			Entry("with default", &v1.StartOptions{Paused: Paused}),
			Entry("with dry-run option", &v1.StartOptions{Paused: Paused, DryRun: withDryRun()}),
		)

		DescribeTable("should patch status on start for VM with RunStrategy",
			func(runStrategy v1.VirtualMachineRunStrategy) {
				vm := newVirtualMachineWithRunStrategy(runStrategy)
				vm.Spec.Template = &v1.VirtualMachineInstanceTemplateSpec{}
				startOptions := &v1.StartOptions{Paused: true}

				vmi := newVirtualMachineInstanceInPhase(v1.Succeeded)

				vmClient.EXPECT().Get(context.Background(), vm.Name, k8smetav1.GetOptions{}).Return(vm, nil)
				vmiClient.EXPECT().Get(context.Background(), vm.Name, k8smetav1.GetOptions{}).Return(vmi, nil)
				vmClient.EXPECT().PatchStatus(context.Background(), vm.Name, types.JSONPatchType, gomock.Any(), k8smetav1.PatchOptions{}).Return(vm, nil)

				statusErr := handler.StartVM(context.Background(), k8smetav1.NamespaceDefault, testVMName, startOptions)
				Expect(statusErr).ToNot(HaveOccurred())
			},
			Entry("Manual RunStrategy", v1.RunStrategyManual),
			Entry("RerunOnFailure RunStrategy", v1.RunStrategyRerunOnFailure),
		)
	})
})

func newVirtualMachineWithRunStrategy(runStrategy v1.VirtualMachineRunStrategy) *v1.VirtualMachine {
	return &v1.VirtualMachine{
		ObjectMeta: k8smetav1.ObjectMeta{
			Name:      testVMName,
			Namespace: k8smetav1.NamespaceDefault,
		},
		Spec: v1.VirtualMachineSpec{
			RunStrategy: &runStrategy,
		},
	}
}

func newVirtualMachineWithRunning(running *bool) *v1.VirtualMachine {
	return &v1.VirtualMachine{
		ObjectMeta: k8smetav1.ObjectMeta{
			Name:      testVMName,
			Namespace: k8smetav1.NamespaceDefault,
		},
		Spec: v1.VirtualMachineSpec{
			Running: running,
		},
	}
}

func newVirtualMachineInstanceInPhase(phase v1.VirtualMachineInstancePhase) *v1.VirtualMachineInstance {
	virtualMachineInstance := v1.VirtualMachineInstance{
		Spec:   v1.VirtualMachineInstanceSpec{},
		Status: v1.VirtualMachineInstanceStatus{Phase: phase},
	}
	virtualMachineInstance.ObjectMeta.SetUID(uuid.NewUUID())
	return &virtualMachineInstance
}

func newMinimalVM(name string) *v1.VirtualMachine {
	return &v1.VirtualMachine{TypeMeta: k8smetav1.TypeMeta{APIVersion: v1.GroupVersion.String(), Kind: "VirtualMachine"}, ObjectMeta: k8smetav1.ObjectMeta{Name: name}}
}
