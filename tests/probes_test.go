package tests_test

import (
	"context"
	"fmt"

	. "github.com/onsi/ginkgo"
	"github.com/onsi/ginkgo/extensions/table"
	. "github.com/onsi/gomega"
	v1 "k8s.io/api/core/v1"
	v13 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"

	v12 "kubevirt.io/client-go/api/v1"
	"kubevirt.io/client-go/kubecli"
	"kubevirt.io/kubevirt/tests"
	cd "kubevirt.io/kubevirt/tests/containerdisk"
	"kubevirt.io/kubevirt/tests/libnet"
)

var _ = FDescribe("[ref_id:1182]Probes", func() {
	var (
		err           error
		virtClient    kubecli.KubevirtClient
		vmi           *v12.VirtualMachineInstance
		blankIPFamily = *new(v1.IPFamily)
	)

	BeforeEach(func() {
		virtClient, err = kubecli.GetKubevirtClient()
		tests.PanicOnError(err)

		tests.BeforeTestCleanup()
	})

	buildProbeBackendPodSpec := func(probe *v12.Probe) (*v1.Pod, func() error) {
		isHTTPProbe := probe.Handler.HTTPGet != nil
		var probeBackendPod *v1.Pod
		if isHTTPProbe {
			port := probe.HTTPGet.Port.IntVal
			probeBackendPod = tests.StartHTTPServerPod(int(port))
		} else {
			port := probe.TCPSocket.Port.IntVal
			probeBackendPod = tests.StartTCPServerPod(int(port))
		}
		return probeBackendPod, func() error {
			return virtClient.CoreV1().Pods(tests.NamespaceTestDefault).Delete(context.Background(), probeBackendPod.Name, v13.DeleteOptions{})
		}
	}

	Context("for readiness", func() {
		const (
			period         = 5
			initialSeconds = 5
			port           = 1500
		)

		tcpProbe := createTCPProbe(period, initialSeconds, port)
		httpProbe := createHTTPProbe(period, initialSeconds, port)

		isVMIReady := func() bool {
			readVmi, err := virtClient.VirtualMachineInstance(vmi.Namespace).Get(vmi.Name, &v13.GetOptions{})
			Expect(err).ToNot(HaveOccurred())
			return vmiReady(readVmi) == v1.ConditionTrue
		}

		table.DescribeTable("should succeed", func(readinessProbe *v12.Probe, IPFamily v1.IPFamily) {

			if IPFamily == v1.IPv6Protocol {
				libnet.SkipWhenNotDualStackCluster(virtClient)
				By("Create a support pod which will reply to kubelet's probes ...")
				probeBackendPod, supportPodCleanupFunc := buildProbeBackendPodSpec(readinessProbe)
				defer func() {
					Expect(supportPodCleanupFunc()).To(Succeed(), "The support pod responding to the probes should be cleaned-up at test tear-down.")
				}()

				By("Attaching the readiness probe to an external pod server")
				readinessProbe, err = pointProbeToSupportPod(probeBackendPod, IPFamily, readinessProbe)
				Expect(err).ToNot(HaveOccurred(), "should attach the backend pod with readiness probe")

				By("Specifying a VMI with a readiness probe")
				vmi = createReadyCirrosVMIWithReadinessProbe(virtClient, readinessProbe)
			} else {
				By("Specifying a VMI with a readiness probe")
				vmi = createReadyCirrosVMIWithReadinessProbe(virtClient, readinessProbe)

				By("Starting the server inside the VMI")
				serverStarter(vmi, readinessProbe, 1500)
			}

			// pod is not ready until our probe contacts the server
			assertPodNotReady(virtClient, vmi)

			By("Checking that the VMI and the pod will be marked as ready to receive traffic")
			Eventually(isVMIReady, 60, 1).Should(Equal(true))
			Expect(tests.PodReady(tests.GetRunningPodByVirtualMachineInstance(vmi, tests.NamespaceTestDefault))).To(Equal(v1.ConditionTrue))
		},
			table.Entry("[test_id:1202][posneg:positive]with working TCP probe and tcp server,no ip family specified ", tcpProbe, blankIPFamily),
			table.Entry("[test_id:1202][posneg:positive]with working TCP probe and tcp server on ipv4", tcpProbe, v1.IPv4Protocol),
			table.Entry("[test_id:1202][posneg:positive]with working TCP probe and tcp server on ipv6", tcpProbe, v1.IPv6Protocol),
			table.Entry("[QUARANTINE][owner:@sig-network][test_id:1202][posneg:positive]with working HTTP probe and http server, no ip family is specified ", httpProbe, blankIPFamily),
			table.Entry("[test_id:1200][posneg:positive]with working HTTP probe and http server on ipv4", httpProbe, v1.IPv4Protocol),
			table.Entry("[test_id:1200][posneg:positive]with working HTTP probe and http server on ipv6", httpProbe, v1.IPv6Protocol),
		)

		table.DescribeTable("should fail", func(readinessProbe *v12.Probe) {
			By("Specifying a VMI with a readiness probe")
			vmi = createReadyCirrosVMIWithReadinessProbe(virtClient, readinessProbe)

			// pod is not ready until our probe contacts the server
			assertPodNotReady(virtClient, vmi)

			By("Checking that the VMI and the pod will consistently stay in a not-ready state")
			Consistently(isVMIReady).Should(Equal(false))
			Expect(tests.PodReady(tests.GetRunningPodByVirtualMachineInstance(vmi, tests.NamespaceTestDefault))).To(Equal(v1.ConditionFalse))
		},
			table.Entry("[test_id:1220][posneg:negative]with working TCP probe and no running server", tcpProbe),
			table.Entry("[test_id:1219][posneg:negative]with working HTTP probe and no running server", httpProbe),
		)
	})

	Context("for liveness", func() {
		const (
			period         = 5
			initialSeconds = 90
			port           = 1500
		)

		tcpProbe := createTCPProbe(period, initialSeconds, port)
		httpProbe := createHTTPProbe(period, initialSeconds, port)

		table.DescribeTable("should not fail the VMI", func(livenessProbe *v12.Probe, IPFamily v1.IPFamily) {

			if IPFamily == v1.IPv6Protocol {
				libnet.SkipWhenNotDualStackCluster(virtClient)

				By("Create a support pod which will reply to kubelet's probes ...")
				probeBackendPod, supportPodCleanupFunc := buildProbeBackendPodSpec(livenessProbe)
				defer func() {
					Expect(supportPodCleanupFunc()).To(Succeed(), "The support pod responding to the probes should be cleaned-up at test tear-down.")
				}()

				By("Attaching the liveness probe to an external pod server")
				livenessProbe, err = pointProbeToSupportPod(probeBackendPod, IPFamily, livenessProbe)
				Expect(err).ToNot(HaveOccurred(), "should attach the backend pod with livness probe")

				By("Specifying a VMI with a readiness probe")
				vmi = createReadyCirrosVMIWithLivenessProbe(virtClient, livenessProbe)
			} else {
				By("Specifying a VMI with a readiness probe")
				vmi = createReadyCirrosVMIWithLivenessProbe(virtClient, livenessProbe)

				By("Starting the server inside the VMI")
				serverStarter(vmi, livenessProbe, 1500)
			}

			By("Checking that the VMI is still running after a minute")
			Consistently(func() bool {
				vmi, err := virtClient.VirtualMachineInstance(tests.NamespaceTestDefault).Get(vmi.Name, &v13.GetOptions{})
				Expect(err).ToNot(HaveOccurred())
				return vmi.IsFinal()
			}, 120, 1).Should(Not(BeTrue()))
		},
			table.Entry("[test_id:1199][posneg:positive]with working TCP probe and tcp server, no ip family is specified", tcpProbe, blankIPFamily),
			table.Entry("[test_id:1199][posneg:positive]with working TCP probe and tcp server on ipv4", tcpProbe, v1.IPv4Protocol),
			table.Entry("[test_id:1199][posneg:positive]with working TCP probe and tcp server on ipv6", tcpProbe, v1.IPv6Protocol),
			table.Entry("[test_id:1201][posneg:positive]with working HTTP probe and http server, no ip family is specified", httpProbe, blankIPFamily),
			table.Entry("[test_id:1201][posneg:positive]with working HTTP probe and http server on ipv4", httpProbe, v1.IPv4Protocol),
			table.Entry("[test_id:1201][posneg:positive]with working HTTP probe and http server on ipv6", httpProbe, v1.IPv6Protocol),
		)

		table.DescribeTable("should fail the VMI", func(livenessProbe *v12.Probe) {
			By("Specifying a VMI with a livenessProbe probe")
			vmi := createReadyCirrosVMIWithLivenessProbe(virtClient, livenessProbe)

			By("Checking that the VMI is in a final state after a minute")
			Eventually(func() bool {
				vmi, err := virtClient.VirtualMachineInstance(tests.NamespaceTestDefault).Get(vmi.Name, &v13.GetOptions{})
				Expect(err).ToNot(HaveOccurred())
				return vmi.IsFinal()
			}, 120, 1).Should(BeTrue())
		},
			table.Entry("[test_id:1217][posneg:negative]with working TCP probe and no running server", tcpProbe),
			table.Entry("[test_id:1218][posneg:negative]with working HTTP probe and no running server", httpProbe),
		)
	})
})

func createReadyCirrosVMIWithReadinessProbe(virtClient kubecli.KubevirtClient, probe *v12.Probe) *v12.VirtualMachineInstance {
	dummyUserData := "#!/bin/bash\necho 'hello'\n"
	vmi := tests.NewRandomVMIWithEphemeralDiskAndUserdata(
		cd.ContainerDiskFor(cd.ContainerDiskCirros), dummyUserData)
	vmi.Spec.ReadinessProbe = probe

	return createAndBlockUntilVMIHasStarted(virtClient, vmi)
}

func createReadyCirrosVMIWithLivenessProbe(virtClient kubecli.KubevirtClient, probe *v12.Probe) *v12.VirtualMachineInstance {
	dummyUserData := "#!/bin/bash\necho 'hello'\n"
	vmi := tests.NewRandomVMIWithEphemeralDiskAndUserdata(
		cd.ContainerDiskFor(cd.ContainerDiskCirros), dummyUserData)
	vmi.Spec.LivenessProbe = probe

	return createAndBlockUntilVMIHasStarted(virtClient, vmi)
}

func createAndBlockUntilVMIHasStarted(virtClient kubecli.KubevirtClient, vmi *v12.VirtualMachineInstance) *v12.VirtualMachineInstance {
	_, err := virtClient.VirtualMachineInstance(tests.NamespaceTestDefault).Create(vmi)
	Expect(err).ToNot(HaveOccurred())

	// It may come to modify retries on the VMI because of the kubelet updating the pod, which can trigger controllers more often
	tests.WaitForSuccessfulVMIStartIgnoreWarnings(vmi)

	// read back the created VMI, so it has the UID available on it
	startedVMI, err := virtClient.VirtualMachineInstance(tests.NamespaceTestDefault).Get(vmi.Name, &v13.GetOptions{})
	Expect(err).ToNot(HaveOccurred())
	return startedVMI
}

func vmiReady(vmi *v12.VirtualMachineInstance) v1.ConditionStatus {
	for _, cond := range vmi.Status.Conditions {
		if cond.Type == v12.VirtualMachineInstanceConditionType(v1.PodReady) {
			return cond.Status
		}
	}
	return v1.ConditionFalse
}

func assertPodNotReady(virtClient kubecli.KubevirtClient, vmi *v12.VirtualMachineInstance) {
	Expect(tests.PodReady(tests.GetRunningPodByVirtualMachineInstance(vmi, tests.NamespaceTestDefault))).To(Equal(v1.ConditionFalse))
	readVmi, err := virtClient.VirtualMachineInstance(vmi.Namespace).Get(vmi.Name, &v13.GetOptions{})
	Expect(err).ToNot(HaveOccurred())
	Expect(vmiReady(readVmi)).To(Equal(v1.ConditionFalse))
}

func createTCPProbe(period int32, initialSeconds int32, port int) *v12.Probe {
	httpHandler := v12.Handler{
		TCPSocket: &v1.TCPSocketAction{
			Port: intstr.FromInt(port),
		},
	}
	return createProbeSpecification(period, initialSeconds, httpHandler)
}

func patchProbeWithIPAddr(existingProbe *v12.Probe, ipHostIP string) *v12.Probe {
	if isHTTPProbe(*existingProbe) {
		existingProbe.HTTPGet.Host = ipHostIP
	} else {
		existingProbe.TCPSocket.Host = ipHostIP
	}
	return existingProbe
}

func createHTTPProbe(period int32, initialSeconds int32, port int) *v12.Probe {
	httpHandler := v12.Handler{
		HTTPGet: &v1.HTTPGetAction{
			Port: intstr.FromInt(port),
		},
	}
	return createProbeSpecification(period, initialSeconds, httpHandler)
}

func createProbeSpecification(period int32, initialSeconds int32, handler v12.Handler) *v12.Probe {
	return &v12.Probe{
		PeriodSeconds:       period,
		InitialDelaySeconds: initialSeconds,
		Handler:             handler,
	}
}

func isHTTPProbe(probe v12.Probe) bool {
	return probe.Handler.HTTPGet != nil
}

func serverStarter(vmi *v12.VirtualMachineInstance, probe *v12.Probe, port int) {
	if isHTTPProbe(*probe) {
		tests.StartHTTPServer(vmi, port)
	} else {
		tests.StartTCPServer(vmi, port)
	}
}

func pointProbeToSupportPod(pod *v1.Pod, IPFamily v1.IPFamily, probe *v12.Probe) (*v12.Probe, error) {
	supportPodIP := libnet.GetPodIpByFamily(pod, IPFamily)
	if supportPodIP == "" {
		return nil, fmt.Errorf("pod's %s %s IP address does not exist", pod.Name, IPFamily)
	}

	return patchProbeWithIPAddr(probe, supportPodIP), nil
}
