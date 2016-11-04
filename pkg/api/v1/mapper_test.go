package v1

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/rmohr/go-model"
	"github.com/satori/go.uuid"
	kubeapi "k8s.io/client-go/1.5/pkg/api"
	"k8s.io/client-go/1.5/pkg/types"
	"kubevirt/core/pkg/api"
)

var _ = Describe("Mapper", func() {
	Context("With a VM supplied", func() {
		It("should map to v1.VM", func() {
			vm := api.VM{ObjectMeta: kubeapi.ObjectMeta{Name: "testvm", UID: types.UID(uuid.NewV4().String())}}
			vm.Spec = api.VMSpec{NodeSelector: map[string]string{"key": "val"}}
			dto := VM{}
			errs := model.Copy(&dto, &vm)
			Expect(errs).To(BeNil())
			Expect(dto.Name).To(Equal("testvm"))
			Expect(dto.GetUID()).To(Equal(vm.GetUID()))
			Expect(dto.Spec.NodeSelector).To(Equal(vm.Spec.NodeSelector))
		})
	})
	Context("With a v1.VM supplied", func() {
		It("should map to VM", func() {
			vm := api.VM{}
			dto := VM{ObjectMeta: kubeapi.ObjectMeta{Name: "testvm", UID: types.UID(uuid.NewV4().String())}}
			dto.Spec = VMSpec{NodeSelector: map[string]string{"key": "val"}}
			errs := model.Copy(&vm, &dto)
			Expect(errs).To(BeNil())
			Expect(vm.Name).To(Equal("testvm"))
			Expect(vm.UID).To(Equal(dto.UID))
			Expect(vm.Spec.NodeSelector).To(Equal(vm.Spec.NodeSelector))
		})
	})
})
