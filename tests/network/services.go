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
 * Copyright 2020 Red Hat, Inc.
 *
 */

package network

import (
	"context"
	"fmt"
	"strconv"
	"time"

	. "github.com/onsi/ginkgo"
	"github.com/onsi/ginkgo/extensions/table"
	. "github.com/onsi/gomega"

	batchv1 "k8s.io/api/batch/v1"
	k8sv1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	k8smetav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"

	v1 "kubevirt.io/client-go/api/v1"
	"kubevirt.io/client-go/kubecli"
	"kubevirt.io/kubevirt/tests"
	"kubevirt.io/kubevirt/tests/console"
	"kubevirt.io/kubevirt/tests/libnet"
	"kubevirt.io/kubevirt/tests/libvmi"
)

var _ = SIGDescribe("[Serial]Services", func() {
	var virtClient kubecli.KubevirtClient

	runTCPClientExpectingHelloWorldFromServer := func(host, port, namespace string, retries int32) *batchv1.Job {
		job := tests.NewHelloWorldJob(host, port)
		job.Spec.BackoffLimit = &retries
		var err error
		job, err = virtClient.BatchV1().Jobs(namespace).Create(context.Background(), job, k8smetav1.CreateOptions{})
		ExpectWithOffset(1, err).ToNot(HaveOccurred())
		return job
	}

	exposeExistingVMISpec := func(vmi *v1.VirtualMachineInstance, subdomain string, hostname string, selectorLabelKey string, selectorLabelValue string) *v1.VirtualMachineInstance {
		vmi.Labels = map[string]string{selectorLabelKey: selectorLabelValue}
		vmi.Spec.Subdomain = subdomain
		vmi.Spec.Hostname = hostname

		return vmi
	}

	readyVMI := func(vmi *v1.VirtualMachineInstance) *v1.VirtualMachineInstance {
		createdVMI, err := virtClient.VirtualMachineInstance(tests.NamespaceTestDefault).Create(vmi)
		Expect(err).ToNot(HaveOccurred())

		return tests.WaitUntilVMIReady(createdVMI, libnet.WithIPv6(console.LoginToCirros))
	}

	cleanupVMI := func(virtClient kubecli.KubevirtClient, vmi *v1.VirtualMachineInstance) {
		By("Deleting the VMI")
		Expect(virtClient.VirtualMachineInstance(tests.NamespaceTestDefault).Delete(vmi.GetName(), &k8smetav1.DeleteOptions{})).To(Succeed())

		By("Waiting for the VMI to be gone")
		Eventually(func() error {
			_, err := virtClient.VirtualMachineInstance(tests.NamespaceTestDefault).Get(vmi.GetName(), &k8smetav1.GetOptions{})
			return err
		}, 2*time.Minute, time.Second).Should(SatisfyAll(HaveOccurred(), WithTransform(errors.IsNotFound, BeTrue())), "The VMI should be gone within the given timeout")
	}

	cleanupService := func(namespace string, serviceName string) error {
		return virtClient.CoreV1().Services(namespace).Delete(context.Background(), serviceName, k8smetav1.DeleteOptions{})
	}

	assertConnectivityToService := func(serviceName, namespace string, servicePort int) (func() error, error) {
		serviceFQDN := fmt.Sprintf("%s.%s", serviceName, namespace)

		By(fmt.Sprintf("starting a job which tries to reach the vmi via service %s", serviceFQDN))
		job := runTCPClientExpectingHelloWorldFromServer(serviceFQDN, strconv.Itoa(servicePort), namespace, 3)

		By(fmt.Sprintf("waiting for the job to report a SUCCESSFUL connection attempt to service %s on port %d", serviceFQDN, servicePort))
		err := tests.WaitForJobToSucceed(job, 90*time.Second)
		return func() error {
			return virtClient.BatchV1().Jobs(tests.NamespaceTestDefault).Delete(context.Background(), job.Name, k8smetav1.DeleteOptions{})
		}, err
	}

	assertNoConnectivityToService := func(serviceName, namespace string, servicePort int) (func() error, error) {
		serviceFQDN := fmt.Sprintf("%s.%s", serviceName, namespace)

		By(fmt.Sprintf("starting a job which tries to reach the vmi via service %s", serviceFQDN))
		job := runTCPClientExpectingHelloWorldFromServer(serviceFQDN, strconv.Itoa(servicePort), namespace, 0)

		By(fmt.Sprintf("waiting for the job to report a FAILED connection attempt to service %s on port %d", serviceFQDN, servicePort))
		err := tests.WaitForJobToFail(job, 90*time.Second)
		return func() error {
			return virtClient.BatchV1().Jobs(tests.NamespaceTestDefault).Delete(context.Background(), job.Name, k8smetav1.DeleteOptions{})
		}, err
	}

	BeforeEach(func() {
		var err error
		virtClient, err = kubecli.GetKubevirtClient()
		Expect(err).NotTo(HaveOccurred(), "Should successfully initialize an API client")
	})

	Context("bridge interface binding", func() {
		var inboundVMI *v1.VirtualMachineInstance
		var serviceName string

		const (
			selectorLabelKey   = "expose"
			selectorLabelValue = "me"
			servicePort        = 1500
		)

		createVMISpecWithBridgeInterface := func() *v1.VirtualMachineInstance {
			return libvmi.NewCirros(
				libvmi.WithInterface(libvmi.InterfaceDeviceWithBridgeBinding()),
				libvmi.WithNetwork(v1.DefaultPodNetwork()))
		}

		createReadyVMIWithBridgeBindingAndExposedService := func(hostname string, subdomain string) *v1.VirtualMachineInstance {
			return readyVMI(
				exposeExistingVMISpec(
					createVMISpecWithBridgeInterface(), subdomain, hostname, selectorLabelKey, selectorLabelValue))
		}

		BeforeEach(func() {
			subdomain := "vmi"
			hostname := "inbound"

			inboundVMI = createReadyVMIWithBridgeBindingAndExposedService(hostname, subdomain)
			tests.StartTCPServer(inboundVMI, servicePort)
		})

		AfterEach(func() {
			Expect(inboundVMI).NotTo(BeNil(), "the VMI object must exist in order to be deleted.")
			cleanupVMI(virtClient, inboundVMI)
		})

		Context("with a service matching the vmi exposed", func() {
			var jobCleanup func() error

			BeforeEach(func() {
				serviceName = "myservice"

				service := buildServiceSpec(serviceName, servicePort, servicePort, selectorLabelKey, selectorLabelValue)
				_, err := virtClient.CoreV1().Services(inboundVMI.Namespace).Create(context.Background(), service, k8smetav1.CreateOptions{})
				Expect(err).ToNot(HaveOccurred())
			})

			AfterEach(func() {
				Expect(cleanupService(inboundVMI.GetNamespace(), serviceName)).To(Succeed(), "cleaning up the k8sv1.Service entity should have succeeded.")
			})

			AfterEach(func() {
				Expect(jobCleanup).NotTo(BeNil(), "a k8sv1.Job cleaning up function should exist")
				Expect(jobCleanup()).To(Succeed(), "cleaning up the k8sv1.Job entity should have succeeded.")
				jobCleanup = nil
			})

			It("[test_id:1547] should be able to reach the vmi based on labels specified on the vmi", func() {
				var err error

				jobCleanup, err = assertConnectivityToService(serviceName, inboundVMI.Namespace, servicePort)
				Expect(err).NotTo(HaveOccurred(), "connectivity is expected to the exposed service")
			})

			It("[test_id:1548] should fail to reach the vmi if an invalid servicename is used", func() {
				var err error

				jobCleanup, err = assertNoConnectivityToService("wrongservice", inboundVMI.Namespace, servicePort)
				Expect(err).NotTo(HaveOccurred(), "connectivity is *not* expected, since there isn't an exposed service")
			})
		})

		Context("with a subdomain and a headless service given", func() {
			var jobCleanup func() error

			BeforeEach(func() {
				serviceName = inboundVMI.Spec.Subdomain

				service := buildHeadlessServiceSpec(serviceName, servicePort, servicePort, selectorLabelKey, selectorLabelValue)
				_, err := virtClient.CoreV1().Services(inboundVMI.Namespace).Create(context.Background(), service, k8smetav1.CreateOptions{})
				Expect(err).ToNot(HaveOccurred())
			})

			AfterEach(func() {
				Expect(virtClient.CoreV1().Services(inboundVMI.Namespace).Delete(context.Background(), serviceName, k8smetav1.DeleteOptions{})).To(Succeed())
			})

			AfterEach(func() {
				Expect(jobCleanup()).To(Succeed(), "cleaning up the k8sv1.Service entity should have succeeded.")
			})

			It("[test_id:1549]should be able to reach the vmi via its unique fully qualified domain name", func() {
				var err error
				serviceHostnameWithSubdomain := fmt.Sprintf("%s.%s", inboundVMI.Spec.Hostname, inboundVMI.Spec.Subdomain)

				jobCleanup, err = assertConnectivityToService(serviceHostnameWithSubdomain, inboundVMI.Namespace, servicePort)
				Expect(err).NotTo(HaveOccurred(), "connectivity is expected to the exposed service")
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

		createReadyVMIWithMasqueradeBindingAndExposedService := func(hostname string, subdomain string) *v1.VirtualMachineInstance {
			vmi := libvmi.NewCirros(
				libvmi.WithInterface(libvmi.InterfaceDeviceWithMasqueradeBinding()),
				libvmi.WithNetwork(v1.DefaultPodNetwork()))
			return readyVMI(
				exposeExistingVMISpec(vmi, subdomain, hostname, selectorLabelKey, selectorLabelValue))
		}

		BeforeEach(func() {
			subdomain := "vmi"
			hostname := "inbound"

			inboundVMI = createReadyVMIWithMasqueradeBindingAndExposedService(hostname, subdomain)
			tests.StartTCPServer(inboundVMI, servicePort)
		})

		AfterEach(func() {
			Expect(inboundVMI).NotTo(BeNil(), "the VMI object must exist in order to be deleted.")
			cleanupVMI(virtClient, inboundVMI)
		})

		Context("with a service matching the vmi exposed", func() {
			var jobCleanup func() error
			var service *k8sv1.Service

			AfterEach(func() {
				Expect(jobCleanup).NotTo(BeNil(), "a k8sv1.Job cleaning up function should exist")
				Expect(jobCleanup()).To(Succeed(), "cleaning up the k8sv1.Job entity should have succeeded.")
				jobCleanup = nil
			})

			AfterEach(func() {
				Expect(cleanupService(inboundVMI.GetNamespace(), service.Name)).To(Succeed(), "cleaning up the k8sv1.Service entity should have succeeded.")
			})

			table.DescribeTable("[Conformance] should be able to reach the vmi based on labels specified on the vmi", func(ipFamily k8sv1.IPFamily) {
				serviceName := "myservice"
				By("setting up resources to expose the VMI via a service", func() {
					if ipFamily == k8sv1.IPv6Protocol {
						libnet.SkipWhenNotDualStackCluster(virtClient)

						serviceName = serviceName + "v6"
						service = buildIPv6ServiceSpec(serviceName, servicePort, servicePort, selectorLabelKey, selectorLabelValue)
					} else {
						service = buildServiceSpec(serviceName, servicePort, servicePort, selectorLabelKey, selectorLabelValue)
					}

					_, err := virtClient.CoreV1().Services(inboundVMI.Namespace).Create(context.Background(), service, k8smetav1.CreateOptions{})
					Expect(err).NotTo(HaveOccurred(), "the k8sv1.Service entity should have been created.")
				})

				By("checking connectivity the the exposed service")
				var err error

				jobCleanup, err = assertConnectivityToService(serviceName, inboundVMI.Namespace, servicePort)
				Expect(err).NotTo(HaveOccurred(), "connectivity is expected to the exposed service")
			},
				table.Entry("when the service is exposed by an IPv4 address.", k8sv1.IPv4Protocol),
				table.Entry("when the service is exposed by an IPv6 address.", k8sv1.IPv6Protocol),
			)
		})

		Context("*without* a service matching the vmi exposed", func() {
			var jobCleanup func() error
			var serviceName string

			AfterEach(func() {
				Expect(jobCleanup).NotTo(BeNil(), "a k8sv1.Job cleaning up function should exist")
				Expect(jobCleanup()).To(Succeed(), "cleaning up the k8sv1.Job entity should have succeeded.")
				jobCleanup = nil
			})

			It("should fail to reach the vmi", func() {
				var err error
				serviceName = "missingservice"

				jobCleanup, err = assertNoConnectivityToService(serviceName, inboundVMI.Namespace, servicePort)
				Expect(err).NotTo(HaveOccurred(), "connectivity is *not* expected, since there isn't an exposed service")
			})
		})
	})
})

func buildHeadlessServiceSpec(serviceName string, exposedPort int, portToExpose int, selectorKey string, selectorValue string) *k8sv1.Service {
	service := buildServiceSpec(serviceName, exposedPort, portToExpose, selectorKey, selectorValue)
	service.Spec.ClusterIP = k8sv1.ClusterIPNone
	return service
}

func buildIPv6ServiceSpec(serviceName string, exposedPort int, portToExpose int, selectorKey string, selectorValue string) *k8sv1.Service {
	service := buildServiceSpec(serviceName, exposedPort, portToExpose, selectorKey, selectorValue)
	ipv6Family := k8sv1.IPv6Protocol
	service.Spec.IPFamilies = []k8sv1.IPFamily{ipv6Family}

	return service
}

func buildServiceSpec(serviceName string, exposedPort int, portToExpose int, selectorKey string, selectorValue string) *k8sv1.Service {
	return &k8sv1.Service{
		ObjectMeta: k8smetav1.ObjectMeta{
			Name: serviceName,
		},
		Spec: k8sv1.ServiceSpec{
			Selector: map[string]string{
				selectorKey: selectorValue,
			},
			Ports: []k8sv1.ServicePort{
				{Protocol: k8sv1.ProtocolTCP, Port: int32(portToExpose), TargetPort: intstr.FromInt(exposedPort)},
			},
		},
	}
}
