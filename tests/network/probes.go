package network

import (
	"context"
	"time"

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
	specifyingVMReadinessProbe          = "Specifying a VMI with a readiness probe"
	specifyingVMStartupAndLivenessProbe = "Specifying a VMI with startup and liveness probes"
)

var _ = Describe(SIG("[ref_id:1182]Probes", func() {
	var (
		err        error
		virtClient kubecli.KubevirtClient
		vmi        *v1.VirtualMachineInstance
	)

	BeforeEach(func() {
		virtClient = kubevirt.Client()
	})

	Context("for readiness", func() {
		const (
			period         = 5
			initialSeconds = 5
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

		It("should fail when there is no TCP server listening inside the guest", func() {
			By(specifyingVMReadinessProbe)
			readinessProbe := createTCPProbe(period, initialSeconds, port)
			vmi = libvmifact.NewAlpine(withReadinessProbe(readinessProbe))
			vmi = libvmops.RunVMIAndExpectLaunchIgnoreWarnings(vmi, 180)

			By("Checking that the VMI is consistently non-ready")
			Consistently(matcher.ThisVMI(vmi)).
				WithTimeout(30 * time.Second).
				WithPolling(100 * time.Millisecond).
				Should(matcher.HaveConditionMissingOrFalse(v1.VirtualMachineInstanceReady))
		})
	})

	Context("for startup and liveness", func() {
		const (
			period               = 5
			initialSeconds       = 5
			port                 = 1500
			failureThreshold     = 30
			livenessInitialDelay = 30
		)

		It("should succeed with working TCP startup and liveness probes and tcp server on ipv4", func() {
			libnet.SkipWhenClusterNotSupportIPFamily(k8sv1.IPv4Protocol)

			By(specifyingVMStartupAndLivenessProbe)
			startupProbe := createTCPProbeWithFailureThreshold(period, initialSeconds, port, failureThreshold)
			livenessProbe := createTCPProbe(period, livenessInitialDelay, port)
			vmi = createReadyAlpineVMIWithStartupAndLivenessProbe(startupProbe, livenessProbe)

			Expect(matcher.ThisVMI(vmi)()).To(matcher.HaveConditionMissingOrFalse(v1.VirtualMachineInstanceReady))

			By("Starting the server inside the VMI")
			Eventually(matcher.ThisVMI(vmi)).WithTimeout(3 * time.Minute).WithPolling(3 * time.Second).Should(matcher.HaveConditionTrue(v1.VirtualMachineInstanceAgentConnected))
			vmi, err = kubevirt.Client().VirtualMachineInstance(testsuite.GetTestNamespace(nil)).Get(context.Background(), vmi.Name, metav1.GetOptions{})
			Expect(err).ToNot(HaveOccurred())
			vmnetserver.StartTCPServer(vmi, 1500, console.LoginToAlpine)

			By("Checking that the VMI becomes ready after startup probe succeeds")
			Eventually(matcher.ThisVMI(vmi)).
				WithTimeout(2 * time.Minute).
				WithPolling(2 * time.Second).
				Should(matcher.HaveConditionTrue(v1.VirtualMachineInstanceReady))

			By("Checking that the VMI remains running with liveness probe succeeding")
			Consistently(func() bool {
				vmi, err := virtClient.VirtualMachineInstance(testsuite.GetTestNamespace(vmi)).Get(context.Background(), vmi.Name, metav1.GetOptions{})
				Expect(err).ToNot(HaveOccurred())
				return vmi.IsFinal()
			}).WithTimeout(2 * time.Minute).
				WithPolling(1 * time.Second).
				Should(Not(BeTrue()))
		})

		It("should fail when startup probe fails and liveness probe is not evaluated", func() {
			By(specifyingVMStartupAndLivenessProbe)
			startupProbe := createTCPProbeWithFailureThreshold(period, initialSeconds, port, failureThreshold)
			livenessProbe := createTCPProbe(period, livenessInitialDelay, port)
			vmi = libvmifact.NewAlpine(withStartupProbe(startupProbe), withLivenessProbe(livenessProbe))
			vmi = libvmops.RunVMIAndExpectLaunchIgnoreWarnings(vmi, 180)

			By("Checking that the VMI is consistently non-ready")
			Consistently(matcher.ThisVMI(vmi)).
				WithTimeout(30 * time.Second).
				WithPolling(100 * time.Millisecond).
				Should(matcher.HaveConditionMissingOrFalse(v1.VirtualMachineInstanceReady))

			By("Checking that the VMI enters a final state after startup probe failure")
			Eventually(func() bool {
				vmi, err := virtClient.VirtualMachineInstance(testsuite.GetTestNamespace(vmi)).Get(context.Background(), vmi.Name, metav1.GetOptions{})
				Expect(err).ToNot(HaveOccurred())
				return vmi.IsFinal()
			}).WithTimeout(3 * time.Minute).
				WithPolling(1 * time.Second).
				Should(BeTrue())
		})
	})
}))

func createReadyAlpineVMIWithReadinessProbe(probe *v1.Probe) *v1.VirtualMachineInstance {
	vmi := libvmifact.NewAlpineWithTestTooling(libnet.WithMasqueradeNetworking(), withReadinessProbe(probe))
	return libvmops.RunVMIAndExpectLaunchIgnoreWarnings(vmi, 180)
}

func createReadyAlpineVMIWithStartupAndLivenessProbe(startupProbe, livenessProbe *v1.Probe) *v1.VirtualMachineInstance {
	vmi := libvmifact.NewAlpineWithTestTooling(libnet.WithMasqueradeNetworking(), withStartupProbe(startupProbe), withLivenessProbe(livenessProbe))
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
func createTCPProbeWithFailureThreshold(period, initialSeconds, port int, failureThreshold int32) *v1.Probe {
	httpHandler := v1.Handler{
		TCPSocket: &k8sv1.TCPSocketAction{
			Port: intstr.FromInt(port),
		},
	}
	return createProbeSpecificationWithFailureThreshold(int32(period), int32(initialSeconds), 1, httpHandler, failureThreshold)
}

func createProbeSpecification(period, initialSeconds, timeoutSeconds int32, handler v1.Handler) *v1.Probe {
	return &v1.Probe{
		PeriodSeconds:       period,
		InitialDelaySeconds: initialSeconds,
		Handler:             handler,
		TimeoutSeconds:      timeoutSeconds,
	}
}

func createProbeSpecificationWithFailureThreshold(period, initialSeconds, timeoutSeconds int32, handler v1.Handler, failureThreshold int32) *v1.Probe {
	return &v1.Probe{
		PeriodSeconds:       period,
		InitialDelaySeconds: initialSeconds,
		Handler:             handler,
		TimeoutSeconds:      timeoutSeconds,
		FailureThreshold:    failureThreshold,
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

func withStartupProbe(probe *v1.Probe) libvmi.Option {
	return func(vmi *v1.VirtualMachineInstance) {
		vmi.Spec.StartupProbe = probe
	}
}
