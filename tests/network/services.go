/*
 * This file is part of the KubeVirt project
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
	"strconv"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	batchv1 "k8s.io/api/batch/v1"
	k8sv1 "k8s.io/api/core/v1"
	k8smetav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	v1 "kubevirt.io/api/core/v1"

	"kubevirt.io/kubevirt/pkg/libvmi"

	"kubevirt.io/kubevirt/tests/console"
	"kubevirt.io/kubevirt/tests/decorators"
	"kubevirt.io/kubevirt/tests/framework/kubevirt"
	"kubevirt.io/kubevirt/tests/libnet"
	"kubevirt.io/kubevirt/tests/libnet/job"
	netservice "kubevirt.io/kubevirt/tests/libnet/service"
	"kubevirt.io/kubevirt/tests/libnet/vmnetserver"
	"kubevirt.io/kubevirt/tests/libvmifact"
	"kubevirt.io/kubevirt/tests/libwait"
	"kubevirt.io/kubevirt/tests/testsuite"
)

const (
	expectConnectivityToExposedService = "connectivity is expected to the exposed service"

	jobSuccessRetry = 3
	jobFailureRetry = 0
)

var _ = Describe(SIG("Services", func() {
	Context("bridge interface binding", func() {
		var inboundVMI *v1.VirtualMachineInstance

		const (
			selectorLabelKey   = "expose"
			selectorLabelValue = "me"
			servicePort        = 1500
		)

		BeforeEach(func() {
			libnet.SkipWhenClusterNotSupportIpv4()

			inboundVMI = libvmifact.NewCirros(
				libvmi.WithInterface(libvmi.InterfaceDeviceWithBridgeBinding(v1.DefaultPodNetwork().Name)),
				libvmi.WithNetwork(v1.DefaultPodNetwork()),
				libvmi.WithLabel(selectorLabelKey, selectorLabelValue),
				libvmi.WithSubdomain("vmi"),
				libvmi.WithHostname("inbound"),
			)
			var err error
			inboundVMI, err = kubevirt.Client().VirtualMachineInstance(testsuite.NamespaceTestDefault).Create(context.Background(), inboundVMI, metav1.CreateOptions{})
			Expect(err).ToNot(HaveOccurred())

			inboundVMI = libwait.WaitUntilVMIReady(inboundVMI, console.LoginToCirros)
			vmnetserver.StartTCPServer(inboundVMI, servicePort, console.LoginToCirros)
		})

		Context("with a service matching the vmi exposed", func() {
			const serviceName = "myservice"

			BeforeEach(func() {
				virtClient := kubevirt.Client()
				service := netservice.BuildSpec(serviceName, servicePort, servicePort, selectorLabelKey, selectorLabelValue)
				var err error
				service, err = virtClient.CoreV1().Services(inboundVMI.Namespace).Create(context.Background(), service, k8smetav1.CreateOptions{})
				Expect(err).ToNot(HaveOccurred())
			})

			It("[test_id:1547] should be able to reach the vmi based on labels specified on the vmi", func() {
				tcpJob, err := createServiceConnectivityJob(serviceName, inboundVMI.Namespace, servicePort, jobSuccessRetry)
				Expect(err).NotTo(HaveOccurred())

				Expect(job.WaitForJobToSucceed(tcpJob, 90*time.Second)).To(Succeed(), expectConnectivityToExposedService)
			})

			It("[test_id:1548] should fail to reach the vmi if an invalid servicename is used", func() {
				tcpJob, err := createServiceConnectivityJob("wrongservice", inboundVMI.Namespace, servicePort, jobFailureRetry)
				Expect(err).NotTo(HaveOccurred())

				err = job.WaitForJobToFail(tcpJob, 90*time.Second)
				Expect(err).NotTo(HaveOccurred(), "connectivity is *not* expected, since there isn't an exposed service")
			})
		})

		Context("with a subdomain and a headless service given", func() {
			BeforeEach(func() {
				namespace, name := inboundVMI.Namespace, inboundVMI.Spec.Subdomain
				service := netservice.BuildHeadlessSpec(name, servicePort, servicePort, selectorLabelKey, selectorLabelValue)
				var err error
				virtClient := kubevirt.Client()
				service, err = virtClient.CoreV1().Services(namespace).Create(context.Background(), service, k8smetav1.CreateOptions{})
				Expect(err).ToNot(HaveOccurred())
			})

			It("[test_id:1549]should be able to reach the vmi via its unique fully qualified domain name", func() {
				var err error
				serviceHostnameWithSubdomain := fmt.Sprintf("%s.%s", inboundVMI.Spec.Hostname, inboundVMI.Spec.Subdomain)

				tcpJob, err := createServiceConnectivityJob(serviceHostnameWithSubdomain, inboundVMI.Namespace, servicePort, jobSuccessRetry)
				Expect(err).NotTo(HaveOccurred())

				Expect(job.WaitForJobToSucceed(tcpJob, 90*time.Second)).To(Succeed(), expectConnectivityToExposedService)
			})
		})
	})

	Context("Masquerade interface binding", func() {
		var inboundVMI *v1.VirtualMachineInstance

		const (
			selectorLabelKey   = "expose"
			selectorLabelValue = "me"
			servicePort        = 1500
		)

		BeforeEach(func() {
			inboundVMI = libvmifact.NewFedora(
				libnet.WithMasqueradeNetworking(),
				libvmi.WithLabel(selectorLabelKey, selectorLabelValue),
				libvmi.WithSubdomain("vmi"),
				libvmi.WithHostname("inbound"),
			)
			var err error
			inboundVMI, err = kubevirt.Client().VirtualMachineInstance(testsuite.NamespaceTestDefault).Create(context.Background(), inboundVMI, metav1.CreateOptions{})
			Expect(err).ToNot(HaveOccurred())

			inboundVMI = libwait.WaitUntilVMIReady(inboundVMI, console.LoginToFedora)
			vmnetserver.StartTCPServer(inboundVMI, servicePort, console.LoginToFedora)
		})

		Context("with a service matching the vmi exposed", func() {
			DescribeTable("should be able to reach the vmi based on labels specified on the vmi", decorators.Conformance, func(ipFamily k8sv1.IPFamily) {
				var service *k8sv1.Service
				serviceName := "myservice"

				libnet.SkipWhenClusterNotSupportIPFamily(ipFamily)

				By("setting up resources to expose the VMI via a service")
				if ipFamily == k8sv1.IPv6Protocol {
					serviceName += "v6"
					service = netservice.BuildIPv6Spec(serviceName, servicePort, servicePort, selectorLabelKey, selectorLabelValue)
				} else {
					service = netservice.BuildSpec(serviceName, servicePort, servicePort, selectorLabelKey, selectorLabelValue)
				}

				virtClient := kubevirt.Client()
				var err error
				service, err = virtClient.CoreV1().Services(inboundVMI.Namespace).Create(context.Background(), service, k8smetav1.CreateOptions{})
				Expect(err).NotTo(HaveOccurred(), "the k8sv1.Service entity should have been created.")

				By("checking connectivity the exposed service")
				tcpJob, err := createServiceConnectivityJob(serviceName, inboundVMI.Namespace, servicePort, jobSuccessRetry)
				Expect(err).NotTo(HaveOccurred())

				Expect(job.WaitForJobToSucceed(tcpJob, 90*time.Second)).To(Succeed(), expectConnectivityToExposedService)
			},
				Entry("when the service is exposed by an IPv4 address.", k8sv1.IPv4Protocol),
				Entry("when the service is exposed by an IPv6 address.", k8sv1.IPv6Protocol),
			)
		})

		Context("*without* a service matching the vmi exposed", func() {
			It("should fail to reach the vmi", func() {
				tcpJob, err := createServiceConnectivityJob("missingservice", inboundVMI.Namespace, servicePort, jobFailureRetry)
				Expect(err).NotTo(HaveOccurred())

				err = job.WaitForJobToFail(tcpJob, 90*time.Second)
				Expect(err).NotTo(HaveOccurred(), "connectivity is *not* expected, since there isn't an exposed service")
			})
		})
	})
}))

func createServiceConnectivityJob(serviceName, namespace string, servicePort int, retries int32) (*batchv1.Job, error) {
	serviceFQDN := fmt.Sprintf("%s.%s", serviceName, namespace)

	By(fmt.Sprintf("starting a job which tries to reach the vmi via service %s, on port %d", serviceFQDN, servicePort))
	tcpJob := job.NewHelloWorldJobTCP(serviceFQDN, strconv.Itoa(servicePort))
	tcpJob.Spec.BackoffLimit = &retries
	return kubevirt.Client().BatchV1().Jobs(namespace).Create(context.Background(), tcpJob, k8smetav1.CreateOptions{})
}
