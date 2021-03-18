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
 * Copyright 2017 Red Hat, Inc.
 *
 */

package tests_test

import (
	"context"
	"crypto/tls"
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"net/http"
	neturl "net/url"
	"reflect"
	"sort"
	"strconv"
	"strings"
	"time"

	expect "github.com/google/goexpect"
	. "github.com/onsi/ginkgo"
	"github.com/onsi/ginkgo/extensions/table"
	. "github.com/onsi/gomega"
	gomegatypes "github.com/onsi/gomega/types"
	aggregatorclient "k8s.io/kube-aggregator/pkg/client/clientset_generated/clientset"
	netutils "k8s.io/utils/net"

	k8sv1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/fields"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/client-go/tools/leaderelection/resourcelock"
	"k8s.io/client-go/util/retry"

	v1ext "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	extclient "k8s.io/apiextensions-apiserver/pkg/client/clientset/clientset"

	v1 "kubevirt.io/client-go/api/v1"
	"kubevirt.io/client-go/kubecli"
	clusterutil "kubevirt.io/kubevirt/pkg/util/cluster"
	"kubevirt.io/kubevirt/pkg/virt-controller/leaderelectionconfig"
	"kubevirt.io/kubevirt/pkg/virt-operator/resource/generate/components"
	crds "kubevirt.io/kubevirt/pkg/virt-operator/resource/generate/components"
	"kubevirt.io/kubevirt/tests"
	"kubevirt.io/kubevirt/tests/console"
	cd "kubevirt.io/kubevirt/tests/containerdisk"
	"kubevirt.io/kubevirt/tests/flags"
	"kubevirt.io/kubevirt/tests/libnet"
)

var _ = FDescribe("[Serial]Infrastructure", func() {
	var (
		virtClient       kubecli.KubevirtClient
		aggregatorClient *aggregatorclient.Clientset
		err              error
	)
	BeforeEach(func() {
		virtClient, err = kubecli.GetKubevirtClient()
		tests.PanicOnError(err)

		if aggregatorClient == nil {
			config, err := kubecli.GetConfig()
			if err != nil {
				panic(err)
			}

			aggregatorClient = aggregatorclient.NewForConfigOrDie(config)
		}
	})

	Describe("CRDs", func() {
		It("[test_id:5177]Should have structural schema", func() {
			ourCRDs := []string{crds.VIRTUALMACHINE, crds.VIRTUALMACHINEINSTANCE, crds.VIRTUALMACHINEINSTANCEPRESET,
				crds.VIRTUALMACHINEINSTANCEREPLICASET, crds.VIRTUALMACHINEINSTANCEMIGRATION, crds.KUBEVIRT,
				crds.VIRTUALMACHINESNAPSHOT, crds.VIRTUALMACHINESNAPSHOTCONTENT,
			}

			for _, name := range ourCRDs {
				ext, err := extclient.NewForConfig(virtClient.Config())
				Expect(err).ToNot(HaveOccurred())

				crd, err := ext.ApiextensionsV1().CustomResourceDefinitions().Get(context.Background(), name, metav1.GetOptions{})
				Expect(err).ToNot(HaveOccurred())

				for _, condition := range crd.Status.Conditions {
					if condition.Type == v1ext.NonStructuralSchema {
						Expect(condition.Status).NotTo(BeTrue())
					}
				}
			}
		})
	})

	Describe("[rfe_id:4102][crit:medium][vendor:cnv-qe@redhat.com][level:component]certificates", func() {

		BeforeEach(func() {
			tests.BeforeTestCleanup()
		})

		It("[test_id:4099] should be rotated when a new CA is created", func() {
			By("checking that the config-map gets the new CA bundle attached")
			Eventually(func() int {
				_, crts := tests.GetBundleFromConfigMap(components.KubeVirtCASecretName)
				return len(crts)
			}, 10*time.Second, 1*time.Second).Should(BeNumerically(">", 0))

			By("destroying the certificate")
			secret, err := virtClient.CoreV1().Secrets(flags.KubeVirtInstallNamespace).Get(context.Background(), components.KubeVirtCASecretName, metav1.GetOptions{})
			Expect(err).ToNot(HaveOccurred())
			secret.Data = map[string][]byte{
				"random": []byte("nonsense"),
			}
			_, err = virtClient.CoreV1().Secrets(flags.KubeVirtInstallNamespace).Update(context.Background(), secret, metav1.UpdateOptions{})
			Expect(err).ToNot(HaveOccurred())

			By("checking that the CA secret gets restored with a new ca bundle")
			var newCA []byte
			Eventually(func() []byte {
				newCA = tests.GetCertFromSecret(components.KubeVirtCASecretName)
				return newCA
			}, 10*time.Second, 1*time.Second).Should(Not(BeEmpty()))

			By("checking that one of the CAs in the config-map is the new one")
			var caBundle []byte
			Eventually(func() bool {
				caBundle, _ = tests.GetBundleFromConfigMap(components.KubeVirtCASecretName)
				return tests.ContainsCrt(caBundle, newCA)
			}, 10*time.Second, 1*time.Second).Should(BeTrue(), "the new CA should be added to the config-map")

			By("checking that the ca bundle gets propagated to the validating webhook")
			Eventually(func() bool {
				webhook, err := virtClient.AdmissionregistrationV1beta1().ValidatingWebhookConfigurations().Get(context.Background(), components.VirtAPIValidatingWebhookName, metav1.GetOptions{})
				Expect(err).ToNot(HaveOccurred())
				if len(webhook.Webhooks) > 0 {
					return tests.ContainsCrt(webhook.Webhooks[0].ClientConfig.CABundle, newCA)
				}
				return false
			}, 10*time.Second, 1*time.Second).Should(BeTrue())
			By("checking that the ca bundle gets propagated to the mutating webhook")
			Eventually(func() bool {
				webhook, err := virtClient.AdmissionregistrationV1beta1().MutatingWebhookConfigurations().Get(context.Background(), components.VirtAPIMutatingWebhookName, metav1.GetOptions{})
				Expect(err).ToNot(HaveOccurred())
				if len(webhook.Webhooks) > 0 {
					return tests.ContainsCrt(webhook.Webhooks[0].ClientConfig.CABundle, newCA)
				}
				return false
			}, 10*time.Second, 1*time.Second).Should(BeTrue())

			By("checking that the ca bundle gets propagated to the apiservice")
			Eventually(func() bool {
				apiService, err := aggregatorClient.ApiregistrationV1beta1().APIServices().Get(context.Background(), fmt.Sprintf("%s.subresources.kubevirt.io", v1.ApiLatestVersion), metav1.GetOptions{})
				Expect(err).ToNot(HaveOccurred())
				return tests.ContainsCrt(apiService.Spec.CABundle, newCA)
			}, 10*time.Second, 1*time.Second).Should(BeTrue())

			By("checking that we can still start virtual machines and connect to the VMI")
			vmi := tests.NewRandomVMIWithEphemeralDisk(cd.ContainerDiskFor(cd.ContainerDiskAlpine))
			vmi = tests.RunVMI(vmi, 60)
			Expect(console.LoginToAlpine(vmi)).To(Succeed())
		})

		It("[test_id:4100] should be valid during the whole rotation process", func() {
			oldAPICert := tests.EnsurePodsCertIsSynced(fmt.Sprintf("%s=%s", v1.AppLabel, "virt-api"), flags.KubeVirtInstallNamespace, "8443")
			oldHandlerCert := tests.EnsurePodsCertIsSynced(fmt.Sprintf("%s=%s", v1.AppLabel, "virt-handler"), flags.KubeVirtInstallNamespace, "8186")
			Expect(err).ToNot(HaveOccurred())

			By("destroying the CA certificate")
			err = virtClient.CoreV1().Secrets(flags.KubeVirtInstallNamespace).Delete(context.Background(), components.KubeVirtCASecretName, metav1.DeleteOptions{})
			Expect(err).ToNot(HaveOccurred())

			By("repeatedly starting VMIs until virt-api and virt-handler certificates are updated")
			Eventually(func() (rotated bool) {
				vmi := tests.NewRandomVMIWithEphemeralDisk(cd.ContainerDiskFor(cd.ContainerDiskAlpine))
				vmi = tests.RunVMI(vmi, 60)
				Expect(console.LoginToAlpine(vmi)).To(Succeed())
				err = virtClient.VirtualMachineInstance(vmi.Namespace).Delete(vmi.Name, &metav1.DeleteOptions{})
				Expect(err).ToNot(HaveOccurred())
				newAPICert, _, err := tests.GetPodsCertIfSynced(fmt.Sprintf("%s=%s", v1.AppLabel, "virt-api"), flags.KubeVirtInstallNamespace, "8443")
				Expect(err).ToNot(HaveOccurred())
				newHandlerCert, _, err := tests.GetPodsCertIfSynced(fmt.Sprintf("%s=%s", v1.AppLabel, "virt-handler"), flags.KubeVirtInstallNamespace, "8186")
				Expect(err).ToNot(HaveOccurred())
				return !reflect.DeepEqual(oldHandlerCert, newHandlerCert) && !reflect.DeepEqual(oldAPICert, newAPICert)
			}, 120*time.Second).Should(BeTrue())
		})

		table.DescribeTable("should be rotated when deleted for ", func(secretName string) {
			By("destroying the certificate")
			err := retry.RetryOnConflict(retry.DefaultRetry, func() error {
				secret, err := virtClient.CoreV1().Secrets(flags.KubeVirtInstallNamespace).Get(context.Background(), secretName, metav1.GetOptions{})
				if err != nil {
					return err
				}
				secret.Data = map[string][]byte{
					"random": []byte("nonsense"),
				}
				_, err = virtClient.CoreV1().Secrets(flags.KubeVirtInstallNamespace).Update(context.Background(), secret, metav1.UpdateOptions{})

				return err
			})
			Expect(err).ToNot(HaveOccurred())

			By("checking that the secret gets restored with a new certificate")
			Eventually(func() []byte {
				return tests.GetCertFromSecret(secretName)
			}, 10*time.Second, 1*time.Second).Should(Not(BeEmpty()))
		},
			table.Entry("[test_id:4101] virt-operator", components.VirtOperatorCertSecretName),
			table.Entry("[test_id:4103] virt-api", components.VirtApiCertSecretName),
			table.Entry("[test_id:4104] virt-controller", components.VirtControllerCertSecretName),
			table.Entry("[test_id:4105] virt-handlers client side", components.VirtHandlerCertSecretName),
			table.Entry("[test_id:4106] virt-handlers server side", components.VirtHandlerServerCertSecretName),
		)
	})

	// start a VMI, wait for it to run and return the node it runs on
	startVMI := func(vmi *v1.VirtualMachineInstance) string {
		By("Starting a new VirtualMachineInstance")
		obj, err := virtClient.
			RestClient().
			Post().
			Resource("virtualmachineinstances").
			Namespace(tests.NamespaceTestDefault).
			Body(vmi).
			Do(context.Background()).Get()
		Expect(err).ToNot(HaveOccurred(), "Should create VMI")

		By("Waiting until the VM is ready")
		return tests.WaitForSuccessfulVMIStart(obj)
	}

	Describe("[rfe_id:4126][crit:medium][vendor:cnv-qe@redhat.com][level:component]Taints and toleration", func() {

		Context("CriticalAddonsOnly taint set on a node", func() {

			var selectedNodeName string

			BeforeEach(func() {
				selectedNodeName = ""
			})

			AfterEach(func() {
				if selectedNodeName != "" {
					By("removing the taint from the tainted node")
					err := retry.RetryOnConflict(retry.DefaultRetry, func() error {
						selectedNode, err := virtClient.CoreV1().Nodes().Get(context.Background(), selectedNodeName, metav1.GetOptions{})
						if err != nil {
							return err
						}

						var taints []k8sv1.Taint
						for _, taint := range selectedNode.Spec.Taints {
							if taint.Key != "CriticalAddonsOnly" {
								taints = append(taints, taint)
							}
						}

						nodeCopy := selectedNode.DeepCopy()
						nodeCopy.ResourceVersion = ""
						nodeCopy.Spec.Taints = taints

						_, err = virtClient.CoreV1().Nodes().Update(context.Background(), nodeCopy, metav1.UpdateOptions{})
						return err
					})
					Expect(err).ShouldNot(HaveOccurred())
				}
			})

			It("[test_id:4134] kubevirt components on that node should not evict", func() {

				By("finding all kubevirt pods")
				pods, err := virtClient.CoreV1().Pods(flags.KubeVirtInstallNamespace).List(context.Background(), metav1.ListOptions{})
				Expect(err).ShouldNot(HaveOccurred(), "failed listing kubevirt pods")
				Expect(len(pods.Items)).To(BeNumerically(">", 0), "no kubevirt pods found")

				By("finding all schedulable nodes")
				schedulableNodesList := tests.GetAllSchedulableNodes(virtClient)
				schedulableNodes := map[string]*k8sv1.Node{}
				for _, node := range schedulableNodesList.Items {
					schedulableNodes[node.Name] = node.DeepCopy()
				}

				By("selecting one compute only node that runs kubevirt components")
				// master nodes should never have the CriticalAddonsOnly taint because core components might not
				// tolerate this taint because it is meant to be used on compute nodes only. If we set this taint
				// on a master node, we risk in breaking the test cluster.
				for _, pod := range pods.Items {
					node, ok := schedulableNodes[pod.Spec.NodeName]
					if !ok {
						// Pod is running on a non-schedulable node?
						continue
					}
					if _, isMaster := node.Labels["node-role.kubernetes.io/master"]; isMaster {
						continue
					}
					selectedNodeName = node.Name
					break
				}

				// It is possible to run this test on a cluster that simply does not have worker nodes.
				// Since KubeVirt can't control that, the only correct action is to halt the test.
				if selectedNodeName == "" {
					Skip("Could nould determine a node to safely taint")
				}

				By("setting up a watch for terminated pods")
				lw, err := virtClient.CoreV1().Pods(flags.KubeVirtInstallNamespace).Watch(context.Background(), metav1.ListOptions{})
				Expect(err).ToNot(HaveOccurred())
				// in the test env, we also deploy non core-kubevirt apps
				kvCoreApps := map[string]string{
					"virt-handler":    "",
					"virt-controller": "",
					"virt-api":        "",
					"virt-operator":   "",
				}

				signalTerminatedPods := func(stopCn <-chan bool, eventsCn <-chan watch.Event, terminatedPodsCn chan<- bool) {
					for {
						select {
						case <-stopCn:
							return
						case e := <-eventsCn:
							pod, ok := e.Object.(*k8sv1.Pod)
							Expect(ok).To(BeTrue())
							if _, isCoreApp := kvCoreApps[pod.Name]; !isCoreApp {
								continue
							}
							if pod.DeletionTimestamp != nil {
								By(fmt.Sprintf("%s terminated", pod.Name))
								terminatedPodsCn <- true
								return
							}
						}
					}
				}
				stopCn := make(chan bool, 1)
				terminatedPodsCn := make(chan bool, 1)
				go signalTerminatedPods(stopCn, lw.ResultChan(), terminatedPodsCn)

				By("tainting the selected node")
				err = retry.RetryOnConflict(retry.DefaultRetry, func() error {
					selectedNode, err := virtClient.CoreV1().Nodes().Get(context.Background(), selectedNodeName, metav1.GetOptions{})
					if err != nil {
						return err
					}

					selectedNodeCopy := selectedNode.DeepCopy()
					selectedNodeCopy.Spec.Taints = append(selectedNodeCopy.Spec.Taints, k8sv1.Taint{
						Key:    "CriticalAddonsOnly",
						Value:  "",
						Effect: k8sv1.TaintEffectNoExecute,
					})

					_, err = virtClient.CoreV1().Nodes().Update(context.Background(), selectedNodeCopy, metav1.UpdateOptions{})
					return err
				})
				Expect(err).ShouldNot(HaveOccurred())

				Consistently(terminatedPodsCn, 5*time.Second).ShouldNot(Receive(), "pods should not terminate")
				stopCn <- true
			})

		})
	})

	Describe("[rfe_id:3187][crit:medium][vendor:cnv-qe@redhat.com][level:component]Prometheus scraped metrics", func() {

		/*
			This test is querying the metrics from Prometheus *after* they were
			scraped and processed by the different components on the way.
		*/

		tests.BeforeAll(func() {
			onOCP, err := clusterutil.IsOnOpenShift(virtClient)
			Expect(err).ToNot(HaveOccurred(), "failed to detect cluster type")

			if !onOCP {
				Skip("test is verifying integration with OCP's cluster monitoring stack")
			}
		})

		It("[test_id:4135]should find VMI namespace on namespace label of the metric", func() {

			/*
				This test is required because in cases of misconfigurations on
				monitoring objects (such for the ServiceMonitor), our rules will
				still be picked up by the monitoring-operator, but Prometheus
				will fail to load it.
			*/

			By("creating a VMI in a user defined namespace")
			vmi := tests.NewRandomVMIWithEphemeralDisk(
				cd.ContainerDiskFor(cd.ContainerDiskAlpine))
			startVMI(vmi)

			By("finding virt-operator pod")
			ops, err := virtClient.CoreV1().Pods(flags.KubeVirtInstallNamespace).List(context.Background(), metav1.ListOptions{LabelSelector: "kubevirt.io=virt-operator"})
			Expect(err).ToNot(HaveOccurred(), "failed to list virt-operators")
			Expect(ops.Size).ToNot(Equal(0), "no virt-operators found")
			op := ops.Items[0]
			Expect(op).ToNot(BeNil(), "virt-operator pod should not be nil")

			var ep *k8sv1.Endpoints
			By("finding Prometheus endpoint")
			Eventually(func() bool {
				ep, err = virtClient.CoreV1().Endpoints("openshift-monitoring").Get(context.Background(), "prometheus-k8s", metav1.GetOptions{})
				Expect(err).ToNot(HaveOccurred(), "failed to retrieve Prometheus endpoint")

				if len(ep.Subsets) == 0 || len(ep.Subsets[0].Addresses) == 0 {
					return false
				}
				return true
			}, 10*time.Second, time.Second).Should(BeTrue())

			promIP := ep.Subsets[0].Addresses[0].IP
			Expect(promIP).ToNot(Equal(0), "could not get Prometheus IP from endpoint")
			var promPort int32
			for _, port := range ep.Subsets[0].Ports {
				if port.Name == "web" {
					promPort = port.Port
				}
			}
			Expect(promPort).ToNot(Equal(0), "could not get Prometheus port from endpoint")

			// We need a token from a service account that can view all namespaces in the cluster
			By("extracting virt-operator sa token")
			token, _, err := tests.ExecuteCommandOnPodV2(virtClient,
				&op,
				"virt-operator",
				[]string{
					"cat",
					"/var/run/secrets/kubernetes.io/serviceaccount/token",
				})
			Expect(err).ToNot(HaveOccurred(), "failed executing command on virt-operator")
			Expect(token).ToNot(BeEmpty(), "virt-operator sa token returned empty")

			By("querying Prometheus API endpoint for a VMI exported metric")
			stdout, _, err := tests.ExecuteCommandOnPodV2(virtClient,
				&op,
				"virt-operator",
				[]string{
					"curl",
					"-L",
					"-k",
					fmt.Sprintf("https://%s:%d/api/v1/query", promIP, promPort),
					"-H",
					fmt.Sprintf("Authorization: Bearer %s", token),
					"--data-urlencode",
					fmt.Sprintf(
						`query=kubevirt_vmi_memory_resident_bytes{namespace="%s",name="%s"}`,
						vmi.Namespace,
						vmi.Name,
					),
				})
			Expect(err).ToNot(HaveOccurred(), "failed to execute query")

			// the Prometheus go-client does not export queryResult, and
			// using an HTTP client for queries would require a port-forwarding
			// since the cluster is running in a different network.
			var queryResult map[string]json.RawMessage

			err = json.Unmarshal([]byte(stdout), &queryResult)
			Expect(err).ToNot(HaveOccurred(), "failed to unmarshal query result")

			var status string
			err = json.Unmarshal(queryResult["status"], &status)
			Expect(err).ToNot(HaveOccurred(), "failed to unmarshal query status")
			Expect(status).To(Equal("success"))
		})
	})

	Describe("[rfe_id:3187][crit:medium][vendor:cnv-qe@redhat.com][level:component]Prometheus Endpoints", func() {
		var preparedVMIs []*v1.VirtualMachineInstance
		var pod *k8sv1.Pod
		var metricsIPs []string

		pinVMIOnNode := func(vmi *v1.VirtualMachineInstance, nodeName string) *v1.VirtualMachineInstance {
			if vmi == nil {
				return nil
			}
			if vmi.Spec.NodeSelector == nil {
				vmi.Spec.NodeSelector = make(map[string]string)
			}
			vmi.Spec.NodeSelector["kubernetes.io/hostname"] = nodeName
			return vmi
		}

		// returns metrics from the node the VMI(s) runs on
		getKubevirtVMMetrics := func(ip string) string {
			metricsURL := prepareMetricsURL(ip, 8443)
			stdout, _, err := tests.ExecuteCommandOnPodV2(virtClient,
				pod,
				"virt-handler",
				[]string{
					"curl",
					"-L",
					"-k",
					metricsURL,
				})
			Expect(err).ToNot(HaveOccurred())
			return stdout
		}

		// collect metrics whose key contains the given string, expects non-empty result
		collectMetrics := func(ip, metricSubstring string) map[string]float64 {
			By("Scraping the Prometheus endpoint")
			var metrics map[string]float64
			var lines []string

			Eventually(func() map[string]float64 {
				out := getKubevirtVMMetrics(ip)
				lines = takeMetricsWithPrefix(out, metricSubstring)
				metrics, err = parseMetricsToMap(lines)
				Expect(err).ToNot(HaveOccurred())
				return metrics
			}, 30*time.Second, 2*time.Second).ShouldNot(BeEmpty())

			// troubleshooting helper
			fmt.Fprintf(GinkgoWriter, "metrics [%s]:\nlines=%s\n%#v\n", metricSubstring, lines, metrics)
			Expect(len(metrics)).To(BeNumerically(">=", float64(1.0)))
			Expect(metrics).To(HaveLen(len(lines)))

			return metrics
		}

		prepareVMIForTests := func(preferredNodeName string) string {
			By("Creating the VirtualMachineInstance")

			// WARNING: we assume the VM will have a VirtIO disk (vda)
			// and we add our own vdb on which we do our test.
			// but if the default disk is not vda, the test will break
			// TODO: introspect the VMI and get the device name of this
			// block device?
			vmi := tests.NewRandomVMIWithEphemeralDisk(cd.ContainerDiskFor(cd.ContainerDiskAlpine))
			tests.AppendEmptyDisk(vmi, "testdisk", "virtio", "1Gi")

			if preferredNodeName != "" {
				pinVMIOnNode(vmi, preferredNodeName)
			}
			nodeName := startVMI(vmi)
			if preferredNodeName != "" {
				Expect(nodeName).To(Equal(preferredNodeName), "Should run VMIs on the same node")
			}

			By("Expecting the VirtualMachineInstance console")
			// This also serves as a sync point to make sure the VM completed the boot
			// (and reduce the risk of false negatives)
			Expect(console.LoginToAlpine(vmi)).To(Succeed())

			By("Writing some data to the disk")
			Expect(console.SafeExpectBatch(vmi, []expect.Batcher{
				&expect.BSnd{S: "dd if=/dev/zero of=/dev/vdb bs=1M count=1\n"},
				&expect.BExp{R: console.PromptExpression},
				&expect.BSnd{S: "sync\n"},
				&expect.BExp{R: console.PromptExpression},
			}, 10)).To(Succeed())

			preparedVMIs = append(preparedVMIs, vmi)
			return nodeName
		}

		tests.BeforeAll(func() {
			tests.BeforeTestCleanup()

			// The initial test for the metrics subsystem used only a single VM for the sake of simplicity.
			// However, testing a single entity is a corner case (do we test handling sequences? potential clashes
			// in maps? and so on).
			// Thus, we run now two VMIs per testcase. A more realistic test would use a random number of VMIs >= 3,
			// but we don't do now to make test run quickly and (more important) because lack of resources on CI.

			nodeName := prepareVMIForTests("")
			// any node is fine, we don't really care, as long as we run all VMIs on it.
			prepareVMIForTests(nodeName)

			By("Finding the prometheus endpoint")
			pod, err = kubecli.NewVirtHandlerClient(virtClient).Namespace(flags.KubeVirtInstallNamespace).ForNode(nodeName).Pod()
			Expect(err).ToNot(HaveOccurred(), "Should find the virt-handler pod")
			for _, ip := range pod.Status.PodIPs {
				metricsIPs = append(metricsIPs, ip.IP)
			}
		})

		PIt("[test_id:4136][flaky] should find one leading virt-controller and two ready", func() {
			endpoint, err := virtClient.CoreV1().Endpoints(flags.KubeVirtInstallNamespace).Get(context.Background(), "kubevirt-prometheus-metrics", metav1.GetOptions{})
			Expect(err).ToNot(HaveOccurred())
			foundMetrics := map[string]int{
				"ready":   0,
				"leading": 0,
			}
			By("scraping the metrics endpoint on virt-controller pods")
			for _, ep := range endpoint.Subsets[0].Addresses {
				if !strings.HasPrefix(ep.TargetRef.Name, "virt-controller") {
					continue
				}
				stdout, _, err := tests.ExecuteCommandOnPodV2(
					virtClient,
					pod,
					"virt-handler",
					[]string{
						"curl", "-L", "-k",
						fmt.Sprintf("https://%s:8443/metrics", tests.FormatIPForURL(ep.IP)),
					})
				Expect(err).ToNot(HaveOccurred())
				scrapedData := strings.Split(stdout, "\n")
				for _, data := range scrapedData {
					if strings.HasPrefix(data, "#") {
						continue
					}
					switch data {
					case "leading_virt_controller 1":
						foundMetrics["leading"]++
					case "ready_virt_controller 1":
						foundMetrics["ready"]++
					}
				}
			}

			Expect(foundMetrics["ready"]).To(Equal(2), "expected 2 ready virt-controllers")
			Expect(foundMetrics["leading"]).To(Equal(1), "expected 1 leading virt-controller")
		})

		It("[test_id:4137]should find one leading virt-operator and two ready", func() {
			endpoint, err := virtClient.CoreV1().Endpoints(flags.KubeVirtInstallNamespace).Get(context.Background(), "kubevirt-prometheus-metrics", metav1.GetOptions{})
			Expect(err).ToNot(HaveOccurred())
			foundMetrics := map[string]int{
				"ready":   0,
				"leading": 0,
			}
			By("scraping the metrics endpoint on virt-operator pods")
			for _, ep := range endpoint.Subsets[0].Addresses {
				if !strings.HasPrefix(ep.TargetRef.Name, "virt-operator") {
					continue
				}
				stdout, _, err := tests.ExecuteCommandOnPodV2(
					virtClient,
					pod,
					"virt-handler",
					[]string{
						"curl", "-L", "-k",
						fmt.Sprintf("https://%s:8443/metrics", tests.FormatIPForURL(ep.IP)),
					})
				Expect(err).ToNot(HaveOccurred())
				scrapedData := strings.Split(stdout, "\n")
				for _, data := range scrapedData {
					if strings.HasPrefix(data, "#") {
						continue
					}
					switch data {
					case "leading_virt_operator 1":
						foundMetrics["leading"]++
					case "ready_virt_operator 1":
						foundMetrics["ready"]++
					}
				}
			}

			Expect(foundMetrics["ready"]).To(Equal(2), "expected 2 ready virt-operators")
			Expect(foundMetrics["leading"]).To(Equal(1), "expected 1 leading virt-operator")
		})

		It("[test_id:4138]should be exposed and registered on the metrics endpoint", func() {
			endpoint, err := virtClient.CoreV1().Endpoints(flags.KubeVirtInstallNamespace).Get(context.Background(), "kubevirt-prometheus-metrics", metav1.GetOptions{})
			Expect(err).ToNot(HaveOccurred())
			l, err := labels.Parse("prometheus.kubevirt.io")
			Expect(err).ToNot(HaveOccurred())
			pods, err := virtClient.CoreV1().Pods(flags.KubeVirtInstallNamespace).List(context.Background(), metav1.ListOptions{LabelSelector: l.String()})
			Expect(err).ToNot(HaveOccurred())
			Expect(endpoint.Subsets).To(HaveLen(1))

			By("checking if the endpoint contains the metrics port and only one matching subset")
			Expect(endpoint.Subsets[0].Ports).To(HaveLen(1))
			Expect(endpoint.Subsets[0].Ports[0].Name).To(Equal("metrics"))
			Expect(endpoint.Subsets[0].Ports[0].Port).To(Equal(int32(8443)))

			By("checking if  the IPs in the subset match the KubeVirt system Pod count")
			Expect(len(pods.Items)).To(BeNumerically(">=", 3), "At least one api, controller and handler need to be present")
			Expect(endpoint.Subsets[0].Addresses).To(HaveLen(len(pods.Items)))

			ips := map[string]string{}
			for _, ep := range endpoint.Subsets[0].Addresses {
				ips[ep.IP] = ""
			}
			for _, pod := range pods.Items {
				Expect(ips).To(HaveKey(pod.Status.PodIP), fmt.Sprintf("IP of Pod %s not found in metrics endpoint", pod.Name))
			}
		})
		It("[test_id:4139]should return Prometheus metrics", func() {
			endpoint, err := virtClient.CoreV1().Endpoints(flags.KubeVirtInstallNamespace).Get(context.Background(), "kubevirt-prometheus-metrics", metav1.GetOptions{})
			Expect(err).ToNot(HaveOccurred())
			for _, ep := range endpoint.Subsets[0].Addresses {
				stdout, _, err := tests.ExecuteCommandOnPodV2(virtClient,
					pod,
					"virt-handler",
					[]string{
						"curl",
						"-L",
						"-k",
						fmt.Sprintf("https://%s:%s/metrics", tests.FormatIPForURL(ep.IP), "8443"),
					})
				Expect(err).ToNot(HaveOccurred())
				Expect(stdout).To(ContainSubstring("go_goroutines"))
			}
		})

		table.DescribeTable("should throttle the Prometheus metrics access", func(family k8sv1.IPFamily) {
			if family == k8sv1.IPv6Protocol {
				libnet.SkipWhenNotDualStackCluster(virtClient)
			}

			ip := getSupportedIP(metricsIPs, family)

			if netutils.IsIPv6String(ip) {
				Skip("Skip testing with IPv6 until https://github.com/kubevirt/kubevirt/issues/4145 is fixed")
			}

			concurrency := 100 // random value "much higher" than maxRequestsInFlight

			tr := &http.Transport{
				MaxIdleConnsPerHost: concurrency,
				TLSClientConfig: &tls.Config{
					InsecureSkipVerify: true,
				},
			}

			client := http.Client{
				Timeout:   time.Duration(1 * time.Second),
				Transport: tr,
			}

			errorsChan := make(chan error)
			By("Scraping the Prometheus endpoint")
			metricsURL := prepareMetricsURL(ip, 8443)
			for ix := 0; ix < concurrency; ix++ {
				go func(ix int) {
					req, _ := http.NewRequest("GET", metricsURL, nil)
					resp, err := client.Do(req)
					if err != nil {
						fmt.Fprintf(GinkgoWriter, "client: request: %v #%d: %v\n", req, ix, err) // troubleshooting helper
					} else {
						resp.Body.Close()
					}
					errorsChan <- err
				}(ix)
			}

			err := validatedHTTPResponses(errorsChan, concurrency)
			Expect(err).ToNot(HaveOccurred(), "Should throttle HTTP access without unexpected errors")
		},
			table.Entry("[test_id:4140] by using IPv4", k8sv1.IPv4Protocol),
			table.Entry("by using IPv6", k8sv1.IPv6Protocol),
		)

		table.DescribeTable("should include the metrics for a running VM", func(family k8sv1.IPFamily) {
			if family == k8sv1.IPv6Protocol {
				libnet.SkipWhenNotDualStackCluster(virtClient)
			}

			ip := getSupportedIP(metricsIPs, family)

			By("Scraping the Prometheus endpoint")
			Eventually(func() string {
				out := getKubevirtVMMetrics(ip)
				lines := takeMetricsWithPrefix(out, "kubevirt")
				return strings.Join(lines, "\n")
			}, 30*time.Second, 2*time.Second).Should(ContainSubstring("kubevirt"))
		},
			table.Entry("[test_id:4141] by using IPv4", k8sv1.IPv4Protocol),
			table.Entry("by using IPv6", k8sv1.IPv6Protocol),
		)

		table.DescribeTable("should include the storage metrics for a running VM", func(family k8sv1.IPFamily) {
			if family == k8sv1.IPv6Protocol {
				libnet.SkipWhenNotDualStackCluster(virtClient)
			}

			ip := getSupportedIP(metricsIPs, family)

			metrics := collectMetrics(ip, "kubevirt_vmi_storage_")
			By("Checking the collected metrics")
			keys := getKeysFromMetrics(metrics)
			for _, key := range keys {
				if strings.Contains(key, `drive="vdb"`) {
					value := metrics[key]
					Expect(value).To(BeNumerically(">", float64(0.0)))
				}
			}
		},
			table.Entry("[test_id:4142] by using IPv4", k8sv1.IPv4Protocol),
			table.Entry("by using IPv6", k8sv1.IPv6Protocol),
		)

		table.DescribeTable("should include metrics for a running VM", func(family k8sv1.IPFamily, metricSubstring, operator string) {
			if family == k8sv1.IPv6Protocol {
				libnet.SkipWhenNotDualStackCluster(virtClient)
			}

			ip := getSupportedIP(metricsIPs, family)

			metrics := collectMetrics(ip, metricSubstring)
			By("Checking the collected metrics")
			keys := getKeysFromMetrics(metrics)
			for _, key := range keys {
				value := metrics[key]
				fmt.Fprintf(GinkgoWriter, "metric value was %f\n", value)
				Expect(value).To(BeNumerically(operator, float64(0.0)))
			}
		},
			table.Entry("[test_id:4143] network metrics by IPv4", k8sv1.IPv4Protocol, "kubevirt_vmi_network_", ">="),
			table.Entry("network metrics by IPv6", k8sv1.IPv6Protocol, "kubevirt_vmi_network_", ">="),
			table.Entry("[test_id:4144] memory metrics by IPv4", k8sv1.IPv4Protocol, "kubevirt_vmi_memory", ">="),
			table.Entry("memory metrics by IPv6", k8sv1.IPv6Protocol, "kubevirt_vmi_memory", ">="),
			table.Entry("[test_id:4553] vcpu wait by IPv4", k8sv1.IPv4Protocol, "kubevirt_vmi_vcpu_wait", "=="),
			table.Entry("vcpu wait by IPv6", k8sv1.IPv6Protocol, "kubevirt_vmi_vcpu_wait", "=="),
			table.Entry("[test_id:4554] vcpu seconds by IPv4", k8sv1.IPv4Protocol, "kubevirt_vmi_vcpu_seconds", ">="),
			table.Entry("vcpu seconds by IPv6", k8sv1.IPv6Protocol, "kubevirt_vmi_vcpu_seconds", ">="),
			table.Entry("[test_id:4556] vmi unused memory by IPv4", k8sv1.IPv4Protocol, "kubevirt_vmi_memory_unused_bytes", ">="),
			table.Entry("vmi unused memory by IPv6", k8sv1.IPv6Protocol, "kubevirt_vmi_memory_unused_bytes", ">="),
		)

		table.DescribeTable("should include VMI infos for a running VM", func(family k8sv1.IPFamily) {
			if family == k8sv1.IPv6Protocol {
				libnet.SkipWhenNotDualStackCluster(virtClient)
			}

			ip := getSupportedIP(metricsIPs, family)

			metrics := collectMetrics(ip, "kubevirt_vmi_")
			By("Checking the collected metrics")
			keys := getKeysFromMetrics(metrics)
			nodeName := pod.Spec.NodeName

			nameMatchers := []gomegatypes.GomegaMatcher{}
			for _, vmi := range preparedVMIs {
				nameMatchers = append(nameMatchers, ContainSubstring(`name="%s"`, vmi.Name))
			}

			for _, key := range keys {
				// we don't care about the ordering of the labels
				if strings.HasPrefix(key, "kubevirt_vmi_phase_count") {
					// special case: namespace and name don't make sense for this metric
					Expect(key).To(ContainSubstring(`node="%s"`, nodeName))
					continue
				}

				Expect(key).To(SatisfyAll(
					ContainSubstring(`node="%s"`, nodeName),
					// all testing VMIs are on the same node and namespace,
					// so checking the namespace of any random VMI is fine
					ContainSubstring(`namespace="%s"`, preparedVMIs[0].Namespace),
					// otherwise, each key must refer to exactly one the prepared VMIs.
					SatisfyAny(nameMatchers...),
				))
			}
		},
			table.Entry("[test_id:4145] by IPv4", k8sv1.IPv4Protocol),
			table.Entry("by IPv6", k8sv1.IPv6Protocol),
		)

		table.DescribeTable("should include VMI phase metrics for all running VMs", func(family k8sv1.IPFamily) {
			if family == k8sv1.IPv6Protocol {
				libnet.SkipWhenNotDualStackCluster(virtClient)
			}

			ip := getSupportedIP(metricsIPs, family)

			metrics := collectMetrics(ip, "kubevirt_vmi_")
			By("Checking the collected metrics")
			keys := getKeysFromMetrics(metrics)
			for _, key := range keys {
				if strings.Contains(key, `phase="running"`) {
					value := metrics[key]
					Expect(value).To(Equal(float64(len(preparedVMIs))))
				}
			}
		},
			table.Entry("[test_id:4146] by IPv4", k8sv1.IPv4Protocol),
			table.Entry("by IPv6", k8sv1.IPv6Protocol),
		)

		table.DescribeTable("should include kubernetes labels to VMI metrics", func(family k8sv1.IPFamily) {
			if family == k8sv1.IPv6Protocol {
				libnet.SkipWhenNotDualStackCluster(virtClient)
			}

			ip := getSupportedIP(metricsIPs, family)

			// Every VMI is labeled with kubevirt.io/nodeName, so just creating a VMI should
			// be enough to its metrics to contain a kubernetes label
			metrics := collectMetrics(ip, "kubevirt_vmi_vcpu_seconds")
			By("Checking collected metrics")
			keys := getKeysFromMetrics(metrics)
			containK8sLabel := false
			for _, key := range keys {
				if strings.Contains(key, "kubernetes_vmi_label_") {
					containK8sLabel = true
				}
			}
			Expect(containK8sLabel).To(Equal(true))
		},
			table.Entry("[test_id:4147] by IPv4", k8sv1.IPv4Protocol),
			table.Entry("by IPv6", k8sv1.IPv6Protocol),
		)

		// explicit test fo swap metrics as test_id:4144 doesn't catch if they are missing
		table.DescribeTable("should include swap metrics", func(family k8sv1.IPFamily) {
			if family == k8sv1.IPv6Protocol {
				libnet.SkipWhenNotDualStackCluster(virtClient)
			}

			ip := getSupportedIP(metricsIPs, family)

			metrics := collectMetrics(ip, "kubevirt_vmi_memory_swap_")
			var in, out bool
			for k := range metrics {
				if in && out {
					break
				}
				if strings.Contains(k, `swap_in`) {
					in = true
				}
				if strings.Contains(k, `swap_out`) {
					out = true
				}
			}

			Expect(in).To(BeTrue())
			Expect(out).To(BeTrue())
		},
			table.Entry("[test_id:4555] by IPv4", k8sv1.IPv4Protocol),
			table.Entry("by IPv6", k8sv1.IPv6Protocol),
		)
	})

	Describe("Start a VirtualMachineInstance", func() {
		BeforeEach(func() {
			tests.BeforeTestCleanup()
		})

		Context("when the controller pod is not running and an election happens", func() {
			It("[test_id:4642]should succeed afterwards", func() {
				newLeaderPod := getNewLeaderPod(virtClient)
				Expect(newLeaderPod).NotTo(BeNil())

				// TODO: It can be race condition when newly deployed pod receive leadership, in this case we will need
				// to reduce Deployment replica before destroying the pod and to restore it after the test
				By("Destroying the leading controller pod")
				Eventually(func() string {
					leaderPodName := getLeader()

					Expect(virtClient.CoreV1().Pods(flags.KubeVirtInstallNamespace).Delete(context.Background(), leaderPodName, metav1.DeleteOptions{})).To(BeNil())

					Eventually(getLeader, 30*time.Second, 5*time.Second).ShouldNot(Equal(leaderPodName))

					leaderPod, err := virtClient.CoreV1().Pods(flags.KubeVirtInstallNamespace).Get(context.Background(), getLeader(), metav1.GetOptions{})
					Expect(err).To(BeNil())

					return leaderPod.Name
				}, 90*time.Second, 5*time.Second).Should(Equal(newLeaderPod.Name))

				Expect(func() k8sv1.ConditionStatus {
					leaderPod, err := virtClient.CoreV1().Pods(flags.KubeVirtInstallNamespace).Get(context.Background(), newLeaderPod.Name, metav1.GetOptions{})
					Expect(err).To(BeNil())

					for _, condition := range leaderPod.Status.Conditions {
						if condition.Type == k8sv1.PodReady {
							return condition.Status
						}
					}
					return k8sv1.ConditionUnknown
				}()).To(Equal(k8sv1.ConditionTrue))

				vmi := tests.NewRandomVMI()

				By("Starting a new VirtualMachineInstance")
				obj, err := virtClient.RestClient().Post().Resource("virtualmachineinstances").Namespace(tests.NamespaceTestDefault).Body(vmi).Do(context.Background()).Get()
				Expect(err).To(BeNil())
				tests.WaitForSuccessfulVMIStart(obj)
			}, 150)
		})

	})
})

func getLeader() string {
	virtClient, err := kubecli.GetKubevirtClient()
	tests.PanicOnError(err)

	controllerEndpoint, err := virtClient.CoreV1().Endpoints(flags.KubeVirtInstallNamespace).Get(context.Background(), leaderelectionconfig.DefaultEndpointName, metav1.GetOptions{})
	tests.PanicOnError(err)

	var record resourcelock.LeaderElectionRecord
	if recordBytes, found := controllerEndpoint.Annotations[resourcelock.LeaderElectionRecordAnnotationKey]; found {
		err := json.Unmarshal([]byte(recordBytes), &record)
		tests.PanicOnError(err)
	}
	return record.HolderIdentity
}

func getNewLeaderPod(virtClient kubecli.KubevirtClient) *k8sv1.Pod {
	labelSelector, err := labels.Parse(fmt.Sprint(v1.AppLabel + "=virt-controller"))
	tests.PanicOnError(err)
	fieldSelector := fields.ParseSelectorOrDie("status.phase=" + string(k8sv1.PodRunning))
	controllerPods, err := virtClient.CoreV1().Pods(flags.KubeVirtInstallNamespace).List(context.Background(),
		metav1.ListOptions{LabelSelector: labelSelector.String(), FieldSelector: fieldSelector.String()})
	tests.PanicOnError(err)
	leaderPodName := getLeader()
	for _, pod := range controllerPods.Items {
		if pod.Name != leaderPodName {
			return &pod
		}
	}
	return nil
}

func parseMetricsToMap(lines []string) (map[string]float64, error) {
	metrics := make(map[string]float64)
	for _, line := range lines {
		items := strings.Split(line, " ")
		if len(items) != 2 {
			return nil, fmt.Errorf("can't split properly line '%s'", line)
		}
		v, err := strconv.ParseFloat(items[1], 64)
		if err != nil {
			return nil, err
		}
		metrics[items[0]] = v
	}
	return metrics, nil
}

func takeMetricsWithPrefix(output, prefix string) []string {
	lines := strings.Split(output, "\n")
	var ret []string
	for _, line := range lines {
		if strings.HasPrefix(line, prefix) {
			ret = append(ret, line)
		}
	}
	return ret
}

func getKeysFromMetrics(metrics map[string]float64) []string {
	var keys []string
	for metric := range metrics {
		keys = append(keys, metric)
	}
	// we sort keys only to make debug of test failures easier
	sort.Strings(keys)
	return keys
}

// validatedHTTPResponses checks the HTTP responses.
// It expects timeout errors, due to the throttling on the producer side.
// In case of unexpected errors or no errors at all it would fail,
// returning the first unexpected error if any, or a custom error in case
// there were no errors at all.
func validatedHTTPResponses(errorsChan chan error, concurrency int) error {
	var expectedErrorsCount int = 0
	var unexpectedError error
	for ix := 0; ix < concurrency; ix++ {
		err := <-errorsChan
		if unexpectedError == nil && err != nil {
			var e *neturl.Error
			if errors.As(err, &e) && e.Timeout() {
				expectedErrorsCount++
			} else {
				unexpectedError = err
			}
		}
	}

	if unexpectedError == nil && expectedErrorsCount == 0 {
		return fmt.Errorf("timeout errors were expected due to throttling")
	}

	return unexpectedError
}

func getSupportedIP(ips []string, family k8sv1.IPFamily) string {
	ip := libnet.GetIp(ips, family)
	ExpectWithOffset(1, ip).NotTo(BeEmpty())

	return ip
}

func prepareMetricsURL(ip string, port int) string {
	return fmt.Sprintf("https://%s/metrics", net.JoinHostPort(ip, strconv.Itoa(port)))
}
