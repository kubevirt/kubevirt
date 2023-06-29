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
		const (
			sevResourceName = "devices.kubevirt.io/sev"
			sevDevicePath   = "/proc/1/root/dev/sev"
		)

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

			checkCmd := []string{"ls", sevDevicePath}
			_, err = tests.ExecuteCommandInVirtHandlerPod(nodeName, checkCmd)
			isDevicePresent = (err == nil)

			if !isDevicePresent {
				By(fmt.Sprintf("Creating a fake SEV device on %s", nodeName))
				mknodCmd := []string{"mknod", sevDevicePath, "c", "10", "124"}
				_, err = tests.ExecuteCommandInVirtHandlerPod(nodeName, mknodCmd)
				Expect(err).ToNot(HaveOccurred())
			}

			Eventually(func() bool {
				node, err := virtClient.CoreV1().Nodes().Get(context.Background(), nodeName, k8smetav1.GetOptions{})
				Expect(err).ToNot(HaveOccurred())
				val, ok := node.Status.Capacity[sevResourceName]
				return ok && !val.IsZero()
			}, 180*time.Second, 1*time.Second).Should(BeTrue(), fmt.Sprintf("SEV capacity should not be zero on %s", nodeName))
		})

		AfterEach(func() {
			if !isDevicePresent {
				By(fmt.Sprintf("Removing the fake SEV device from %s", nodeName))
				rmCmd := []string{"rm", "-f", sevDevicePath}
				_, err = tests.ExecuteCommandInVirtHandlerPod(nodeName, rmCmd)
				Expect(err).ToNot(HaveOccurred())
			}
		})

		It("should reset SEV capacity when the feature gate is disabled", func() {
			By(fmt.Sprintf("Disabling %s feature gate", virtconfig.WorkloadEncryptionSEV))
			tests.DisableFeatureGate(virtconfig.WorkloadEncryptionSEV)
			Eventually(func() bool {
				node, err := virtClient.CoreV1().Nodes().Get(context.Background(), nodeName, k8smetav1.GetOptions{})
				Expect(err).ToNot(HaveOccurred())
				val, ok := node.Status.Capacity[sevResourceName]
				return !ok || val.IsZero()
			}, 180*time.Second, 1*time.Second).Should(BeTrue(), fmt.Sprintf("SEV capacity should be zero on %s", nodeName))
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
