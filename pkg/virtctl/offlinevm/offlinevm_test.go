package offlinevm

import (
	"testing"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("VirtualMachine", func() {

	Context("VirtualMachine command invocation", func() {
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
