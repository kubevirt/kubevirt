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
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"

	"github.com/emicklei/go-restful/v3"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"go.uber.org/mock/gomock"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/resource"
	k8smetav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"

	kubevirtcore "kubevirt.io/api/core"
	v1 "kubevirt.io/api/core/v1"
	instancetypev1beta1 "kubevirt.io/api/instancetype/v1beta1"
	"kubevirt.io/client-go/kubecli"
	"kubevirt.io/client-go/kubevirt/fake"

	"kubevirt.io/kubevirt/pkg/instancetype/conflict"
	"kubevirt.io/kubevirt/pkg/pointer"
	"kubevirt.io/kubevirt/pkg/testutils"
)

var _ = Describe("Instancetype expansion subresources", func() {
	const (
		vmName      = "test-vm"
		vmNamespace = "test-namespace"
		volumeName  = "volumeName"
	)

	var (
		vmClient   *kubecli.MockVirtualMachineInterface
		virtClient *kubecli.MockKubevirtClient
		app        *SubresourceAPIApp

		request  *restful.Request
		recorder *httptest.ResponseRecorder
		response *restful.Response

		vm *v1.VirtualMachine
	)

	BeforeEach(func() {
		ctrl := gomock.NewController(GinkgoT())
		vmClient = kubecli.NewMockVirtualMachineInterface(ctrl)
		virtClient = kubecli.NewMockKubevirtClient(ctrl)
		virtClient.EXPECT().GeneratedKubeVirtClient().Return(fake.NewSimpleClientset()).AnyTimes()
		virtClient.EXPECT().VirtualMachine(vmNamespace).Return(vmClient).AnyTimes()

		fakeInstancetypeClients := fake.NewSimpleClientset().InstancetypeV1beta1()
		virtClient.EXPECT().VirtualMachineClusterInstancetype().Return(fakeInstancetypeClients.VirtualMachineClusterInstancetypes()).AnyTimes()
		virtClient.EXPECT().VirtualMachineClusterPreference().Return(fakeInstancetypeClients.VirtualMachineClusterPreferences()).AnyTimes()

		kv := &v1.KubeVirt{
			ObjectMeta: k8smetav1.ObjectMeta{
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
		app = NewSubresourceAPIApp(virtClient, 0, nil, config)

		request = restful.NewRequest(&http.Request{})
		recorder = httptest.NewRecorder()
		response = restful.NewResponse(recorder)
		response.SetRequestAccepts(restful.MIME_JSON)

		vm = &v1.VirtualMachine{
			TypeMeta: metav1.TypeMeta{},
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

	testCommonFunctionality := func(callExpandSpecApi func(vm *v1.VirtualMachine) *httptest.ResponseRecorder, expectedStatusError int) {
		It("should return unchanged VM, if no instancetype and preference is assigned", func() {
			vm.Spec.Instancetype = nil

			recorder := callExpandSpecApi(vm)
			Expect(recorder.Code).To(Equal(http.StatusOK))

			responseVm := &v1.VirtualMachine{}
			Expect(json.NewDecoder(recorder.Body).Decode(responseVm)).To(Succeed())
			Expect(responseVm).To(Equal(vm))
		})

		It("should fail if VM points to nonexistent instancetype", func() {
			vm.Spec.Instancetype = &v1.InstancetypeMatcher{
				Name: "nonexistent-instancetype",
			}

			recorder := callExpandSpecApi(vm)
			statusErr := ExpectStatusErrorWithCode(recorder, expectedStatusError)
			Expect(statusErr.Status().Message).To(ContainSubstring("not found"))
		})

		It("should fail if VM points to nonexistent preference", func() {
			vm.Spec.Preference = &v1.PreferenceMatcher{
				Name: "nonexistent-preference",
			}

			recorder := callExpandSpecApi(vm)
			statusErr := ExpectStatusErrorWithCode(recorder, expectedStatusError)
			Expect(statusErr.Status().Message).To(ContainSubstring("not found"))
		})

		It("should expand instancetype and preference within VM", func() {
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
			}

			recorder := callExpandSpecApi(vm)
			Expect(recorder.Code).To(Equal(http.StatusOK))
			responseVm := &v1.VirtualMachine{}
			Expect(json.NewDecoder(recorder.Body).Decode(responseVm)).To(Succeed())

			Expect(responseVm.Spec.Instancetype).To(BeNil())
			Expect(responseVm.Spec.Preference).To(BeNil())

			Expect(responseVm.Spec.Template.ObjectMeta.Annotations).To(Equal(clusterInstancetype.Spec.Annotations))
			Expect(responseVm.Spec.Template.Spec.Domain.CPU.Cores).To(Equal(clusterInstancetype.Spec.CPU.Guest))
			Expect(responseVm.Spec.Template.Spec.Domain.Memory.Guest.Value()).To(Equal(clusterInstancetype.Spec.Memory.Guest.Value()))

			Expect(responseVm.Spec.Template.Spec.Domain.Devices.Disks).To(HaveLen(1))
			Expect(responseVm.Spec.Template.Spec.Domain.Devices.Disks[0].Name).To(Equal(volumeName))
			Expect(responseVm.Spec.Template.Spec.Domain.Devices.Disks[0].DiskDevice.Disk).ToNot(BeNil())
			Expect(responseVm.Spec.Template.Spec.Domain.Devices.Disks[0].DiskDevice.Disk.Bus).To(Equal(v1.DiskBusVirtio))
			Expect(responseVm.Spec.Template.Spec.Networks).To(HaveLen(1))
			Expect(responseVm.Spec.Template.Spec.Networks[0].Name).To(Equal("default"))
		})

		It("should fail, if there is a conflict when applying instancetype", func() {
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
			}
			vm.Spec.Template.Spec.Domain.CPU = &v1.CPU{
				Sockets: 4,
			}

			recorder := callExpandSpecApi(vm)
			statusErr := ExpectStatusErrorWithCode(recorder, expectedStatusError)
			Expect(statusErr.Status().Message).To(ContainSubstring(conflict.New("spec.template.spec.domain.cpu.sockets").Error()))
		})
	}

	Context("VirtualMachine expand-spec endpoint", func() {
		callExpandSpecApi := func(vm *v1.VirtualMachine) *httptest.ResponseRecorder {
			request.PathParameters()["name"] = vmName
			request.PathParameters()["namespace"] = vmNamespace

			vmClient.EXPECT().Get(context.Background(), vmName, gomock.Any()).Return(vm, nil).AnyTimes()

			app.ExpandSpecVMRequestHandler(request, response)
			return recorder
		}

		testCommonFunctionality(callExpandSpecApi, http.StatusInternalServerError)

		It("should fail if VM does not exist", func() {
			request.PathParameters()["name"] = "nonexistent-vm"
			request.PathParameters()["namespace"] = vmNamespace

			vmClient.EXPECT().Get(context.Background(), gomock.Any(), gomock.Any()).Return(nil, errors.NewNotFound(
				schema.GroupResource{
					Group:    kubevirtcore.GroupName,
					Resource: "VirtualMachine",
				},
				"",
			)).AnyTimes()

			app.ExpandSpecVMRequestHandler(request, response)
			statusErr := ExpectStatusErrorWithCode(recorder, http.StatusNotFound)
			Expect(statusErr.Status().Message).To(Equal("virtualmachine.kubevirt.io \"nonexistent-vm\" not found"))
		})
	})

	Context("expand-vm-spec endpoint", func() {
		callExpandSpecApi := func(vm *v1.VirtualMachine) *httptest.ResponseRecorder {
			request.PathParameters()["namespace"] = vmNamespace

			vmJson, err := json.Marshal(vm)
			Expect(err).ToNot(HaveOccurred())
			request.Request.Body = io.NopCloser(bytes.NewBuffer(vmJson))

			app.ExpandSpecRequestHandler(request, response)
			return recorder
		}

		testCommonFunctionality(callExpandSpecApi, http.StatusBadRequest)

		It("should fail if received invalid JSON", func() {
			request.PathParameters()["namespace"] = vmNamespace

			invalidJson := "this is invalid JSON {{{{"
			request.Request.Body = io.NopCloser(strings.NewReader(invalidJson))

			app.ExpandSpecRequestHandler(request, response)
			statusErr := ExpectStatusErrorWithCode(recorder, http.StatusBadRequest)
			Expect(statusErr.Status().Message).To(ContainSubstring("Can not unmarshal Request body to struct"))
		})

		It("should fail if received object is not a VirtualMachine", func() {
			request.PathParameters()["namespace"] = vmNamespace

			notVm := struct {
				StringField string `json:"stringField"`
				IntField    int    `json:"intField"`
			}{
				StringField: "test",
				IntField:    10,
			}

			jsonBytes, err := json.Marshal(notVm)
			Expect(err).ToNot(HaveOccurred())
			request.Request.Body = io.NopCloser(bytes.NewBuffer(jsonBytes))

			app.ExpandSpecRequestHandler(request, response)
			statusErr := ExpectStatusErrorWithCode(recorder, http.StatusBadRequest)
			Expect(statusErr.Status().Message).To(Equal("Object is not a valid VirtualMachine"))
		})

		It("should fail if endpoint namespace is empty", func() {
			request.PathParameters()["namespace"] = ""

			vmJson, err := json.Marshal(vm)
			Expect(err).ToNot(HaveOccurred())
			request.Request.Body = io.NopCloser(bytes.NewBuffer(vmJson))

			app.ExpandSpecRequestHandler(request, response)
			statusErr := ExpectStatusErrorWithCode(recorder, http.StatusBadRequest)
			Expect(statusErr.Status().Message).To(Equal("The request namespace must not be empty"))
		})

		It("should fail, if VM and endpoint namespace are different", func() {
			vm.Namespace = "madethisup"

			recorder = callExpandSpecApi(vm)
			statusErr := ExpectStatusErrorWithCode(recorder, http.StatusBadRequest)
			errMsg := fmt.Sprintf("VM namespace must be empty or %s", vmNamespace)
			Expect(statusErr.Status().Message).To(Equal(errMsg))
		})
	})
})
