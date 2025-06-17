package dra

import (
	"errors"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"go.uber.org/mock/gomock"
	k8sv1 "k8s.io/api/core/v1"
	resourcev1beta1 "k8s.io/api/resource/v1beta1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	k8scache "k8s.io/client-go/tools/cache"
	"k8s.io/client-go/tools/record"
	"k8s.io/utils/ptr"
	virtv1 "kubevirt.io/api/core/v1"
	"kubevirt.io/client-go/kubecli"
	"kubevirt.io/client-go/log"

	"kubevirt.io/kubevirt/pkg/testutils"
	"kubevirt.io/kubevirt/pkg/util"
)

var _ = Describe("DRA Status Controller", func() {
	var ctrl *gomock.Controller
	var vmiClient *kubecli.MockVirtualMachineInstanceInterface
	var kubeClient *kubecli.MockKubevirtClient
	var pod *k8sv1.Pod

	BeforeEach(func() {
		ctrl = gomock.NewController(GinkgoT())
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
		var vmi *virtv1.VirtualMachineInstance
		var logger *log.FilteredLogger

		BeforeEach(func() {
			vmi = &virtv1.VirtualMachineInstance{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "testvmi",
					Namespace: "default",
				},
				Status: virtv1.VirtualMachineInstanceStatus{},
			}
			logger = log.Log.Object(vmi)
		})

		It("should return early if pod resource claim status is not filled", func() {
			draController := testDRAStatusController(kubeClient, vmi, pod, nil, nil)

			err := draController.updateStatus(logger, vmi, pod)
			Expect(err).ToNot(HaveOccurred())
		})

		It("should not update if GPU status hasn't changed", func() {
			vmi.Spec.Domain.Devices.GPUs = []virtv1.GPU{
				{
					Name: "gpu1",
					ClaimRequest: &virtv1.ClaimRequest{
						ClaimName:   ptr.To("gpu-claim"),
						RequestName: ptr.To("gpu-resource"),
					},
				},
			}
			vmi.Status.DeviceStatus = &virtv1.DeviceStatus{
				GPUStatuses: []virtv1.DeviceStatusInfo{
					{
						Name: "gpu1",
						DeviceResourceClaimStatus: &virtv1.DeviceResourceClaimStatus{
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
			vmi.Spec.Domain.Devices.GPUs = []virtv1.GPU{
				{
					Name: "gpu1",
					ClaimRequest: &virtv1.ClaimRequest{
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
			vmi.Spec.Domain.Devices.GPUs = []virtv1.GPU{
				{
					Name: "gpu1",
					ClaimRequest: &virtv1.ClaimRequest{
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
		var vmi *virtv1.VirtualMachineInstance

		BeforeEach(func() {
			vmi = &virtv1.VirtualMachineInstance{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "testvmi",
					Namespace: "default",
				},
				Spec: virtv1.VirtualMachineInstanceSpec{
					Domain: virtv1.DomainSpec{},
				},
				Status: virtv1.VirtualMachineInstanceStatus{},
			}
		})

		It("should return true if there are no GPUs with DRA", func() {
			status := &virtv1.DeviceStatus{}
			Expect(util.IsAllDRAGPUsReconciled(vmi, status)).To(BeTrue())
		})

		It("should return true if all GPUs with DRA are reconciled with PCI addresses", func() {
			vmi.Spec.Domain.Devices.GPUs = []virtv1.GPU{
				{
					Name: "gpu1",
					ClaimRequest: &virtv1.ClaimRequest{
						ClaimName:   ptr.To("claim1"),
						RequestName: ptr.To("request1"),
					},
				},
			}

			status := &virtv1.DeviceStatus{
				GPUStatuses: []virtv1.DeviceStatusInfo{
					{
						Name: "gpu1",
						DeviceResourceClaimStatus: &virtv1.DeviceResourceClaimStatus{
							ResourceClaimName: ptr.To("claim1"),
							Name:              ptr.To("device1"),
							Attributes: &virtv1.DeviceAttribute{
								PCIAddress: ptr.To("0000:00:01.0"),
							},
						},
					},
				},
			}

			Expect(util.IsAllDRAGPUsReconciled(vmi, status)).To(BeTrue())
		})

		It("should return true if all GPUs with DRA are reconciled with mdev UUIDs", func() {
			vmi.Spec.Domain.Devices.GPUs = []virtv1.GPU{
				{
					Name: "gpu1",
					ClaimRequest: &virtv1.ClaimRequest{
						ClaimName:   ptr.To("claim1"),
						RequestName: ptr.To("request1"),
					},
				},
			}

			status := &virtv1.DeviceStatus{
				GPUStatuses: []virtv1.DeviceStatusInfo{
					{
						Name: "gpu1",
						DeviceResourceClaimStatus: &virtv1.DeviceResourceClaimStatus{
							ResourceClaimName: ptr.To("claim1"),
							Name:              ptr.To("device1"),
							Attributes: &virtv1.DeviceAttribute{
								MDevUUID: ptr.To("mdev-uuid-123"),
							},
						},
					},
				},
			}

			Expect(util.IsAllDRAGPUsReconciled(vmi, status)).To(BeTrue())
		})

		It("should return false if any GPU with DRA is missing status", func() {
			vmi.Spec.Domain.Devices.GPUs = []virtv1.GPU{
				{
					Name: "gpu1",
					ClaimRequest: &virtv1.ClaimRequest{
						ClaimName:   ptr.To("claim1"),
						RequestName: ptr.To("request1"),
					},
				},
				{
					Name: "gpu2",
					ClaimRequest: &virtv1.ClaimRequest{
						ClaimName:   ptr.To("claim2"),
						RequestName: ptr.To("request2"),
					},
				},
			}

			status := &virtv1.DeviceStatus{
				GPUStatuses: []virtv1.DeviceStatusInfo{
					{
						Name: "gpu1",
						DeviceResourceClaimStatus: &virtv1.DeviceResourceClaimStatus{
							ResourceClaimName: ptr.To("claim1"),
							Name:              ptr.To("device1"),
							Attributes: &virtv1.DeviceAttribute{
								PCIAddress: ptr.To("0000:00:01.0"),
							},
						},
					},
					// Missing gpu2 status
				},
			}

			Expect(util.IsAllDRAGPUsReconciled(vmi, status)).To(BeFalse())
		})

		It("should return false if any GPU with DRA is missing attributes", func() {
			vmi.Spec.Domain.Devices.GPUs = []virtv1.GPU{
				{
					Name: "gpu1",
					ClaimRequest: &virtv1.ClaimRequest{
						ClaimName:   ptr.To("claim1"),
						RequestName: ptr.To("request1"),
					},
				},
			}

			status := &virtv1.DeviceStatus{
				GPUStatuses: []virtv1.DeviceStatusInfo{
					{
						Name: "gpu1",
						DeviceResourceClaimStatus: &virtv1.DeviceResourceClaimStatus{
							ResourceClaimName: ptr.To("claim1"),
							Name:              ptr.To("device1"),
							// Missing attributes
						},
					},
				},
			}

			Expect(util.IsAllDRAGPUsReconciled(vmi, status)).To(BeFalse())
		})

		It("should return false if any GPU with DRA has attributes but no PCI address or mdev UUID", func() {
			vmi.Spec.Domain.Devices.GPUs = []virtv1.GPU{
				{
					Name: "gpu1",
					ClaimRequest: &virtv1.ClaimRequest{
						ClaimName:   ptr.To("claim1"),
						RequestName: ptr.To("request1"),
					},
				},
			}

			status := &virtv1.DeviceStatus{
				GPUStatuses: []virtv1.DeviceStatusInfo{
					{
						Name: "gpu1",
						DeviceResourceClaimStatus: &virtv1.DeviceResourceClaimStatus{
							ResourceClaimName: ptr.To("claim1"),
							Name:              ptr.To("device1"),
							Attributes:        &virtv1.DeviceAttribute{},
						},
					},
				},
			}

			Expect(util.IsAllDRAGPUsReconciled(vmi, status)).To(BeFalse())
		})

		It("should handle nil status gracefully", func() {
			vmi.Spec.Domain.Devices.GPUs = []virtv1.GPU{
				{
					Name: "gpu1",
					ClaimRequest: &virtv1.ClaimRequest{
						ClaimName:   ptr.To("claim1"),
						RequestName: ptr.To("request1"),
					},
				},
			}

			Expect(util.IsAllDRAGPUsReconciled(vmi, nil)).To(BeFalse())
		})
	})
})

func testDRAStatusController(kubeClient kubecli.KubevirtClient, vmi *virtv1.VirtualMachineInstance, pod *k8sv1.Pod, resourceClaim *resourcev1beta1.ResourceClaim, resourceSlice *resourcev1beta1.ResourceSlice) *DRAStatusController {
	vmiInformer, _ := testutils.NewFakeInformerFor(&virtv1.VirtualMachineInstance{})
	podInformer, _ := testutils.NewFakeInformerFor(&k8sv1.Pod{})
	resourceClaimInformer, _ := testutils.NewFakeInformerFor(&resourcev1beta1.ResourceClaim{})
	resourceSliceInformer, _ := testutils.NewFakeInformerFor(&resourcev1beta1.ResourceSlice{})
	recorder := record.NewFakeRecorder(100)

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
		vmiInformer:          vmiInformer,
		podInformer:          podInformer,
		recorder:             recorder,
		podIndexer:           podInformer.GetIndexer(),
		vmiIndexer:           vmiInformer.GetIndexer(),
		resourceClaimIndexer: resourceClaimInformer.GetIndexer(),
		resourceSliceIndexer: resourceSliceInformer.GetIndexer(),
	}
}

func getTestResourceClaim(name, namespace, requestName, deviceName, driverName string) *resourcev1beta1.ResourceClaim {
	resourceClaim := &resourcev1beta1.ResourceClaim{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
		},
		Status: resourcev1beta1.ResourceClaimStatus{
			Allocation: &resourcev1beta1.AllocationResult{},
		},
	}

	expectedDeviceResult := resourcev1beta1.DeviceRequestAllocationResult{
		Request: requestName,
		Device:  deviceName,
		Driver:  driverName,
		Pool:    "default",
	}

	resourceClaim.Status.Allocation.Devices = resourcev1beta1.DeviceAllocationResult{
		Results: []resourcev1beta1.DeviceRequestAllocationResult{expectedDeviceResult},
	}

	return resourceClaim
}

func getTestResourceSlice(name, nodeName, deviceName, driverName string) *resourcev1beta1.ResourceSlice {
	pciAddress := "0000:00:01.0"
	mdevUUID := "mdev-uuid-123"

	return &resourcev1beta1.ResourceSlice{
		ObjectMeta: metav1.ObjectMeta{
			Name: name,
		},
		Spec: resourcev1beta1.ResourceSliceSpec{
			NodeName: nodeName,
			Driver:   driverName,
			Pool: resourcev1beta1.ResourcePool{
				Name:               "default",
				Generation:         1,
				ResourceSliceCount: 1,
			},
			Devices: []resourcev1beta1.Device{
				{
					Name: deviceName,
					Basic: &resourcev1beta1.BasicDevice{
						Attributes: map[resourcev1beta1.QualifiedName]resourcev1beta1.DeviceAttribute{
							"pciAddress": {
								StringValue: ptr.To(pciAddress),
							},
							"uuid": {
								StringValue: ptr.To(mdevUUID),
							},
						},
					},
				},
			},
		},
	}
}
