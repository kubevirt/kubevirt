package libvirtxml

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	api "kubevirt.io/kubevirt/pkg/virt-launcher/virtwrap/api"

	"libvirt.org/go/libvirtxml"
)

var _ = Describe("Convert KubeVirt domain types to Libvirtxml", func() {
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

})
