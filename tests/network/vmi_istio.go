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

	"kubevirt.io/kubevirt/tests/decorators"
	"kubevirt.io/kubevirt/tests/libmigration"

	k8sv1 "k8s.io/api/core/v1"

	"kubevirt.io/kubevirt/tests/framework/checks"
	"kubevirt.io/kubevirt/tests/framework/kubevirt"
	"kubevirt.io/kubevirt/tests/testsuite"

	expect "github.com/google/goexpect"
	k8snetworkplumbingwgv1 "github.com/k8snetworkplumbingwg/network-attachment-definition-client/pkg/apis/k8s.cni.cncf.io/v1"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	netservice "kubevirt.io/kubevirt/tests/libnet/service"

	batchv1 "k8s.io/api/batch/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"

	v1 "kubevirt.io/api/core/v1"
	"kubevirt.io/client-go/kubecli"

	"kubevirt.io/kubevirt/pkg/network/istio"
	virtconfig "kubevirt.io/kubevirt/pkg/virt-config"
	"kubevirt.io/kubevirt/tests"
	"kubevirt.io/kubevirt/tests/console"
	"kubevirt.io/kubevirt/tests/libnet"
	"kubevirt.io/kubevirt/tests/libnet/job"
	"kubevirt.io/kubevirt/tests/libvmi"
	"kubevirt.io/kubevirt/tests/libwait"
)

const (
	istioDeployedEnvVariable  = "KUBEVIRT_DEPLOY_ISTIO"
	vmiAppSelectorKey         = "app"
	vmiAppSelectorValue       = "istio-vmi-app"
	vmiServerAppSelectorValue = "vmi-server-app"
	vmiServerHostName         = "vmi-server"
	vmiServerGateway          = "vmi-server-gw"
	vmiServerTestPort         = 4200
	svcDeclaredTestPort       = 1500
	svcUndeclaredTestPort     = 1501
	sshPort                   = 22
	// Istio uses certain ports for it's own purposes, this port server to verify that traffic is not routed
	// into the VMI for these ports. https://istio.io/latest/docs/ops/deployment/requirements/
	istioRestrictedPort = istio.EnvoyTunnelPort

	istioInjectNamespaceLabel = "istio-injection"
)

const (
	networkingIstioIO = "networking.istio.io"
	securityIstioIO   = "security.istio.io"
	istioApiVersion   = "v1beta1"
)

type VmType int

const (
	Passt VmType = iota
	Masquerade
)

var istioTests = func(vmType VmType) {
	var (
		err        error
		vmi        *v1.VirtualMachineInstance
		namespace  string
		virtClient kubecli.KubevirtClient
		vmiPorts   []v1.Port
		// Istio Envoy treats traffic differently for ports declared and undeclared in an associated k8s service.
		// Having both, declared and undeclared ports specified for VMIs with explicit ports allows to test both cases.
		explicitPorts = []v1.Port{
			{Port: svcDeclaredTestPort},
			{Port: svcUndeclaredTestPort},
			{Port: sshPort},
		}
	)
	BeforeEach(func() {
		namespace = testsuite.GetTestNamespace(nil)
		if vmType == Passt {
			Expect(checks.HasFeature(virtconfig.PasstGate)).To(BeTrue())
		}
	})

	BeforeEach(func() {
		if !istioServiceMeshDeployed() {
			Skip("Istio service mesh is required for service-mesh tests to run")
		}
	})

	Context("Virtual Machine with istio supported interface", func() {
		createJobCheckingVMIReachability := func(serverVMI *v1.VirtualMachineInstance, targetPort int) (*batchv1.Job, error) {
			By("Starting HTTP Server")
			tests.StartPythonHttpServer(vmi, targetPort)

			By("Getting back the VMI IP")
			vmi, err = virtClient.VirtualMachineInstance(vmi.Namespace).Get(context.Background(), vmi.Name, &metav1.GetOptions{})
			Expect(err).ShouldNot(HaveOccurred())
			vmiIP := libnet.GetVmiPrimaryIPByFamily(vmi, k8sv1.IPv4Protocol)

			By("Running job to send a request to the server")
			return virtClient.BatchV1().Jobs(namespace).Create(
				context.Background(),
				job.NewHelloWorldJobHTTP(vmiIP, fmt.Sprintf("%d", targetPort)),
				metav1.CreateOptions{},
			)
		}
		BeforeEach(func() {
			virtClient = kubevirt.Client()

			libnet.SkipWhenClusterNotSupportIpv4()

			By("Create NetworkAttachmentDefinition")
			nad := generateIstioCNINetworkAttachmentDefinition()
			_, err = virtClient.NetworkClient().K8sCniCncfIoV1().NetworkAttachmentDefinitions(namespace).Create(context.TODO(), nad, metav1.CreateOptions{})
			Expect(err).ShouldNot(HaveOccurred())

			By("Creating k8s service for the VMI")

			serviceName := fmt.Sprintf("%s-service", vmiAppSelectorValue)
			service := netservice.BuildSpec(serviceName, svcDeclaredTestPort, svcDeclaredTestPort, vmiAppSelectorKey, vmiAppSelectorValue)
			_, err = virtClient.CoreV1().Services(namespace).Create(context.Background(), service, metav1.CreateOptions{})
			Expect(err).ShouldNot(HaveOccurred())
		})
		JustBeforeEach(func() {
			// Enable sidecar injection by setting the namespace label
			Expect(libnet.AddLabelToNamespace(virtClient, namespace, istioInjectNamespaceLabel, "enabled")).ShouldNot(HaveOccurred())
			defer func() {
				Expect(libnet.RemoveLabelFromNamespace(virtClient, namespace, istioInjectNamespaceLabel)).ShouldNot(HaveOccurred())
			}()

			By("Creating VMI")
			vmi, err = newVMIWithIstioSidecar(vmiPorts, vmType)
			Expect(err).ShouldNot(HaveOccurred())
			vmi, err = virtClient.VirtualMachineInstance(namespace).Create(context.Background(), vmi)
			Expect(err).ShouldNot(HaveOccurred())

			By("Waiting for VMI to be ready")
			libwait.WaitUntilVMIReady(vmi, console.LoginToAlpine)
		})
		Describe("Live Migration", decorators.SigComputeMigrations, func() {
			var (
				sourcePodName string
			)
			allContainersCompleted := func(podName string) error {
				pod, err := virtClient.CoreV1().Pods(vmi.Namespace).Get(context.TODO(), podName, metav1.GetOptions{})
				if err != nil {
					return err
				}
				for _, containerStatus := range pod.Status.ContainerStatuses {
					if containerStatus.State.Terminated == nil {
						return fmt.Errorf("container %s is not terminated, state: %s", containerStatus.Name, containerStatus.State.String())
					}
				}
				return nil
			}
			BeforeEach(func() {
				checks.SkipIfMigrationIsNotPossible()
				if vmType == Passt {
					Skip("passt doesn't support live migration.")
				}
			})
			JustBeforeEach(func() {
				sourcePodName = tests.GetVmPodName(virtClient, vmi)
				migration := tests.NewRandomMigration(vmi.Name, vmi.Namespace)
				libmigration.RunMigrationAndExpectToCompleteWithDefaultTimeout(virtClient, migration)
			})
			It("All containers should complete in source virt-launcher pod after migration", func() {
				const containerCompletionWaitTime = 60
				Eventually(func() error {
					return allContainersCompleted(sourcePodName)
				}, containerCompletionWaitTime, time.Second).Should(Succeed(), fmt.Sprintf("all containers should complete in source virt-launcher pod"))
			})
		})
		Describe("SSH traffic", func() {
			var bastionVMI *v1.VirtualMachineInstance
			sshCommand := func(user, ipAddress string) string {
				return fmt.Sprintf("ssh -y %s@%s\n", user, ipAddress)
			}
			checkSSHConnection := func(vmi *v1.VirtualMachineInstance, vmiAddress string) error {
				user := "doesntmatter"
				return console.SafeExpectBatch(vmi, []expect.Batcher{
					&expect.BSnd{S: "\n"},
					&expect.BExp{R: console.PromptExpression},
					&expect.BSnd{S: sshCommand(user, vmiAddress)},
					&expect.BExp{R: fmt.Sprintf("%s@%s's password: ", user, vmiAddress)},
				}, 60)
			}

			BeforeEach(func() {
				bastionVMI = libvmi.NewCirros(
					libvmi.WithNetwork(v1.DefaultPodNetwork()),
					libvmi.WithInterface(libvmi.InterfaceDeviceWithMasqueradeBinding([]v1.Port{}...)),
				)

				bastionVMI, err = virtClient.VirtualMachineInstance(namespace).Create(context.Background(), bastionVMI)
				Expect(err).ToNot(HaveOccurred())
				bastionVMI = libwait.WaitUntilVMIReady(bastionVMI, console.LoginToCirros)
			})
			Context("With VMI having explicit ports specified", func() {
				BeforeEach(func() {
					vmiPorts = explicitPorts
				})
				It("should ssh to VMI with Istio proxy", func() {
					By("Getting the VMI IP")
					vmi, err = virtClient.VirtualMachineInstance(vmi.Namespace).Get(context.Background(), vmi.Name, &metav1.GetOptions{})
					Expect(err).ShouldNot(HaveOccurred())
					vmiIP := libnet.GetVmiPrimaryIPByFamily(vmi, k8sv1.IPv4Protocol)

					Expect(
						checkSSHConnection(bastionVMI, vmiIP),
					).Should(Succeed())
				})
			})
			Context("With VMI having no explicit ports specified", func() {
				BeforeEach(func() {
					vmiPorts = []v1.Port{}
				})
				It("should ssh to VMI with Istio proxy", func() {
					By("Getting the VMI IP")
					vmi, err = virtClient.VirtualMachineInstance(vmi.Namespace).Get(context.Background(), vmi.Name, &metav1.GetOptions{})
					Expect(err).ShouldNot(HaveOccurred())
					vmiIP := libnet.GetVmiPrimaryIPByFamily(vmi, k8sv1.IPv4Protocol)

					Expect(
						checkSSHConnection(bastionVMI, vmiIP),
					).Should(Succeed())
				})
			})
		})
		Describe("Inbound traffic", func() {
			checkVMIReachability := func(vmi *v1.VirtualMachineInstance, targetPort int) error {
				httpJob, err := createJobCheckingVMIReachability(vmi, targetPort)
				if err != nil {
					return err
				}
				By("Waiting for the job to succeed")
				return job.WaitForJobToSucceed(httpJob, 480*time.Second)
			}

			Context("With VMI having explicit ports specified", func() {
				BeforeEach(func() {
					vmiPorts = explicitPorts
				})
				DescribeTable("request to VMI should reach HTTP server", func(targetPort int) {
					Expect(checkVMIReachability(vmi, targetPort)).To(Succeed())
				},
					Entry("on service declared port on VMI with explicit ports", svcDeclaredTestPort),
					Entry("on service undeclared port on VMI with explicit ports", svcUndeclaredTestPort),
				)
			})
			Context("With VMI having no explicit ports specified", func() {
				BeforeEach(func() {
					vmiPorts = []v1.Port{}
				})
				DescribeTable("request to VMI should reach HTTP server", func(targetPort int) {
					Expect(checkVMIReachability(vmi, targetPort)).To(Succeed())
				},
					Entry("on service declared port on VMI with no explicit ports", svcDeclaredTestPort),
					Entry("on service undeclared port on VMI with no explicit ports", svcUndeclaredTestPort),
				)
				It("Should not be able to reach service running on Istio restricted port", func() {
					Expect(checkVMIReachability(vmi, istioRestrictedPort)).NotTo(Succeed())
				})
			})

			Context("With PeerAuthentication enforcing mTLS", func() {
				BeforeEach(func() {
					peerAuthenticationRes := schema.GroupVersionResource{Group: "security.istio.io", Version: istioApiVersion, Resource: "peerauthentications"}
					peerAuthentication := generateStrictPeerAuthentication()
					_, err = virtClient.DynamicClient().Resource(peerAuthenticationRes).Namespace(namespace).Create(context.Background(), peerAuthentication, metav1.CreateOptions{})
					Expect(err).ShouldNot(HaveOccurred())
				})
				Context("With VMI having explicit ports specified", func() {
					BeforeEach(func() {
						vmiPorts = explicitPorts
					})
					DescribeTable("client outside mesh should NOT reach VMI HTTP server", func(targetPort int) {
						Expect(checkVMIReachability(vmi, targetPort)).NotTo(Succeed())
					},
						Entry("on service declared port on VMI with explicit ports", svcDeclaredTestPort),
						Entry("on service undeclared port on VMI with explicit ports", svcUndeclaredTestPort),
					)
				})
				Context("With VMI having no explicit ports specified", func() {
					BeforeEach(func() {
						vmiPorts = []v1.Port{}
					})
					DescribeTable("client outside mesh should NOT reach VMI HTTP server", func(targetPort int) {
						Expect(checkVMIReachability(vmi, targetPort)).NotTo(Succeed())
					},
						Entry("on service declared port on VMI with no explicit ports", svcDeclaredTestPort),
						Entry("on service undeclared port on VMI with no explicit ports", svcUndeclaredTestPort),
					)
				})
			})
		})
		Describe("Outbound traffic", func() {
			const (
				externalServiceCheckTimeout  = 5 * time.Second
				externalServiceCheckInterval = 1 * time.Second
				istioNamespace               = "istio-system"
			)
			var (
				ingressGatewayServiceIP string
				serverVMI               *v1.VirtualMachineInstance
			)

			curlCommand := func(ingressGatewayIP string) string {
				return fmt.Sprintf("curl -sD - -o /dev/null -Hhost:%s.example.com http://%s | head -n 1\n", vmiServerHostName, ingressGatewayIP)
			}

			generateExpectedHTTPReturnCodeRegex := func(codeRegex string) string {
				return fmt.Sprintf("HTTP\\/[123456789\\.]{1,3}\\s(%s)", codeRegex)
			}

			BeforeEach(func() {
				networkData := libnet.CreateDefaultCloudInitNetworkData()

				serverVMI = libvmi.NewAlpineWithTestTooling(
					libvmi.WithNetwork(v1.DefaultPodNetwork()),
					libvmi.WithInterface(libvmi.InterfaceDeviceWithMasqueradeBinding([]v1.Port{}...)),
					libvmi.WithLabel("version", "v1"),
					libvmi.WithLabel(vmiAppSelectorKey, vmiServerAppSelectorValue),
					libvmi.WithCloudInitNoCloudNetworkData(networkData),
					libvmi.WithNamespace(namespace),
				)
				By("Starting VirtualMachineInstance")
				serverVMI = tests.RunVMIAndExpectLaunch(serverVMI, 240)

				serverVMIService := netservice.BuildSpec("vmi-server", vmiServerTestPort, vmiServerTestPort, vmiAppSelectorKey, vmiServerAppSelectorValue)
				_, err = virtClient.CoreV1().Services(namespace).Create(context.Background(), serverVMIService, metav1.CreateOptions{})
				Expect(err).ToNot(HaveOccurred())

				By("Starting HTTP Server")
				Expect(console.LoginToAlpine(serverVMI)).To(Succeed())
				tests.StartPythonHttpServer(serverVMI, vmiServerTestPort)

				By("Creating Istio VirtualService")
				virtualServicesRes := schema.GroupVersionResource{Group: networkingIstioIO, Version: istioApiVersion, Resource: "virtualservices"}
				virtualService := generateVirtualService()
				_, err = virtClient.DynamicClient().Resource(virtualServicesRes).Namespace(namespace).Create(context.TODO(), virtualService, metav1.CreateOptions{})
				Expect(err).ToNot(HaveOccurred())

				By("Creating Istio DestinationRule")
				destinationRulesRes := schema.GroupVersionResource{Group: networkingIstioIO, Version: istioApiVersion, Resource: "destinationrules"}
				destinationRule := generateDestinationRule()
				_, err = virtClient.DynamicClient().Resource(destinationRulesRes).Namespace(namespace).Create(context.TODO(), destinationRule, metav1.CreateOptions{})
				Expect(err).ToNot(HaveOccurred())

				By("Creating Istio Gateway")
				gatewaysRes := schema.GroupVersionResource{Group: networkingIstioIO, Version: istioApiVersion, Resource: "gateways"}
				gateway := generateGateway()
				_, err = virtClient.DynamicClient().Resource(gatewaysRes).Namespace(namespace).Create(context.TODO(), gateway, metav1.CreateOptions{})
				Expect(err).ToNot(HaveOccurred())

				By("Getting Istio ingressgateway IP")
				ingressGatewayService, err := virtClient.CoreV1().Services(istioNamespace).Get(context.TODO(), "istio-ingressgateway", metav1.GetOptions{})
				Expect(err).ToNot(HaveOccurred())
				ingressGatewayServiceIP = ingressGatewayService.Spec.ClusterIP

			})

			checkHTTPServiceReturnCode := func(ingressGatewayAddress string, returnCode string) error {
				return console.SafeExpectBatch(vmi, []expect.Batcher{
					&expect.BSnd{S: "\n"},
					&expect.BExp{R: console.PromptExpression},
					&expect.BSnd{S: curlCommand(ingressGatewayAddress)},
					&expect.BExp{R: returnCode},
					&expect.BSnd{S: "echo $?\n"},
					&expect.BExp{R: console.RetValue("0")},
				}, 60)
			}

			Context("VMI with explicit ports", func() {
				BeforeEach(func() {
					vmiPorts = explicitPorts
				})
				It("Should be able to reach http server outside of mesh", func() {
					Expect(
						checkHTTPServiceReturnCode(ingressGatewayServiceIP, generateExpectedHTTPReturnCodeRegex("200")),
					).ToNot(HaveOccurred())
				})
			})
			Context("VMI with no explicit ports", func() {
				BeforeEach(func() {
					vmiPorts = []v1.Port{}
				})
				It("Should be able to reach http server outside of mesh", func() {
					Expect(
						checkHTTPServiceReturnCode(ingressGatewayServiceIP, generateExpectedHTTPReturnCodeRegex("200")),
					).ToNot(HaveOccurred())
				})
			})

			Context("With Sidecar allowing only registered external services", func() {
				// Istio Envoy will intercept the request because of the OutboundTrafficPolicy set to REGISTRY_ONLY.
				// Envoy responds with 502 Bad Gateway return code.
				// After Sidecar with OutboundTrafficPolicy is created, it may take a while for the Envoy proxy
				// to sync with the change, first request may still get through, hence the Eventually used for assertions.

				BeforeEach(func() {
					sidecarRes := schema.GroupVersionResource{Group: networkingIstioIO, Version: istioApiVersion, Resource: "sidecars"}
					registryOnlySidecar := generateRegistryOnlySidecar()
					_, err = virtClient.DynamicClient().Resource(sidecarRes).Namespace(namespace).Create(context.TODO(), registryOnlySidecar, metav1.CreateOptions{})
					Expect(err).ToNot(HaveOccurred())
				})

				Context("VMI with explicit ports", func() {
					BeforeEach(func() {
						vmiPorts = explicitPorts
					})
					It("Should not be able to reach http service outside of mesh", func() {
						Eventually(func() error {
							return checkHTTPServiceReturnCode(ingressGatewayServiceIP, generateExpectedHTTPReturnCodeRegex("5.."))
						}, externalServiceCheckTimeout, externalServiceCheckInterval).Should(Succeed())
					})
				})
				Context("VMI with no explicit ports", func() {
					BeforeEach(func() {
						vmiPorts = []v1.Port{}
					})
					It("Should not be able to reach http service outside of mesh", func() {
						Eventually(func() error {
							return checkHTTPServiceReturnCode(ingressGatewayServiceIP, generateExpectedHTTPReturnCodeRegex("5.."))
						}, externalServiceCheckTimeout, externalServiceCheckInterval).Should(Succeed())
					})
				})
			})
		})
	})
}

var istioTestsWithMasqueradeBinding = func() {
	istioTests(Masquerade)
}

var istioTestsWithPasstBinding = func() {
	istioTests(Passt)
}

var _ = SIGDescribe("[Serial] Istio with masquerade binding", decorators.Istio, Serial, istioTestsWithMasqueradeBinding)

var _ = SIGDescribe("[Serial] Istio with passt binding", decorators.Istio, decorators.PasstGate, Serial, istioTestsWithPasstBinding)

func istioServiceMeshDeployed() bool {
	return strings.ToLower(os.Getenv(istioDeployedEnvVariable)) == "true"
}

func newVMIWithIstioSidecar(ports []v1.Port, vmType VmType) (*v1.VirtualMachineInstance, error) {
	if vmType == Masquerade {
		return createMasqueradeVm(ports), nil

	}
	if vmType == Passt {
		return createPasstVm(ports), nil
	}
	return nil, nil
}

func createMasqueradeVm(ports []v1.Port) *v1.VirtualMachineInstance {
	networkData := libnet.CreateDefaultCloudInitNetworkData()
	vmi := libvmi.NewAlpineWithTestTooling(
		libvmi.WithNetwork(v1.DefaultPodNetwork()),
		libvmi.WithInterface(libvmi.InterfaceDeviceWithMasqueradeBinding(ports...)),
		libvmi.WithLabel(vmiAppSelectorKey, vmiAppSelectorValue),
		libvmi.WithAnnotation(istio.ISTIO_INJECT_ANNOTATION, "true"),
		libvmi.WithCloudInitNoCloudNetworkData(networkData),
		libvmi.WithCloudInitNoCloudUserData(enablePasswordAuth(), true),
	)
	return vmi
}

func createPasstVm(ports []v1.Port) *v1.VirtualMachineInstance {
	vmi := libvmi.NewAlpineWithTestTooling(
		libvmi.WithNetwork(v1.DefaultPodNetwork()),
		libvmi.WithInterface(libvmi.InterfaceDeviceWithPasstBinding(ports...)),
		libvmi.WithLabel(vmiAppSelectorKey, vmiAppSelectorValue),
		libvmi.WithAnnotation(istio.ISTIO_INJECT_ANNOTATION, "true"),
		libvmi.WithCloudInitNoCloudUserData(enablePasswordAuth(), true),
	)
	return vmi
}

func enablePasswordAuth() string {
	sedCmd := "sed -ri 's/PasswordAuthentication no/PasswordAuthentication yes/' /etc/ssh/sshd_config"

	return "#!/bin/sh\n" + sedCmd + "\n"
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
			"apiVersion": fmt.Sprintf("%s/%s", securityIstioIO, istioApiVersion),
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
			"apiVersion": fmt.Sprintf("%s/%s", networkingIstioIO, istioApiVersion),
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

func generateVirtualService() *unstructured.Unstructured {
	return &unstructured.Unstructured{
		Object: map[string]interface{}{
			"apiVersion": fmt.Sprintf("%s/%s", networkingIstioIO, istioApiVersion),
			"kind":       "VirtualService",
			"metadata": map[string]interface{}{
				"name": "vmi-server-vs",
			},
			"spec": map[string]interface{}{
				"gateways": []string{
					vmiServerGateway,
				},
				"hosts": []string{
					fmt.Sprintf("%s.example.com", vmiServerHostName),
				},
				"http": []interface{}{
					map[string]interface{}{
						"match": []map[string]interface{}{
							{
								"uri": map[string]interface{}{
									"prefix": "/",
								},
							},
						},
						"route": []map[string]interface{}{
							{
								"destination": map[string]interface{}{
									"port": map[string]interface{}{
										"number": vmiServerTestPort,
									},
									"host":   vmiServerHostName,
									"subset": "v1",
								},
							},
						},
					},
				},
			},
		},
	}
}

func generateDestinationRule() *unstructured.Unstructured {
	return &unstructured.Unstructured{
		Object: map[string]interface{}{
			"apiVersion": fmt.Sprintf("%s/%s", networkingIstioIO, istioApiVersion),
			"kind":       "DestinationRule",
			"metadata": map[string]interface{}{
				"name": "vmi-server-dr",
			},
			"spec": map[string]interface{}{
				"host": vmiServerHostName,
				"subsets": []map[string]interface{}{
					{
						"name": "v1",
						"labels": map[string]interface{}{
							"version": "v1",
						},
					},
				},
			},
		},
	}
}

func generateGateway() *unstructured.Unstructured {
	return &unstructured.Unstructured{
		Object: map[string]interface{}{
			"apiVersion": fmt.Sprintf("%s/%s", networkingIstioIO, istioApiVersion),
			"kind":       "Gateway",
			"metadata": map[string]interface{}{
				"name": vmiServerGateway,
			},
			"spec": map[string]interface{}{
				"selector": map[string]interface{}{
					"istio": "ingressgateway",
				},
				"servers": []map[string]interface{}{
					{
						"port": map[string]interface{}{
							"number":   80,
							"name":     "http",
							"protocol": "HTTP",
						},
						"hosts": []string{
							fmt.Sprintf("%s.example.com", vmiServerHostName),
						},
					},
				},
			},
		},
	}
}
