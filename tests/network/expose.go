package network

import (
	"context"
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
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/rand"
	"k8s.io/utils/net"
	netutils "k8s.io/utils/net"

	v1 "kubevirt.io/client-go/api/v1"
	"kubevirt.io/client-go/kubecli"
	"kubevirt.io/kubevirt/pkg/virtctl/expose"
	"kubevirt.io/kubevirt/tests"
	"kubevirt.io/kubevirt/tests/assert"
	"kubevirt.io/kubevirt/tests/console"
	cd "kubevirt.io/kubevirt/tests/containerdisk"
	"kubevirt.io/kubevirt/tests/libnet"
	"kubevirt.io/kubevirt/tests/libvmi"
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
		tests.WaitUntilVMIReady(vmi, libnet.WithIPv6(console.LoginToCirros))
	}
	return
}

var _ = SIGDescribe("[rfe_id:253][crit:medium][vendor:cnv-qe@redhat.com][level:component]Expose", func() {

	var virtClient kubecli.KubevirtClient
	var err error

	const testPort = 1500

	type ipFamily string
	const (
		ipv4            ipFamily = "ipv4"
		ipv6            ipFamily = "ipv6"
		dualIPv4Primary ipFamily = "ipv4,ipv6"
		dualIPv6Primary ipFamily = "ipv6,ipv4"
	)

	isDualStack := func(ipFamily ipFamily) bool {
		return ipFamily == dualIPv4Primary || ipFamily == dualIPv6Primary
	}

	doesSupportIpv6 := func(ipFamily ipFamily) bool {
		return ipFamily != ipv4
	}

	const xfailError = "Secondary ip on dual stack service is not working. Tracking issue - https://github.com/kubevirt/kubevirt/issues/5477"

	BeforeEach(func() {
		virtClient, err = kubecli.GetKubevirtClient()
		tests.PanicOnError(err)
	})

	runHelloWorldJob := func(host, port, namespace string) *batchv1.Job {
		job, err := virtClient.BatchV1().Jobs(namespace).Create(context.Background(), generateBatchJobSpec(host, port, tests.NewHelloWorldJob), metav1.CreateOptions{})
		ExpectWithOffset(2, err).ToNot(HaveOccurred())
		return job
	}

	runHelloWorldJobUDP := func(host, port, namespace string) *batchv1.Job {
		job, err := virtClient.BatchV1().Jobs(namespace).Create(context.Background(), generateBatchJobSpec(host, port, tests.NewHelloWorldJobUDP), metav1.CreateOptions{})
		ExpectWithOffset(2, err).ToNot(HaveOccurred())
		return job
	}

	runHelloWorldJobHttp := func(host, port, namespace string) *batchv1.Job {
		job, err := virtClient.BatchV1().Jobs(namespace).Create(context.Background(), generateBatchJobSpec(host, port, tests.NewHelloWorldJobHTTP), metav1.CreateOptions{})
		ExpectWithOffset(2, err).ToNot(HaveOccurred())
		return job
	}

	randomizeName := func(currentName string) string {
		return currentName + rand.String(5)
	}

	validateClusterIp := func(clusterIp string, ipFamily ipFamily) error {
		var correctPrimaryFamily bool
		switch ipFamily {
		case ipv4, dualIPv4Primary:
			correctPrimaryFamily = netutils.IsIPv4String(clusterIp)
		case ipv6, dualIPv6Primary:
			correctPrimaryFamily = netutils.IsIPv6String(clusterIp)
		}
		if !correctPrimaryFamily {
			return fmt.Errorf("the ClusterIP %s belongs to the wrong ip family", clusterIp)
		}
		return nil
	}

	skipIfNotSupportedCluster := func(ipFamily ipFamily) {
		if doesSupportIpv6(ipFamily) {
			libnet.SkipWhenNotDualStackCluster(virtClient)
			if isDualStack(ipFamily) {
				tests.SkipIfVersionBelow("Dual stack service requires v1.20 and above", "1.20")
			}
		}
	}

	appendIpFamilyToExposeArgs := func(ipFamily ipFamily, vmiExposeArgs []string) []string {
		if doesSupportIpv6(ipFamily) {
			vmiExposeArgs = append(vmiExposeArgs, "--ip-family", string(ipFamily))
		}
		return vmiExposeArgs
	}

	createAndWaitForJobToSucceed := func(helloWorldJobCreator func(host, port, namespace string) *batchv1.Job, namespace, ip, port, viaMessage string) error {
		By(fmt.Sprintf("Starting a job which tries to reach the VMI via the %s", viaMessage))
		job := helloWorldJobCreator(ip, port, namespace)

		By("Waiting for the job to report a successful connection attempt")
		return tests.WaitForJobToSucceed(job, time.Duration(120)*time.Second)
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
			const serviceNamePrefix = "cluster-ip-vmi"

			var serviceName string
			var vmiExposeArgs []string

			BeforeEach(func() {
				serviceName = randomizeName(serviceNamePrefix)

				vmiExposeArgs = []string{
					expose.COMMAND_EXPOSE,
					"virtualmachineinstance", "--namespace", tcpVM.GetNamespace(), tcpVM.GetName(),
					"--port", servicePort, "--name", serviceName,
					"--target-port", strconv.Itoa(testPort),
				}
			})

			table.DescribeTable("[label:masquerade_binding_connectivity]Should expose a Cluster IP service on a VMI and connect to it", func(ipFamily ipFamily) {
				skipIfNotSupportedCluster(ipFamily)
				vmiExposeArgs = appendIpFamilyToExposeArgs(ipFamily, vmiExposeArgs)

				By("Exposing the service via virtctl command")
				virtctl := tests.NewRepeatableVirtctlCommand(vmiExposeArgs...)
				Expect(virtctl()).To(Succeed(), "should expose a service via `virtctl expose ...`")

				By("Getting back the cluster IP given for the service")
				svc, err := virtClient.CoreV1().Services(tcpVM.Namespace).Get(context.Background(), serviceName, k8smetav1.GetOptions{})
				Expect(err).ToNot(HaveOccurred())
				Expect(validateClusterIp(svc.Spec.ClusterIP, ipFamily)).To(Succeed())

				By("Iterating over the ClusterIPs")
				serviceIPs := svc.Spec.ClusterIPs
				for ipOrderNum, ip := range serviceIPs {
					assert.XFail(xfailError, func() {
						Expect(createAndWaitForJobToSucceed(runHelloWorldJob, tcpVM.Namespace, ip, servicePort, fmt.Sprintf("%dst ClusterIP", ipOrderNum+1))).To(Succeed())
					}, ipOrderNum > 0)
				}
			},
				table.Entry("[test_id:1531] over default IPv4 IP family", ipv4),
				table.Entry("over IPv6 IP family", ipv6),
				table.Entry("over dual stack, primary ipv4", dualIPv4Primary),
				table.Entry("over dual stack, primary ipv6", dualIPv6Primary),
			)
		})

		Context("Expose ClusterIP service with string target-port", func() {
			const servicePort = "27017"
			const serviceNamePrefix = "cluster-ip-target-vmi"

			var serviceName string
			var vmiExposeArgs []string

			BeforeEach(func() {
				serviceName = randomizeName(serviceNamePrefix)

				vmiExposeArgs = []string{
					expose.COMMAND_EXPOSE,
					"virtualmachineinstance", "--namespace", tcpVM.GetNamespace(), tcpVM.GetName(),
					"--port", servicePort, "--name", serviceName, "--target-port", "http",
				}
			})

			table.DescribeTable("Should expose a ClusterIP service and connect to the vm on port 80", func(ipFamily ipFamily) {
				skipIfNotSupportedCluster(ipFamily)
				vmiExposeArgs = appendIpFamilyToExposeArgs(ipFamily, vmiExposeArgs)

				By("Exposing the service via virtctl command")
				virtctl := tests.NewRepeatableVirtctlCommand(vmiExposeArgs...)
				Expect(virtctl()).To(Succeed(), "should expose a service via `virtctl expose ...`")

				By("Waiting for kubernetes to create the relevant endpoint")
				getEndpoint := func() error {
					_, err := virtClient.CoreV1().Endpoints(tests.NamespaceTestDefault).Get(context.Background(), serviceName, k8smetav1.GetOptions{})
					return err
				}
				Eventually(getEndpoint, 60, 1).Should(BeNil())

				endpoints, err := virtClient.CoreV1().Endpoints(tests.NamespaceTestDefault).Get(context.Background(), serviceName, k8smetav1.GetOptions{})
				Expect(err).ToNot(HaveOccurred())

				Expect(len(endpoints.Subsets)).To(Equal(1))
				endpoint := endpoints.Subsets[0]
				Expect(len(endpoint.Ports)).To(Equal(1))
				Expect(endpoint.Ports[0].Port).To(Equal(int32(80)))

				isDualStack := isDualStack(ipFamily)
				numOfIps := 1
				if isDualStack {
					numOfIps = 2
				}
				assert.XFail(xfailError, func() {
					Expect(len(endpoints.Subsets[0].Addresses)).To(Equal(numOfIps))
				}, isDualStack)
			},
				table.Entry("[test_id:1532] over default IPv4 IP family", ipv4),
				table.Entry("over IPv6 IP family", ipv6),
				table.Entry("over dual stack, primary ipv4", dualIPv4Primary),
				table.Entry("over dual stack, primary ipv6", dualIPv6Primary),
			)
		})

		Context("Expose ClusterIP service with ports on the vmi defined", func() {
			const serviceNamePrefix = "cluster-ip-target-multiple-ports-vmi"

			var serviceName string
			var vmiExposeArgs []string

			BeforeEach(func() {
				serviceName = randomizeName(serviceNamePrefix)

				vmiExposeArgs = []string{
					expose.COMMAND_EXPOSE,
					"virtualmachineinstance", "--namespace", tcpVM.GetNamespace(), tcpVM.GetName(),
					"--name", serviceName,
				}
			})

			table.DescribeTable("Should expose a ClusterIP service and connect to all ports defined on the vmi", func(ipFamily ipFamily) {
				skipIfNotSupportedCluster(ipFamily)
				vmiExposeArgs = appendIpFamilyToExposeArgs(ipFamily, vmiExposeArgs)

				By("Exposing the service via virtctl command")
				virtctl := tests.NewRepeatableVirtctlCommand(vmiExposeArgs...)
				Expect(virtctl()).To(Succeed(), "should expose a service via `virtctl expose ...`")

				By("Waiting for kubernetes to create the relevant endpoint")
				getEndpoint := func() error {
					_, err := virtClient.CoreV1().Endpoints(tests.NamespaceTestDefault).Get(context.Background(), serviceName, k8smetav1.GetOptions{})
					return err
				}
				Eventually(getEndpoint, 60, 1).Should(BeNil())

				endpoints, err := virtClient.CoreV1().Endpoints(tests.NamespaceTestDefault).Get(context.Background(), serviceName, k8smetav1.GetOptions{})
				Expect(err).ToNot(HaveOccurred())

				Expect(len(endpoints.Subsets)).To(Equal(1))
				endpoint := endpoints.Subsets[0]
				Expect(len(endpoint.Ports)).To(Equal(4))
				Expect(endpoint.Ports).To(ContainElement(k8sv1.EndpointPort{Name: "port-1", Port: 80, Protocol: "TCP"}))
				Expect(endpoint.Ports).To(ContainElement(k8sv1.EndpointPort{Name: "port-2", Port: 1500, Protocol: "TCP"}))
				Expect(endpoint.Ports).To(ContainElement(k8sv1.EndpointPort{Name: "port-3", Port: 82, Protocol: "UDP"}))
				Expect(endpoint.Ports).To(ContainElement(k8sv1.EndpointPort{Name: "port-4", Port: 1500, Protocol: "UDP"}))
			},
				table.Entry("[test_id:1533] over default IPv4 IP family", ipv4),
				table.Entry("over IPv6 IP family", ipv6),
				table.Entry("over dual stack, primary ipv4", dualIPv4Primary),
				table.Entry("over dual stack, primary ipv6", dualIPv6Primary),
			)
		})

		Context("Expose NodePort service", func() {
			const servicePort = "27017"
			const serviceNamePrefix = "node-port-vmi"

			var serviceName string
			var vmiExposeArgs []string

			BeforeEach(func() {
				serviceName = randomizeName(serviceNamePrefix)
				vmiExposeArgs = []string{
					expose.COMMAND_EXPOSE,
					"virtualmachineinstance", "--namespace", tcpVM.GetNamespace(), tcpVM.GetName(),
					"--port", servicePort, "--name", serviceName, "--target-port", strconv.Itoa(testPort),
					"--type", "NodePort",
				}
			})

			table.DescribeTable("[label:masquerade_binding_connectivity]Should expose a NodePort service on a VMI and connect to it", func(ipFamily ipFamily) {
				skipIfNotSupportedCluster(ipFamily)
				vmiExposeArgs = appendIpFamilyToExposeArgs(ipFamily, vmiExposeArgs)

				By("Exposing the service via virtctl command")
				virtctl := tests.NewRepeatableVirtctlCommand(vmiExposeArgs...)
				Expect(virtctl()).To(Succeed(), "should expose a service via `virtctl expose ...`")

				By("Getting back the service")
				svc, err := virtClient.CoreV1().Services(tcpVM.Namespace).Get(context.Background(), serviceName, k8smetav1.GetOptions{})
				Expect(err).ToNot(HaveOccurred())
				nodePort := svc.Spec.Ports[0].NodePort
				Expect(nodePort).To(BeNumerically(">", 0))

				By("Getting the node IP from all nodes")
				nodes, err := virtClient.CoreV1().Nodes().List(context.Background(), k8smetav1.ListOptions{})
				Expect(err).ToNot(HaveOccurred())
				Expect(nodes.Items).ToNot(BeEmpty())
				for _, node := range nodes.Items {
					Expect(node.Status.Addresses).ToNot(BeEmpty())
					nodeIP := node.Status.Addresses[0].Address
					var ipv6NodeIP string

					if ipFamily != ipv6 {
						By("Connecting to IPv4 node IP")
						assert.XFail(xfailError, func() {
							Expect(createAndWaitForJobToSucceed(runHelloWorldJob, tcpVM.Namespace, nodeIP, strconv.Itoa(int(nodePort)), fmt.Sprintf("NodePort using %s node ip", ipFamily))).To(Succeed())
						}, ipFamily == dualIPv6Primary)
					}
					if doesSupportIpv6(ipFamily) {
						ipv6NodeIP, err = resolveNodeIPAddrByFamily(
							virtClient,
							libvmi.GetPodByVirtualMachineInstance(tcpVM, tcpVM.GetNamespace()),
							node,
							k8sv1.IPv6Protocol)
						Expect(err).NotTo(HaveOccurred(), "must have been able to resolve an IP address from the node name")
						Expect(ipv6NodeIP).NotTo(BeEmpty(), "must have been able to resolve the IPv6 address of the node")

						By("Connecting to IPv6 node IP")
						assert.XFail(xfailError, func() {
							Expect(createAndWaitForJobToSucceed(runHelloWorldJob, tcpVM.Namespace, ipv6NodeIP, strconv.Itoa(int(nodePort)), fmt.Sprintf("NodePort using %s node ip", ipFamily))).To(Succeed())
						}, ipFamily == dualIPv4Primary)
					}
				}
			},
				table.Entry("[test_id:1534] over default IPv4 IP family", ipv4),
				table.Entry("over IPv6 IP family", ipv6),
				table.Entry("over dual stack, primary ipv4", dualIPv4Primary),
				table.Entry("over dual stack, primary ipv6", dualIPv6Primary),
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
			const serviceNamePrefix = "cluster-ip-udp-vmi"

			var serviceName string
			var vmiExposeArgs []string

			BeforeEach(func() {
				serviceName = randomizeName(serviceNamePrefix)

				vmiExposeArgs = []string{
					expose.COMMAND_EXPOSE,
					"virtualmachineinstance", "--namespace", udpVM.GetNamespace(), udpVM.GetName(),
					"--port", servicePort, "--name", serviceName, "--target-port", strconv.Itoa(testPort),
					"--protocol", "UDP",
				}
			})

			table.DescribeTable("[label:masquerade_binding_connectivity]Should expose a ClusterIP service on a VMI and connect to it", func(ipFamily ipFamily) {
				skipIfNotSupportedCluster(ipFamily)
				vmiExposeArgs = appendIpFamilyToExposeArgs(ipFamily, vmiExposeArgs)

				By("Exposing the service via virtctl command")
				virtctl := tests.NewRepeatableVirtctlCommand(vmiExposeArgs...)
				Expect(virtctl()).To(Succeed(), "should succeed exposing a service via `virtctl expose ...`")

				By("Getting back the cluster IP given for the service")
				svc, err := virtClient.CoreV1().Services(udpVM.Namespace).Get(context.Background(), serviceName, k8smetav1.GetOptions{})
				Expect(err).ToNot(HaveOccurred())

				By("Validating the ClusterIP")
				Expect(validateClusterIp(svc.Spec.ClusterIP, ipFamily)).To(Succeed())
				serviceIPs := svc.Spec.ClusterIPs

				By("Iterating over the ClusterIPs")
				for ipOrderNum, ip := range serviceIPs {
					assert.XFail(xfailError, func() {
						Expect(createAndWaitForJobToSucceed(runHelloWorldJobUDP, udpVM.Namespace, ip, servicePort, fmt.Sprintf("%dst ClusterIP", ipOrderNum+1))).To(Succeed())
					}, ipOrderNum > 0)
				}
			},
				table.Entry("[test_id:1535] over default IPv4 IP family", ipv4),
				table.Entry("over IPv6 IP family", ipv6),
				table.Entry("over dual stack, primary ipv4", dualIPv4Primary),
				table.Entry("over dual stack, primary ipv6", dualIPv6Primary),
			)
		})

		Context("Expose NodePort UDP service", func() {
			const servicePort = "29017"
			const serviceNamePrefix = "node-port-udp-vmi"

			var serviceName string
			var vmiExposeArgs []string

			BeforeEach(func() {
				serviceName = randomizeName(serviceNamePrefix)

				vmiExposeArgs = []string{
					expose.COMMAND_EXPOSE,
					"virtualmachineinstance", "--namespace", udpVM.GetNamespace(), udpVM.GetName(),
					"--port", servicePort, "--name", serviceName, "--target-port", strconv.Itoa(testPort),
					"--type", "NodePort", "--protocol", "UDP",
				}
			})

			table.DescribeTable("[label:masquerade_binding_connectivity]Should expose a NodePort service on a VMI and connect to it", func(ipFamily ipFamily) {
				skipIfNotSupportedCluster(ipFamily)
				vmiExposeArgs = appendIpFamilyToExposeArgs(ipFamily, vmiExposeArgs)

				By("Exposing the service via virtctl command")
				virtctl := tests.NewRepeatableVirtctlCommand(vmiExposeArgs...)
				Expect(virtctl()).ToNot(HaveOccurred())

				By("Getting back the cluster IP given for the service")
				svc, err := virtClient.CoreV1().Services(udpVM.Namespace).Get(context.Background(), serviceName, k8smetav1.GetOptions{})
				Expect(err).ToNot(HaveOccurred())
				Expect(validateClusterIp(svc.Spec.ClusterIP, ipFamily)).To(Succeed())
				nodePort := svc.Spec.Ports[0].NodePort
				Expect(nodePort).To(BeNumerically(">", 0))

				By("Iterating over the ClusterIPs")
				serviceIPs := svc.Spec.ClusterIPs
				for ipOrderNum, ip := range serviceIPs {
					assert.XFail(xfailError, func() {
						Expect(createAndWaitForJobToSucceed(runHelloWorldJobUDP, udpVM.Namespace, ip, servicePort, fmt.Sprintf("%d ClusterIP", ipOrderNum+1))).To(Succeed())
					}, ipOrderNum > 0)
				}

				By("Getting the node IP from all nodes")
				nodes, err := virtClient.CoreV1().Nodes().List(context.Background(), k8smetav1.ListOptions{})
				Expect(err).ToNot(HaveOccurred())
				Expect(nodes.Items).ToNot(BeEmpty())
				for _, node := range nodes.Items {
					Expect(node.Status.Addresses).ToNot(BeEmpty())
					nodeIP := node.Status.Addresses[0].Address

					var ipv6NodeIP string
					if doesSupportIpv6(ipFamily) {
						ipv6NodeIP, err = resolveNodeIPAddrByFamily(
							virtClient,
							libvmi.GetPodByVirtualMachineInstance(udpVM, udpVM.GetNamespace()),
							node,
							k8sv1.IPv6Protocol)
						Expect(err).NotTo(HaveOccurred(), "must have been able to resolve an IP address from the node name")
						Expect(ipv6NodeIP).NotTo(BeEmpty(), "must have been able to resolve the IPv6 address of the node")
					}

					if ipFamily != ipv6 {
						By("Connecting to IPv4 node IP")
						assert.XFail(xfailError, func() {
							Expect(createAndWaitForJobToSucceed(runHelloWorldJobUDP, udpVM.Namespace, nodeIP, strconv.Itoa(int(nodePort)), "NodePort ipv4 address")).To(Succeed())
						}, ipFamily == dualIPv6Primary)
					}
					if doesSupportIpv6(ipFamily) {
						By("Connecting to IPv6 node IP")
						assert.XFail(xfailError, func() {
							Expect(createAndWaitForJobToSucceed(runHelloWorldJobUDP, udpVM.Namespace, ipv6NodeIP, strconv.Itoa(int(nodePort)), "NodePort ipv6 address")).To(Succeed())
						}, ipFamily == dualIPv4Primary)
					}
				}
			},
				table.Entry("[test_id:1536] over default IPv4 IP family", ipv4),
				table.Entry("over IPv6 IP family", ipv6),
				table.Entry("over dual stack, primary ipv4", dualIPv4Primary),
				table.Entry("over dual stack, primary ipv6", dualIPv6Primary),
			)
		})
	})

	Context("Expose service on a VMI replica set", func() {
		const numberOfVMs = 2

		var vmrs *v1.VirtualMachineInstanceReplicaSet
		tests.BeforeAll(func() {
			tests.BeforeTestCleanup()
			By("Creating a VMRS object with 2 replicas")
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
			const serviceNamePrefix = "cluster-ip-vmirs"

			var serviceName string
			var vmirsExposeArgs []string

			BeforeEach(func() {
				serviceName = randomizeName(serviceNamePrefix)

				vmirsExposeArgs = []string{
					expose.COMMAND_EXPOSE,
					"vmirs", "--namespace", vmrs.GetNamespace(), vmrs.GetName(),
					"--port", servicePort, "--name", serviceName, "--target-port", strconv.Itoa(testPort),
				}
			})

			table.DescribeTable("[label:masquerade_binding_connectivity]Should create a ClusterIP service on VMRS and connect to it", func(ipFamily ipFamily) {
				skipIfNotSupportedCluster(ipFamily)
				vmirsExposeArgs = appendIpFamilyToExposeArgs(ipFamily, vmirsExposeArgs)

				By("Expose a service on the VMRS using virtctl")
				virtctl := tests.NewRepeatableVirtctlCommand(vmirsExposeArgs...)

				Expect(virtctl()).To(Succeed(), "should succeed exposing a service via `virtctl expose ...`")

				By("Getting back the cluster IP given for the service")
				svc, err := virtClient.CoreV1().Services(vmrs.Namespace).Get(context.Background(), serviceName, k8smetav1.GetOptions{})
				Expect(err).ToNot(HaveOccurred())
				Expect(validateClusterIp(svc.Spec.ClusterIP, ipFamily)).To(Succeed())

				By("Iterating over the ClusterIPs")
				serviceIPs := svc.Spec.ClusterIPs
				for ipOrderNum, ip := range serviceIPs {
					assert.XFail(xfailError, func() {
						Expect(createAndWaitForJobToSucceed(runHelloWorldJob, vmrs.Namespace, ip, servicePort, fmt.Sprintf("%d ClusterIP", ipOrderNum+1))).To(Succeed())
					}, ipOrderNum > 0)
				}
			},
				table.Entry("[test_id:1537] over default IPv4 IP family", ipv4),
				table.Entry("over IPv6 IP family", ipv6),
				table.Entry("over dual stack, primary ipv4", dualIPv4Primary),
				table.Entry("over dual stack, primary ipv6", dualIPv6Primary),
			)
		})
	})

	Context("Expose a VM as a service.", func() {
		const servicePort = "27017"
		const serviceNamePrefix = "cluster-ip-vm"
		var vm *v1.VirtualMachine

		var vmExposeArgs []string

		startVMIFromVMTemplate := func(virtClient kubecli.KubevirtClient, name string, namespace string) *v1.VirtualMachineInstance {
			By("Calling the start command")
			virtctl := tests.NewRepeatableVirtctlCommand("start", "--namespace", namespace, name)
			Expect(virtctl()).To(Succeed(), "should succeed starting a VMI via `virtctl start ...`")

			By("Getting the status of the VMI")
			Eventually(func() bool {
				vm, err := virtClient.VirtualMachine(namespace).Get(name, &k8smetav1.GetOptions{})
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

		createStoppedVM := func(virtClient kubecli.KubevirtClient, namespace string) (*v1.VirtualMachine, error) {
			By("Creating an VM object")
			template := newLabeledVMI("vm", virtClient, false)
			vm := tests.NewRandomVirtualMachine(template, false)

			By("Creating the VM")
			vm, err = virtClient.VirtualMachine(namespace).Create(vm)
			return vm, err
		}

		startVMWithServer := func(virtClient kubecli.KubevirtClient, protocol string, port int) *v1.VirtualMachineInstance {
			By("Calling the start command on the stopped VM")
			vmi := startVMIFromVMTemplate(virtClient, vm.GetName(), vm.GetNamespace())
			if vmi == nil {
				return nil
			}
			tests.GenerateHelloWorldServer(vmi, port, protocol)
			return vmi
		}

		Context("Expose a VM as a ClusterIP service.", func() {
			var serviceName string

			BeforeEach(tests.BeforeTestCleanup)

			BeforeEach(func() {
				vm, err = createStoppedVM(virtClient, tests.NamespaceTestDefault)
				Expect(err).NotTo(HaveOccurred(), "should create a stopped VM.")
			})

			BeforeEach(func() {
				serviceName = randomizeName(serviceNamePrefix)

				vmExposeArgs = []string{
					expose.COMMAND_EXPOSE,
					"virtualmachine", "--namespace", vm.GetNamespace(), vm.GetName(),
					"--port", servicePort, "--name", serviceName, "--target-port", strconv.Itoa(testPort),
				}
			})

			table.DescribeTable("[label:masquerade_binding_connectivity]Connect to ClusterIP service that was set when VM was offline.", func(ipFamily ipFamily) {
				skipIfNotSupportedCluster(ipFamily)
				vmExposeArgs = appendIpFamilyToExposeArgs(ipFamily, vmExposeArgs)

				By("Exposing a service to the VM using virtctl")
				virtctl := tests.NewRepeatableVirtctlCommand(vmExposeArgs...)
				Expect(virtctl()).ToNot(HaveOccurred(), "should succeed exposing a service via `virtctl expose ...`")

				vmi := startVMWithServer(virtClient, "tcp", testPort)
				Expect(vmi).NotTo(BeNil(), "should have been able to start the VM")

				// This TC also covers:
				// [test_id:1795] Exposed VM (as a service) can be reconnected multiple times.
				By("Getting back the cluster IP given for the service")
				svc, err := virtClient.CoreV1().Services(vm.Namespace).Get(context.Background(), serviceName, k8smetav1.GetOptions{})
				Expect(err).ToNot(HaveOccurred())
				Expect(validateClusterIp(svc.Spec.ClusterIP, ipFamily)).To(Succeed())

				By("Iterating over the ClusterIPs")
				serviceIPs := svc.Spec.ClusterIPs
				for ipOrderNum, ip := range serviceIPs {
					assert.XFail(xfailError, func() {
						Expect(createAndWaitForJobToSucceed(runHelloWorldJob, vm.Namespace, ip, servicePort, fmt.Sprintf("%d ClusterIP", ipOrderNum+1))).To(Succeed())
						Expect(createAndWaitForJobToSucceed(runHelloWorldJobHttp, vm.Namespace, ip, servicePort, fmt.Sprintf("same %dst ClusterIP, this time over HTTP", ipOrderNum+1))).To(Succeed())
					}, ipOrderNum > 0)
				}
			},
				table.Entry("[test_id:1538] over default IPv4 IP family", ipv4),
				table.Entry("over IPv6 IP family", ipv6),
				table.Entry("over dual stack, primary ipv4", dualIPv4Primary),
				table.Entry("over dual stack, primary ipv6", dualIPv6Primary),
			)

			table.DescribeTable("[label:masquerade_binding_connectivity]Should verify the exposed service is functional before and after VM restart.", func(ipFamily ipFamily) {
				skipIfNotSupportedCluster(ipFamily)
				vmExposeArgs = appendIpFamilyToExposeArgs(ipFamily, vmExposeArgs)

				vmObj := vm

				By("Exposing a service to the VM using virtctl")
				virtctl := tests.NewRepeatableVirtctlCommand(vmExposeArgs...)
				Expect(virtctl()).ToNot(HaveOccurred(), "should succeed exposing a service via `virtctl expose ...`")

				vmi := startVMWithServer(virtClient, "tcp", testPort)
				Expect(vmi).NotTo(BeNil(), "should have been able to start the VM")

				By("Getting back the service's allocated cluster IP.")
				svc, err := virtClient.CoreV1().Services(vmObj.Namespace).Get(context.Background(), serviceName, k8smetav1.GetOptions{})
				Expect(err).ToNot(HaveOccurred())
				Expect(validateClusterIp(svc.Spec.ClusterIP, ipFamily)).To(Succeed())

				By("Iterating over the ClusterIPs")
				serviceIPs := svc.Spec.ClusterIPs
				for ipOrderNum, ip := range serviceIPs {
					assert.XFail(xfailError, func() {
						Expect(createAndWaitForJobToSucceed(runHelloWorldJob, vmObj.Namespace, ip, servicePort, fmt.Sprintf("%d ClusterIP", ipOrderNum+1))).To(Succeed())
					}, ipOrderNum > 0)
				}

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
				for ipOrderNum, ip := range serviceIPs {
					assert.XFail(xfailError, func() {
						Expect(createAndWaitForJobToSucceed(runHelloWorldJob, vmObj.Namespace, ip, servicePort, fmt.Sprintf("%d ClusterIP", ipOrderNum+1))).To(Succeed())
					}, ipOrderNum > 0)
				}
			},
				table.Entry("[test_id:345] over default IPv4 IP family", ipv4),
				table.Entry("over IPv6 IP family", ipv6),
				table.Entry("over dual stack, primary ipv4", dualIPv4Primary),
				table.Entry("over dual stack, primary ipv6", dualIPv6Primary),
			)

			table.DescribeTable("[label:masquerade_binding_connectivity]Should Verify an exposed service of a VM is not functional after VM deletion.", func(svcIpFamily ipFamily) {
				skipIfNotSupportedCluster(svcIpFamily)
				vmExposeArgs = appendIpFamilyToExposeArgs(svcIpFamily, vmExposeArgs)

				getPrimaryAndSecondaryAddr := func(ipv4Addr, ipv6Addr string) (string, string) {
					var primaryAddr string
					var secondaryAddr string
					switch svcIpFamily {
					case ipv4:
						primaryAddr = ipv4Addr
					case ipv6:
						primaryAddr = ipv6Addr
					case dualIPv4Primary:
						primaryAddr = ipv4Addr
						secondaryAddr = ipv6Addr
					case dualIPv6Primary:
						primaryAddr = ipv6Addr
						secondaryAddr = ipv4Addr
					}
					return primaryAddr, secondaryAddr
				}

				By("Exposing a service to the VM using virtctl")
				virtctl := tests.NewRepeatableVirtctlCommand(vmExposeArgs...)
				Expect(virtctl()).ToNot(HaveOccurred(), "should succeed exposing a service via `virtctl expose ...`")

				vmi := startVMWithServer(virtClient, "tcp", testPort)
				Expect(vmi).NotTo(BeNil(), "should have been able to start the VM")

				By("Getting back the cluster IP given for the service")
				svc, err := virtClient.CoreV1().Services(vm.Namespace).Get(context.Background(), serviceName, k8smetav1.GetOptions{})
				Expect(err).ToNot(HaveOccurred())
				serviceIP := svc.Spec.ClusterIP
				Expect(validateClusterIp(svc.Spec.ClusterIP, svcIpFamily)).To(Succeed())

				By("Iterating over the ClusterIPs")
				serviceIPs := svc.Spec.ClusterIPs
				for ipOrderNum, ip := range serviceIPs {
					assert.XFail(xfailError, func() {
						Expect(createAndWaitForJobToSucceed(runHelloWorldJob, vm.Namespace, ip, servicePort, fmt.Sprintf("%d ClusterIP", ipOrderNum+1))).To(Succeed())
					}, ipOrderNum > 0)
				}

				By("Comparing the service's endpoints IP address to the VM pod IP address.")
				// Get the IP address of the VM pod.
				vmi, err = virtClient.VirtualMachineInstance(vm.Namespace).Get(vm.Name, &k8smetav1.GetOptions{})
				Expect(err).ToNot(HaveOccurred())
				vmPod := tests.GetRunningPodByVirtualMachineInstance(vmi, vmi.Namespace)
				vmPodIpv4Address := libnet.GetPodIpByFamily(vmPod, k8sv1.IPv4Protocol)
				vmPodIpv6Address := libnet.GetPodIpByFamily(vmPod, k8sv1.IPv6Protocol)

				primaryVmPodAddr, secondaryVmPodAddr := getPrimaryAndSecondaryAddr(vmPodIpv4Address, vmPodIpv6Address)

				// Get the IP address of the service's endpoint.
				endpointsName := serviceName
				svcEndpoints, err := virtClient.CoreV1().Endpoints(vm.Namespace).Get(context.Background(), endpointsName, k8smetav1.GetOptions{})
				Expect(err).ToNot(HaveOccurred())

				// There should be one - and only one - subset for this endpoint,
				// pointing to a single pod (the VMI's virt-launcher pod).
				// This subset should hold a single IP address only - the VM's pod address.
				Expect(len(svcEndpoints.Subsets)).To(Equal(1))

				numOfIps := 1
				if secondaryVmPodAddr != "" {
					numOfIps = 2
				}
				assert.XFail(xfailError, func() {
					Expect(len(svcEndpoints.Subsets[0].Addresses)).To(Equal(numOfIps))
				}, secondaryVmPodAddr != "")

				endptSubsetIpAddress := svcEndpoints.Subsets[0].Addresses[0].IP
				Eventually(endptSubsetIpAddress).Should(BeEquivalentTo(primaryVmPodAddr))

				if secondaryVmPodAddr != "" {
					assert.XFail(xfailError, func() {
						endptSubsetIpAddress := svcEndpoints.Subsets[0].Addresses[1].IP
						Eventually(endptSubsetIpAddress).Should(BeEquivalentTo(secondaryVmPodAddr))
					})
				}

				By("Deleting the VM.")
				Expect(virtClient.VirtualMachine(vm.Namespace).Delete(vm.Name, &k8smetav1.DeleteOptions{})).To(Succeed())
				tests.WaitForVirtualMachineToDisappearWithTimeout(vmi, 120)

				By("Verifying the endpoints' single subset, which points to the VM's pod, is deleted once the VM was deleted.")
				svcEndpoints, err = virtClient.CoreV1().Endpoints(vm.Namespace).Get(context.Background(), endpointsName, k8smetav1.GetOptions{})
				Expect(err).ToNot(HaveOccurred())
				Expect(svcEndpoints.Subsets).To(BeNil())

				By("Starting a job which tries to reach the VMI via the ClusterIP service.")
				job := runHelloWorldJob(serviceIP, servicePort, vm.Namespace)

				By("Waiting for the job to report a failed connection attempt.")
				Expect(tests.WaitForJobToFail(job, 240*time.Second)).To(Succeed())
			},
				table.Entry("[test_id:343] over default IPv4 IP family", ipv4),
				table.Entry("over IPv6 IP family", ipv6),
				table.Entry("over dual stack, primary ipv4", dualIPv4Primary),
				table.Entry("over dual stack, primary ipv6", dualIPv6Primary),
			)
		})
	})
})

func generateBatchJobSpec(host string, port string, clientBuilder func(host string, port string, checkConnectivityCmdPrefixes ...string) *batchv1.Job) *batchv1.Job {
	if net.IsIPv6String(host) {
		// TODO - remove this if condition code once https://github.com/kubevirt/kubevirt/issues/4428 is fixed
		return clientBuilder(host, port, fmt.Sprintf("ping -w 20 %s;", host))
	}
	return clientBuilder(host, port)
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
	ahostsCmd := string("ahosts" + ipFamily[2:])
	output, err := tests.ExecuteCommandOnPod(
		virtclient,
		pod,
		"compute",
		[]string{"getent", ahostsCmd, hostname})

	if err != nil {
		return "", err
	}

	splitGetent := strings.Split(output, "\n")
	if len(splitGetent) > 0 {
		ip := strings.Split(splitGetent[0], " ")[0]
		return ip, nil
	}

	return "", fmt.Errorf("could not resolve an %s address from %s name", ipFamily, hostname)
}

func resolveNodeIPAddrByFamily(virtClient kubecli.KubevirtClient, sourcePod *k8sv1.Pod, node k8sv1.Node, ipFamily k8sv1.IPFamily) (string, error) {
	hostname := getNodeHostname(node.Status.Addresses)
	if hostname == nil {
		return "", fmt.Errorf("could not get node hostname")
	}
	return resolveNodeIp(virtClient, sourcePod, *hostname, ipFamily)
}
