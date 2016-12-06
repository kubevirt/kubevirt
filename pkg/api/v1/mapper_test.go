package v1

import (
	"encoding/json"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/rmohr/go-model"
	"github.com/satori/go.uuid"
	kubeapi "k8s.io/client-go/1.5/pkg/api"
	"k8s.io/client-go/1.5/pkg/types"
	"kubevirt.io/core/pkg/api"
)

var _ = Describe("Mapper", func() {
	Context("With a VM supplied", func() {
		It("should map to v1.VM", func() {
			vm := api.VM{ObjectMeta: kubeapi.ObjectMeta{Name: "testvm", UID: types.UID(uuid.NewV4().String())}}
			vm.Spec = api.VMSpec{NodeSelector: map[string]string{"key": "val"}}
			dto := VM{}
			errs := model.Copy(&dto, &vm)
			Expect(errs).To(BeNil())
			Expect(dto.GetObjectMeta().GetName()).To(Equal("testvm"))
			Expect(dto.GetObjectMeta().GetUID()).To(Equal(vm.GetObjectMeta().GetUID()))
			Expect(dto.Spec.NodeSelector).To(Equal(vm.Spec.NodeSelector))
		})
	})
	// TODO fix this test when we need api.VM again
	PContext("With a v1.VM supplied", func() {
		It("should map to VM", func() {
			vm := api.VM{}
			dto := VM{ObjectMeta: kubeapi.ObjectMeta{Name: "testvm", UID: types.UID(uuid.NewV4().String())}}
			dto.Spec = VMSpec{NodeSelector: map[string]string{"key": "val"}}
			errs := model.Copy(&vm, &dto)
			Expect(errs).To(BeNil())

			Expect(vm.GetObjectMeta().GetName()).To(Equal("testvm"))
			Expect(vm.GetObjectMeta().GetUID()).To(Equal(dto.GetObjectMeta().GetUID()))
			Expect(vm.Spec.NodeSelector).To(Equal(vm.Spec.NodeSelector))
		})
	})
	Context("With a VM supplied", func() {
		It("should map to json and back with custom mapper", func() {
			vm := VM{ObjectMeta: kubeapi.ObjectMeta{Name: "testvm", UID: types.UID(uuid.NewV4().String())}}
			newVM := VM{}
			buf, err := json.Marshal(&vm)
			Expect(err).To(BeNil())
			err = json.Unmarshal(buf, &newVM)
			Expect(err).To(BeNil())
			Expect(newVM).To(Equal(vm))
		})
	})
})
