package network

import (
	"context"
	"fmt"
	"net"
	"strconv"
	"time"

	"kubevirt.io/kubevirt/tests/framework/kubevirt"

	"kubevirt.io/kubevirt/tests/framework/checks"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	corev1 "k8s.io/api/core/v1"
	networkv1 "k8s.io/api/networking/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"

	v1 "kubevirt.io/api/core/v1"
	"kubevirt.io/client-go/kubecli"

	"kubevirt.io/kubevirt/tests/console"
	"kubevirt.io/kubevirt/tests/decorators"
	"kubevirt.io/kubevirt/tests/libnet"
	"kubevirt.io/kubevirt/tests/libnet/vmnetserver"
	"kubevirt.io/kubevirt/tests/libvmifact"
	"kubevirt.io/kubevirt/tests/libwait"
	"kubevirt.io/kubevirt/tests/testsuite"
)

var _ = Describe(SIG("[rfe_id:150][crit:high][vendor:cnv-qe@redhat.com][level:component]Networkpolicy", func() {
	var (
		virtClient      kubecli.KubevirtClient
		serverVMILabels map[string]string
	)
	BeforeEach(func() {
		virtClient = kubevirt.Client()
		if checks.IsRunningOnKindInfra() {
			Fail("Network Policy tests cannot run till issue https://github.com/kubevirt/kubevirt/issues/4081 is fixed")
		}

		serverVMILabels = map[string]string{"type": "test"}
	})

	Context("when three alpine VMs with default networking are started and serverVMI start an HTTP server on port 80 and 81", func() {
		var serverVMI, clientVMI *v1.VirtualMachineInstance

		BeforeEach(func() {
			var err error
			serverVMI, err = createServerVmi(virtClient, testsuite.NamespaceTestDefault, serverVMILabels)
			Expect(err).ToNot(HaveOccurred())
			assertIPsNotEmptyForVMI(serverVMI)
		})

		Context("and connectivity between VMI/s is blocked by Default-deny networkpolicy", func() {
			var policy *networkv1.NetworkPolicy

			BeforeEach(func() {
				var err error
				// deny-by-default networkpolicy will deny all the traffic to the vms in the namespace
				policy = createNetworkPolicy(serverVMI.Namespace, "deny-by-default", metav1.LabelSelector{}, []networkv1.NetworkPolicyIngressRule{})
				clientVMI, err = createClientVmi(testsuite.NamespaceTestDefault, virtClient)
				Expect(err).ToNot(HaveOccurred())
				assertIPsNotEmptyForVMI(clientVMI)
			})

			AfterEach(func() {
				waitForNetworkPolicyDeletion(policy)
			})

			It("[test_id:1511] should fail to reach serverVMI from clientVMI", func() {
				By("Connect serverVMI from clientVMI")
				assertPingFailToPrimaryIP(clientVMI, serverVMI)
			})

			It("[test_id:1512] should fail to reach clientVMI from serverVMI", func() {
				By("Connect clientVMI from serverVMI")
				assertPingFailToPrimaryIP(serverVMI, clientVMI)
			})
			It("[test_id:369] should deny http traffic for ports 80/81 from clientVMI to serverVMI", func() {
				assertHTTPPingFailedToPrimaryIP(clientVMI, serverVMI, 80)
				assertHTTPPingFailedToPrimaryIP(clientVMI, serverVMI, 81)
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

			When("client vmi is on default namespace", func() {
				BeforeEach(func() {
					var err error
					clientVMI, err = createClientVmi(testsuite.NamespaceTestDefault, virtClient)
					Expect(err).ToNot(HaveOccurred())
					assertIPsNotEmptyForVMI(clientVMI)
				})

				It("[test_id:1513] should succeed pinging between two VMI/s in the same namespace", decorators.Conformance, func() {
					assertPingSucceedToPrimaryIP(clientVMI, serverVMI)
				})
			})

			When("client vmi is on alternative namespace", func() {
				var clientVMIAlternativeNamespace *v1.VirtualMachineInstance

				BeforeEach(func() {
					var err error
					clientVMIAlternativeNamespace, err = createClientVmi(testsuite.NamespaceTestAlternative, virtClient)
					Expect(err).ToNot(HaveOccurred())
					assertIPsNotEmptyForVMI(clientVMIAlternativeNamespace)
				})

				It("[test_id:1514] should fail pinging between two VMI/s each on different namespaces", decorators.Conformance, func() {
					assertPingFailToPrimaryIP(clientVMIAlternativeNamespace, serverVMI)
				})
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

			When("client vmi is on alternative namespace", func() {
				var clientVMIAlternativeNamespace *v1.VirtualMachineInstance

				BeforeEach(func() {
					var err error
					clientVMIAlternativeNamespace, err = createClientVmi(testsuite.NamespaceTestAlternative, virtClient)
					Expect(err).ToNot(HaveOccurred())
					assertIPsNotEmptyForVMI(clientVMIAlternativeNamespace)
				})

				It("[test_id:1515] should fail to reach serverVMI from clientVMIAlternativeNamespace", func() {
					By("Connect serverVMI from clientVMIAlternativeNamespace")
					assertPingFailToPrimaryIP(clientVMIAlternativeNamespace, serverVMI)
				})
			})

			When("client vmi is on default namespace", func() {
				BeforeEach(func() {
					var err error
					clientVMI, err = createClientVmi(testsuite.NamespaceTestDefault, virtClient)
					Expect(err).ToNot(HaveOccurred())
					assertIPsNotEmptyForVMI(clientVMI)
				})

				It("[test_id:1515] should fail to reach serverVMI from clientVMI", func() {
					By("Connect serverVMI from clientVMIAlternativeNamespace")
					assertPingFailToPrimaryIP(clientVMI, serverVMI)
				})

				When("another client vmi is on an alternative namespace", func() {
					var clientVMIAlternativeNamespace *v1.VirtualMachineInstance

					BeforeEach(func() {
						var err error
						clientVMIAlternativeNamespace, err = createClientVmi(testsuite.NamespaceTestAlternative, virtClient)
						Expect(err).ToNot(HaveOccurred())
						assertIPsNotEmptyForVMI(clientVMIAlternativeNamespace)
					})

					It("[test_id:1517] should success to reach clientVMI from clientVMIAlternativeNamespace", func() {
						By("Connect clientVMI from clientVMIAlternativeNamespace")
						assertPingSucceedToPrimaryIP(clientVMIAlternativeNamespace, clientVMI)
					})
				})
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

				var err error
				clientVMI, err = createClientVmi(testsuite.NamespaceTestDefault, virtClient)
				Expect(err).ToNot(HaveOccurred())
				assertIPsNotEmptyForVMI(clientVMI)
			})
			AfterEach(func() {
				waitForNetworkPolicyDeletion(policy)
			})
			It("[test_id:2774] should allow http traffic for ports 80 and 81 from clientVMI to serverVMI", func() {
				assertHTTPPingSucceedToPrimaryIP(clientVMI, serverVMI, 80)
				assertHTTPPingSucceedToPrimaryIP(clientVMI, serverVMI, 81)
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

				var err error
				clientVMI, err = createClientVmi(testsuite.NamespaceTestDefault, virtClient)
				Expect(err).ToNot(HaveOccurred())
				assertIPsNotEmptyForVMI(clientVMI)
			})
			AfterEach(func() {
				waitForNetworkPolicyDeletion(policy)
			})
			It("[test_id:2775] should allow http traffic at port 80 and deny at port 81 from clientVMI to serverVMI", func() {
				assertHTTPPingSucceedToPrimaryIP(clientVMI, serverVMI, 80)
				assertHTTPPingFailedToPrimaryIP(clientVMI, serverVMI, 81)
			})
		})
	})
}))

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

// primaryIPForConnectivityCheck returns the IP to use when asserting connectivity is blocked.
// Prefer IPv4 when present (covers IPv4-only and dual-stack where policy typically applies to IPv4),
// then IPv6, then the first reported IP.
func primaryIPForConnectivityCheck(vmi *v1.VirtualMachineInstance) string {
	if vmi == nil {
		Fail("primaryIPForConnectivityCheck: target VMI is nil; cannot determine IP for connectivity check")
		return ""
	}

	if len(vmi.Status.Interfaces) == 0 {
		Fail(fmt.Sprintf(
			"primaryIPForConnectivityCheck: VMI %s has no network interfaces in status; cannot determine IP for connectivity check",
			vmi.Name,
		))
		return ""
	}

	if ip := libnet.GetVmiPrimaryIPByFamily(vmi, corev1.IPv4Protocol); ip != "" {
		return ip
	}

	if ip := libnet.GetVmiPrimaryIPByFamily(vmi, corev1.IPv6Protocol); ip != "" {
		return ip
	}

	iface := vmi.Status.Interfaces[0]
	Fail(fmt.Sprintf(
		"primaryIPForConnectivityCheck: VMI %s interface %q has no IPs reported; cannot determine IP for connectivity check",
		vmi.Name,
		iface.Name,
	))

	return ""
}

// assertPingFailToPrimaryIP asserts that ping from fromVmi to toVmi's primary IP fails (default-deny).
// Works on IPv4-only, IPv6-only, or dual-stack clusters: we assert the primary address (IPv4 if
// present, else first IP) is unreachable, so the test passes regardless of cluster IP family.
func assertPingFailToPrimaryIP(fromVmi, toVmi *v1.VirtualMachineInstance) {
	toIP := primaryIPForConnectivityCheck(toVmi)
	EventuallyWithOffset(1, func() error {
		if err := libnet.PingFromVMConsole(fromVmi, toIP); err == nil {
			return nil
		}
		return fmt.Errorf("ping started to fail as expected")
	}, 15*time.Second, time.Second).ShouldNot(Succeed())

	ConsistentlyWithOffset(1, func() error {
		if err := libnet.PingFromVMConsole(fromVmi, toIP); err == nil {
			return nil
		}
		return fmt.Errorf("ping kept failing as expected")
	}, 5*time.Second, 1*time.Second).ShouldNot(Succeed())
}

// assertPingSucceedToPrimaryIP asserts that ping from fromVmi to toVmi's primary IP succeeds (traffic allowed).
// Uses the same primary-IP selection as assertPingFailToPrimaryIP for consistency across IPv4-only, IPv6-only, and dual-stack.
func assertPingSucceedToPrimaryIP(fromVmi, toVmi *v1.VirtualMachineInstance) {
	toIP := primaryIPForConnectivityCheck(toVmi)
	ExpectWithOffset(1, libnet.PingFromVMConsole(fromVmi, toIP)).To(Succeed())
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

	virtClient := kubevirt.Client()

	By(fmt.Sprintf("Create networkpolicy %s/%s", policy.Namespace, policy.Name))
	var err error
	policy, err = virtClient.NetworkingV1().NetworkPolicies(policy.Namespace).Create(context.Background(), policy, metav1.CreateOptions{})
	ExpectWithOffset(1, err).ToNot(HaveOccurred(), fmt.Sprintf("should succeed creating network policy %s/%s", policy.Namespace, policy.Name))
	return policy
}

func waitForNetworkPolicyDeletion(policy *networkv1.NetworkPolicy) {
	if policy == nil {
		return
	}

	virtClient := kubevirt.Client()

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

// assertHTTPPingSucceedToPrimaryIP asserts HTTP reachability to toVmi's primary IP only (IPv4 if present).
// Use when the server or policy may only listen/apply to one family (e.g. IPv4-only cluster).
func assertHTTPPingSucceedToPrimaryIP(fromVmi, toVmi *v1.VirtualMachineInstance, port int) {
	toIP := primaryIPForConnectivityCheck(toVmi)
	ConsistentlyWithOffset(1, func() error {
		return checkHTTPPing(fromVmi, toIP, port)
	}, 10*time.Second, time.Second).Should(Succeed())
}

// assertHTTPPingFailedToPrimaryIP asserts HTTP is not reachable at toVmi's primary IP only.
func assertHTTPPingFailedToPrimaryIP(fromVmi, toVmi *v1.VirtualMachineInstance, port int) {
	toIP := primaryIPForConnectivityCheck(toVmi)
	EventuallyWithOffset(1, func() error {
		if err := checkHTTPPing(fromVmi, toIP, port); err == nil {
			return nil
		}
		return fmt.Errorf("http ping started to fail as expected")
	}, 10*time.Second, time.Second).ShouldNot(Succeed())
	ConsistentlyWithOffset(1, func() error {
		if err := checkHTTPPing(fromVmi, toIP, port); err == nil {
			return nil
		}
		return fmt.Errorf("http ping kept failing as expected")
	}, 10*time.Second, time.Second).ShouldNot(Succeed())
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
	const wgetCheckCmd = "wget -S --spider %s -T 5\n"
	url := fmt.Sprintf("http://%s", net.JoinHostPort(ip, strconv.Itoa(port)))
	wgetCheck := fmt.Sprintf(wgetCheckCmd, url)
	err := console.RunCommand(vmi, wgetCheck, 10*time.Second)
	if err != nil {
		return fmt.Errorf("failed HTTP ping from vmi(%s/%s) to url(%s): %v", vmi.Namespace, vmi.Name, url, err)
	}
	return nil
}

func assertIPsNotEmptyForVMI(vmi *v1.VirtualMachineInstance) {
	ExpectWithOffset(1, vmi.Status.Interfaces[0].IPs).ToNot(BeEmpty(), "should contain a not empty list of ip addresses")
}

func createClientVmi(namespace string, virtClient kubecli.KubevirtClient) (*v1.VirtualMachineInstance, error) {
	clientVMI := libvmifact.NewAlpineWithTestTooling(libnet.WithMasqueradeNetworking())
	var err error
	clientVMI, err = virtClient.VirtualMachineInstance(namespace).Create(context.Background(), clientVMI, metav1.CreateOptions{})
	if err != nil {
		return nil, err
	}

	clientVMI = libwait.WaitUntilVMIReady(clientVMI, console.LoginToAlpine)
	return clientVMI, nil
}

func createServerVmi(virtClient kubecli.KubevirtClient, namespace string, serverVMILabels map[string]string) (*v1.VirtualMachineInstance, error) {
	serverVMI := libvmifact.NewAlpineWithTestTooling(
		libnet.WithMasqueradeNetworking(
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
		),
	)
	serverVMI.Labels = serverVMILabels
	serverVMI, err := virtClient.VirtualMachineInstance(namespace).Create(context.Background(), serverVMI, metav1.CreateOptions{})
	if err != nil {
		return nil, err
	}
	serverVMI = libwait.WaitUntilVMIReady(serverVMI, console.LoginToAlpine)

	By("Start HTTP server at serverVMI on ports 80 and 81")
	vmnetserver.HTTPServer.Start(serverVMI, 80)
	vmnetserver.HTTPServer.Start(serverVMI, 81)

	return serverVMI, nil
}
