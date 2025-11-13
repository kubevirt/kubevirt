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

package dra

import (
	"errors"

	"kubevirt.io/kubevirt/pkg/dra"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"go.uber.org/mock/gomock"
	k8sv1 "k8s.io/api/core/v1"
	resourcev1beta2 "k8s.io/api/resource/v1beta2"
	extv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	k8scache "k8s.io/client-go/tools/cache"
	"k8s.io/client-go/tools/record"
	"k8s.io/utils/ptr"
	v1 "kubevirt.io/api/core/v1"
	"kubevirt.io/client-go/kubecli"
	"kubevirt.io/client-go/log"

	"kubevirt.io/kubevirt/pkg/testutils"
	virtconfig "kubevirt.io/kubevirt/pkg/virt-config"
	"kubevirt.io/kubevirt/pkg/virt-config/featuregate"
)

var _ = Describe("DRA Status Controller", func() {
	var vmiClient *kubecli.MockVirtualMachineInstanceInterface
	var kubeClient *kubecli.MockKubevirtClient
	var pod *k8sv1.Pod

	BeforeEach(func() {
		ctrl := gomock.NewController(GinkgoT())
		vmiClient = kubecli.NewMockVirtualMachineInstanceInterface(ctrl)
		kubeClient = kubecli.NewMockKubevirtClient(ctrl)

		pod = &k8sv1.Pod{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "testpod",
				Namespace: "default",
			},
		}

	})

	Context("updateStatus", func() {
		var vmi *v1.VirtualMachineInstance
		var logger *log.FilteredLogger

		BeforeEach(func() {
			vmi = &v1.VirtualMachineInstance{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "testvmi",
					Namespace: "default",
				},
				Status: v1.VirtualMachineInstanceStatus{},
			}
			logger = log.Log.Object(vmi)
		})

		It("should return early if pod resource claim status is not filled", func() {
			draController := testDRAStatusController(kubeClient, vmi, pod, nil, nil)

			err := draController.updateStatus(logger, vmi, pod)
			Expect(err).ToNot(HaveOccurred())
		})

		It("should not update if GPU status hasn't changed", func() {
			vmi.Spec.Domain.Devices.GPUs = []v1.GPU{
				{
					Name: "gpu1",
					ClaimRequest: &v1.ClaimRequest{
						ClaimName:   ptr.To("gpu-claim"),
						RequestName: ptr.To("gpu-resource"),
					},
				},
			}
			vmi.Status.DeviceStatus = &v1.DeviceStatus{
				GPUStatuses: []v1.DeviceStatusInfo{
					{
						Name: "gpu1",
						DeviceResourceClaimStatus: &v1.DeviceResourceClaimStatus{
							ResourceClaimName: ptr.To("claim1"),
							Name:              ptr.To("device1"),
						},
					},
				},
			}

			pod.Spec.ResourceClaims = []k8sv1.PodResourceClaim{
				{
					Name:              "claim1",
					ResourceClaimName: ptr.To("claim1"),
				},
			}

			draController := testDRAStatusController(kubeClient, vmi, pod, nil, nil)

			err := draController.updateStatus(logger, vmi, pod)
			Expect(err).ToNot(HaveOccurred())
		})

		It("should update VMI status when GPU status has changed", func() {
			vmi.Spec.Domain.Devices.GPUs = []v1.GPU{
				{
					Name: "gpu1",
					ClaimRequest: &v1.ClaimRequest{
						ClaimName:   ptr.To("claim1"),
						RequestName: ptr.To("request1"),
					},
				},
			}
			vmi.Status.DeviceStatus = nil

			pod.Spec.NodeName = "testnode"
			pod.Spec.ResourceClaims = []k8sv1.PodResourceClaim{
				{
					Name:              "claim1",
					ResourceClaimName: ptr.To("claim1"),
				},
			}
			pod.Status.ResourceClaimStatuses = []k8sv1.PodResourceClaimStatus{
				{
					Name:              "claim1",
					ResourceClaimName: ptr.To("claim1"),
				},
			}

			vmiClient.EXPECT().
				Patch(gomock.Any(), "testvmi", gomock.Any(), gomock.Any(), gomock.Any()).
				Return(vmi, nil)

			kubeClient.EXPECT().VirtualMachineInstance(gomock.Any()).Return(vmiClient).MaxTimes(1)

			draController := testDRAStatusController(kubeClient, vmi, pod,
				getTestResourceClaim("claim1", "default", "request1", "device1", "driver1"),
				getTestResourceSlice("resourceslice1", "testnode", "device1", "driver1"))
			err := draController.updateStatus(logger, vmi, pod)
			Expect(err).ToNot(HaveOccurred())
		})

		It("should handle patch errors when updating VMI status", func() {
			vmi.Spec.Domain.Devices.GPUs = []v1.GPU{
				{
					Name: "gpu1",
					ClaimRequest: &v1.ClaimRequest{
						ClaimName:   ptr.To("claim1"),
						RequestName: ptr.To("request1"),
					},
				},
			}
			vmi.Status.DeviceStatus = nil

			pod.Spec.NodeName = "testnode"
			pod.Spec.ResourceClaims = []k8sv1.PodResourceClaim{
				{
					Name:              "claim1",
					ResourceClaimName: ptr.To("claim1"),
				},
			}
			pod.Status.ResourceClaimStatuses = []k8sv1.PodResourceClaimStatus{
				{
					Name:              "claim1",
					ResourceClaimName: ptr.To("claim1"),
				},
			}

			expectedErr := errors.New("patch failed")
			vmiClient.EXPECT().
				Patch(gomock.Any(), "testvmi", gomock.Any(), gomock.Any(), gomock.Any()).
				Return(nil, expectedErr)

			kubeClient.EXPECT().VirtualMachineInstance(gomock.Any()).Return(vmiClient).MaxTimes(1)

			draController := testDRAStatusController(kubeClient, vmi, pod,
				getTestResourceClaim("claim1", "default", "request1", "device1", "driver1"),
				getTestResourceSlice("resourceslice1", "testnode", "device1", "driver1"))

			err := draController.updateStatus(logger, vmi, pod)
			Expect(err).To(HaveOccurred())
			Expect(err).To(Equal(expectedErr))
		})
	})

	Context("isAllDRAGPUsReconciled", func() {
		var vmi *v1.VirtualMachineInstance

		BeforeEach(func() {
			vmi = &v1.VirtualMachineInstance{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "testvmi",
					Namespace: "default",
				},
				Spec: v1.VirtualMachineInstanceSpec{
					Domain: v1.DomainSpec{},
				},
				Status: v1.VirtualMachineInstanceStatus{},
			}
		})

		It("should return true if there are no GPUs with DRA", func() {
			status := &v1.DeviceStatus{}
			Expect(dra.IsAllDRAGPUsReconciled(vmi, status)).To(BeTrue())
		})

		It("should return true if all GPUs with DRA are reconciled with PCI addresses", func() {
			vmi.Spec.Domain.Devices.GPUs = []v1.GPU{
				{
					Name: "gpu1",
					ClaimRequest: &v1.ClaimRequest{
						ClaimName:   ptr.To("claim1"),
						RequestName: ptr.To("request1"),
					},
				},
			}

			status := &v1.DeviceStatus{
				GPUStatuses: []v1.DeviceStatusInfo{
					{
						Name: "gpu1",
						DeviceResourceClaimStatus: &v1.DeviceResourceClaimStatus{
							ResourceClaimName: ptr.To("claim1"),
							Name:              ptr.To("device1"),
							Attributes: &v1.DeviceAttribute{
								PCIAddress: ptr.To("0000:00:01.0"),
							},
						},
					},
				},
			}

			Expect(dra.IsAllDRAGPUsReconciled(vmi, status)).To(BeTrue())
		})

		It("should return true if all GPUs with DRA are reconciled with mdev UUIDs", func() {
			vmi.Spec.Domain.Devices.GPUs = []v1.GPU{
				{
					Name: "gpu1",
					ClaimRequest: &v1.ClaimRequest{
						ClaimName:   ptr.To("claim1"),
						RequestName: ptr.To("request1"),
					},
				},
			}

			status := &v1.DeviceStatus{
				GPUStatuses: []v1.DeviceStatusInfo{
					{
						Name: "gpu1",
						DeviceResourceClaimStatus: &v1.DeviceResourceClaimStatus{
							ResourceClaimName: ptr.To("claim1"),
							Name:              ptr.To("device1"),
							Attributes: &v1.DeviceAttribute{
								MDevUUID: ptr.To("mdev-uuid-123"),
							},
						},
					},
				},
			}

			Expect(dra.IsAllDRAGPUsReconciled(vmi, status)).To(BeTrue())
		})

		It("should return false if any GPU with DRA is missing status", func() {
			vmi.Spec.Domain.Devices.GPUs = []v1.GPU{
				{
					Name: "gpu1",
					ClaimRequest: &v1.ClaimRequest{
						ClaimName:   ptr.To("claim1"),
						RequestName: ptr.To("request1"),
					},
				},
				{
					Name: "gpu2",
					ClaimRequest: &v1.ClaimRequest{
						ClaimName:   ptr.To("claim2"),
						RequestName: ptr.To("request2"),
					},
				},
			}

			status := &v1.DeviceStatus{
				GPUStatuses: []v1.DeviceStatusInfo{
					{
						Name: "gpu1",
						DeviceResourceClaimStatus: &v1.DeviceResourceClaimStatus{
							ResourceClaimName: ptr.To("claim1"),
							Name:              ptr.To("device1"),
							Attributes: &v1.DeviceAttribute{
								PCIAddress: ptr.To("0000:00:01.0"),
							},
						},
					},
					// Missing gpu2 status
				},
			}

			Expect(dra.IsAllDRAGPUsReconciled(vmi, status)).To(BeFalse())
		})

		It("should return false if any GPU with DRA is missing attributes", func() {
			vmi.Spec.Domain.Devices.GPUs = []v1.GPU{
				{
					Name: "gpu1",
					ClaimRequest: &v1.ClaimRequest{
						ClaimName:   ptr.To("claim1"),
						RequestName: ptr.To("request1"),
					},
				},
			}

			status := &v1.DeviceStatus{
				GPUStatuses: []v1.DeviceStatusInfo{
					{
						Name: "gpu1",
						DeviceResourceClaimStatus: &v1.DeviceResourceClaimStatus{
							ResourceClaimName: ptr.To("claim1"),
							Name:              ptr.To("device1"),
							// Missing attributes
						},
					},
				},
			}

			Expect(dra.IsAllDRAGPUsReconciled(vmi, status)).To(BeFalse())
		})

		It("should return false if any GPU with DRA has attributes but no PCI address or mdev UUID", func() {
			vmi.Spec.Domain.Devices.GPUs = []v1.GPU{
				{
					Name: "gpu1",
					ClaimRequest: &v1.ClaimRequest{
						ClaimName:   ptr.To("claim1"),
						RequestName: ptr.To("request1"),
					},
				},
			}

			status := &v1.DeviceStatus{
				GPUStatuses: []v1.DeviceStatusInfo{
					{
						Name: "gpu1",
						DeviceResourceClaimStatus: &v1.DeviceResourceClaimStatus{
							ResourceClaimName: ptr.To("claim1"),
							Name:              ptr.To("device1"),
							Attributes:        &v1.DeviceAttribute{},
						},
					},
				},
			}

			Expect(dra.IsAllDRAGPUsReconciled(vmi, status)).To(BeFalse())
		})

		It("should handle nil status gracefully", func() {
			vmi.Spec.Domain.Devices.GPUs = []v1.GPU{
				{
					Name: "gpu1",
					ClaimRequest: &v1.ClaimRequest{
						ClaimName:   ptr.To("claim1"),
						RequestName: ptr.To("request1"),
					},
				},
			}

			Expect(dra.IsAllDRAGPUsReconciled(vmi, nil)).To(BeFalse())
		})
	})
})

func testDRAStatusController(kubeClient kubecli.KubevirtClient, vmi *v1.VirtualMachineInstance, pod *k8sv1.Pod, resourceClaim *resourcev1beta2.ResourceClaim, resourceSlice *resourcev1beta2.ResourceSlice) *DRAStatusController {
	vmiInformer, _ := testutils.NewFakeInformerFor(&v1.VirtualMachineInstance{})
	podInformer, _ := testutils.NewFakeInformerFor(&k8sv1.Pod{})
	resourceClaimInformer, _ := testutils.NewFakeInformerFor(&resourcev1beta2.ResourceClaim{})
	resourceSliceInformer, _ := testutils.NewFakeInformerFor(&resourcev1beta2.ResourceSlice{})
	recorder := record.NewFakeRecorder(100)

	clusterConfig := newTestClusterConfigWithGPUDRA()

	if vmi != nil {
		vmiInformer.GetIndexer().Add(vmi)
	}
	if pod != nil {
		podInformer.GetIndexer().Add(pod)
	}
	if resourceClaim != nil {
		resourceClaimInformer.GetIndexer().Add(resourceClaim)
	}
	if resourceSlice != nil {
		resourceSliceInformer.GetIndexer().Add(resourceSlice)
		_ = resourceSliceInformer.GetIndexer().AddIndexers(map[string]k8scache.IndexFunc{
			indexByNodeName: indexResourceSliceByNodeName,
		})
	}

	return &DRAStatusController{
		clientset:            kubeClient,
		recorder:             recorder,
		podIndexer:           podInformer.GetIndexer(),
		vmiIndexer:           vmiInformer.GetIndexer(),
		resourceClaimIndexer: resourceClaimInformer.GetIndexer(),
		resourceSliceIndexer: resourceSliceInformer.GetIndexer(),
		clusterConfig:        clusterConfig,
	}
}

// newTestClusterConfigWithGPUDRA returns a minimal ClusterConfig suitable for unit testing.
// It reports the GPUsWithDRA feature-gate as enabled while every other feature-gate remains disabled.
func newTestClusterConfigWithGPUDRA() *virtconfig.ClusterConfig {
	// Create empty (fake) informers because the real ClusterConfig constructor requires them.
	crdInformer, _ := testutils.NewFakeInformerFor(&extv1.CustomResourceDefinition{})
	kvInformer, _ := testutils.NewFakeInformerFor(&v1.KubeVirt{})

	cc, _ := virtconfig.NewClusterConfig(crdInformer, kvInformer, "default")

	// Enable the GPUsWithDRA feature-gate on the underlying configuration returned by GetConfig().
	cfg := cc.GetConfig()
	cfg.DeveloperConfiguration.FeatureGates = append(cfg.DeveloperConfiguration.FeatureGates, featuregate.GPUsWithDRAGate)

	return cc
}

func getTestResourceClaim(name, namespace, requestName, deviceName, driverName string) *resourcev1beta2.ResourceClaim {
	resourceClaim := &resourcev1beta2.ResourceClaim{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
		},
		Status: resourcev1beta2.ResourceClaimStatus{
			Allocation: &resourcev1beta2.AllocationResult{},
		},
	}

	expectedDeviceResult := resourcev1beta2.DeviceRequestAllocationResult{
		Request: requestName,
		Device:  deviceName,
		Driver:  driverName,
		Pool:    "default",
	}

	resourceClaim.Status.Allocation.Devices = resourcev1beta2.DeviceAllocationResult{
		Results: []resourcev1beta2.DeviceRequestAllocationResult{expectedDeviceResult},
	}

	return resourceClaim
}

func getTestResourceSlice(name, nodeName, deviceName, driverName string) *resourcev1beta2.ResourceSlice {
	pciAddress := "0000:00:01.0"
	mdevUUID := "mdev-uuid-123"

	return &resourcev1beta2.ResourceSlice{
		ObjectMeta: metav1.ObjectMeta{
			Name: name,
		},
		Spec: resourcev1beta2.ResourceSliceSpec{
			NodeName: ptr.To(nodeName),
			Driver:   driverName,
			Pool: resourcev1beta2.ResourcePool{
				Name:               "default",
				Generation:         1,
				ResourceSliceCount: 1,
			},
			Devices: []resourcev1beta2.Device{
				{
					Name: deviceName,
					Attributes: map[resourcev1beta2.QualifiedName]resourcev1beta2.DeviceAttribute{
						PCIAddressDeviceAttributeKey: {
							StringValue: ptr.To(pciAddress),
						},
						MDevUUIDDeviceAttributeKey: {
							StringValue: ptr.To(mdevUUID),
						},
					},
				},
			},
		},
	}
}
