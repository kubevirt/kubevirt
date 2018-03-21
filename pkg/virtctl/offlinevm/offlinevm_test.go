package offlinevm

import (
	"strings"
	"testing"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("OfflineVirtualMachine", func() {

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

		It("should use verb in usage", func() {
			reference := "Test an OfflineVirtualMachine"
			usage := cmd.Usage()
			lines := strings.Split(usage, "\n")
			Expect(lines[0]).To(Equal(reference))
		})

		It("should use a different verb in usage", func() {
			commandName = "grok"
			reference := "Grok an OfflineVirtualMachine"
			cmd = NewCommand(commandName)

			usage := cmd.Usage()
			lines := strings.Split(usage, "\n")
			Expect(lines[0]).To(Equal(reference))
		})
	})
})

func TestOVM(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "OfflineVM")
}
