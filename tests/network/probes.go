package network

import (
	"context"
	"fmt"
	"time"

	expect "github.com/google/goexpect"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	k8sv1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"

	v1 "kubevirt.io/api/core/v1"
	"kubevirt.io/client-go/kubecli"

	"kubevirt.io/kubevirt/pkg/libvmi"
	"kubevirt.io/kubevirt/tests/console"
	"kubevirt.io/kubevirt/tests/framework/kubevirt"
	"kubevirt.io/kubevirt/tests/framework/matcher"
	"kubevirt.io/kubevirt/tests/libnet"
	"kubevirt.io/kubevirt/tests/libnet/vmnetserver"
	"kubevirt.io/kubevirt/tests/libvmifact"
	"kubevirt.io/kubevirt/tests/libvmops"
	"kubevirt.io/kubevirt/tests/testsuite"
)

const (
	specifyingVMReadinessProbe = "Specifying a VMI with a readiness probe"
	specifyingVMLivenessProbe  = "Specifying a VMI with a liveness probe"
)

const (
	startAgent = "start"
	stopAgent  = "stop"
)

var _ = Describe(SIG("[ref_id:1182]Probes", func() {
	var (
		err           error
		virtClient    kubecli.KubevirtClient
		vmi           *v1.VirtualMachineInstance
		blankIPFamily = *new(k8sv1.IPFamily)
	)

	BeforeEach(func() {
		virtClient = kubevirt.Client()
	})

	Context("for readiness", func() {
		const (
			period         = 5
			initialSeconds = 5
			timeoutSeconds = 1
			port           = 1500
		)

		It("should succeed with working TCP probe and tcp server on ipv4", func() {
			libnet.SkipWhenClusterNotSupportIPFamily(k8sv1.IPv4Protocol)

			By(specifyingVMReadinessProbe)
			readinessProbe := createTCPProbe(period, initialSeconds, port)
			vmi = createReadyAlpineVMIWithReadinessProbe(readinessProbe)

			Expect(matcher.ThisVMI(vmi)()).To(matcher.HaveConditionMissingOrFalse(v1.VirtualMachineInstanceReady))

			By("Starting the server inside the VMI")
			Eventually(matcher.ThisVMI(vmi)).WithTimeout(3 * time.Minute).WithPolling(3 * time.Second).Should(matcher.HaveConditionTrue(v1.VirtualMachineInstanceAgentConnected))
			vmi, err = kubevirt.Client().VirtualMachineInstance(testsuite.GetTestNamespace(nil)).Get(context.Background(), vmi.Name, metav1.GetOptions{})
			Expect(err).ToNot(HaveOccurred())
			vmnetserver.StartTCPServer(vmi, 1500, console.LoginToAlpine)

			Eventually(matcher.ThisVMI(vmi)).
				WithTimeout(2 * time.Minute).
				WithPolling(2 * time.Second).
				Should(matcher.HaveConditionTrue(v1.VirtualMachineInstanceReady))
		})

		Context("guest agent ping", func() {
			BeforeEach(func() {
				vmi = libvmifact.NewFedora(libnet.WithMasqueradeNetworking(), withReadinessProbe(createGuestAgentPingProbe(period, initialSeconds)))
				vmi = libvmops.RunVMIAndExpectLaunchIgnoreWarnings(vmi, 180)
				By("Waiting for agent to connect")
				Eventually(matcher.ThisVMI(vmi), 12*time.Minute, 2*time.Second).Should(matcher.HaveConditionTrue(v1.VirtualMachineInstanceAgentConnected))

				Eventually(matcher.ThisVMI(vmi), 2*time.Minute, 2*time.Second).Should(matcher.HaveConditionTrue(v1.VirtualMachineInstanceReady))
				By("Disabling the guest-agent")
				Expect(console.LoginToFedora(vmi)).To(Succeed())
				Expect(stopGuestAgent(vmi)).To(Succeed())
				Eventually(matcher.ThisVMI(vmi)).
					WithTimeout(5 * time.Minute).
					WithPolling(2 * time.Second).
					Should(matcher.HaveConditionMissingOrFalse(v1.VirtualMachineInstanceReady))
			})

			When("the guest agent is enabled, after being disabled", func() {
				BeforeEach(func() {
					Expect(console.LoginToFedora(vmi)).To(Succeed())
					Expect(startGuestAgent(vmi)).To(Succeed())
				})

				It("[test_id:6741] the VMI enters `Ready` state once again", func() {
					Eventually(matcher.ThisVMI(vmi)).
						WithTimeout(2 * time.Minute).
						WithPolling(2 * time.Second).
						Should(matcher.HaveConditionTrue(v1.VirtualMachineInstanceReady))
				})
			})
		})

		DescribeTable("should fail", func(readinessProbe *v1.Probe, vmiFactory func(opts ...libvmi.Option) *v1.VirtualMachineInstance) {
			By(specifyingVMReadinessProbe)
			vmi = vmiFactory(withReadinessProbe(readinessProbe))
			vmi = libvmops.RunVMIAndExpectLaunchIgnoreWarnings(vmi, 180)

			By("Checking that the VMI is consistently non-ready")
			Consistently(matcher.ThisVMI(vmi)).
				WithTimeout(30 * time.Second).
				WithPolling(100 * time.Millisecond).
				Should(matcher.HaveConditionMissingOrFalse(v1.VirtualMachineInstanceReady))
		},
			Entry("[test_id:1220][posneg:negative]with working TCP probe and no running server", createTCPProbe(period, initialSeconds, port), libvmifact.NewAlpine),
			Entry("[test_id:TODO]with working Exec probe and invalid command", createExecProbe(period, initialSeconds, timeoutSeconds, "exit", "1"), libvmifact.NewFedora),
			Entry("[test_id:TODO]with working Exec probe and infinitely running command", createExecProbe(period, initialSeconds, timeoutSeconds, "tail", "-f", "/dev/null"), libvmifact.NewFedora),
		)
	})

	Context("for liveness", func() {
		const (
			period         = 5
			initialSeconds = 90
			timeoutSeconds = 1
			port           = 1500
		)

		DescribeTable("should not fail the VMI", func(livenessProbe *v1.Probe, ipFamily k8sv1.IPFamily) {
			libnet.SkipWhenClusterNotSupportIPFamily(ipFamily)

			if isExecProbe(livenessProbe) {
				By(specifyingVMLivenessProbe)
				vmi = libvmifact.NewFedora(libnet.WithMasqueradeNetworking(), withLivenessProbe(livenessProbe))
				vmi = libvmops.RunVMIAndExpectLaunchIgnoreWarnings(vmi, 180)

				By("Waiting for agent to connect")
				Eventually(matcher.ThisVMI(vmi)).
					WithTimeout(12 * time.Minute).
					WithPolling(2 * time.Second).
					Should(matcher.HaveConditionTrue(v1.VirtualMachineInstanceAgentConnected))
			} else {
				By(specifyingVMLivenessProbe)
				vmi = createReadyAlpineVMIWithLivenessProbe(livenessProbe)

				By("Starting the server inside the VMI")
				vmnetserver.StartTCPServer(vmi, 1500, console.LoginToAlpine)
			}

			By("Checking that the VMI is still running after a while")
			Consistently(func() bool {
				vmi, err := virtClient.VirtualMachineInstance(testsuite.GetTestNamespace(vmi)).Get(context.Background(), vmi.Name, metav1.GetOptions{})
				Expect(err).ToNot(HaveOccurred())
				return vmi.IsFinal()
			}).WithTimeout(2 * time.Minute).
				WithPolling(1 * time.Second).
				Should(Not(BeTrue()))
		},
			Entry("[test_id:1199][posneg:positive]with working TCP probe and tcp server on ipv4", createTCPProbe(period, initialSeconds, port), k8sv1.IPv4Protocol),
			Entry("[test_id:5879]with working Exec probe", createExecProbe(period, initialSeconds, timeoutSeconds, "uname", "-a"), blankIPFamily),
		)

		Context("guest agent ping", func() {
			BeforeEach(func() {
				vmi = libvmifact.NewFedora(libnet.WithMasqueradeNetworking(), withLivenessProbe(createGuestAgentPingProbe(period, initialSeconds)))
				vmi = libvmops.RunVMIAndExpectLaunchIgnoreWarnings(vmi, 180)

				By("Waiting for agent to connect")
				Eventually(matcher.ThisVMI(vmi)).
					WithTimeout(12 * time.Minute).
					WithPolling(2 * time.Second).
					Should(matcher.HaveConditionTrue(v1.VirtualMachineInstanceAgentConnected))
				Expect(console.LoginToFedora(vmi)).To(Succeed())
			})

			It("[test_id:9299] VM stops when guest agent is disabled", func() {
				Expect(stopGuestAgent(vmi)).To(Succeed())

				Eventually(func() (*v1.VirtualMachineInstance, error) {
					return virtClient.VirtualMachineInstance(testsuite.GetTestNamespace(vmi)).Get(context.Background(), vmi.Name, metav1.GetOptions{})
				}).WithTimeout(2 * time.Minute).
					WithPolling(1 * time.Second).
					Should(Or(matcher.BeInPhase(v1.Failed), matcher.HaveSucceeded()))
			})
		})

		DescribeTable("should fail the VMI", func(livenessProbe *v1.Probe, vmiFactory func(opts ...libvmi.Option) *v1.VirtualMachineInstance) {
			By("Specifying a VMI with a livenessProbe probe")
			vmi = vmiFactory(withLivenessProbe(livenessProbe))
			vmi = libvmops.RunVMIAndExpectLaunchIgnoreWarnings(vmi, 180)

			By("Checking that the VMI is in a final state after a while")
			Eventually(func() bool {
				vmi, err := virtClient.VirtualMachineInstance(testsuite.GetTestNamespace(vmi)).Get(context.Background(), vmi.Name, metav1.GetOptions{})
				Expect(err).ToNot(HaveOccurred())
				return vmi.IsFinal()
			}).WithTimeout(2 * time.Minute).
				WithPolling(1 * time.Second).
				Should(BeTrue())
		},
			Entry("[test_id:1217][posneg:negative]with working TCP probe and no running server", createTCPProbe(period, initialSeconds, port), libvmifact.NewCirros),
			Entry("[test_id:5880]with working Exec probe and invalid command", createExecProbe(period, initialSeconds, timeoutSeconds, "exit", "1"), libvmifact.NewFedora),
		)
	})
}))

func isExecProbe(probe *v1.Probe) bool {
	return probe.Exec != nil
}

func startGuestAgent(vmi *v1.VirtualMachineInstance) error {
	return guestAgentOperation(vmi, startAgent)
}

func stopGuestAgent(vmi *v1.VirtualMachineInstance) error {
	return guestAgentOperation(vmi, stopAgent)
}

func guestAgentOperation(vmi *v1.VirtualMachineInstance, startStopOperation string) error {
	if startStopOperation != startAgent && startStopOperation != stopAgent {
		return fmt.Errorf("invalid qemu-guest-agent request: %s. Allowed values are: '%s' *or* '%s'", startStopOperation, startAgent, stopAgent)
	}
	guestAgentSysctlString := fmt.Sprintf("sudo systemctl %s qemu-guest-agent\n", startStopOperation)
	return console.SafeExpectBatch(vmi, []expect.Batcher{
		&expect.BSnd{S: guestAgentSysctlString},
		&expect.BExp{R: console.PromptExpression},
	}, 120)
}

func createReadyAlpineVMIWithReadinessProbe(probe *v1.Probe) *v1.VirtualMachineInstance {
	vmi := libvmifact.NewAlpineWithTestTooling(libnet.WithMasqueradeNetworking(), withReadinessProbe(probe))
	return libvmops.RunVMIAndExpectLaunchIgnoreWarnings(vmi, 180)
}

func createReadyAlpineVMIWithLivenessProbe(probe *v1.Probe) *v1.VirtualMachineInstance {
	vmi := libvmifact.NewAlpineWithTestTooling(libnet.WithMasqueradeNetworking(), withLivenessProbe(probe))

	return libvmops.RunVMIAndExpectLaunchIgnoreWarnings(vmi, 180)
}

func createTCPProbe(period, initialSeconds int32, port int) *v1.Probe {
	httpHandler := v1.Handler{
		TCPSocket: &k8sv1.TCPSocketAction{
			Port: intstr.FromInt(port),
		},
	}
	return createProbeSpecification(period, initialSeconds, 1, httpHandler)
}

func createGuestAgentPingProbe(period, initialSeconds int32) *v1.Probe {
	handler := v1.Handler{GuestAgentPing: &v1.GuestAgentPing{}}
	return createProbeSpecification(period, initialSeconds, 1, handler)
}

func createExecProbe(period, initialSeconds, timeoutSeconds int32, command ...string) *v1.Probe {
	execHandler := v1.Handler{Exec: &k8sv1.ExecAction{Command: command}}
	return createProbeSpecification(period, initialSeconds, timeoutSeconds, execHandler)
}

func createProbeSpecification(period, initialSeconds, timeoutSeconds int32, handler v1.Handler) *v1.Probe {
	return &v1.Probe{
		PeriodSeconds:       period,
		InitialDelaySeconds: initialSeconds,
		Handler:             handler,
		TimeoutSeconds:      timeoutSeconds,
	}
}

func withReadinessProbe(probe *v1.Probe) libvmi.Option {
	return func(vmi *v1.VirtualMachineInstance) {
		vmi.Spec.ReadinessProbe = probe
	}
}

func withLivenessProbe(probe *v1.Probe) libvmi.Option {
	return func(vmi *v1.VirtualMachineInstance) {
		vmi.Spec.LivenessProbe = probe
	}
}
