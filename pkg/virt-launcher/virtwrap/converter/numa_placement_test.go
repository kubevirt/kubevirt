package converter

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	cmdv1 "kubevirt.io/kubevirt/pkg/handler-launcher-com/cmd/v1"
	"kubevirt.io/kubevirt/pkg/virt-launcher/virtwrap/api"
)

var _ = Describe("NumaPlacement", func() {

	var givenSpec *api.DomainSpec
	var givenTopology *cmdv1.Topology
	var expectedSpec *api.DomainSpec

	BeforeEach(func() {
		givenSpec = &api.DomainSpec{
			Memory: api.Memory{
				Value: 1234,
				Unit:  "MiB",
			},
			CPUTune: &api.CPUTune{
				VCPUPin: []api.CPUTuneVCPUPin{
					{
						VCPU:   0,
						CPUSet: "10",
					},
					{
						VCPU:   1,
						CPUSet: "20",
					},
					{
						VCPU:   3,
						CPUSet: "30",
					},
				},
				IOThreadPin: nil,
				EmulatorPin: nil,
			},
		}
		givenTopology = &cmdv1.Topology{
			NumaCells: []*cmdv1.Cell{
				{
					Id: 0,
					Cpus: []*cmdv1.CPU{
						{
							Id: 10,
						},
						{
							Id: 20,
						},
					},
				},
				{
					Id: 4,
					Cpus: []*cmdv1.CPU{
						{
							Id: 30,
						},
						{
							Id: 50,
						},
					},
				},
			},
		}
		expectedSpec = &api.DomainSpec{
			CPU: api.CPU{NUMA: &api.NUMA{Cells: []api.NUMACell{
				{ID: "0", CPUs: "0,1", Memory: 617, Unit: "MiB"},
				{ID: "3", CPUs: "3", Memory: 617, Unit: "MiB"},
			}}},
			CPUTune: &api.CPUTune{VCPUPin: []api.CPUTuneVCPUPin{
				{VCPU: 0, CPUSet: "10"},
				{VCPU: 1, CPUSet: "20"},
				{VCPU: 3, CPUSet: "30"},
			}},
			NUMATune: &api.NUMATune{
				Memory: api.NumaTuneMemory{Mode: "strict", NodeSet: "0,4"},
				MemNodes: []api.MemNode{
					{CellID: 0, Mode: "strict", NodeSet: "0"},
					{CellID: 3, Mode: "strict", NodeSet: "4"},
				}},
		}
	})

	It("should map a basic valid system", func() {
		Expect(numaMapping(givenSpec, givenTopology)).To(Succeed())
		Expect(givenSpec.CPUTune).To(Equal(expectedSpec.CPUTune))
		Expect(givenSpec.NUMATune).To(Equal(expectedSpec.NUMATune))
		Expect(givenSpec.CPU).To(Equal(expectedSpec.CPU))
	})

	It("should detect invalid cpu pinning", func() {
		givenSpec.CPUTune.VCPUPin = append(givenSpec.CPUTune.VCPUPin, api.CPUTuneVCPUPin{
			VCPU:   4,
			CPUSet: "40",
		})
		Expect(numaMapping(givenSpec, givenTopology)).ToNot(Succeed())
	})

})
