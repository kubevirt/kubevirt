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
 * Copyright 2017, 2018 Red Hat, Inc.
 *
 */

package api

import (
	"encoding/xml"
	"fmt"

	. "github.com/onsi/ginkgo"
	"github.com/onsi/ginkgo/extensions/table"
	. "github.com/onsi/gomega"
)

var exampleXML = `<domain type="kvm" xmlns:qemu="http://libvirt.org/schemas/domain/qemu/1.0">
  <name>mynamespace_testvmi</name>
  <memory unit="MB">9</memory>
  <os>
    <type arch="x86_64" machine="q35">hvm</type>
  </os>
  <sysinfo type="smbios">
    <system>
      <entry name="uuid">e4686d2c-6e8d-4335-b8fd-81bee22f4814</entry>
    </system>
    <bios></bios>
    <baseBoard></baseBoard>
    <chassis></chassis>
  </sysinfo>
  <devices>
    <controller type="raw" index="0" model="none"></controller>
    <video>
      <model type="vga" heads="1" vram="16384"></model>
    </video>
    <memballoon model="none"></memballoon>
    <disk device="disk" type="network">
      <source protocol="iscsi" name="iqn.2013-07.com.example:iscsi-nopool/2">
        <host name="example.com" port="3260"></host>
      </source>
      <target dev="vda"></target>
      <driver name="qemu" type="raw"></driver>
      <alias name="ua-mydisk"></alias>
    </disk>
    <disk device="disk" type="file">
      <source file="/var/run/libvirt/cloud-init-dir/mynamespace/testvmi/noCloud.iso"></source>
      <target dev="vdb"></target>
      <driver name="qemu" type="raw"></driver>
      <alias name="ua-mydisk1"></alias>
    </disk>
    <disk device="disk" type="block">
      <source dev="/dev/testdev"></source>
      <target dev="vdc"></target>
      <driver name="qemu" type="raw"></driver>
      <alias name="ua-mydisk2"></alias>
    </disk>
    <input type="tablet" bus="virtio">
      <alias name="ua-tablet0"></alias>
    </input>
    <console type="pty"></console>
    <watchdog model="i6300esb" action="poweroff">
      <alias name="ua-mywatchdog"></alias>
    </watchdog>
    <rng model="virtio">
      <backend model="random">/dev/urandom</backend>
    </rng>
  </devices>
  <metadata>
    <kubevirt xmlns="http://kubevirt.io">
      <uid>f4686d2c-6e8d-4335-b8fd-81bee22f4814</uid>
      <graceperiod>
        <deletionGracePeriodSeconds>5</deletionGracePeriodSeconds>
      </graceperiod>
    </kubevirt>
  </metadata>
  <features>
    <acpi></acpi>
    <smm></smm>
  </features>
  <cpu mode="custom">
    <model>Conroe</model>
    <feature name="pcid" policy="require"></feature>
    <feature name="monitor" policy="disable"></feature>
    <topology sockets="1" cores="2" threads="1"></topology>
  </cpu>
  <vcpu placement="static">2</vcpu>
  <iothreads>2</iothreads>
</domain>`

var exampleXMLppc64le = `<domain type="kvm" xmlns:qemu="http://libvirt.org/schemas/domain/qemu/1.0">
  <name>mynamespace_testvmi</name>
  <memory unit="MB">9</memory>
  <os>
    <type arch="ppc64le" machine="pseries">hvm</type>
  </os>
  <sysinfo type="smbios">
    <system>
      <entry name="uuid">e4686d2c-6e8d-4335-b8fd-81bee22f4814</entry>
    </system>
    <bios></bios>
    <baseBoard></baseBoard>
    <chassis></chassis>
  </sysinfo>
  <devices>
    <controller type="raw" index="0" model="none"></controller>
    <video>
      <model type="vga" heads="1" vram="16384"></model>
    </video>
    <memballoon model="none"></memballoon>
    <disk device="disk" type="network">
      <source protocol="iscsi" name="iqn.2013-07.com.example:iscsi-nopool/2">
        <host name="example.com" port="3260"></host>
      </source>
      <target dev="vda"></target>
      <driver name="qemu" type="raw"></driver>
      <alias name="ua-mydisk"></alias>
    </disk>
    <disk device="disk" type="file">
      <source file="/var/run/libvirt/cloud-init-dir/mynamespace/testvmi/noCloud.iso"></source>
      <target dev="vdb"></target>
      <driver name="qemu" type="raw"></driver>
      <alias name="ua-mydisk1"></alias>
    </disk>
    <disk device="disk" type="block">
      <source dev="/dev/testdev"></source>
      <target dev="vdc"></target>
      <driver name="qemu" type="raw"></driver>
      <alias name="ua-mydisk2"></alias>
    </disk>
    <input type="tablet" bus="virtio">
      <alias name="ua-tablet0"></alias>
    </input>
    <console type="pty"></console>
    <watchdog model="i6300esb" action="poweroff">
      <alias name="ua-mywatchdog"></alias>
    </watchdog>
    <rng model="virtio">
      <backend model="random">/dev/urandom</backend>
    </rng>
  </devices>
  <metadata>
    <kubevirt xmlns="http://kubevirt.io">
      <uid>f4686d2c-6e8d-4335-b8fd-81bee22f4814</uid>
      <graceperiod>
        <deletionGracePeriodSeconds>5</deletionGracePeriodSeconds>
      </graceperiod>
    </kubevirt>
  </metadata>
  <features>
    <acpi></acpi>
    <smm></smm>
  </features>
  <cpu mode="custom">
    <model>Conroe</model>
    <feature name="pcid" policy="require"></feature>
    <feature name="monitor" policy="disable"></feature>
    <topology sockets="1" cores="2" threads="1"></topology>
  </cpu>
  <vcpu placement="static">2</vcpu>
  <iothreads>2</iothreads>
</domain>`

var _ = Describe("Schema", func() {
	//The example domain should stay in sync to the xml above
	var exampleDomain *Domain

	BeforeEach(func() {
		exampleDomain = NewMinimalDomainWithNS("mynamespace", "testvmi")
		exampleDomain.Spec.Devices.Disks = []Disk{
			{Type: "network",
				Device: "disk",
				Driver: &DiskDriver{Name: "qemu",
					Type: "raw"},
				Source: DiskSource{Protocol: "iscsi",
					Name: "iqn.2013-07.com.example:iscsi-nopool/2",
					Host: &DiskSourceHost{Name: "example.com", Port: "3260"}},
				Target: DiskTarget{Device: "vda"},
				Alias: &Alias{
					Name: "mydisk",
				},
			},
			{Type: "file",
				Device: "disk",
				Driver: &DiskDriver{Name: "qemu",
					Type: "raw"},
				Source: DiskSource{
					File: "/var/run/libvirt/cloud-init-dir/mynamespace/testvmi/noCloud.iso",
				},
				Target: DiskTarget{Device: "vdb"},
				Alias: &Alias{
					Name: "mydisk1",
				},
			},
			{Type: "block",
				Device: "disk",
				Driver: &DiskDriver{Name: "qemu",
					Type: "raw"},
				Source: DiskSource{
					Dev: "/dev/testdev",
				},
				Target: DiskTarget{Device: "vdc"},
				Alias: &Alias{
					Name: "mydisk2",
				},
			},
		}

		exampleDomain.Spec.Devices.Inputs = []Input{
			{
				Type: "tablet",
				Bus:  "virtio",
				Alias: &Alias{
					Name: "tablet0",
				},
			},
		}

		var heads uint = 1
		var vram uint = 16384
		exampleDomain.Spec.Devices.Video = []Video{
			{Model: VideoModel{Type: "vga", Heads: &heads, VRam: &vram}},
		}
		exampleDomain.Spec.Devices.Consoles = []Console{
			{Type: "pty"},
		}
		exampleDomain.Spec.Devices.Watchdog = &Watchdog{
			Model:  "i6300esb",
			Action: "poweroff",
			Alias: &Alias{
				Name: "mywatchdog",
			},
		}
		exampleDomain.Spec.Devices.Rng = &Rng{
			Model:   "virtio",
			Backend: &RngBackend{Source: "/dev/urandom", Model: "random"},
		}
		exampleDomain.Spec.Devices.Controllers = []Controller{
			{
				Type:  "raw",
				Model: "none",
				Index: "0",
			},
		}
		exampleDomain.Spec.Features = &Features{
			ACPI: &FeatureEnabled{},
			SMM:  &FeatureEnabled{},
		}
		exampleDomain.Spec.SysInfo = &SysInfo{
			Type: "smbios",
			System: []Entry{
				{Name: "uuid", Value: "e4686d2c-6e8d-4335-b8fd-81bee22f4814"},
			},
		}
		exampleDomain.Spec.CPU.Topology = &CPUTopology{
			Sockets: 1,
			Cores:   2,
			Threads: 1,
		}
		exampleDomain.Spec.VCPU = &VCPU{
			Placement: "static",
			CPUs:      2,
		}
		exampleDomain.Spec.CPU.Mode = "custom"
		exampleDomain.Spec.CPU.Model = "Conroe"
		exampleDomain.Spec.CPU.Features = []CPUFeature{
			{
				Name:   "pcid",
				Policy: "require",
			},
			{
				Name:   "monitor",
				Policy: "disable",
			},
		}
		exampleDomain.Spec.Metadata.KubeVirt.UID = "f4686d2c-6e8d-4335-b8fd-81bee22f4814"
		exampleDomain.Spec.Metadata.KubeVirt.GracePeriod = &GracePeriodMetadata{}
		exampleDomain.Spec.Metadata.KubeVirt.GracePeriod.DeletionGracePeriodSeconds = 5
		exampleDomain.Spec.IOThreads = &IOThreads{IOThreads: 2}
	})

	Context("With schema", func() {
		It("Generate expected libvirt xml", func() {
			domain := NewMinimalDomainSpec("mynamespace_testvmi")
			buf, err := xml.Marshal(domain)
			Expect(err).To(BeNil())

			newDomain := DomainSpec{}
			err = xml.Unmarshal(buf, &newDomain)
			Expect(err).To(BeNil())

			domain.XMLName.Local = "domain"
			Expect(newDomain).To(Equal(*domain))
		})
	})
	Context("With example schema", func() {
		table.DescribeTable("Unmarshal into struct", func(arch string, domainStr string) {
			NewDefaulter(arch).SetObjectDefaults_Domain(exampleDomain)
			var err error
			newDomain := DomainSpec{}
			err = xml.Unmarshal([]byte(domainStr), &newDomain)
			newDomain.XMLName.Local = ""
			newDomain.XmlNS = "http://libvirt.org/schemas/domain/qemu/1.0"
			Expect(err).To(BeNil())

			Expect(newDomain).To(Equal(exampleDomain.Spec))
		},
			table.Entry("for ppc64le", "ppc64le", exampleXMLppc64le),
			table.Entry("for amd64", "amd64", exampleXML),
		)
		table.DescribeTable("Marshal into xml", func(arch string, domainStr string) {
			NewDefaulter(arch).SetObjectDefaults_Domain(exampleDomain)
			buf, err := xml.MarshalIndent(exampleDomain.Spec, "", "  ")
			Expect(err).To(BeNil())
			Expect(string(buf)).To(Equal(domainStr))
		},
			table.Entry("for ppc64le", "ppc64le", exampleXMLppc64le),
			table.Entry("for amd64", "amd64", exampleXML),
		)
	})
	Context("With cpu pinning", func() {
		var testXML = `<cputune>
<vcpupin vcpu="0" cpuset="1"/>
<vcpupin vcpu="1" cpuset="5"/>
<iothreadpin iothread="0" cpuset="1"/>
<iothreadpin iothread="1" cpuset="5"/>
<emulatorpin cpuset="6"/>
</cputune>`
		var exampleCpuTune = CPUTune{
			VCPUPin: []CPUTuneVCPUPin{
				CPUTuneVCPUPin{
					VCPU:   0,
					CPUSet: "1",
				},
				CPUTuneVCPUPin{
					VCPU:   1,
					CPUSet: "5",
				},
			},
			IOThreadPin: []CPUTuneIOThreadPin{
				CPUTuneIOThreadPin{
					IOThread: 0,
					CPUSet:   "1",
				},
				CPUTuneIOThreadPin{
					IOThread: 1,
					CPUSet:   "5",
				},
			},
			EmulatorPin: &CPUEmulatorPin{
				CPUSet: "6",
			},
		}

		It("Unmarshal into struct", func() {
			newCpuTune := CPUTune{}
			err := xml.Unmarshal([]byte(testXML), &newCpuTune)
			Expect(err).To(BeNil())
			Expect(newCpuTune).To(Equal(exampleCpuTune))
		})
	})
	Context("With NUMA mapping", func() {
		var testXML = `<CPU><feature name="a" policy="1"></feature><numa><cell id="0" cpus="0-2" memory="1" unit="KB" memAccess="shared"></cell></numa></CPU>`
		var exampleCPU = CPU{
			Features: []CPUFeature{
				{
					Name:   "a",
					Policy: "1",
				},
			},
			NUMA: &NUMA{
				Cell: []Cell{
					{
						Id:        0,
						CPUs:      "0-2",
						Memory:    1,
						Unit:      "KB",
						MemAccess: "shared",
					},
				},
			},
		}
		It("Unmarshal into struct", func() {
			cpu := CPU{}
			err := xml.Unmarshal([]byte(testXML), &cpu)
			Expect(err).To(BeNil())
			Expect(cpu).To(Equal(exampleCPU))
		})
		It("Marshal into xml", func() {
			buf, err := xml.Marshal(exampleCPU)
			Expect(err).To(BeNil())
			fmt.Printf("%s", buf)
			Expect(string(buf)).To(Equal(testXML))
		})
	})
})
