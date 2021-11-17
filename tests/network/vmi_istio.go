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

	netservice "kubevirt.io/kubevirt/tests/libnet/service"
	"kubevirt.io/kubevirt/tests/util"

	batchv1 "k8s.io/api/batch/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"

	v1 "kubevirt.io/api/core/v1"
	"kubevirt.io/client-go/kubecli"
	"kubevirt.io/kubevirt/pkg/network/istio"
	"kubevirt.io/kubevirt/tests"
	"kubevirt.io/kubevirt/tests/console"
	"kubevirt.io/kubevirt/tests/libnet"
	"kubevirt.io/kubevirt/tests/libvmi"
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
	// Istio uses certain ports for it's own purposes, this port server to verify that traffic is not routed
	// into the VMI for these ports. https://istio.io/latest/docs/ops/deployment/requirements/
	istioRestrictedPort = istio.EnvoyTunnelPort
)

const (
	networkingIstioIO = "networking.istio.io"
	securityIstioIO   = "security.istio.io"
	istioApiVersion   = "v1beta1"
)

var _ = SIGDescribe("[Serial] Istio", func() {
	var (
		err        error
		vmi        *v1.VirtualMachineInstance
		virtClient kubecli.KubevirtClient
		vmiPorts   []v1.Port
		// Istio Envoy treats traffic differently for ports declared and undeclared in an associated k8s service.
		// Having both, declared and undeclared ports specified for VMIs with explicit ports allows to test both cases.
		explicitPorts = []v1.Port{
			{Port: svcDeclaredTestPort},
			{Port: svcUndeclaredTestPort},
		}
	)
	BeforeEach(func() {
		if !istioServiceMeshDeployed() {
			Skip("Istio service mesh is required for service-mesh tests to run")
		}
	})

	Context("Virtual Machine with masquerade interface", func() {
		createJobCheckingVMIReachability := func(serverVMI *v1.VirtualMachineInstance, targetPort int) (*batchv1.Job, error) {
			By("Starting HTTP Server")
			tests.StartHTTPServer(vmi, targetPort)

			By("Getting back the VMI IP")
			vmi, err = virtClient.VirtualMachineInstance(vmi.Namespace).Get(vmi.Name, &metav1.GetOptions{})
			Expect(err).ShouldNot(HaveOccurred())
			vmiIP := vmi.Status.Interfaces[0].IP

			By("Running job to send a request to the server")
			return virtClient.BatchV1().Jobs(util.NamespaceTestDefault).Create(
				context.Background(),
				tests.NewHelloWorldJobHTTP(vmiIP, fmt.Sprintf("%d", targetPort)),
				metav1.CreateOptions{},
			)
		}
		BeforeEach(func() {
			tests.BeforeTestCleanup()

			virtClient, err = kubecli.GetKubevirtClient()
			util.PanicOnError(err)

			By("Create NetworkAttachmentDefinition")
			nad := generateIstioCNINetworkAttachmentDefinition()
			_, err = virtClient.NetworkClient().K8sCniCncfIoV1().NetworkAttachmentDefinitions(util.NamespaceTestDefault).Create(context.TODO(), nad, metav1.CreateOptions{})
			Expect(err).ShouldNot(HaveOccurred())

			By("Creating k8s service for the VMI")

			serviceName := fmt.Sprintf("%s-service", vmiAppSelectorValue)
			service := netservice.BuildSpec(serviceName, svcDeclaredTestPort, svcDeclaredTestPort, vmiAppSelectorKey, vmiAppSelectorValue)
			_, err = virtClient.CoreV1().Services(util.NamespaceTestDefault).Create(context.Background(), service, metav1.CreateOptions{})
			Expect(err).ShouldNot(HaveOccurred())
		})
		JustBeforeEach(func() {
			// Enable sidecar injection by setting the namespace label
			Expect(libnet.AddLabelToNamespace(virtClient, util.NamespaceTestDefault, tests.IstioInjectNamespaceLabel, "enabled")).ShouldNot(HaveOccurred())
			defer func() {
				Expect(libnet.RemoveLabelFromNamespace(virtClient, util.NamespaceTestDefault, tests.IstioInjectNamespaceLabel)).ShouldNot(HaveOccurred())
			}()

			By("Creating VMI")
			vmi = newVMIWithIstioSidecar(vmiPorts)
			vmi, err = virtClient.VirtualMachineInstance(util.NamespaceTestDefault).Create(vmi)
			Expect(err).ShouldNot(HaveOccurred())

			By("Waiting for VMI to be ready")
			tests.WaitUntilVMIReady(vmi, console.LoginToCirros)
		})
		Describe("Live Migration", func() {
			var (
				sourcePodName string
			)
			migrationCompleted := func(migration *v1.VirtualMachineInstanceMigration) error {
				migration, err := virtClient.VirtualMachineInstanceMigration(migration.Namespace).Get(migration.Name, &metav1.GetOptions{})
				if err != nil {
					return err
				}
				if migration.Status.Phase == v1.MigrationSucceeded {
					return nil
				}
				return fmt.Errorf("migration is in phase %s", migration.Status.Phase)
			}
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
				tests.SkipIfMigrationIsNotPossible()
			})
			JustBeforeEach(func() {
				sourcePodName = tests.GetVmPodName(virtClient, vmi)
				migration := tests.NewRandomMigration(vmi.Name, vmi.Namespace)
				migration, err = virtClient.VirtualMachineInstanceMigration(migration.Namespace).Create(migration)
				Expect(err).NotTo(HaveOccurred())
				Eventually(func() error {
					return migrationCompleted(migration)
				}, tests.MigrationWaitTime, time.Second).Should(Succeed(), fmt.Sprintf(" migration should succeed"))
			})
			It("All containers should complete in source virt-launcher pod after migration", func() {
				Eventually(func() error {
					return allContainersCompleted(sourcePodName)
				}, tests.ContainerCompletionWaitTime, time.Second).Should(Succeed(), fmt.Sprintf("all containers should complete in source virt-launcher pod"))
			})
		})
		Describe("Inbound traffic", func() {
			checkVMIReachability := func(vmi *v1.VirtualMachineInstance, targetPort int) error {
				job, err := createJobCheckingVMIReachability(vmi, targetPort)
				if err != nil {
					return err
				}
				By("Waiting for the job to succeed")
				return tests.WaitForJobToSucceed(job, 480*time.Second)
			}

			Context("With VMI having explicit ports specified", func() {
				BeforeEach(func() {
					vmiPorts = explicitPorts
				})
				table.DescribeTable("request to VMI should reach HTTP server", func(targetPort int) {
					Expect(checkVMIReachability(vmi, targetPort)).To(Succeed())
				},
					table.Entry("on service declared port on VMI with explicit ports", svcDeclaredTestPort),
					table.Entry("on service undeclared port on VMI with explicit ports", svcUndeclaredTestPort),
				)
			})
			Context("With VMI having no explicit ports specified", func() {
				BeforeEach(func() {
					vmiPorts = []v1.Port{}
				})
				table.DescribeTable("request to VMI should reach HTTP server", func(targetPort int) {
					Expect(checkVMIReachability(vmi, targetPort)).To(Succeed())
				},
					table.Entry("on service declared port on VMI with no explicit ports", svcDeclaredTestPort),
					table.Entry("on service undeclared port on VMI with no explicit ports", svcUndeclaredTestPort),
				)
				It("Should not be able to reach service running on Istio restricted port", func() {
					Expect(checkVMIReachability(vmi, istioRestrictedPort)).NotTo(Succeed())
				})
			})

			Context("With PeerAuthentication enforcing mTLS", func() {
				BeforeEach(func() {
					peerAuthenticationRes := schema.GroupVersionResource{Group: "security.istio.io", Version: istioApiVersion, Resource: "peerauthentications"}
					peerAuthentication := generateStrictPeerAuthentication()
					_, err = virtClient.DynamicClient().Resource(peerAuthenticationRes).Namespace(util.NamespaceTestDefault).Create(context.Background(), peerAuthentication, metav1.CreateOptions{})
					Expect(err).ShouldNot(HaveOccurred())
				})
				Context("With VMI having explicit ports specified", func() {
					BeforeEach(func() {
						vmiPorts = explicitPorts
					})
					table.DescribeTable("client outside mesh should NOT reach VMI HTTP server", func(targetPort int) {
						Expect(checkVMIReachability(vmi, targetPort)).NotTo(Succeed())
					},
						table.Entry("on service declared port on VMI with explicit ports", svcDeclaredTestPort),
						table.Entry("on service undeclared port on VMI with explicit ports", svcUndeclaredTestPort),
					)
				})
				Context("With VMI having no explicit ports specified", func() {
					BeforeEach(func() {
						vmiPorts = []v1.Port{}
					})
					table.DescribeTable("client outside mesh should NOT reach VMI HTTP server", func(targetPort int) {
						Expect(checkVMIReachability(vmi, targetPort)).NotTo(Succeed())
					},
						table.Entry("on service declared port on VMI with no explicit ports", svcDeclaredTestPort),
						table.Entry("on service undeclared port on VMI with no explicit ports", svcUndeclaredTestPort),
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
				serverVMI = libvmi.NewCirros(
					libvmi.WithNetwork(v1.DefaultPodNetwork()),
					libvmi.WithInterface(libvmi.InterfaceDeviceWithMasqueradeBinding([]v1.Port{}...)),
					libvmi.WithLabel("version", "v1"),
					libvmi.WithLabel(vmiAppSelectorKey, vmiServerAppSelectorValue),
				)
				serverVMI, err = virtClient.VirtualMachineInstance(util.NamespaceTestDefault).Create(serverVMI)
				Expect(err).ToNot(HaveOccurred())

				serverVMIService := netservice.BuildSpec("vmi-server", vmiServerTestPort, vmiServerTestPort, vmiAppSelectorKey, vmiServerAppSelectorValue)
				_, err = virtClient.CoreV1().Services(util.NamespaceTestDefault).Create(context.Background(), serverVMIService, metav1.CreateOptions{})
				Expect(err).ToNot(HaveOccurred())

				Expect(console.LoginToCirros(serverVMI)).To(Succeed())
				By("Starting HTTP Server")
				tests.StartHTTPServer(serverVMI, vmiServerTestPort)

				By("Creating Istio VirtualService")
				virtualServicesRes := schema.GroupVersionResource{Group: networkingIstioIO, Version: istioApiVersion, Resource: "virtualservices"}
				virtualService := generateVirtualService()
				_, err = virtClient.DynamicClient().Resource(virtualServicesRes).Namespace(util.NamespaceTestDefault).Create(context.TODO(), virtualService, metav1.CreateOptions{})
				Expect(err).ToNot(HaveOccurred())

				By("Creating Istio DestinationRule")
				destinationRulesRes := schema.GroupVersionResource{Group: networkingIstioIO, Version: istioApiVersion, Resource: "destinationrules"}
				destinationRule := generateDestinationRule()
				_, err = virtClient.DynamicClient().Resource(destinationRulesRes).Namespace(util.NamespaceTestDefault).Create(context.TODO(), destinationRule, metav1.CreateOptions{})
				Expect(err).ToNot(HaveOccurred())

				By("Creating Istio Gateway")
				gatewaysRes := schema.GroupVersionResource{Group: networkingIstioIO, Version: istioApiVersion, Resource: "gateways"}
				gateway := generateGateway()
				_, err = virtClient.DynamicClient().Resource(gatewaysRes).Namespace(util.NamespaceTestDefault).Create(context.TODO(), gateway, metav1.CreateOptions{})
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
					_, err = virtClient.DynamicClient().Resource(sidecarRes).Namespace(util.NamespaceTestDefault).Create(context.TODO(), registryOnlySidecar, metav1.CreateOptions{})
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
})

func istioServiceMeshDeployed() bool {
	value := os.Getenv(istioDeployedEnvVariable)
	if strings.ToLower(value) == "true" {
		return true
	}
	return false
}

func newVMIWithIstioSidecar(ports []v1.Port) *v1.VirtualMachineInstance {
	vmi := libvmi.NewCirros(
		libvmi.WithNetwork(v1.DefaultPodNetwork()),
		libvmi.WithInterface(libvmi.InterfaceDeviceWithMasqueradeBinding(ports...)),
		libvmi.WithLabel(vmiAppSelectorKey, vmiAppSelectorValue),
		libvmi.WithAnnotation(istio.ISTIO_INJECT_ANNOTATION, "true"),
	)
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
