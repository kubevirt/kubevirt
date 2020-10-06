package tests_test

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	. "github.com/onsi/ginkgo"
	"github.com/onsi/ginkgo/extensions/table"
	. "github.com/onsi/gomega"
	batchv1 "k8s.io/api/batch/v1"
	k8sv1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	k8smetav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	v1 "kubevirt.io/client-go/api/v1"
	"kubevirt.io/client-go/kubecli"
	"kubevirt.io/kubevirt/pkg/virtctl/expose"
	"kubevirt.io/kubevirt/tests"
	cd "kubevirt.io/kubevirt/tests/containerdisk"
	"kubevirt.io/kubevirt/tests/libnet"
)

func newLabeledVMI(label string, virtClient kubecli.KubevirtClient, createVMI bool) (vmi *v1.VirtualMachineInstance) {
	ports := []v1.Port{{Name: "http", Port: 80},
		{Name: "test-port-tcp", Port: 1500, Protocol: "TCP"},
		{Name: "udp", Port: 82, Protocol: "UDP"},
		{Name: "test-port-udp", Port: 1500, Protocol: "UDP"}}
	vmi = tests.NewRandomVMIWithMasqueradeInterfaceEphemeralDiskAndUserdata(cd.ContainerDiskFor(cd.ContainerDiskCirros), "#!/bin/bash\necho 'hello'\n", ports)
	vmi.Labels = map[string]string{"expose": label}

	var err error
	if createVMI {
		vmi, err = virtClient.VirtualMachineInstance(tests.NamespaceTestDefault).Create(vmi)
		Expect(err).ToNot(HaveOccurred())
		tests.WaitForSuccessfulVMIStartIgnoreWarnings(vmi)
		vmi, err = virtClient.VirtualMachineInstance(tests.NamespaceTestDefault).Get(vmi.ObjectMeta.Name, &k8smetav1.GetOptions{})
		Expect(err).ToNot(HaveOccurred())
		tests.WaitUntilVMIReady(vmi, tests.LoggedInCirrosExpecter)
	}
	return
}

var _ = Describe("[rfe_id:253][crit:medium][vendor:cnv-qe@redhat.com][level:component]Expose", func() {

	var virtClient kubecli.KubevirtClient
	var err error

	const testPort = 1500

	BeforeEach(func() {
		virtClient, err = kubecli.GetKubevirtClient()
		tests.PanicOnError(err)
	})
	runHelloWorldJob := func(host, port, namespace string) *batchv1.Job {
		job := tests.NewHelloWorldJob(host, port)
		job, err := virtClient.BatchV1().Jobs(namespace).Create(job)
		ExpectWithOffset(1, err).ToNot(HaveOccurred())
		return job
	}

	runHelloWorldJobUDP := func(host, port, namespace string) *batchv1.Job {
		job := tests.NewHelloWorldJobUDP(host, port)
		job, err := virtClient.BatchV1().Jobs(namespace).Create(job)
		ExpectWithOffset(1, err).ToNot(HaveOccurred())
		return job
	}

	runHelloWorldJobHttp := func(host, port, namespace string) *batchv1.Job {
		job := tests.NewHelloWorldJobHTTP(host, port)
		job, err := virtClient.BatchV1().Jobs(namespace).Create(job)
		ExpectWithOffset(1, err).ToNot(HaveOccurred())
		return job
	}

	cleanupService := func(serviceName string, namespace string) func() error {
		return func() error {
			return virtClient.CoreV1().Services(namespace).Delete(serviceName, &k8smetav1.DeleteOptions{})
		}
	}

	cleanupJob := func(jobName string, namespace string) func() error {
		return func() error {
			return virtClient.BatchV1().Jobs(namespace).Delete(jobName, &k8smetav1.DeleteOptions{})
		}
	}

	Context("Expose service on a VM", func() {
		var tcpVM *v1.VirtualMachineInstance
		tests.BeforeAll(func() {
			tests.BeforeTestCleanup()
			tcpVM = newLabeledVMI("vm", virtClient, true)
			tests.GenerateHelloWorldServer(tcpVM, testPort, "tcp")
		})

		Context("Expose ClusterIP service", func() {
			const servicePort = "27017"
			const serviceName = "cluster-ip-vmi"

			var vmiExposeArgs []string

			var jobCleanupFunc func() error
			var serviceCleanupFunc func() error

			BeforeEach(func() {
				vmiExposeArgs = []string{
					expose.COMMAND_EXPOSE,
					"virtualmachineinstance", "--namespace", tcpVM.GetNamespace(), tcpVM.GetName(),
					"--port", servicePort, "--name", serviceName,
					"--target-port", strconv.Itoa(testPort),
				}
			})

			BeforeEach(func() {
				jobCleanupFunc = nil
			})

			BeforeEach(func() {
				serviceCleanupFunc = nil
			})

			AfterEach(func() {
				Expect(jobCleanupFunc).NotTo(BeNil(), "a successful test must have stored a way to delete the batchv1.Job entity")
				Expect(jobCleanupFunc()).To(Succeed(), "should be able to delete the batchv1.Job entity")
			})

			AfterEach(func() {
				Expect(serviceCleanupFunc).NotTo(BeNil(), "a successful test must have stored a way to delete the k8sv1.Service entity")
				Expect(serviceCleanupFunc()).To(Succeed(), "should be able to delete the k8sv1.Service entity")
			})

			table.DescribeTable("[test_id:1531][label:masquerade_binding_connectivity]Should expose a Cluster IP service on a VMI and connect to it", func(ipFamily k8sv1.IPFamily) {
				if ipFamily == k8sv1.IPv6Protocol {
					vmiExposeArgs = append(vmiExposeArgs, "--ip-family", "ipv6")
				}
				By("Exposing the service via virtctl command")
				virtctl := tests.NewRepeatableVirtctlCommand(vmiExposeArgs...)
				err := virtctl()
				Expect(err).ToNot(HaveOccurred())

				By("Getting back the cluster IP given for the service")
				svc, err := virtClient.CoreV1().Services(tcpVM.Namespace).Get(serviceName, k8smetav1.GetOptions{})
				Expect(err).ToNot(HaveOccurred())
				serviceCleanupFunc = cleanupService(serviceName, tcpVM.GetNamespace())
				serviceIP := svc.Spec.ClusterIP

				By("Starting a job which tries to reach the VMI via ClusterIP")
				job := runHelloWorldJob(serviceIP, servicePort, tcpVM.Namespace)
				jobCleanupFunc = cleanupJob(job.GetName(), job.GetNamespace())

				By("Waiting for the job to report a successful connection attempt")
				Expect(tests.WaitForJobToSucceed(job, 420*time.Second)).To(Succeed())
			},
				table.Entry("over default IPv4 IP family", k8sv1.IPv4Protocol),
				table.Entry("over IPv6 IP family", k8sv1.IPv6Protocol),
			)
		})

		Context("Expose ClusterIP service with string target-port", func() {
			const servicePort = "27017"
			const serviceName = "cluster-ip-target-vmi"

			var vmiExposeArgs []string

			var serviceCleanupFunc func() error

			BeforeEach(func() {
				vmiExposeArgs = []string{
					expose.COMMAND_EXPOSE,
					"virtualmachineinstance", "--namespace", tcpVM.GetNamespace(), tcpVM.GetName(),
					"--port", servicePort, "--name", serviceName, "--target-port", "http",
				}
			})

			BeforeEach(func() {
				serviceCleanupFunc = nil
			})

			AfterEach(func() {
				Expect(serviceCleanupFunc).NotTo(BeNil(), "a successful test must have stored a way to delete the k8sv1.Service entity")
				Expect(serviceCleanupFunc()).To(Succeed(), "should be able to delete the k8sv1.Service entity")
			})

			table.DescribeTable("[test_id:1532]Should expose a ClusterIP service and connect to the vm on port 80", func(ipFamily k8sv1.IPFamily) {
				if ipFamily == k8sv1.IPv6Protocol {
					vmiExposeArgs = append(vmiExposeArgs, "--ip-family", "ipv6")
				}
				virtctl := tests.NewRepeatableVirtctlCommand(vmiExposeArgs...)
				err := virtctl()
				Expect(err).ToNot(HaveOccurred())

				By("Getting back the service so we can clean it up later on")
				service, err := virtClient.CoreV1().Services(tcpVM.Namespace).Get(serviceName, k8smetav1.GetOptions{})
				Expect(err).ToNot(HaveOccurred())
				serviceCleanupFunc = cleanupService(service.GetName(), service.GetNamespace())

				By("Waiting for kubernetes to create the relevant endpoint")
				getEndpoint := func() error {
					_, err := virtClient.CoreV1().Endpoints(tests.NamespaceTestDefault).Get(serviceName, k8smetav1.GetOptions{})
					return err
				}
				Eventually(getEndpoint, 60, 1).Should(BeNil())

				endpoints, err := virtClient.CoreV1().Endpoints(tests.NamespaceTestDefault).Get(serviceName, k8smetav1.GetOptions{})
				Expect(err).ToNot(HaveOccurred())

				Expect(len(endpoints.Subsets)).To(Equal(1))
				endpoint := endpoints.Subsets[0]
				Expect(len(endpoint.Ports)).To(Equal(1))
				Expect(endpoint.Ports[0].Port).To(Equal(int32(80)))
			},
				table.Entry("over default IPv4 IP family", k8sv1.IPv4Protocol),
				table.Entry("over IPv6 IP family", k8sv1.IPv6Protocol),
			)
		})

		Context("Expose ClusterIP service with ports on the vmi defined", func() {
			const serviceName = "cluster-ip-target-multiple-ports-vmi"

			var vmiExposeArgs []string

			var serviceCleanupFunc func() error

			BeforeEach(func() {
				vmiExposeArgs = []string{
					expose.COMMAND_EXPOSE,
					"virtualmachineinstance", "--namespace", tcpVM.GetNamespace(), tcpVM.GetName(),
					"--name", serviceName,
				}
			})

			BeforeEach(func() {
				serviceCleanupFunc = nil
			})

			AfterEach(func() {
				Expect(serviceCleanupFunc).NotTo(BeNil(), "a successful test must have stored a way to delete the k8sv1.Service entity")
				Expect(serviceCleanupFunc()).To(Succeed(), "should be able to delete the k8sv1.Service entity")
			})

			table.DescribeTable("[test_id:1533]Should expose a ClusterIP service and connect to all ports defined on the vmi", func(ipFamily k8sv1.IPFamily) {
				if ipFamily == k8sv1.IPv6Protocol {
					vmiExposeArgs = append(vmiExposeArgs, "--ip-family", "ipv6")
				}
				virtctl := tests.NewRepeatableVirtctlCommand(vmiExposeArgs...)
				Expect(virtctl()).To(Succeed(), "should expose a service via `virtctl expose ...`")

				By("Getting back the service so we can clean it up later on")
				service, err := virtClient.CoreV1().Services(tcpVM.Namespace).Get(serviceName, k8smetav1.GetOptions{})
				Expect(err).ToNot(HaveOccurred())
				serviceCleanupFunc = cleanupService(service.GetName(), service.GetNamespace())

				By("Waiting for kubernetes to create the relevant endpoint")
				getEndpoint := func() error {
					_, err := virtClient.CoreV1().Endpoints(tests.NamespaceTestDefault).Get(serviceName, k8smetav1.GetOptions{})
					return err
				}
				Eventually(getEndpoint, 60, 1).Should(BeNil())

				endpoints, err := virtClient.CoreV1().Endpoints(tests.NamespaceTestDefault).Get(serviceName, k8smetav1.GetOptions{})
				Expect(err).ToNot(HaveOccurred())

				Expect(len(endpoints.Subsets)).To(Equal(1))
				endpoint := endpoints.Subsets[0]
				Expect(len(endpoint.Ports)).To(Equal(4))
				Expect(endpoint.Ports).To(ContainElement(k8sv1.EndpointPort{Name: "port-1", Port: 80, Protocol: "TCP"}))
				Expect(endpoint.Ports).To(ContainElement(k8sv1.EndpointPort{Name: "port-2", Port: 1500, Protocol: "TCP"}))
				Expect(endpoint.Ports).To(ContainElement(k8sv1.EndpointPort{Name: "port-3", Port: 82, Protocol: "UDP"}))
				Expect(endpoint.Ports).To(ContainElement(k8sv1.EndpointPort{Name: "port-4", Port: 1500, Protocol: "UDP"}))
			},
				table.Entry("over default IPv4 IP family", k8sv1.IPv4Protocol),
				table.Entry("over IPv6 IP family", k8sv1.IPv6Protocol),
			)
		})

		Context("Expose NodePort service", func() {
			const servicePort = "27017"
			const serviceName = "node-port-vmi"

			var vmiExposeArgs []string

			var jobsCleanupFuncs []func() error
			var serviceCleanupFunc func() error

			BeforeEach(func() {
				vmiExposeArgs = []string{
					expose.COMMAND_EXPOSE,
					"virtualmachineinstance", "--namespace", tcpVM.GetNamespace(), tcpVM.GetName(),
					"--port", servicePort, "--name", serviceName, "--target-port", strconv.Itoa(testPort),
					"--type", "NodePort",
				}
			})

			BeforeEach(func() {
				jobsCleanupFuncs = nil
			})

			BeforeEach(func() {
				serviceCleanupFunc = nil
			})

			AfterEach(func() {
				Expect(jobsCleanupFuncs).NotTo(BeNil(), "a successful test must have stored a way to delete the batchv1.Job entity")
				Expect(cleanupJobs(jobsCleanupFuncs)).To(BeEmpty(), "should be able to delete the multiple k8sv1.`Job` entities")
			})

			AfterEach(func() {
				Expect(serviceCleanupFunc).NotTo(BeNil(), "a successful test must have stored a way to delete the k8sv1.Service entity")
				Expect(serviceCleanupFunc()).To(Succeed(), "should be able to delete the k8sv1.Service entity")
			})

			table.DescribeTable("[test_id:1534[label:masquerade_binding_connectivity]Should expose a NodePort service on a VMI and connect to it", func(ipFamily k8sv1.IPFamily) {
				if ipFamily == k8sv1.IPv6Protocol {
					libnet.SkipWhenNotDualStackCluster(virtClient)
					vmiExposeArgs = append(vmiExposeArgs, "--ip-family", "ipv6")
				}

				By("Exposing the service via virtctl command")
				virtctl := tests.NewRepeatableVirtctlCommand(expose.COMMAND_EXPOSE, "virtualmachineinstance", "--namespace",
					tcpVM.Namespace, tcpVM.Name, "--port", servicePort, "--name", serviceName, "--target-port", strconv.Itoa(testPort),
					"--type", "NodePort")
				err := virtctl()
				Expect(err).ToNot(HaveOccurred())

				By("Getting back the service")
				svc, err := virtClient.CoreV1().Services(tcpVM.Namespace).Get(serviceName, k8smetav1.GetOptions{})
				Expect(err).ToNot(HaveOccurred())
				serviceCleanupFunc = cleanupService(svc.GetName(), svc.GetNamespace())
				nodePort := svc.Spec.Ports[0].NodePort
				Expect(nodePort).To(BeNumerically(">", 0))

				By("Getting the node IP from all nodes")
				nodes, err := virtClient.CoreV1().Nodes().List(k8smetav1.ListOptions{})
				Expect(err).ToNot(HaveOccurred())
				Expect(nodes.Items).ToNot(BeEmpty())
				for _, node := range nodes.Items {
					Expect(node.Status.Addresses).ToNot(BeEmpty())
					nodeIP := node.Status.Addresses[0].Address

					By("Starting a job which tries to reach the VMI via NodePort")
					job := runHelloWorldJob(nodeIP, strconv.Itoa(int(nodePort)), tcpVM.Namespace)
					jobsCleanupFuncs = append(jobsCleanupFuncs, cleanupJob(job.GetName(), job.GetNamespace()))

					By("Waiting for the job to report a successful connection attempt")
					Expect(tests.WaitForJobToSucceed(job, 420*time.Second)).To(Succeed())
				}
			},
				table.Entry("over default IPv4 IP family", k8sv1.IPv4Protocol),
				table.Entry("over IPv6 IP family", k8sv1.IPv6Protocol),
			)
		})
	})

	Context("Expose UDP service on a VMI", func() {
		var udpVM *v1.VirtualMachineInstance
		tests.BeforeAll(func() {
			tests.BeforeTestCleanup()
			udpVM = newLabeledVMI("udp-vm", virtClient, true)
			tests.GenerateHelloWorldServer(udpVM, testPort, "udp")
		})

		Context("Expose ClusterIP UDP service", func() {
			const servicePort = "28017"
			const serviceName = "cluster-ip-udp-vmi"

			var vmiExposeArgs []string

			var jobCleanupFunc func() error
			var serviceCleanupFunc func() error

			BeforeEach(func() {
				vmiExposeArgs = []string{
					expose.COMMAND_EXPOSE,
					"virtualmachineinstance", "--namespace", udpVM.GetNamespace(), udpVM.GetName(),
					"--port", servicePort, "--name", serviceName, "--target-port", strconv.Itoa(testPort),
					"--protocol", "UDP",
				}
			})

			BeforeEach(func() {
				jobCleanupFunc = nil
			})

			BeforeEach(func() {
				serviceCleanupFunc = nil
			})

			AfterEach(func() {
				Expect(jobCleanupFunc).NotTo(BeNil(), "a successful test must have stored a way to delete the batchv1.Job entity")
				Expect(jobCleanupFunc()).To(Succeed(), "should be able to delete the batchv1.Job entity")
			})

			AfterEach(func() {
				Expect(serviceCleanupFunc).NotTo(BeNil(), "a successful test must have stored a way to delete the k8sv1.Service entity")
				Expect(serviceCleanupFunc()).To(Succeed(), "should be able to delete the k8sv1.Service entity")
			})

			table.DescribeTable("[test_id:1535][label:masquerade_binding_connectivity]Should expose a ClusterIP service on a VMI and connect to it", func(ipFamily k8sv1.IPFamily) {
				if ipFamily == k8sv1.IPv6Protocol {
					vmiExposeArgs = append(vmiExposeArgs, "--ip-family", "ipv6")
				}
				By("Exposing the service via virtctl command")
				virtctl := tests.NewRepeatableVirtctlCommand(vmiExposeArgs...)
				Expect(virtctl()).To(Succeed(), "should succeed exposing a service via `virtctl expose ...`")

				By("Getting back the cluster IP given for the service")
				svc, err := virtClient.CoreV1().Services(udpVM.Namespace).Get(serviceName, k8smetav1.GetOptions{})
				Expect(err).ToNot(HaveOccurred())
				serviceCleanupFunc = cleanupService(svc.GetName(), svc.GetNamespace())
				serviceIP := svc.Spec.ClusterIP

				By("Starting a job which tries to reach the VMI via ClusterIP")
				job := runHelloWorldJobUDP(serviceIP, servicePort, udpVM.Namespace)
				jobCleanupFunc = cleanupJob(job.GetName(), job.GetNamespace())

				By("Waiting for the job to report a successful connection attempt")
				Expect(tests.WaitForJobToSucceed(job, 420*time.Second)).To(Succeed())
			},
				table.Entry("over default IPv4 IP family", k8sv1.IPv4Protocol),
				table.Entry("over IPv6 IP family", k8sv1.IPv6Protocol),
			)
		})

		Context("Expose NodePort UDP service", func() {
			const servicePort = "29017"
			const serviceName = "node-port-udp-vmi"

			var vmiExposeArgs []string

			var jobsCleanupFuncs []func() error
			var serviceCleanupFunc func() error

			BeforeEach(func() {
				vmiExposeArgs = []string{
					expose.COMMAND_EXPOSE,
					"virtualmachineinstance", "--namespace", udpVM.GetNamespace(), udpVM.GetName(),
					"--port", servicePort, "--name", serviceName, "--target-port", strconv.Itoa(testPort),
					"--type", "NodePort", "--protocol", "UDP",
				}
			})

			BeforeEach(func() {
				jobsCleanupFuncs = nil
			})

			BeforeEach(func() {
				serviceCleanupFunc = nil
			})

			AfterEach(func() {
				Expect(jobsCleanupFuncs).NotTo(BeNil(), "a successful test must have stored a way to delete the batchv1.Job entity")
				Expect(cleanupJobs(jobsCleanupFuncs)).To(BeEmpty(), "should be able to delete the multiple k8sv1.`Job` entities")
			})

			AfterEach(func() {
				Expect(serviceCleanupFunc).NotTo(BeNil(), "a successful test must have stored a way to delete the k8sv1.Service entity")
				Expect(serviceCleanupFunc()).To(Succeed(), "should be able to delete the k8sv1.Service entity")
			})

			table.DescribeTable("[test_id:1536][label:masquerade_binding_connectivity]Should expose a NodePort service on a VMI and connect to it", func(ipFamily k8sv1.IPFamily) {
				if ipFamily == k8sv1.IPv6Protocol {
					libnet.SkipWhenNotDualStackCluster(virtClient)
					vmiExposeArgs = append(vmiExposeArgs, "--ip-family", "ipv6")
				}
				By("Exposing the service via virtctl command")
				virtctl := tests.NewRepeatableVirtctlCommand(vmiExposeArgs...)
				Expect(virtctl()).ToNot(HaveOccurred())

				By("Getting back the cluster IP given for the service")
				svc, err := virtClient.CoreV1().Services(udpVM.Namespace).Get(serviceName, k8smetav1.GetOptions{})
				Expect(err).ToNot(HaveOccurred())
				serviceCleanupFunc = cleanupService(svc.GetName(), svc.GetNamespace())
				serviceIP := svc.Spec.ClusterIP
				nodePort := svc.Spec.Ports[0].NodePort
				Expect(nodePort).To(BeNumerically(">", 0))

				By("Starting a job which tries to reach the VMI via ClusterIP")
				job := runHelloWorldJobUDP(serviceIP, servicePort, udpVM.Namespace)

				By("Waiting for the job to report a successful connection attempt")
				Expect(tests.WaitForJobToSucceed(job, 120*time.Second)).To(Succeed())

				By("Getting the node IP from all nodes")
				nodes, err := virtClient.CoreV1().Nodes().List(k8smetav1.ListOptions{})
				Expect(err).ToNot(HaveOccurred())
				Expect(nodes.Items).ToNot(BeEmpty())
				for _, node := range nodes.Items {
					Expect(node.Status.Addresses).ToNot(BeEmpty())
					nodeIP := node.Status.Addresses[0].Address

					if ipFamily == k8sv1.IPv6Protocol {
						hostname := getNodeHostname(node.Status.Addresses)
						Expect(hostname).NotTo(BeNil(), "must have been able to retrieve the node hostname")
						pod := tests.GetPodByVirtualMachineInstance(udpVM, udpVM.GetNamespace())

						var err error
						nodeIP, err = resolveNodeIp(virtClient, pod, *hostname, ipFamily)
						Expect(err).NotTo(HaveOccurred(), "must have been able to resolve an IP address from the node name")
						Expect(nodeIP).NotTo(BeEmpty(), "must have been able to resolve the IPv6 address of the node")
					}

					By("Starting a job which tries to reach the VMI via NodePort")
					job := runHelloWorldJobUDP(nodeIP, strconv.Itoa(int(nodePort)), udpVM.Namespace)
					jobsCleanupFuncs = append(jobsCleanupFuncs, cleanupJob(job.GetName(), job.GetNamespace()))

					By("Waiting for the job to report a successful connection attempt")
					Expect(tests.WaitForJobToSucceed(job, 420*time.Second)).To(Succeed())
				}
			},
				table.Entry("over default IPv4 IP family", k8sv1.IPv4Protocol),
				table.Entry("over IPv6 IP family", k8sv1.IPv6Protocol),
			)
		})
	})

	Context("Expose service on a VMI replica set", func() {
		var vmrs *v1.VirtualMachineInstanceReplicaSet
		tests.BeforeAll(func() {
			tests.BeforeTestCleanup()
			By("Creating a VMRS object with 2 replicas")
			const numberOfVMs = 2
			template := newLabeledVMI("vmirs", virtClient, false)
			vmrs = tests.NewRandomReplicaSetFromVMI(template, int32(numberOfVMs))
			vmrs.Labels = map[string]string{"expose": "vmirs"}

			By("Start the replica set")
			vmrs, err = virtClient.ReplicaSet(tests.NamespaceTestDefault).Create(vmrs)
			Expect(err).ToNot(HaveOccurred())

			By("Checking the number of ready replicas")
			Eventually(func() int {
				rs, err := virtClient.ReplicaSet(tests.NamespaceTestDefault).Get(vmrs.ObjectMeta.Name, k8smetav1.GetOptions{})
				Expect(err).ToNot(HaveOccurred())
				return int(rs.Status.ReadyReplicas)
			}, 120*time.Second, 1*time.Second).Should(Equal(numberOfVMs))

			By("Add an 'hello world' server on each VMI in the replica set")
			// TODO: add label to list options
			// check size of list
			// remove check for owner
			vms, err := virtClient.VirtualMachineInstance(vmrs.ObjectMeta.Namespace).List(&k8smetav1.ListOptions{})
			Expect(err).ToNot(HaveOccurred())
			for _, vm := range vms.Items {
				if vm.OwnerReferences != nil {
					tests.GenerateHelloWorldServer(&vm, testPort, "tcp")
				}
			}
		})

		Context("Expose ClusterIP service", func() {
			const servicePort = "27017"
			const serviceName = "cluster-ip-vmirs"

			var vmirsExposeArgs []string

			var jobCleanupFunc func() error
			var serviceCleanupFunc func() error

			BeforeEach(func() {
				vmirsExposeArgs = []string{
					expose.COMMAND_EXPOSE,
					"vmirs", "--namespace", vmrs.GetNamespace(), vmrs.GetName(),
					"--port", servicePort, "--name", serviceName, "--target-port", strconv.Itoa(testPort),
				}
			})

			BeforeEach(func() {
				jobCleanupFunc = nil
			})

			BeforeEach(func() {
				serviceCleanupFunc = nil
			})

			AfterEach(func() {
				Expect(jobCleanupFunc).NotTo(BeNil(), "a successful test must have stored a way to delete the batchv1.Job entity")
				Expect(jobCleanupFunc()).To(Succeed(), "should be able to delete the batchv1.Job entity")
			})

			AfterEach(func() {
				Expect(serviceCleanupFunc).NotTo(BeNil(), "a successful test must have stored a way to delete the k8sv1.Service entity")
				Expect(serviceCleanupFunc()).To(Succeed(), "should be able to delete the k8sv1.Service entity")
			})

			table.DescribeTable("[test_id:1537][label:masquerade_binding_connectivity]Should create a ClusterIP service on VMRS and connect to it", func(ipFamily k8sv1.IPFamily) {
				if ipFamily == k8sv1.IPv6Protocol {
					vmirsExposeArgs = append(vmirsExposeArgs, "--ip-family", "ipv6")
				}
				By("Expose a service on the VMRS using virtctl")
				virtctl := tests.NewRepeatableVirtctlCommand(vmirsExposeArgs...)
				Expect(virtctl()).To(Succeed(), "should succeed exposing a service via `virtctl expose ...`")

				By("Getting back the cluster IP given for the service")
				svc, err := virtClient.CoreV1().Services(vmrs.Namespace).Get(serviceName, k8smetav1.GetOptions{})
				Expect(err).ToNot(HaveOccurred())
				serviceCleanupFunc = cleanupService(svc.GetName(), svc.GetNamespace())
				serviceIP := svc.Spec.ClusterIP

				By("Starting a job which tries to reach the VMI via ClusterIP")
				job := runHelloWorldJob(serviceIP, servicePort, vmrs.Namespace)
				jobCleanupFunc = cleanupJob(job.GetName(), job.GetNamespace())

				By("Waiting for the job to report a successful connection attempt")
				Expect(tests.WaitForJobToSucceed(job, 420*time.Second)).To(Succeed())
			},
				table.Entry("over default IPv4 IP family", k8sv1.IPv4Protocol),
				table.Entry("over IPv6 IP family", k8sv1.IPv6Protocol),
			)
		})
	})

	Context("Expose a VM as a service.", func() {
		const servicePort = "27017"
		const serviceName = "cluster-ip-vm"
		var vm *v1.VirtualMachine

		var vmExposeArgs []string
		var jobCleanupFuncs []func() error
		var serviceCleanupFunc func() error

		startVMIFromVMTemplate := func(name string, namespace string) *v1.VirtualMachineInstance {
			By("Calling the start command")
			virtctl := tests.NewRepeatableVirtctlCommand("start", "--namespace", namespace, name)
			Expect(virtctl()).To(Succeed(), "should succeed starting a VMI via `virtctl start ...`")

			By("Getting the status of the VMI")
			Eventually(func() bool {
				vm, err = virtClient.VirtualMachine(namespace).Get(name, &k8smetav1.GetOptions{})
				Expect(err).ToNot(HaveOccurred())
				return vm.Status.Ready
			}, 120*time.Second, 1*time.Second).Should(BeTrue())

			By("Getting the running VMI")
			var vmi *v1.VirtualMachineInstance
			Eventually(func() bool {
				vmi, err = virtClient.VirtualMachineInstance(namespace).Get(name, &k8smetav1.GetOptions{})
				Expect(err).ToNot(HaveOccurred())
				return vmi.Status.Phase == v1.Running
			}, 120*time.Second, 1*time.Second).Should(BeTrue())

			return vmi
		}

		createStoppedVM := func(virtClient kubecli.KubevirtClient, namespace string) error {
			By("Creating an VM object")
			template := newLabeledVMI("vm", virtClient, false)
			vm = tests.NewRandomVirtualMachine(template, false)

			By("Creating the VM")
			vm, err = virtClient.VirtualMachine(namespace).Create(vm)
			return err
		}

		Context("Expose a VM as a ClusterIP service.", func() {
			tests.BeforeAll(func() {
				tests.BeforeTestCleanup()
			})

			BeforeEach(func() {
				Expect(createStoppedVM(virtClient, tests.NamespaceTestDefault)).To(Succeed(), "should have a stopped VM.")
			})

			BeforeEach(func() {
				vmExposeArgs = []string{
					expose.COMMAND_EXPOSE,
					"virtualmachine", "--namespace", vm.GetNamespace(), vm.GetName(),
					"--port", servicePort, "--name", serviceName, "--target-port", strconv.Itoa(testPort),
				}
			})

			BeforeEach(func() {
				jobCleanupFuncs = nil
			})

			BeforeEach(func() {
				serviceCleanupFunc = nil
			})

			AfterEach(func() {
				Expect(jobCleanupFuncs).NotTo(BeNil(), "a successful test must have stored a way to delete the batchv1.Job entity")
				Expect(cleanupJobs(jobCleanupFuncs)).To(BeEmpty(), "should be able to delete the multiple k8sv1.`Job` entities")
			})

			AfterEach(func() {
				Expect(serviceCleanupFunc).NotTo(BeNil(), "a successful test must have stored a way to delete the k8sv1.Service entity")
				Expect(serviceCleanupFunc()).To(Succeed(), "should be able to delete the k8sv1.Service entity")
			})

			AfterEach(func() {
				if vm != nil {
					By("Calling the stop command on the running VMI")
					Expect(vm).NotTo(BeNil(), "should have a VM running so we can get rid of the VMI")
					virtctl := tests.NewRepeatableVirtctlCommand("stop", "--namespace", vm.GetNamespace(), vm.GetName())
					Expect(virtctl()).To(Succeed(), "should succeed stopping a VMI via `virtctl stop ...`")
					Eventually(func() bool {
						vm, err := virtClient.VirtualMachine(vm.GetNamespace()).Get(vm.GetName(), &k8smetav1.GetOptions{})
						Expect(err).ToNot(HaveOccurred())
						return vm.Status.Ready
					}, 120*time.Second, 1*time.Second).Should(BeFalse())
				}
			})

			table.DescribeTable("[test_id:1538][label:masquerade_binding_connectivity]Connect to ClusterIP service that was set when VM was offline.", func(ipFamily k8sv1.IPFamily) {
				if ipFamily == k8sv1.IPv6Protocol {
					vmExposeArgs = append(vmExposeArgs, "--ip-family", "ipv6")
				}
				By("Exposing a service to the VM using virtctl")
				virtctl := tests.NewRepeatableVirtctlCommand(vmExposeArgs...)
				Expect(virtctl()).ToNot(HaveOccurred(), "should succeed exposing a service via `virtctl expose ...`")

				By("Calling the start command on the stopped VM")
				vmi := startVMIFromVMTemplate(vm.GetName(), vm.GetNamespace())
				Expect(vmi).NotTo(BeNil(), "should have been able to create a VMI from a VM")
				tests.GenerateHelloWorldServer(vmi, testPort, "tcp")

				// This TC also covers:
				// [test_id:1795] Exposed VM (as a service) can be reconnected multiple times.
				By("Getting back the cluster IP given for the service")
				svc, err := virtClient.CoreV1().Services(vm.Namespace).Get(serviceName, k8smetav1.GetOptions{})
				Expect(err).ToNot(HaveOccurred())
				serviceCleanupFunc = cleanupService(svc.GetName(), svc.GetNamespace())
				serviceIP := svc.Spec.ClusterIP

				By("Starting a job which tries to reach the VMI via ClusterIP")
				job := runHelloWorldJob(serviceIP, servicePort, vm.Namespace)
				jobCleanupFuncs = append(jobCleanupFuncs, cleanupJob(job.GetName(), job.GetNamespace()))

				By("Waiting for the job to report a successful connection attempt")
				Expect(tests.WaitForJobToSucceed(job, 420*time.Second)).To(Succeed())

				By("Starting a job which tries to reach the VMI again via the same ClusterIP, this time over HTTP.")
				job = runHelloWorldJobHttp(serviceIP, servicePort, vm.Namespace)
				jobCleanupFuncs = append(jobCleanupFuncs, cleanupJob(job.GetName(), job.GetNamespace()))

				By("Waiting for the HTTP job to report a successful connection attempt.")
				Expect(tests.WaitForJobToSucceed(job, 120*time.Second)).To(Succeed())
			},
				table.Entry("over default IPv4 IP family", k8sv1.IPv4Protocol),
				table.Entry("over IPv6 IP family", k8sv1.IPv6Protocol),
			)

			table.DescribeTable("[test_id:345][label:masquerade_binding_connectivity]Should verify the exposed service is functional before and after VM restart.", func(ipFamily k8sv1.IPFamily) {
				if ipFamily == k8sv1.IPv6Protocol {
					vmExposeArgs = append(vmExposeArgs, "--ip-family", "ipv6")
				}
				vmObj := vm

				By("Calling the start command on the stopped VM")
				vmi := startVMIFromVMTemplate(vm.GetName(), vm.GetNamespace())
				Expect(vmi).NotTo(BeNil(), "should have been able to create a VMI from a VM")
				tests.GenerateHelloWorldServer(vmi, testPort, "tcp")

				By("Exposing a service to the VM using virtctl")
				virtctl := tests.NewRepeatableVirtctlCommand(vmExposeArgs...)
				Expect(virtctl()).ToNot(HaveOccurred(), "should succeed exposing a service via `virtctl expose ...`")

				By("Getting back the service's allocated cluster IP.")
				svc, err := virtClient.CoreV1().Services(vmObj.Namespace).Get(serviceName, k8smetav1.GetOptions{})
				Expect(err).ToNot(HaveOccurred())
				serviceCleanupFunc = cleanupService(svc.GetName(), svc.GetNamespace())
				serviceIP := svc.Spec.ClusterIP

				By("Starting a job which tries to reach the VMI via ClusterIP.")
				job := runHelloWorldJob(serviceIP, servicePort, vmObj.Namespace)
				jobCleanupFuncs = append(jobCleanupFuncs, cleanupJob(job.GetName(), job.GetNamespace()))

				By("Waiting for the job to report a successful connection attempt.")
				Expect(tests.WaitForJobToSucceed(job, 120*time.Second)).To(Succeed())

				// Retrieve the current VMI UID, to be compared with the new UID after restart.
				vmi, err = virtClient.VirtualMachineInstance(vmObj.Namespace).Get(vmObj.Name, &k8smetav1.GetOptions{})
				Expect(err).ToNot(HaveOccurred())
				vmiUIdBeforeRestart := vmi.GetObjectMeta().GetUID()

				By("Restarting the running VM.")
				virtctl = tests.NewRepeatableVirtctlCommand("restart", "--namespace", vmObj.Namespace, vmObj.Name)
				err = virtctl()
				Expect(err).ToNot(HaveOccurred())

				By("Verifying the VMI is back up AFTER restart (in Running status with new UID).")
				Eventually(func() bool {
					vmi, err = virtClient.VirtualMachineInstance(vmObj.Namespace).Get(vmObj.Name, &k8smetav1.GetOptions{})
					if errors.IsNotFound(err) {
						return false
					}
					Expect(err).ToNot(HaveOccurred())
					vmiUIdAfterRestart := vmi.GetObjectMeta().GetUID()
					newUId := vmiUIdAfterRestart != vmiUIdBeforeRestart
					return vmi.Status.Phase == v1.Running && newUId
				}, 120*time.Second, 1*time.Second).Should(BeTrue())

				By("Creating a TCP server on the VM.")
				tests.GenerateHelloWorldServer(vmi, testPort, "tcp")

				By("Repeating the sequence as prior to restarting the VM: Connect to exposed ClusterIP service.")
				By("Starting a job which tries to reach the VMI via ClusterIP.")
				job = runHelloWorldJob(serviceIP, servicePort, vmObj.Namespace)
				jobCleanupFuncs = append(jobCleanupFuncs, cleanupJob(job.GetName(), job.GetNamespace()))

				By("Waiting for the job to report a successful connection attempt.")
				Expect(tests.WaitForJobToSucceed(job, 120*time.Second)).To(Succeed())
			},
				table.Entry("over default IPv4 IP family", k8sv1.IPv4Protocol),
				table.Entry("over IPv6 IP family", k8sv1.IPv6Protocol),
			)

			table.DescribeTable("[test_id:343][label:masquerade_binding_connectivity]Should Verify an exposed service of a VM is not functional after VM deletion.", func(ipFamily k8sv1.IPFamily) {
				if ipFamily == k8sv1.IPv6Protocol {
					libnet.SkipWhenNotDualStackCluster(virtClient)
					vmExposeArgs = append(vmExposeArgs, "--ip-family", "ipv6")
				}
				By("Calling the start command on the stopped VM")
				vmi := startVMIFromVMTemplate(vm.GetName(), vm.GetNamespace())
				Expect(vmi).NotTo(BeNil(), "should have been able to create a VMI from a VM")
				tests.GenerateHelloWorldServer(vmi, testPort, "tcp")

				By("Exposing a service to the VM using virtctl")
				virtctl := tests.NewRepeatableVirtctlCommand(vmExposeArgs...)
				Expect(virtctl()).ToNot(HaveOccurred(), "should succeed exposing a service via `virtctl expose ...`")

				By("Getting back the cluster IP given for the service")
				svc, err := virtClient.CoreV1().Services(vm.Namespace).Get(serviceName, k8smetav1.GetOptions{})
				Expect(err).ToNot(HaveOccurred())
				serviceCleanupFunc = cleanupService(svc.GetName(), svc.GetNamespace())
				serviceIP := svc.Spec.ClusterIP

				By("Starting a job which tries to reach the VMI via ClusterIP")
				job := runHelloWorldJob(serviceIP, servicePort, vm.Namespace)

				By("Waiting for the job to report a successful connection attempt")
				Expect(tests.WaitForJobToSucceed(job, 120*time.Second)).To(Succeed())
				jobCleanupFuncs = append(jobCleanupFuncs, cleanupJob(job.GetName(), job.GetNamespace()))

				By("Comparing the service's endpoints IP address to the VM pod IP address.")
				// Get the IP address of the VM pod.
				vmi, err = virtClient.VirtualMachineInstance(vm.Namespace).Get(vm.Name, &k8smetav1.GetOptions{})
				Expect(err).ToNot(HaveOccurred())
				vmPod := tests.GetRunningPodByVirtualMachineInstance(vmi, vmi.Namespace)
				vmPodIpAddress := libnet.GetPodIpByFamily(vmPod, ipFamily)

				// Get the IP address of the service's endpoint.
				endpointsName := serviceName
				svcEndpoints, err := virtClient.CoreV1().Endpoints(vm.Namespace).Get(endpointsName, k8smetav1.GetOptions{})
				Expect(err).ToNot(HaveOccurred())

				// There should be one - and only one - subset for this endpoint,
				// pointing to a single pod (the VMI's virt-launcher pod).
				// This subset should hold a single IP address only - the VM's pod address.
				Expect(len(svcEndpoints.Subsets)).To(Equal(1))
				Expect(len(svcEndpoints.Subsets[0].Addresses)).To(Equal(1))
				endptSubsetIpAddress := svcEndpoints.Subsets[0].Addresses[0].IP

				Eventually(endptSubsetIpAddress).Should(BeEquivalentTo(vmPodIpAddress))

				By("Deleting the VM.")
				Expect(virtClient.VirtualMachine(vm.Namespace).Delete(vm.Name, &k8smetav1.DeleteOptions{})).To(Succeed())
				tests.WaitForVirtualMachineToDisappearWithTimeout(vmi, 120)

				By("Verifying the endpoints' single subset, which points to the VM's pod, is deleted once the VM was deleted.")
				svcEndpoints, err = virtClient.CoreV1().Endpoints(vm.Namespace).Get(endpointsName, k8smetav1.GetOptions{})
				Expect(err).ToNot(HaveOccurred())
				Expect(svcEndpoints.Subsets).To(BeNil())

				By("Starting a job which tries to reach the VMI via the ClusterIP service.")
				job = runHelloWorldJob(serviceIP, servicePort, vm.Namespace)
				jobCleanupFuncs = append(jobCleanupFuncs, cleanupJob(job.GetName(), job.GetNamespace()))

				By("Waiting for the job to report a failed connection attempt.")
				Expect(tests.WaitForJobToFail(job, 120*time.Second)).To(Succeed())

				// this way, the test does not attempt to stop the VM in the `AfterEach` section
				vm = nil
			},
				table.Entry("over default IPv4 IP family", k8sv1.IPv4Protocol),
				table.Entry("over IPv6 IP family", k8sv1.IPv6Protocol),
			)
		})
	})
})

func cleanupJobs(jobsCleanupFunctions []func() error) []error {
	var errorBucket []error
	for _, jobCleanupFunc := range jobsCleanupFunctions {
		err := jobCleanupFunc()
		if err != nil {
			errorBucket = append(errorBucket, err)
		}
	}
	return errorBucket
}

func getNodeHostname(nodeAddresses []k8sv1.NodeAddress) *string {
	for _, address := range nodeAddresses {
		if address.Type == k8sv1.NodeHostName {
			return &address.Address
		}
	}
	return nil
}

func resolveNodeIp(virtclient kubecli.KubevirtClient, pod *k8sv1.Pod, hostname string, ipFamily k8sv1.IPFamily) (string, error) {
	output, err := tests.ExecuteCommandOnPod(
		virtclient,
		pod,
		"compute",
		[]string{"getent", "ahosts", hostname})

	if err != nil {
		return "", err
	}
	for _, ipStr := range strings.Split(output, "\n") {
		ip := strings.Split(ipStr, " ")[0]
		if libnet.GetFamily(ip) == ipFamily {
			return ip, nil
		}
	}

	return "", fmt.Errorf("could not resolve an %s address from %s name", ipFamily, hostname)
}
