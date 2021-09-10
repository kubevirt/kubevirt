package nodelabeller

import (
	"encoding/xml"
	"os"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"kubevirt.io/kubevirt/pkg/virt-handler/node-labeller/api"
)

var _ = Describe("Capabilities", func() {

	It("should be able to read the TSC timer freqemency from the host", func() {
		f, err := os.Open("testdata/capabilities.xml")
		Expect(err).ToNot(HaveOccurred())
		defer f.Close()
		capabilities := &api.Capabilities{}
		Expect(xml.NewDecoder(f).Decode(capabilities)).To(Succeed())
		Expect(capabilities.Host.CPU.Counter).To(HaveLen(1))
		Expect(capabilities.Host.CPU.Counter[0].Name).To(Equal("tsc"))
		Expect(capabilities.Host.CPU.Counter[0].Frequency).To(BeNumerically("==", 4008012000))
		Expect(bool(capabilities.Host.CPU.Counter[0].Scaling)).To(BeFalse())
		counter, err := capabilities.GetTSCCounter()
		Expect(err).ToNot(HaveOccurred())
		Expect(counter.Frequency).To(BeNumerically("==", 4008012000))
		Expect(bool(counter.Scaling)).To(BeFalse())
		Expect(capabilities.Host.Topology.Cells.Cell[0].Cpus.CPU[7].Siblings).To(HaveLen(29))
	})

	It("should properly read cpu siblings", func() {
		f, err := os.Open("testdata/capabilities.xml")
		Expect(err).ToNot(HaveOccurred())
		defer f.Close()
		capabilities := &api.Capabilities{}
		Expect(xml.NewDecoder(f).Decode(capabilities)).To(Succeed())
		Expect(capabilities.Host.Topology.Cells.Cell).To(HaveLen(1))
		Expect(capabilities.Host.Topology.Cells.Cell[0].Cpus.CPU).To(HaveLen(8))
		Expect(capabilities.Host.Topology.Cells.Cell[0].Cpus.CPU[0].Siblings).To(ConsistOf(uint32(0), uint32(4)))
	})

	It("should read the numa topology from the host", func() {

		expectedCell := api.Cell{
			ID: 0,
			Memory: api.Memory{
				Amount: 1289144,
				Unit:   "KiB",
			},
			Pages: []api.Pages{
				{
					Count: 314094,
					Unit:  "KiB",
					Size:  4,
				},
				{
					Count: 16,
					Unit:  "KiB",
					Size:  2048,
				},
				{
					Count: 0,
					Unit:  "KiB",
					Size:  1048576,
				},
			},
			Distances: api.Distances{
				Sibling: []api.Sibling{
					{
						ID:    0,
						Value: 10,
					},
					{
						ID:    1,
						Value: 10,
					},
					{
						ID:    2,
						Value: 10,
					},
					{
						ID:    3,
						Value: 10,
					},
				},
			},
			Cpus: api.CPUs{
				Num: 6,
				CPU: []api.CPU{
					{
						ID:       0,
						SocketID: 0,
						DieID:    0,
						CoreID:   0,
						Siblings: []uint32{0},
					},
					{
						ID:       1,
						SocketID: 1,
						DieID:    0,
						CoreID:   0,
						Siblings: []uint32{1},
					},
					{
						ID:       2,
						SocketID: 2,
						DieID:    0,
						CoreID:   0,
						Siblings: []uint32{2},
					},
					{
						ID:       3,
						SocketID: 3,
						DieID:    0,
						CoreID:   0,
						Siblings: []uint32{3},
					},
					{
						ID:       4,
						SocketID: 4,
						DieID:    0,
						CoreID:   0,
						Siblings: []uint32{4},
					},
					{
						ID:       5,
						SocketID: 5,
						DieID:    0,
						CoreID:   0,
						Siblings: []uint32{5},
					},
				},
			},
		}

		f, err := os.Open("testdata/capabilities_with_numa.xml")
		Expect(err).ToNot(HaveOccurred())
		defer f.Close()
		capabilities := &api.Capabilities{}
		Expect(xml.NewDecoder(f).Decode(capabilities)).To(Succeed())
		Expect(capabilities.Host.Topology.Cells.Num).To(BeNumerically("==", 4))
		Expect(capabilities.Host.Topology.Cells.Cell).To(HaveLen(4))
		Expect(capabilities.Host.Topology.Cells.Cell[0]).To(Equal(expectedCell))
	})
})
