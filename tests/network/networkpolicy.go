package network

import (
	"context"
	"fmt"
	"net"
	"strconv"
	"time"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	corev1 "k8s.io/api/core/v1"
	networkv1 "k8s.io/api/networking/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"

	v1 "kubevirt.io/client-go/api/v1"
	"kubevirt.io/client-go/kubecli"
	"kubevirt.io/kubevirt/tests"
	"kubevirt.io/kubevirt/tests/console"
	cd "kubevirt.io/kubevirt/tests/containerdisk"
	"kubevirt.io/kubevirt/tests/libnet"
	"kubevirt.io/kubevirt/tests/libvmi"
)

var _ = SIGDescribe("[Serial][rfe_id:150][crit:high][vendor:cnv-qe@redhat.com][level:component]Networkpolicy", func() {
	var (
		virtClient                                          kubecli.KubevirtClient
		serverVMI, clientVMI, clientVMIAlternativeNamespace *v1.VirtualMachineInstance
		serverVMILabels                                     = map[string]string{"type": "test"}
	)
	BeforeEach(func() {
		var err error
		virtClient, err = kubecli.GetKubevirtClient()
		Expect(err).ToNot(HaveOccurred(), "should succeed retrieving the kubevirt client")

		tests.SkipIfUseFlannel(virtClient)
		skipNetworkPolicyRunningOnKindInfra()

	})
	Context("when three cirros VMs with default networking are started and serverVMI start an HTTP server on port 80 and 81", func() {
		tests.BeforeAll(func() {
			tests.BeforeTestCleanup()

			// Create three vmis, serverVMI and clientVMI are in same namespace, clientVMIAlternativeNamespace is in different namespace
			serverVMI = createVMICirros(virtClient,
				tests.NamespaceTestDefault,
				serverVMILabels,
				libvmi.WithInterface(libvmi.InterfaceDeviceWithMasqueradeBinding(
					v1.Port{
						Name:     "http80",
						Port:     80,
						Protocol: "TCP",
					},
					v1.Port{
						Name:     "http81",
						Port:     81,
						Protocol: "TCP",
					},
				)),
			)
			clientVMI = createVMICirros(virtClient, tests.NamespaceTestDefault, map[string]string{}, libvmi.WithInterface(libvmi.InterfaceDeviceWithMasqueradeBinding()))
			clientVMIAlternativeNamespace = createVMICirros(virtClient, tests.NamespaceTestAlternative, map[string]string{}, libvmi.WithInterface(libvmi.InterfaceDeviceWithMasqueradeBinding()))

			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			serverVMIFuture := tests.WaitUntilVMIReadyAsync(ctx, serverVMI, libnet.WithIPv6(console.LoginToCirros))
			clientVMIFuture := tests.WaitUntilVMIReadyAsync(ctx, clientVMI, libnet.WithIPv6(console.LoginToCirros))
			clientVMIAlternativeNamespaceFuture := tests.WaitUntilVMIReadyAsync(ctx, clientVMIAlternativeNamespace, libnet.WithIPv6(console.LoginToCirros))

			serverVMI = serverVMIFuture()
			clientVMI = clientVMIFuture()
			clientVMIAlternativeNamespace = clientVMIAlternativeNamespaceFuture()

			By("Start HTTP server at serverVMI on ports 80 and 81")
			tests.HTTPServer.Start(serverVMI, 80)
			tests.HTTPServer.Start(serverVMI, 81)

			assertIPsNotEmptyForVMI(serverVMI)
			assertIPsNotEmptyForVMI(clientVMI)
			assertIPsNotEmptyForVMI(clientVMIAlternativeNamespace)

		})

		Context("and connectivity between VMI/s is blocked by Default-deny networkpolicy", func() {
			var policy *networkv1.NetworkPolicy

			BeforeEach(func() {
				// deny-by-default networkpolicy will deny all the traffic to the vms in the namespace
				policy = createNetworkPolicy(serverVMI.Namespace, "deny-by-default", metav1.LabelSelector{}, []networkv1.NetworkPolicyIngressRule{})
			})

			AfterEach(func() {
				waitForNetworkPolicyDeletion(policy)
			})

			It("[test_id:1511] should fail to reach serverVMI from clientVMI", func() {
				By("Connect serverVMI from clientVMI")
				assertPingFail(clientVMI, serverVMI)
			})

			It("[test_id:1512] should fail to reach clientVMI from serverVMI", func() {
				By("Connect clientVMI from serverVMI")
				assertPingFail(serverVMI, clientVMI)
			})
			It("[test_id:369] should deny http traffic for ports 80/81 from clientVMI to serverVMI", func() {
				assertHTTPPingFailed(clientVMI, serverVMI, 80)
				assertHTTPPingFailed(clientVMI, serverVMI, 81)
			})

		})

		Context("and vms limited by allow same namespace networkpolicy", func() {
			var policy *networkv1.NetworkPolicy

			BeforeEach(func() {
				// allow-same-namespace networkpolicy will only allow the traffic inside the namespace
				By("Create allow-same-namespace networkpolicy")
				policy = createNetworkPolicy(serverVMI.Namespace, "allow-same-namespace", metav1.LabelSelector{},
					[]networkv1.NetworkPolicyIngressRule{
						{
							From: []networkv1.NetworkPolicyPeer{
								{
									PodSelector: &metav1.LabelSelector{},
								},
							},
						},
					},
				)

			})

			AfterEach(func() {
				waitForNetworkPolicyDeletion(policy)
			})

			It("[Conformance][test_id:1513] should succeed pinging between two VMI/s in the same namespace", func() {
				assertPingSucceed(clientVMI, serverVMI)
			})

			It("[Conformance][test_id:1514] should fail pinging between two VMI/s each on different namespaces", func() {
				assertPingFail(clientVMIAlternativeNamespace, serverVMI)
			})

		})

		Context("and ingress traffic to VMI identified via label at networkprofile's labelSelector is blocked", func() {
			var policy *networkv1.NetworkPolicy

			BeforeEach(func() {
				// deny-by-label networkpolicy will deny the traffic for the vm which have the same label
				By("Create deny-by-label networkpolicy")
				policy = &networkv1.NetworkPolicy{
					ObjectMeta: metav1.ObjectMeta{
						Namespace: serverVMI.Namespace,
						Name:      "deny-by-label",
					},
					Spec: networkv1.NetworkPolicySpec{
						PodSelector: metav1.LabelSelector{
							MatchLabels: serverVMILabels,
						},
						Ingress: []networkv1.NetworkPolicyIngressRule{},
					},
				}
				policy = createNetworkPolicy(serverVMI.Namespace, "deny-by-label", metav1.LabelSelector{MatchLabels: serverVMILabels}, []networkv1.NetworkPolicyIngressRule{})
			})

			AfterEach(func() {
				waitForNetworkPolicyDeletion(policy)
			})

			It("[test_id:1515] should fail to reach serverVMI from clientVMIAlternativeNamespace", func() {
				By("Connect serverVMI from clientVMIAlternativeNamespace")
				assertPingFail(clientVMIAlternativeNamespace, serverVMI)
			})

			It("[test_id:1516] should fail to reach serverVMI from clientVMI", func() {
				By("Connect serverVMI from clientVMI")
				assertPingFail(clientVMI, serverVMI)
			})

			It("[test_id:1517] should success to reach clientVMI from clientVMIAlternativeNamespace", func() {
				By("Connect clientVMI from clientVMIAlternativeNamespace")
				assertPingSucceed(clientVMIAlternativeNamespace, clientVMI)
			})

		})
		Context("and TCP connectivity on ports 80 and 81 between VMI/s is allowed by networkpolicy", func() {
			var policy *networkv1.NetworkPolicy
			BeforeEach(func() {
				port80 := intstr.FromInt(80)
				port81 := intstr.FromInt(81)
				tcp := corev1.ProtocolTCP
				policy = createNetworkPolicy(serverVMI.Namespace, "allow-all-http-ports", metav1.LabelSelector{},
					[]networkv1.NetworkPolicyIngressRule{
						{
							Ports: []networkv1.NetworkPolicyPort{
								{Port: &port80, Protocol: &tcp},
								{Port: &port81, Protocol: &tcp},
							},
						},
					},
				)
			})
			AfterEach(func() {
				waitForNetworkPolicyDeletion(policy)
			})
			It("[test_id:2774] should allow http traffic for ports 80 and 81 from clientVMI to serverVMI", func() {
				assertHTTPPingSucceed(clientVMI, serverVMI, 80)
				assertHTTPPingSucceed(clientVMI, serverVMI, 81)
			})
		})
		Context("and TCP connectivity on ports 80 between VMI/s is allowed by networkpolicy", func() {
			var policy *networkv1.NetworkPolicy
			BeforeEach(func() {
				port80 := intstr.FromInt(80)
				tcp := corev1.ProtocolTCP
				policy = createNetworkPolicy(serverVMI.Namespace, "allow-http80-ports", metav1.LabelSelector{},
					[]networkv1.NetworkPolicyIngressRule{
						{
							Ports: []networkv1.NetworkPolicyPort{
								{Port: &port80, Protocol: &tcp},
							},
						},
					},
				)
			})
			AfterEach(func() {
				waitForNetworkPolicyDeletion(policy)
			})
			It("[test_id:2775] should allow http traffic at port 80 and deny at port 81 from clientVMI to serverVMI", func() {
				assertHTTPPingSucceed(clientVMI, serverVMI, 80)
				assertHTTPPingFailed(clientVMI, serverVMI, 81)
			})
		})

	})
})

func skipNetworkPolicyRunningOnKindInfra() {
	if tests.IsRunningOnKindInfra() {
		Skip("Skip Network Policy tests till issue https://github.com/kubevirt/kubevirt/issues/4081 is fixed")
	}
}

func createVMICirros(virtClient kubecli.KubevirtClient, namespace string, labels map[string]string, opts ...libvmi.Option) *v1.VirtualMachineInstance {
	var err error
	vmi := tests.NewRandomVMIWithEphemeralDiskAndUserdata(cd.ContainerDiskFor(cd.ContainerDiskCirros), "#!/bin/bash\necho 'hello'\n")
	vmi.Namespace = namespace
	vmi.Labels = labels

	// Clean up interfaces since we configure them with `libvmi.Option`
	vmi.Spec.Domain.Devices.Interfaces = []v1.Interface{}

	for _, opt := range opts {
		opt(vmi)
	}

	vmi, err = virtClient.VirtualMachineInstance(vmi.Namespace).Create(vmi)
	Expect(err).ToNot(HaveOccurred())

	return vmi
}

func assertPingSucceed(fromVmi, toVmi *v1.VirtualMachineInstance) {
	ConsistentlyWithOffset(1, func() error {
		for _, toIp := range toVmi.Status.Interfaces[0].IPs {
			if err := libnet.PingFromVMConsole(fromVmi, toIp); err != nil {
				return err
			}
		}
		return nil
	}, 15*time.Second, 1*time.Second).Should(Succeed())
}

func assertPingFail(fromVmi, toVmi *v1.VirtualMachineInstance) {

	EventuallyWithOffset(1, func() error {
		var err error
		for _, toIp := range toVmi.Status.Interfaces[0].IPs {
			if err = libnet.PingFromVMConsole(fromVmi, toIp); err == nil {
				return nil
			}
		}
		return err
	}, 15*time.Second, time.Second).ShouldNot(Succeed())

	ConsistentlyWithOffset(1, func() error {
		var err error
		for _, toIp := range toVmi.Status.Interfaces[0].IPs {
			if err = libnet.PingFromVMConsole(fromVmi, toIp); err == nil {
				return nil
			}
		}
		return err
	}, 5*time.Second, 1*time.Second).ShouldNot(Succeed())
}

func createNetworkPolicy(namespace, name string, labelSelector metav1.LabelSelector, ingress []networkv1.NetworkPolicyIngressRule) *networkv1.NetworkPolicy {
	policy := &networkv1.NetworkPolicy{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: namespace,
			Name:      name,
		},
		Spec: networkv1.NetworkPolicySpec{
			PodSelector: labelSelector,
			Ingress:     ingress,
		},
	}

	virtClient, err := kubecli.GetKubevirtClient()
	ExpectWithOffset(1, err).ToNot(HaveOccurred())

	By(fmt.Sprintf("Create networkpolicy %s/%s", policy.Namespace, policy.Name))
	policy, err = virtClient.NetworkingV1().NetworkPolicies(policy.Namespace).Create(context.Background(), policy, metav1.CreateOptions{})
	ExpectWithOffset(1, err).ToNot(HaveOccurred(), fmt.Sprintf("should succeed creating network policy %s/%s", policy.Namespace, policy.Name))
	return policy
}

func waitForNetworkPolicyDeletion(policy *networkv1.NetworkPolicy) {
	if policy == nil {
		return
	}

	virtClient, err := kubecli.GetKubevirtClient()
	ExpectWithOffset(1, err).ToNot(HaveOccurred())

	ExpectWithOffset(1, virtClient.NetworkingV1().NetworkPolicies(policy.Namespace).Delete(context.Background(), policy.Name, metav1.DeleteOptions{})).To(Succeed())
	EventuallyWithOffset(1, func() error {
		_, err := virtClient.NetworkingV1().NetworkPolicies(policy.Namespace).Get(context.Background(), policy.Name, metav1.GetOptions{})
		return err
	}, 10*time.Second, time.Second).Should(SatisfyAll(HaveOccurred(), WithTransform(errors.IsNotFound, BeTrue())))
}

func assertHTTPPingSucceed(fromVmi, toVmi *v1.VirtualMachineInstance, port int) {
	ConsistentlyWithOffset(1, checkHTTPPingAndStopOnFailure(fromVmi, toVmi, port), 10*time.Second, time.Second).Should(Succeed())
}

func assertHTTPPingFailed(vmiFrom, vmiTo *v1.VirtualMachineInstance, port int) {
	EventuallyWithOffset(1, checkHTTPPingAndStopOnSucceed(vmiFrom, vmiTo, port), 10*time.Second, time.Second).ShouldNot(Succeed())
	ConsistentlyWithOffset(1, checkHTTPPingAndStopOnSucceed(vmiFrom, vmiTo, port), 10*time.Second, time.Second).ShouldNot(Succeed())
}

func checkHTTPPingAndStopOnSucceed(fromVmi, toVmi *v1.VirtualMachineInstance, port int) func() error {
	return func() error {
		var err error
		for _, ip := range toVmi.Status.Interfaces[0].IPs {
			err = checkHTTPPing(fromVmi, ip, port)
			if err == nil {
				return nil
			}
		}
		return err
	}
}

func checkHTTPPingAndStopOnFailure(fromVmi, toVmi *v1.VirtualMachineInstance, port int) func() error {
	return func() error {
		for _, ip := range toVmi.Status.Interfaces[0].IPs {
			err := checkHTTPPing(fromVmi, ip, port)
			if err != nil {
				return err
			}
		}
		return nil
	}
}

func checkHTTPPing(vmi *v1.VirtualMachineInstance, ip string, port int) error {
	const curlCheckCmd = "curl --head %s --connect-timeout 5\n"
	url := fmt.Sprintf("http://%s", net.JoinHostPort(ip, strconv.Itoa(port)))
	curlCheck := fmt.Sprintf(curlCheckCmd, url)
	err := console.RunCommand(vmi, curlCheck, 10*time.Second)
	if err != nil {
		return fmt.Errorf("failed HTTP ping from vmi(%s/%s) to url(%s): %v", vmi.Namespace, vmi.Name, url, err)
	}
	return nil
}

func assertIPsNotEmptyForVMI(vmi *v1.VirtualMachineInstance) {
	ExpectWithOffset(1, vmi.Status.Interfaces[0].IPs).ToNot(BeEmpty(), "should contain a not empy list of ip addresses")
}
