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

var _ = ginkgo.Describe("LaunchSecurity SEV-SNP", func() {
	ginkgo.Context("XML marshalling and unmarshalling", func() {
		ginkgo.It("should marshal SEV-SNP launch security with all fields", func() {
			launchSecurity := &LaunchSecurity{
				Type:            "sev-snp",
				Policy:          "0x30000",
				AuthorKey:       "yes",
				VCEK:            "yes",
				IdAuth:          "test-id-auth",
				IdBlock:         "test-id-block",
				HostData:        "test-host-data",
				DHCert:          "test-dh-cert",
				Session:         "test-session",
				Cbitpos:         "51",
				ReducedPhysBits: "1",
			}

			xmlBytes, err := xml.Marshal(launchSecurity)
			Expect(err).ToNot(HaveOccurred())

			// Test that XML contains all expected attributes and elements
			xmlStr := string(xmlBytes)
			Expect(xmlStr).To(ContainSubstring(`type="sev-snp"`))
			Expect(xmlStr).To(ContainSubstring(`authorKey="yes"`))
			Expect(xmlStr).To(ContainSubstring(`vcek="yes"`))
			Expect(xmlStr).To(ContainSubstring(`<policy>0x30000</policy>`))
			Expect(xmlStr).To(ContainSubstring(`<idAuth>test-id-auth</idAuth>`))
			Expect(xmlStr).To(ContainSubstring(`<idBlock>test-id-block</idBlock>`))
			Expect(xmlStr).To(ContainSubstring(`<hostData>test-host-data</hostData>`))
			Expect(xmlStr).To(ContainSubstring(`<dhCert>test-dh-cert</dhCert>`))
			Expect(xmlStr).To(ContainSubstring(`<session>test-session</session>`))
			Expect(xmlStr).To(ContainSubstring(`<cbitpos>51</cbitpos>`))
			Expect(xmlStr).To(ContainSubstring(`<reducedPhysBits>1</reducedPhysBits>`))
		})

		ginkgo.It("should unmarshal SEV-SNP launch security with all fields", func() {
			xmlData := `<LaunchSecurity type="sev-snp" authorKey="yes" vcek="yes"><policy>0x30000</policy><idAuth>test-id-auth</idAuth><idBlock>test-id-block</idBlock><hostData>test-host-data</hostData><dhCert>test-dh-cert</dhCert><session>test-session</session><cbitpos>51</cbitpos><reducedPhysBits>1</reducedPhysBits></LaunchSecurity>`

			var launchSecurity LaunchSecurity
			err := xml.Unmarshal([]byte(xmlData), &launchSecurity)
			Expect(err).ToNot(HaveOccurred())

			Expect(launchSecurity.Type).To(Equal("sev-snp"))
			Expect(launchSecurity.Policy).To(Equal("0x30000"))
			Expect(launchSecurity.AuthorKey).To(Equal("yes"))
			Expect(launchSecurity.VCEK).To(Equal("yes"))
			Expect(launchSecurity.IdAuth).To(Equal("test-id-auth"))
			Expect(launchSecurity.IdBlock).To(Equal("test-id-block"))
			Expect(launchSecurity.HostData).To(Equal("test-host-data"))
			Expect(launchSecurity.DHCert).To(Equal("test-dh-cert"))
			Expect(launchSecurity.Session).To(Equal("test-session"))
			Expect(launchSecurity.Cbitpos).To(Equal("51"))
			Expect(launchSecurity.ReducedPhysBits).To(Equal("1"))
		})

		ginkgo.It("should marshal SEV-SNP launch security with minimal fields", func() {
			launchSecurity := &LaunchSecurity{
				Type:   "sev-snp",
				Policy: "0x30000",
			}

			xmlBytes, err := xml.Marshal(launchSecurity)
			Expect(err).ToNot(HaveOccurred())

			expectedXML := `<LaunchSecurity type="sev-snp"><policy>0x30000</policy></LaunchSecurity>`
			Expect(string(xmlBytes)).To(Equal(expectedXML))
		})

		ginkgo.It("should unmarshal SEV-SNP launch security with minimal fields", func() {
			xmlData := `<LaunchSecurity type="sev-snp"><policy>0x30000</policy></LaunchSecurity>`

			var launchSecurity LaunchSecurity
			err := xml.Unmarshal([]byte(xmlData), &launchSecurity)
			Expect(err).ToNot(HaveOccurred())

			Expect(launchSecurity.Type).To(Equal("sev-snp"))
			Expect(launchSecurity.Policy).To(Equal("0x30000"))
			Expect(launchSecurity.AuthorKey).To(BeEmpty())
			Expect(launchSecurity.VCEK).To(BeEmpty())
			Expect(launchSecurity.IdAuth).To(BeEmpty())
			Expect(launchSecurity.IdBlock).To(BeEmpty())
			Expect(launchSecurity.HostData).To(BeEmpty())
		})

		ginkgo.It("should handle empty structure", func() {
			launchSecurity := &LaunchSecurity{
				Type: "sev-snp",
			}

			xmlBytes, err := xml.Marshal(launchSecurity)
			Expect(err).ToNot(HaveOccurred())

			expectedXML := `<LaunchSecurity type="sev-snp"></LaunchSecurity>`
			Expect(string(xmlBytes)).To(Equal(expectedXML))
		})
	})

	ginkgo.Context("Domain with SEV-SNP launch security", func() {
		ginkgo.It("should marshal domain with SEV-SNP launch security", func() {
			domain := NewMinimalDomainSpec("test-domain")
			domain.LaunchSecurity = &LaunchSecurity{
				Type:         "sev-snp",
				Policy:       "0x30000",
				AuthorKey:    "yes",
				VCEK:         "yes",
				KernelHashes: "yes",
				IdAuth:       "test-id-auth",
				IdBlock:      "test-id-block",
				HostData:     "test-host-data",
			}

			xmlBytes, err := xml.Marshal(domain)
			Expect(err).ToNot(HaveOccurred())

			// Verify the XML contains the launch security section
			xmlString := string(xmlBytes)
			Expect(xmlString).To(ContainSubstring(`<launchSecurity type="sev-snp"`))
			Expect(xmlString).To(ContainSubstring(`<policy>0x30000</policy>`))
			Expect(xmlString).To(ContainSubstring(`authorKey="yes"`))
			Expect(xmlString).To(ContainSubstring(`vcek="yes"`))
			Expect(xmlString).To(ContainSubstring(`kernelHashes="yes"`))
			Expect(xmlString).To(ContainSubstring(`<idAuth>test-id-auth</idAuth>`))
			Expect(xmlString).To(ContainSubstring(`<idBlock>test-id-block</idBlock>`))
			Expect(xmlString).To(ContainSubstring(`<hostData>test-host-data</hostData>`))
		})

		ginkgo.It("should unmarshal domain with SEV-SNP launch security", func() {
			xmlData := `<domain xmlns="http://libvirt.org/schemas/domain/qemu/1.0" type="kvm">
				<name>test-domain</name>
				<memory unit="KiB">8388608</memory>
				<currentMemory unit="KiB">8388608</currentMemory>
				<vcpu placement="static">2</vcpu>
				<launchSecurity type="sev-snp" authorKey="yes" vcek="yes">
					<policy>0x30000</policy>
					<idAuth>test-id-auth</idAuth>
					<idBlock>test-id-block</idBlock>
					<hostData>test-host-data</hostData>
				</launchSecurity>
				<os>
					<type arch="x86_64" machine="pc-i440fx-2.1">hvm</type>
				</os>
				<devices>
					<emulator>/usr/bin/qemu-system-x86_64</emulator>
				</devices>
			</domain>`

			var domain DomainSpec
			err := xml.Unmarshal([]byte(xmlData), &domain)
			Expect(err).ToNot(HaveOccurred())

			Expect(domain.LaunchSecurity).ToNot(BeNil())
			Expect(domain.LaunchSecurity.Type).To(Equal("sev-snp"))
			Expect(domain.LaunchSecurity.Policy).To(Equal("0x30000"))
			Expect(domain.LaunchSecurity.AuthorKey).To(Equal("yes"))
			Expect(domain.LaunchSecurity.VCEK).To(Equal("yes"))
			Expect(domain.LaunchSecurity.IdAuth).To(Equal("test-id-auth"))
			Expect(domain.LaunchSecurity.IdBlock).To(Equal("test-id-block"))
			Expect(domain.LaunchSecurity.HostData).To(Equal("test-host-data"))
		})
	})

	ginkgo.Context("SEV vs SEV-SNP differentiation", func() {
		ginkgo.It("should handle regular SEV launch security", func() {
			launchSecurity := &LaunchSecurity{
				Type:    "sev",
				Policy:  "0x0001",
				DHCert:  "test-dh-cert",
				Session: "test-session",
			}

			xmlBytes, err := xml.Marshal(launchSecurity)
			Expect(err).ToNot(HaveOccurred())

			// Test that XML contains all expected attributes and elements
			xmlStr := string(xmlBytes)
			Expect(xmlStr).To(ContainSubstring(`type="sev"`))
			Expect(xmlStr).To(ContainSubstring(`<policy>0x0001</policy>`))
			Expect(xmlStr).To(ContainSubstring(`<dhCert>test-dh-cert</dhCert>`))
			Expect(xmlStr).To(ContainSubstring(`<session>test-session</session>`))
		})

		ginkgo.It("should differentiate between SEV and SEV-SNP types", func() {
			sevLaunchSecurity := &LaunchSecurity{
				Type:   "sev",
				Policy: "0x0001",
			}

			sevSnpLaunchSecurity := &LaunchSecurity{
				Type:   "sev-snp",
				Policy: "0x30000",
			}

			sevXML, err := xml.Marshal(sevLaunchSecurity)
			Expect(err).ToNot(HaveOccurred())
			Expect(string(sevXML)).To(ContainSubstring(`type="sev"`))
			Expect(string(sevXML)).ToNot(ContainSubstring(`type="sev-snp"`))

			sevSnpXML, err := xml.Marshal(sevSnpLaunchSecurity)
			Expect(err).ToNot(HaveOccurred())
			Expect(string(sevSnpXML)).To(ContainSubstring(`type="sev-snp"`))
			Expect(string(sevSnpXML)).ToNot(ContainSubstring(`type="sev"`))
		})
	})
})
