package v1

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/rmohr/go-model"
	"github.com/satori/go.uuid"
	"kubevirt/core/pkg/api"
)

var _ = Describe("Mapper", func() {
	Context("With a VM supplied", func() {
		It("should map to v1.VM", func() {
			vm := api.VM{Name: "testvm", UUID: uuid.NewV4()}
			dto := VM{}
			errs := model.Copy(&dto, &vm)
			Expect(errs).To(BeNil())
			Expect(dto.Name).To(Equal("testvm"))
			Expect(dto.UUID).To(Equal(vm.UUID.String()))
		})
	})
	Context("With a v1.VM supplied", func() {
		It("should map to VM", func() {
			vm := api.VM{}
			dto := VM{Name: "testvm", UUID: uuid.NewV4().String()}
			errs := model.Copy(&vm, &dto)
			Expect(errs).To(BeNil())
			Expect(vm.Name).To(Equal("testvm"))
			Expect(vm.UUID).To(Equal(uuid.FromStringOrNil(dto.UUID)))
		})
	})
})
