package dra

import (
	"fmt"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/utils/ptr"

	v1 "kubevirt.io/api/core/v1"

	drautil "kubevirt.io/kubevirt/pkg/dra"
	"kubevirt.io/kubevirt/pkg/virt-launcher/virtwrap/api"
)

var _ = Describe("CreateDRAGPUHostDevices", func() {
	BeforeEach(func() {
		DeferCleanup(func() {
			getMDevUUID = drautil.GetMDevUUIDForClaim
			getPCIAddress = drautil.GetPCIAddressForClaim
		})
	})

	Context("when the VMI has no GPUs with DRA", func() {
		It("should return an empty slice without error", func() {
			vmi := &v1.VirtualMachineInstance{
				ObjectMeta: metav1.ObjectMeta{Name: "testvmi", Namespace: "default"},
				Spec:       v1.VirtualMachineInstanceSpec{Domain: v1.DomainSpec{}},
			}

			hostDevices, err := CreateDRAGPUHostDevices(vmi)
			Expect(err).ToNot(HaveOccurred())
			Expect(hostDevices).To(BeEmpty())
		})
	})

	Context("when the VMI has a physical GPU (PCI) allocated through DRA", func() {
		It("should create exactly one PCI host device", func() {
			pciAddr := "0000:02:00.0"
			getMDevUUID = func(_ []v1.VirtualMachineInstanceResourceClaim, _, _ string) (string, error) {
				return "", fmt.Errorf("no mdev")
			}
			getPCIAddress = func(_ []v1.VirtualMachineInstanceResourceClaim, _, _ string) (string, error) { return pciAddr, nil }

			vmi := &v1.VirtualMachineInstance{
				ObjectMeta: metav1.ObjectMeta{Name: "testvmi", Namespace: "default"},
				Spec: v1.VirtualMachineInstanceSpec{
					ResourceClaims: []v1.VirtualMachineInstanceResourceClaim{{
						Name:              "claim1",
						ResourceClaimName: ptr.To("claim1"),
					}},
					Domain: v1.DomainSpec{
						Devices: v1.Devices{
							GPUs: []v1.GPU{{
								Name:         "gpu1",
								ClaimRequest: &v1.ClaimRequest{ClaimName: "claim1", RequestName: "req1"},
							}},
						},
					},
				},
			}

			hostDevices, err := CreateDRAGPUHostDevices(vmi)
			Expect(err).ToNot(HaveOccurred())
			Expect(hostDevices).To(HaveLen(1))

			dev := hostDevices[0]
			Expect(dev.Type).To(Equal(api.HostDevicePCI))
			Expect(dev.Managed).To(Equal("no"))
			Expect(dev.Alias.GetName()).To(Equal(AliasPrefix + "gpu1"))
			Expect(dev.Source.Address.Type).To(Equal(api.AddressPCI))
		})
	})

	Context("when the VMI has a virtual GPU (mdev) allocated through DRA", func() {
		It("should create exactly one mdev host device with display enabled", func() {
			uuid := "123e4567-e89b-12d3-a456-426614174000"
			getMDevUUID = func(_ []v1.VirtualMachineInstanceResourceClaim, _, _ string) (string, error) { return uuid, nil }

			vmi := &v1.VirtualMachineInstance{
				ObjectMeta: metav1.ObjectMeta{Name: "testvmi", Namespace: "default"},
				Spec: v1.VirtualMachineInstanceSpec{
					ResourceClaims: []v1.VirtualMachineInstanceResourceClaim{{
						Name:              "claim1",
						ResourceClaimName: ptr.To("claim1"),
					}},
					Domain: v1.DomainSpec{
						Devices: v1.Devices{
							GPUs: []v1.GPU{{
								Name:         "vgpu1",
								ClaimRequest: &v1.ClaimRequest{ClaimName: "claim1", RequestName: "req1"},
							}},
						},
					},
				},
			}

			hostDevices, err := CreateDRAGPUHostDevices(vmi)
			Expect(err).ToNot(HaveOccurred())
			Expect(hostDevices).To(HaveLen(1))

			dev := hostDevices[0]
			Expect(dev.Type).To(Equal(api.HostDeviceMDev))
			Expect(dev.Display).To(Equal("on"))
			Expect(dev.RamFB).To(Equal("on"))
			Expect(dev.Alias.GetName()).To(Equal(AliasPrefix + "vgpu1"))
			Expect(dev.Source.Address.UUID).To(Equal(uuid))
		})
	})

	Context("when the device has both pciBusID and mdevUUID", func() {
		It("should prefer mdevUUID and create an mdev host device", func() {
			pciAddr := "0000:01:01.0"
			mdevUUID := "abcd1234-e89b-12d3-a456-426614174000"
			getMDevUUID = func(_ []v1.VirtualMachineInstanceResourceClaim, _, _ string) (string, error) { return mdevUUID, nil }
			getPCIAddress = func(_ []v1.VirtualMachineInstanceResourceClaim, _, _ string) (string, error) { return pciAddr, nil }

			vmi := &v1.VirtualMachineInstance{
				ObjectMeta: metav1.ObjectMeta{Name: "testvmi", Namespace: "default"},
				Spec: v1.VirtualMachineInstanceSpec{
					ResourceClaims: []v1.VirtualMachineInstanceResourceClaim{{
						Name:              "claim1",
						ResourceClaimName: ptr.To("claim1"),
					}},
					Domain: v1.DomainSpec{
						Devices: v1.Devices{
							GPUs: []v1.GPU{{
								Name:         "vgpu1",
								ClaimRequest: &v1.ClaimRequest{ClaimName: "claim1", RequestName: "req1"},
							}},
						},
					},
				},
			}

			hostDevices, err := CreateDRAGPUHostDevices(vmi)
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
			mdevUUID := "deadbeef-e89b-12d3-a456-426614174000"

			getMDevUUID = func(_ []v1.VirtualMachineInstanceResourceClaim, claimRefName, _ string) (string, error) {
				if claimRefName == "vgpu" {
					return mdevUUID, nil
				}
				return "", fmt.Errorf("no mdev for %q", claimRefName)
			}
			getPCIAddress = func(_ []v1.VirtualMachineInstanceResourceClaim, claimRefName, _ string) (string, error) {
				if claimRefName == "pgpu" {
					return pciAddr, nil
				}
				return "", fmt.Errorf("no pci for %q", claimRefName)
			}

			vmi := &v1.VirtualMachineInstance{
				ObjectMeta: metav1.ObjectMeta{Name: "testvmi", Namespace: "default"},
				Spec: v1.VirtualMachineInstanceSpec{
					ResourceClaims: []v1.VirtualMachineInstanceResourceClaim{
						{Name: "pgpu", ResourceClaimName: ptr.To("pgpu-claim")},
						{Name: "vgpu", ResourceClaimName: ptr.To("vgpu-claim")},
					},
					Domain: v1.DomainSpec{
						Devices: v1.Devices{
							GPUs: []v1.GPU{
								{Name: "pgpu0", ClaimRequest: &v1.ClaimRequest{ClaimName: "pgpu", RequestName: "gpu"}},
								{Name: "vgpu0", ClaimRequest: &v1.ClaimRequest{ClaimName: "vgpu", RequestName: "vgpu"}},
							},
						},
					},
				},
			}

			hostDevices, err := CreateDRAGPUHostDevices(vmi)
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
			getMDevUUID = func(_ []v1.VirtualMachineInstanceResourceClaim, _, _ string) (string, error) {
				return "", fmt.Errorf("no mdev")
			}
			getPCIAddress = func(_ []v1.VirtualMachineInstanceResourceClaim, claimRefName, _ string) (string, error) {
				if claimRefName == "claim1" {
					return pciAddr, nil
				}
				return "", fmt.Errorf("attribute not found")
			}

			vmi := &v1.VirtualMachineInstance{
				ObjectMeta: metav1.ObjectMeta{Name: "testvmi", Namespace: "default"},
				Spec: v1.VirtualMachineInstanceSpec{
					ResourceClaims: []v1.VirtualMachineInstanceResourceClaim{
						{Name: "claim1", ResourceClaimName: ptr.To("claim1")},
						{Name: "claim2", ResourceClaimName: ptr.To("claim2")},
					},
					Domain: v1.DomainSpec{
						Devices: v1.Devices{
							GPUs: []v1.GPU{
								{Name: "gpu1", ClaimRequest: &v1.ClaimRequest{ClaimName: "claim1", RequestName: "req1"}},
								{Name: "gpu2", ClaimRequest: &v1.ClaimRequest{ClaimName: "claim2", RequestName: "req2"}},
							},
						},
					},
				},
			}

			hostDevices, err := CreateDRAGPUHostDevices(vmi)
			Expect(err).To(HaveOccurred())
			Expect(hostDevices).To(BeNil())
		})
	})
})
