package tests_test

import (
	"flag"
	"fmt"
	"strconv"
	"time"

	expect "github.com/google/goexpect"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	k8sv1 "k8s.io/api/core/v1"
	k8smetav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	v1 "kubevirt.io/kubevirt/pkg/api/v1"
	"kubevirt.io/kubevirt/pkg/kubecli"
	"kubevirt.io/kubevirt/pkg/virtctl/expose"
	"kubevirt.io/kubevirt/tests"
)

func newLabeledVM(label string, virtClient kubecli.KubevirtClient) (vmi *v1.VirtualMachineInstance) {
	ports := []v1.Port{{Name: "http", Port: 80}, {Name: "udp", Port: 82, Protocol: "UDP"}}
	vmi = tests.NewRandomVMIWithBridgeInterfaceEphemeralDiskAndUserdata(tests.ContainerDiskFor(tests.ContainerDiskCirros), "#!/bin/bash\necho 'hello'\n", ports)
	vmi.Labels = map[string]string{"expose": label}
	vmi, err := virtClient.VirtualMachineInstance(tests.NamespaceTestDefault).Create(vmi)
	Expect(err).ToNot(HaveOccurred())
	tests.WaitForSuccessfulVMIStartIgnoreWarnings(vmi)
	vmi, err = virtClient.VirtualMachineInstance(tests.NamespaceTestDefault).Get(vmi.ObjectMeta.Name, &k8smetav1.GetOptions{})
	Expect(err).ToNot(HaveOccurred())
	return
}

func generateHelloWorldServer(vmi *v1.VirtualMachineInstance, virtClient kubecli.KubevirtClient, testPort int, protocol string) {
	expecter, err := tests.LoggedInCirrosExpecter(vmi)
	Expect(err).ToNot(HaveOccurred())
	defer expecter.Close()

	serverCommand := fmt.Sprintf("screen -d -m sudo nc -klp %d -e echo -e 'Hello World!'\n", testPort)
	if protocol == "udp" {
		// nc has to be in a while loop in case of UDP, since it exists after one message
		serverCommand = fmt.Sprintf("screen -d -m sh -c \"while true\n do nc -uklp %d -e echo -e 'Hello UDP World!'\ndone\n\"\n", testPort)
	}
	_, err = expecter.ExpectBatch([]expect.Batcher{
		&expect.BSnd{S: serverCommand},
		&expect.BExp{R: "\\$ "},
		&expect.BSnd{S: "echo $?\n"},
		&expect.BExp{R: "0"},
	}, 60*time.Second)
	Expect(err).ToNot(HaveOccurred())
}

var _ = Describe("Expose", func() {

	flag.Parse()

	virtClient, err := kubecli.GetKubevirtClient()
	tests.PanicOnError(err)
	const testPort = 1500

	Context("Expose service on a VM", func() {
		var tcpVM *v1.VirtualMachineInstance
		tests.BeforeAll(func() {
			tcpVM = newLabeledVM("vm", virtClient)
			generateHelloWorldServer(tcpVM, virtClient, testPort, "tcp")
		})

		Context("Expose ClusterIP service", func() {
			const servicePort = "27017"
			const serviceName = "cluster-ip-vmi"
			It("Should expose a Cluster IP service on a VMI and connect to it", func() {
				By("Exposing the service via virtctl command")
				virtctl := tests.NewRepeatableVirtctlCommand(expose.COMMAND_EXPOSE, "virtualmachineinstance", "--namespace",
					tcpVM.Namespace, tcpVM.Name, "--port", servicePort, "--name", serviceName, "--target-port", strconv.Itoa(testPort))
				err := virtctl()
				Expect(err).ToNot(HaveOccurred())

				By("Getting back the cluster IP given for the service")
				svc, err := virtClient.CoreV1().Services(tcpVM.Namespace).Get(serviceName, k8smetav1.GetOptions{})
				Expect(err).ToNot(HaveOccurred())
				serviceIP := svc.Spec.ClusterIP

				By("Starting a pod which tries to reach the VMI via ClusterIP")
				job := tests.NewHelloWorldJob(serviceIP, servicePort)
				job, err = virtClient.CoreV1().Pods(tcpVM.Namespace).Create(job)
				Expect(err).ToNot(HaveOccurred())

				By("Waiting for the pod to report a successful connection attempt")
				getStatus := func() k8sv1.PodPhase {
					pod, err := virtClient.CoreV1().Pods(job.Namespace).Get(job.Name, k8smetav1.GetOptions{})
					Expect(err).ToNot(HaveOccurred())
					return pod.Status.Phase
				}
				Eventually(getStatus, 60, 1).Should(Equal(k8sv1.PodSucceeded))
			})
		})

		Context("Expose ClusterIP service with string target-port", func() {
			const servicePort = "27017"
			const serviceName = "cluster-ip-target-vmi"
			It("Should expose a ClusterIP service and connect to the vm on port 80", func() {
				virtctl := tests.NewRepeatableVirtctlCommand(expose.COMMAND_EXPOSE, "virtualmachineinstance", "--namespace",
					tcpVM.Namespace, tcpVM.Name, "--port", servicePort, "--name", serviceName, "--target-port", "http")
				err := virtctl()
				Expect(err).ToNot(HaveOccurred())

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
			})
		})

		Context("Expose ClusterIP service wiht ports on the vmi defined", func() {
			const serviceName = "cluster-ip-target-multiple-ports-vmi"
			It("Should expose a ClusterIP service and connect to all ports defined on the vmi", func() {
				virtctl := tests.NewRepeatableVirtctlCommand(expose.COMMAND_EXPOSE, "virtualmachineinstance", "--namespace",
					tcpVM.Namespace, tcpVM.Name, "--name", serviceName)
				err := virtctl()
				Expect(err).ToNot(HaveOccurred())

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
				Expect(len(endpoint.Ports)).To(Equal(2))
				Expect(endpoint.Ports).To(ContainElement(k8sv1.EndpointPort{Name: "port-1", Port: 80, Protocol: "TCP"}))
				Expect(endpoint.Ports).To(ContainElement(k8sv1.EndpointPort{Name: "port-2", Port: 82, Protocol: "UDP"}))
			})
		})

		Context("Expose NodePort service", func() {
			const servicePort = "27017"
			const serviceName = "node-port-vmi"

			It("Should expose a NodePort service on a VMI and connect to it", func() {
				By("Exposing the service via virtctl command")
				virtctl := tests.NewRepeatableVirtctlCommand(expose.COMMAND_EXPOSE, "virtualmachineinstance", "--namespace",
					tcpVM.Namespace, tcpVM.Name, "--port", servicePort, "--name", serviceName, "--target-port", strconv.Itoa(testPort),
					"--type", "NodePort")
				err := virtctl()
				Expect(err).ToNot(HaveOccurred())

				By("Getting back the the service")
				svc, err := virtClient.CoreV1().Services(tcpVM.Namespace).Get(serviceName, k8smetav1.GetOptions{})
				Expect(err).ToNot(HaveOccurred())
				nodePort := svc.Spec.Ports[0].NodePort
				Expect(nodePort).To(BeNumerically(">", 0))

				By("Getting the node IP from all nodes")
				nodes, err := virtClient.CoreV1().Nodes().List(k8smetav1.ListOptions{})
				Expect(err).ToNot(HaveOccurred())
				Expect(nodes.Items).ToNot(BeEmpty())
				for _, node := range nodes.Items {
					Expect(node.Status.Addresses).ToNot(BeEmpty())
					nodeIP := node.Status.Addresses[0].Address

					By("Starting a pod which tries to reach the VMI via NodePort")
					job := tests.NewHelloWorldJob(nodeIP, strconv.Itoa(int(nodePort)))
					job, err = virtClient.CoreV1().Pods(tcpVM.Namespace).Create(job)
					Expect(err).ToNot(HaveOccurred())

					By("Waiting for the pod to report a successful connection attempt")
					getStatus := func() k8sv1.PodPhase {
						pod, err := virtClient.CoreV1().Pods(job.Namespace).Get(job.Name, k8smetav1.GetOptions{})
						Expect(err).ToNot(HaveOccurred())
						return pod.Status.Phase
					}
					Eventually(getStatus, 60, 1).Should(Equal(k8sv1.PodSucceeded))
				}
			})
		})
	})

	Context("Expose UDP service on a VMI", func() {
		var udpVM *v1.VirtualMachineInstance
		tests.BeforeAll(func() {
			udpVM = newLabeledVM("udp-vm", virtClient)
			generateHelloWorldServer(udpVM, virtClient, testPort, "udp")
		})

		Context("Expose ClusterIP UDP service", func() {
			const servicePort = "28017"
			const serviceName = "cluster-ip-udp-vmi"

			It("Should expose a ClusterIP service on a VMI and connect to it", func() {
				By("Exposing the service via virtctl command")
				virtctl := tests.NewRepeatableVirtctlCommand(expose.COMMAND_EXPOSE, "virtualmachineinstance", "--namespace",
					udpVM.Namespace, udpVM.Name, "--port", servicePort, "--name", serviceName, "--target-port", strconv.Itoa(testPort),
					"--protocol", "UDP")
				err := virtctl()
				Expect(err).ToNot(HaveOccurred())

				By("Getting back the cluster IP given for the service")
				svc, err := virtClient.CoreV1().Services(udpVM.Namespace).Get(serviceName, k8smetav1.GetOptions{})
				Expect(err).ToNot(HaveOccurred())
				serviceIP := svc.Spec.ClusterIP

				By("Starting a pod which tries to reach the VMI via ClusterIP")
				job := tests.NewHelloWorldJobUDP(serviceIP, servicePort)
				job, err = virtClient.CoreV1().Pods(udpVM.Namespace).Create(job)
				Expect(err).ToNot(HaveOccurred())

				By("Waiting for the pod to report a successful connection attempt")
				getStatus := func() k8sv1.PodPhase {
					pod, err := virtClient.CoreV1().Pods(job.Namespace).Get(job.Name, k8smetav1.GetOptions{})
					Expect(err).ToNot(HaveOccurred())
					return pod.Status.Phase
				}
				Eventually(getStatus, 60, 1).Should(Equal(k8sv1.PodSucceeded))
			})
		})

		Context("Expose NodePort UDP service", func() {
			const servicePort = "29017"
			const serviceName = "node-port-udp-vmi"

			It("Should expose a NodePort service on a VMI and connect to it", func() {
				By("Exposing the service via virtctl command")
				virtctl := tests.NewRepeatableVirtctlCommand(expose.COMMAND_EXPOSE, "virtualmachineinstance", "--namespace",
					udpVM.Namespace, udpVM.Name, "--port", servicePort, "--name", serviceName, "--target-port", strconv.Itoa(testPort),
					"--type", "NodePort", "--protocol", "UDP")
				err := virtctl()
				Expect(err).ToNot(HaveOccurred())

				By("Getting back the cluster IP given for the service")
				svc, err := virtClient.CoreV1().Services(udpVM.Namespace).Get(serviceName, k8smetav1.GetOptions{})
				Expect(err).ToNot(HaveOccurred())
				serviceIP := svc.Spec.ClusterIP
				nodePort := svc.Spec.Ports[0].NodePort
				Expect(nodePort).To(BeNumerically(">", 0))

				By("Starting a pod which tries to reach the VMI via ClusterIP")
				job := tests.NewHelloWorldJobUDP(serviceIP, servicePort)
				job, err = virtClient.CoreV1().Pods(udpVM.Namespace).Create(job)
				Expect(err).ToNot(HaveOccurred())

				By("Getting the node IP from all nodes")
				nodes, err := virtClient.CoreV1().Nodes().List(k8smetav1.ListOptions{})
				Expect(err).ToNot(HaveOccurred())
				Expect(nodes.Items).ToNot(BeEmpty())
				for _, node := range nodes.Items {
					Expect(node.Status.Addresses).ToNot(BeEmpty())
					nodeIP := node.Status.Addresses[0].Address

					By("Starting a pod which tries to reach the VMI via NodePort")
					job := tests.NewHelloWorldJobUDP(nodeIP, strconv.Itoa(int(nodePort)))
					job, err = virtClient.CoreV1().Pods(udpVM.Namespace).Create(job)
					Expect(err).ToNot(HaveOccurred())

					By("Waiting for the pod to report a successful connection attempt")
					getStatus := func() k8sv1.PodPhase {
						pod, err := virtClient.CoreV1().Pods(job.Namespace).Get(job.Name, k8smetav1.GetOptions{})
						Expect(err).ToNot(HaveOccurred())
						return pod.Status.Phase
					}
					Eventually(getStatus, 60, 1).Should(Equal(k8sv1.PodSucceeded))
				}
			})
		})
	})

	Context("Expose service on a VMI replica set", func() {
		var vmrs *v1.VirtualMachineInstanceReplicaSet
		tests.BeforeAll(func() {
			By("Creating a VMRS object with 2 replicas")
			const numberOfVMs = 2
			template := tests.NewRandomVMIWithEphemeralDiskAndUserdata(tests.ContainerDiskFor(tests.ContainerDiskCirros), "#!/bin/bash\necho 'hello'\n")
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

			By("add an 'hello world' server on each VMI in the replica set")
			// TODO: add label to list options
			// check size of list
			// remove check for owner
			vms, err := virtClient.VirtualMachineInstance(vmrs.ObjectMeta.Namespace).List(&k8smetav1.ListOptions{})
			Expect(err).ToNot(HaveOccurred())
			for _, vm := range vms.Items {
				if vm.OwnerReferences != nil {
					generateHelloWorldServer(&vm, virtClient, testPort, "tcp")
				}
			}
		})

		Context("Expose ClusterIP service", func() {
			const servicePort = "27017"
			const serviceName = "cluster-ip-vmirs"

			It("Should create a ClusterIP service on VMRS and connect to it", func() {
				By("Expose a service on the VMRS using virtctl")
				virtctl := tests.NewRepeatableVirtctlCommand(expose.COMMAND_EXPOSE, "vmirs", "--namespace",
					vmrs.Namespace, vmrs.Name, "--port", servicePort, "--name", serviceName, "--target-port", strconv.Itoa(testPort))
				err = virtctl()
				Expect(err).ToNot(HaveOccurred())

				By("Getting back the cluster IP given for the service")
				svc, err := virtClient.CoreV1().Services(vmrs.Namespace).Get(serviceName, k8smetav1.GetOptions{})
				Expect(err).ToNot(HaveOccurred())
				serviceIP := svc.Spec.ClusterIP

				By("Starting a pod which tries to reach the VMI via ClusterIP")
				job := tests.NewHelloWorldJob(serviceIP, servicePort)
				job, err = virtClient.CoreV1().Pods(vmrs.Namespace).Create(job)
				Expect(err).ToNot(HaveOccurred())

				By("Waiting for the pod to report a successful connection attempt")
				getStatus := func() k8sv1.PodPhase {
					pod, err := virtClient.CoreV1().Pods(job.Namespace).Get(job.Name, k8smetav1.GetOptions{})
					Expect(err).ToNot(HaveOccurred())
					return pod.Status.Phase
				}
				Eventually(getStatus, 60, 1).Should(Equal(k8sv1.PodSucceeded))
			})
		})
	})

	Context("Expose service on an VM", func() {
		const servicePort = "27017"
		const serviceName = "cluster-ip-vm"
		var vm *v1.VirtualMachine

		tests.BeforeAll(func() {
			By("Creating an VM object")
			template := tests.NewRandomVMIWithEphemeralDiskAndUserdata(tests.ContainerDiskFor(tests.ContainerDiskCirros), "#!/bin/bash\necho 'hello'\n")
			template.Labels = map[string]string{"expose": "vm"}
			vm = NewRandomVirtualMachine(template, false)

			By("Creating the VM")
			_, err := virtClient.VirtualMachine(tests.NamespaceTestDefault).Create(vm)
			Expect(err).ToNot(HaveOccurred())

			By("Exposing a service on the VM using virtctl")
			virtctl := tests.NewRepeatableVirtctlCommand(expose.COMMAND_EXPOSE, "virtualmachine", "--namespace",
				vm.Namespace, vm.Name, "--port", servicePort, "--name", serviceName, "--target-port", strconv.Itoa(testPort))
			err = virtctl()
			Expect(err).ToNot(HaveOccurred())

			By("Calling the start command")
			virtctl = tests.NewRepeatableVirtctlCommand("start", "--namespace", vm.Namespace, vm.Name)
			err = virtctl()
			Expect(err).ToNot(HaveOccurred())

			By("Getting the status of the VMI")
			Eventually(func() bool {
				vm, err = virtClient.VirtualMachine(vm.Namespace).Get(vm.Name, &k8smetav1.GetOptions{})
				Expect(err).ToNot(HaveOccurred())
				return vm.Status.Ready
			}, 120*time.Second, 1*time.Second).Should(BeTrue())

			By("Getting the running VMI")
			var vmi *v1.VirtualMachineInstance
			Eventually(func() bool {
				vmi, err = virtClient.VirtualMachineInstance(vm.Namespace).Get(vm.Name, &k8smetav1.GetOptions{})
				Expect(err).ToNot(HaveOccurred())
				return vmi.Status.Phase == v1.Running
			}, 120*time.Second, 1*time.Second).Should(BeTrue())

			generateHelloWorldServer(vmi, virtClient, testPort, "tcp")
		})

		Context("Expose ClusterIP service", func() {
			It("Connect to ClusterIP services that was set when VM was offline", func() {
				By("Getting back the cluster IP given for the service")
				svc, err := virtClient.CoreV1().Services(vm.Namespace).Get(serviceName, k8smetav1.GetOptions{})
				Expect(err).ToNot(HaveOccurred())
				serviceIP := svc.Spec.ClusterIP

				By("Starting a pod which tries to reach the VMI via ClusterIP")
				job := tests.NewHelloWorldJob(serviceIP, servicePort)
				job, err = virtClient.CoreV1().Pods(vm.Namespace).Create(job)
				Expect(err).ToNot(HaveOccurred())

				By("Waiting for the pod to report a successful connection attempt")
				getStatus := func() k8sv1.PodPhase {
					pod, err := virtClient.CoreV1().Pods(job.Namespace).Get(job.Name, k8smetav1.GetOptions{})
					Expect(err).ToNot(HaveOccurred())
					return pod.Status.Phase
				}

				Eventually(getStatus, 60, 1).Should(Equal(k8sv1.PodSucceeded))
			})
		})
	})
})
