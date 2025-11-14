/*
 * This file is part of the kubevirt project
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 *
 * Copyright The KubeVirt Authors.
 *
 */

package network

import (
	"context"
	"fmt"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	expect "github.com/google/goexpect"
	k8sv1 "k8s.io/api/core/v1"
	k8smetav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	v1 "kubevirt.io/api/core/v1"
	"kubevirt.io/client-go/kubecli"

	"kubevirt.io/kubevirt/pkg/libvmi"

	"kubevirt.io/kubevirt/tests/console"
	"kubevirt.io/kubevirt/tests/framework/k8s"
	"kubevirt.io/kubevirt/tests/framework/kubevirt"
	"kubevirt.io/kubevirt/tests/libnet"
	"kubevirt.io/kubevirt/tests/libnet/dns"
	netservice "kubevirt.io/kubevirt/tests/libnet/service"
	"kubevirt.io/kubevirt/tests/libvmifact"
	"kubevirt.io/kubevirt/tests/libwait"
	"kubevirt.io/kubevirt/tests/testsuite"
)

var _ = Describe(SIG("Subdomain", func() {
	var virtClient kubecli.KubevirtClient

	const (
		subdomain          = "testsubdomain"
		selectorLabelKey   = "expose"
		selectorLabelValue = "this"
		hostname           = "testhostname"
	)

	BeforeEach(func() {
		virtClient = kubevirt.Client()

		// Should be skipped as long as masquerade binding doesn't have dhcpv6 + ra (issue- https://github.com/kubevirt/kubevirt/issues/7184)
		libnet.SkipWhenClusterNotSupportIpv4()
	})

	Context("with a headless service given", func() {
		const servicePort = 22

		BeforeEach(func() {
			serviceName := subdomain
			service := netservice.BuildHeadlessSpec(serviceName, servicePort, servicePort, selectorLabelKey, selectorLabelValue)
			_, err := k8s.Client().CoreV1().Services(testsuite.NamespaceTestDefault).Create(context.Background(), service, k8smetav1.CreateOptions{})
			Expect(err).ToNot(HaveOccurred())
		})

		DescribeTable("VMI should have the expected FQDN", func(f func() *v1.VirtualMachineInstance, subdom, hostname string) {
			vmiSpec := f()
			var expectedFQDN, domain string
			if subdom != "" {
				vmiSpec.Spec.Subdomain = subdom
				if hostname != "" {
					domain = hostname
					vmiSpec.Spec.Hostname = domain
				} else {
					domain = vmiSpec.Name
				}
				expectedFQDN = fmt.Sprintf("%s.%s.%s.svc.cluster.local", domain, subdom, testsuite.NamespaceTestDefault)
			} else {
				expectedFQDN = vmiSpec.Name
			}
			vmiSpec.Labels = map[string]string{selectorLabelKey: selectorLabelValue}

			vmi, err := virtClient.VirtualMachineInstance(testsuite.NamespaceTestDefault).Create(context.Background(), vmiSpec, metav1.CreateOptions{})
			Expect(err).ToNot(HaveOccurred())
			vmi = libwait.WaitUntilVMIReady(vmi, console.LoginToFedora)

			Expect(assertFQDNinGuest(vmi, expectedFQDN)).To(Succeed(), "failed to get expected FQDN")
		},
			Entry("with Masquerade binding and subdomain and hostname", fedoraMasqueradeVMI, subdomain, hostname),
			Entry("with Bridge binding and subdomain", fedoraBridgeBindingVMI, subdomain, ""),
			Entry("with Masquerade binding without subdomain", fedoraMasqueradeVMI, "", ""),
			Entry("with Bridge binding without subdomain", fedoraBridgeBindingVMI, "", ""),
		)

		It("VMI with custom DNSPolicy should have the expected FQDN", func() {
			vmiSpec := fedoraBridgeBindingVMI()
			vmiSpec.Spec.Subdomain = subdomain
			expectedFQDN := fmt.Sprintf("%s.%s.%s.svc.cluster.local", vmiSpec.Name, subdomain, testsuite.NamespaceTestDefault)
			vmiSpec.Labels = map[string]string{selectorLabelKey: selectorLabelValue}

			dnsServerIP, err := dns.ClusterDNSServiceIP()
			Expect(err).ToNot(HaveOccurred())

			vmiSpec.Spec.DNSPolicy = "None"
			vmiSpec.Spec.DNSConfig = &k8sv1.PodDNSConfig{
				Nameservers: []string{dnsServerIP},
				Searches: []string{
					testsuite.NamespaceTestDefault + ".svc.cluster.local",
					"svc.cluster.local", "cluster.local", testsuite.NamespaceTestDefault + ".this.is.just.a.very.long.dummy",
				},
			}

			vmi, err := virtClient.VirtualMachineInstance(testsuite.NamespaceTestDefault).Create(context.Background(), vmiSpec, metav1.CreateOptions{})
			Expect(err).ToNot(HaveOccurred())
			vmi = libwait.WaitUntilVMIReady(vmi, console.LoginToFedora)

			Expect(assertFQDNinGuest(vmi, expectedFQDN)).To(Succeed(), "failed to get expected FQDN")
		})
	})

	It("VMI with custom DNSPolicy, a subdomain and no service entry, should not include the subdomain in the searchlist", func() {
		vmiSpec := fedoraBridgeBindingVMI()
		vmiSpec.Spec.Subdomain = subdomain
		expectedFQDN := fmt.Sprintf("%s.%s.%s.svc.cluster.local", vmiSpec.Name, subdomain, testsuite.NamespaceTestDefault)
		vmiSpec.Labels = map[string]string{selectorLabelKey: selectorLabelValue}

		dnsServerIP, err := dns.ClusterDNSServiceIP()
		Expect(err).ToNot(HaveOccurred())

		vmiSpec.Spec.DNSPolicy = "None"
		vmiSpec.Spec.DNSConfig = &k8sv1.PodDNSConfig{
			Nameservers: []string{dnsServerIP},
			Searches:    []string{"example.com"},
		}

		vmi, err := virtClient.VirtualMachineInstance(testsuite.NamespaceTestDefault).Create(context.Background(), vmiSpec, metav1.CreateOptions{})
		Expect(err).ToNot(HaveOccurred())
		vmi = libwait.WaitUntilVMIReady(vmi, console.LoginToFedora)

		Expect(assertFQDNinGuest(vmi, expectedFQDN)).To(Not(Succeed()), "found unexpected FQDN")
		Expect(assertSearchEntriesinGuest(vmi, "search example.com")).To(Succeed(), "failed to get expected search entries")
	})
}))

func fedoraMasqueradeVMI() *v1.VirtualMachineInstance {
	return libvmifact.NewFedora(
		libvmi.WithInterface(libvmi.InterfaceDeviceWithMasqueradeBinding()),
		libvmi.WithNetwork(v1.DefaultPodNetwork()))
}

func fedoraBridgeBindingVMI() *v1.VirtualMachineInstance {
	return libvmifact.NewFedora(
		libvmi.WithInterface(libvmi.InterfaceDeviceWithBridgeBinding(v1.DefaultPodNetwork().Name)),
		libvmi.WithNetwork(v1.DefaultPodNetwork()))
}

func assertFQDNinGuest(vmi *v1.VirtualMachineInstance, expectedFQDN string) error {
	return console.SafeExpectBatch(vmi, []expect.Batcher{
		&expect.BSnd{S: "\n"},
		&expect.BExp{R: ""},
		&expect.BSnd{S: "hostname -f\n"},
		&expect.BExp{R: expectedFQDN},
	}, 10)
}

func assertSearchEntriesinGuest(vmi *v1.VirtualMachineInstance, expectedSearch string) error {
	return console.SafeExpectBatch(vmi, []expect.Batcher{
		&expect.BSnd{S: "\n"},
		&expect.BExp{R: ""},
		&expect.BSnd{S: "cat /etc/resolv.conf\n"},
		&expect.BExp{R: expectedSearch + console.CRLF},
	}, 20)
}
