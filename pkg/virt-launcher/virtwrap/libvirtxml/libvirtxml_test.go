package libvirtxml

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	api "kubevirt.io/kubevirt/pkg/virt-launcher/virtwrap/api"

	"libvirt.org/go/libvirtxml"
)

var _ = Describe("Convert KubeVirt domain types to Libvirtxml", func() {
	id := uint(123)

	DescribeTable("ConvertKubeVirtCPUTopologyToDomainCPUTopology", func(v *api.CPUTopology, expected *libvirtxml.DomainCPUTopology) {
		res := ConvertKubeVirtCPUTopologyToDomainCPUTopology(v)
		Expect(res).To(Equal(expected))
	},
		Entry("empty", nil, nil),
		Entry("with all the elements set",
			&api.CPUTopology{Sockets: uint32(2), Cores: uint32(2), Threads: uint32(2)},
			&libvirtxml.DomainCPUTopology{Sockets: 2, Cores: 2, Threads: 2},
		),
	)

	DescribeTable("ConvertKubeVirtVCPUToDomainVCPU", func(v *api.VCPU, expected *libvirtxml.DomainVCPU) {
		res := ConvertKubeVirtVCPUToDomainVCPU(v)
		Expect(res).To(Equal(expected))
	},
		Entry("empty", nil, nil),
		Entry("with all the elements set", &api.VCPU{Placement: "test", CPUs: uint32(2)},
			&libvirtxml.DomainVCPU{Placement: "test", Value: uint(2)}),
	)

	Context("CPU tune", func() {

		iothreadpin := []api.CPUTuneIOThreadPin{
			{IOThread: uint32(1), CPUSet: "test1"},
			{IOThread: uint32(2), CPUSet: "test2"},
		}
		diothreadpin := []libvirtxml.DomainCPUTuneIOThreadPin{
			{IOThread: uint(1), CPUSet: "test1"},
			{IOThread: uint(2), CPUSet: "test2"},
		}
		pin := []api.CPUTuneVCPUPin{
			{VCPU: uint32(1), CPUSet: "test1"},
			{VCPU: uint32(2), CPUSet: "test2"},
		}
		dpin := []libvirtxml.DomainCPUTuneVCPUPin{
			{VCPU: uint(1), CPUSet: "test1"},
			{VCPU: uint(2), CPUSet: "test2"},
		}

		DescribeTable("ConvertKubeVirtCPUTuneIOThreadPinToDomainCPUTuneIOThreadPin", func(v []api.CPUTuneIOThreadPin,
			expected []libvirtxml.DomainCPUTuneIOThreadPin) {
			res := ConvertKubeVirtCPUTuneIOThreadPinToDomainCPUTuneIOThreadPin(v)
			Expect(res).To(Equal(expected))
		},
			Entry("empty", []api.CPUTuneIOThreadPin{}, []libvirtxml.DomainCPUTuneIOThreadPin{}),
			Entry("with values", iothreadpin, diothreadpin),
		)

		DescribeTable("ConvertKubeVirtCPUTuneVCPUPinToDomainCPUTuneVCPUPin", func(v []api.CPUTuneVCPUPin,
			expected []libvirtxml.DomainCPUTuneVCPUPin) {
			res := ConvertKubeVirtCPUTuneVCPUPinToDomainCPUTuneVCPUPin(v)
			Expect(res).To(Equal(expected))
		},
			Entry("empty", []api.CPUTuneVCPUPin{}, []libvirtxml.DomainCPUTuneVCPUPin{}),
			Entry("with values", pin, dpin),
		)

		DescribeTable("ConvertKubeVirtCPUTuneToDomainCPUTune", func(v *api.CPUTune,
			expected *libvirtxml.DomainCPUTune) {
			res := ConvertKubeVirtCPUTuneToDomainCPUTune(v)
			Expect(res).To(Equal(expected))
		},
			Entry("empty", nil, nil),
			Entry("with values", &api.CPUTune{
				VCPUPin:     pin,
				IOThreadPin: iothreadpin,
				EmulatorPin: &api.CPUEmulatorPin{CPUSet: "test"},
			}, &libvirtxml.DomainCPUTune{
				VCPUPin:     dpin,
				IOThreadPin: diothreadpin,
				EmulatorPin: &libvirtxml.DomainCPUTuneEmulatorPin{CPUSet: "test"}},
			),
		)

	})

	Context("NUMA tune", func() {
		memNode := []api.MemNode{
			{CellID: uint32(123), Mode: "test1", NodeSet: "test1"},
			{CellID: uint32(321), Mode: "test2", NodeSet: "test2"},
		}
		pmemNode := []libvirtxml.DomainNUMATuneMemNode{
			{CellID: uint(123), Mode: "test1", Nodeset: "test1"},
			{CellID: uint(321), Mode: "test2", Nodeset: "test2"},
		}

		DescribeTable("ConvertKubeVirtMemNodeToDomainNUMATuneMemNode", func(v []api.MemNode, expected []libvirtxml.DomainNUMATuneMemNode) {
			res := ConvertKubeVirtMemNodeToDomainNUMATuneMemNode(v)
			Expect(res).To(Equal(expected))
		},
			Entry("empty", []api.MemNode{}, []libvirtxml.DomainNUMATuneMemNode{}),
			Entry("with values", memNode, pmemNode),
		)

		DescribeTable("ConvertKubeVirtNUMATuneToDomainNUMATune", func(v *api.NUMATune,
			expected *libvirtxml.DomainNUMATune) {
			res := ConvertKubeVirtNUMATuneToDomainNUMATune(v)
			Expect(res).To(Equal(expected))

		},
			Entry("empty", nil, nil),
			Entry("with values", &api.NUMATune{
				Memory:   api.NumaTuneMemory{Mode: "test", NodeSet: "test"},
				MemNodes: memNode,
			}, &libvirtxml.DomainNUMATune{
				Memory:   &libvirtxml.DomainNUMATuneMemory{Mode: "test", Nodeset: "test"},
				MemNodes: pmemNode,
			}),
		)

	})

	Context("MemoryBacking", func() {

		hugePage := &api.HugePages{
			HugePage: []api.HugePage{
				{Size: "1", Unit: "G", NodeSet: "test1"},
				{Size: "2", Unit: "G", NodeSet: "test2"},
			},
		}
		dhugePage := &libvirtxml.DomainMemoryHugepages{
			Hugepages: []libvirtxml.DomainMemoryHugepage{
				{Size: uint(1), Unit: "G", Nodeset: "test1"},
				{Size: uint(2), Unit: "G", Nodeset: "test2"},
			},
		}

		DescribeTable("ConvertKubeVirtHugepageToDomainMemoryHugepages", func(v *api.HugePages,
			expected *libvirtxml.DomainMemoryHugepages, expectErr string) {
			res, err := ConvertKubeVirtHugepageToDomainMemoryHugepages(v)
			if expectErr != "" {
				Expect(err).Should(MatchError(ContainSubstring(expectErr)))
				return
			}
			Expect(err).ToNot(HaveOccurred())
			Expect(res).To(Equal(expected))
		},
			Entry("empty", nil, nil, nil),
			Entry("error parsing the size", &api.HugePages{
				HugePage: []api.HugePage{{Size: "wrongid"}}},
				nil, "invalid syntax"),
			Entry("with values", hugePage, dhugePage, nil),
		)

		DescribeTable("ConvertKubeVirtMemoryBackingToDomainMemoryBacking", func(v *api.MemoryBacking,
			expected *libvirtxml.DomainMemoryBacking) {
			res, err := ConvertKubeVirtMemoryBackingToDomainMemoryBacking(v)
			Expect(err).ToNot(HaveOccurred())
			Expect(res).To(Equal(expected))
		},
			Entry("empty", nil, nil),
			Entry("with values", &api.MemoryBacking{
				HugePages:    hugePage,
				Source:       &api.MemoryBackingSource{Type: "test"},
				Access:       &api.MemoryBackingAccess{Mode: "test"},
				Allocation:   &api.MemoryAllocation{Mode: api.MemoryAllocationModeImmediate},
				NoSharePages: &api.NoSharePages{},
			}, &libvirtxml.DomainMemoryBacking{
				MemoryHugePages:    dhugePage,
				MemorySource:       &libvirtxml.DomainMemorySource{Type: "test"},
				MemoryAccess:       &libvirtxml.DomainMemoryAccess{Mode: "test"},
				MemoryAllocation:   &libvirtxml.DomainMemoryAllocation{Mode: "immediate"},
				MemoryNosharepages: &libvirtxml.DomainMemoryNosharepages{},
			}),
		)
	})

	Context("CPU with NUMA", func() {
		cellMem := uint64(123)
		cell := api.NUMACell{ID: "123", CPUs: "1", Memory: &cellMem, Unit: "G", MemoryAccess: "test"}
		dcell := libvirtxml.DomainCell{ID: &id, CPUs: "1", Memory: uint(123), Unit: "G", MemAccess: "test"}

		DescribeTable("ConvertKubeVirtNUMACellToDomainDomainCell", func(v []api.NUMACell, expected []libvirtxml.DomainCell, expectErr string) {
			res, err := ConvertKubeVirtNUMACellToDomainDomainCell(v)
			if expectErr != "" {
				Expect(err).Should(MatchError(ContainSubstring(expectErr)))
				return
			}
			Expect(err).ToNot(HaveOccurred())
			Expect(res).To(Equal(expected))
		},
			Entry("empty", []api.NUMACell{}, []libvirtxml.DomainCell{}, ""),
			Entry("error parsing the ID", []api.NUMACell{{ID: "wrongid"}}, nil, "invalid syntax"),
			Entry("set all the field", []api.NUMACell{cell}, []libvirtxml.DomainCell{dcell}, ""),
			Entry("nil memory defaults to zero", []api.NUMACell{{ID: "123", CPUs: "1", Unit: "G"}},
				[]libvirtxml.DomainCell{{ID: &id, CPUs: "1", Memory: 0, Unit: "G"}}, ""),
			Entry("with distances", []api.NUMACell{{ID: "123", CPUs: "1", Memory: &cellMem, Unit: "G", Distances: &api.NUMACellDistances{
				Siblings: []api.NUMACellSibling{{ID: "0", Value: 10}, {ID: "1", Value: 20}},
			}}}, []libvirtxml.DomainCell{{ID: &id, CPUs: "1", Memory: uint(123), Unit: "G", Distances: &libvirtxml.DomainCellDistances{
				Siblings: []libvirtxml.DomainCellSibling{{ID: 0, Value: 10}, {ID: 1, Value: 20}},
			}}}, ""),
			Entry("nil distances omitted", []api.NUMACell{cell}, []libvirtxml.DomainCell{dcell}, ""),
			Entry("invalid sibling ID", []api.NUMACell{{ID: "123", CPUs: "1", Distances: &api.NUMACellDistances{
				Siblings: []api.NUMACellSibling{{ID: "bad", Value: 10}},
			}}}, nil, "invalid sibling ID"),
		)

		DescribeTable("ConvertKubeVirtNUMAToDomainNUMA", func(v *api.NUMA, expected *libvirtxml.DomainNuma) {
			res, err := ConvertKubeVirtNUMAToDomainNUMA(v)
			Expect(err).ToNot(HaveOccurred())
			Expect(res).To(Equal(expected))
		},
			Entry("empty", nil, nil),
			Entry("with some values", &api.NUMA{Cells: []api.NUMACell{cell}},
				&libvirtxml.DomainNuma{Cell: []libvirtxml.DomainCell{dcell}}),
		)

	})
})
