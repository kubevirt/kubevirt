package dra

import (
	"encoding/json"
	"os"
	"path/filepath"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"go.uber.org/mock/gomock"

	k8sv1 "k8s.io/api/core/v1"
	resourcev1 "k8s.io/api/resource/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/utils/ptr"

	v1 "kubevirt.io/api/core/v1"

	"kubevirt.io/kubevirt/pkg/dra/metadata"
	"kubevirt.io/kubevirt/pkg/virt-launcher/virtwrap/api"
	"kubevirt.io/kubevirt/pkg/virt-launcher/virtwrap/vfio"
)

var _ = Describe("CreateDRAHostDevices", func() {
	var pciAddr0 = "0000:03:00.1"
	var mdevUUID0 = "abcd1234-1111-2222-3333-444455556666"

	var tempDir string
	var vfioSpec vfio.VFIOSpec

	DescribeTableSubtree("creation", func(viaIOMMUFD bool) {
		BeforeEach(func() {
			var err error
			tempDir, err = os.MkdirTemp("", "dra-hostdev-test")
			Expect(err).ToNot(HaveOccurred())

			mockVFIOSpec := vfio.NewMockVFIOSpec(gomock.NewController(GinkgoT()))
			mockVFIOSpec.EXPECT().IsPCIAssignableViaIOMMUFD(pciAddr0).Return(viaIOMMUFD).AnyTimes()
			mockVFIOSpec.EXPECT().IsPCIAssignableViaIOMMUFD(gomock.Any()).Times(0)
			mockVFIOSpec.EXPECT().IsMDevAssignableViaIOMMUFD(mdevUUID0).Return(viaIOMMUFD).AnyTimes()
			mockVFIOSpec.EXPECT().IsMDevAssignableViaIOMMUFD(gomock.Any()).Times(0)
			vfioSpec = mockVFIOSpec
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

		Context("when the VMI has no host devices with DRA", func() {
			It("should return an empty slice without error", func() {
				vmi := &v1.VirtualMachineInstance{
					ObjectMeta: metav1.ObjectMeta{Namespace: "default", Name: "vmi"},
					Spec:       v1.VirtualMachineInstanceSpec{Domain: v1.DomainSpec{}},
				}

				hostDevs, err := CreateDRAHostDevices(vmi, tempDir, vfioSpec)
				Expect(err).ToNot(HaveOccurred())
				Expect(hostDevs).To(BeEmpty())
			})
		})

		Context("when the VMI has a PCI host device allocated through DRA", func() {
			It("should create a PCI HostDevice with correct attributes", func() {
				createMetadataFile("claim1", "req1", "device.example.com", &metadata.DeviceMetadata{
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
							Driver: "device.example.com",
							Pool:   "device-pool",
							Name:   "device1",
							Attributes: map[resourcev1.QualifiedName]resourcev1.DeviceAttribute{
								metadata.PCIBusIDAttribute: {StringValue: &pciAddr0},
							},
						}},
					}},
				})

				vmi := &v1.VirtualMachineInstance{
					ObjectMeta: metav1.ObjectMeta{Namespace: "default", Name: "vmi"},
					Spec: v1.VirtualMachineInstanceSpec{
						ResourceClaims: []k8sv1.PodResourceClaim{{
							Name:              "claim1",
							ResourceClaimName: ptr.To("claim1"),
						}},
						Domain: v1.DomainSpec{
							Devices: v1.Devices{
								HostDevices: []v1.HostDevice{{
									Name:         "hd1",
									ClaimRequest: &v1.ClaimRequest{ClaimName: ptr.To("claim1"), RequestName: ptr.To("req1")},
								}},
							},
						},
					},
				}

				hostDevs, err := CreateDRAHostDevices(vmi, tempDir, vfioSpec)
				Expect(err).ToNot(HaveOccurred())
				Expect(hostDevs).To(HaveLen(1))

				dev := hostDevs[0]
				Expect(dev.Type).To(Equal(api.HostDevicePCI))
				Expect(dev.Managed).To(Equal("no"))
				Expect(dev.Alias.GetName()).To(Equal(DRAHostDeviceAliasPrefix + "hd1"))
				Expect(dev.Source.Address.Type).To(Equal(api.AddressPCI))
				if viaIOMMUFD {
					Expect(dev.Driver).ToNot(BeNil())
					Expect(dev.Driver.IOMMUFD).To(Equal("yes"))
				} else {
					Expect(dev.Driver).To(BeNil())
				}
			})
		})

		Context("when the VMI has an MDEV host device allocated through DRA", func() {
			It("should create an MDEV HostDevice", func() {
				createMetadataFile("claim1", "req1", "mdev.example.com", &metadata.DeviceMetadata{
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
							Driver: "mdev.example.com",
							Pool:   "mdev-pool",
							Name:   "device1",
							Attributes: map[resourcev1.QualifiedName]resourcev1.DeviceAttribute{
								metadata.MDevUUIDAttribute: {StringValue: &mdevUUID0},
							},
						}},
					}},
				})

				vmi := &v1.VirtualMachineInstance{
					ObjectMeta: metav1.ObjectMeta{Namespace: "default", Name: "vmi"},
					Spec: v1.VirtualMachineInstanceSpec{
						ResourceClaims: []k8sv1.PodResourceClaim{{
							Name:              "claim1",
							ResourceClaimName: ptr.To("claim1"),
						}},
						Domain: v1.DomainSpec{
							Devices: v1.Devices{
								HostDevices: []v1.HostDevice{{
									Name:         "vhd1",
									ClaimRequest: &v1.ClaimRequest{ClaimName: ptr.To("claim1"), RequestName: ptr.To("req1")},
								}},
							},
						},
					},
				}

				hostDevs, err := CreateDRAHostDevices(vmi, tempDir, vfioSpec)
				Expect(err).ToNot(HaveOccurred())
				Expect(hostDevs).To(HaveLen(1))
				dev := hostDevs[0]
				Expect(dev.Type).To(Equal(api.HostDeviceMDev))
				Expect(dev.Alias.GetName()).To(Equal(DRAHostDeviceAliasPrefix + "vhd1"))
				Expect(dev.Source.Address.UUID).To(Equal(mdevUUID0))
				Expect(dev.Source.Address.Type).To(Equal(""))
				Expect(dev.Source.Address.Bus).To(Equal(""))
				Expect(dev.Mode).To(Equal("subsystem"))
				Expect(dev.Model).To(Equal("vfio-pci"))
				if viaIOMMUFD {
					Expect(dev.Driver).ToNot(BeNil())
					Expect(dev.Driver.IOMMUFD).To(Equal("yes"))
				} else {
					Expect(dev.Driver).To(BeNil())
				}
			})
		})

		Context("when the VMI has an MDEV host device allocated through DRA (s390x)", func() {
			It("should create an MDEV HostDevice", func() {
				createMetadataFile("claim1", "req1", "mdev.example.com", &metadata.DeviceMetadata{
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
							Driver: "mdev.example.com",
							Pool:   "mdev-pool",
							Name:   "device1",
							Attributes: map[resourcev1.QualifiedName]resourcev1.DeviceAttribute{
								metadata.MDevUUIDAttribute: {StringValue: &mdevUUID0},
							},
						}},
					}},
				})

				vmi := &v1.VirtualMachineInstance{
					ObjectMeta: metav1.ObjectMeta{Namespace: "default", Name: "vmi"},
					Spec: v1.VirtualMachineInstanceSpec{
						ResourceClaims: []k8sv1.PodResourceClaim{{
							Name:              "claim1",
							ResourceClaimName: ptr.To("claim1"),
						}},
						Domain: v1.DomainSpec{
							Devices: v1.Devices{
								HostDevices: []v1.HostDevice{{
									Name:         "vhd1",
									ClaimRequest: &v1.ClaimRequest{ClaimName: ptr.To("claim1"), RequestName: ptr.To("req1")},
								}},
							},
						},
						Architecture: "s390x",
					},
				}

				hostDevs, err := CreateDRAHostDevices(vmi, tempDir, vfioSpec)
				Expect(err).ToNot(HaveOccurred())
				Expect(hostDevs).To(HaveLen(1))
				dev := hostDevs[0]
				Expect(dev.Type).To(Equal(api.HostDeviceMDev))
				Expect(dev.Alias.GetName()).To(Equal(DRAHostDeviceAliasPrefix + "vhd1"))
				Expect(dev.Source.Address.UUID).To(Equal(mdevUUID0))
				Expect(dev.Source.Address.Type).To(Equal(""))
				Expect(dev.Source.Address.Bus).To(Equal(""))
				Expect(dev.Mode).To(Equal("subsystem"))
				Expect(dev.Model).To(Equal("vfio-ap"))
				if viaIOMMUFD {
					Expect(dev.Driver).ToNot(BeNil())
					Expect(dev.Driver.IOMMUFD).To(Equal("yes"))
				} else {
					Expect(dev.Driver).To(BeNil())
				}
			})
		})

		Context("validation mismatch", func() {
			It("should error when metadata is missing for a DRA host device", func() {
				createMetadataFile("claim1", "req1", "device.example.com", &metadata.DeviceMetadata{
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
							Driver: "device.example.com",
							Pool:   "device-pool",
							Name:   "device1",
							Attributes: map[resourcev1.QualifiedName]resourcev1.DeviceAttribute{
								metadata.PCIBusIDAttribute: {StringValue: &pciAddr0},
							},
						}},
					}},
				})

				vmi := &v1.VirtualMachineInstance{
					ObjectMeta: metav1.ObjectMeta{Namespace: "default", Name: "vmi"},
					Spec: v1.VirtualMachineInstanceSpec{
						ResourceClaims: []k8sv1.PodResourceClaim{
							{Name: "claim1", ResourceClaimName: ptr.To("claim1")},
							{Name: "claim2", ResourceClaimName: ptr.To("claim2")},
						},
						Domain: v1.DomainSpec{
							Devices: v1.Devices{
								HostDevices: []v1.HostDevice{{
									Name:         "hd1",
									ClaimRequest: &v1.ClaimRequest{ClaimName: ptr.To("claim1"), RequestName: ptr.To("req1")},
								}, {
									Name:         "hd2",
									ClaimRequest: &v1.ClaimRequest{ClaimName: ptr.To("claim2"), RequestName: ptr.To("req2")},
								}},
							},
						},
					},
				}

				hostDevs, err := CreateDRAHostDevices(vmi, tempDir, vfioSpec)
				Expect(err).To(HaveOccurred())
				Expect(hostDevs).To(BeNil())
			})
		})
	},
		Entry("via IOMMUFD", true),
		Entry("via VFIO legacy", false),
	)
})
