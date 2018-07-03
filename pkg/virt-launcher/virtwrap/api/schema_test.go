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
 * Copyright 2017 Red Hat, Inc.
 *
 */

package api

import (
	"encoding/xml"

	. "github.com/onsi/ginkgo"
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
  </sysinfo>
  <devices>
    <video>
      <model type="vga" heads="1" vram="16384"></model>
    </video>
    <disk device="disk" type="network">
      <source protocol="iscsi" name="iqn.2013-07.com.example:iscsi-nopool/2">
        <host name="example.com" port="3260"></host>
      </source>
      <target dev="vda"></target>
      <driver name="qemu" type="raw"></driver>
      <alias name="mydisk"></alias>
    </disk>
    <disk device="disk" type="file">
      <source file="/var/run/libvirt/cloud-init-dir/mynamespace/testvmi/noCloud.iso"></source>
      <target dev="vdb"></target>
      <driver name="qemu" type="raw"></driver>
      <alias name="mydisk1"></alias>
    </disk>
    <console type="pty"></console>
    <watchdog model="i6300esb" action="poweroff">
      <alias name="mywatchdog"></alias>
    </watchdog>
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
  </features>
  <cpu mode="custom">
    <model>Conroe</model>
    <topology sockets="1" cores="2" threads="1"></topology>
  </cpu>
  <vcpu placement="static">2</vcpu>
</domain>`

var _ = Describe("Schema", func() {
	//The example domain should stay in sync to the xml above
	var exampleDomain = NewMinimalDomainWithNS("mynamespace", "testvmi")
	SetObjectDefaults_Domain(exampleDomain)
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
	exampleDomain.Spec.Features = &Features{
		ACPI: &FeatureEnabled{},
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
	exampleDomain.Spec.Metadata.KubeVirt.UID = "f4686d2c-6e8d-4335-b8fd-81bee22f4814"
	exampleDomain.Spec.Metadata.KubeVirt.GracePeriod.DeletionGracePeriodSeconds = 5

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
		It("Unmarshal into struct", func() {
			newDomain := DomainSpec{}
			err := xml.Unmarshal([]byte(exampleXML), &newDomain)
			newDomain.XMLName.Local = ""
			newDomain.XmlNS = "http://libvirt.org/schemas/domain/qemu/1.0"
			Expect(err).To(BeNil())

			Expect(newDomain).To(Equal(exampleDomain.Spec))
		})
		It("Marshal into xml", func() {
			buf, err := xml.MarshalIndent(exampleDomain.Spec, "", "  ")
			Expect(err).To(BeNil())
			Expect(string(buf)).To(Equal(exampleXML))
		})

	})
})
