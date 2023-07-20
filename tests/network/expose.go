package network

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"time"

	"kubevirt.io/kubevirt/tests/decorators"
	"kubevirt.io/kubevirt/tests/framework/kubevirt"

	"kubevirt.io/kubevirt/tests/console"
	"kubevirt.io/kubevirt/tests/exec"
	"kubevirt.io/kubevirt/tests/testsuite"

	"kubevirt.io/kubevirt/tests/libnet/cluster"
	"kubevirt.io/kubevirt/tests/libnet/job"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	batchv1 "k8s.io/api/batch/v1"
	k8sv1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	k8smetav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/rand"
	netutils "k8s.io/utils/net"

	v1 "kubevirt.io/api/core/v1"
	"kubevirt.io/client-go/kubecli"

	"kubevirt.io/kubevirt/tests"
	"kubevirt.io/kubevirt/tests/clientcmd"
	"kubevirt.io/kubevirt/tests/libnet"
	"kubevirt.io/kubevirt/tests/libvmi"
)

const (
	shouldExposeServiceViaVirtctl = "should expose a service via `virtctl expose ...`"
	gettingValidatingClusterIP    = "Getting and validating the cluster IP given for the service"
	iteratingClusterIPs           = "Iterating over the ClusterIPs and run hello-world job"
	overIPv6Family                = "over IPv6 IP family"
	overDualStackIPv4             = "over dual stack, primary ipv4"
	overDualStackIPv6             = "over dual stack, primary ipv6"
	shouldStartVM                 = "should have been able to start the VM"
)

func newLabeledVMI(label string) (vmi *v1.VirtualMachineInstance) {
	ports := []v1.Port{{Name: "http", Port: 80},
		{Name: "test-port-tcp", Port: 1500, Protocol: "TCP"},
		{Name: "udp", Port: 82, Protocol: "UDP"},
		{Name: "test-port-udp", Port: 1500, Protocol: "UDP"}}
	vmi = libvmi.NewAlpineWithTestTooling(
		libvmi.WithMasqueradeNetworking(ports...)...,
	)
	vmi.Labels = map[string]string{"expose": label}
	return
}

type ipFamily string

const (
	ipv4            ipFamily = "ipv4"
	ipv6            ipFamily = "ipv6"
	dualIPv4Primary ipFamily = "ipv4,ipv6"
	dualIPv6Primary ipFamily = "ipv6,ipv4"
)

func isDualStack(ipFamily ipFamily) bool {
	return ipFamily == dualIPv4Primary || ipFamily == dualIPv6Primary
}

func inlcudesIpv6(ipFamily ipFamily) bool {
	return ipFamily != ipv4
}

func includesIpv4(ipFamily ipFamily) bool {
	return ipFamily != ipv6
}

var _ = SIGDescribe("[rfe_id:253][crit:medium][vendor:cnv-qe@redhat.com][level:component]Expose", decorators.Expose, func() {

	var virtClient kubecli.KubevirtClient
	var err error

	const testPort = 1500

	BeforeEach(func() {
		virtClient = kubevirt.Client()
	})

	createAndWaitForJobToSucceed := func(jobFactory func(host, port string) *batchv1.Job, namespace, ip, port, viaMessage string) error {
		By(fmt.Sprintf("Starting a job which tries to reach the VMI via the %s", viaMessage))
		jobInstance := jobFactory(ip, port)
		jobInstance, err := virtClient.BatchV1().Jobs(namespace).Create(context.Background(), jobInstance, metav1.CreateOptions{})
		ExpectWithOffset(1, err).ToNot(HaveOccurred())

		By("Waiting for the job to report a successful connection attempt")
		return job.WaitForJobToSucceed(jobInstance, time.Duration(120)*time.Second)
	}

	getService := func(namespace, serviceName string) (*k8sv1.Service, error) {
		svc, err := virtClient.CoreV1().Services(namespace).Get(context.Background(), serviceName, k8smetav1.GetOptions{})
		if err != nil {
			return nil, err
		}

		return svc, nil
	}

	runJobsAgainstService := func(svc *k8sv1.Service, namespace string, jobFactories ...func(host, port string) *batchv1.Job) {
		serviceIPs := svc.Spec.ClusterIPs
		for _, jobFactory := range jobFactories {
			for ipOrderNum, ip := range serviceIPs {
				servicePort := fmt.Sprint(svc.Spec.Ports[0].Port)
				Expect(createAndWaitForJobToSucceed(jobFactory, namespace, ip, servicePort, fmt.Sprintf("%d ClusterIP", ipOrderNum+1))).To(Succeed())
			}
		}
	}

	Context("Expose service on a VM", func() {
		var tcpVM *v1.VirtualMachineInstance
		BeforeEach(func() {
			tcpVM = newLabeledVMI("vm")
			tcpVM = tests.RunVMIAndExpectLaunch(tcpVM, 180)
			tests.GenerateHelloWorldServer(tcpVM, testPort, "tcp", console.LoginToAlpine, false)
		})

		Context("Expose ClusterIP service", func() {
			const servicePort = "27017"
			const serviceNamePrefix = "cluster-ip-vmi"

			var serviceName string
			var vmiExposeArgs []string

			BeforeEach(func() {
				serviceName = randomizeName(serviceNamePrefix)
				vmiExposeArgs = libnet.NewVMIExposeArgs(tcpVM,
					libnet.WithPort(servicePort),
					libnet.WithServiceName(serviceName),
					libnet.WithTargetPort(strconv.Itoa(testPort)))
			})

			DescribeTable("[label:masquerade_binding_connectivity]Should expose a Cluster IP service on a VMI and connect to it", func(ipFamily ipFamily) {
				skipIfNotSupportedCluster(ipFamily)
				vmiExposeArgs = appendIpFamilyToExposeArgs(ipFamily, vmiExposeArgs)

				By("Exposing the service via virtctl command")
				Expect(executeVirtctlExposeCommand(vmiExposeArgs)).To(Succeed(), shouldExposeServiceViaVirtctl)

				By(gettingValidatingClusterIP)
				svc, err := getService(tcpVM.Namespace, serviceName)
				Expect(err).ToNot(HaveOccurred())
				err = validateClusterIp(svc.Spec.ClusterIP, ipFamily)
				Expect(err).ToNot(HaveOccurred())

				By(iteratingClusterIPs)
				runJobsAgainstService(svc, tcpVM.Namespace, job.NewHelloWorldJobTCP)
			},
				Entry("[test_id:1531] over default IPv4 IP family", ipv4),
				Entry(overIPv6Family, ipv6),
				Entry(overDualStackIPv4, dualIPv4Primary),
				Entry(overDualStackIPv6, dualIPv6Primary),
			)
		})

		Context("Expose ClusterIP service with string target-port", func() {
			const servicePort = "27017"
			const serviceNamePrefix = "cluster-ip-target-vmi"

			var serviceName string
			var vmiExposeArgs []string

			BeforeEach(func() {
				serviceName = randomizeName(serviceNamePrefix)
				vmiExposeArgs = libnet.NewVMIExposeArgs(tcpVM,
					libnet.WithPort(servicePort),
					libnet.WithServiceName(serviceName),
					libnet.WithTargetPort("http"))
			})

			DescribeTable("Should expose a ClusterIP service and connect to the vm on port 80", func(ipFamily ipFamily) {
				skipIfNotSupportedCluster(ipFamily)
				vmiExposeArgs = appendIpFamilyToExposeArgs(ipFamily, vmiExposeArgs)

				By("Exposing the service via virtctl command")
				Expect(executeVirtctlExposeCommand(vmiExposeArgs)).To(Succeed(), shouldExposeServiceViaVirtctl)

				By("Waiting for kubernetes to create the relevant endpoint")
				getEndpoint := func() error {
					_, err := virtClient.CoreV1().Endpoints(testsuite.GetTestNamespace(nil)).Get(context.Background(), serviceName, k8smetav1.GetOptions{})
					return err
				}
				Eventually(getEndpoint, 60, 1).Should(BeNil())

				endpoints, err := virtClient.CoreV1().Endpoints(testsuite.GetTestNamespace(nil)).Get(context.Background(), serviceName, k8smetav1.GetOptions{})
				Expect(err).ToNot(HaveOccurred())

				Expect(endpoints.Subsets).To(HaveLen(1))
				endpoint := endpoints.Subsets[0]
				Expect(endpoint.Ports).To(HaveLen(1))
				Expect(endpoint.Ports[0].Port).To(Equal(int32(80)))

				endpointSlices, err := virtClient.DiscoveryV1().EndpointSlices(testsuite.GetTestNamespace(nil)).List(context.Background(), metav1.ListOptions{})
				Expect(err).ToNot(HaveOccurred())

				numOfExpectedAddresses := 1
				addresses := []string{}
				isDualStack := isDualStack(ipFamily)
				if isDualStack {
					numOfExpectedAddresses = 2
				}

				for _, endpointSlice := range endpointSlices.Items {
					for _, endpoint := range endpointSlice.Endpoints {
						addresses = append(addresses, endpoint.Addresses...)
					}
				}
				Expect(addresses).To(HaveLen(numOfExpectedAddresses))
			},
				Entry("[test_id:1532] over default IPv4 IP family", ipv4),
				Entry(overIPv6Family, ipv6),
				Entry(overDualStackIPv4, dualIPv4Primary),
				Entry(overDualStackIPv6, dualIPv6Primary),
			)
		})

		Context("Expose ClusterIP service with ports on the vmi defined", func() {
			const serviceNamePrefix = "cluster-ip-target-multiple-ports-vmi"

			var serviceName string
			var vmiExposeArgs []string

			BeforeEach(func() {
				serviceName = randomizeName(serviceNamePrefix)
				vmiExposeArgs = libnet.NewVMIExposeArgs(tcpVM,
					libnet.WithServiceName(serviceName))
			})

			DescribeTable("Should expose a ClusterIP service and connect to all ports defined on the vmi", func(ipFamily ipFamily) {
				skipIfNotSupportedCluster(ipFamily)
				vmiExposeArgs = appendIpFamilyToExposeArgs(ipFamily, vmiExposeArgs)

				By("Exposing the service via virtctl command")
				Expect(executeVirtctlExposeCommand(vmiExposeArgs)).To(Succeed(), shouldExposeServiceViaVirtctl)

				By("Waiting for kubernetes to create the relevant endpoint")
				getEndpoint := func() error {
					_, err := virtClient.CoreV1().Endpoints(testsuite.GetTestNamespace(nil)).Get(context.Background(), serviceName, k8smetav1.GetOptions{})
					return err
				}
				Eventually(getEndpoint, 60, 1).Should(BeNil())

				endpoints, err := virtClient.CoreV1().Endpoints(testsuite.GetTestNamespace(nil)).Get(context.Background(), serviceName, k8smetav1.GetOptions{})
				Expect(err).ToNot(HaveOccurred())

				Expect(endpoints.Subsets).To(HaveLen(1))
				endpoint := endpoints.Subsets[0]
				Expect(endpoint.Ports).To(HaveLen(4))
				Expect(endpoint.Ports).To(ContainElement(k8sv1.EndpointPort{Name: "port-1", Port: 80, Protocol: "TCP"}))
				Expect(endpoint.Ports).To(ContainElement(k8sv1.EndpointPort{Name: "port-2", Port: 1500, Protocol: "TCP"}))
				Expect(endpoint.Ports).To(ContainElement(k8sv1.EndpointPort{Name: "port-3", Port: 82, Protocol: "UDP"}))
				Expect(endpoint.Ports).To(ContainElement(k8sv1.EndpointPort{Name: "port-4", Port: 1500, Protocol: "UDP"}))
			},
				Entry("[test_id:1533] over default IPv4 IP family", ipv4),
				Entry(overIPv6Family, ipv6),
				Entry(overDualStackIPv4, dualIPv4Primary),
				Entry(overDualStackIPv6, dualIPv6Primary),
			)
		})

		Context("Expose ClusterIP service with IPFamilyPolicy", func() {
			const serviceNamePrefix = "cluster-ip-with-ip-family-policy"

			var serviceName string
			var vmiExposeArgs []string

			BeforeEach(func() {
				serviceName = randomizeName(serviceNamePrefix)
				vmiExposeArgs = libnet.NewVMIExposeArgs(tcpVM,
					libnet.WithServiceName(serviceName))
			})

			DescribeTable("Should expose a ClusterIP service with the correct IPFamilyPolicy", func(ipFamiyPolicy k8sv1.IPFamilyPolicyType) {
				if ipFamiyPolicy == k8sv1.IPFamilyPolicyRequireDualStack {
					libnet.SkipWhenNotDualStackCluster()
				}

				calcNumOfClusterIPs := func() int {
					switch ipFamiyPolicy {
					case k8sv1.IPFamilyPolicySingleStack:
						return 1
					case k8sv1.IPFamilyPolicyPreferDualStack:
						isClusterDualStack, err := cluster.DualStack()
						ExpectWithOffset(1, err).NotTo(HaveOccurred(), "should have been able to infer if the cluster is dual stack")
						if isClusterDualStack {
							return 2
						}
						return 1
					case k8sv1.IPFamilyPolicyRequireDualStack:
						return 2
					}
					return 0
				}

				vmiExposeArgs = append(vmiExposeArgs, "--ip-family-policy", string(ipFamiyPolicy))

				By("Exposing the service via virtctl command")
				Expect(executeVirtctlExposeCommand(vmiExposeArgs)).To(Succeed(), shouldExposeServiceViaVirtctl)

				By("Getting the service")
				svc, err := getService(tcpVM.Namespace, serviceName)
				Expect(err).ToNot(HaveOccurred())

				By("Validating the num of cluster ips")
				Expect(svc.Spec.ClusterIPs).To(HaveLen(calcNumOfClusterIPs()))
			},
				Entry("over SingleStack IP family policy", k8sv1.IPFamilyPolicySingleStack),
				Entry("over PreferDualStack IP family policy", k8sv1.IPFamilyPolicyPreferDualStack),
				Entry("over RequireDualStack IP family policy", k8sv1.IPFamilyPolicyRequireDualStack),
			)
		})

		Context("Expose NodePort service", func() {
			const servicePort = "27017"
			const serviceNamePrefix = "node-port-vmi"

			var serviceName string
			var vmiExposeArgs []string

			BeforeEach(func() {
				serviceName = randomizeName(serviceNamePrefix)
				vmiExposeArgs = libnet.NewVMIExposeArgs(tcpVM,
					libnet.WithPort(servicePort),
					libnet.WithServiceName(serviceName),
					libnet.WithTargetPort(strconv.Itoa(testPort)),
					libnet.WithType("NodePort"))
			})

			DescribeTable("[label:masquerade_binding_connectivity]Should expose a NodePort service on a VMI and connect to it", func(ipFamily ipFamily) {
				skipIfNotSupportedCluster(ipFamily)
				vmiExposeArgs = appendIpFamilyToExposeArgs(ipFamily, vmiExposeArgs)

				By("Exposing the service via virtctl command")
				Expect(executeVirtctlExposeCommand(vmiExposeArgs)).To(Succeed(), shouldExposeServiceViaVirtctl)

				By("Getting the service")
				svc, err := getService(tcpVM.Namespace, serviceName)
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

					if includesIpv4(ipFamily) {
						By("Connecting to IPv4 node IP")
						Expect(createAndWaitForJobToSucceed(job.NewHelloWorldJobTCP, tcpVM.Namespace, nodeIP, strconv.Itoa(int(nodePort)), fmt.Sprintf("NodePort using %s node ip", ipFamily))).To(Succeed())
					}
					if inlcudesIpv6(ipFamily) {
						launcher, err := libvmi.GetPodByVirtualMachineInstance(tcpVM, tcpVM.GetNamespace())
						Expect(err).ToNot(HaveOccurred())
						ipv6NodeIP, err = resolveNodeIPAddrByFamily(
							virtClient,
							launcher,
							node,
							k8sv1.IPv6Protocol)
						Expect(err).NotTo(HaveOccurred(), "must have been able to resolve an IP address from the node name")
						Expect(ipv6NodeIP).NotTo(BeEmpty(), "must have been able to resolve the IPv6 address of the node")

						By("Connecting to IPv6 node IP")
						Expect(createAndWaitForJobToSucceed(job.NewHelloWorldJobTCP, tcpVM.Namespace, ipv6NodeIP, strconv.Itoa(int(nodePort)), fmt.Sprintf("NodePort using %s node ip", ipFamily))).To(Succeed())
					}
				}
			},
				Entry("[test_id:1534] over default IPv4 IP family", ipv4),
				Entry(overIPv6Family, ipv6),
				Entry(overDualStackIPv4, dualIPv4Primary),
				Entry(overDualStackIPv6, dualIPv6Primary),
			)
		})
	})

	Context("Expose UDP service on a VMI", func() {
		var udpVM *v1.VirtualMachineInstance
		BeforeEach(func() {
			udpVM = newLabeledVMI("udp-vm")
			udpVM = tests.RunVMIAndExpectLaunch(udpVM, 180)
			tests.GenerateHelloWorldServer(udpVM, testPort, "udp", console.LoginToAlpine, false)
		})

		Context("Expose ClusterIP UDP service", func() {
			const servicePort = "28017"
			const serviceNamePrefix = "cluster-ip-udp-vmi"

			var serviceName string
			var vmiExposeArgs []string

			BeforeEach(func() {
				serviceName = randomizeName(serviceNamePrefix)
				vmiExposeArgs = libnet.NewVMIExposeArgs(udpVM,
					libnet.WithPort(servicePort),
					libnet.WithServiceName(serviceName),
					libnet.WithTargetPort(strconv.Itoa(testPort)),
					libnet.WithProtocol("UDP"))
			})

			DescribeTable("[label:masquerade_binding_connectivity]Should expose a ClusterIP service on a VMI and connect to it", func(ipFamily ipFamily) {
				skipIfNotSupportedCluster(ipFamily)
				vmiExposeArgs = appendIpFamilyToExposeArgs(ipFamily, vmiExposeArgs)

				By("Exposing the service via virtctl command")
				Expect(executeVirtctlExposeCommand(vmiExposeArgs)).To(Succeed(), shouldExposeServiceViaVirtctl)

				By(gettingValidatingClusterIP)
				svc, err := getService(udpVM.Namespace, serviceName)
				Expect(err).ToNot(HaveOccurred())
				err = validateClusterIp(svc.Spec.ClusterIP, ipFamily)
				Expect(err).ToNot(HaveOccurred())

				By(iteratingClusterIPs)
				runJobsAgainstService(svc, udpVM.Namespace, job.NewHelloWorldJobUDP)
			},
				Entry("[test_id:1535] over default IPv4 IP family", ipv4),
				Entry(overIPv6Family, ipv6),
				Entry(overDualStackIPv4, dualIPv4Primary),
				Entry(overDualStackIPv6, dualIPv6Primary),
			)
		})

		Context("Expose NodePort UDP service", func() {
			const servicePort = "29017"
			const serviceNamePrefix = "node-port-udp-vmi"

			var serviceName string
			var vmiExposeArgs []string

			BeforeEach(func() {
				serviceName = randomizeName(serviceNamePrefix)
				vmiExposeArgs = libnet.NewVMIExposeArgs(udpVM,
					libnet.WithPort(servicePort),
					libnet.WithServiceName(serviceName),
					libnet.WithTargetPort(strconv.Itoa(testPort)),
					libnet.WithType("NodePort"),
					libnet.WithProtocol("UDP"))
			})

			DescribeTable("[label:masquerade_binding_connectivity]Should expose a NodePort service on a VMI and connect to it", func(ipFamily ipFamily) {
				skipIfNotSupportedCluster(ipFamily)
				vmiExposeArgs = appendIpFamilyToExposeArgs(ipFamily, vmiExposeArgs)

				By("Exposing the service via virtctl command")
				Expect(executeVirtctlExposeCommand(vmiExposeArgs)).To(Succeed(), shouldExposeServiceViaVirtctl)

				By(gettingValidatingClusterIP)
				svc, err := getService(udpVM.Namespace, serviceName)
				Expect(err).ToNot(HaveOccurred())
				err = validateClusterIp(svc.Spec.ClusterIP, ipFamily)
				Expect(err).ToNot(HaveOccurred())

				nodePort := svc.Spec.Ports[0].NodePort
				Expect(nodePort).To(BeNumerically(">", 0))

				By(iteratingClusterIPs)
				runJobsAgainstService(svc, udpVM.Namespace, job.NewHelloWorldJobUDP)

				By("Getting the node IP from all nodes")
				nodes, err := virtClient.CoreV1().Nodes().List(context.Background(), k8smetav1.ListOptions{})
				Expect(err).ToNot(HaveOccurred())
				Expect(nodes.Items).ToNot(BeEmpty())
				for _, node := range nodes.Items {
					Expect(node.Status.Addresses).ToNot(BeEmpty())
					nodeIP := node.Status.Addresses[0].Address
					var ipv6NodeIP string
					if inlcudesIpv6(ipFamily) {
						launcher, err := libvmi.GetPodByVirtualMachineInstance(udpVM, udpVM.GetNamespace())
						Expect(err).ToNot(HaveOccurred())
						ipv6NodeIP, err = resolveNodeIPAddrByFamily(
							virtClient,
							launcher,
							node,
							k8sv1.IPv6Protocol)
						Expect(err).NotTo(HaveOccurred(), "must have been able to resolve an IP address from the node name")
						Expect(ipv6NodeIP).NotTo(BeEmpty(), "must have been able to resolve the IPv6 address of the node")
					}

					if includesIpv4(ipFamily) {
						By("Connecting to IPv4 node IP")
						Expect(createAndWaitForJobToSucceed(job.NewHelloWorldJobUDP, udpVM.Namespace, nodeIP, strconv.Itoa(int(nodePort)), "NodePort ipv4 address")).To(Succeed())
					}
					if inlcudesIpv6(ipFamily) {
						By("Connecting to IPv6 node IP")
						Expect(createAndWaitForJobToSucceed(job.NewHelloWorldJobUDP, udpVM.Namespace, ipv6NodeIP, strconv.Itoa(int(nodePort)), "NodePort ipv6 address")).To(Succeed())
					}
				}
			},
				Entry("[test_id:1536] over default IPv4 IP family", ipv4),
				Entry(overIPv6Family, ipv6),
				Entry(overDualStackIPv4, dualIPv4Primary),
				Entry(overDualStackIPv6, dualIPv6Primary),
			)
		})
	})

	Context("Expose service on a VMI replica set", func() {
		const numberOfVMs = 2

		var vmrs *v1.VirtualMachineInstanceReplicaSet
		BeforeEach(func() {
			By("Creating a VMRS object with 2 replicas")
			vmrs = tests.NewRandomReplicaSetFromVMI(newLabeledVMI("vmirs"), int32(numberOfVMs))
			vmrs.Labels = map[string]string{"expose": "vmirs"}

			By("Start the replica set")
			vmrs, err = virtClient.ReplicaSet(testsuite.GetTestNamespace(nil)).Create(vmrs)
			Expect(err).ToNot(HaveOccurred())

			By("Checking the number of ready replicas")
			Eventually(func() int {
				rs, err := virtClient.ReplicaSet(testsuite.GetTestNamespace(nil)).Get(vmrs.ObjectMeta.Name, k8smetav1.GetOptions{})
				Expect(err).ToNot(HaveOccurred())
				return int(rs.Status.ReadyReplicas)
			}, 120*time.Second, 1*time.Second).Should(Equal(numberOfVMs))

			By("Add an 'hello world' server on each VMI in the replica set")
			// TODO: add label to list options
			// check size of list
			// remove check for owner
			vms, err := virtClient.VirtualMachineInstance(vmrs.ObjectMeta.Namespace).List(context.Background(), &k8smetav1.ListOptions{})
			Expect(err).ToNot(HaveOccurred())
			for _, vm := range vms.Items {
				if vm.OwnerReferences != nil {
					tests.GenerateHelloWorldServer(&vm, testPort, "tcp", console.LoginToAlpine, false)
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
				vmirsExposeArgs = libnet.NewVMIRSExposeArgs(vmrs,
					libnet.WithPort(servicePort),
					libnet.WithServiceName(serviceName),
					libnet.WithTargetPort(strconv.Itoa(testPort)))
			})

			DescribeTable("[label:masquerade_binding_connectivity]Should create a ClusterIP service on VMRS and connect to it", func(ipFamily ipFamily) {
				skipIfNotSupportedCluster(ipFamily)
				vmirsExposeArgs = appendIpFamilyToExposeArgs(ipFamily, vmirsExposeArgs)

				By("Exposing the service via virtctl command")
				Expect(executeVirtctlExposeCommand(vmirsExposeArgs)).To(Succeed(), shouldExposeServiceViaVirtctl)

				By(gettingValidatingClusterIP)
				svc, err := getService(vmrs.Namespace, serviceName)
				Expect(err).ToNot(HaveOccurred())
				err = validateClusterIp(svc.Spec.ClusterIP, ipFamily)
				Expect(err).ToNot(HaveOccurred())

				By(iteratingClusterIPs)
				runJobsAgainstService(svc, vmrs.Namespace, job.NewHelloWorldJobTCP)
			},
				Entry("[test_id:1537] over default IPv4 IP family", ipv4),
				Entry(overIPv6Family, ipv6),
				Entry(overDualStackIPv4, dualIPv4Primary),
				Entry(overDualStackIPv6, dualIPv6Primary),
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
			virtctl := clientcmd.NewRepeatableVirtctlCommand("start", "--namespace", namespace, name)
			Expect(virtctl()).To(Succeed(), "should succeed starting a VMI via `virtctl start ...`")

			By("Getting the status of the VMI")
			Eventually(func() bool {
				vm, err := virtClient.VirtualMachine(namespace).Get(context.Background(), name, &k8smetav1.GetOptions{})
				Expect(err).ToNot(HaveOccurred())
				return vm.Status.Ready
			}, 120*time.Second, 1*time.Second).Should(BeTrue())

			By("Getting the running VMI")
			var vmi *v1.VirtualMachineInstance
			Eventually(func() bool {
				vmi, err = virtClient.VirtualMachineInstance(namespace).Get(context.Background(), name, &k8smetav1.GetOptions{})
				Expect(err).ToNot(HaveOccurred())
				return vmi.Status.Phase == v1.Running
			}, 120*time.Second, 1*time.Second).Should(BeTrue())

			return vmi
		}

		createStoppedVM := func(virtClient kubecli.KubevirtClient, namespace string) (*v1.VirtualMachine, error) {
			By("Creating an VM object")
			vm := tests.NewRandomVirtualMachine(newLabeledVMI("vm"), false)

			By("Creating the VM")
			vm, err = virtClient.VirtualMachine(namespace).Create(context.Background(), vm)
			return vm, err
		}

		startVMWithServer := func(virtClient kubecli.KubevirtClient, protocol string, port int) *v1.VirtualMachineInstance {
			By("Calling the start command on the stopped VM")
			vmi := startVMIFromVMTemplate(virtClient, vm.GetName(), vm.GetNamespace())
			if vmi == nil {
				return nil
			}
			tests.GenerateHelloWorldServer(vmi, port, protocol, console.LoginToAlpine, false)
			return vmi
		}

		Context("Expose a VM as a ClusterIP service.", func() {
			var serviceName string

			BeforeEach(func() {
				vm, err = createStoppedVM(virtClient, testsuite.GetTestNamespace(nil))
				Expect(err).NotTo(HaveOccurred(), "should create a stopped VM.")
			})

			BeforeEach(func() {
				serviceName = randomizeName(serviceNamePrefix)
				vmExposeArgs = libnet.NewVMExposeArgs(vm,
					libnet.WithPort(servicePort),
					libnet.WithServiceName(serviceName),
					libnet.WithTargetPort(strconv.Itoa(testPort)))
			})

			DescribeTable("[label:masquerade_binding_connectivity]Connect to ClusterIP service that was set when VM was offline.", func(ipFamily ipFamily) {
				skipIfNotSupportedCluster(ipFamily)
				vmExposeArgs = appendIpFamilyToExposeArgs(ipFamily, vmExposeArgs)

				By("Exposing the service via virtctl command")
				Expect(executeVirtctlExposeCommand(vmExposeArgs)).To(Succeed(), shouldExposeServiceViaVirtctl)

				vmi := startVMWithServer(virtClient, "tcp", testPort)
				Expect(vmi).NotTo(BeNil(), shouldStartVM)

				// This TC also covers:
				// [test_id:1795] Exposed VM (as a service) can be reconnected multiple times.
				By(gettingValidatingClusterIP)
				svc, err := getService(vm.Namespace, serviceName)
				Expect(err).ToNot(HaveOccurred())
				err = validateClusterIp(svc.Spec.ClusterIP, ipFamily)
				Expect(err).ToNot(HaveOccurred())

				By(iteratingClusterIPs)
				runJobsAgainstService(svc, vm.Namespace, job.NewHelloWorldJobTCP)
			},
				Entry("[test_id:1538] over default IPv4 IP family", ipv4),
				Entry(overIPv6Family, ipv6),
				Entry(overDualStackIPv4, dualIPv4Primary),
				Entry(overDualStackIPv6, dualIPv6Primary),
			)

			DescribeTable("[label:masquerade_binding_connectivity]Should verify the exposed service is functional before and after VM restart.", func(ipFamily ipFamily) {
				skipIfNotSupportedCluster(ipFamily)
				vmExposeArgs = appendIpFamilyToExposeArgs(ipFamily, vmExposeArgs)

				By("Exposing the service via virtctl command")
				Expect(executeVirtctlExposeCommand(vmExposeArgs)).To(Succeed(), shouldExposeServiceViaVirtctl)

				vmi := startVMWithServer(virtClient, "tcp", testPort)
				Expect(vmi).NotTo(BeNil(), shouldStartVM)

				vmObj := vm

				By(gettingValidatingClusterIP)
				svc, err := getService(vmObj.Namespace, serviceName)
				Expect(err).ToNot(HaveOccurred())
				err = validateClusterIp(svc.Spec.ClusterIP, ipFamily)
				Expect(err).ToNot(HaveOccurred())

				By(iteratingClusterIPs)
				runJobsAgainstService(svc, vmObj.Namespace, job.NewHelloWorldJobTCP)

				// Retrieve the current VMI UID, to be compared with the new UID after restart.
				vmi, err = virtClient.VirtualMachineInstance(vmObj.Namespace).Get(context.Background(), vmObj.Name, &k8smetav1.GetOptions{})
				Expect(err).ToNot(HaveOccurred())
				vmiUIdBeforeRestart := vmi.GetObjectMeta().GetUID()

				By("Restarting the running VM.")
				virtctl := clientcmd.NewRepeatableVirtctlCommand("restart", "--namespace", vmObj.Namespace, vmObj.Name)
				err = virtctl()
				Expect(err).ToNot(HaveOccurred())

				By("Verifying the VMI is back up AFTER restart (in Running status with new UID).")
				Eventually(func() bool {
					vmi, err = virtClient.VirtualMachineInstance(vmObj.Namespace).Get(context.Background(), vmObj.Name, &k8smetav1.GetOptions{})
					if errors.IsNotFound(err) {
						return false
					}
					Expect(err).ToNot(HaveOccurred())
					vmiUIdAfterRestart := vmi.GetObjectMeta().GetUID()
					newUId := vmiUIdAfterRestart != vmiUIdBeforeRestart
					return vmi.Status.Phase == v1.Running && newUId
				}, 120*time.Second, 1*time.Second).Should(BeTrue())

				By("Creating a TCP server on the VM.")
				tests.GenerateHelloWorldServer(vmi, testPort, "tcp", console.LoginToAlpine, false)

				By("Repeating the sequence as prior to restarting the VM: Connect to exposed ClusterIP service.")
				By(iteratingClusterIPs)
				runJobsAgainstService(svc, vmObj.Namespace, job.NewHelloWorldJobTCP)
			},
				Entry("[test_id:345] over default IPv4 IP family", ipv4),
				Entry(overIPv6Family, ipv6),
				Entry(overDualStackIPv4, dualIPv4Primary),
				Entry(overDualStackIPv6, dualIPv6Primary),
			)
		})
	})
})

func randomizeName(currentName string) string {
	return currentName + rand.String(5)
}

func validateClusterIp(clusterIp string, ipFamily ipFamily) error {
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

func skipIfNotSupportedCluster(ipFamily ipFamily) {
	if includesIpv4(ipFamily) {
		libnet.SkipWhenClusterNotSupportIpv4()
	}
	if inlcudesIpv6(ipFamily) {
		libnet.SkipWhenClusterNotSupportIpv6()
	}
}

func appendIpFamilyToExposeArgs(ipFamily ipFamily, vmiExposeArgs []string) []string {
	if inlcudesIpv6(ipFamily) {
		vmiExposeArgs = append(vmiExposeArgs, "--ip-family", string(ipFamily))
	}
	return vmiExposeArgs
}

func executeVirtctlExposeCommand(ExposeArgs []string) error {
	virtctl := clientcmd.NewRepeatableVirtctlCommand(ExposeArgs...)
	return virtctl()
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
	output, err := exec.ExecuteCommandOnPod(
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
