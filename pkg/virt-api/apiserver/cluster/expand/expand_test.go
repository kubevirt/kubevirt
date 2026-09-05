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

package expand

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"go.uber.org/mock/gomock"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apiserver/pkg/endpoints/request"

	v1 "kubevirt.io/api/core/v1"
	instancetypeapi "kubevirt.io/api/instancetype"
	instancetypev1beta1 "kubevirt.io/api/instancetype/v1beta1"
	"kubevirt.io/client-go/kubecli"
	"kubevirt.io/client-go/kubevirt/fake"

	"kubevirt.io/kubevirt/pkg/instancetype/conflict"
	"kubevirt.io/kubevirt/pkg/pointer"
	"kubevirt.io/kubevirt/pkg/testutils"
)

var _ = Describe("expand-vm-spec subresource", func() {
	const (
		vmName      = "test-vm"
		vmNamespace = "test-namespace"
		volumeName  = "volumeName"
	)

	var (
		virtClient *kubecli.MockKubevirtClient
		handler    *Handler
		vm         *v1.VirtualMachine
	)

	// callExpandVMSpec serializes the VM into the request body, injects the
	// namespace via RequestInfo and invokes the handler, returning the recorded response
	callExpandVMSpec := func(namespace string, body []byte) *httptest.ResponseRecorder {
		recorder := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodPut, "/", bytes.NewBuffer(body))
		ctx := request.WithRequestInfo(req.Context(), &request.RequestInfo{
			IsResourceRequest: true,
			APIGroup:          v1.SubresourceGroupName,
			APIVersion:        v1.ApiLatestVersion,
			Namespace:         namespace,
			Resource:          "expand-vm-spec",
			Verb:              "update",
		})
		handler.ServeHTTP(recorder, req.WithContext(ctx))
		return recorder
	}

	callExpandVMSpecAPI := func(vm *v1.VirtualMachine) *httptest.ResponseRecorder {
		vmJSON, err := json.Marshal(vm)
		Expect(err).ToNot(HaveOccurred())
		return callExpandVMSpec(vmNamespace, vmJSON)
	}

	expectStatusError := func(recorder *httptest.ResponseRecorder, code int) *metav1.Status {
		Expect(recorder.Code).To(Equal(code))
		status := &metav1.Status{}
		Expect(json.NewDecoder(recorder.Body).Decode(status)).To(Succeed())
		return status
	}

	expectExpandedVM := func(recorder *httptest.ResponseRecorder) *v1.VirtualMachine {
		Expect(recorder.Code).To(Equal(http.StatusOK))
		responseVM := &v1.VirtualMachine{}
		Expect(json.NewDecoder(recorder.Body).Decode(responseVM)).To(Succeed())
		// Keep compatibility with existing clients, the body is decoded as a
		// kubevirt.io/v1 VirtualMachine even though it is served from the
		// subresources.kubevirt.io group
		Expect(responseVM.APIVersion).To(Equal("kubevirt.io/" + v1.ApiLatestVersion))
		Expect(responseVM.Kind).To(Equal("VirtualMachine"))
		return responseVM
	}

	BeforeEach(func() {
		ctrl := gomock.NewController(GinkgoT())
		virtClient = kubecli.NewMockKubevirtClient(ctrl)
		virtClient.EXPECT().GeneratedKubeVirtClient().Return(fake.NewSimpleClientset()).AnyTimes()

		fakeInstancetypeClients := fake.NewSimpleClientset().InstancetypeV1beta1()
		virtClient.EXPECT().VirtualMachineClusterInstancetype().Return(fakeInstancetypeClients.VirtualMachineClusterInstancetypes()).AnyTimes()
		virtClient.EXPECT().VirtualMachineClusterPreference().Return(fakeInstancetypeClients.VirtualMachineClusterPreferences()).AnyTimes()
		virtClient.EXPECT().VirtualMachineInstancetype(vmNamespace).Return(fakeInstancetypeClients.VirtualMachineInstancetypes(vmNamespace)).AnyTimes()
		virtClient.EXPECT().VirtualMachinePreference(vmNamespace).Return(fakeInstancetypeClients.VirtualMachinePreferences(vmNamespace)).AnyTimes()

		kv := &v1.KubeVirt{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "kubevirt",
				Namespace: "kubevirt",
			},
			Spec: v1.KubeVirtSpec{
				Configuration: v1.KubeVirtConfiguration{
					DeveloperConfiguration: &v1.DeveloperConfiguration{},
				},
			},
			Status: v1.KubeVirtStatus{
				Phase: v1.KubeVirtPhaseDeployed,
			},
		}
		config, _, _ := testutils.NewFakeClusterConfigUsingKV(kv)

		handler = NewHandler(config, virtClient)

		vm = &v1.VirtualMachine{
			ObjectMeta: metav1.ObjectMeta{
				Name:      vmName,
				Namespace: vmNamespace,
			},
			Spec: v1.VirtualMachineSpec{
				Template: &v1.VirtualMachineInstanceTemplateSpec{
					Spec: v1.VirtualMachineInstanceSpec{
						Domain: v1.DomainSpec{},
						Volumes: []v1.Volume{{
							Name: volumeName,
						}},
					},
				},
			},
		}
	})

	It("should return unchanged VM, if no instancetype and preference is assigned", func() {
		vm.Spec.Instancetype = nil

		responseVM := expectExpandedVM(callExpandVMSpecAPI(vm))
		Expect(responseVM.Spec).To(Equal(vm.Spec))
	})

	DescribeTable("should fail if VM points to nonexistent instancetype", func(kind string) {
		vm.Spec.Instancetype = &v1.InstancetypeMatcher{
			Name: "nonexistent-instancetype",
			Kind: kind,
		}

		status := expectStatusError(callExpandVMSpecAPI(vm), http.StatusBadRequest)
		Expect(status.Message).To(ContainSubstring("not found"))
	},
		Entry("default (empty kind)", ""),
		Entry("singular kind", instancetypeapi.ClusterSingularResourceName),
		Entry("plural kind", instancetypeapi.ClusterPluralResourceName),
	)

	DescribeTable("should fail if VM points to nonexistent preference", func(kind string) {
		vm.Spec.Preference = &v1.PreferenceMatcher{
			Name: "nonexistent-preference",
			Kind: kind,
		}

		status := expectStatusError(callExpandVMSpecAPI(vm), http.StatusBadRequest)
		Expect(status.Message).To(ContainSubstring("not found"))
	},
		Entry("default (empty kind)", ""),
		Entry("singular kind", instancetypeapi.ClusterSingularPreferenceResourceName),
		Entry("plural kind", instancetypeapi.ClusterPluralPreferenceResourceName),
	)

	DescribeTable("should expand instancetype and preference within VM", func(instancetypeKind, preferenceKind string) {
		clusterInstancetype := &instancetypev1beta1.VirtualMachineClusterInstancetype{
			TypeMeta: metav1.TypeMeta{
				Kind:       "VirtualMachineClusterInstancetype",
				APIVersion: instancetypev1beta1.SchemeGroupVersion.String(),
			},
			ObjectMeta: metav1.ObjectMeta{
				Name: "test-cluster-instancetype",
			},
			Spec: instancetypev1beta1.VirtualMachineInstancetypeSpec{
				CPU: instancetypev1beta1.CPUInstancetype{
					Guest: uint32(2),
				},
				Memory: instancetypev1beta1.MemoryInstancetype{
					Guest: resource.MustParse("128Mi"),
				},
			},
		}
		_, err := virtClient.VirtualMachineClusterInstancetype().Create(context.Background(), clusterInstancetype, metav1.CreateOptions{})
		Expect(err).ToNot(HaveOccurred())

		vm.Spec.Instancetype = &v1.InstancetypeMatcher{
			Name: clusterInstancetype.Name,
			Kind: instancetypeKind,
		}

		clusterPreference := &instancetypev1beta1.VirtualMachineClusterPreference{
			TypeMeta: metav1.TypeMeta{
				Kind:       "VirtualMachineClusterPreference",
				APIVersion: instancetypev1beta1.SchemeGroupVersion.String(),
			},
			ObjectMeta: metav1.ObjectMeta{
				Name: "test-cluster-preference",
			},
			Spec: instancetypev1beta1.VirtualMachinePreferenceSpec{
				CPU: &instancetypev1beta1.CPUPreferences{
					PreferredCPUTopology: pointer.P(instancetypev1beta1.Cores),
				},
				Devices: &instancetypev1beta1.DevicePreferences{
					PreferredDiskBus: v1.DiskBusVirtio,
				},
			},
		}
		_, err = virtClient.VirtualMachineClusterPreference().Create(context.Background(), clusterPreference, metav1.CreateOptions{})
		Expect(err).ToNot(HaveOccurred())

		vm.Spec.Preference = &v1.PreferenceMatcher{
			Name: clusterPreference.Name,
			Kind: preferenceKind,
		}

		responseVM := expectExpandedVM(callExpandVMSpecAPI(vm))

		Expect(responseVM.Spec.Instancetype).To(BeNil())
		Expect(responseVM.Spec.Preference).To(BeNil())
		Expect(responseVM.Spec.Template.Spec.Domain.CPU.Cores).To(Equal(clusterInstancetype.Spec.CPU.Guest))
		Expect(responseVM.Spec.Template.Spec.Domain.Memory.Guest.Value()).To(Equal(clusterInstancetype.Spec.Memory.Guest.Value()))
		Expect(responseVM.Spec.Template.Spec.Domain.Devices.Disks).To(HaveLen(1))
		Expect(responseVM.Spec.Template.Spec.Domain.Devices.Disks[0].Name).To(Equal(volumeName))
		Expect(responseVM.Spec.Template.Spec.Domain.Devices.Disks[0].DiskDevice.Disk).ToNot(BeNil())
		Expect(responseVM.Spec.Template.Spec.Domain.Devices.Disks[0].DiskDevice.Disk.Bus).To(Equal(v1.DiskBusVirtio))
	},
		Entry("default (empty kind)", "", ""),
		Entry("singular kind", instancetypeapi.ClusterSingularResourceName, instancetypeapi.ClusterSingularPreferenceResourceName),
		Entry("plural kind", instancetypeapi.ClusterPluralResourceName, instancetypeapi.ClusterPluralPreferenceResourceName),
	)

	DescribeTable("should fail, if there is a conflict when applying instancetype", func(kind string) {
		clusterInstancetype := &instancetypev1beta1.VirtualMachineClusterInstancetype{
			TypeMeta: metav1.TypeMeta{
				Kind:       "VirtualMachineClusterInstancetype",
				APIVersion: instancetypev1beta1.SchemeGroupVersion.String(),
			},
			ObjectMeta: metav1.ObjectMeta{
				Name: "test-cluster-instancetype",
			},
			Spec: instancetypev1beta1.VirtualMachineInstancetypeSpec{
				CPU: instancetypev1beta1.CPUInstancetype{
					Guest: uint32(2),
				},
				Memory: instancetypev1beta1.MemoryInstancetype{
					Guest: resource.MustParse("128Mi"),
				},
			},
		}
		_, err := virtClient.VirtualMachineClusterInstancetype().Create(context.Background(), clusterInstancetype, metav1.CreateOptions{})
		Expect(err).ToNot(HaveOccurred())

		vm.Spec.Instancetype = &v1.InstancetypeMatcher{
			Name: clusterInstancetype.Name,
			Kind: kind,
		}
		vm.Spec.Template.Spec.Domain.CPU = &v1.CPU{
			Sockets: 4,
		}

		status := expectStatusError(callExpandVMSpecAPI(vm), http.StatusBadRequest)
		Expect(status.Message).To(ContainSubstring(conflict.New("spec.template.spec.domain.cpu.sockets").Error()))
	},
		Entry("default (empty kind)", ""),
		Entry("singular kind", instancetypeapi.ClusterSingularResourceName),
		Entry("plural kind", instancetypeapi.ClusterPluralResourceName),
	)

	It("should fail if received invalid JSON", func() {
		status := expectStatusError(callExpandVMSpec(vmNamespace, []byte("this is invalid JSON {{{{")), http.StatusBadRequest)
		Expect(status.Message).To(ContainSubstring("Can not unmarshal Request body to struct"))
	})

	It("should fail if received object is not a VirtualMachine", func() {
		notVM := struct {
			StringField string `json:"stringField"`
			IntField    int    `json:"intField"`
		}{
			StringField: "test",
			IntField:    10,
		}
		jsonBytes, err := json.Marshal(notVM)
		Expect(err).ToNot(HaveOccurred())

		status := expectStatusError(callExpandVMSpec(vmNamespace, jsonBytes), http.StatusBadRequest)
		Expect(status.Message).To(Equal("Object is not a valid VirtualMachine"))
	})

	It("should fail if endpoint namespace is empty", func() {
		vmJSON, err := json.Marshal(vm)
		Expect(err).ToNot(HaveOccurred())

		status := expectStatusError(callExpandVMSpec("", vmJSON), http.StatusBadRequest)
		Expect(status.Message).To(Equal("The request namespace must not be empty"))
	})

	It("should fail, if VM and endpoint namespace are different", func() {
		vm.Namespace = "madethisup"

		status := expectStatusError(callExpandVMSpecAPI(vm), http.StatusBadRequest)
		Expect(status.Message).To(Equal("VM namespace must be empty or " + vmNamespace))
	})

	It("should fail if received empty request body", func() {
		recorder := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodPut, "/", strings.NewReader(""))
		req.Body = nil
		ctx := request.WithRequestInfo(req.Context(), &request.RequestInfo{Namespace: vmNamespace})
		handler.ServeHTTP(recorder, req.WithContext(ctx))

		status := expectStatusError(recorder, http.StatusBadRequest)
		Expect(status.Message).To(Equal("empty request body"))
	})
})
