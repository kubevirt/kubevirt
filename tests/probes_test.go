package tests_test

import (
	"flag"

	. "github.com/onsi/ginkgo"
	"github.com/onsi/ginkgo/extensions/table"
	. "github.com/onsi/gomega"
	v1 "k8s.io/api/core/v1"
	v13 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"

	v12 "kubevirt.io/kubevirt/pkg/api/v1"
	"kubevirt.io/kubevirt/pkg/kubecli"
	"kubevirt.io/kubevirt/tests"
)

var _ = Describe("[ref_id:1182]Probes", func() {
	flag.Parse()

	virtClient, err := kubecli.GetKubevirtClient()
	tests.PanicOnError(err)

	BeforeEach(func() {
		tests.BeforeTestCleanup()
	})

	Context("for readiness", func() {

		tcpProbe := &v12.Probe{
			PeriodSeconds:       5,
			InitialDelaySeconds: 5,
			Handler: v12.Handler{

				TCPSocket: &v1.TCPSocketAction{
					Port: intstr.Parse("1500"),
				},
			},
		}

		httpProbe := &v12.Probe{
			PeriodSeconds:       5,
			InitialDelaySeconds: 5,
			Handler: v12.Handler{

				HTTPGet: &v1.HTTPGetAction{
					Port: intstr.Parse("1500"),
				},
			},
		}
		table.DescribeTable("should succeed", func(readinessProbe *v12.Probe, serverStarter func(vmi *v12.VirtualMachineInstance, port int)) {
			By("Specifying a VMI with a readiness probe")
			vmi := tests.NewRandomVMIWithEphemeralDiskAndUserdata(tests.ContainerDiskFor(tests.ContainerDiskCirros), "#!/bin/bash\necho 'hello'\n")
			vmi.Spec.ReadinessProbe = readinessProbe
			vmi, err = virtClient.VirtualMachineInstance(tests.NamespaceTestDefault).Create(vmi)
			Expect(err).ToNot(HaveOccurred())
			// It may come to modify retries on the VMI because of the kubelet updating the pod, which can trigger controllers more often
			tests.WaitForSuccessfulVMIStartIgnoreWarnings(vmi)

			Expect(podReady(tests.GetRunningPodByVirtualMachineInstance(vmi, tests.NamespaceTestDefault))).To(Equal(v1.ConditionFalse))
			vmi, err = virtClient.VirtualMachineInstance(vmi.Namespace).Get(vmi.Name, &v13.GetOptions{})
			Expect(err).ToNot(HaveOccurred())
			Expect(vmiReady(vmi)).To(Equal(v1.ConditionFalse))

			By("Starting the server inside the VMI")
			serverStarter(vmi, 1500)

			By("Checking that the VMI and the pod will be marked as ready to receive traffic")
			Eventually(func() v1.ConditionStatus {
				vmi, err = virtClient.VirtualMachineInstance(vmi.Namespace).Get(vmi.Name, &v13.GetOptions{})
				Expect(err).ToNot(HaveOccurred())
				return vmiReady(vmi)
			}, 60, 1).Should(Equal(v1.ConditionTrue))
			Expect(podReady(tests.GetRunningPodByVirtualMachineInstance(vmi, tests.NamespaceTestDefault))).To(Equal(v1.ConditionTrue))
		},
			table.Entry("[test_id:1202][posneg:positive]with working TCP probe and tcp server", tcpProbe, tests.StartTCPServer),
			table.Entry("[test_id:1200][posneg:positive]with working HTTP probe and http server", httpProbe, tests.StartHTTPServer),
		)

		table.DescribeTable("should fail", func(readinessProbe *v12.Probe) {
			By("Specifying a VMI with a readiness probe")
			vmi := tests.NewRandomVMIWithEphemeralDiskAndUserdata(tests.ContainerDiskFor(tests.ContainerDiskCirros), "#!/bin/bash\necho 'hello'\n")
			vmi.Spec.ReadinessProbe = readinessProbe
			vmi, err = virtClient.VirtualMachineInstance(tests.NamespaceTestDefault).Create(vmi)
			Expect(err).ToNot(HaveOccurred())
			// It may come to modify retries on the VMI because of the kubelet updating the pod, which can trigger controllers more often
			tests.WaitForSuccessfulVMIStartIgnoreWarnings(vmi)

			Expect(podReady(tests.GetRunningPodByVirtualMachineInstance(vmi, tests.NamespaceTestDefault))).To(Equal(v1.ConditionFalse))
			vmi, err = virtClient.VirtualMachineInstance(vmi.Namespace).Get(vmi.Name, &v13.GetOptions{})
			Expect(err).ToNot(HaveOccurred())
			Expect(vmiReady(vmi)).To(Equal(v1.ConditionFalse))

			By("Checking that the VMI and the pod will consistently stay in a not-ready state")
			Consistently(func() v1.ConditionStatus {
				vmi, err = virtClient.VirtualMachineInstance(vmi.Namespace).Get(vmi.Name, &v13.GetOptions{})
				Expect(err).ToNot(HaveOccurred())
				return vmiReady(vmi)
			}, 60, 1).Should(Equal(v1.ConditionFalse))
			Expect(podReady(tests.GetRunningPodByVirtualMachineInstance(vmi, tests.NamespaceTestDefault))).To(Equal(v1.ConditionFalse))
		},
			table.Entry("[test_id:1220][posneg:negative]with working TCP probe and no running server", tcpProbe),
			table.Entry("[test_id:1219][posneg:negative]with working HTTP probe and no running server", httpProbe),
		)
	})

	Context("for liveness", func() {

		tcpProbe := &v12.Probe{
			PeriodSeconds:       5,
			InitialDelaySeconds: 90,
			Handler: v12.Handler{

				TCPSocket: &v1.TCPSocketAction{
					Port: intstr.Parse("1500"),
				},
			},
		}

		httpProbe := &v12.Probe{
			PeriodSeconds:       5,
			InitialDelaySeconds: 90,
			Handler: v12.Handler{

				HTTPGet: &v1.HTTPGetAction{
					Port: intstr.Parse("1500"),
				},
			},
		}
		table.DescribeTable("should not fail the VMI", func(livenessProbe *v12.Probe, serverStarter func(vmi *v12.VirtualMachineInstance, port int)) {
			By("Specifying a VMI with a readiness probe")
			vmi := tests.NewRandomVMIWithEphemeralDiskAndUserdata(tests.ContainerDiskFor(tests.ContainerDiskCirros), "#!/bin/bash\necho 'hello'\n")
			vmi.Spec.LivenessProbe = livenessProbe
			vmi, err = virtClient.VirtualMachineInstance(tests.NamespaceTestDefault).Create(vmi)
			Expect(err).ToNot(HaveOccurred())
			// It may come to modify retries on the VMI because of the kubelet updating the pod, which can trigger controllers more often
			tests.WaitForSuccessfulVMIStartIgnoreWarnings(vmi)

			By("Starting the server inside the VMI")
			serverStarter(vmi, 1500)

			By("Checking that the VMI is still running after a minute")
			Consistently(func() bool {
				vmi, err := virtClient.VirtualMachineInstance(tests.NamespaceTestDefault).Get(vmi.Name, &v13.GetOptions{})
				Expect(err).ToNot(HaveOccurred())
				return vmi.IsFinal()
			}, 120, 1).Should(Not(BeTrue()))
		},
			table.Entry("[test_id:1199][posneg:positive]with working TCP probe and tcp server", tcpProbe, tests.StartTCPServer),
			table.Entry("[test_id:1201][posneg:positive]with working HTTP probe and http server", httpProbe, tests.StartHTTPServer),
		)

		table.DescribeTable("should fail the VMI", func(livenessProbe *v12.Probe) {
			By("Specifying a VMI with a readiness probe")
			vmi := tests.NewRandomVMIWithEphemeralDiskAndUserdata(tests.ContainerDiskFor(tests.ContainerDiskCirros), "#!/bin/bash\necho 'hello'\n")
			vmi.Spec.LivenessProbe = livenessProbe
			vmi, err = virtClient.VirtualMachineInstance(tests.NamespaceTestDefault).Create(vmi)
			Expect(err).ToNot(HaveOccurred())
			// It may come to modify retries on the VMI because of the kubelet updating the pod, which can trigger controllers more often
			tests.WaitForSuccessfulVMIStartIgnoreWarnings(vmi)

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

func podReady(pod *v1.Pod) v1.ConditionStatus {
	for _, cond := range pod.Status.Conditions {
		if cond.Type == v1.PodReady {
			return cond.Status
		}
	}
	return v1.ConditionFalse
}

func vmiReady(vmi *v12.VirtualMachineInstance) v1.ConditionStatus {
	for _, cond := range vmi.Status.Conditions {
		if cond.Type == v12.VirtualMachineInstanceConditionType(v1.PodReady) {
			return cond.Status
		}
	}
	return v1.ConditionFalse
}
