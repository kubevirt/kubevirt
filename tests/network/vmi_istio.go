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
	"crypto/rand"
	"fmt"
	"regexp"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	expect "github.com/google/goexpect"
	batchv1 "k8s.io/api/batch/v1"
	k8sv1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"

	v1 "kubevirt.io/api/core/v1"
	"kubevirt.io/client-go/kubecli"

	"kubevirt.io/kubevirt/pkg/libvmi"
	libvmici "kubevirt.io/kubevirt/pkg/libvmi/cloudinit"
	"kubevirt.io/kubevirt/pkg/network/istio"
	"kubevirt.io/kubevirt/tests/console"
	"kubevirt.io/kubevirt/tests/decorators"
	"kubevirt.io/kubevirt/tests/flags"
	"kubevirt.io/kubevirt/tests/framework/kubevirt"
	"kubevirt.io/kubevirt/tests/libmigration"
	"kubevirt.io/kubevirt/tests/libnamespace"
	"kubevirt.io/kubevirt/tests/libnet"
	"kubevirt.io/kubevirt/tests/libnet/cloudinit"
	"kubevirt.io/kubevirt/tests/libnet/job"
	netservice "kubevirt.io/kubevirt/tests/libnet/service"
	"kubevirt.io/kubevirt/tests/libnet/vmnetserver"
	"kubevirt.io/kubevirt/tests/libpod"
	"kubevirt.io/kubevirt/tests/libvmifact"
	"kubevirt.io/kubevirt/tests/libvmops"
	"kubevirt.io/kubevirt/tests/libwait"
	"kubevirt.io/kubevirt/tests/testsuite"
)

const (
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
		namespace  string
		virtClient kubecli.KubevirtClient
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
	})

	Context("Virtual Machine with istio supported interface", func() {
		createJobCheckingVMIReachability := func(serverVMI *v1.VirtualMachineInstance, targetPort int) (*batchv1.Job, error) {
			By("Verifying the guest HTTP listener")
			expectIstioHTTPServerReady(serverVMI, targetPort)

			By("Getting back the VMI IP")
			serverVMI, err := virtClient.VirtualMachineInstance(serverVMI.Namespace).Get(context.Background(), serverVMI.Name, metav1.GetOptions{})
			Expect(err).ShouldNot(HaveOccurred())
			vmiIP := libnet.GetVmiPrimaryIPByFamily(serverVMI, k8sv1.IPv4Protocol)

			By("Running job to send a request to the server")
			check := fmt.Sprintf(
				`if response=$(curl --fail --silent --show-error --max-time 5 http://%s:%d) && [ "$response" = 'Hello World!' ]; then exit 0; else exit 1; fi`,
				libnet.FormatIPForURL(vmiIP), targetPort,
			)
			return virtClient.BatchV1().Jobs(namespace).Create(
				context.Background(),
				job.NewHelloWorldJob(check),
				metav1.CreateOptions{},
			)
		}
		setupIstioResources := func() {
			libnet.SkipWhenClusterNotSupportIpv4()

			By("Create NetworkAttachmentDefinition")
			nad := libnet.NewNetAttachDef("istio-cni", "")
			_, err = libnet.CreateNetAttachDef(context.Background(), namespace, nad)
			Expect(err).ShouldNot(HaveOccurred())

			By("Creating k8s service for the VMI")

			virtClient = kubevirt.Client()
			serviceName := fmt.Sprintf("%s-service", vmiAppSelectorValue)
			service := netservice.BuildSpec(serviceName, svcDeclaredTestPort, svcDeclaredTestPort, vmiAppSelectorKey, vmiAppSelectorValue)
			_, err = virtClient.CoreV1().Services(namespace).Create(context.Background(), service, metav1.CreateOptions{})
			Expect(err).ShouldNot(HaveOccurred())
		}
		startIstioVMI := func(ports []v1.Port) *v1.VirtualMachineInstance {
			// Enable sidecar injection by setting the namespace label
			Expect(libnamespace.AddLabelToNamespace(virtClient, namespace, istioInjectNamespaceLabel, "enabled")).ShouldNot(HaveOccurred())
			defer func() {
				Expect(libnamespace.RemoveLabelFromNamespace(virtClient, namespace, istioInjectNamespaceLabel)).ShouldNot(HaveOccurred())
			}()

			By("Creating VMI")
			istioVMI, err := newVMIWithIstioSidecar(ports, vmType)
			Expect(err).ShouldNot(HaveOccurred())
			istioVMI, err = virtClient.VirtualMachineInstance(namespace).Create(context.Background(), istioVMI, metav1.CreateOptions{})
			Expect(err).ShouldNot(HaveOccurred())

			By("Waiting for VMI to be ready")
			libwait.WaitUntilVMIReady(istioVMI, console.LoginToAlpine)
			return istioVMI
		}
		Describe("Live Migration", decorators.RequiresTwoSchedulableNodes, func() {
			var vmi *v1.VirtualMachineInstance
			BeforeEach(setupIstioResources)
			JustBeforeEach(func() {
				vmi = startIstioVMI(nil)
			})

			var sourcePodName string
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
			JustBeforeEach(func() {
				sourcePod, err := libpod.GetPodByVirtualMachineInstance(vmi, vmi.Namespace)
				Expect(err).ToNot(HaveOccurred())
				sourcePodName = sourcePod.Name

				migration := libmigration.New(vmi.Name, vmi.Namespace)
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
			BeforeEach(setupIstioResources)
			var vmi, bastionVMI *v1.VirtualMachineInstance
			sshCommand := func(user, ipAddress string) string {
				return fmt.Sprintf("ssh -y %s@%s\n", user, ipAddress)
			}
			checkSSHConnection := func(vmi *v1.VirtualMachineInstance, vmiAddress string) error {
				user := "doesntmatter"
				// Can't use SafeExpectBatch here because the last command never returns to prompt
				return console.ExpectBatch(vmi, []expect.Batcher{
					&expect.BSnd{S: "\n"},
					&expect.BExp{R: console.PromptExpression},
					&expect.BSnd{S: sshCommand(user, vmiAddress)},
					&expect.BExp{R: fmt.Sprintf("%s@%s's password: ", user, vmiAddress)},
				}, 60*time.Second)
			}

			BeforeEach(func() {

				bastionVMI = libvmifact.NewCirros(
					libvmi.WithNetwork(v1.DefaultPodNetwork()),
					libvmi.WithInterface(libvmi.NewInterface(
						v1.DefaultPodNetwork().Name, libvmi.WithMasqueradeBinding())),
				)

				bastionVMI, err = virtClient.VirtualMachineInstance(namespace).Create(context.Background(), bastionVMI, metav1.CreateOptions{})
				Expect(err).ToNot(HaveOccurred())
				bastionVMI = libwait.WaitUntilVMIReady(bastionVMI, console.LoginToCirros)
			})
			Context("With VMI having explicit ports specified", func() {
				JustBeforeEach(func() {
					vmi = startIstioVMI(explicitPorts)
				})
				It("should ssh to VMI with Istio proxy", func() {
					By("Getting the VMI IP")
					vmi, err = virtClient.VirtualMachineInstance(vmi.Namespace).Get(context.Background(), vmi.Name, metav1.GetOptions{})
					Expect(err).ShouldNot(HaveOccurred())
					vmiIP := libnet.GetVmiPrimaryIPByFamily(vmi, k8sv1.IPv4Protocol)

					Expect(
						checkSSHConnection(bastionVMI, vmiIP),
					).Should(Succeed())
				})
			})
			Context("With VMI having no explicit ports specified", func() {
				JustBeforeEach(func() {
					vmi = startIstioVMI(nil)
				})
				It("should ssh to VMI with Istio proxy", func() {
					By("Getting the VMI IP")
					vmi, err = virtClient.VirtualMachineInstance(vmi.Namespace).Get(context.Background(), vmi.Name, metav1.GetOptions{})
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

			expectVMIUnreachable := func(vmi *v1.VirtualMachineInstance, targetPort int) {
				httpJob, err := createJobCheckingVMIReachability(vmi, targetPort)
				Expect(err).ToNot(HaveOccurred())
				Expect(job.WaitForJobToFail(httpJob, 480*time.Second)).To(Succeed())
				expectIstioHTTPJobRejected(httpJob)
			}

			Context("With VMI having explicit ports specified", Ordered, ContinueOnFailure, decorators.OncePerOrderedCleanup, func() {
				var vmi *v1.VirtualMachineInstance

				BeforeAll(func() {
					setupIstioResources()
					vmi = startIstioVMI(explicitPorts)
					vmnetserver.StartPythonHTTPServer(vmi, svcDeclaredTestPort)
					vmnetserver.StartPythonHTTPServer(vmi, svcUndeclaredTestPort)
				})
				DescribeTable("request to VMI should reach HTTP server", func(targetPort int) {
					Expect(checkVMIReachability(vmi, targetPort)).To(Succeed())
				},
					Entry("on service declared port on VMI with explicit ports", svcDeclaredTestPort),
					Entry("on service undeclared port on VMI with explicit ports", svcUndeclaredTestPort),
				)
			})
			Context("With VMI having no explicit ports specified", Ordered, ContinueOnFailure, decorators.OncePerOrderedCleanup, func() {
				var vmi *v1.VirtualMachineInstance

				BeforeAll(func() {
					setupIstioResources()
					vmi = startIstioVMI([]v1.Port{})
					vmnetserver.StartPythonHTTPServer(vmi, svcDeclaredTestPort)
					vmnetserver.StartPythonHTTPServer(vmi, svcUndeclaredTestPort)
					vmnetserver.StartPythonHTTPServer(vmi, istioRestrictedPort)
				})
				DescribeTable("request to VMI should reach HTTP server", func(targetPort int) {
					Expect(checkVMIReachability(vmi, targetPort)).To(Succeed())
				},
					Entry("on service declared port on VMI with no explicit ports", svcDeclaredTestPort),
					Entry("on service undeclared port on VMI with no explicit ports", svcUndeclaredTestPort),
				)
				It("Should not be able to reach service running on Istio restricted port", func() {
					expectVMIUnreachable(vmi, istioRestrictedPort)
				})
			})

			Context("With PeerAuthentication enforcing mTLS", func() {
				createStrictPeerAuthentication := func() {
					peerAuthenticationRes := schema.GroupVersionResource{Group: "security.istio.io", Version: istioApiVersion, Resource: "peerauthentications"}
					peerAuthentication := generateStrictPeerAuthentication()
					_, err = virtClient.DynamicClient().Resource(peerAuthenticationRes).Namespace(namespace).Create(context.Background(), peerAuthentication, metav1.CreateOptions{})
					Expect(err).ShouldNot(HaveOccurred())
				}
				Context("With VMI having explicit ports specified", Ordered, ContinueOnFailure, decorators.OncePerOrderedCleanup, func() {
					var vmi *v1.VirtualMachineInstance

					BeforeAll(func() {
						setupIstioResources()
						createStrictPeerAuthentication()
						vmi = startIstioVMI(explicitPorts)
						vmnetserver.StartPythonHTTPServer(vmi, svcDeclaredTestPort)
						vmnetserver.StartPythonHTTPServer(vmi, svcUndeclaredTestPort)
					})
					DescribeTable("client outside mesh should NOT reach VMI HTTP server", func(targetPort int) {
						expectVMIUnreachable(vmi, targetPort)
					},
						Entry("on service declared port on VMI with explicit ports", svcDeclaredTestPort),
						Entry("on service undeclared port on VMI with explicit ports", svcUndeclaredTestPort),
					)
				})
				Context("With VMI having no explicit ports specified", Ordered, ContinueOnFailure, decorators.OncePerOrderedCleanup, func() {
					var vmi *v1.VirtualMachineInstance

					BeforeAll(func() {
						setupIstioResources()
						createStrictPeerAuthentication()
						vmi = startIstioVMI([]v1.Port{})
						vmnetserver.StartPythonHTTPServer(vmi, svcDeclaredTestPort)
						vmnetserver.StartPythonHTTPServer(vmi, svcUndeclaredTestPort)
					})
					DescribeTable("client outside mesh should NOT reach VMI HTTP server", func(targetPort int) {
						expectVMIUnreachable(vmi, targetPort)
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
			startExternalService := func() (*v1.VirtualMachineInstance, string) {
				networkData := cloudinit.CreateDefaultCloudInitNetworkData()

				serverVMI := libvmifact.NewAlpineWithTestTooling(
					libvmi.WithNetwork(v1.DefaultPodNetwork()),
					libvmi.WithInterface(libvmi.NewInterface(
						v1.DefaultPodNetwork().Name, libvmi.WithMasqueradeBinding())),
					libvmi.WithAnnotation(istio.InjectSidecarAnnotation, "false"),
					libvmi.WithLabel("version", "v1"),
					libvmi.WithLabel(vmiAppSelectorKey, vmiServerAppSelectorValue),
					libvmi.WithCloudInitNoCloud(libvmici.WithNoCloudNetworkData(networkData)),
					libvmi.WithNamespace(namespace),
				)
				By("Starting VirtualMachineInstance")
				serverVMI = libvmops.RunVMIAndExpectLaunch(serverVMI, flags.StartupTimeoutSecondsHuge())

				serverVMIService := netservice.BuildSpec("vmi-server", vmiServerTestPort, vmiServerTestPort, vmiAppSelectorKey, vmiServerAppSelectorValue)
				_, err = virtClient.CoreV1().Services(namespace).Create(context.Background(), serverVMIService, metav1.CreateOptions{})
				Expect(err).ToNot(HaveOccurred())

				By("Starting HTTP Server")
				Expect(console.LoginToAlpine(serverVMI)).To(Succeed())
				vmnetserver.StartPythonHTTPServer(serverVMI, vmiServerTestPort)

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
				return serverVMI, ingressGatewayService.Spec.ClusterIP
			}

			checkHTTPServiceReturnCode := func(requestVMI *v1.VirtualMachineInstance, ingressGatewayAddress, returnCode string) error {
				command := fmt.Sprintf(
					`code=$(curl --silent --show-error --max-time 2 -o /dev/null -w '%%{http_code}' -Hhost:%s.example.com http://%s); result=$?; echo "HTTP status: $code"; test "$result" -eq 0 && echo "$code" | grep -Eq '^%s$'`,
					vmiServerHostName, ingressGatewayAddress, returnCode,
				)
				return runIstioCommand(requestVMI, command, externalServiceCheckTimeout)
			}

			expectExternalServiceReachable := func(requestVMI, serverVMI *v1.VirtualMachineInstance, ingressGatewayServiceIP string) {
				Expect(checkHTTPServiceReturnCode(serverVMI, ingressGatewayServiceIP, "200")).To(Succeed())
				command := fmt.Sprintf(
					`test "$(curl --fail --silent --show-error --max-time 2 http://%s.%s.svc.cluster.local:%d)" = 'Hello World!'`,
					vmiServerHostName, namespace, vmiServerTestPort,
				)
				Expect(runIstioCommand(requestVMI, command, externalServiceCheckTimeout)).To(Succeed())
			}

			Context("With default outbound traffic policy", Ordered, ContinueOnFailure, decorators.OncePerOrderedCleanup, func() {
				var explicitPortsVMI, noExplicitPortsVMI *v1.VirtualMachineInstance
				var ingressGatewayServiceIP string

				BeforeAll(func() {
					setupIstioResources()
					_, ingressGatewayServiceIP = startExternalService()
					explicitPortsVMI = startIstioVMI(explicitPorts)
					noExplicitPortsVMI = startIstioVMI(nil)
				})

				Context("VMI with explicit ports", func() {
					It("Should be able to reach http server outside of mesh", func() {
						Expect(checkHTTPServiceReturnCode(explicitPortsVMI, ingressGatewayServiceIP, "200")).To(Succeed())
					})
				})
				Context("VMI with no explicit ports", func() {
					It("Should be able to reach http server outside of mesh", func() {
						Expect(checkHTTPServiceReturnCode(noExplicitPortsVMI, ingressGatewayServiceIP, "200")).To(Succeed())
					})
				})
			})

			Context("With Sidecar allowing only registered external services", Ordered, ContinueOnFailure, decorators.OncePerOrderedCleanup, func() {
				var explicitPortsVMI, noExplicitPortsVMI, serverVMI *v1.VirtualMachineInstance
				var ingressGatewayServiceIP string

				BeforeAll(func() {
					setupIstioResources()
					serverVMI, ingressGatewayServiceIP = startExternalService()

					sidecarRes := schema.GroupVersionResource{Group: networkingIstioIO, Version: istioApiVersion, Resource: "sidecars"}
					registryOnlySidecar := generateRegistryOnlySidecar()
					_, err = virtClient.DynamicClient().Resource(sidecarRes).Namespace(namespace).Create(context.TODO(), registryOnlySidecar, metav1.CreateOptions{})
					Expect(err).ToNot(HaveOccurred())

					explicitPortsVMI = startIstioVMI(explicitPorts)
					noExplicitPortsVMI = startIstioVMI(nil)
				})

				Context("VMI with explicit ports", func() {
					It("Should not be able to reach http service outside of mesh", func() {
						expectExternalServiceReachable(explicitPortsVMI, serverVMI, ingressGatewayServiceIP)
						// Policy propagation can briefly allow a request after startup.
						Eventually(func() error {
							return checkHTTPServiceReturnCode(explicitPortsVMI, ingressGatewayServiceIP, "5[0-9][0-9]")
						}, externalServiceCheckTimeout, externalServiceCheckInterval).Should(Succeed())
						expectExternalServiceReachable(explicitPortsVMI, serverVMI, ingressGatewayServiceIP)
					})
				})
				Context("VMI with no explicit ports", func() {
					It("Should not be able to reach http service outside of mesh", func() {
						expectExternalServiceReachable(noExplicitPortsVMI, serverVMI, ingressGatewayServiceIP)
						Eventually(func() error {
							return checkHTTPServiceReturnCode(noExplicitPortsVMI, ingressGatewayServiceIP, "5[0-9][0-9]")
						}, externalServiceCheckTimeout, externalServiceCheckInterval).Should(Succeed())
						expectExternalServiceReachable(noExplicitPortsVMI, serverVMI, ingressGatewayServiceIP)
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

var _ = Describe(SIG(" Istio with masquerade binding", decorators.Istio, Serial, istioTestsWithMasqueradeBinding))

var _ = Describe(SIG(" Istio with passt binding", decorators.Istio, Serial, istioTestsWithPasstBinding))

func runIstioCommand(vmi *v1.VirtualMachineInstance, command string, timeout time.Duration) error {
	marker := "ISTIO_COMMAND_RESULT_" + rand.Text() + ":"
	responses, err := console.SafeExpectBatchWithResponse(vmi, []expect.Batcher{
		&expect.BSnd{S: command + "; printf '\\n" + marker + "%s\\n' \"$?\"\n"},
		&expect.BExp{R: console.RetValue(marker + "[0-9]+")},
	}, int(timeout/time.Second))
	if err != nil {
		return err
	}

	results := regexp.MustCompile(`(?m)^`+regexp.QuoteMeta(marker)+`([0-9]+)\r?$`).FindAllStringSubmatch(responses[0].Match[0], -1)
	if len(results) == 0 || results[len(results)-1][1] != "0" {
		return fmt.Errorf("command failed at VMI %s: %s", vmi.Name, responses[0].Match[0])
	}
	return nil
}

// The loopback request verifies the guest listener without passing through Envoy.
func expectIstioHTTPServerReady(vmi *v1.VirtualMachineInstance, port int) {
	command := fmt.Sprintf(`test "$(curl --fail --silent --show-error --max-time 2 http://127.0.0.1:%d)" = 'Hello World!'`, port)
	Eventually(func() error {
		return runIstioCommand(vmi, command, 5*time.Second)
	}, 5*time.Second, time.Second).Should(Succeed())
}

func expectIstioHTTPJobRejected(httpJob *batchv1.Job) {
	virtClient := kubevirt.Client()
	failedJob, err := virtClient.BatchV1().Jobs(httpJob.Namespace).Get(context.Background(), httpJob.Name, metav1.GetOptions{})
	Expect(err).ToNot(HaveOccurred())
	Expect(failedJob.Status.Conditions).To(ContainElement(SatisfyAll(
		HaveField("Type", Equal(batchv1.JobFailed)),
		HaveField("Status", Equal(k8sv1.ConditionTrue)),
		HaveField("Reason", Equal(batchv1.JobReasonBackoffLimitExceeded)),
	)))

	pods, err := virtClient.CoreV1().Pods(httpJob.Namespace).List(context.Background(), metav1.ListOptions{
		LabelSelector: fmt.Sprintf("%s=%s", batchv1.ControllerUidLabel, httpJob.UID),
	})
	Expect(err).ToNot(HaveOccurred())
	Expect(pods.Items).ToNot(BeEmpty())
	for _, pod := range pods.Items {
		Expect(pod.Status.ContainerStatuses).To(ContainElement(SatisfyAll(
			HaveField("Name", Equal("container")),
			HaveField("State.Terminated.ExitCode", Equal(int32(1))),
		)))
	}
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

const enablePasswordAuth = "#cloud-config\nssh_pwauth: true\n"

func createMasqueradeVm(ports []v1.Port) *v1.VirtualMachineInstance {
	networkData := cloudinit.CreateDefaultCloudInitNetworkData()
	vmi := libvmifact.NewAlpineWithTestTooling(
		libvmi.WithNetwork(v1.DefaultPodNetwork()),
		libvmi.WithInterface(libvmi.NewInterface(
			v1.DefaultPodNetwork().Name,
			libvmi.WithMasqueradeBinding(),
			libvmi.WithPorts(ports...),
		)),
		libvmi.WithLabel(vmiAppSelectorKey, vmiAppSelectorValue),
		libvmi.WithAnnotation(istio.InjectSidecarAnnotation, "true"),
		libvmi.WithCloudInitNoCloud(
			libvmici.WithNoCloudNetworkData(networkData),
			libvmici.WithNoCloudEncodedUserData(enablePasswordAuth),
		),
	)
	return vmi
}

func createPasstVm(ports []v1.Port) *v1.VirtualMachineInstance {
	passtIface := libvmi.NewInterface(
		v1.DefaultPodNetwork().Name, libvmi.WithPasstBinding(), libvmi.WithPorts(ports...))
	vmi := libvmifact.NewAlpineWithTestTooling(
		libvmi.WithNetwork(v1.DefaultPodNetwork()),
		libvmi.WithInterface(passtIface),
		libvmi.WithLabel(vmiAppSelectorKey, vmiAppSelectorValue),
		libvmi.WithAnnotation(istio.InjectSidecarAnnotation, "true"),
		libvmi.WithCloudInitNoCloud(libvmici.WithNoCloudEncodedUserData(enablePasswordAuth)),
	)
	return vmi
}

func generateStrictPeerAuthentication() *unstructured.Unstructured {
	return &unstructured.Unstructured{
		Object: map[string]any{
			"apiVersion": fmt.Sprintf("%s/%s", securityIstioIO, istioApiVersion),
			"kind":       "PeerAuthentication",
			"metadata": map[string]any{
				"name": "strict-pa",
			},
			"spec": map[string]any{
				"mtls": map[string]any{
					"mode": "STRICT",
				},
			},
		},
	}
}

func generateRegistryOnlySidecar() *unstructured.Unstructured {
	return &unstructured.Unstructured{
		Object: map[string]any{
			"apiVersion": fmt.Sprintf("%s/%s", networkingIstioIO, istioApiVersion),
			"kind":       "Sidecar",
			"metadata": map[string]any{
				"name": "registry-only-sidecar",
			},
			"spec": map[string]any{
				"outboundTrafficPolicy": map[string]any{
					"mode": "REGISTRY_ONLY",
				},
			},
		},
	}
}

func generateVirtualService() *unstructured.Unstructured {
	return &unstructured.Unstructured{
		Object: map[string]any{
			"apiVersion": fmt.Sprintf("%s/%s", networkingIstioIO, istioApiVersion),
			"kind":       "VirtualService",
			"metadata": map[string]any{
				"name": "vmi-server-vs",
			},
			"spec": map[string]any{
				"gateways": []string{
					vmiServerGateway,
				},
				"hosts": []string{
					fmt.Sprintf("%s.example.com", vmiServerHostName),
				},
				"http": []any{
					map[string]any{
						"match": []map[string]any{
							{
								"uri": map[string]any{
									"prefix": "/",
								},
							},
						},
						"route": []map[string]any{
							{
								"destination": map[string]any{
									"port": map[string]any{
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
		Object: map[string]any{
			"apiVersion": fmt.Sprintf("%s/%s", networkingIstioIO, istioApiVersion),
			"kind":       "DestinationRule",
			"metadata": map[string]any{
				"name": "vmi-server-dr",
			},
			"spec": map[string]any{
				"host": vmiServerHostName,
				"subsets": []map[string]any{
					{
						"name": "v1",
						"labels": map[string]any{
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
		Object: map[string]any{
			"apiVersion": fmt.Sprintf("%s/%s", networkingIstioIO, istioApiVersion),
			"kind":       "Gateway",
			"metadata": map[string]any{
				"name": vmiServerGateway,
			},
			"spec": map[string]any{
				"selector": map[string]any{
					"istio": "ingressgateway",
				},
				"servers": []map[string]any{
					{
						"port": map[string]any{
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
