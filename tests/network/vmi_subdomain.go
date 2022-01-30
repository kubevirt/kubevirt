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
 * Copyright 2021 Red Hat, Inc.
 *
 */

package network

import (
	"context"
	"fmt"

	. "github.com/onsi/ginkgo"
	"github.com/onsi/ginkgo/extensions/table"
	. "github.com/onsi/gomega"

	expect "github.com/google/goexpect"
	k8sv1 "k8s.io/api/core/v1"
	k8smetav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	v1 "kubevirt.io/api/core/v1"
	"kubevirt.io/client-go/kubecli"
	"kubevirt.io/kubevirt/tests"
	"kubevirt.io/kubevirt/tests/console"
	"kubevirt.io/kubevirt/tests/libnet"
	netservice "kubevirt.io/kubevirt/tests/libnet/service"
	"kubevirt.io/kubevirt/tests/libvmi"
	"kubevirt.io/kubevirt/tests/util"
)

var _ = SIGDescribe("Subdomain", func() {
	var virtClient kubecli.KubevirtClient

	const (
		subdomain          = "testsubdomain"
		selectorLabelKey   = "expose"
		selectorLabelValue = "this"
	)

	BeforeEach(func() {
		var err error
		virtClient, err = kubecli.GetKubevirtClient()
		Expect(err).NotTo(HaveOccurred(), "Should successfully initialize an API client")

		tests.BeforeTestCleanup()
	})

	Context("with a headless service given", func() {
		const servicePort = 22

		BeforeEach(func() {
			serviceName := subdomain
			service := netservice.BuildHeadlessSpec(serviceName, servicePort, servicePort, selectorLabelKey, selectorLabelValue)
			_, err := virtClient.CoreV1().Services(util.NamespaceTestDefault).Create(context.Background(), service, k8smetav1.CreateOptions{})
			Expect(err).ToNot(HaveOccurred())
		})

		table.DescribeTable("VMI should have the expected FQDN", func(f func() *v1.VirtualMachineInstance, subdom string) {
			vmiSpec := f()
			var expectedFQDN string
			if subdom != "" {
				vmiSpec.Spec.Subdomain = subdom
				expectedFQDN = fmt.Sprintf("%s.%s.%s.svc.cluster.local", vmiSpec.Name, subdom, util.NamespaceTestDefault)
			} else {
				expectedFQDN = vmiSpec.Name
			}
			vmiSpec.Labels = map[string]string{selectorLabelKey: selectorLabelValue}

			vmi, err := virtClient.VirtualMachineInstance(util.NamespaceTestDefault).Create(vmiSpec)
			Expect(err).ToNot(HaveOccurred())
			vmi = tests.WaitUntilVMIReady(vmi, console.LoginToFedora)

			Expect(assertFQDNinGuest(vmi, expectedFQDN)).To(Succeed(), "failed to get expected FQDN")
		},
			table.Entry("with Masquerade binding and subdomain", fedoraMasqueradeVMI, subdomain),
			table.Entry("with Bridge binding and subdomain", fedoraBridgeBindingVMI, subdomain),
			table.Entry("with Masquerade binding without subdomain", fedoraMasqueradeVMI, ""),
			table.Entry("with Bridge binding without subdomain", fedoraBridgeBindingVMI, ""),
		)

		It("VMI with custom DNSPolicy should have the expected FQDN", func() {
			vmiSpec := fedoraBridgeBindingVMI()
			vmiSpec.Spec.Subdomain = subdomain
			expectedFQDN := fmt.Sprintf("%s.%s.%s.svc.cluster.local", vmiSpec.Name, subdomain, util.NamespaceTestDefault)
			vmiSpec.Labels = map[string]string{selectorLabelKey: selectorLabelValue}

			dnsServerIP, err := libnet.ClusterDNSServiceIP()
			Expect(err).ToNot(HaveOccurred())

			vmiSpec.Spec.DNSPolicy = "None"
			vmiSpec.Spec.DNSConfig = &k8sv1.PodDNSConfig{
				Nameservers: []string{dnsServerIP},
				Searches: []string{util.NamespaceTestDefault + ".svc.cluster.local",
					"svc.cluster.local", "cluster.local", util.NamespaceTestDefault + ".this.is.just.a.very.long.dummy"},
			}

			vmi, err := virtClient.VirtualMachineInstance(util.NamespaceTestDefault).Create(vmiSpec)
			Expect(err).ToNot(HaveOccurred())
			vmi = tests.WaitUntilVMIReady(vmi, console.LoginToFedora)

			Expect(assertFQDNinGuest(vmi, expectedFQDN)).To(Succeed(), "failed to get expected FQDN")
		})
	})

	It("VMI with custom DNSPolicy, a subdomain and no service entry, should not include the subdomain in the searchlist", func() {
		vmiSpec := fedoraBridgeBindingVMI()
		vmiSpec.Spec.Subdomain = subdomain
		expectedFQDN := fmt.Sprintf("%s.%s.%s.svc.cluster.local", vmiSpec.Name, subdomain, util.NamespaceTestDefault)
		vmiSpec.Labels = map[string]string{selectorLabelKey: selectorLabelValue}

		dnsServerIP, err := libnet.ClusterDNSServiceIP()
		Expect(err).ToNot(HaveOccurred())

		vmiSpec.Spec.DNSPolicy = "None"
		vmiSpec.Spec.DNSConfig = &k8sv1.PodDNSConfig{
			Nameservers: []string{dnsServerIP},
			Searches:    []string{"example.com"},
		}

		vmi, err := virtClient.VirtualMachineInstance(util.NamespaceTestDefault).Create(vmiSpec)
		Expect(err).ToNot(HaveOccurred())
		vmi = tests.WaitUntilVMIReady(vmi, console.LoginToFedora)

		Expect(assertFQDNinGuest(vmi, expectedFQDN)).To(Not(Succeed()), "found unexpected FQDN")
		Expect(assertSearchEntriesinGuest(vmi, "search example.com")).To(Succeed(), "failed to get expected search entries")
	})
})

func fedoraMasqueradeVMI() *v1.VirtualMachineInstance {
	return libvmi.NewFedora(
		libvmi.WithInterface(libvmi.InterfaceDeviceWithMasqueradeBinding()),
		libvmi.WithNetwork(v1.DefaultPodNetwork()))
}

func fedoraBridgeBindingVMI() *v1.VirtualMachineInstance {
	return libvmi.NewFedora(
		libvmi.WithInterface(libvmi.InterfaceDeviceWithBridgeBinding(libvmi.DefaultInterfaceName)),
		libvmi.WithNetwork(v1.DefaultPodNetwork()))
}

func assertFQDNinGuest(vmi *v1.VirtualMachineInstance, expectedFQDN string) error {
	return console.SafeExpectBatch(vmi, []expect.Batcher{
		&expect.BSnd{S: "\n"},
		&expect.BExp{R: console.PromptExpression},
		&expect.BSnd{S: "hostname -f\n"},
		&expect.BExp{R: expectedFQDN},
	}, 10)
}

func assertSearchEntriesinGuest(vmi *v1.VirtualMachineInstance, expectedSearch string) error {
	return console.SafeExpectBatch(vmi, []expect.Batcher{
		&expect.BSnd{S: "\n"},
		&expect.BExp{R: console.PromptExpression},
		&expect.BSnd{S: "cat /etc/resolv.conf\n"},
		&expect.BExp{R: expectedSearch + console.CRLF},
	}, 20)
}
