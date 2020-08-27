package tests_test

import (
	"fmt"
	"time"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	v1network "k8s.io/api/networking/v1"
	v13 "k8s.io/apimachinery/pkg/apis/meta/v1"

	v1 "kubevirt.io/client-go/api/v1"
	"kubevirt.io/client-go/kubecli"
	"kubevirt.io/kubevirt/tests"
	cd "kubevirt.io/kubevirt/tests/containerdisk"
)

var _ = Describe("[rfe_id:150][crit:high][vendor:cnv-qe@redhat.com][level:component]Networkpolicy", func() {
	var (
		virtClient kubecli.KubevirtClient

		vmia *v1.VirtualMachineInstance
		vmib *v1.VirtualMachineInstance
		vmic *v1.VirtualMachineInstance

		pingEventually = func(fromVmi, toVmi *v1.VirtualMachineInstance) AsyncAssertion {
			return Eventually(func() error {
				toIp := toVmi.Status.Interfaces[0].IP
				By(fmt.Sprintf("Pinging from VMI %s/%s to VMI %s/%s(%s)", fromVmi.Namespace, fromVmi.Name, toVmi.Namespace, toVmi.Name, toIp))
				return tests.PingFromVMConsole(fromVmi, toIp)
			}, 10*time.Second, time.Second)
		}
	)

	tests.BeforeAll(func() {
		var err error

		virtClient, err = kubecli.GetKubevirtClient()
		tests.PanicOnError(err)

		tests.SkipIfUseFlannel(virtClient)
		tests.BeforeTestCleanup()
		// Create three vmis, vmia and vmib are in same namespace, vmic is in different namespace
		vmia = tests.NewRandomVMIWithEphemeralDiskAndUserdata(cd.ContainerDiskFor(cd.ContainerDiskCirros), "#!/bin/bash\necho 'hello'\n")
		vmia.Labels = map[string]string{"type": "test"}
		vmia, err = virtClient.VirtualMachineInstance(tests.NamespaceTestDefault).Create(vmia)
		Expect(err).ToNot(HaveOccurred())

		vmib = tests.NewRandomVMIWithEphemeralDiskAndUserdata(cd.ContainerDiskFor(cd.ContainerDiskCirros), "#!/bin/bash\necho 'hello'\n")
		_, err = virtClient.VirtualMachineInstance(tests.NamespaceTestDefault).Create(vmib)
		Expect(err).ToNot(HaveOccurred())

		vmic = tests.NewRandomVMIWithEphemeralDiskAndUserdata(cd.ContainerDiskFor(cd.ContainerDiskCirros), "#!/bin/bash\necho 'hello'\n")
		vmic.Namespace = tests.NamespaceTestAlternative
		_, err = virtClient.VirtualMachineInstance(tests.NamespaceTestAlternative).Create(vmic)
		Expect(err).ToNot(HaveOccurred())

		vmia = tests.WaitUntilVMIReady(vmia, tests.LoggedInCirrosExpecter)
		vmib = tests.WaitUntilVMIReady(vmib, tests.LoggedInCirrosExpecter)
		vmic = tests.WaitUntilVMIReady(vmic, tests.LoggedInCirrosExpecter)
	})

	Context("vms limited by Default-deny networkpolicy", func() {
		var ()
		BeforeEach(func() {
			// deny-by-default networkpolicy will deny all the traffice to the vms in the namespace
			By("Create deny-by-default networkpolicy")
			networkpolicy := &v1network.NetworkPolicy{
				ObjectMeta: v13.ObjectMeta{
					Name: "deny-by-default",
				},
				Spec: v1network.NetworkPolicySpec{
					PodSelector: v13.LabelSelector{},
					Ingress:     []v1network.NetworkPolicyIngressRule{},
				},
			}
			_, err := virtClient.NetworkingV1().NetworkPolicies(vmia.Namespace).Create(networkpolicy)
			Expect(err).ToNot(HaveOccurred())
		})

		It("[test_id:1511] should be failed to reach vmia from vmib", func() {
			pingEventually(vmib, vmia).ShouldNot(Succeed())
		})

		It("[test_id:1512] should be failed to reach vmib from vmia", func() {
			pingEventually(vmia, vmib).ShouldNot(Succeed())
		})

		AfterEach(func() {
			Expect(virtClient.NetworkingV1().NetworkPolicies(vmia.Namespace).Delete("deny-by-default", &v13.DeleteOptions{})).To(Succeed())
		})

	})

	Context("vms limited by allow same namespace networkpolicy", func() {
		BeforeEach(func() {
			// allow-same-namespave networkpolicy will only allow the traffice inside the namespace
			By("Create allow-same-namespace networkpolicy")
			networkpolicy := &v1network.NetworkPolicy{
				ObjectMeta: v13.ObjectMeta{
					Name: "allow-same-namespace",
				},
				Spec: v1network.NetworkPolicySpec{
					PodSelector: v13.LabelSelector{},
					Ingress: []v1network.NetworkPolicyIngressRule{
						{
							From: []v1network.NetworkPolicyPeer{
								{
									PodSelector: &v13.LabelSelector{},
								},
							},
						},
					},
				},
			}
			_, err := virtClient.NetworkingV1().NetworkPolicies(vmia.Namespace).Create(networkpolicy)
			Expect(err).ToNot(HaveOccurred())
		})

		It("[test_id:1513] should be successful to reach vmia from vmib", func() {
			pingEventually(vmib, vmia).Should(Succeed())
		})

		It("[test_id:1514] should be failed to reach vmia from vmic", func() {
			pingEventually(vmic, vmia).ShouldNot(Succeed())
		})

		AfterEach(func() {
			Expect(virtClient.NetworkingV1().NetworkPolicies(vmia.Namespace).Delete("allow-same-namespace", &v13.DeleteOptions{})).To(Succeed())
		})

	})

	Context("vms limited by deny by label networkpolicy", func() {
		BeforeEach(func() {
			// deny-by-label networkpolicy will deny the traffice for the vm which have the same label
			By("Create deny-by-label networkpolicy")
			networkpolicy := &v1network.NetworkPolicy{
				ObjectMeta: v13.ObjectMeta{
					Name: "deny-by-label",
				},
				Spec: v1network.NetworkPolicySpec{
					PodSelector: v13.LabelSelector{
						MatchLabels: map[string]string{
							"type": "test",
						},
					},
					Ingress: []v1network.NetworkPolicyIngressRule{},
				},
			}
			_, err := virtClient.NetworkingV1().NetworkPolicies(vmia.Namespace).Create(networkpolicy)
			Expect(err).ToNot(HaveOccurred())
		})

		It("[test_id:1515] should be failed to reach vmia from vmic", func() {
			pingEventually(vmic, vmia).ShouldNot(Succeed())
		})

		It("[test_id:1516] should be failed to reach vmia from vmib", func() {
			pingEventually(vmib, vmia).ShouldNot(Succeed())
		})

		It("[test_id:1517] should be successful to reach vmib from vmic", func() {
			pingEventually(vmic, vmib).Should(Succeed())
		})

		AfterEach(func() {
			Expect(virtClient.NetworkingV1().NetworkPolicies(vmia.Namespace).Delete("deny-by-label", &v13.DeleteOptions{})).To(Succeed())
		})

	})

})
