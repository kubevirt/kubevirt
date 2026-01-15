package dra

/*
These unit tests verify the behaviour of CreateDRAGPUHostDevices,
which converts the DRA-related information stored in KEP-5304 metadata files
into libvirt HostDevice definitions that virt-launcher will add to the
libvirt domain.
*/

import (
	"encoding/json"
	"os"
	"path/filepath"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	k8sv1 "k8s.io/api/core/v1"
	resourcev1 "k8s.io/api/resource/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/utils/ptr"

	v1 "kubevirt.io/api/core/v1"

	drautil "kubevirt.io/kubevirt/pkg/dra"
	"kubevirt.io/kubevirt/pkg/dra/metadata"
	"kubevirt.io/kubevirt/pkg/virt-launcher/virtwrap/api"
)

var _ = Describe("CreateDRAGPUHostDevices", func() {
	var (
		tempDir string
	)

	BeforeEach(func() {
		var err error
		tempDir, err = os.MkdirTemp("", "dra-gpu-test")
		Expect(err).ToNot(HaveOccurred())
	})

	AfterEach(func() {
		os.RemoveAll(tempDir)
	})

	// KEP-5304 path: {base}/{claimName}/{requestName}/{driver}-metadata.json
	createMetadataFile := func(claimName, requestName, driver string, md *metadata.DeviceMetadata) {
		dir := filepath.Join(tempDir, claimName, requestName)
		Expect(os.MkdirAll(dir, 0755)).To(Succeed())

		data, err := json.Marshal(md)
		Expect(err).ToNot(HaveOccurred())

		Expect(os.WriteFile(filepath.Join(dir, driver+"-metadata.json"), data, 0644)).To(Succeed())
	}

	Context("when the VMI has no GPUs with DRA", func() {
		It("should return an empty slice without error", func() {
			vmi := &v1.VirtualMachineInstance{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "testvmi",
					Namespace: "default",
				},
				Spec: v1.VirtualMachineInstanceSpec{
					Domain: v1.DomainSpec{},
				},
			}

			downwardAPIAttributes, err := drautil.NewDownwardAPIAttributesWithBasePath(tempDir, nil)
			Expect(err).ToNot(HaveOccurred())

			hostDevices, err := CreateDRAGPUHostDevices(vmi, downwardAPIAttributes)
			Expect(err).ToNot(HaveOccurred())
			Expect(hostDevices).To(BeEmpty())
		})
	})

	Context("when the VMI has a physical GPU (PCI) allocated through DRA", func() {
		It("should create exactly one PCI host device", func() {
			pciAddr := "0000:02:00.0"

			createMetadataFile("claim1", "req1", "gpu.example.com", &metadata.DeviceMetadata{
				TypeMeta: metav1.TypeMeta{
					Kind:       "DeviceMetadata",
					APIVersion: "v1alpha1",
				},
				ObjectMeta: metav1.ObjectMeta{
					Name: "claim1",
				},
				Requests: []metadata.DeviceMetadataRequest{{
					Name: "req1",
					Devices: []metadata.Device{{
						Driver: "gpu.example.com",
						Pool:   "gpu-pool",
						Name:   "device1",
						Attributes: map[resourcev1.QualifiedName]resourcev1.DeviceAttribute{
							metadata.PCIBusIDAttribute: {StringValue: &pciAddr},
						},
					}},
				}},
			})

			vmi := &v1.VirtualMachineInstance{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "testvmi",
					Namespace: "default",
				},
				Spec: v1.VirtualMachineInstanceSpec{
					ResourceClaims: []k8sv1.PodResourceClaim{{
						Name:              "claim1",
						ResourceClaimName: ptr.To("claim1"),
					}},
					Domain: v1.DomainSpec{
						Devices: v1.Devices{
							GPUs: []v1.GPU{{
								Name: "gpu1",
								ClaimRequest: &v1.ClaimRequest{
									ClaimName:   ptr.To("claim1"),
									RequestName: ptr.To("req1"),
								},
							}},
						},
					},
				},
			}

			downwardAPIAttributes, err := drautil.NewDownwardAPIAttributesWithBasePath(tempDir, vmi.Spec.ResourceClaims)
			Expect(err).ToNot(HaveOccurred())

			hostDevices, err := CreateDRAGPUHostDevices(vmi, downwardAPIAttributes)
			Expect(err).ToNot(HaveOccurred())
			Expect(hostDevices).To(HaveLen(1))

			dev := hostDevices[0]
			Expect(dev.Type).To(Equal(api.HostDevicePCI))
			Expect(dev.Managed).To(Equal("no"))
			Expect(dev.Alias).ToNot(BeNil())
			Expect(dev.Alias.GetName()).To(Equal(AliasPrefix + "gpu1"))
			Expect(dev.Source.Address).ToNot(BeNil())
			Expect(dev.Source.Address.Type).To(Equal(api.AddressPCI))
		})
	})

	Context("when the VMI has a virtual GPU (mdev) allocated through DRA", func() {
		It("should create exactly one mdev host device with display enabled", func() {
			uuid := "123e4567-e89b-12d3-a456-426614174000"

			createMetadataFile("claim1", "req1", "gpu.example.com", &metadata.DeviceMetadata{
				TypeMeta: metav1.TypeMeta{
					Kind:       "DeviceMetadata",
					APIVersion: "v1alpha1",
				},
				ObjectMeta: metav1.ObjectMeta{
					Name: "claim1",
				},
				Requests: []metadata.DeviceMetadataRequest{{
					Name: "req1",
					Devices: []metadata.Device{{
						Driver: "gpu.example.com",
						Pool:   "gpu-pool",
						Name:   "device1",
						Attributes: map[resourcev1.QualifiedName]resourcev1.DeviceAttribute{
							metadata.MDevUUIDAttribute: {StringValue: &uuid},
						},
					}},
				}},
			})

			vmi := &v1.VirtualMachineInstance{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "testvmi",
					Namespace: "default",
				},
				Spec: v1.VirtualMachineInstanceSpec{
					ResourceClaims: []k8sv1.PodResourceClaim{{
						Name:              "claim1",
						ResourceClaimName: ptr.To("claim1"),
					}},
					Domain: v1.DomainSpec{
						Devices: v1.Devices{
							GPUs: []v1.GPU{{
								Name: "vgpu1",
								ClaimRequest: &v1.ClaimRequest{
									ClaimName:   ptr.To("claim1"),
									RequestName: ptr.To("req1"),
								},
							}},
						},
					},
				},
			}

			downwardAPIAttributes, err := drautil.NewDownwardAPIAttributesWithBasePath(tempDir, vmi.Spec.ResourceClaims)
			Expect(err).ToNot(HaveOccurred())

			hostDevices, err := CreateDRAGPUHostDevices(vmi, downwardAPIAttributes)
			Expect(err).ToNot(HaveOccurred())
			Expect(hostDevices).To(HaveLen(1))

			dev := hostDevices[0]
			Expect(dev.Type).To(Equal(api.HostDeviceMDev))
			Expect(dev.Display).To(Equal("on"))
			Expect(dev.RamFB).To(Equal("on"))
			Expect(dev.Alias.GetName()).To(Equal(AliasPrefix + "vgpu1"))
			Expect(dev.Source.Address).ToNot(BeNil())
			Expect(dev.Source.Address.UUID).To(Equal(uuid))
		})
	})

	Context("when the device has both pciBusID and mdevUUID", func() {
		It("should prefer mdevUUID and create an mdev host device", func() {
			pciAddr := "0000:01:01.0"
			mdevUUID := "abcd1234-e89b-12d3-a456-426614174000"

			createMetadataFile("claim1", "req1", "gpu.example.com", &metadata.DeviceMetadata{
				TypeMeta: metav1.TypeMeta{
					Kind:       "DeviceMetadata",
					APIVersion: "v1alpha1",
				},
				ObjectMeta: metav1.ObjectMeta{
					Name: "claim1",
				},
				Requests: []metadata.DeviceMetadataRequest{{
					Name: "req1",
					Devices: []metadata.Device{{
						Driver: "gpu.example.com",
						Pool:   "gpu-pool",
						Name:   "vgpu-device",
						Attributes: map[resourcev1.QualifiedName]resourcev1.DeviceAttribute{
							metadata.PCIBusIDAttribute: {StringValue: &pciAddr},
							metadata.MDevUUIDAttribute: {StringValue: &mdevUUID},
						},
					}},
				}},
			})

			vmi := &v1.VirtualMachineInstance{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "testvmi",
					Namespace: "default",
				},
				Spec: v1.VirtualMachineInstanceSpec{
					ResourceClaims: []k8sv1.PodResourceClaim{{
						Name:              "claim1",
						ResourceClaimName: ptr.To("claim1"),
					}},
					Domain: v1.DomainSpec{
						Devices: v1.Devices{
							GPUs: []v1.GPU{{
								Name: "vgpu1",
								ClaimRequest: &v1.ClaimRequest{
									ClaimName:   ptr.To("claim1"),
									RequestName: ptr.To("req1"),
								},
							}},
						},
					},
				},
			}

			downwardAPIAttributes, err := drautil.NewDownwardAPIAttributesWithBasePath(tempDir, vmi.Spec.ResourceClaims)
			Expect(err).ToNot(HaveOccurred())

			hostDevices, err := CreateDRAGPUHostDevices(vmi, downwardAPIAttributes)
			Expect(err).ToNot(HaveOccurred())
			Expect(hostDevices).To(HaveLen(1))

			dev := hostDevices[0]
			Expect(dev.Type).To(Equal(api.HostDeviceMDev), "device with both pciBusID and mdevUUID should be treated as mdev")
			Expect(dev.Source.Address.UUID).To(Equal(mdevUUID))
			Expect(dev.Alias.GetName()).To(Equal(AliasPrefix + "vgpu1"))
			Expect(dev.Display).To(Equal("on"))
			Expect(dev.RamFB).To(Equal("on"))
		})
	})

	Context("when VMI has both a pGPU and a vGPU", func() {
		It("should create one PCI and one mdev host device", func() {
			pciAddr := "0000:00:01.0"
			vgpuPCIAddr := "0000:01:01.0"
			mdevUUID := "deadbeef-e89b-12d3-a456-426614174000"

			createMetadataFile("pgpu-claim", "gpu", "gpu.example.com", &metadata.DeviceMetadata{
				TypeMeta: metav1.TypeMeta{
					Kind:       "DeviceMetadata",
					APIVersion: "v1alpha1",
				},
				ObjectMeta: metav1.ObjectMeta{
					Name: "pgpu-claim",
				},
				Requests: []metadata.DeviceMetadataRequest{{
					Name: "gpu",
					Devices: []metadata.Device{{
						Driver: "gpu.example.com",
						Pool:   "node01",
						Name:   "gpu-0",
						Attributes: map[resourcev1.QualifiedName]resourcev1.DeviceAttribute{
							metadata.PCIBusIDAttribute: {StringValue: &pciAddr},
						},
					}},
				}},
			})

			createMetadataFile("vgpu-claim", "vgpu", "gpu.example.com", &metadata.DeviceMetadata{
				TypeMeta: metav1.TypeMeta{
					Kind:       "DeviceMetadata",
					APIVersion: "v1alpha1",
				},
				ObjectMeta: metav1.ObjectMeta{
					Name: "vgpu-claim",
				},
				Requests: []metadata.DeviceMetadataRequest{{
					Name: "vgpu",
					Devices: []metadata.Device{{
						Driver: "gpu.example.com",
						Pool:   "node01",
						Name:   "gpu-0-vgpu-0",
						Attributes: map[resourcev1.QualifiedName]resourcev1.DeviceAttribute{
							metadata.PCIBusIDAttribute: {StringValue: &vgpuPCIAddr},
							metadata.MDevUUIDAttribute: {StringValue: &mdevUUID},
						},
					}},
				}},
			})

			vmi := &v1.VirtualMachineInstance{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "testvmi",
					Namespace: "default",
				},
				Spec: v1.VirtualMachineInstanceSpec{
					ResourceClaims: []k8sv1.PodResourceClaim{
						{Name: "pgpu", ResourceClaimName: ptr.To("pgpu-claim")},
						{Name: "vgpu", ResourceClaimName: ptr.To("vgpu-claim")},
					},
					Domain: v1.DomainSpec{
						Devices: v1.Devices{
							GPUs: []v1.GPU{
								{
									Name: "pgpu0",
									ClaimRequest: &v1.ClaimRequest{
										ClaimName:   ptr.To("pgpu"),
										RequestName: ptr.To("gpu"),
									},
								},
								{
									Name: "vgpu0",
									ClaimRequest: &v1.ClaimRequest{
										ClaimName:   ptr.To("vgpu"),
										RequestName: ptr.To("vgpu"),
									},
								},
							},
						},
					},
				},
			}

			downwardAPIAttributes, err := drautil.NewDownwardAPIAttributesWithBasePath(tempDir, vmi.Spec.ResourceClaims)
			Expect(err).ToNot(HaveOccurred())

			hostDevices, err := CreateDRAGPUHostDevices(vmi, downwardAPIAttributes)
			Expect(err).ToNot(HaveOccurred())
			Expect(hostDevices).To(HaveLen(2))

			var pciDev, mdevDev *api.HostDevice
			for i := range hostDevices {
				switch hostDevices[i].Type {
				case api.HostDevicePCI:
					pciDev = &hostDevices[i]
				case api.HostDeviceMDev:
					mdevDev = &hostDevices[i]
				}
			}

			Expect(pciDev).ToNot(BeNil(), "expected a PCI host device for the pGPU")
			Expect(pciDev.Alias.GetName()).To(Equal(AliasPrefix + "pgpu0"))
			Expect(pciDev.Managed).To(Equal("no"))

			Expect(mdevDev).ToNot(BeNil(), "expected an mdev host device for the vGPU")
			Expect(mdevDev.Alias.GetName()).To(Equal(AliasPrefix + "vgpu0"))
			Expect(mdevDev.Source.Address.UUID).To(Equal(mdevUUID))
		})
	})

	Context("validation errors", func() {
		It("should return an error when metadata is missing for a DRA GPU", func() {
			pciAddr := "0000:02:00.0"
			createMetadataFile("claim1", "req1", "gpu.example.com", &metadata.DeviceMetadata{
				TypeMeta: metav1.TypeMeta{
					Kind:       "DeviceMetadata",
					APIVersion: "v1alpha1",
				},
				ObjectMeta: metav1.ObjectMeta{
					Name: "claim1",
				},
				Requests: []metadata.DeviceMetadataRequest{{
					Name: "req1",
					Devices: []metadata.Device{{
						Driver: "gpu.example.com",
						Pool:   "gpu-pool",
						Name:   "device1",
						Attributes: map[resourcev1.QualifiedName]resourcev1.DeviceAttribute{
							metadata.PCIBusIDAttribute: {StringValue: &pciAddr},
						},
					}},
				}},
			})

			vmi := &v1.VirtualMachineInstance{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "testvmi",
					Namespace: "default",
				},
				Spec: v1.VirtualMachineInstanceSpec{
					ResourceClaims: []k8sv1.PodResourceClaim{
						{Name: "claim1", ResourceClaimName: ptr.To("claim1")},
						{Name: "claim2", ResourceClaimName: ptr.To("claim2")},
					},
					Domain: v1.DomainSpec{
						Devices: v1.Devices{
							GPUs: []v1.GPU{
								{
									Name: "gpu1",
									ClaimRequest: &v1.ClaimRequest{
										ClaimName:   ptr.To("claim1"),
										RequestName: ptr.To("req1"),
									},
								},
								{
									Name: "gpu2",
									ClaimRequest: &v1.ClaimRequest{
										ClaimName:   ptr.To("claim2"),
										RequestName: ptr.To("req2"),
									},
								},
							},
						},
					},
				},
			}

			downwardAPIAttributes, err := drautil.NewDownwardAPIAttributesWithBasePath(tempDir, vmi.Spec.ResourceClaims)
			Expect(err).ToNot(HaveOccurred())

			hostDevices, err := CreateDRAGPUHostDevices(vmi, downwardAPIAttributes)
			Expect(err).To(HaveOccurred())
			Expect(hostDevices).To(BeNil())
		})
	})
})
