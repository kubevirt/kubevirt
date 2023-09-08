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
	"bytes"
	_ "embed"
	"encoding/json"
	"encoding/xml"
	"strings"
	"text/template"

	ginkgo "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	v1 "kubevirt.io/api/core/v1"
)

var (
	//go:embed testdata/domain_x86_64.xml.tmpl
	exampleXML                   string
	exampleXMLwithNoneMemballoon string

	//go:embed testdata/domain_ppc64le.xml.tmpl
	exampleXMLppc64le                   string
	exampleXMLppc64lewithNoneMemballoon string

	//go:embed testdata/domain_arm64.xml.tmpl
	exampleXMLarm64                   string
	exampleXMLarm64withNoneMemballoon string

	//go:embed testdata/domain_numa_topology.xml
	domainNumaTopology []byte

	//go:embed testdata/cpu_pinning.xml
	cpuPinningXML []byte
)

const (
	argNoMemBalloon     = `<memballoon model="none"></memballoon>`
	argMemBalloonVirtio = `<memballoon model="virtio">
      <stats period="10"></stats>
    </memballoon>`
)

var _ = ginkgo.Describe("Schema", func() {
	templateToString := func(templateStr, templateInput string) string {
		tmpl, err := template.New("schema").Parse(templateStr)
		Expect(err).ToNot(HaveOccurred())
		buf := bytes.Buffer{}
		err = tmpl.Execute(&buf, templateInput)
		Expect(err).ToNot(HaveOccurred())
		return strings.TrimSpace(buf.String())
	}

	exampleXMLwithNoneMemballoon = templateToString(exampleXML, argNoMemBalloon)
	exampleXML = templateToString(exampleXML, argMemBalloonVirtio)

	exampleXMLppc64lewithNoneMemballoon = templateToString(exampleXMLppc64le, argNoMemBalloon)
	exampleXMLppc64le = templateToString(exampleXMLppc64le, argMemBalloonVirtio)

	exampleXMLarm64withNoneMemballoon = templateToString(exampleXMLarm64, argNoMemBalloon)
	exampleXMLarm64 = templateToString(exampleXMLarm64, argMemBalloonVirtio)

	//The example domain should stay in sync to the xml above
	var exampleDomain *Domain
	var exampleDomainWithMemballonDevice *Domain

	ginkgo.BeforeEach(func() {
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
				Bus:   v1.VirtIO,
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
		exampleDomain.Spec.Devices.Watchdogs = []Watchdog{
			{
				Model:  "i6300esb",
				Action: "poweroff",
				Alias:  NewUserDefinedAlias("mywatchdog"),
			},
		}
		exampleDomain.Spec.Devices.Rng = &Rng{
			Model:   v1.VirtIO,
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
		exampleDomain.Spec.Devices.Ballooning = &MemBalloon{Model: v1.VirtIO, Stats: &Stats{Period: 10}}
		exampleDomainWithMemballonDevice = exampleDomain.DeepCopy()
		exampleDomainWithMemballonDevice.Spec.Devices.Ballooning = &MemBalloon{Model: "none"}
	})

	ginkgo.Context("With schema", func() {
		ginkgo.It("Generate expected libvirt xml", func() {
			domain := NewMinimalDomainSpec("mynamespace_testvmi")
			buf, err := xml.Marshal(domain)
			Expect(err).ToNot(HaveOccurred())

			newDomain := DomainSpec{}
			Expect(xml.Unmarshal(buf, &newDomain)).To(Succeed())

			domain.XMLName.Local = "domain"
			Expect(newDomain).To(Equal(*domain))
		})
	})
	unmarshalTest := func(arch, domainStr string, domain *Domain) {
		NewDefaulter(arch).SetObjectDefaults_Domain(domain)
		newDomain := DomainSpec{}
		Expect(xml.Unmarshal([]byte(domainStr), &newDomain)).To(Succeed())
		newDomain.XMLName.Local = ""
		newDomain.XmlNS = "http://libvirt.org/schemas/domain/qemu/1.0"

		Expect(newDomain).To(Equal(domain.Spec))
	}

	marshalTest := func(arch, domainStr string, domain *Domain) {
		NewDefaulter(arch).SetObjectDefaults_Domain(domain)
		buf, err := xml.MarshalIndent(domain.Spec, "", "  ")
		Expect(err).ToNot(HaveOccurred())
		Expect(string(buf)).To(Equal(domainStr))
	}
	ginkgo.Context("With example schema", func() {
		ginkgo.DescribeTable("Unmarshal into struct", func(arch string, domainStr string) {
			unmarshalTest(arch, domainStr, exampleDomain)
		},
			ginkgo.Entry("for ppc64le", "ppc64le", exampleXMLppc64le),
			ginkgo.Entry("for arm64", "arm64", exampleXMLarm64),
			ginkgo.Entry("for amd64", "amd64", exampleXML),
		)
		ginkgo.DescribeTable("Marshal into xml", func(arch string, domainStr string) {
			marshalTest(arch, domainStr, exampleDomain)
		},
			ginkgo.Entry("for ppc64le", "ppc64le", exampleXMLppc64le),
			ginkgo.Entry("for arm64", "arm64", exampleXMLarm64),
			ginkgo.Entry("for amd64", "amd64", exampleXML),
		)

		ginkgo.DescribeTable("Unmarshal into struct", func(arch string, domainStr string) {
			unmarshalTest(arch, domainStr, exampleDomainWithMemballonDevice)
		},
			ginkgo.Entry("for ppc64le and Memballoon device is specified", "ppc64le", exampleXMLppc64lewithNoneMemballoon),
			ginkgo.Entry("for arm64 and Memballoon device is specified", "arm64", exampleXMLarm64withNoneMemballoon),
			ginkgo.Entry("for amd64 and Memballoon device is specified", "amd64", exampleXMLwithNoneMemballoon),
		)
		ginkgo.DescribeTable("Marshal into xml", func(arch string, domainStr string) {
			marshalTest(arch, domainStr, exampleDomainWithMemballonDevice)
		},
			ginkgo.Entry("for ppc64le and Memballoon device is specified", "ppc64le", exampleXMLppc64lewithNoneMemballoon),
			ginkgo.Entry("for arm64 and Memballoon device is specified", "arm64", exampleXMLarm64withNoneMemballoon),
			ginkgo.Entry("for amd64 and Memballoon device is specified", "amd64", exampleXMLwithNoneMemballoon),
		)
	})

	ginkgo.Context("With numa topology", func() {
		ginkgo.It("should marshal and unmarshal the values", func() {
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
			Expect(xml.Unmarshal(domainNumaTopology, spec)).To(Succeed())
			Expect(spec.NUMATune).To(Equal(expectedSpec.NUMATune))
			Expect(spec.CPUTune).To(Equal(expectedSpec.CPUTune))
			Expect(spec.CPU).To(Equal(expectedSpec.CPU))
		})

	})

	ginkgo.Context("With cpu pinning", func() {
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

		ginkgo.It("Unmarshal into struct", func() {
			newCpuTune := CPUTune{}
			Expect(xml.Unmarshal(cpuPinningXML, &newCpuTune)).To(Succeed())
			Expect(newCpuTune).To(Equal(exampleCpuTune))
		})
	})

	ginkgo.Context("With vsock", func() {
		ginkgo.It("Generate expected libvirt xml", func() {
			domain := NewMinimalDomainSpec("mynamespace_testvmi")
			domain.Devices.VSOCK = &VSOCK{
				Model: "virtio",
				CID: CID{
					Auto:    "no",
					Address: 3,
				},
			}
			buf, err := xml.Marshal(domain)
			Expect(err).ToNot(HaveOccurred())

			newDomain := DomainSpec{}
			err = xml.Unmarshal(buf, &newDomain)
			Expect(err).ToNot(HaveOccurred())

			domain.XMLName.Local = "domain"
			Expect(newDomain).To(Equal(*domain))
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

var _ = ginkgo.Describe("XML marshal of domain device", func() {
	ginkgo.It("should not add user alias prefix to the name of a non-user-defined alias", func() {
		alias := newLibvirtManagedAlias(testAliasName)
		xmlBytes, err := xml.Marshal(alias)
		Expect(err).ToNot(HaveOccurred())
		Expect(string(xmlBytes)).To(Equal(exampleXMLnonUserDefinedAlias))
		newAlias := &Alias{}
		Expect(xml.Unmarshal(xmlBytes, newAlias)).To(Succeed())
		Expect(newAlias.GetName()).To(Equal(testAliasName))
		Expect(newAlias.IsUserDefined()).To(BeFalse())
	})
	ginkgo.It("should add user alias prefix to the name of a user-defined alias", func() {
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

var _ = ginkgo.Describe("JSON marshal of the alias of a domain device", func() {
	ginkgo.It("should deal with package-private struct members for non-user-defined alias", func() {
		alias := newLibvirtManagedAlias(testAliasName)
		jsonBytes, err := json.Marshal(alias)
		Expect(err).ToNot(HaveOccurred())
		Expect(string(jsonBytes)).To(Equal(exampleJSONnonUserDefinedAlias))
		newAlias := &Alias{}
		Expect(json.Unmarshal(jsonBytes, newAlias)).To(Succeed())
		Expect(newAlias.GetName()).To(Equal(testAliasName))
		Expect(newAlias.IsUserDefined()).To(BeFalse())
	})
	ginkgo.It("should deal with package-private struct members for user-defined alias", func() {
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
