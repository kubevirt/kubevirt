package network

import (
	"context"
	"fmt"

	expect "github.com/google/goexpect"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"

	"kubevirt.io/kubevirt/tests/util"

	v1 "kubevirt.io/api/core/v1"
	"kubevirt.io/client-go/kubecli"
	"kubevirt.io/kubevirt/tests"
	"kubevirt.io/kubevirt/tests/console"
	"kubevirt.io/kubevirt/tests/libnet"
	"kubevirt.io/kubevirt/tests/libvmi"
)

const (
	specifyingVMReadinessProbe = "Specifying a VMI with a readiness probe"
	specifyingVMLivenessProbe  = "Specifying a VMI with a liveness probe"
)

const (
	startAgent = "start"
	stopAgent  = "stop"
)

var _ = SIGDescribe("[ref_id:1182]Probes", func() {
	var (
		err           error
		virtClient    kubecli.KubevirtClient
		vmi           *v1.VirtualMachineInstance
		blankIPFamily = *new(corev1.IPFamily)
	)

	BeforeEach(func() {
		virtClient, err = kubecli.GetKubevirtClient()
		util.PanicOnError(err)
	})

	buildProbeBackendPodSpec := func(probe *v1.Probe) (*corev1.Pod, func() error) {
		var probeBackendPod *corev1.Pod
		if isHTTPProbe(*probe) {
			port := probe.HTTPGet.Port.IntVal
			probeBackendPod = tests.StartHTTPServerPod(int(port))
		} else {
			port := probe.TCPSocket.Port.IntVal
			probeBackendPod = tests.StartTCPServerPod(int(port))
		}
		return probeBackendPod, func() error {
			return virtClient.CoreV1().Pods(util.NamespaceTestDefault).Delete(context.Background(), probeBackendPod.Name, metav1.DeleteOptions{})
		}
	}

	Context("for readiness", func() {
		const (
			period         = 5
			initialSeconds = 5
			timeoutSeconds = 1
			port           = 1500
		)

		tcpProbe := createTCPProbe(period, initialSeconds, port)
		httpProbe := createHTTPProbe(period, initialSeconds, port)

		DescribeTable("should succeed", func(readinessProbe *v1.Probe, ipFamily corev1.IPFamily) {
			libnet.SkipWhenClusterNotSupportIPFamily(virtClient, ipFamily)

			if ipFamily == corev1.IPv6Protocol {
				By("Create a support pod which will reply to kubelet's probes ...")
				probeBackendPod, supportPodCleanupFunc := buildProbeBackendPodSpec(readinessProbe)
				defer func() {
					Expect(supportPodCleanupFunc()).To(Succeed(), "The support pod responding to the probes should be cleaned-up at test tear-down.")
				}()

				By("Attaching the readiness probe to an external pod server")
				readinessProbe, err = pointIpv6ProbeToSupportPod(probeBackendPod, readinessProbe)
				Expect(err).ToNot(HaveOccurred(), "should attach the backend pod with readiness probe")

				By(specifyingVMReadinessProbe)
				vmi = createReadyAlpineVMIWithReadinessProbe(readinessProbe)
			} else if !isExecProbe(readinessProbe) {
				By(specifyingVMReadinessProbe)
				vmi = createReadyAlpineVMIWithReadinessProbe(readinessProbe)

				Expect(getVMIConditions(virtClient, vmi)).NotTo(ContainElement(v1.VirtualMachineInstanceReady))

				By("Starting the server inside the VMI")
				serverStarter(vmi, readinessProbe, 1500)
			} else {
				By(specifyingVMReadinessProbe)
				vmi = libvmi.NewFedora(
					withMasqueradeNetworkingAndFurtherUserConfig(withReadinessProbe(readinessProbe))...)
				vmi = tests.VMILauncherIgnoreWarnings(virtClient)(vmi)

				By("Waiting for agent to connect")
				tests.WaitAgentConnected(virtClient, vmi)
			}

			tests.WaitForVMICondition(virtClient, vmi, v1.VirtualMachineInstanceReady, 120)
		},
			Entry("[test_id:1202][posneg:positive]with working TCP probe and tcp server on ipv4", tcpProbe, corev1.IPv4Protocol),
			Entry("[test_id:1202][posneg:positive]with working TCP probe and tcp server on ipv6", tcpProbe, corev1.IPv6Protocol),
			Entry("[test_id:1200][posneg:positive]with working HTTP probe and http server on ipv4", httpProbe, corev1.IPv4Protocol),
			Entry("[test_id:1200][posneg:positive]with working HTTP probe and http server on ipv6", httpProbe, corev1.IPv6Protocol),
			Entry("[test_id:TODO]with working Exec probe", createExecProbe(period, initialSeconds, timeoutSeconds, "uname", "-a"), blankIPFamily),
		)

		Context("guest agent ping", func() {
			const (
				guestAgentConnectTimeout    = 120
				guestAgentDisconnectTimeout = 300 // Marking the status to not ready can take a little more time
			)

			BeforeEach(func() {
				vmi = libvmi.NewFedora(
					withMasqueradeNetworkingAndFurtherUserConfig(
						withReadinessProbe(
							createGuestAgentPingProbe(period, initialSeconds)))...)
				vmi = tests.VMILauncherIgnoreWarnings(virtClient)(vmi)
				By("Waiting for agent to connect")
				tests.WaitAgentConnected(virtClient, vmi)
				tests.WaitForVMICondition(virtClient, vmi, v1.VirtualMachineInstanceReady, guestAgentConnectTimeout)
				By("Disabling the guest-agent")
				Expect(console.LoginToFedora(vmi)).To(Succeed())
				Expect(stopGuestAgent(vmi)).To(Succeed())
				tests.WaitForVMIConditionRemovedOrFalse(virtClient, vmi, v1.VirtualMachineInstanceReady, guestAgentDisconnectTimeout)
			})

			When("the guest agent is enabled, after being disabled", func() {
				BeforeEach(func() {
					Expect(console.LoginToFedora(vmi)).To(Succeed())
					Expect(startGuestAgent(vmi)).To(Succeed())
				})

				It("[test_id:6741] the VMI enters `Ready` state once again", func() {
					tests.WaitForVMICondition(virtClient, vmi, v1.VirtualMachineInstanceReady, guestAgentConnectTimeout)
				})
			})
		})

		DescribeTable("should fail", func(readinessProbe *v1.Probe, vmiFactory func(opts ...libvmi.Option) *v1.VirtualMachineInstance) {
			By(specifyingVMReadinessProbe)
			vmi = vmiFactory(withReadinessProbe(readinessProbe))
			vmi = tests.VMILauncherIgnoreWarnings(virtClient)(vmi)

			By("Checking that the VMI is consistently non-ready")
			Consistently(func() []v1.VirtualMachineInstanceCondition {
				return getVMIConditions(virtClient, vmi)
			}).ShouldNot(ContainElement(v1.VirtualMachineInstanceReady))
		},
			Entry("[test_id:1220][posneg:negative]with working TCP probe and no running server", tcpProbe, libvmi.NewAlpine),
			Entry("[test_id:1219][posneg:negative]with working HTTP probe and no running server", httpProbe, libvmi.NewAlpine),
			Entry("[test_id:TODO]with working Exec probe and invalid command", createExecProbe(period, initialSeconds, timeoutSeconds, "exit", "1"), libvmi.NewFedora),
			Entry("[test_id:TODO]with working Exec probe and infinitely running command", createExecProbe(period, initialSeconds, timeoutSeconds, "tail", "-f", "/dev/null"), libvmi.NewFedora),
		)
	})

	Context("for liveness", func() {
		const (
			period         = 5
			initialSeconds = 90
			timeoutSeconds = 1
			port           = 1500
		)

		tcpProbe := createTCPProbe(period, initialSeconds, port)
		httpProbe := createHTTPProbe(period, initialSeconds, port)

		DescribeTable("should not fail the VMI", func(livenessProbe *v1.Probe, ipFamily corev1.IPFamily) {
			libnet.SkipWhenClusterNotSupportIPFamily(virtClient, ipFamily)

			if ipFamily == corev1.IPv6Protocol {

				By("Create a support pod which will reply to kubelet's probes ...")
				probeBackendPod, supportPodCleanupFunc := buildProbeBackendPodSpec(livenessProbe)
				defer func() {
					Expect(supportPodCleanupFunc()).To(Succeed(), "The support pod responding to the probes should be cleaned-up at test tear-down.")
				}()

				By("Attaching the liveness probe to an external pod server")
				livenessProbe, err = pointIpv6ProbeToSupportPod(probeBackendPod, livenessProbe)
				Expect(err).ToNot(HaveOccurred(), "should attach the backend pod with livness probe")

				By(specifyingVMLivenessProbe)
				vmi = createReadyAlpineVMIWithLivenessProbe(livenessProbe)
			} else if !isExecProbe(livenessProbe) {
				By(specifyingVMLivenessProbe)
				vmi = createReadyAlpineVMIWithLivenessProbe(livenessProbe)

				By("Starting the server inside the VMI")
				serverStarter(vmi, livenessProbe, 1500)
			} else {
				By(specifyingVMLivenessProbe)
				vmi = libvmi.NewFedora(
					withMasqueradeNetworkingAndFurtherUserConfig(
						withLivelinessProbe(livenessProbe))...)
				vmi = tests.VMILauncherIgnoreWarnings(virtClient)(vmi)

				By("Waiting for agent to connect")
				tests.WaitAgentConnected(virtClient, vmi)
			}

			By("Checking that the VMI is still running after a while")
			Consistently(func() bool {
				vmi, err := virtClient.VirtualMachineInstance(util.NamespaceTestDefault).Get(vmi.Name, &metav1.GetOptions{})
				Expect(err).ToNot(HaveOccurred())
				return vmi.IsFinal()
			}, 120, 1).Should(Not(BeTrue()))
		},
			Entry("[test_id:1199][posneg:positive]with working TCP probe and tcp server on ipv4", tcpProbe, corev1.IPv4Protocol),
			Entry("[test_id:1199][posneg:positive]with working TCP probe and tcp server on ipv6", tcpProbe, corev1.IPv6Protocol),
			Entry("[test_id:1201][posneg:positive]with working HTTP probe and http server on ipv4", httpProbe, corev1.IPv4Protocol),
			Entry("[test_id:1201][posneg:positive]with working HTTP probe and http server on ipv6", httpProbe, corev1.IPv6Protocol),
			Entry("[test_id:TODO]with working Exec probe", createExecProbe(period, initialSeconds, timeoutSeconds, "uname", "-a"), blankIPFamily),
		)

		DescribeTable("should fail the VMI", func(livenessProbe *v1.Probe, vmiFactory func(opts ...libvmi.Option) *v1.VirtualMachineInstance) {
			By("Specifying a VMI with a livenessProbe probe")
			vmi = vmiFactory(withLivelinessProbe(livenessProbe))
			vmi = tests.VMILauncherIgnoreWarnings(virtClient)(vmi)

			By("Checking that the VMI is in a final state after a while")
			Eventually(func() bool {
				vmi, err := virtClient.VirtualMachineInstance(util.NamespaceTestDefault).Get(vmi.Name, &metav1.GetOptions{})
				Expect(err).ToNot(HaveOccurred())
				return vmi.IsFinal()
			}, 120, 1).Should(BeTrue())
		},
			Entry("[test_id:1217][posneg:negative]with working TCP probe and no running server", tcpProbe, libvmi.NewCirros),
			Entry("[test_id:1218][posneg:negative]with working HTTP probe and no running server", httpProbe, libvmi.NewCirros),
			Entry("[test_id:TODO]with working Exec probe and invalid command", createExecProbe(period, initialSeconds, timeoutSeconds, "exit", "1"), libvmi.NewFedora),
		)
	})
})

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
	vmi := libvmi.NewAlpineWithTestTooling(
		withMasqueradeNetworkingAndFurtherUserConfig(withReadinessProbe(probe))...)

	return tests.RunVMIAndExpectLaunchIgnoreWarnings(vmi, 180)
}

func createReadyAlpineVMIWithLivenessProbe(probe *v1.Probe) *v1.VirtualMachineInstance {
	vmi := libvmi.NewAlpineWithTestTooling(
		withMasqueradeNetworkingAndFurtherUserConfig(withLivelinessProbe(probe))...)

	return tests.RunVMIAndExpectLaunchIgnoreWarnings(vmi, 180)
}

func createTCPProbe(period int32, initialSeconds int32, port int) *v1.Probe {
	httpHandler := v1.Handler{
		TCPSocket: &corev1.TCPSocketAction{
			Port: intstr.FromInt(port),
		},
	}
	return createProbeSpecification(period, initialSeconds, 1, httpHandler)
}

func createGuestAgentPingProbe(period int32, initialSeconds int32) *v1.Probe {
	handler := v1.Handler{GuestAgentPing: &v1.GuestAgentPing{}}
	return createProbeSpecification(period, initialSeconds, 1, handler)
}

func patchProbeWithIPAddr(existingProbe *v1.Probe, ipHostIP string) *v1.Probe {
	if isHTTPProbe(*existingProbe) {
		existingProbe.HTTPGet.Host = ipHostIP
	} else {
		existingProbe.TCPSocket.Host = ipHostIP
	}
	return existingProbe
}

func createHTTPProbe(period int32, initialSeconds int32, port int) *v1.Probe {
	httpHandler := v1.Handler{
		HTTPGet: &corev1.HTTPGetAction{
			Port: intstr.FromInt(port),
		},
	}
	return createProbeSpecification(period, initialSeconds, 1, httpHandler)
}

func createExecProbe(period int32, initialSeconds int32, timeoutSeconds int32, command ...string) *v1.Probe {
	execHandler := v1.Handler{Exec: &corev1.ExecAction{Command: command}}
	return createProbeSpecification(period, initialSeconds, timeoutSeconds, execHandler)
}

func createProbeSpecification(period int32, initialSeconds int32, timeoutSeconds int32, handler v1.Handler) *v1.Probe {
	return &v1.Probe{
		PeriodSeconds:       period,
		InitialDelaySeconds: initialSeconds,
		Handler:             handler,
		TimeoutSeconds:      timeoutSeconds,
	}
}

func isHTTPProbe(probe v1.Probe) bool {
	return probe.Handler.HTTPGet != nil
}

func serverStarter(vmi *v1.VirtualMachineInstance, probe *v1.Probe, port int) {
	if isHTTPProbe(*probe) {
		tests.StartHTTPServer(vmi, port, console.LoginToAlpine)
	} else {
		tests.StartTCPServer(vmi, port, console.LoginToAlpine)
	}
}

func pointIpv6ProbeToSupportPod(pod *corev1.Pod, probe *v1.Probe) (*v1.Probe, error) {
	supportPodIP := libnet.GetPodIPByFamily(pod, corev1.IPv6Protocol)
	if supportPodIP == "" {
		return nil, fmt.Errorf("pod/%s does not have an IPv6 address", pod.Name)
	}

	return patchProbeWithIPAddr(probe, supportPodIP), nil
}

func getVMIConditions(virtClient kubecli.KubevirtClient, vmi *v1.VirtualMachineInstance) []v1.VirtualMachineInstanceCondition {
	readVmi, err := virtClient.VirtualMachineInstance(vmi.Namespace).Get(vmi.Name, &metav1.GetOptions{})
	Expect(err).ToNot(HaveOccurred())
	return readVmi.Status.Conditions
}

func withMasqueradeNetworkingAndFurtherUserConfig(opts ...libvmi.Option) []libvmi.Option {
	return append(libvmi.WithMasqueradeNetworking(), opts...)
}

func withReadinessProbe(probe *v1.Probe) libvmi.Option {
	return func(vmi *v1.VirtualMachineInstance) {
		vmi.Spec.ReadinessProbe = probe
	}
}

func withLivelinessProbe(probe *v1.Probe) libvmi.Option {
	return func(vmi *v1.VirtualMachineInstance) {
		vmi.Spec.LivenessProbe = probe
	}
}
