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
	"os"
	"strings"
	"time"

	expect "github.com/google/goexpect"
	k8snetworkplumbingwgv1 "github.com/k8snetworkplumbingwg/network-attachment-definition-client/pkg/apis/k8s.cni.cncf.io/v1"
	. "github.com/onsi/ginkgo"
	"github.com/onsi/ginkgo/extensions/table"
	. "github.com/onsi/gomega"

	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"

	v1 "kubevirt.io/client-go/api/v1"
	"kubevirt.io/client-go/kubecli"
	"kubevirt.io/kubevirt/tests"
	"kubevirt.io/kubevirt/tests/console"
	"kubevirt.io/kubevirt/tests/libnet"
	"kubevirt.io/kubevirt/tests/libvmi"
)

const (
	istioInjectSidecarAnnotation = "sidecar.istio.io/inject"
	istioDeployedEnvVariable     = "KUBEVIRT_DEPLOY_ISTIO"
	vmiAppSelector               = "istio-vmi-app"
	svcDeclaredTestPort          = 1500
	svcUndeclaredTestPort        = 1501
)

var _ = SIGDescribe("[Serial] Istio", func() {
	var (
		err        error
		vmi        *v1.VirtualMachineInstance
		virtClient kubecli.KubevirtClient
	)
	BeforeEach(func() {
		if !istioServiceMeshDeployed() {
			Skip("Istio service mesh is required for service-mesh tests to run")
		}
	})

	Context("Virtual Machine with masquerade interface and explicitly specified ports", func() {
		createJobCheckingVMIReachability := func(serverVMI *v1.VirtualMachineInstance, targetPort int) (*batchv1.Job, error) {
			By("Starting HTTP Server")
			tests.StartHTTPServer(vmi, targetPort)

			By("Getting back the VMI IP")
			vmi, err = virtClient.VirtualMachineInstance(vmi.Namespace).Get(vmi.Name, &metav1.GetOptions{})
			Expect(err).ShouldNot(HaveOccurred())
			vmiIP := vmi.Status.Interfaces[0].IP

			By("Running job to send a request to the server")
			return virtClient.BatchV1().Jobs(tests.NamespaceTestDefault).Create(
				context.Background(),
				tests.NewHelloWorldJobHTTP(vmiIP, fmt.Sprintf("%d", targetPort)),
				metav1.CreateOptions{},
			)
		}
		BeforeEach(func() {
			tests.BeforeTestCleanup()

			virtClient, err = kubecli.GetKubevirtClient()
			tests.PanicOnError(err)

			// Enable sidecar injection by setting the namespace label
			Expect(libnet.AddLabelToNamespace(virtClient, tests.NamespaceTestDefault, tests.IstioInjectNamespaceLabel, "enabled")).ShouldNot(HaveOccurred())
			defer func() {
				Expect(libnet.RemoveLabelFromNamespace(virtClient, tests.NamespaceTestDefault, tests.IstioInjectNamespaceLabel)).ShouldNot(HaveOccurred())
			}()

			By("Create NetworkAttachmentDefinition")
			nad := generateIstioCNINetworkAttachmentDefinition()
			_, err = virtClient.NetworkClient().K8sCniCncfIoV1().NetworkAttachmentDefinitions(tests.NamespaceTestDefault).Create(context.TODO(), nad, metav1.CreateOptions{})
			Expect(err).ShouldNot(HaveOccurred())

			By("Creating VMI")
			ports := []v1.Port{
				{Port: svcDeclaredTestPort},
				{Port: svcUndeclaredTestPort},
			}
			vmi = newVMIWithIstioSidecar(ports)
			vmi, err = virtClient.VirtualMachineInstance(tests.NamespaceTestDefault).Create(vmi)
			Expect(err).ShouldNot(HaveOccurred())

			By("Creating k8s service")
			service := newService()
			_, err = virtClient.CoreV1().Services(tests.NamespaceTestDefault).Create(context.Background(), service, metav1.CreateOptions{})
			Expect(err).ShouldNot(HaveOccurred())

			By("Waiting for VMI to be ready")
			tests.WaitUntilVMIReady(vmi, console.LoginToCirros)
		})
		Describe("Inbound traffic", func() {
			table.DescribeTable("request to VMI", func(port int) {
				job, err := createJobCheckingVMIReachability(vmi, port)
				Expect(err).ShouldNot(HaveOccurred())

				By("Waiting for the job to succeed")
				Expect(tests.WaitForJobToSucceed(job, 480*time.Second)).To(Succeed())
			},
				table.Entry("should reach VMI HTTP server on service declared port", svcDeclaredTestPort),
				table.Entry("should reach VMI HTTP server on service undeclared port", svcUndeclaredTestPort),
			)
			Context("With PeerAuthentication enforcing mTLS", func() {
				BeforeEach(func() {
					peerAuthenticationRes := schema.GroupVersionResource{Group: "security.istio.io", Version: "v1beta1", Resource: "peerauthentications"}
					peerAuthentication := generateStrictPeerAuthentication()
					_, err = virtClient.DynamicClient().Resource(peerAuthenticationRes).Namespace(tests.NamespaceTestDefault).Create(context.Background(), peerAuthentication, metav1.CreateOptions{})
					Expect(err).ShouldNot(HaveOccurred())
				})
				table.DescribeTable("request to VMI", func(port int) {
					job, err := createJobCheckingVMIReachability(vmi, port)
					Expect(err).ShouldNot(HaveOccurred())

					By("Waiting for the job to fail")
					Expect(tests.WaitForJobToSucceed(job, 480*time.Second)).NotTo(Succeed())
				},
					table.Entry("client outside mesh should not reach VMI HTTP server on service declared port", svcDeclaredTestPort),
					table.Entry("client outside mesh should not reach VMI HTTP server on service undeclared port", svcUndeclaredTestPort),
				)
			})
		})
		Describe("Outbound traffic", func() {
			var (
				serverVMIAddress string
				serverVMI        *v1.VirtualMachineInstance
				testPort         = 4200
			)

			curlCommand := func(serverIP string, port int) string {
				return fmt.Sprintf("curl -sD - -o /dev/null http://%s:%d | head -n 1\n", serverIP, port)
			}

			generateExpectedHTTPReturnCodeRegex := func(codeRegex string) string {
				return fmt.Sprintf("HTTP\\/[123456789\\.]{1,3}\\s(%s)", codeRegex)
			}

			BeforeEach(func() {
				serverVMI = libvmi.NewCirros(
					libvmi.WithNetwork(v1.DefaultPodNetwork()),
					libvmi.WithInterface(libvmi.InterfaceDeviceWithMasqueradeBinding([]v1.Port{}...)),
				)
				serverVMI, err = virtClient.VirtualMachineInstance(tests.NamespaceTestDefault).Create(serverVMI)
				Expect(err).ToNot(HaveOccurred())
				Expect(console.LoginToCirros(serverVMI)).To(Succeed())

				By("Starting HTTP Server")
				tests.StartHTTPServer(serverVMI, testPort)

				By("Getting back the Server VMI IP")
				serverVMI, err = virtClient.VirtualMachineInstance(serverVMI.Namespace).Get(serverVMI.Name, &metav1.GetOptions{})
				Expect(err).ShouldNot(HaveOccurred())
				serverVMIAddress = serverVMI.Status.Interfaces[0].IP
			})

			// Istio envoy may intercept the request, we need to check the header response, not just the
			// return code of curl, because if envoy proxy forbids a request, it responds with 502 Bad Gateway code
			It("Should be able to reach http server outside of mesh", func() {
				err = console.SafeExpectBatch(vmi, []expect.Batcher{
					&expect.BSnd{S: "\n"},
					&expect.BExp{R: console.PromptExpression},
					&expect.BSnd{S: curlCommand(serverVMIAddress, testPort)},
					&expect.BExp{R: generateExpectedHTTPReturnCodeRegex("200")},
					&expect.BSnd{S: "echo $?\n"},
					&expect.BExp{R: console.RetValue("0")},
				}, 60)
				Expect(err).ToNot(HaveOccurred())
			})
			Context("With Sidecar allowing only registered external services", func() {
				BeforeEach(func() {
					sidecarRes := schema.GroupVersionResource{Group: "networking.istio.io", Version: "v1beta1", Resource: "sidecars"}
					registryOnlySidecar := generateRegistryOnlySidecar()
					_, err = virtClient.DynamicClient().Resource(sidecarRes).Namespace(tests.NamespaceTestDefault).Create(context.TODO(), registryOnlySidecar, metav1.CreateOptions{})
					Expect(err).ShouldNot(HaveOccurred())
				})

				// Istio Envoy will intercept the request because of the OutboundTrafficPolicy set to REGISTRY_ONLY.
				// Envoy responds with 502 Bad Gateway return code.
				// After Sidecar with OutboundTrafficPolicy is created, it may take a while for the Envoy proxy
				// to sync with the change, first request may still get through, hence the Eventually.
				It("Should not be able to reach http server outside of mesh", func() {
					Eventually(func() error {
						return console.SafeExpectBatch(vmi, []expect.Batcher{
							&expect.BSnd{S: "\n"},
							&expect.BExp{R: console.PromptExpression},
							&expect.BSnd{S: curlCommand(serverVMIAddress, testPort)},
							&expect.BExp{R: generateExpectedHTTPReturnCodeRegex("5..")},
							&expect.BSnd{S: "echo $?\n"},
							&expect.BExp{R: console.RetValue("0")},
						}, 60)
					}, 90*time.Second, 10*time.Second)
				})
			})
		})
	})
})

func istioServiceMeshDeployed() bool {
	value := os.Getenv(istioDeployedEnvVariable)
	if strings.ToLower(value) == "true" {
		return true
	}
	return false
}

func newService() *corev1.Service {
	return &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name: fmt.Sprintf("%s-service", vmiAppSelector),
		},
		Spec: corev1.ServiceSpec{
			Selector: map[string]string{
				"app": vmiAppSelector,
			},
			Ports: []corev1.ServicePort{
				{
					Port: svcDeclaredTestPort,
				},
			},
		},
	}
}

func newVMIWithIstioSidecar(ports []v1.Port) *v1.VirtualMachineInstance {
	vmi := libvmi.NewCirros(
		libvmi.WithNetwork(v1.DefaultPodNetwork()),
		libvmi.WithInterface(libvmi.InterfaceDeviceWithMasqueradeBinding(ports...)),
		libvmi.WithLabel("app", vmiAppSelector),
		libvmi.WithAnnotation(istioInjectSidecarAnnotation, "true"),
	)
	// Istio-proxy requires service account token to be mounted
	tests.AddServiceAccountDisk(vmi, "default")
	return vmi
}

func generateIstioCNINetworkAttachmentDefinition() *k8snetworkplumbingwgv1.NetworkAttachmentDefinition {
	return &k8snetworkplumbingwgv1.NetworkAttachmentDefinition{
		ObjectMeta: metav1.ObjectMeta{
			Name: "istio-cni",
		},
		Spec: k8snetworkplumbingwgv1.NetworkAttachmentDefinitionSpec{},
	}
}

func generateStrictPeerAuthentication() *unstructured.Unstructured {
	return &unstructured.Unstructured{
		Object: map[string]interface{}{
			"apiVersion": "security.istio.io/v1beta1",
			"kind":       "PeerAuthentication",
			"metadata": map[string]interface{}{
				"name": "strict-pa",
			},
			"spec": map[string]interface{}{
				"mtls": map[string]interface{}{
					"mode": "STRICT",
				},
			},
		},
	}
}

func generateRegistryOnlySidecar() *unstructured.Unstructured {
	return &unstructured.Unstructured{
		Object: map[string]interface{}{
			"apiVersion": "networking.istio.io/v1beta1",
			"kind":       "Sidecar",
			"metadata": map[string]interface{}{
				"name": "registry-only-sidecar",
			},
			"spec": map[string]interface{}{
				"outboundTrafficPolicy": map[string]interface{}{
					"mode": "REGISTRY_ONLY",
				},
			},
		},
	}
}
