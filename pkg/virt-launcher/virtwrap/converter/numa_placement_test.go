package converter

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"k8s.io/apimachinery/pkg/api/resource"

	v1 "kubevirt.io/client-go/api/v1"

	cmdv1 "kubevirt.io/kubevirt/pkg/handler-launcher-com/cmd/v1"
	"kubevirt.io/kubevirt/pkg/virt-launcher/virtwrap/api"
)

var _ = Describe("NumaPlacement", func() {

	var givenSpec *api.DomainSpec
	var givenVMI *v1.VirtualMachineInstance
	var givenTopology *cmdv1.Topology
	var expectedSpec *api.DomainSpec

	BeforeEach(func() {
		var err error
		givenSpec = &api.DomainSpec{
			CPUTune: &api.CPUTune{
				VCPUPin: []api.CPUTuneVCPUPin{
					{VCPU: 0, CPUSet: "10"},
					{VCPU: 1, CPUSet: "20"},
					{VCPU: 3, CPUSet: "30"},
				},
				IOThreadPin: nil,
				EmulatorPin: nil,
			},
		}
		givenSpec.Memory, err = QuantityToByte(resource.MustParse("64Mi"))
		Expect(err).ToNot(HaveOccurred())
		givenTopology = &cmdv1.Topology{
			NumaCells: []*cmdv1.Cell{
				{
					Id: 0,
					Cpus: []*cmdv1.CPU{
						{Id: 10},
						{Id: 20},
					},
				},
				{
					Id: 4,
					Cpus: []*cmdv1.CPU{
						{Id: 30},
						{Id: 50},
					},
				},
			},
		}
		expectedSpec = &api.DomainSpec{
			CPU: api.CPU{NUMA: &api.NUMA{Cells: []api.NUMACell{
				{ID: "0", CPUs: "0,1", Memory: 32, Unit: "MiB"},
				{ID: "1", CPUs: "3", Memory: 32, Unit: "MiB"},
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
					{CellID: 1, Mode: "strict", NodeSet: "4"},
				}},
		}
		givenVMI = &v1.VirtualMachineInstance{}
		memory := resource.MustParse("64Mi")
		givenVMI.Spec.Domain.Memory = &v1.Memory{Guest: &memory}
	})

	It("should map a basic valid system", func() {
		Expect(numaMapping(givenVMI, givenSpec, givenTopology)).To(Succeed())
		Expect(givenSpec.CPUTune).To(Equal(expectedSpec.CPUTune))
		Expect(givenSpec.NUMATune).To(Equal(expectedSpec.NUMATune))
		Expect(givenSpec.CPU).To(Equal(expectedSpec.CPU))
	})

	It("should detect invalid cpu pinning", func() {
		givenSpec.CPUTune.VCPUPin = append(givenSpec.CPUTune.VCPUPin, api.CPUTuneVCPUPin{
			VCPU:   4,
			CPUSet: "40",
		})
		Expect(numaMapping(givenVMI, givenSpec, givenTopology)).ToNot(Succeed())
	})

	Context("with hugepages", func() {
		var expectedMemoryBacking *api.MemoryBacking
		BeforeEach(func() {
			givenVMI.Spec.Domain.Memory.Hugepages = &v1.Hugepages{
				PageSize: "2Mi",
			}

			givenSpec.MemoryBacking = &api.MemoryBacking{
				HugePages: &api.HugePages{},
			}
			expectedMemoryBacking = &api.MemoryBacking{
				HugePages: &api.HugePages{HugePage: []api.HugePage{
					{Size: "2", Unit: "M", NodeSet: "0"},
					{Size: "2", Unit: "M", NodeSet: "1"},
				}},
			}
		})
		It("should detect hugepages and map them equally to nodes", func() {
			Expect(numaMapping(givenVMI, givenSpec, givenTopology)).To(Succeed())
			Expect(givenSpec.CPUTune).To(Equal(expectedSpec.CPUTune))
			Expect(givenSpec.NUMATune).To(Equal(expectedSpec.NUMATune))
			Expect(givenSpec.CPU).To(Equal(expectedSpec.CPU))
			Expect(givenSpec.MemoryBacking).To(Equal(expectedMemoryBacking))
		})

		It("should detect not divisable hugepages and shuffle the memory", func() {
			var err error
			givenSpec.Memory, err = QuantityToByte(resource.MustParse("66Mi"))
			givenSpec.CPUTune.VCPUPin = append(givenSpec.CPUTune.VCPUPin, api.CPUTuneVCPUPin{
				VCPU: 4, CPUSet: "40",
			})
			givenTopology.NumaCells = append(givenTopology.NumaCells, &cmdv1.Cell{
				Id: 5,
				Cpus: []*cmdv1.CPU{
					{Id: 40},
				},
			})

			expectedSpec.CPUTune.VCPUPin = append(expectedSpec.CPUTune.VCPUPin, api.CPUTuneVCPUPin{
				VCPU: 4, CPUSet: "40",
			})
			expectedSpec.NUMATune.Memory = api.NumaTuneMemory{
				Mode: "strict", NodeSet: "0,4,5",
			}
			expectedSpec.NUMATune.MemNodes = append(expectedSpec.NUMATune.MemNodes, api.MemNode{
				CellID: 2, Mode: "strict", NodeSet: "5",
			})
			expectedMemoryBacking := &api.MemoryBacking{
				HugePages: &api.HugePages{HugePage: []api.HugePage{
					{Size: "2", Unit: "M", NodeSet: "0"},
					{Size: "2", Unit: "M", NodeSet: "1"},
					{Size: "2", Unit: "M", NodeSet: "2"},
				}},
			}
			expectedSpec.CPU.NUMA.Cells = []api.NUMACell{
				{ID: "0", CPUs: "0,1", Memory: 22, Unit: "MiB"},
				{ID: "1", CPUs: "3", Memory: 22, Unit: "MiB"},
				{ID: "2", CPUs: "4", Memory: 20, Unit: "MiB"},
			}

			Expect(err).ToNot(HaveOccurred())
			Expect(numaMapping(givenVMI, givenSpec, givenTopology)).To(Succeed())
			Expect(givenSpec.CPUTune).To(Equal(expectedSpec.CPUTune))
			Expect(givenSpec.NUMATune).To(Equal(expectedSpec.NUMATune))
			Expect(givenSpec.CPU).To(Equal(expectedSpec.CPU))
			Expect(givenSpec.MemoryBacking).To(Equal(expectedMemoryBacking))
		})
	})
})
