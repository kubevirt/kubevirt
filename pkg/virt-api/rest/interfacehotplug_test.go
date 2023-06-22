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
 * Copyright 2023 Red Hat, Inc.
 *
 */

package rest

import (
	"bytes"
	"context"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"sync"
	"time"

	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes/fake"
	"k8s.io/client-go/testing"

	"kubevirt.io/kubevirt/pkg/testutils"
	"kubevirt.io/kubevirt/pkg/util/status"

	"github.com/golang/mock/gomock"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/emicklei/go-restful/v3"

	k8smetav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"

	v1 "kubevirt.io/api/core/v1"
	"kubevirt.io/client-go/kubecli"

	virtconfig "kubevirt.io/kubevirt/pkg/virt-config"
)

var _ = Describe("Interface Hotplug Subresource", func() {
	var (
		request    *restful.Request
		response   *restful.Response
		kubeClient *fake.Clientset
		recorder   *httptest.ResponseRecorder
		virtClient *kubecli.MockKubevirtClient
		vmClient   *kubecli.MockVirtualMachineInterface
	)

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
			Phase: v1.KubeVirtPhaseDeploying,
		},
	}

	config, _, kvInformer := testutils.NewFakeClusterConfigUsingKV(kv)
	app := SubresourceAPIApp{}
	BeforeEach(func() {
		ctrl := gomock.NewController(GinkgoT())
		kubeClient = fake.NewSimpleClientset()
		virtClient = kubecli.NewMockKubevirtClient(ctrl)
		vmClient = kubecli.NewMockVirtualMachineInterface(ctrl)

		virtClient.EXPECT().CoreV1().Return(kubeClient.CoreV1()).AnyTimes()
		virtClient.EXPECT().VirtualMachine(k8smetav1.NamespaceDefault).Return(vmClient).AnyTimes()
		virtClient.EXPECT().VirtualMachine("").Return(vmClient).AnyTimes()

		app.virtCli = virtClient
		app.statusUpdater = status.NewVMStatusUpdater(app.virtCli)
		app.credentialsLock = &sync.Mutex{}
		app.handlerTLSConfiguration = &tls.Config{InsecureSkipVerify: true}
		app.clusterConfig = config
		app.handlerHttpClient = &http.Client{
			Transport: &http.Transport{
				TLSClientConfig: app.handlerTLSConfiguration,
			},
			Timeout: 10 * time.Second,
		}

		request = restful.NewRequest(&http.Request{})
		recorder = httptest.NewRecorder()
		response = restful.NewResponse(recorder)
		// Make sure that any unexpected call to the client will fail
		kubeClient.Fake.PrependReactor("*", "*", func(action testing.Action) (handled bool, obj runtime.Object, err error) {
			Expect(action).To(BeNil())
			return true, nil, nil
		})
	})

	enableFeatureGate := func(featureGate string) {
		kvConfig := kv.DeepCopy()
		kvConfig.Spec.Configuration.DeveloperConfiguration.FeatureGates = []string{featureGate}
		testutils.UpdateFakeKubeVirtClusterConfig(kvInformer, kvConfig)
	}
	restoreKubeVirtClusterConfig := func() {
		testutils.UpdateFakeKubeVirtClusterConfig(kvInformer, kv)
	}

	Context("Add Interface Subresource api", func() {
		const (
			ifaceToHotplug      = "pluggediface1"
			networkToHotplug    = "meganet2000"
			existingIfaceName   = "iface1"
			existingNetworkName = "existing-net"
		)
		newAddInterfaceBody := func(opts *v1.AddInterfaceOptions) io.ReadCloser {
			optsJson, _ := json.Marshal(opts)
			return &readCloserWrapper{bytes.NewReader(optsJson)}
		}

		BeforeEach(func() {
			request.PathParameters()["name"] = "testvm"
			request.PathParameters()["namespace"] = "default"
		})

		AfterEach(func() {
			restoreKubeVirtClusterConfig()
		})

		mutateIfaceRequest := func(addOpts *v1.AddInterfaceOptions) {
			if addOpts != nil {
				request.Request.Body = newAddInterfaceBody(addOpts)
			}
		}

		createVM := func(addOpts *v1.AddInterfaceOptions) *v1.VirtualMachine {
			mutateIfaceRequest(addOpts)
			vm := newMinimalVM(request.PathParameter("name"))
			vm.Namespace = "default"

			patchedVM := vm.DeepCopy()
			patchedVM.Status.InterfaceRequests = append(patchedVM.Status.InterfaceRequests, v1.VirtualMachineInterfaceRequest{AddInterfaceOptions: addOpts})

			return vm
		}

		successfulMockScenarioForVM := func(addOpts *v1.AddInterfaceOptions) {
			vm := createVM(addOpts)

			vmClient.EXPECT().Get(context.Background(), vm.Name, &k8smetav1.GetOptions{}).Return(vm, nil)
			vmClient.EXPECT().PatchStatus(context.Background(), vm.Name, types.JSONPatchType, gomock.Any(), &k8smetav1.PatchOptions{}).Return(vm, nil)

			if addOpts != nil {
				app.VMAddInterfaceRequestHandler(request, response)
			}
		}

		failedMockScenarioForVM := func(addOpts *v1.AddInterfaceOptions) {
			_ = createVM(addOpts)

			if addOpts != nil {
				app.VMAddInterfaceRequestHandler(request, response)
			}
		}

		It("Should succeed a dynamic interface request for a VM with a valid add interface request", func() {
			enableFeatureGate(virtconfig.HotplugNetworkIfacesGate)

			successfulMockScenarioForVM(&v1.AddInterfaceOptions{
				NetworkAttachmentDefinitionName: networkToHotplug,
				Name:                            ifaceToHotplug,
			})
			Expect(response.StatusCode()).To(Equal(http.StatusAccepted))
		})

		DescribeTable("Should fail on invalid requests for a dynamic interfaces", func(addOpts *v1.AddInterfaceOptions, mockScenario func(addOpts *v1.AddInterfaceOptions), featuresToEnable ...string) {
			for _, featureToEnable := range featuresToEnable {
				enableFeatureGate(featureToEnable)
			}

			mockScenario(addOpts)
			Expect(response.StatusCode()).To(Equal(http.StatusBadRequest))
		},
			Entry("VM with an invalid add interface request missing a network name", &v1.AddInterfaceOptions{
				Name: ifaceToHotplug,
			}, failedMockScenarioForVM, virtconfig.HotplugNetworkIfacesGate),
			Entry("VM with an invalid add interface request missing the interface name", &v1.AddInterfaceOptions{
				NetworkAttachmentDefinitionName: networkToHotplug,
			}, failedMockScenarioForVM, virtconfig.HotplugNetworkIfacesGate),
			Entry("VM with a valid add interface request but no feature gate", &v1.AddInterfaceOptions{
				NetworkAttachmentDefinitionName: networkToHotplug,
				Name:                            ifaceToHotplug,
			}, failedMockScenarioForVM),
		)

		DescribeTable("Should generate expected vm patch", func(interfaceRequest *v1.VirtualMachineInterfaceRequest, existingInterfaceRequests []v1.VirtualMachineInterfaceRequest, expectedPatch string) {
			vm := newMinimalVM(request.PathParameter("name"))
			vm.Namespace = "default"

			if len(existingInterfaceRequests) > 0 {
				vm.Status.InterfaceRequests = existingInterfaceRequests
			}

			Expect(generateVMInterfaceRequestPatch(vm, interfaceRequest)).To(Equal(expectedPatch))
		},
			Entry(
				"add interface request with no existing interfaces",
				&v1.VirtualMachineInterfaceRequest{
					AddInterfaceOptions: &v1.AddInterfaceOptions{
						NetworkAttachmentDefinitionName: networkToHotplug,
						Name:                            ifaceToHotplug,
					},
				},
				nil,
				fmt.Sprintf(`[{ "op": "test", "path": "/status/interfaceRequests", "value": null}, { "op": "add", "path": "/status/interfaceRequests", "value": [{"addInterfaceOptions":{"networkAttachmentDefinitionName":%q,"name":%q}}]}]`, networkToHotplug, ifaceToHotplug)),
			Entry("add interface request when interface requests already exists",
				&v1.VirtualMachineInterfaceRequest{
					AddInterfaceOptions: &v1.AddInterfaceOptions{
						NetworkAttachmentDefinitionName: networkToHotplug,
						Name:                            ifaceToHotplug,
					},
				},
				[]v1.VirtualMachineInterfaceRequest{
					{
						AddInterfaceOptions: &v1.AddInterfaceOptions{
							NetworkAttachmentDefinitionName: existingNetworkName,
							Name:                            existingIfaceName,
						},
					},
				},
				fmt.Sprintf(`[{ "op": "test", "path": "/status/interfaceRequests", "value": [{"addInterfaceOptions":{"networkAttachmentDefinitionName":%[1]q,"name":%[2]q}}]}, { "op": "add", "path": "/status/interfaceRequests", "value": [{"addInterfaceOptions":{"networkAttachmentDefinitionName":%[1]q,"name":%[2]q}},{"addInterfaceOptions":{"networkAttachmentDefinitionName":%[3]q,"name":%[4]q}}]}]`, existingNetworkName, existingIfaceName, networkToHotplug, ifaceToHotplug)),
			Entry("empty add interface request",
				&v1.VirtualMachineInterfaceRequest{AddInterfaceOptions: nil},
				[]v1.VirtualMachineInterfaceRequest{
					{
						AddInterfaceOptions: &v1.AddInterfaceOptions{
							NetworkAttachmentDefinitionName: existingNetworkName,
							Name:                            existingIfaceName,
						},
					},
				},
				"",
			),
			Entry("already exising add interface request",
				&v1.VirtualMachineInterfaceRequest{AddInterfaceOptions: &v1.AddInterfaceOptions{
					NetworkAttachmentDefinitionName: existingNetworkName,
					Name:                            existingIfaceName,
				},
				},
				[]v1.VirtualMachineInterfaceRequest{
					{
						AddInterfaceOptions: &v1.AddInterfaceOptions{
							NetworkAttachmentDefinitionName: existingNetworkName,
							Name:                            existingIfaceName,
						},
					},
				},
				"",
			),
		)

		It("Should fail to generate expected vm patch when the add interface request already exists", func() {
			vm := newMinimalVM(request.PathParameter("name"))
			vm.Namespace = "default"

			vm.Status.InterfaceRequests = []v1.VirtualMachineInterfaceRequest{
				{
					AddInterfaceOptions: &v1.AddInterfaceOptions{
						NetworkAttachmentDefinitionName: networkToHotplug,
						Name:                            ifaceToHotplug,
					},
				},
			}

			Expect(
				generateVMInterfaceRequestPatch(vm, &v1.VirtualMachineInterfaceRequest{
					AddInterfaceOptions: &v1.AddInterfaceOptions{
						NetworkAttachmentDefinitionName: networkToHotplug,
						Name:                            ifaceToHotplug,
					},
				}),
			).To(BeEmpty())
		})
	})

	Context("Remove Interface Subresource api", func() {
		const (
			ifaceToHotUnplug    = "pluggediface1"
			iface2ToHotUnplug   = "pluggediface2"
			existingNetworkName = ifaceToHotUnplug
		)

		BeforeEach(func() {
			request.PathParameters()["name"] = "testvm"
			request.PathParameters()["namespace"] = "default"
		})

		AfterEach(func() {
			restoreKubeVirtClusterConfig()
		})

		successfulMockScenarioForVM := func(removeOpts *v1.RemoveInterfaceOptions) {
			vm := newMinimalVM(request.PathParameter("name"))

			request.Request.Body = newRemoveInterfaceBody(removeOpts)

			vmClient.EXPECT().Get(context.Background(), vm.Name, &k8smetav1.GetOptions{}).Return(vm, nil)
			vmClient.EXPECT().PatchStatus(context.Background(), vm.Name, types.JSONPatchType, gomock.Any(), &k8smetav1.PatchOptions{}).Return(vm, nil)
		}

		It("Should succeed a remove interface request on VM", func() {
			enableFeatureGate(virtconfig.HotplugNetworkIfacesGate)
			removeOpts := &v1.RemoveInterfaceOptions{Name: ifaceToHotUnplug}
			successfulMockScenarioForVM(removeOpts)

			app.VMRemoveInterfaceRequestHandler(request, response)

			Expect(response.StatusCode()).To(Equal(http.StatusAccepted))
		})

		It("Should return empty string given a VM with existing remove iface request", func() {
			vm := newMinimalVM(request.PathParameter("name"))
			vm.Status.InterfaceRequests = []v1.VirtualMachineInterfaceRequest{{
				RemoveInterfaceOptions: &v1.RemoveInterfaceOptions{
					Name: ifaceToHotUnplug,
				}}}

			req := &v1.VirtualMachineInterfaceRequest{
				RemoveInterfaceOptions: &v1.RemoveInterfaceOptions{
					Name: ifaceToHotUnplug,
				},
			}

			Expect(generateVMInterfaceRequestPatch(vm, req)).To(BeEmpty())
		})

		DescribeTable("should generate expected VM patch", func(interfaceRequest *v1.VirtualMachineInterfaceRequest, existingInterfaceRequests []v1.VirtualMachineInterfaceRequest, expectedPatch string) {
			vm := newMinimalVM(request.PathParameter("name"))
			vm.Status.InterfaceRequests = existingInterfaceRequests

			Expect(generateVMInterfaceRequestPatch(vm, interfaceRequest)).To(MatchJSON(expectedPatch))
		},
			Entry(
				"when no interface request exist",
				&v1.VirtualMachineInterfaceRequest{
					RemoveInterfaceOptions: &v1.RemoveInterfaceOptions{
						Name: ifaceToHotUnplug,
					},
				},
				nil,
				fmt.Sprintf(`[
					{ "op": "test", "path": "/status/interfaceRequests", "value": null}, 
					{ "op": "add", "path": "/status/interfaceRequests", "value": [{"removeInterfaceOptions":{"name":%q}}]}
				]`, ifaceToHotUnplug),
			),
			Entry("when there are other remove interface requests",
				&v1.VirtualMachineInterfaceRequest{
					RemoveInterfaceOptions: &v1.RemoveInterfaceOptions{
						Name: iface2ToHotUnplug,
					},
				},
				[]v1.VirtualMachineInterfaceRequest{
					{
						RemoveInterfaceOptions: &v1.RemoveInterfaceOptions{
							Name: ifaceToHotUnplug,
						},
					},
				},
				fmt.Sprintf(`[
					{ "op": "test", "path": "/status/interfaceRequests", "value": [ 
						{"removeInterfaceOptions": {"name":%[1]q}}
					]}, 
					{ "op": "add", "path": "/status/interfaceRequests", "value": [ 
						{"removeInterfaceOptions": {"name":%[1]q}},
						{"removeInterfaceOptions": {"name":%[2]q}}
					]}
				]`, ifaceToHotUnplug, iface2ToHotUnplug),
			),
		)
	})

	It("VM interface request patch, should return empty string given VM with empty iface request", func() {
		vm := newMinimalVM(request.PathParameter("name"))

		Expect(generateVMInterfaceRequestPatch(vm, &v1.VirtualMachineInterfaceRequest{})).To(BeEmpty())
	})

	It("VM interface request patch, should generate patch given remove iface request and a VM with add and remove iface requests", func() {
		const (
			iface1             = "red"
			iface1NetAttachDef = "red-br"
			iface2             = "blue"
			iface3             = "green"
		)
		vm := newMinimalVM(request.PathParameter("name"))
		vm.Status.InterfaceRequests = []v1.VirtualMachineInterfaceRequest{
			{AddInterfaceOptions: &v1.AddInterfaceOptions{Name: iface1, NetworkAttachmentDefinitionName: iface1NetAttachDef}},
			{RemoveInterfaceOptions: &v1.RemoveInterfaceOptions{Name: iface2}},
		}

		Expect(
			generateVMInterfaceRequestPatch(
				vm,
				&v1.VirtualMachineInterfaceRequest{
					RemoveInterfaceOptions: &v1.RemoveInterfaceOptions{Name: iface3},
				},
			),
		).To(
			MatchJSON(
				fmt.Sprintf(`[
						{ "op": "test", "path": "/status/interfaceRequests", "value": [
							{"addInterfaceOptions":{"name":%[1]q, "networkAttachmentDefinitionName": %[2]q}},
							{"removeInterfaceOptions":{"name":%[3]q}}
						]}, 
						{ "op": "add", "path": "/status/interfaceRequests", "value": [
							{"addInterfaceOptions":{"name":%[1]q, "networkAttachmentDefinitionName": %[2]q}},
							{"removeInterfaceOptions":{"name":%[3]q}},
							{"removeInterfaceOptions":{"name":%[4]q}}
						]}]`,
					iface1, iface1NetAttachDef, iface2, iface3),
			),
		)
	})

	const (
		iface1            = "red"
		iface1NetworkName = "red-br"
	)
	DescribeTable("VM interface request patch, should return empty string",
		func(interfaceRequest *v1.VirtualMachineInterfaceRequest, existingInterfaceRequests []v1.VirtualMachineInterfaceRequest) {
			vm := newMinimalVM(request.PathParameter("name"))
			vm.Status.InterfaceRequests = existingInterfaceRequests

			Expect(generateVMInterfaceRequestPatch(vm, interfaceRequest)).To(BeEmpty())
		},
		Entry("given add iface request, and a VM with exising remove request for the same iface",
			&v1.VirtualMachineInterfaceRequest{
				AddInterfaceOptions: &v1.AddInterfaceOptions{Name: iface1},
			},
			[]v1.VirtualMachineInterfaceRequest{
				{RemoveInterfaceOptions: &v1.RemoveInterfaceOptions{Name: iface1}},
			},
		),
		Entry("given remove iface request, and a VM with exising add request for the same iface",
			&v1.VirtualMachineInterfaceRequest{
				RemoveInterfaceOptions: &v1.RemoveInterfaceOptions{Name: iface1},
			},
			[]v1.VirtualMachineInterfaceRequest{
				{AddInterfaceOptions: &v1.AddInterfaceOptions{Name: iface1, NetworkAttachmentDefinitionName: iface1NetworkName}},
			},
		),
	)
})

func newRemoveInterfaceBody(opts *v1.RemoveInterfaceOptions) io.ReadCloser {
	optsJson, _ := json.Marshal(opts)
	return &readCloserWrapper{bytes.NewReader(optsJson)}
}
