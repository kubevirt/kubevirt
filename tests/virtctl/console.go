package virtctl

import (
	"bytes"
	"os"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"kubevirt.io/kubevirt/pkg/pointer"
	"kubevirt.io/kubevirt/tests"
	"kubevirt.io/kubevirt/tests/decorators"

	"kubevirt.io/kubevirt/tests/clientcmd"
	"kubevirt.io/kubevirt/tests/libvmi"
	"kubevirt.io/kubevirt/tests/util"
)

const alpineStartupTimeout = 60
const logWarning = "Caution: the output of this console connection can be streamed into system logs.\nIf you are inputting any screen visible sensitive information please consider using SSH unless the serial console is absolutely necessary.\n"
const staticMessage = "console. The escape sequence is ^]\n"

var _ = Describe("[sig-compute][virtctl]console", decorators.SigCompute, func() {
	Context("virtctl caution message", func() {
		DescribeTable("check the presence of a caution message according to LogSerialConsole configuration", func(logSerialConsole *bool, expected bool) {
			By("Starting a VMI")
			alpineVmi := libvmi.NewAlpine()
			alpineVmi.Spec.Domain.Devices.AutoattachSerialConsole = pointer.P(true)
			alpineVmi.Spec.Domain.Devices.LogSerialConsole = logSerialConsole
			vmi := tests.RunVMIAndExpectLaunch(alpineVmi, alpineStartupTimeout)

			_, cmd, err := clientcmd.CreateCommandWithNS(util.NamespaceTestDefault, "virtctl", "console", vmi.Name)
			Expect(err).ToNot(HaveOccurred())
			outerr := &bytes.Buffer{}
			cmd.Stdout = outerr
			cmd.Stderr = outerr
			err = cmd.Start()
			Expect(err).ToNot(HaveOccurred())
			defer func(Process *os.Process) {
				err := Process.Kill()
				Expect(err).ToNot(HaveOccurred())
			}(cmd.Process)

			By("Checking for the caution message")
			Eventually(func(g Gomega) {
				output := string(outerr.Bytes())
				g.Expect(output).To(ContainSubstring(staticMessage))
				if expected {
					g.Expect(output).To(ContainSubstring(logWarning))
				} else {
					g.Expect(output).ToNot(ContainSubstring(logWarning))
				}
			}, 5*time.Second, time.Second).Should(Succeed())
		},
			Entry("with true LogSerialConsole", pointer.P(true), true),
			Entry("with false LogSerialConsole", pointer.P(false), false),
			Entry("without LogSerialConsole", nil, true),
		)
	})
})
