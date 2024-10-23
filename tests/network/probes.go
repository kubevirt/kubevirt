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
	"kubevirt.io/kubevirt/tests/libpod"
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

var _ = SIGDescribe("[ref_id:1182]Probes", func() {
	var (
		err           error
		virtClient    kubecli.KubevirtClient
		vmi           *v1.VirtualMachineInstance
		blankIPFamily = *new(k8sv1.IPFamily)
	)

	BeforeEach(func() {
		virtClient = kubevirt.Client()
	})

	buildProbeBackendPodSpec := func(ipFamily k8sv1.IPFamily, probe *v1.Probe) (*k8sv1.Pod, func() error) {
		family := 4
		if ipFamily == k8sv1.IPv6Protocol {
			family = 6
		}

		var probeBackendPod *k8sv1.Pod
		if isHTTPProbe(*probe) {
			port := probe.HTTPGet.Port.IntVal
			probeBackendPod = startHTTPServerPod(family, int(port))
		} else {
			port := probe.TCPSocket.Port.IntVal
			probeBackendPod = startTCPServerPod(family, int(port))
		}
		return probeBackendPod, func() error {
			return virtClient.CoreV1().Pods(testsuite.GetTestNamespace(probeBackendPod)).Delete(context.Background(), probeBackendPod.Name, metav1.DeleteOptions{})
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

		DescribeTable("should succeed", func(readinessProbe *v1.Probe, ipFamily k8sv1.IPFamily) {
			libnet.SkipWhenClusterNotSupportIPFamily(ipFamily)

			if ipFamily == k8sv1.IPv6Protocol {
				By("Create a support pod which will reply to kubelet's probes ...")
				probeBackendPod, supportPodCleanupFunc := buildProbeBackendPodSpec(ipFamily, readinessProbe)
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

				Expect(matcher.ThisVMI(vmi)()).To(matcher.HaveConditionMissingOrFalse(v1.VirtualMachineInstanceReady))

				By("Starting the server inside the VMI")
				serverStarter(vmi, readinessProbe, 1500)
			} else {
				By(specifyingVMReadinessProbe)
				vmi = libvmifact.NewFedora(libnet.WithMasqueradeNetworking(), withReadinessProbe(readinessProbe))
				vmi = libvmops.RunVMIAndExpectLaunchIgnoreWarnings(vmi, 180)

				By("Waiting for agent to connect")
				Eventually(matcher.ThisVMI(vmi), 12*time.Minute, 2*time.Second).Should(matcher.HaveConditionTrue(v1.VirtualMachineInstanceAgentConnected))
			}

			Eventually(matcher.ThisVMI(vmi), 2*time.Minute, 2*time.Second).Should(matcher.HaveConditionTrue(v1.VirtualMachineInstanceReady))
		},
			Entry("[test_id:1202][posneg:positive]with working TCP probe and tcp server on ipv4", tcpProbe, k8sv1.IPv4Protocol),
			Entry("[test_id:1202][posneg:positive]with working TCP probe and tcp server on ipv6", tcpProbe, k8sv1.IPv6Protocol),
			Entry("[test_id:1200][posneg:positive]with working HTTP probe and http server on ipv4", httpProbe, k8sv1.IPv4Protocol),
			Entry("[test_id:1200][posneg:positive]with working HTTP probe and http server on ipv6", httpProbe, k8sv1.IPv6Protocol),
			Entry("[test_id:TODO]with working Exec probe", createExecProbe(period, initialSeconds, timeoutSeconds, "uname", "-a"), blankIPFamily),
		)

		Context("guest agent ping", func() {
			const (
				guestAgentDisconnectTimeout = 300 // Marking the status to not ready can take a little more time
			)

			BeforeEach(func() {
				vmi = libvmifact.NewFedora(libnet.WithMasqueradeNetworking(), withReadinessProbe(createGuestAgentPingProbe(period, initialSeconds)))
				vmi = libvmops.RunVMIAndExpectLaunchIgnoreWarnings(vmi, 180)
				By("Waiting for agent to connect")
				Eventually(matcher.ThisVMI(vmi), 12*time.Minute, 2*time.Second).Should(matcher.HaveConditionTrue(v1.VirtualMachineInstanceAgentConnected))

				Eventually(matcher.ThisVMI(vmi), 2*time.Minute, 2*time.Second).Should(matcher.HaveConditionTrue(v1.VirtualMachineInstanceReady))
				By("Disabling the guest-agent")
				Expect(console.LoginToFedora(vmi)).To(Succeed())
				Expect(stopGuestAgent(vmi)).To(Succeed())
				Eventually(matcher.ThisVMI(vmi), 5*time.Minute, 2*time.Second).Should(matcher.HaveConditionMissingOrFalse(v1.VirtualMachineInstanceReady))
			})

			When("the guest agent is enabled, after being disabled", func() {
				BeforeEach(func() {
					Expect(console.LoginToFedora(vmi)).To(Succeed())
					Expect(startGuestAgent(vmi)).To(Succeed())
				})

				It("[test_id:6741] the VMI enters `Ready` state once again", func() {
					Eventually(matcher.ThisVMI(vmi), 2*time.Minute, 2*time.Second).Should(matcher.HaveConditionTrue(v1.VirtualMachineInstanceReady))
				})
			})
		})

		DescribeTable("should fail", func(readinessProbe *v1.Probe, vmiFactory func(opts ...libvmi.Option) *v1.VirtualMachineInstance) {
			By(specifyingVMReadinessProbe)
			vmi = vmiFactory(withReadinessProbe(readinessProbe))
			vmi = libvmops.RunVMIAndExpectLaunchIgnoreWarnings(vmi, 180)

			By("Checking that the VMI is consistently non-ready")
			Consistently(matcher.ThisVMI(vmi), 30*time.Second, 100*time.Millisecond).Should(matcher.HaveConditionMissingOrFalse(v1.VirtualMachineInstanceReady))
		},
			Entry("[test_id:1220][posneg:negative]with working TCP probe and no running server", tcpProbe, libvmifact.NewAlpine),
			Entry("[test_id:1219][posneg:negative]with working HTTP probe and no running server", httpProbe, libvmifact.NewAlpine),
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

		tcpProbe := createTCPProbe(period, initialSeconds, port)
		httpProbe := createHTTPProbe(period, initialSeconds, port)

		DescribeTable("should not fail the VMI", func(livenessProbe *v1.Probe, ipFamily k8sv1.IPFamily) {
			libnet.SkipWhenClusterNotSupportIPFamily(ipFamily)

			if ipFamily == k8sv1.IPv6Protocol {

				By("Create a support pod which will reply to kubelet's probes ...")
				probeBackendPod, supportPodCleanupFunc := buildProbeBackendPodSpec(ipFamily, livenessProbe)
				defer func() {
					Expect(supportPodCleanupFunc()).To(Succeed(), "The support pod responding to the probes should be cleaned-up at test tear-down.")
				}()

				By("Attaching the liveness probe to an external pod server")
				livenessProbe, err = pointIpv6ProbeToSupportPod(probeBackendPod, livenessProbe)
				Expect(err).ToNot(HaveOccurred(), "should attach the backend pod with liveness probe")

				By(specifyingVMLivenessProbe)
				vmi = createReadyAlpineVMIWithLivenessProbe(livenessProbe)
			} else if !isExecProbe(livenessProbe) {
				By(specifyingVMLivenessProbe)
				vmi = createReadyAlpineVMIWithLivenessProbe(livenessProbe)

				By("Starting the server inside the VMI")
				serverStarter(vmi, livenessProbe, 1500)
			} else {
				By(specifyingVMLivenessProbe)
				vmi = libvmifact.NewFedora(libnet.WithMasqueradeNetworking(), withLivelinessProbe(livenessProbe))
				vmi = libvmops.RunVMIAndExpectLaunchIgnoreWarnings(vmi, 180)

				By("Waiting for agent to connect")
				Eventually(matcher.ThisVMI(vmi), 12*time.Minute, 2*time.Second).Should(matcher.HaveConditionTrue(v1.VirtualMachineInstanceAgentConnected))
			}

			By("Checking that the VMI is still running after a while")
			Consistently(func() bool {
				vmi, err := virtClient.VirtualMachineInstance(testsuite.GetTestNamespace(vmi)).Get(context.Background(), vmi.Name, metav1.GetOptions{})
				Expect(err).ToNot(HaveOccurred())
				return vmi.IsFinal()
			}, 120, 1).Should(Not(BeTrue()))
		},
			Entry("[test_id:1199][posneg:positive]with working TCP probe and tcp server on ipv4", tcpProbe, k8sv1.IPv4Protocol),
			Entry("[test_id:1199][posneg:positive]with working TCP probe and tcp server on ipv6", tcpProbe, k8sv1.IPv6Protocol),
			Entry("[test_id:1201][posneg:positive]with working HTTP probe and http server on ipv4", httpProbe, k8sv1.IPv4Protocol),
			Entry("[test_id:1201][posneg:positive]with working HTTP probe and http server on ipv6", httpProbe, k8sv1.IPv6Protocol),
			Entry("[test_id:5879]with working Exec probe", createExecProbe(period, initialSeconds, timeoutSeconds, "uname", "-a"), blankIPFamily),
		)

		Context("guest agent ping", func() {
			BeforeEach(func() {
				vmi = libvmifact.NewFedora(libnet.WithMasqueradeNetworking(), withLivelinessProbe(createGuestAgentPingProbe(period, initialSeconds)))
				vmi = libvmops.RunVMIAndExpectLaunchIgnoreWarnings(vmi, 180)

				By("Waiting for agent to connect")
				Eventually(matcher.ThisVMI(vmi), 12*time.Minute, 2*time.Second).Should(matcher.HaveConditionTrue(v1.VirtualMachineInstanceAgentConnected))
				Expect(console.LoginToFedora(vmi)).To(Succeed())
			})

			It("[test_id:9299] VM stops when guest agent is disabled", func() {
				Expect(stopGuestAgent(vmi)).To(Succeed())

				Eventually(func() (*v1.VirtualMachineInstance, error) {
					return virtClient.VirtualMachineInstance(testsuite.GetTestNamespace(vmi)).Get(context.Background(), vmi.Name, metav1.GetOptions{})
				}, 2*time.Minute, 1*time.Second).Should(Or(matcher.BeInPhase(v1.Failed), matcher.HaveSucceeded()))
			})
		})

		DescribeTable("should fail the VMI", func(livenessProbe *v1.Probe, vmiFactory func(opts ...libvmi.Option) *v1.VirtualMachineInstance) {
			By("Specifying a VMI with a livenessProbe probe")
			vmi = vmiFactory(withLivelinessProbe(livenessProbe))
			vmi = libvmops.RunVMIAndExpectLaunchIgnoreWarnings(vmi, 180)

			By("Checking that the VMI is in a final state after a while")
			Eventually(func() bool {
				vmi, err := virtClient.VirtualMachineInstance(testsuite.GetTestNamespace(vmi)).Get(context.Background(), vmi.Name, metav1.GetOptions{})
				Expect(err).ToNot(HaveOccurred())
				return vmi.IsFinal()
			}, 120, 1).Should(BeTrue())
		},
			Entry("[test_id:1217][posneg:negative]with working TCP probe and no running server", tcpProbe, libvmifact.NewAlpine),
			Entry("[test_id:1218][posneg:negative]with working HTTP probe and no running server", httpProbe, libvmifact.NewAlpine),
			Entry("[test_id:5880]with working Exec probe and invalid command", createExecProbe(period, initialSeconds, timeoutSeconds, "exit", "1"), libvmifact.NewFedora),
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
	vmi := libvmifact.NewAlpineWithTestTooling(libnet.WithMasqueradeNetworking(), withReadinessProbe(probe))
	return libvmops.RunVMIAndExpectLaunchIgnoreWarnings(vmi, 180)
}

func createReadyAlpineVMIWithLivenessProbe(probe *v1.Probe) *v1.VirtualMachineInstance {
	vmi := libvmifact.NewAlpineWithTestTooling(libnet.WithMasqueradeNetworking(), withLivelinessProbe(probe))

	return libvmops.RunVMIAndExpectLaunchIgnoreWarnings(vmi, 180)
}

func createTCPProbe(period int32, initialSeconds int32, port int) *v1.Probe {
	httpHandler := v1.Handler{
		TCPSocket: &k8sv1.TCPSocketAction{
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
		HTTPGet: &k8sv1.HTTPGetAction{
			Port: intstr.FromInt(port),
		},
	}
	return createProbeSpecification(period, initialSeconds, 1, httpHandler)
}

func createExecProbe(period int32, initialSeconds int32, timeoutSeconds int32, command ...string) *v1.Probe {
	execHandler := v1.Handler{Exec: &k8sv1.ExecAction{Command: command}}
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
		vmnetserver.StartHTTPServer(vmi, port, console.LoginToAlpine)
	} else {
		vmnetserver.StartTCPServer(vmi, port, console.LoginToAlpine)
	}
}

func pointIpv6ProbeToSupportPod(pod *k8sv1.Pod, probe *v1.Probe) (*v1.Probe, error) {
	supportPodIP := libnet.GetPodIPByFamily(pod, k8sv1.IPv6Protocol)
	if supportPodIP == "" {
		return nil, fmt.Errorf("pod/%s does not have an IPv6 address", pod.Name)
	}

	return patchProbeWithIPAddr(probe, supportPodIP), nil
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

func newHTTPServerPod(ipFamily, port int) *k8sv1.Pod {
	serverCommand := fmt.Sprintf("nc -%d -klp %d --sh-exec 'echo -e \"HTTP/1.1 200 OK\\nContent-Length: 12\\n\\nHello World!\"'", ipFamily, port)
	return libpod.RenderPrivilegedPod("http-hello-world-server", []string{"/bin/bash"}, []string{"-c", serverCommand})
}

func newTCPServerPod(ipFamily, port int) *k8sv1.Pod {
	serverCommand := fmt.Sprintf("nc -%d -klp %d --sh-exec 'echo \"Hello World!\"'", ipFamily, port)
	return libpod.RenderPrivilegedPod("tcp-hello-world-server", []string{"/bin/bash"}, []string{"-c", serverCommand})
}

func createPodAndWaitUntil(pod *k8sv1.Pod, phaseToWait k8sv1.PodPhase) *k8sv1.Pod {
	virtClient := kubevirt.Client()

	var err error
	pod, err = virtClient.CoreV1().Pods(testsuite.GetTestNamespace(pod)).Create(context.Background(), pod, metav1.CreateOptions{})
	Expect(err).ToNot(HaveOccurred(), "should succeed creating pod")

	getStatus := func() k8sv1.PodPhase {
		pod, err = virtClient.CoreV1().Pods(testsuite.GetTestNamespace(pod)).Get(context.Background(), pod.Name, metav1.GetOptions{})
		Expect(err).ToNot(HaveOccurred())
		return pod.Status.Phase
	}
	Eventually(getStatus, 30, 1).Should(Equal(phaseToWait), "should reach %s phase", phaseToWait)
	return pod
}

func startTCPServerPod(ipFamily, port int) *k8sv1.Pod {
	By(fmt.Sprintf("Start TCP Server pod at port %d", port))
	return createPodAndWaitUntil(newTCPServerPod(ipFamily, port), k8sv1.PodRunning)
}

func startHTTPServerPod(ipFamily, port int) *k8sv1.Pod {
	By(fmt.Sprintf("Start HTTP Server pod at port %d", port))
	return createPodAndWaitUntil(newHTTPServerPod(ipFamily, port), k8sv1.PodRunning)
}
