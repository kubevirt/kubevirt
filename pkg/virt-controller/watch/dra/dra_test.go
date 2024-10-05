package dra

import (
	"errors"

	"github.com/golang/mock/gomock"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	k8sv1 "k8s.io/api/core/v1"
	"k8s.io/api/resource/v1alpha3"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/tools/record"
	"k8s.io/utils/ptr"
	virtv1 "kubevirt.io/api/core/v1"
	"kubevirt.io/client-go/kubecli"
	"kubevirt.io/client-go/log"
	"kubevirt.io/kubevirt/pkg/testutils"
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
})

func testDRAStatusController(kubeClient kubecli.KubevirtClient, vmi *virtv1.VirtualMachineInstance, pod *k8sv1.Pod, resourceClaim *v1alpha3.ResourceClaim, resourceSlice *v1alpha3.ResourceSlice) *DRAStatusController {
	vmiInformer, _ := testutils.NewFakeInformerFor(&virtv1.VirtualMachineInstance{})
	podInformer, _ := testutils.NewFakeInformerFor(&k8sv1.Pod{})
	resourceClaimInformer, _ := testutils.NewFakeInformerFor(&v1alpha3.ResourceClaim{})
	resourceSliceInformer, _ := testutils.NewFakeInformerFor(&v1alpha3.ResourceSlice{})
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

func getTestResourceClaim(name, namespace, requestName, deviceName, driverName string) *v1alpha3.ResourceClaim {
	resourceClaim := &v1alpha3.ResourceClaim{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
		},
		Status: v1alpha3.ResourceClaimStatus{
			Allocation: &v1alpha3.AllocationResult{},
		},
	}

	expectedDeviceResult := v1alpha3.DeviceRequestAllocationResult{
		Request: requestName,
		Device:  deviceName,
		Driver:  driverName,
		Pool:    "default",
	}

	resourceClaim.Status.Allocation.Devices = v1alpha3.DeviceAllocationResult{
		Results: []v1alpha3.DeviceRequestAllocationResult{expectedDeviceResult},
	}

	return resourceClaim
}

func getTestResourceSlice(name, nodeName, deviceName, driverName string) *v1alpha3.ResourceSlice {
	pciAddress := "0000:00:01.0"
	mdevUUID := "mdev-uuid-123"

	return &v1alpha3.ResourceSlice{
		ObjectMeta: metav1.ObjectMeta{
			Name: name,
		},
		Spec: v1alpha3.ResourceSliceSpec{
			NodeName: nodeName,
			Driver:   driverName,
			Pool: v1alpha3.ResourcePool{
				Name:               "default",
				Generation:         1,
				ResourceSliceCount: 1,
			},
			Devices: []v1alpha3.Device{
				{
					Name: deviceName,
					Basic: &v1alpha3.BasicDevice{
						Attributes: map[v1alpha3.QualifiedName]v1alpha3.DeviceAttribute{
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
