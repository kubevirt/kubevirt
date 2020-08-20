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

	v1 "kubevirt.io/client-go/api/v1"
	"kubevirt.io/client-go/kubecli"
	"kubevirt.io/kubevirt/tests"
	"kubevirt.io/kubevirt/tests/libvmi"
)

var _ = SIGDescribe("Services", func() {
	var virtClient kubecli.KubevirtClient

	runTCPClientExpectingHelloWorldFromServer := func(host, port, namespace string) *batchv1.Job {
		job := tests.NewHelloWorldJob(host, port)
		job, err := virtClient.BatchV1().Jobs(namespace).Create(job)
		ExpectWithOffset(1, err).ToNot(HaveOccurred())
		return job
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

		BeforeEach(func() {
			// inboundVMI expects implicitly to be added to the pod network
			inboundVMI = libvmi.NewCirros()
			inboundVMI.Labels = map[string]string{selectorLabelKey: selectorLabelValue}
			inboundVMI.Spec.Subdomain = "vmi"
			inboundVMI.Spec.Hostname = "inbound"

			_, err := virtClient.VirtualMachineInstance(tests.NamespaceTestDefault).Create(inboundVMI)
			Expect(err).ToNot(HaveOccurred())

			inboundVMI = tests.WaitUntilVMIReady(inboundVMI, tests.LoggedInCirrosExpecter)

			tests.StartTCPServer(inboundVMI, servicePort)
		})

		AfterEach(func() {
			if inboundVMI != nil {
				By("Deleting the VMI")
				Expect(virtClient.VirtualMachineInstance(tests.NamespaceTestDefault).Delete(inboundVMI.GetName(), &k8smetav1.DeleteOptions{})).To(Succeed())

				By("Waiting for the VMI to be gone")
				Eventually(func() error {
					_, err := virtClient.VirtualMachineInstance(tests.NamespaceTestDefault).Get(inboundVMI.GetName(), &k8smetav1.GetOptions{})
					return err
				}, 2*time.Minute, time.Second).Should(SatisfyAll(HaveOccurred(), WithTransform(errors.IsNotFound, BeTrue())), "The VMI should be gone within the given timeout")
			}
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
				By("starting a job which tries to reach the vmi via the defined service")
				job := runTCPClientExpectingHelloWorldFromServer(fmt.Sprintf("%s.%s", serviceName, inboundVMI.Namespace), strconv.Itoa(servicePort), inboundVMI.Namespace)

				By("waiting for the job to report a successful connection attempt")
				tests.WaitForJobToSucceed(job, 90)
			})

			It("[test_id:1548] should fail to reach the vmi if an invalid servicename is used", func() {

				By("starting a job which tries to reach the vmi via a non-existent service")
				job := runTCPClientExpectingHelloWorldFromServer(fmt.Sprintf("%s.%s", "wrongservice", inboundVMI.Namespace), strconv.Itoa(servicePort), inboundVMI.Namespace)

				By("waiting for the job to report an  unsuccessful connection attempt")
				tests.WaitForJobToFail(job, 90)
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
				By("starting a job which tries to reach the vm via the defined service")
				job := runTCPClientExpectingHelloWorldFromServer(fmt.Sprintf("%s.%s.%s", inboundVMI.Spec.Hostname, inboundVMI.Spec.Subdomain, inboundVMI.Namespace), strconv.Itoa(servicePort), inboundVMI.Namespace)

				By("waiting for the job to report a successful connection attempt")
				tests.WaitForJobToSucceed(job, 90)
			})
		})
	})
})

func buildHeadlessServiceSpec(serviceName string, exposedPort int, portToExpose int, selectorKey string, selectorValue string) *k8sv1.Service {
	service := buildServiceSpec(serviceName, exposedPort, portToExpose, selectorKey, selectorValue)
	service.Spec.ClusterIP = k8sv1.ClusterIPNone
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
