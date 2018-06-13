package vm

import (
	"testing"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	k8smetav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"kubevirt.io/kubevirt/pkg/api/v1"
)

var _ = Describe("VirtualMachine", func() {

	Context("OfflineVirtualMachine command invocation", func() {
		var commandName string
		var cmd *Command

		BeforeEach(func() {
			commandName = "test"
			cmd = NewCommand(commandName)
		})

		It("should create commands based on given verb", func() {
			Expect(cmd.command).To(Equal(commandName))
		})


	})
})

func TestVM(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "OfflineVMI")
}

func NewMinimalOVM(name string) *v1.VirtualMachine {
	return &v1.VirtualMachine{TypeMeta: k8smetav1.TypeMeta{APIVersion: v1.GroupVersion.String(), Kind: "OfflineVirtualMachine"}, ObjectMeta: k8smetav1.ObjectMeta{Name: name}}
}
