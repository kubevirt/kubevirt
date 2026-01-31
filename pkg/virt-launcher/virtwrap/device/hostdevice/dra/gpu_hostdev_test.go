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

	createMetadataFile := func(driver, claimName string, md *metadata.DeviceMetadata) {
		dir := filepath.Join(tempDir, driver, claimName)
		Expect(os.MkdirAll(dir, 0755)).To(Succeed())

		data, err := json.Marshal(md)
		Expect(err).ToNot(HaveOccurred())

		Expect(os.WriteFile(filepath.Join(dir, drautil.MetadataFileName), data, 0644)).To(Succeed())
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

			draFileData, err := drautil.NewDRAFileDataWithBasePath(tempDir, nil)
			Expect(err).ToNot(HaveOccurred())

			hostDevices, err := CreateDRAGPUHostDevices(vmi, draFileData)
			Expect(err).ToNot(HaveOccurred())
			Expect(hostDevices).To(BeEmpty())
		})
	})

	Context("when the VMI has a physical GPU (PCI) allocated through DRA", func() {
		It("should create exactly one PCI host device", func() {
			pciAddr := "0000:02:00.0"

			// Create metadata file
			createMetadataFile("gpu.example.com", "claim1", &metadata.DeviceMetadata{
				TypeMeta: metav1.TypeMeta{
					Kind:       "DeviceMetadata",
					APIVersion: "v1alpha1",
				},
				ObjectMeta: metav1.ObjectMeta{
					Name: "claim1",
				},
				Requests: []metadata.DeviceRequest{{
					Name: "req1",
					Devices: []metadata.Device{{
						Driver: "gpu.example.com",
						Pool:   "gpu-pool",
						Device: "device1",
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

			draFileData, err := drautil.NewDRAFileDataWithBasePath(tempDir, vmi.Spec.ResourceClaims)
			Expect(err).ToNot(HaveOccurred())

			hostDevices, err := CreateDRAGPUHostDevices(vmi, draFileData)
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

			// Create metadata file
			createMetadataFile("gpu.example.com", "claim1", &metadata.DeviceMetadata{
				TypeMeta: metav1.TypeMeta{
					Kind:       "DeviceMetadata",
					APIVersion: "v1alpha1",
				},
				ObjectMeta: metav1.ObjectMeta{
					Name: "claim1",
				},
				Requests: []metadata.DeviceRequest{{
					Name: "req1",
					Devices: []metadata.Device{{
						Driver: "gpu.example.com",
						Pool:   "gpu-pool",
						Device: "device1",
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

			draFileData, err := drautil.NewDRAFileDataWithBasePath(tempDir, vmi.Spec.ResourceClaims)
			Expect(err).ToNot(HaveOccurred())

			hostDevices, err := CreateDRAGPUHostDevices(vmi, draFileData)
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

	Context("validation errors", func() {
		It("should return an error when metadata is missing for a DRA GPU", func() {
			// Only create metadata for one of the two claims
			pciAddr := "0000:02:00.0"
			createMetadataFile("gpu.example.com", "claim1", &metadata.DeviceMetadata{
				TypeMeta: metav1.TypeMeta{
					Kind:       "DeviceMetadata",
					APIVersion: "v1alpha1",
				},
				ObjectMeta: metav1.ObjectMeta{
					Name: "claim1",
				},
				Requests: []metadata.DeviceRequest{{
					Name: "req1",
					Devices: []metadata.Device{{
						Driver: "gpu.example.com",
						Pool:   "gpu-pool",
						Device: "device1",
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

			draFileData, err := drautil.NewDRAFileDataWithBasePath(tempDir, vmi.Spec.ResourceClaims)
			Expect(err).ToNot(HaveOccurred())

			hostDevices, err := CreateDRAGPUHostDevices(vmi, draFileData)
			Expect(err).To(HaveOccurred())
			Expect(hostDevices).To(BeNil())
		})
	})
})
