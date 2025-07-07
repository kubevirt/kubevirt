package vsock

import (
	"fmt"
	"math"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	virtv1 "kubevirt.io/api/core/v1"

	"kubevirt.io/kubevirt/pkg/controller"
	"kubevirt.io/kubevirt/tests/libvmifact"
)

var _ = Describe("VSOCK", func() {
	var m *cidsMap

	newRandomVMIsWithORWithoutVSOCK := func(totalVMINum, vsockVMINum int) []*virtv1.VirtualMachineInstance {
		var vmis []*virtv1.VirtualMachineInstance
		for i := 0; i < totalVMINum; i++ {
			vmi := libvmifact.NewAlpine()
			// NewAlpine isn't guaranteed to have a unique name.
			vmi.Name = fmt.Sprintf("%s-%d", vmi.Name, i)
			vmi.Namespace = "vsock"
			if i < vsockVMINum {
				cid := uint32(i + 3)
				vmi.Status.VSOCKCID = &cid
			}
			vmis = append(vmis, vmi)
		}
		return vmis
	}

	BeforeEach(func() {
		m = NewCIDsMap()
	})

	DescribeTable("Syncing CIDs from exsisting VMIs",
		func(totalVMINum, vsockVMINum int) {
			vmis := newRandomVMIsWithORWithoutVSOCK(totalVMINum, vsockVMINum)
			m.Sync(vmis)
			Expect(m.cids).To(HaveLen(vsockVMINum))
			Expect(m.reverse).To(HaveLen(vsockVMINum))
			for _, vmi := range vmis {
				if vmi.Status.VSOCKCID == nil {
					Expect(m.cids).NotTo(HaveKey(controller.VirtualMachineInstanceKey(vmi)))
				} else {
					Expect(m.cids).To(HaveKey(controller.VirtualMachineInstanceKey(vmi)))
					Expect(m.reverse).To(HaveKey(*vmi.Status.VSOCKCID))
				}
			}
		},
		Entry("When all VMIs have VSOCK enabled", 3, 3),
		Entry("When some VMIs have VSOCK enabled", 5, 3),
	)

	DescribeTable("Allocating CIDs for VMIs",
		func(num int) {
			seen := make(map[uint32]bool)
			vmis := newRandomVMIsWithORWithoutVSOCK(num, 0)
			for _, vmi := range vmis {
				Expect(m.Allocate(vmi)).To(Succeed())
				Expect(vmi.Status.VSOCKCID).NotTo(BeNil())
				gotCID := *vmi.Status.VSOCKCID
				Expect(gotCID).To(BeNumerically(">=", 3))
				Expect(seen).NotTo(HaveKey(gotCID))
				seen[gotCID] = true
			}
			Expect(m.cids).To(HaveLen(len(vmis)))
			Expect(m.reverse).To(HaveLen(len(vmis)))
		},
		Entry("When there are 5 VMIs", 5),
		Entry("When there are 100 VMIs", 100),
		Entry("When there are 1000 VMIs", 1000),
	)

	It("should remove CIDs for VMIs", func() {
		vmis := newRandomVMIsWithORWithoutVSOCK(5, 5)
		m.Sync(vmis)
		Expect(m.cids).To(HaveLen(len(vmis)))
		Expect(m.reverse).To(HaveLen(len(vmis)))
		for _, vmi := range vmis {
			key, err := controller.KeyFunc(vmi)
			Expect(err).NotTo(HaveOccurred())
			m.Remove(key)
		}
		Expect(m.cids).To(BeEmpty())
		Expect(m.reverse).To(BeEmpty())
	})

	Context("CIDs iteration", func() {
		It("should wrap arround if reaches the maximum", func() {
			m.randCID = func() uint32 { return math.MaxUint32 }
			vmi := libvmifact.NewAlpine()
			Expect(m.Allocate(vmi)).To(Succeed())
			Expect(vmi.Status.VSOCKCID).NotTo(BeNil())
			Expect(*vmi.Status.VSOCKCID).To(BeNumerically("==", math.MaxUint32))

			vmi2 := libvmifact.NewAlpine()
			Expect(m.Allocate(vmi2)).To(Succeed())
			Expect(vmi2.Status.VSOCKCID).NotTo(BeNil())
			Expect(*vmi2.Status.VSOCKCID).To(BeNumerically("==", 3))
		})

		It("should return error if CIDs are exhausted", func() {
			// Simulate only 3, 4, 5 are allocatable.
			// next(3) = 4, next(4) = 5, next(6) = 3
			m.randCID = func() uint32 { return 3 }
			m.nextCID = func(u uint32) uint32 { return (u+1)%3 + 3 }

			vmis := newRandomVMIsWithORWithoutVSOCK(3, 0)
			for _, vmi := range vmis {
				Expect(m.Allocate(vmi)).To(Succeed())
				Expect(vmi.Status.VSOCKCID).NotTo(BeNil())
				Expect(*vmi.Status.VSOCKCID).To(BeNumerically(">=", 3))
			}
			Expect(m.cids).To(HaveLen(len(vmis)))
			Expect(m.reverse).To(HaveLen(len(vmis)))

			// The next one will fail
			vmi := libvmifact.NewAlpine()
			Expect(m.Allocate(vmi)).NotTo(Succeed())
		})
	})
})
