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

package rest

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"

	"github.com/emicklei/go-restful/v3"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"go.uber.org/mock/gomock"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"

	v1 "kubevirt.io/api/core/v1"
	"kubevirt.io/client-go/kubecli"

	"kubevirt.io/kubevirt/pkg/apimachinery/patch"
	"kubevirt.io/kubevirt/pkg/hibernation"
	"kubevirt.io/kubevirt/pkg/testutils"
	"kubevirt.io/kubevirt/pkg/virt-config/featuregate"
)

var _ = Describe("Hibernation subresources", func() {
	const vmName = "testvm"

	var (
		request       *restful.Request
		recorder      *httptest.ResponseRecorder
		response      *restful.Response
		vmClient      *kubecli.MockVirtualMachineInterface
		vmiClient     *kubecli.MockVirtualMachineInstanceInterface
		app           *SubresourceAPIApp
		vm            *v1.VirtualMachine
		vmi           *v1.VirtualMachineInstance
		statusOps     []patch.PatchOperation
		statusPatches [][]patch.PatchOperation
		specPatch     []byte
		specPatchErr  error

		kv = &v1.KubeVirt{
			ObjectMeta: metav1.ObjectMeta{Name: "kubevirt", Namespace: "kubevirt"},
			Spec: v1.KubeVirtSpec{Configuration: v1.KubeVirtConfiguration{
				DeveloperConfiguration: &v1.DeveloperConfiguration{},
			}},
		}
	)

	config, _, kvStore := testutils.NewFakeClusterConfigUsingKV(kv)

	enableGate := func() {
		kvConfig := kv.DeepCopy()
		kvConfig.Spec.Configuration.DeveloperConfiguration.FeatureGates = []string{featuregate.VMHibernation, featuregate.UtilityVolumesGate}
		testutils.UpdateFakeKubeVirtClusterConfig(kvStore, kvConfig)
	}

	BeforeEach(func() {
		request = restful.NewRequest(&http.Request{})
		request.PathParameters()["name"] = vmName
		request.PathParameters()["namespace"] = metav1.NamespaceDefault
		recorder = httptest.NewRecorder()
		response = restful.NewResponse(recorder)

		ctrl := gomock.NewController(GinkgoT())
		virtClient := kubecli.NewMockKubevirtClient(ctrl)
		vmClient = kubecli.NewMockVirtualMachineInterface(ctrl)
		vmiClient = kubecli.NewMockVirtualMachineInstanceInterface(ctrl)
		virtClient.EXPECT().VirtualMachine(metav1.NamespaceDefault).Return(vmClient).AnyTimes()
		virtClient.EXPECT().VirtualMachineInstance(metav1.NamespaceDefault).Return(vmiClient).AnyTimes()
		app = NewSubresourceAPIApp(virtClient, nil, nil, 0, nil, config)

		vm = &v1.VirtualMachine{
			ObjectMeta: metav1.ObjectMeta{Name: vmName, Namespace: metav1.NamespaceDefault, UID: "vm-uid"},
			Spec: v1.VirtualMachineSpec{
				RunStrategy: new(v1.RunStrategyAlways),
				Template:    &v1.VirtualMachineInstanceTemplateSpec{},
			},
			Status: v1.VirtualMachineStatus{Created: true},
		}
		vmi = &v1.VirtualMachineInstance{
			ObjectMeta: metav1.ObjectMeta{
				Name: vmName, Namespace: metav1.NamespaceDefault, UID: "vmi-uid",
				OwnerReferences: []metav1.OwnerReference{{UID: vm.UID}},
			},
			Status: v1.VirtualMachineInstanceStatus{Phase: v1.Running},
		}
		statusOps = nil
		statusPatches = nil
		specPatch = nil
		specPatchErr = nil

		vmClient.EXPECT().Get(context.Background(), vmName, gomock.Any()).DoAndReturn(
			func(context.Context, string, metav1.GetOptions) (*v1.VirtualMachine, error) { return vm, nil }).AnyTimes()
		vmiClient.EXPECT().Get(context.Background(), vmName, gomock.Any()).DoAndReturn(
			func(context.Context, string, metav1.GetOptions) (*v1.VirtualMachineInstance, error) { return vmi, nil }).AnyTimes()
		vmClient.EXPECT().PatchStatus(context.Background(), vmName, types.JSONPatchType, gomock.Any(), gomock.Any()).DoAndReturn(
			func(_ context.Context, _ string, _ types.PatchType, body []byte, _ metav1.PatchOptions) (*v1.VirtualMachine, error) {
				statusOps = nil
				Expect(json.Unmarshal(body, &statusOps)).To(Succeed())
				statusPatches = append(statusPatches, statusOps)
				return vm, nil
			}).AnyTimes()
		vmClient.EXPECT().Patch(context.Background(), vmName, types.JSONPatchType, gomock.Any(), gomock.Any()).DoAndReturn(
			func(_ context.Context, _ string, _ types.PatchType, body []byte, _ metav1.PatchOptions, _ ...string) (*v1.VirtualMachine, error) {
				specPatch = body
				return vm, specPatchErr
			}).AnyTimes()
	})

	AfterEach(func() {
		testutils.UpdateFakeKubeVirtClusterConfig(kvStore, kv)
	})

	opPaths := func() []string {
		var paths []string
		for _, op := range statusOps {
			paths = append(paths, op.Op+" "+op.Path)
		}
		return paths
	}

	It("should be rejected without the feature gate", func() {
		app.HibernateVMRequestHandler(request, response)
		Expect(recorder.Code).To(Equal(http.StatusBadRequest))
	})

	It("should request the hibernation and halt a VM that always runs", func() {
		enableGate()
		app.HibernateVMRequestHandler(request, response)
		Expect(recorder.Code).To(Equal(http.StatusAccepted))
		Expect(opPaths()).To(Equal([]string{"test /status/savedState", "add /status/savedState"}))
		state, ok := statusOps[1].Value.(map[string]any)
		Expect(ok).To(BeTrue())
		Expect(state).To(HaveKeyWithValue("phase", string(v1.HibernationPending)))
		Expect(state).To(HaveKeyWithValue("sourceUID", "vmi-uid"))
		Expect(state).To(HaveKeyWithValue("runStrategy", string(v1.RunStrategyAlways)), "a cancel restores it")
		Expect(state).To(HaveKeyWithValue("claimName", HavePrefix(hibernation.ClaimName(vmName, "vmi-uid", "")+"-")))
		Expect(string(specPatch)).To(ContainSubstring(`"value":"Halted"`))
	})

	It("should request the hibernation together with a stop for a manually run VM", func() {
		enableGate()
		vm.Spec.RunStrategy = new(v1.RunStrategyManual)
		app.HibernateVMRequestHandler(request, response)
		Expect(recorder.Code).To(Equal(http.StatusAccepted))
		Expect(opPaths()).To(Equal([]string{
			"test /status/savedState", "add /status/savedState",
			"test /status/stateChangeRequests", "add /status/stateChangeRequests",
		}))
		Expect(specPatch).To(BeNil())
	})

	DescribeTable("should stop the VM the way its run strategy asks for", func(runStrategy v1.VirtualMachineRunStrategy, specHalted bool) {
		enableGate()
		vm.Spec.RunStrategy = new(runStrategy)
		app.HibernateVMRequestHandler(request, response)
		Expect(recorder.Code).To(Equal(http.StatusAccepted))
		state, ok := statusOps[1].Value.(map[string]any)
		Expect(ok).To(BeTrue())
		Expect(state).To(HaveKeyWithValue("runStrategy", string(runStrategy)))
		Expect(state).To(HaveKey("requestTimestamp"))
		if specHalted {
			Expect(string(specPatch)).To(ContainSubstring(`"value":"Halted"`))
			return
		}
		Expect(specPatch).To(BeNil())
		Expect(statusOps).To(HaveLen(4))
		requests, err := json.Marshal(statusOps[3].Value)
		Expect(err).ToNot(HaveOccurred())
		Expect(string(requests)).To(ContainSubstring(`"kubevirt.io/hibernation":"`+state["claimName"].(string)+`"`), "the stop request of the hibernation is marked")
	},
		Entry("Always", v1.RunStrategyAlways, true),
		Entry("Once", v1.RunStrategyOnce, true),
		Entry("Manual", v1.RunStrategyManual, false),
		Entry("RerunOnFailure", v1.RunStrategyRerunOnFailure, false),
	)

	It("should withdraw a request whose stop failed, without testing what the controller added", func() {
		enableGate()
		specPatchErr = fmt.Errorf("admission webhook timed out")
		app.HibernateVMRequestHandler(request, response)
		Expect(recorder.Code).To(Equal(http.StatusInternalServerError))
		Expect(statusPatches).To(HaveLen(2))
		var paths []string
		for _, op := range statusPatches[1] {
			paths = append(paths, op.Op+" "+op.Path)
		}
		Expect(paths).To(Equal([]string{
			"test /status/savedState/phase", "test /status/savedState/claimName", "test /status/savedState/sourceUID",
			"add /status/savedState/discard",
		}))
	})

	It("should reject a VM that is not running", func() {
		enableGate()
		vmi.Status.Phase = v1.Scheduled
		app.HibernateVMRequestHandler(request, response)
		Expect(recorder.Code).To(Equal(http.StatusConflict))
		Expect(statusOps).To(BeEmpty())
	})

	It("should reject a halted VM", func() {
		enableGate()
		vm.Spec.RunStrategy = new(v1.RunStrategyHalted)
		app.HibernateVMRequestHandler(request, response)
		Expect(recorder.Code).To(Equal(http.StatusConflict))
	})

	It("should discard a saved state", func() {
		vm.Status.SavedState = &v1.VirtualMachineSavedState{Phase: v1.HibernationSaved, ClaimName: "state"}
		app.DiscardSavedStateVMRequestHandler(request, response)
		Expect(recorder.Code).To(Equal(http.StatusAccepted))
		Expect(opPaths()).To(Equal([]string{
			"test /status/savedState/phase", "test /status/savedState/claimName", "add /status/savedState/discard",
		}))
	})

	It("should cancel a pending hibernation", func() {
		vm.Status.SavedState = &v1.VirtualMachineSavedState{Phase: v1.HibernationPending, ClaimName: "state"}
		app.DiscardSavedStateVMRequestHandler(request, response)
		Expect(recorder.Code).To(Equal(http.StatusAccepted))
		Expect(opPaths()).To(Equal([]string{
			"test /status/savedState/phase", "test /status/savedState/claimName", "add /status/savedState/discard",
		}))
	})

	It("should not cancel a hibernation whose state is being saved", func() {
		vm.Status.SavedState = &v1.VirtualMachineSavedState{Phase: v1.HibernationSaving, ClaimName: "state"}
		app.DiscardSavedStateVMRequestHandler(request, response)
		Expect(recorder.Code).To(Equal(http.StatusConflict))
		Expect(recorder.Body.String()).To(ContainSubstring("cannot be cancelled anymore"))
	})

	DescribeTable("should discard a saved state", func(phase v1.VirtualMachineSavedStatePhase) {
		vm.Status.SavedState = &v1.VirtualMachineSavedState{Phase: phase, ClaimName: "state"}
		app.DiscardSavedStateVMRequestHandler(request, response)
		Expect(recorder.Code).To(Equal(http.StatusAccepted))
		Expect(opPaths()).To(ContainElement("add /status/savedState/discard"))
	},
		Entry("whose restore failed, to boot the VM", v1.HibernationRestoreFailed),
		Entry("while a VMI waits to resume from it", v1.HibernationRestoring),
	)

	Context("starting a hibernating VM", func() {
		BeforeEach(func() {
			enableGate()
			vm.Spec.RunStrategy = new(v1.RunStrategyHalted)
		})

		It("should cancel a pending hibernation", func() {
			vm.Status.SavedState = &v1.VirtualMachineSavedState{Phase: v1.HibernationPending, ClaimName: "state", SourceUID: vmi.UID}
			app.StartVMRequestHandler(request, response)
			Expect(recorder.Code).To(Equal(http.StatusAccepted))
			Expect(opPaths()).To(Equal([]string{
				"test /status/savedState/phase", "test /status/savedState/claimName", "add /status/savedState/discard",
			}))
			Expect(specPatch).To(BeNil(), "the controller restores the run strategy")
		})

		It("should refuse while the state is being saved", func() {
			vm.Status.SavedState = &v1.VirtualMachineSavedState{Phase: v1.HibernationSaving, ClaimName: "state", SourceUID: vmi.UID}
			app.StartVMRequestHandler(request, response)
			Expect(recorder.Code).To(Equal(http.StatusConflict))
			Expect(recorder.Body.String()).To(ContainSubstring("start it once it is hibernated"))
			Expect(statusOps).To(BeEmpty())
		})

		It("should retry to resume a halted VM whose resume failed", func() {
			vm.Status.SavedState = &v1.VirtualMachineSavedState{Phase: v1.HibernationRestoreFailed, ClaimName: "state", Message: "corrupt"}
			vmi.Status.Phase = v1.Failed
			app.StartVMRequestHandler(request, response)
			Expect(recorder.Code).To(Equal(http.StatusAccepted))
			Expect(opPaths()).To(Equal([]string{
				"test /status/savedState/phase", "test /status/savedState/claimName",
				"replace /status/savedState/phase", "add /status/savedState/message",
			}))
			Expect(statusOps[2].Value).To(Equal(string(v1.HibernationSaved)))
			Expect(string(specPatch)).To(ContainSubstring(`"value":"Always"`))
		})

		It("should retry to resume a VM that always runs without touching its spec", func() {
			vm.Spec.RunStrategy = new(v1.RunStrategyAlways)
			vm.Status.SavedState = &v1.VirtualMachineSavedState{Phase: v1.HibernationRestoreFailed, ClaimName: "state", Message: "corrupt"}
			vmi.Status.Phase = v1.Failed
			app.StartVMRequestHandler(request, response)
			Expect(recorder.Code).To(Equal(http.StatusAccepted))
			Expect(opPaths()).To(ContainElement("replace /status/savedState/phase"))
			Expect(specPatch).To(BeNil())
		})

		DescribeTable("should retry to resume a VM whose run strategy starts it on its own", func(runStrategy v1.VirtualMachineRunStrategy) {
			vm.Spec.RunStrategy = new(runStrategy)
			vm.Status.SavedState = &v1.VirtualMachineSavedState{
				Phase: v1.HibernationRestoreFailed, ClaimName: "state", Message: "corrupt", RunStrategy: new(runStrategy), RestoreUID: vmi.UID,
			}
			vmi.Status.Phase = v1.Failed
			vmi.Spec.SavedStateSource = &v1.SavedStateSource{ClaimName: "state"}
			vmi.Status.SavedState = &v1.VirtualMachineInstanceSavedStateStatus{Phase: v1.SavedStateRestoreFailed}
			app.StartVMRequestHandler(request, response)
			Expect(recorder.Code).To(Equal(http.StatusAccepted), recorder.Body.String())
			Expect(opPaths()).To(Equal([]string{
				"test /status/savedState/phase", "test /status/savedState/claimName",
				"replace /status/savedState/phase", "add /status/savedState/message",
				"test /status/savedState/restoreUID", "remove /status/savedState/restoreUID",
				"add /status/savedState/supersededUID",
			}))
			Expect(statusOps[2].Value).To(Equal(string(v1.HibernationSaved)))
			Expect(statusOps[4].Value).To(Equal(string(vmi.UID)))
			Expect(statusOps[6].Value).To(Equal(string(vmi.UID)), "the VM controller replaces only the VMI the retry superseded")
			Expect(specPatch).To(BeNil(), "the VM controller replaces the failed VMI and resumes")
		},
			Entry("Always", v1.RunStrategyAlways),
			Entry("RerunOnFailure", v1.RunStrategyRerunOnFailure),
			Entry("Once", v1.RunStrategyOnce),
		)

		It("should supersede no VMI that did not fail to resume from the saved state", func() {
			vm.Spec.RunStrategy = new(v1.RunStrategyOnce)
			vm.Status.SavedState = &v1.VirtualMachineSavedState{Phase: v1.HibernationRestoreFailed, ClaimName: "state", Message: "corrupt", RestoreUID: "earlier-uid"}
			vmi.Status.Phase = v1.Failed
			vmi.Spec.SavedStateSource = &v1.SavedStateSource{ClaimName: "earlier-state"}
			vmi.Status.SavedState = &v1.VirtualMachineInstanceSavedStateStatus{Phase: v1.SavedStateRestoreFailed}
			app.StartVMRequestHandler(request, response)
			Expect(recorder.Code).To(Equal(http.StatusAccepted), recorder.Body.String())
			Expect(opPaths()).To(ContainElement("remove /status/savedState/restoreUID"))
			Expect(opPaths()).ToNot(ContainElement("add /status/savedState/supersededUID"))
		})

		It("should retry to resume a manually run VM with a start request", func() {
			vm.Spec.RunStrategy = new(v1.RunStrategyManual)
			vm.Status.SavedState = &v1.VirtualMachineSavedState{Phase: v1.HibernationRestoreFailed, ClaimName: "state", Message: "corrupt"}
			vmi.Status.Phase = v1.Failed
			app.StartVMRequestHandler(request, response)
			Expect(recorder.Code).To(Equal(http.StatusAccepted), recorder.Body.String())
			Expect(statusPatches).To(HaveLen(2))
			Expect(statusPatches[0][2].Value).To(Equal(string(v1.HibernationSaved)), "the retry first")
			Expect(fmt.Sprint(statusPatches[1])).To(ContainSubstring("action:Start"), "then the start request of a Manual VM")
		})

		It("should resume a hibernated VM with its run strategy Once", func() {
			vm.Status.SavedState = &v1.VirtualMachineSavedState{Phase: v1.HibernationSaved, ClaimName: "state", SourceUID: vmi.UID, RunStrategy: new(v1.RunStrategyOnce)}
			vmi.Status.Phase = v1.Succeeded
			app.StartVMRequestHandler(request, response)
			Expect(recorder.Code).To(Equal(http.StatusAccepted))
			Expect(string(specPatch)).To(ContainSubstring(`"value":"Once"`), "a resumed Once VM must not become Always")
		})

		It("should resume a hibernated VM as usual", func() {
			vm.Status.SavedState = &v1.VirtualMachineSavedState{Phase: v1.HibernationSaved, ClaimName: "state", SourceUID: vmi.UID}
			vmi.Status.Phase = v1.Succeeded
			app.StartVMRequestHandler(request, response)
			Expect(recorder.Code).To(Equal(http.StatusAccepted))
			Expect(string(specPatch)).To(ContainSubstring(`"value":"Always"`))
		})
	})

	DescribeTable("should refuse to stop a hibernating VM", func(phase v1.VirtualMachineSavedStatePhase) {
		enableGate()
		vm.Spec.RunStrategy = new(v1.RunStrategyManual)
		vm.Status.SavedState = &v1.VirtualMachineSavedState{Phase: phase, ClaimName: "state", SourceUID: vmi.UID}
		app.StopVMRequestHandler(request, response)
		Expect(recorder.Code).To(Equal(http.StatusConflict))
		Expect(recorder.Body.String()).To(ContainSubstring("is hibernating"))
	},
		Entry("Pending", v1.HibernationPending),
		Entry("Saving", v1.HibernationSaving),
	)

	It("should not discard a missing saved state", func() {
		app.DiscardSavedStateVMRequestHandler(request, response)
		Expect(recorder.Code).To(Equal(http.StatusConflict))
	})
})
