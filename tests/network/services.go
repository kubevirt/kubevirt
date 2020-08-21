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
	"fmt"
	"strconv"
	"time"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	batchv1 "k8s.io/api/batch/v1"
	k8sv1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	k8smetav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
	"k8s.io/utils/net"

	v1 "kubevirt.io/client-go/api/v1"
	"kubevirt.io/client-go/kubecli"
	"kubevirt.io/kubevirt/tests"
	"kubevirt.io/kubevirt/tests/libvmi"
)

type vmiServiceManager struct {
	vmi                    *v1.VirtualMachineInstance
	primaryPodInterfaceIPs []string
	services               []k8sv1.Service
	port                   int
	serviceNamePrefix      string
	labelKey               string
	labelValue             string
}

var _ = SIGDescribe("Services", func() {
	var virtClient kubecli.KubevirtClient

	runTCPClientExpectingHelloWorldFromServer := func(host, port, namespace string) *batchv1.Job {
		job := tests.NewHelloWorldJob(host, port)
		job, err := virtClient.BatchV1().Jobs(namespace).Create(job)
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
		_, err := virtClient.VirtualMachineInstance(tests.NamespaceTestDefault).Create(vmi)
		Expect(err).ToNot(HaveOccurred())

		return tests.WaitUntilVMIReady(vmi, tests.LoggedInCirrosExpecter)
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

	assertConnectivityToService := func(serviceName, namespace string, servicePort int) {
		serviceFQDN := fmt.Sprintf("%s.%s", serviceName, namespace)

		By(fmt.Sprintf("starting a job which tries to reach the vmi via service %s", serviceFQDN))
		job := runTCPClientExpectingHelloWorldFromServer(serviceFQDN, strconv.Itoa(servicePort), namespace)

		By(fmt.Sprintf("waiting for the job to report a SUCCESSFUL connection attempt to service %s on port %d", serviceFQDN, servicePort))
		tests.WaitForJobToSucceed(job, 90)
	}

	assertNoConnectivityToService := func(serviceName, namespace string, servicePort int) {
		serviceFQDN := fmt.Sprintf("%s.%s", serviceName, namespace)

		By(fmt.Sprintf("starting a job which tries to reach the vmi via service %s", serviceFQDN))
		job := runTCPClientExpectingHelloWorldFromServer(serviceFQDN, strconv.Itoa(servicePort), namespace)

		By(fmt.Sprintf("waiting for the job to report a FAILED connection attempt to service %s on port %d", serviceFQDN, servicePort))
		tests.WaitForJobToFail(job, 90)
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
			BeforeEach(func() {
				serviceName = "myservice"

				service := buildServiceSpec(serviceName, servicePort, servicePort, selectorLabelKey, selectorLabelValue)
				_, err := virtClient.CoreV1().Services(inboundVMI.Namespace).Create(service)
				Expect(err).ToNot(HaveOccurred())
			})

			AfterEach(func() {
				Expect(virtClient.CoreV1().Services(inboundVMI.Namespace).Delete(serviceName, &k8smetav1.DeleteOptions{})).To(Succeed())
			})

			It("[test_id:1547] should be able to reach the vmi based on labels specified on the vmi", func() {
				assertConnectivityToService(serviceName, inboundVMI.Namespace, servicePort)
			})

			It("[test_id:1548] should fail to reach the vmi if an invalid servicename is used", func() {
				assertNoConnectivityToService("wrongservice", inboundVMI.Namespace, servicePort)
			})
		})

		Context("with a subdomain and a headless service given", func() {
			BeforeEach(func() {
				serviceName = inboundVMI.Spec.Subdomain

				service := buildHeadlessServiceSpec(serviceName, servicePort, servicePort, selectorLabelKey, selectorLabelValue)
				_, err := virtClient.CoreV1().Services(inboundVMI.Namespace).Create(service)
				Expect(err).ToNot(HaveOccurred())
			})

			AfterEach(func() {
				Expect(virtClient.CoreV1().Services(inboundVMI.Namespace).Delete(serviceName, &k8smetav1.DeleteOptions{})).To(Succeed())
			})

			It("[test_id:1549]should be able to reach the vmi via its unique fully qualified domain name", func() {
				serviceHostnameWithSubdomain := fmt.Sprintf("%s.%s", inboundVMI.Spec.Hostname, inboundVMI.Spec.Subdomain)
				assertConnectivityToService(serviceHostnameWithSubdomain, inboundVMI.Namespace, servicePort)
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
			var serviceManager *vmiServiceManager
			var serviceNamePrefix string

			BeforeEach(func() {
				serviceNamePrefix = "myservice"
				serviceManager = newVMIServiceManager(inboundVMI, servicePort, serviceNamePrefix, selectorLabelKey, selectorLabelValue)

				services := serviceManager.buildK8sServicesSpec()
				Expect(services).NotTo(BeEmpty(), "a service should be exposed per each iface. At least one *must* be present.")

				for _, exposedService := range services {
					_, err := virtClient.CoreV1().Services(inboundVMI.Namespace).Create(&exposedService)
					Expect(err).NotTo(HaveOccurred())
				}
			})

			AfterEach(func() {
				var errors []error
				for _, exposedService := range serviceManager.services {
					err := virtClient.CoreV1().Services(inboundVMI.Namespace).Delete(exposedService.Name, &k8smetav1.DeleteOptions{})
					errors = append(errors, err)
				}
				for _, err := range errors {
					Expect(err).NotTo(HaveOccurred())
				}
			})

			It("should be able to reach the vmi based on labels specified on the vmi", func() {
				for _, exposedService := range serviceManager.services {
					assertConnectivityToService(exposedService.Name, inboundVMI.Namespace, servicePort)
				}
			})

			It("should fail to reach the vmi if an invalid servicename is used", func() {
				assertNoConnectivityToService("wrongservice", inboundVMI.Namespace, servicePort)
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
	service.Spec.IPFamily = &ipv6Family

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

func newVMIServiceManager(vmi *v1.VirtualMachineInstance, port int, serviceNamePrefix string, selectorLabelKey string, selectorLabelValue string) *vmiServiceManager {
	primaryIfaceIPs := []string{}
	ifaces := vmi.Status.Interfaces
	if len(ifaces) > 0 {
		primaryIfaceIPs = ifaces[0].IPs
	}

	return &vmiServiceManager{
		vmi:                    vmi,
		primaryPodInterfaceIPs: primaryIfaceIPs,
		port:                   port,
		serviceNamePrefix:      serviceNamePrefix,
		labelKey:               selectorLabelKey,
		labelValue:             selectorLabelValue,
	}
}

func (si *vmiServiceManager) buildK8sServicesSpec() []k8sv1.Service {
	for _, ipAddr := range si.primaryPodInterfaceIPs {
		isIpv6 := net.IsIPv6String(ipAddr)
		var service *k8sv1.Service

		if isIpv6 {
			service = buildIPv6ServiceSpec(si.serviceNamePrefix+"v6", si.port, si.port, si.labelKey, si.labelValue)
		} else {
			service = buildServiceSpec(si.serviceNamePrefix, si.port, si.port, si.labelKey, si.labelValue)
		}

		si.services = append(si.services, *service)
	}
	return si.services
}
