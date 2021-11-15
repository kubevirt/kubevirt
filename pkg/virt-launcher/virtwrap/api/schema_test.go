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
	"encoding/json"
	"encoding/xml"
	"fmt"

	. "github.com/onsi/ginkgo"
	"github.com/onsi/ginkgo/extensions/table"
	. "github.com/onsi/gomega"
)

var exampleXMLwithNoneMemballoon string
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
    %s
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
    <kvm>
      <hidden state="on"></hidden>
      <hint-dedicated state="on"></hint-dedicated>
    </kvm>
    <pvspinlock state="off"></pvspinlock>
    <pmu state="off"></pmu>
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

var exampleXMLppc64lewithNoneMemballoon string
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
    %s
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
    <kvm>
      <hidden state="on"></hidden>
      <hint-dedicated state="on"></hint-dedicated>
    </kvm>
    <pvspinlock state="off"></pvspinlock>
    <pmu state="off"></pmu>
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

// TODO: Make the XML fit for real arm64 configuration
var exampleXMLarm64withNoneMemballoon string
var exampleXMLarm64 = `<domain type="kvm" xmlns:qemu="http://libvirt.org/schemas/domain/qemu/1.0">
  <name>mynamespace_testvmi</name>
  <memory unit="MB">9</memory>
  <os>
    <type arch="aarch64" machine="virt">hvm</type>
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
    %s
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
    <kvm>
      <hidden state="on"></hidden>
      <hint-dedicated state="on"></hint-dedicated>
    </kvm>
    <pvspinlock state="off"></pvspinlock>
    <pmu state="off"></pmu>
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
	exampleXMLwithNoneMemballoon = fmt.Sprintf(exampleXML,
		`<memballoon model="none"></memballoon>`)
	exampleXML = fmt.Sprintf(exampleXML,
		`<memballoon model="virtio">
      <stats period="10"></stats>
    </memballoon>`)

	exampleXMLppc64lewithNoneMemballoon = fmt.Sprintf(exampleXMLppc64le,
		`<memballoon model="none"></memballoon>`)
	exampleXMLppc64le = fmt.Sprintf(exampleXMLppc64le,
		`<memballoon model="virtio">
      <stats period="10"></stats>
    </memballoon>`)

	exampleXMLarm64withNoneMemballoon = fmt.Sprintf(exampleXMLarm64,
		`<memballoon model="none"></memballoon>`)
	exampleXMLarm64 = fmt.Sprintf(exampleXMLarm64,
		`<memballoon model="virtio">
      <stats period="10"></stats>
    </memballoon>`)

	//The example domain should stay in sync to the xml above
	var exampleDomain *Domain
	var exampleDomainWithMemballonDevice *Domain

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
				Alias:  NewUserDefinedAlias("mydisk"),
			},
			{Type: "file",
				Device: "disk",
				Driver: &DiskDriver{Name: "qemu",
					Type: "raw"},
				Source: DiskSource{
					File: "/var/run/libvirt/cloud-init-dir/mynamespace/testvmi/noCloud.iso",
				},
				Target: DiskTarget{Device: "vdb"},
				Alias:  NewUserDefinedAlias("mydisk1"),
			},
			{Type: "block",
				Device: "disk",
				Driver: &DiskDriver{Name: "qemu",
					Type: "raw"},
				Source: DiskSource{
					Dev: "/dev/testdev",
				},
				Target: DiskTarget{Device: "vdc"},
				Alias:  NewUserDefinedAlias("mydisk2"),
			},
		}

		exampleDomain.Spec.Devices.Inputs = []Input{
			{
				Type:  "tablet",
				Bus:   "virtio",
				Alias: NewUserDefinedAlias("tablet0"),
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
			Alias:  NewUserDefinedAlias("mywatchdog"),
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
			KVM: &FeatureKVM{
				Hidden:        &FeatureState{State: "on"},
				HintDedicated: &FeatureState{State: "on"},
			},
			PVSpinlock: &FeaturePVSpinlock{State: "off"},
			PMU:        &FeatureState{State: "off"},
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
		exampleDomain.Spec.Devices.Ballooning = &MemBalloon{Model: "virtio", Stats: &Stats{Period: 10}}
		exampleDomainWithMemballonDevice = exampleDomain.DeepCopy()
		exampleDomainWithMemballonDevice.Spec.Devices.Ballooning = &MemBalloon{Model: "none"}
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
	unmarshalTest := func(arch, domainStr string, domain *Domain) {
		NewDefaulter(arch).SetObjectDefaults_Domain(domain)
		newDomain := DomainSpec{}
		err := xml.Unmarshal([]byte(domainStr), &newDomain)
		newDomain.XMLName.Local = ""
		newDomain.XmlNS = "http://libvirt.org/schemas/domain/qemu/1.0"
		Expect(err).To(BeNil())

		Expect(newDomain).To(Equal(domain.Spec))
	}

	marshalTest := func(arch, domainStr string, domain *Domain) {
		NewDefaulter(arch).SetObjectDefaults_Domain(domain)
		buf, err := xml.MarshalIndent(domain.Spec, "", "  ")
		Expect(err).To(BeNil())
		Expect(string(buf)).To(Equal(domainStr))
	}
	Context("With example schema", func() {
		table.DescribeTable("Unmarshal into struct", func(arch string, domainStr string) {
			unmarshalTest(arch, domainStr, exampleDomain)
		},
			table.Entry("for ppc64le", "ppc64le", exampleXMLppc64le),
			table.Entry("for arm64", "arm64", exampleXMLarm64),
			table.Entry("for amd64", "amd64", exampleXML),
		)
		table.DescribeTable("Marshal into xml", func(arch string, domainStr string) {
			marshalTest(arch, domainStr, exampleDomain)
		},
			table.Entry("for ppc64le", "ppc64le", exampleXMLppc64le),
			table.Entry("for arm64", "arm64", exampleXMLarm64),
			table.Entry("for amd64", "amd64", exampleXML),
		)

		table.DescribeTable("Unmarshal into struct", func(arch string, domainStr string) {
			unmarshalTest(arch, domainStr, exampleDomainWithMemballonDevice)
		},
			table.Entry("for ppc64le and Memballoon device is specified", "ppc64le", exampleXMLppc64lewithNoneMemballoon),
			table.Entry("for arm64 and Memballoon device is specified", "arm64", exampleXMLarm64withNoneMemballoon),
			table.Entry("for amd64 and Memballoon device is specified", "amd64", exampleXMLwithNoneMemballoon),
		)
		table.DescribeTable("Marshal into xml", func(arch string, domainStr string) {
			marshalTest(arch, domainStr, exampleDomainWithMemballonDevice)
		},
			table.Entry("for ppc64le and Memballoon device is specified", "ppc64le", exampleXMLppc64lewithNoneMemballoon),
			table.Entry("for arm64 and Memballoon device is specified", "arm64", exampleXMLarm64withNoneMemballoon),
			table.Entry("for amd64 and Memballoon device is specified", "amd64", exampleXMLwithNoneMemballoon),
		)
	})

	Context("With numa topology", func() {
		It("should marshal and unmarshal the values", func() {
			var testXML = `
<domain>
<cputune>
	<vcpupin vcpu="0" cpuset="1"/>
	<vcpupin vcpu="1" cpuset="5"/>
	<vcpupin vcpu="2" cpuset="2"/>
	<vcpupin vcpu="3" cpuset="6"/>
</cputune>
<numatune>
  <memory mode="strict" nodeset="1-2"/> 
  <memnode cellid="0" mode="strict" nodeset="1"/>
  <memnode cellid="2" mode="preferred" nodeset="2"/>
</numatune>
<cpu>
	<numa>
		<cell id="0" cpus="0-1" memory="3" unit="GiB"/>
		<cell id="1" cpus="2-3" memory="3" unit="GiB"/>
	</numa>
</cpu>
</domain>
`
			spec := &DomainSpec{}
			expectedSpec := &DomainSpec{
				CPU: CPU{NUMA: &NUMA{Cells: []NUMACell{
					{ID: "0", CPUs: "0-1", Memory: 3, Unit: "GiB"},
					{ID: "1", CPUs: "2-3", Memory: 3, Unit: "GiB"},
				}}},
				CPUTune: &CPUTune{
					VCPUPin: []CPUTuneVCPUPin{
						{VCPU: 0, CPUSet: "1"},
						{VCPU: 1, CPUSet: "5"},
						{VCPU: 2, CPUSet: "2"},
						{VCPU: 3, CPUSet: "6"},
					},
				},
				NUMATune: &NUMATune{
					Memory: NumaTuneMemory{
						Mode:    "strict",
						NodeSet: "1-2",
					},
					MemNodes: []MemNode{
						{CellID: 0, Mode: "strict", NodeSet: "1"},
						{CellID: 2, Mode: "preferred", NodeSet: "2"},
					},
				},
			}
			Expect(xml.Unmarshal([]byte(testXML), spec)).To(Succeed())
			Expect(spec.NUMATune).To(Equal(expectedSpec.NUMATune))
			Expect(spec.CPUTune).To(Equal(expectedSpec.CPUTune))
			Expect(spec.CPU).To(Equal(expectedSpec.CPU))
		})

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
				{
					VCPU:   0,
					CPUSet: "1",
				},
				{
					VCPU:   1,
					CPUSet: "5",
				},
			},
			IOThreadPin: []CPUTuneIOThreadPin{
				{
					IOThread: 0,
					CPUSet:   "1",
				},
				{
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
})

var testAliasName = "alias0"
var exampleXMLnonUserDefinedAlias = `<Alias name="alias0"></Alias>`
var exampleXMLUserDefinedAlias = `<Alias name="ua-alias0"></Alias>`
var exampleJSONnonUserDefinedAlias = `{"Name":"alias0","UserDefined":false}`
var exampleJSONuserDefinedAlias = `{"Name":"alias0","UserDefined":true}`

func newLibvirtManagedAlias(aliasName string) *Alias {
	return &Alias{name: aliasName}
}

var _ = Describe("XML marshal of domain device", func() {
	It("should not add user alias prefix to the name of a non-user-defined alias", func() {
		alias := newLibvirtManagedAlias(testAliasName)
		xmlBytes, err := xml.Marshal(alias)
		Expect(err).ToNot(HaveOccurred())
		Expect(string(xmlBytes)).To(Equal(exampleXMLnonUserDefinedAlias))
		newAlias := &Alias{}
		Expect(xml.Unmarshal(xmlBytes, newAlias)).To(Succeed())
		Expect(newAlias.GetName()).To(Equal(testAliasName))
		Expect(newAlias.IsUserDefined()).To(BeFalse())
	})
	It("should add user alias prefix to the name of a user-defined alias", func() {
		alias := NewUserDefinedAlias(testAliasName)
		xmlBytes, err := xml.Marshal(alias)
		Expect(err).ToNot(HaveOccurred())
		Expect(string(xmlBytes)).To(Equal(exampleXMLUserDefinedAlias))
		newAlias := &Alias{}
		Expect(xml.Unmarshal(xmlBytes, newAlias)).To(Succeed())
		Expect(newAlias.GetName()).To(Equal(testAliasName))
		Expect(newAlias.IsUserDefined()).To(BeTrue())
	})
})

var _ = Describe("JSON marshal of the alias of a domain device", func() {
	It("should deal with package-private struct members for non-user-defined alias", func() {
		alias := newLibvirtManagedAlias(testAliasName)
		jsonBytes, err := json.Marshal(alias)
		Expect(err).ToNot(HaveOccurred())
		Expect(string(jsonBytes)).To(Equal(exampleJSONnonUserDefinedAlias))
		newAlias := &Alias{}
		Expect(json.Unmarshal(jsonBytes, newAlias)).To(Succeed())
		Expect(newAlias.GetName()).To(Equal(testAliasName))
		Expect(newAlias.IsUserDefined()).To(BeFalse())
	})
	It("should deal with package-private struct members for user-defined alias", func() {
		alias := NewUserDefinedAlias(testAliasName)
		jsonBytes, err := json.Marshal(alias)
		Expect(err).ToNot(HaveOccurred())
		Expect(string(jsonBytes)).To(Equal(exampleJSONuserDefinedAlias))
		newAlias := &Alias{}
		Expect(json.Unmarshal(jsonBytes, newAlias)).To(Succeed())
		Expect(newAlias.GetName()).To(Equal(testAliasName))
		Expect(newAlias.IsUserDefined()).To(BeTrue())
	})
})
