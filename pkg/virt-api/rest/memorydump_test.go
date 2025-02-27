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
	"crypto/tls"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"

	"github.com/emicklei/go-restful/v3"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/ghttp"
	"github.com/onsi/gomega/gstruct"
	"go.uber.org/mock/gomock"

	k8sv1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes/fake"
	"k8s.io/client-go/testing"

	v1 "kubevirt.io/api/core/v1"
	cdifake "kubevirt.io/client-go/containerizeddataimporter/fake"
	"kubevirt.io/client-go/kubecli"
	kubevirtfake "kubevirt.io/client-go/kubevirt/fake"
	cdiv1 "kubevirt.io/containerized-data-importer-api/pkg/apis/core/v1beta1"

	"kubevirt.io/kubevirt/pkg/apimachinery/patch"
	"kubevirt.io/kubevirt/pkg/libvmi"
	libvmistatus "kubevirt.io/kubevirt/pkg/libvmi/status"
	storagetypes "kubevirt.io/kubevirt/pkg/storage/types"
	"kubevirt.io/kubevirt/pkg/testutils"
	"kubevirt.io/kubevirt/pkg/virt-config/featuregate"
)

var _ = Describe("Memory dump Subresource api", func() {
	const (
		fs          = false
		block       = true
		notReadOnly = false
		readOnly    = true
		testPVCName = "testPVC"
	)

	var (
		request        *restful.Request
		response       *restful.Response
		kubeClient     *fake.Clientset
		fakeVirtClient *kubevirtfake.Clientset
		virtClient     *kubecli.MockKubevirtClient
		app            *SubresourceAPIApp

		kv = &v1.KubeVirt{
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
				Phase: v1.KubeVirtPhaseDeploying,
			},
		}
	)

	config, _, kvStore := testutils.NewFakeClusterConfigUsingKV(kv)

	enableFeatureGate := func(featureGate string) {
		kvConfig := kv.DeepCopy()
		kvConfig.Spec.Configuration.DeveloperConfiguration.FeatureGates = []string{featureGate}
		testutils.UpdateFakeKubeVirtClusterConfig(kvStore, kvConfig)
	}

	disableFeatureGates := func() {
		testutils.UpdateFakeKubeVirtClusterConfig(kvStore, kv)
	}

	cdiConfigInit := func() (cdiConfig *cdiv1.CDIConfig) {
		cdiConfig = &cdiv1.CDIConfig{
			ObjectMeta: metav1.ObjectMeta{
				Name: storagetypes.ConfigName,
			},
			Spec: cdiv1.CDIConfigSpec{
				UploadProxyURLOverride: nil,
			},
			Status: cdiv1.CDIConfigStatus{
				FilesystemOverhead: &cdiv1.FilesystemOverhead{
					Global: cdiv1.Percent(storagetypes.DefaultFSOverhead),
				},
			},
		}
		return
	}

	BeforeEach(func() {
		request = restful.NewRequest(&http.Request{})
		request.PathParameters()["name"] = testVMName
		request.PathParameters()["namespace"] = metav1.NamespaceDefault
		recorder := httptest.NewRecorder()
		response = restful.NewResponse(recorder)

		backend := ghttp.NewTLSServer()
		backendAddr := strings.Split(backend.Addr(), ":")
		backendPort, err := strconv.Atoi(backendAddr[1])
		Expect(err).ToNot(HaveOccurred())
		ctrl := gomock.NewController(GinkgoT())

		virtClient = kubecli.NewMockKubevirtClient(ctrl)
		kubeClient = fake.NewSimpleClientset()
		fakeVirtClient = kubevirtfake.NewSimpleClientset()

		virtClient.EXPECT().CoreV1().Return(kubeClient.CoreV1()).AnyTimes()
		virtClient.EXPECT().VirtualMachine(metav1.NamespaceDefault).Return(fakeVirtClient.KubevirtV1().VirtualMachines(metav1.NamespaceDefault)).AnyTimes()
		virtClient.EXPECT().VirtualMachineInstance(metav1.NamespaceDefault).Return(fakeVirtClient.KubevirtV1().VirtualMachineInstances(metav1.NamespaceDefault)).AnyTimes()

		cdiConfig := cdiConfigInit()
		cdiClient := cdifake.NewSimpleClientset(cdiConfig)
		virtClient.EXPECT().CdiClient().Return(cdiClient).AnyTimes()

		app = NewSubresourceAPIApp(virtClient, backendPort, &tls.Config{InsecureSkipVerify: true}, config)
	})

	AfterEach(func() {
		disableFeatureGates()
	})

	newMemoryDumpBody := func(req *v1.VirtualMachineMemoryDumpRequest) io.ReadCloser {
		reqJson, _ := json.Marshal(req)
		return &readCloserWrapper{bytes.NewReader(reqJson)}
	}

	createTestPVC := func(size string, blockMode bool, readOnlyMode bool) *k8sv1.PersistentVolumeClaim {
		quantity, _ := resource.ParseQuantity(size)
		pvc := &k8sv1.PersistentVolumeClaim{
			ObjectMeta: metav1.ObjectMeta{
				Name:      testPVCName,
				Namespace: metav1.NamespaceDefault,
			},
			Spec: k8sv1.PersistentVolumeClaimSpec{
				Resources: k8sv1.VolumeResourceRequirements{
					Requests: k8sv1.ResourceList{
						k8sv1.ResourceStorage: quantity,
					},
				},
			},
		}
		if blockMode {
			volumeMode := k8sv1.PersistentVolumeBlock
			pvc.Spec.VolumeMode = &volumeMode
		}
		if readOnlyMode {
			pvc.Spec.AccessModes = []k8sv1.PersistentVolumeAccessMode{k8sv1.ReadOnlyMany}
		}
		return pvc
	}

	DescribeTable("With memory dump request", func(memDumpReq *v1.VirtualMachineMemoryDumpRequest, statusCode int, enableGate bool, vmiRunning bool, pvc *k8sv1.PersistentVolumeClaim) {
		if enableGate {
			enableFeatureGate(featuregate.HotplugVolumesGate)
		}
		request.Request.Body = newMemoryDumpBody(memDumpReq)

		vmi := libvmi.New()
		vm := libvmi.NewVirtualMachine(vmi)
		vm.Name = request.PathParameter("name")
		vm.Namespace = metav1.NamespaceDefault
		vm, err := fakeVirtClient.KubevirtV1().VirtualMachines(vm.Namespace).Create(context.TODO(), vm, metav1.CreateOptions{})
		Expect(err).ToNot(HaveOccurred())

		if vmiRunning {
			vmi = libvmi.New(
				libvmi.WithName(testVMName),
				libvmi.WithResourceMemory("1Gi"),
				libvmistatus.WithStatus(libvmistatus.New(libvmistatus.WithPhase(v1.Running))),
			)
			kubeClient.Fake.PrependReactor("get", "persistentvolumeclaims", func(action testing.Action) (bool, runtime.Object, error) {
				get, ok := action.(testing.GetAction)
				Expect(ok).To(BeTrue())
				Expect(get.GetNamespace()).To(Equal(metav1.NamespaceDefault))
				Expect(get.GetName()).To(Equal(testPVCName))
				if pvc == nil {
					return true, nil, errors.NewNotFound(v1.Resource("persistentvolumeclaim"), testPVCName)
				}
				return true, pvc, nil
			})
			vmi, err = fakeVirtClient.KubevirtV1().VirtualMachineInstances(vm.Namespace).Create(context.TODO(), vmi, metav1.CreateOptions{})
			Expect(err).ToNot(HaveOccurred())
		}

		app.MemoryDumpVMRequestHandler(request, response)

		Expect(response.StatusCode()).To(Equal(statusCode))
		if statusCode == http.StatusAccepted {
			patchedVM, err := fakeVirtClient.KubevirtV1().VirtualMachines(vm.Namespace).Get(context.TODO(), vm.Name, metav1.GetOptions{})
			Expect(err).ToNot(HaveOccurred())
			Expect(patchedVM.Status.MemoryDumpRequest).To(gstruct.PointTo(gstruct.MatchFields(gstruct.IgnoreExtras, gstruct.Fields{
				"ClaimName": Equal(memDumpReq.ClaimName),
				"Phase":     Equal(v1.MemoryDumpAssociating),
			})))
		}
	},
		Entry("VM with a valid memory dump request should succeed", &v1.VirtualMachineMemoryDumpRequest{
			ClaimName: testPVCName,
		}, http.StatusAccepted, true, true, createTestPVC("2Gi", fs, notReadOnly)),
		Entry("VM with a valid memory dump request but no feature gate should fail", &v1.VirtualMachineMemoryDumpRequest{
			ClaimName: testPVCName,
		}, http.StatusBadRequest, false, true, createTestPVC("2Gi", fs, notReadOnly)),
		Entry("VM with a valid memory dump request vmi not running should fail", &v1.VirtualMachineMemoryDumpRequest{
			ClaimName: testPVCName,
		}, http.StatusNotFound, true, false, createTestPVC("2Gi", fs, notReadOnly)),
		Entry("VM with a memory dump request with a non existing PVC", &v1.VirtualMachineMemoryDumpRequest{
			ClaimName: testPVCName,
		}, http.StatusNotFound, true, true, nil),
		Entry("VM with a memory dump request pvc block mode should fail", &v1.VirtualMachineMemoryDumpRequest{
			ClaimName: testPVCName,
		}, http.StatusConflict, true, true, createTestPVC("2Gi", block, notReadOnly)),
		Entry("VM with a memory dump request pvc read only mode should fail", &v1.VirtualMachineMemoryDumpRequest{
			ClaimName: testPVCName,
		}, http.StatusConflict, true, true, createTestPVC("2Gi", fs, readOnly)),
		Entry("VM with a memory dump request pvc size too small should fail", &v1.VirtualMachineMemoryDumpRequest{
			ClaimName: testPVCName,
		}, http.StatusConflict, true, true, createTestPVC("1Gi", fs, notReadOnly)),
	)

	DescribeTable("With memory dump request", func(memDumpReq, prevMemDumpReq *v1.VirtualMachineMemoryDumpRequest, statusCode int) {
		enableFeatureGate(featuregate.HotplugVolumesGate)
		request.Request.Body = newMemoryDumpBody(memDumpReq)
		vmi := libvmi.New(
			libvmi.WithName(testVMName),
			libvmi.WithResourceMemory("1Gi"),
			libvmistatus.WithStatus(libvmistatus.New(libvmistatus.WithPhase(v1.Running))),
		)
		vm := libvmi.NewVirtualMachine(vmi)
		vm.Name = request.PathParameter("name")
		vm.Namespace = metav1.NamespaceDefault
		if prevMemDumpReq != nil {
			vm.Status.MemoryDumpRequest = prevMemDumpReq
		}

		vm, err := fakeVirtClient.KubevirtV1().VirtualMachines(vm.Namespace).Create(context.TODO(), vm, metav1.CreateOptions{})
		Expect(err).ToNot(HaveOccurred())
		vmi, err = fakeVirtClient.KubevirtV1().VirtualMachineInstances(vm.Namespace).Create(context.TODO(), vmi, metav1.CreateOptions{})
		Expect(err).ToNot(HaveOccurred())

		kubeClient.Fake.PrependReactor("get", "persistentvolumeclaims", func(action testing.Action) (bool, runtime.Object, error) {
			_, ok := action.(testing.GetAction)
			Expect(ok).To(BeTrue())
			return true, createTestPVC("2Gi", fs, notReadOnly), nil
		})
		app.MemoryDumpVMRequestHandler(request, response)

		Expect(response.StatusCode()).To(Equal(statusCode))
		if statusCode == http.StatusAccepted {
			patchedVM, err := fakeVirtClient.KubevirtV1().VirtualMachines(vm.Namespace).Get(context.TODO(), vm.Name, metav1.GetOptions{})
			Expect(err).ToNot(HaveOccurred())
			Expect(patchedVM.Status.MemoryDumpRequest).To(gstruct.PointTo(gstruct.MatchFields(gstruct.IgnoreExtras, gstruct.Fields{
				"ClaimName": Equal(prevMemDumpReq.ClaimName),
				"Phase":     Equal(v1.MemoryDumpAssociating),
			})))
		}
	},
		Entry("VM with a memory dump request without claim name with assocaited memory dump should succeed",
			&v1.VirtualMachineMemoryDumpRequest{},
			&v1.VirtualMachineMemoryDumpRequest{
				ClaimName: testPVCName,
				Phase:     v1.MemoryDumpCompleted,
			}, http.StatusAccepted),
		Entry("VM with a memory dump request missing claim name without previous memory dump should fail",
			&v1.VirtualMachineMemoryDumpRequest{}, nil, http.StatusBadRequest),
		Entry("VM with a memory dump request with claim name different then assocaited memory dump should fail",
			&v1.VirtualMachineMemoryDumpRequest{
				ClaimName: "diffPVCName",
			},
			&v1.VirtualMachineMemoryDumpRequest{
				ClaimName: testPVCName,
				Phase:     v1.MemoryDumpCompleted,
			}, http.StatusConflict),
	)

	DescribeTable("Should generate expected vm patch", func(memDumpReq *v1.VirtualMachineMemoryDumpRequest, existingMemDumpReq *v1.VirtualMachineMemoryDumpRequest, expectedPatchSet *patch.PatchSet, expectError bool, removeReq bool) {
		vm := libvmi.NewVirtualMachine(libvmi.New())
		vm.Name = request.PathParameter("name")
		vm.Namespace = metav1.NamespaceDefault

		if existingMemDumpReq != nil {
			vm.Status.MemoryDumpRequest = existingMemDumpReq
		}

		patch, err := generateVMMemoryDumpRequestPatch(vm, memDumpReq, removeReq)
		if expectError {
			Expect(err).To(HaveOccurred())
			Expect(patch).To(BeEmpty())
			return
		}

		Expect(err).ToNot(HaveOccurred())
		patchBytes, err := expectedPatchSet.GeneratePayload()
		Expect(err).ToNot(HaveOccurred())
		Expect(patch).To(Equal(patchBytes))
	},
		Entry("add memory dump request with no existing request",
			&v1.VirtualMachineMemoryDumpRequest{
				ClaimName: "vol1",
				Phase:     v1.MemoryDumpAssociating,
			},
			nil,
			patch.New(
				patch.WithTest("/status/memoryDumpRequest", nil),
				patch.WithAdd("/status/memoryDumpRequest", v1.VirtualMachineMemoryDumpRequest{
					ClaimName: "vol1",
					Phase:     v1.MemoryDumpAssociating,
				}),
			),
			false, false),
		Entry("add memory dump request to the same vol after completed",
			&v1.VirtualMachineMemoryDumpRequest{
				ClaimName: "vol1",
				Phase:     v1.MemoryDumpAssociating,
			},
			&v1.VirtualMachineMemoryDumpRequest{
				ClaimName: "vol1",
				Phase:     v1.MemoryDumpCompleted,
			},
			patch.New(
				patch.WithTest("/status/memoryDumpRequest", v1.VirtualMachineMemoryDumpRequest{
					ClaimName: "vol1",
					Phase:     v1.MemoryDumpCompleted,
				}),
				patch.WithReplace("/status/memoryDumpRequest", v1.VirtualMachineMemoryDumpRequest{
					ClaimName: "vol1",
					Phase:     v1.MemoryDumpAssociating,
				}),
			),
			false, false),
		Entry("add memory dump request to the same vol after previous failed",
			&v1.VirtualMachineMemoryDumpRequest{
				ClaimName: "vol1",
				Phase:     v1.MemoryDumpAssociating,
			},
			&v1.VirtualMachineMemoryDumpRequest{
				ClaimName: "vol1",
				Phase:     v1.MemoryDumpFailed,
			},
			patch.New(
				patch.WithTest("/status/memoryDumpRequest", v1.VirtualMachineMemoryDumpRequest{
					ClaimName: "vol1",
					Phase:     v1.MemoryDumpFailed,
				}),
				patch.WithReplace("/status/memoryDumpRequest", v1.VirtualMachineMemoryDumpRequest{
					ClaimName: "vol1",
					Phase:     v1.MemoryDumpAssociating,
				}),
			),
			false, false),
		Entry("add memory dump request to the same vol while memory dump in progress should fail",
			&v1.VirtualMachineMemoryDumpRequest{
				ClaimName: "vol1",
				Phase:     v1.MemoryDumpAssociating,
			},
			&v1.VirtualMachineMemoryDumpRequest{
				ClaimName: "vol1",
				Phase:     v1.MemoryDumpInProgress,
			},
			nil,
			true, false),
		Entry("add memory dump request to the same vol while it is being dissociated should fail",
			&v1.VirtualMachineMemoryDumpRequest{
				ClaimName: "vol1",
				Phase:     v1.MemoryDumpAssociating,
			},
			&v1.VirtualMachineMemoryDumpRequest{
				ClaimName: "vol1",
				Phase:     v1.MemoryDumpDissociating,
			},
			nil,
			true, false),
		Entry("remove memory dump request to already removed memory dump should fail",
			&v1.VirtualMachineMemoryDumpRequest{
				Phase:  v1.MemoryDumpDissociating,
				Remove: true,
			},
			nil,
			nil,
			true, true),
		Entry("remove memory dump request to memory dump in progress should succeed",
			&v1.VirtualMachineMemoryDumpRequest{
				Phase:  v1.MemoryDumpDissociating,
				Remove: true,
			},
			&v1.VirtualMachineMemoryDumpRequest{
				ClaimName: "vol1",
				Phase:     v1.MemoryDumpInProgress,
			},
			patch.New(
				patch.WithTest("/status/memoryDumpRequest", v1.VirtualMachineMemoryDumpRequest{
					ClaimName: "vol1",
					Phase:     v1.MemoryDumpInProgress,
				}),
				patch.WithReplace("/status/memoryDumpRequest", v1.VirtualMachineMemoryDumpRequest{
					ClaimName: "vol1",
					Phase:     v1.MemoryDumpDissociating,
					Remove:    true,
				}),
			),
			false, true),
		Entry("remove memory dump request with Remove request should fail",
			&v1.VirtualMachineMemoryDumpRequest{
				Phase:  v1.MemoryDumpDissociating,
				Remove: true,
			},
			&v1.VirtualMachineMemoryDumpRequest{
				ClaimName: "vol1",
				Phase:     v1.MemoryDumpDissociating,
				Remove:    true,
			},
			nil,
			true, true),
		Entry("remove memory dump request to completed memory dump should succeed",
			&v1.VirtualMachineMemoryDumpRequest{
				Phase:  v1.MemoryDumpDissociating,
				Remove: true,
			},
			&v1.VirtualMachineMemoryDumpRequest{
				ClaimName: "vol1",
				Phase:     v1.MemoryDumpCompleted,
			},
			patch.New(
				patch.WithTest("/status/memoryDumpRequest", v1.VirtualMachineMemoryDumpRequest{
					ClaimName: "vol1",
					Phase:     v1.MemoryDumpCompleted,
				}),
				patch.WithReplace("/status/memoryDumpRequest", v1.VirtualMachineMemoryDumpRequest{
					ClaimName: "vol1",
					Phase:     v1.MemoryDumpDissociating,
					Remove:    true,
				}),
			),
			false, true),
	)
})
