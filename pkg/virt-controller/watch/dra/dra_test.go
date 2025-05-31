package dra

import (
	"errors"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"go.uber.org/mock/gomock"
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

		It("should set DRAStatusAttributesAvailable condition when all GPU DRA devices are reconciled", func() {
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

			// Simulate that the first patch has been applied and now status has been updated
			vmi.Status.DeviceStatus = &virtv1.DeviceStatus{
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

			// The second patch should add the virtv1.VirtualMachineInstanceDRAStatusAttributesAvailable condition
			patchedVmi := vmi.DeepCopy()
			condition := virtv1.VirtualMachineInstanceCondition{
				Type:               virtv1.VirtualMachineInstanceDRAStatusAttributesAvailable,
				Status:             k8sv1.ConditionTrue,
				LastTransitionTime: metav1.Now(),
				Reason:             virtv1.VirtualMachineInstanceReasonDRADevicesAllocated,
				Message:            draAttributesAvailableMessage,
			}
			patchedVmi.Status.Conditions = append(patchedVmi.Status.Conditions, condition)

			vmiClient.EXPECT().
				Patch(gomock.Any(), "testvmi", gomock.Any(), gomock.Any(), gomock.Any()).
				Do(func(ctx interface{}, name string, pt interface{}, data interface{}, opts interface{}) {
					Expect(data).To(ContainSubstring(string(virtv1.VirtualMachineInstanceDRAStatusAttributesAvailable)))
				}).
				Return(patchedVmi, nil)

			kubeClient.EXPECT().VirtualMachineInstance(gomock.Any()).Return(vmiClient).MaxTimes(1)

			// Call updateStatus again, now it should add the condition
			err = draController.updateStatus(logger, vmi, pod)
			Expect(err).ToNot(HaveOccurred())
		})

		It("should not set virtv1.VirtualMachineInstanceDRAStatusAttributesAvailable condition when only some GPU DRA devices are reconciled", func() {
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
			vmi.Status.DeviceStatus = nil

			pod.Spec.NodeName = "testnode"
			pod.Spec.ResourceClaims = []k8sv1.PodResourceClaim{
				{
					Name:              "claim1",
					ResourceClaimName: ptr.To("claim1"),
				},
				{
					Name:              "claim2",
					ResourceClaimName: ptr.To("claim2"),
				},
			}
			pod.Status.ResourceClaimStatuses = []k8sv1.PodResourceClaimStatus{
				{
					Name:              "claim1",
					ResourceClaimName: ptr.To("claim1"),
				},
				{
					Name:              "claim2",
					ResourceClaimName: ptr.To("claim2"),
				},
			}

			vmiClient.EXPECT().
				Patch(gomock.Any(), "testvmi", gomock.Any(), gomock.Any(), gomock.Any()).
				Return(vmi, nil)

			kubeClient.EXPECT().VirtualMachineInstance(gomock.Any()).Return(vmiClient).MaxTimes(1)

			draController := testDRAStatusController(kubeClient, vmi, pod,
				getTestResourceClaim("claim1", "default", "request1", "device1", "driver1"),
				getTestResourceSlice("resourceslice1", "testnode", "device1", "driver1"))

			// ResourceClaim for the second GPU is missing, so not all GPUs can be reconciled
			err := draController.updateStatus(logger, vmi, pod)
			Expect(err).ToNot(HaveOccurred())

			// Simulate that the first patch has been applied, but only one GPU has status
			vmi.Status.DeviceStatus = &virtv1.DeviceStatus{
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
					// Missing status for gpu2
				},
			}

			// No second patch should happen since not all GPUs are reconciled
			// The existing device status shouldn't be modified since it contains partial information
			err = draController.updateStatus(logger, vmi, pod)
			Expect(err).ToNot(HaveOccurred())

			// Verify that condition wasn't set
			found := false
			for _, condition := range vmi.Status.Conditions {
				if condition.Type == virtv1.VirtualMachineInstanceDRAStatusAttributesAvailable {
					found = true
					break
				}
			}
			Expect(found).To(BeFalse(), virtv1.VirtualMachineInstanceDRAStatusAttributesAvailable+" condition should "+
				"not be set when not all GPUs are reconciled")
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
			Expect(isAllDRAGPUsReconciled(vmi, status)).To(BeTrue())
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

			Expect(isAllDRAGPUsReconciled(vmi, status)).To(BeTrue())
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

			Expect(isAllDRAGPUsReconciled(vmi, status)).To(BeTrue())
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

			Expect(isAllDRAGPUsReconciled(vmi, status)).To(BeFalse())
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

			Expect(isAllDRAGPUsReconciled(vmi, status)).To(BeFalse())
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

			Expect(isAllDRAGPUsReconciled(vmi, status)).To(BeFalse())
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

			Expect(isAllDRAGPUsReconciled(vmi, nil)).To(BeFalse())
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
