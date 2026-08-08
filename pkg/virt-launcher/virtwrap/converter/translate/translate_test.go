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

package translate

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	libvirtxml "libvirt.org/go/libvirtxml"

	"k8s.io/apimachinery/pkg/types"

	api "kubevirt.io/kubevirt/pkg/virt-launcher/virtwrap/api"
)

const (
	domainTypeKVM   = "kvm"
	diskTypeFile    = "file"
	diskDeviceDisk  = "disk"
	diskBusVirtio   = "virtio"
	diskDeviceVDA   = "vda"
	diskDriverQEMU  = "qemu"
	diskDriverRaw   = "raw"
	ifaceTypeBridge = "bridge"
	ifaceBridgeName = "br0"
	ifaceMAC        = "52:54:00:00:00:01"
	testUID         = "test-uid-12345"
	testVMName      = "test-vm"
	diskDeviceCDROM = "cdrom"
)

var _ = Describe("Domain translation", func() {
	Context("ToLibvirtDomain", func() {
		It("should convert a minimal DomainSpec", func() {
			spec := api.NewMinimalDomainSpec(testVMName)
			spec.Type = domainTypeKVM
			domain, err := ToLibvirtDomain(spec)
			Expect(err).ToNot(HaveOccurred())
			Expect(domain).ToNot(BeNil())
			Expect(domain.Name).To(Equal(testVMName))
			Expect(domain.Type).To(Equal(domainTypeKVM))

			assertDomainSpecRoundTrip(spec)
		})

		It("should convert a DomainSpec with file disk", func() {
			spec := api.NewMinimalDomainSpec(testVMName)
			spec.Devices.Disks = []api.Disk{
				{
					Type:   diskTypeFile,
					Device: diskDeviceDisk,
					Source: api.DiskSource{
						File: "/var/run/libvirt/images/disk.img",
					},
					Target: api.DiskTarget{
						Bus:    diskBusVirtio,
						Device: diskDeviceVDA,
					},
					Driver: &api.DiskDriver{
						Name: diskDriverQEMU,
						Type: diskDriverRaw,
					},
				},
			}

			domain, err := ToLibvirtDomain(spec)
			Expect(err).ToNot(HaveOccurred())
			Expect(domain.Devices.Disks).To(HaveLen(1))
			Expect(domain.Devices.Disks[0].Source.File).ToNot(BeNil())
			Expect(domain.Devices.Disks[0].Source.File.File).To(Equal("/var/run/libvirt/images/disk.img"))

			assertDomainSpecRoundTrip(spec)
		})

		It("should convert a DomainSpec with bridge interface", func() {
			spec := api.NewMinimalDomainSpec(testVMName)
			spec.Devices.Interfaces = []api.Interface{
				{
					Type: ifaceTypeBridge,
					Source: api.InterfaceSource{
						Bridge: ifaceBridgeName,
					},
					Model: &api.Model{Type: diskBusVirtio},
					MAC:   &api.MAC{MAC: ifaceMAC},
				},
			}

			domain, err := ToLibvirtDomain(spec)
			Expect(err).ToNot(HaveOccurred())
			Expect(domain.Devices.Interfaces).To(HaveLen(1))
			Expect(domain.Devices.Interfaces[0].Source.Bridge).ToNot(BeNil())
			Expect(domain.Devices.Interfaces[0].Source.Bridge.Bridge).To(Equal(ifaceBridgeName))

			assertDomainSpecRoundTrip(spec)
		})

		It("should convert a DomainSpec with QEMU commandline", func() {
			spec := api.NewMinimalDomainSpec(testVMName)
			spec.XmlNS = "http://libvirt.org/schemas/domain/qemu/1.0"
			spec.QEMUCmd = &api.Commandline{
				QEMUArg: []api.Arg{
					{Value: "-fw_cfg"},
					{Value: "name=opt/test,string=value"},
				},
				QEMUEnv: []api.Env{
					{Name: "QEMU_TEST", Value: "1"},
				},
			}

			domain, err := ToLibvirtDomain(spec)
			Expect(err).ToNot(HaveOccurred())
			Expect(domain.QEMUCommandline).ToNot(BeNil())
			Expect(domain.QEMUCommandline.Args).To(HaveLen(2))
			Expect(domain.QEMUCommandline.Args[0].Value).To(Equal("-fw_cfg"))
			Expect(domain.QEMUCommandline.Args[1].Value).To(Equal("name=opt/test,string=value"))
			Expect(domain.QEMUCommandline.Envs).To(HaveLen(1))
			Expect(domain.QEMUCommandline.Envs[0].Name).To(Equal("QEMU_TEST"))
			Expect(domain.QEMUCommandline.Envs[0].Value).To(Equal("1"))

			assertDomainSpecRoundTrip(spec)
		})

		It("should preserve metadata through conversion", func() {
			spec := api.NewMinimalDomainSpec(testVMName)
			spec.Metadata = api.Metadata{
				KubeVirt: api.KubeVirtMetadata{
					UID: testUID,
					GracePeriod: &api.GracePeriodMetadata{
						DeletionGracePeriodSeconds: 30,
					},
				},
			}

			domain, err := ToLibvirtDomain(spec)
			Expect(err).ToNot(HaveOccurred())
			Expect(domain.Metadata).ToNot(BeNil())
			Expect(domain.Metadata.XML).To(ContainSubstring(testUID))
		})

		It("should handle an empty DomainSpec", func() {
			spec := &api.DomainSpec{}
			domain, err := ToLibvirtDomain(spec)
			Expect(err).ToNot(HaveOccurred())
			Expect(domain).ToNot(BeNil())
		})

		It("should handle a DomainSpec with empty devices", func() {
			spec := api.NewMinimalDomainSpec(testVMName)
			spec.Devices = api.Devices{}
			domain, err := ToLibvirtDomain(spec)
			Expect(err).ToNot(HaveOccurred())
			Expect(domain).ToNot(BeNil())
		})

		It("should return error for nil DomainSpec", func() {
			_, err := ToLibvirtDomain(nil)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("must not be nil"))
		})
	})

	Context("FromLibvirtDomain", func() {
		It("should convert a libvirtxml Domain back to DomainSpec", func() {
			spec := api.NewMinimalDomainSpec(testVMName)
			spec.Type = domainTypeKVM
			domain, err := ToLibvirtDomain(spec)
			Expect(err).ToNot(HaveOccurred())

			spec2, err := FromLibvirtDomain(domain)
			Expect(err).ToNot(HaveOccurred())
			Expect(spec2).ToNot(BeNil())
			Expect(spec2.Name).To(Equal(testVMName))
			Expect(spec2.Type).To(Equal(domainTypeKVM))
		})

		It("should return error for nil libvirtxml Domain", func() {
			_, err := FromLibvirtDomain(nil)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("must not be nil"))
		})

		It("should set XmlNS but not QEMUCmd when QEMUCommandline has no args or envs", func() {
			domain := &libvirtxml.Domain{
				Name:            testVMName,
				QEMUCommandline: &libvirtxml.DomainQEMUCommandline{},
			}
			spec, err := FromLibvirtDomain(domain)
			Expect(err).ToNot(HaveOccurred())
			Expect(spec.XmlNS).To(Equal("http://libvirt.org/schemas/domain/qemu/1.0"))
			Expect(spec.QEMUCmd).To(BeNil())
		})

		It("should gracefully handle libvirt-only fields that have no KubeVirt equivalent", func() {
			domain := &libvirtxml.Domain{
				Name: testVMName,
				Type: domainTypeKVM,
				QEMUCapabilities: &libvirtxml.DomainQEMUCapabilities{
					Add: []libvirtxml.DomainQEMUCapabilitiesEntry{{Name: "cap-test"}},
				},
			}
			spec, err := FromLibvirtDomain(domain)
			Expect(err).ToNot(HaveOccurred())
			Expect(spec.Name).To(Equal(testVMName))
		})
	})

	Context("TransferQEMUCommandline", func() {
		It("should extract args and envs from domain XML", func() {
			domainXML := `<domain type="kvm" xmlns:qemu="http://libvirt.org/schemas/domain/qemu/1.0">
				<name>test-vm</name>
				<qemu:commandline>
					<qemu:arg value="-fw_cfg"/>
					<qemu:arg value="name=opt/com.coreos/config,file=/data.ign"/>
					<qemu:env name="QEMU_AUDIO_DRV" value="none"/>
				</qemu:commandline>
			</domain>`

			spec := &api.DomainSpec{}
			TransferQEMUCommandline(domainXML, spec)

			Expect(spec.XmlNS).To(Equal("http://libvirt.org/schemas/domain/qemu/1.0"))
			Expect(spec.QEMUCmd).NotTo(BeNil())
			Expect(spec.QEMUCmd.QEMUArg).To(HaveLen(2))
			Expect(spec.QEMUCmd.QEMUArg[0].Value).To(Equal("-fw_cfg"))
			Expect(spec.QEMUCmd.QEMUArg[1].Value).To(Equal("name=opt/com.coreos/config,file=/data.ign"))
			Expect(spec.QEMUCmd.QEMUEnv).To(HaveLen(1))
			Expect(spec.QEMUCmd.QEMUEnv[0].Name).To(Equal("QEMU_AUDIO_DRV"))
			Expect(spec.QEMUCmd.QEMUEnv[0].Value).To(Equal("none"))
		})

		It("should set XmlNS but not QEMUCmd for empty commandline", func() {
			domainXML := `<domain type="kvm" xmlns:qemu="http://libvirt.org/schemas/domain/qemu/1.0">
				<name>test-vm</name>
				<qemu:commandline></qemu:commandline>
			</domain>`

			spec := &api.DomainSpec{}
			TransferQEMUCommandline(domainXML, spec)

			Expect(spec.XmlNS).To(Equal("http://libvirt.org/schemas/domain/qemu/1.0"))
			Expect(spec.QEMUCmd).To(BeNil())
		})

		It("should not modify spec when no qemu namespace is present", func() {
			domainXML := `<domain type="kvm">
				<name>test-vm</name>
				<devices><disk type="file"/></devices>
			</domain>`

			spec := &api.DomainSpec{}
			TransferQEMUCommandline(domainXML, spec)

			Expect(spec.XmlNS).To(BeEmpty())
			Expect(spec.QEMUCmd).To(BeNil())
		})

		It("should handle envs without value attribute", func() {
			domainXML := `<domain type="kvm" xmlns:qemu="http://libvirt.org/schemas/domain/qemu/1.0">
				<name>test-vm</name>
				<qemu:commandline>
					<qemu:env name="DISPLAY"/>
				</qemu:commandline>
			</domain>`

			spec := &api.DomainSpec{}
			TransferQEMUCommandline(domainXML, spec)

			Expect(spec.QEMUCmd).NotTo(BeNil())
			Expect(spec.QEMUCmd.QEMUEnv).To(HaveLen(1))
			Expect(spec.QEMUCmd.QEMUEnv[0].Name).To(Equal("DISPLAY"))
			Expect(spec.QEMUCmd.QEMUEnv[0].Value).To(BeEmpty())
		})

		It("should skip env entries without a name attribute", func() {
			domainXML := `<domain type="kvm" xmlns:qemu="http://libvirt.org/schemas/domain/qemu/1.0">
				<name>test-vm</name>
				<qemu:commandline>
					<qemu:env value="orphan"/>
					<qemu:env name="VALID" value="yes"/>
				</qemu:commandline>
			</domain>`

			spec := &api.DomainSpec{}
			TransferQEMUCommandline(domainXML, spec)

			Expect(spec.QEMUCmd).NotTo(BeNil())
			Expect(spec.QEMUCmd.QEMUEnv).To(HaveLen(1))
			Expect(spec.QEMUCmd.QEMUEnv[0].Name).To(Equal("VALID"))
		})

		It("should include arg with empty value attribute", func() {
			domainXML := `<domain type="kvm" xmlns:qemu="http://libvirt.org/schemas/domain/qemu/1.0">
				<name>test-vm</name>
				<qemu:commandline>
					<qemu:arg value=""/>
					<qemu:arg value="-nographic"/>
				</qemu:commandline>
			</domain>`

			spec := &api.DomainSpec{}
			TransferQEMUCommandline(domainXML, spec)

			Expect(spec.QEMUCmd).NotTo(BeNil())
			Expect(spec.QEMUCmd.QEMUArg).To(HaveLen(2))
			Expect(spec.QEMUCmd.QEMUArg[0].Value).To(BeEmpty())
			Expect(spec.QEMUCmd.QEMUArg[1].Value).To(Equal("-nographic"))
		})

		It("should aggregate args across multiple commandline blocks", func() {
			domainXML := `<domain type="kvm" xmlns:qemu="http://libvirt.org/schemas/domain/qemu/1.0">
				<name>test-vm</name>
				<qemu:commandline>
					<qemu:arg value="-fw_cfg"/>
				</qemu:commandline>
				<qemu:commandline>
					<qemu:arg value="-device"/>
				</qemu:commandline>
			</domain>`

			spec := &api.DomainSpec{}
			TransferQEMUCommandline(domainXML, spec)

			Expect(spec.QEMUCmd).NotTo(BeNil())
			Expect(spec.QEMUCmd.QEMUArg).To(HaveLen(2))
			Expect(spec.QEMUCmd.QEMUArg[0].Value).To(Equal("-fw_cfg"))
			Expect(spec.QEMUCmd.QEMUArg[1].Value).To(Equal("-device"))
		})

		It("should skip qemu elements outside commandline block", func() {
			domainXML := `<domain type="kvm" xmlns:qemu="http://libvirt.org/schemas/domain/qemu/1.0">
				<name>test-vm</name>
				<qemu:arg value="stray"/>
				<qemu:commandline>
					<qemu:arg value="real"/>
				</qemu:commandline>
			</domain>`

			spec := &api.DomainSpec{}
			TransferQEMUCommandline(domainXML, spec)

			Expect(spec.QEMUCmd).NotTo(BeNil())
			Expect(spec.QEMUCmd.QEMUArg).To(HaveLen(1))
			Expect(spec.QEMUCmd.QEMUArg[0].Value).To(Equal("real"))
		})
	})

	Context("Round-trip fidelity", func() {
		It("should round-trip a DomainSpec with metadata", func() {
			spec := api.NewMinimalDomainSpec(testVMName)
			spec.Metadata = api.Metadata{
				KubeVirt: api.KubeVirtMetadata{
					UID: testUID,
					GracePeriod: &api.GracePeriodMetadata{
						DeletionGracePeriodSeconds: 30,
					},
				},
			}

			domain, err := ToLibvirtDomain(spec)
			Expect(err).ToNot(HaveOccurred())

			roundTripped, err := FromLibvirtDomain(domain)
			Expect(err).ToNot(HaveOccurred())

			Expect(roundTripped.Metadata.KubeVirt.UID).To(Equal(types.UID(testUID)))
			Expect(roundTripped.Metadata.KubeVirt.GracePeriod).ToNot(BeNil())
			Expect(roundTripped.Metadata.KubeVirt.GracePeriod.DeletionGracePeriodSeconds).To(Equal(int64(30)))
		})

		It("should round-trip a DomainSpec with CPU topology", func() {
			spec := api.NewMinimalDomainSpec(testVMName)
			spec.CPU = api.CPU{
				Mode: "host-passthrough",
				Topology: &api.CPUTopology{
					Sockets: 1,
					Cores:   4,
					Threads: 2,
				},
			}
			spec.VCPU = &api.VCPU{
				Placement: "static",
				CPUs:      8,
			}
			assertDomainSpecRoundTrip(spec)
		})

		It("should round-trip a DomainSpec with OS and boot order", func() {
			spec := api.NewMinimalDomainSpec(testVMName)
			spec.OS = api.OS{
				Type: api.OSType{
					OS:      "hvm",
					Arch:    "x86_64",
					Machine: "q35",
				},
				BootOrder: []api.Boot{
					{Dev: "hd"},
					{Dev: diskDeviceCDROM},
				},
			}
			assertDomainSpecRoundTrip(spec)
		})

		It("should round-trip a DomainSpec with clock and timers", func() {
			spec := api.NewMinimalDomainSpec(testVMName)
			spec.Clock = &api.Clock{
				Offset: "utc",
				Timer: []api.Timer{
					{Name: "rtc", TickPolicy: "catchup", Track: "guest"},
					{Name: "pit", TickPolicy: "delay"},
					{Name: "hpet", Present: "no"},
				},
			}
			assertDomainSpecRoundTrip(spec)
		})

		It("should round-trip a DomainSpec with features", func() {
			spec := api.NewMinimalDomainSpec(testVMName)
			spec.Features = &api.Features{
				ACPI: &api.FeatureEnabled{},
				APIC: &api.FeatureEnabled{},
				SMM:  &api.FeatureEnabled{},
			}
			assertDomainSpecRoundTrip(spec)
		})

		It("should round-trip a DomainSpec with multiple devices", func() {
			spec := api.NewMinimalDomainSpec(testVMName)
			spec.Devices.Disks = []api.Disk{
				{
					Type:   diskTypeFile,
					Device: diskDeviceDisk,
					Source: api.DiskSource{File: "/images/disk1.img"},
					Target: api.DiskTarget{Bus: diskBusVirtio, Device: diskDeviceVDA},
					Driver: &api.DiskDriver{Name: diskDriverQEMU, Type: diskDriverRaw},
				},
				{
					Type:     diskTypeFile,
					Device:   diskDeviceCDROM,
					Source:   api.DiskSource{File: "/images/cloud-init.iso"},
					Target:   api.DiskTarget{Bus: "sata", Device: "sda"},
					Driver:   &api.DiskDriver{Name: diskDriverQEMU, Type: diskDriverRaw},
					ReadOnly: &api.ReadOnly{},
				},
			}
			spec.Devices.Interfaces = []api.Interface{
				{
					Type:   ifaceTypeBridge,
					Source: api.InterfaceSource{Bridge: ifaceBridgeName},
					Model:  &api.Model{Type: diskBusVirtio},
					MAC:    &api.MAC{MAC: ifaceMAC},
				},
			}
			assertDomainSpecRoundTrip(spec)
		})
	})
})

func assertDomainSpecRoundTrip(original *api.DomainSpec) {
	domain, err := ToLibvirtDomain(original)
	Expect(err).ToNot(HaveOccurred())

	roundTripped, err := FromLibvirtDomain(domain)
	Expect(err).ToNot(HaveOccurred())

	expected := original.DeepCopy()
	normalizeForComparison(expected)
	normalizeForComparison(roundTripped)

	Expect(roundTripped).To(Equal(expected))
}

func normalizeForComparison(spec *api.DomainSpec) {
	// XMLName gets set during unmarshal; normalize it
	spec.XMLName.Local = "domain"
	spec.XMLName.Space = ""

	for i := range spec.Devices.Interfaces {
		spec.Devices.Interfaces[i].XMLName.Local = ""
	}

	// KubeVirt-only fields not present in libvirt XML are lost in round-trip
	for i := range spec.Devices.Disks {
		spec.Devices.Disks[i].FilesystemOverhead = nil
		spec.Devices.Disks[i].Capacity = nil
	}
}
