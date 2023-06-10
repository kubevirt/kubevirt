package launchsecurity

import (
	"context"
	"fmt"
	"time"

	"kubevirt.io/kubevirt/tests/decorators"

	expect "github.com/google/goexpect"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	k8smetav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"kubevirt.io/client-go/kubecli"

	virtconfig "kubevirt.io/kubevirt/pkg/virt-config"
	"kubevirt.io/kubevirt/tests"
	"kubevirt.io/kubevirt/tests/console"
	"kubevirt.io/kubevirt/tests/framework/checks"
	"kubevirt.io/kubevirt/tests/framework/kubevirt"
	"kubevirt.io/kubevirt/tests/libvmi"
)

var _ = Describe("[sig-compute]AMD Secure Encrypted Virtualization (SEV)", decorators.SEV, decorators.SigCompute, func() {
	BeforeEach(func() {
		checks.SkipTestIfNoFeatureGate(virtconfig.WorkloadEncryptionSEV)
	})

	Context("[Serial]device management", Serial, func() {
		var (
			virtClient      kubecli.KubevirtClient
			nodeName        string
			isDevicePresent bool
			err             error
		)

		BeforeEach(func() {
			virtClient = kubevirt.Client()

			nodeName = tests.NodeNameWithHandler()
			Expect(nodeName).ToNot(BeEmpty())

			checkCmd := []string{"ls", "/dev/sev"}
			_, err = tests.ExecuteCommandInVirtHandlerPod(nodeName, checkCmd)
			isDevicePresent = (err == nil)

			if !isDevicePresent {
				// Create a fake SEV device
				mknodCmd := []string{"mknod", "/dev/sev", "c", "10", "124"}
				_, err = tests.ExecuteCommandInVirtHandlerPod(nodeName, mknodCmd)
				Expect(err).ToNot(HaveOccurred())
			}

			Eventually(func() bool {
				node, err := virtClient.CoreV1().Nodes().Get(context.Background(), nodeName, k8smetav1.GetOptions{})
				Expect(err).ToNot(HaveOccurred())
				val, ok := node.Status.Capacity["devices.kubevirt.io/sev"]
				return ok && !val.IsZero()
			}, 90*time.Second, 1*time.Second).Should(BeTrue(), "SEV capacity should not be zero")
		})

		AfterEach(func() {
			if !isDevicePresent {
				// Remove the fake SEV device
				rmCmd := []string{"rm", "-f", "/dev/sev"}
				_, err = tests.ExecuteCommandInVirtHandlerPod(nodeName, rmCmd)
				Expect(err).ToNot(HaveOccurred())
			}

			tests.EnableFeatureGate(virtconfig.WorkloadEncryptionSEV)
		})

		It("should reset SEV capacity when the feature gate is disabled", func() {
			By(fmt.Sprintf("Disabling %s feature gate", virtconfig.WorkloadEncryptionSEV))
			tests.DisableFeatureGate(virtconfig.WorkloadEncryptionSEV)
			Eventually(func() bool {
				node, err := virtClient.CoreV1().Nodes().Get(context.Background(), nodeName, k8smetav1.GetOptions{})
				Expect(err).ToNot(HaveOccurred())
				val, ok := node.Status.Capacity["devices.kubevirt.io/sev"]
				return !ok || val.IsZero()
			}, 90*time.Second, 1*time.Second).Should(BeTrue(), "SEV capacity should be zero")
		})
	})

	Context("lifecycle", func() {
		BeforeEach(func() {
			checks.SkipTestIfNotSEVCapable()
		})

		DescribeTable("should start a SEV or SEV-ES VM",
			func(withES bool, sevstr string) {
				if withES {
					checks.SkipTestIfNotSEVESCapable()
				}
				const secureBoot = false
				vmi := libvmi.NewFedora(libvmi.WithUefi(secureBoot), libvmi.WithSEV(withES))
				vmi = tests.RunVMIAndExpectLaunch(vmi, 240)

				By("Expecting the VirtualMachineInstance console")
				Expect(console.LoginToFedora(vmi)).To(Succeed())

				By("Verifying that SEV is enabled in the guest")
				err := console.SafeExpectBatch(vmi, []expect.Batcher{
					&expect.BSnd{S: "\n"},
					&expect.BExp{R: console.PromptExpression},
					&expect.BSnd{S: "dmesg | grep --color=never SEV\n"},
					&expect.BExp{R: "AMD Memory Encryption Features active: " + sevstr},
					&expect.BSnd{S: "\n"},
					&expect.BExp{R: console.PromptExpression},
				}, 30)
				Expect(err).ToNot(HaveOccurred())
			},
			// SEV-ES disabled, SEV enabled
			Entry("It should launch with base SEV features enabled", false, "SEV"),
			// SEV-ES enabled
			Entry("It should launch with SEV-ES features enabled", true, "SEV SEV-ES"),
		)
	})
})
