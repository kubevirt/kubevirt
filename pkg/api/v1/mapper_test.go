package v1

import (
	"encoding/json"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/satori/go.uuid"
	"k8s.io/client-go/pkg/api/v1"
	"k8s.io/client-go/pkg/types"
)

var _ = Describe("Mapper", func() {
	Context("With a VM supplied", func() {
		It("should map to json and back with custom mapper", func() {
			vm := VM{ObjectMeta: v1.ObjectMeta{Name: "testvm", UID: types.UID(uuid.NewV4().String())}}
			newVM := VM{}
			buf, err := json.Marshal(&vm)
			Expect(err).To(BeNil())
			err = json.Unmarshal(buf, &newVM)
			Expect(err).To(BeNil())
			Expect(newVM).To(Equal(vm))
		})
	})
})
