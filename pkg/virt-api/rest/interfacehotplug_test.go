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
	"net/http/httptest"
	"sync"
	"time"

	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes/fake"
	"k8s.io/client-go/testing"

	"kubevirt.io/kubevirt/pkg/testutils"
	"kubevirt.io/kubevirt/pkg/util/status"

	"net/http"

	"github.com/golang/mock/gomock"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/emicklei/go-restful"

	k8smetav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"

	v1 "kubevirt.io/api/core/v1"
	"kubevirt.io/client-go/api"
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
		vmiClient  *kubecli.MockVirtualMachineInstanceInterface
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
		vmiClient = kubecli.NewMockVirtualMachineInstanceInterface(ctrl)

		virtClient.EXPECT().CoreV1().Return(kubeClient.CoreV1()).AnyTimes()
		virtClient.EXPECT().VirtualMachine(k8smetav1.NamespaceDefault).Return(vmClient).AnyTimes()
		virtClient.EXPECT().VirtualMachine("").Return(vmClient).AnyTimes()
		virtClient.EXPECT().VirtualMachineInstance(k8smetav1.NamespaceDefault).Return(vmiClient).AnyTimes()
		virtClient.EXPECT().VirtualMachineInstance("").Return(vmiClient).AnyTimes()

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

		createVMI := func(addOpts *v1.AddInterfaceOptions) *v1.VirtualMachineInstance {
			mutateIfaceRequest(addOpts)
			vmi := api.NewMinimalVMI(request.PathParameter("name"))
			vmi.Namespace = "default"
			vmi.Status.Phase = v1.Running
			vmi.Spec.Domain.Devices.Interfaces = append(vmi.Spec.Domain.Devices.Interfaces, v1.Interface{
				Name: existingIfaceName,
			})
			vmi.Spec.Networks = append(vmi.Spec.Networks, v1.Network{
				Name: existingIfaceName,
				NetworkSource: v1.NetworkSource{
					Multus: &v1.MultusNetwork{
						NetworkName: existingNetworkName,
					},
				},
			})
			return vmi
		}

		successfulMockScenarioForVMI := func(addOpts *v1.AddInterfaceOptions) {
			vmi := createVMI(addOpts)

			vmiClient.EXPECT().Get(context.Background(), vmi.Name, &k8smetav1.GetOptions{}).Return(vmi, nil)
			vmiClient.EXPECT().Patch(context.Background(), vmi.Name, types.JSONPatchType, gomock.Any(), &k8smetav1.PatchOptions{}).Return(vmi, nil)

			if addOpts != nil {
				app.VMIAddInterfaceRequestHandler(request, response)
			}
		}

		DescribeTable("Should succeed a dynamic interface request", func(addOpts *v1.AddInterfaceOptions, mockScenario func(addOpts *v1.AddInterfaceOptions)) {
			enableFeatureGate(virtconfig.HotplugNetworkIfacesGate)

			mockScenario(addOpts)
			Expect(response.StatusCode()).To(Equal(http.StatusAccepted))
		},
			Entry("VM with a valid add interface request", &v1.AddInterfaceOptions{
				NetworkName:   networkToHotplug,
				InterfaceName: ifaceToHotplug,
			}, successfulMockScenarioForVM),
			Entry("VMI with a valid add interface request", &v1.AddInterfaceOptions{
				NetworkName:   networkToHotplug,
				InterfaceName: ifaceToHotplug,
			}, successfulMockScenarioForVMI),
		)

		DescribeTable("Should fail on invalid requests for a dynamic interfaces", func(addOpts *v1.AddInterfaceOptions, mockScenario func(addOpts *v1.AddInterfaceOptions), featuresToEnable ...string) {
			for _, featureToEnable := range featuresToEnable {
				enableFeatureGate(featureToEnable)
			}

			mockScenario(addOpts)
			Expect(response.StatusCode()).To(Equal(http.StatusBadRequest))
		},
			Entry("VM with an invalid add interface request missing a network name", &v1.AddInterfaceOptions{
				InterfaceName: ifaceToHotplug,
			}, failedMockScenarioForVM, virtconfig.HotplugNetworkIfacesGate),
			Entry("VM with an invalid add interface request missing the interface name", &v1.AddInterfaceOptions{
				NetworkName: networkToHotplug,
			}, failedMockScenarioForVM, virtconfig.HotplugNetworkIfacesGate),
			Entry("VM with a valid add interface request but no feature gate", &v1.AddInterfaceOptions{
				NetworkName:   networkToHotplug,
				InterfaceName: ifaceToHotplug,
			}, failedMockScenarioForVM),
		)

		It("Should generate expected vmi patch when the add interface request features all required data", func() {
			vmi := api.NewMinimalVMI(request.PathParameter("name"))
			vmi.Namespace = "default"
			vmi.Status.Phase = v1.Running
			vmi.Spec.Domain.Devices.Interfaces = append(vmi.Spec.Domain.Devices.Interfaces, v1.Interface{
				Name: existingIfaceName,
			})
			vmi.Spec.Networks = append(vmi.Spec.Networks, v1.Network{
				Name: existingIfaceName,
			})

			Expect(
				generateVMIInterfaceRequestPatch(
					vmi,
					&v1.VirtualMachineInterfaceRequest{
						AddInterfaceOptions: &v1.AddInterfaceOptions{
							NetworkName:   networkToHotplug,
							InterfaceName: ifaceToHotplug,
						},
					},
				),
			).To(
				Equal(
					fmt.Sprintf(`[{ "op": "test", "path": "/spec/networks", "value": [{"name":%[1]q}]}, { "op": "test", "path": "/spec/domain/devices/interfaces", "value": [{"name":%[1]q}]}, { "op": "add", "path": "/spec/networks", "value": [{"name":%[1]q},{"name":%[2]q,"multus":{"networkName":%[3]q}}]}, { "op": "add", "path": "/spec/domain/devices/interfaces", "value": [{"name":%[1]q},{"name":%[2]q,"bridge":{}}]}]`, existingIfaceName, ifaceToHotplug, networkToHotplug),
				),
			)
		})

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
						NetworkName:   networkToHotplug,
						InterfaceName: ifaceToHotplug,
					},
				},
				nil,
				fmt.Sprintf(`[{ "op": "test", "path": "/status/interfaceRequests", "value": null}, { "op": "add", "path": "/status/interfaceRequests", "value": [{"addInterfaceOptions":{"networkName":%q,"interfaceName":%q}}]}]`, networkToHotplug, ifaceToHotplug)),
			Entry("add interface request when interface requests already exists",
				&v1.VirtualMachineInterfaceRequest{
					AddInterfaceOptions: &v1.AddInterfaceOptions{
						NetworkName:   networkToHotplug,
						InterfaceName: ifaceToHotplug,
					},
				},
				[]v1.VirtualMachineInterfaceRequest{
					{
						AddInterfaceOptions: &v1.AddInterfaceOptions{
							NetworkName:   existingNetworkName,
							InterfaceName: existingIfaceName,
						},
					},
				},
				fmt.Sprintf(`[{ "op": "test", "path": "/status/interfaceRequests", "value": [{"addInterfaceOptions":{"networkName":%[1]q,"interfaceName":%[2]q}}]}, { "op": "add", "path": "/status/interfaceRequests", "value": [{"addInterfaceOptions":{"networkName":%[1]q,"interfaceName":%[2]q}},{"addInterfaceOptions":{"networkName":%[3]q,"interfaceName":%[4]q}}]}]`, existingNetworkName, existingIfaceName, networkToHotplug, ifaceToHotplug)),
			Entry("empty add interface request",
				&v1.VirtualMachineInterfaceRequest{AddInterfaceOptions: nil},
				[]v1.VirtualMachineInterfaceRequest{
					{
						AddInterfaceOptions: &v1.AddInterfaceOptions{
							NetworkName:   existingNetworkName,
							InterfaceName: existingIfaceName,
						},
					},
				}, ""))

		It("Should fail to generate expected vm patch when the add interface request already exists", func() {
			vm := newMinimalVM(request.PathParameter("name"))
			vm.Namespace = "default"

			vm.Status.InterfaceRequests = []v1.VirtualMachineInterfaceRequest{
				{
					AddInterfaceOptions: &v1.AddInterfaceOptions{
						NetworkName:   networkToHotplug,
						InterfaceName: ifaceToHotplug,
					},
				},
			}

			Expect(
				generateVMInterfaceRequestPatch(vm, &v1.VirtualMachineInterfaceRequest{
					AddInterfaceOptions: &v1.AddInterfaceOptions{
						NetworkName:   networkToHotplug,
						InterfaceName: ifaceToHotplug,
					},
				}),
			).To(BeEmpty())
		})
	})
})
