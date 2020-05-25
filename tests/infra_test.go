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
	"crypto/tls"
	"encoding/json"
	"fmt"
	"net/http"
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

	k8sv1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/fields"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/client-go/tools/leaderelection/resourcelock"

	v1 "kubevirt.io/client-go/api/v1"
	"kubevirt.io/client-go/kubecli"
	clusterutil "kubevirt.io/kubevirt/pkg/util/cluster"
	"kubevirt.io/kubevirt/pkg/virt-controller/leaderelectionconfig"
	"kubevirt.io/kubevirt/pkg/virt-operator/creation/components"
	"kubevirt.io/kubevirt/tests"
)

var _ = Describe("Infrastructure", func() {
	tests.FlagParse()

	virtClient, err := kubecli.GetKubevirtClient()
	tests.PanicOnError(err)

	config, err := kubecli.GetConfig()
	if err != nil {
		panic(err)
	}

	aggregatorClient := aggregatorclient.NewForConfigOrDie(config)

	Describe("[rfe_id:4102][crit:medium][vendor:cnv-qe@redhat.com][level:component]certificates", func() {

		It("[test_id:4099]should be rotated when a new CA is created", func() {
			By("checking that the config-map gets the new CA bundle attached")
			Eventually(func() int {
				_, crts := tests.GetBundleFromConfigMap(components.KubeVirtCASecretName)
				return len(crts)
			}, 10*time.Second, 1*time.Second).Should(BeNumerically(">", 0))

			By("destroying the certificate")
			secret, err := virtClient.CoreV1().Secrets(tests.KubeVirtInstallNamespace).Get(components.KubeVirtCASecretName, metav1.GetOptions{})
			Expect(err).ToNot(HaveOccurred())
			secret.Data = map[string][]byte{
				"random": []byte("nonsense"),
			}
			_, err = virtClient.CoreV1().Secrets(tests.KubeVirtInstallNamespace).Update(secret)
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
				webhook, err := virtClient.AdmissionregistrationV1beta1().ValidatingWebhookConfigurations().Get(components.VirtAPIValidatingWebhookName, metav1.GetOptions{})
				Expect(err).ToNot(HaveOccurred())
				if len(webhook.Webhooks) > 0 {
					return tests.ContainsCrt(webhook.Webhooks[0].ClientConfig.CABundle, newCA)
				}
				return false
			}, 10*time.Second, 1*time.Second).Should(BeTrue())
			By("checking that the ca bundle gets propagated to the mutating webhook")
			Eventually(func() bool {
				webhook, err := virtClient.AdmissionregistrationV1beta1().MutatingWebhookConfigurations().Get(components.VirtAPIMutatingWebhookName, metav1.GetOptions{})
				Expect(err).ToNot(HaveOccurred())
				if len(webhook.Webhooks) > 0 {
					return tests.ContainsCrt(webhook.Webhooks[0].ClientConfig.CABundle, newCA)
				}
				return false
			}, 10*time.Second, 1*time.Second).Should(BeTrue())

			By("checking that the ca bundle gets propagated to the apiservice")
			Eventually(func() bool {
				apiService, err := aggregatorClient.ApiregistrationV1beta1().APIServices().Get("v1alpha3.subresources.kubevirt.io", metav1.GetOptions{})
				Expect(err).ToNot(HaveOccurred())
				return tests.ContainsCrt(apiService.Spec.CABundle, newCA)
			}, 10*time.Second, 1*time.Second).Should(BeTrue())

			By("checking that we can still start virtual machines and connect to the VMI")
			vmi := tests.NewRandomVMIWithEphemeralDisk(tests.ContainerDiskFor(tests.ContainerDiskAlpine))
			vmi = tests.RunVMI(vmi, 60)
			expecter, err := tests.LoggedInAlpineExpecter(vmi)
			Expect(err).ToNot(HaveOccurred())
			defer expecter.Close()
		})

		It("[test_id:4100]should be valid during the whole rotation process", func() {
			oldAPICert := tests.EnsurePodsCertIsSynced(fmt.Sprintf("%s=%s", v1.AppLabel, "virt-api"), tests.KubeVirtInstallNamespace, "8443")
			oldHandlerCert := tests.EnsurePodsCertIsSynced(fmt.Sprintf("%s=%s", v1.AppLabel, "virt-handler"), tests.KubeVirtInstallNamespace, "8186")
			Expect(err).ToNot(HaveOccurred())

			By("destroying the CA certificate")
			err = virtClient.CoreV1().Secrets(tests.KubeVirtInstallNamespace).Delete(components.KubeVirtCASecretName, &metav1.DeleteOptions{})
			Expect(err).ToNot(HaveOccurred())

			By("repeatedly starting VMIs until virt-api and virt-handler certificates are updated")
			Eventually(func() (rotated bool) {
				vmi := tests.NewRandomVMIWithEphemeralDisk(tests.ContainerDiskFor(tests.ContainerDiskAlpine))
				vmi = tests.RunVMI(vmi, 60)
				expecter, err := tests.LoggedInAlpineExpecter(vmi)
				Expect(err).ToNot(HaveOccurred())
				expecter.Close()
				err = virtClient.VirtualMachineInstance(vmi.Namespace).Delete(vmi.Name, &metav1.DeleteOptions{})
				Expect(err).ToNot(HaveOccurred())
				newAPICert, _, err := tests.GetPodsCertIfSynced(fmt.Sprintf("%s=%s", v1.AppLabel, "virt-api"), tests.KubeVirtInstallNamespace, "8443")
				Expect(err).ToNot(HaveOccurred())
				newHandlerCert, _, err := tests.GetPodsCertIfSynced(fmt.Sprintf("%s=%s", v1.AppLabel, "virt-handler"), tests.KubeVirtInstallNamespace, "8186")
				Expect(err).ToNot(HaveOccurred())
				return !reflect.DeepEqual(oldHandlerCert, newHandlerCert) && !reflect.DeepEqual(oldAPICert, newAPICert)
			}, 120*time.Second).Should(BeTrue())
		})

		table.DescribeTable("should be rotated when deleted for ", func(secretName string) {
			By("destroying the certificate")
			secret, err := virtClient.CoreV1().Secrets(tests.KubeVirtInstallNamespace).Get(secretName, metav1.GetOptions{})
			Expect(err).ToNot(HaveOccurred())
			secret.Data = map[string][]byte{
				"random": []byte("nonsense"),
			}
			_, err = virtClient.CoreV1().Secrets(tests.KubeVirtInstallNamespace).Update(secret)
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
			Do().Get()
		Expect(err).ToNot(HaveOccurred(), "Should create VMI")

		By("Waiting until the VM is ready")
		return tests.WaitForSuccessfulVMIStart(obj)
	}

	Describe("Taints and toleration", func() {

		It("should tolerate CriticalAddonsOnly toleration", func() {

			var kvPods *k8sv1.PodList
			By("finding all nodes that are running kubevirt components", func() {
				kvPods, err = virtClient.CoreV1().
					Pods(tests.KubeVirtInstallNamespace).List(metav1.ListOptions{})
				Expect(err).ToNot(HaveOccurred(), "failed listing kubevirt pods")
				Expect(len(kvPods.Items)).
					To(BeNumerically(">", 0), "no kubevirt pods found")
			})

			// nodes that run kubevirt components.
			kvTaintedNodes := make(map[string]*k8sv1.Node)
			By("adding taints to any node that runs kubevirt component", func() {
				criticalPodTaint := k8sv1.Taint{
					Key:    "CriticalAddonsOnly",
					Value:  "",
					Effect: k8sv1.TaintEffectNoExecute,
				}

				for _, kvPod := range kvPods.Items {
					if _, exists := kvTaintedNodes[kvPod.Spec.NodeName]; exists {
						continue
					}
					kvNode, err := virtClient.CoreV1().Nodes().Get(kvPod.Spec.NodeName, metav1.GetOptions{})
					Expect(err).ToNot(HaveOccurred(), "failed retrieving node")
					hasTaint := false
					// check if node already has the taint
					for _, taint := range kvNode.Spec.Taints {
						if reflect.DeepEqual(taint, criticalPodTaint) {
							// node already have the taint set
							hasTaint = true
							break
						}
					}
					if hasTaint {
						continue
					}
					kvNode.ResourceVersion = ""
					kvTaintedNodes[kvPod.Spec.NodeName] = kvNode
					kvNodeCopy := kvNode.DeepCopy()
					kvNodeCopy.Spec.Taints = append(kvNodeCopy.Spec.Taints, criticalPodTaint)
					_, err = virtClient.CoreV1().Nodes().Update(kvNodeCopy)
					Expect(err).ToNot(HaveOccurred(), "failed setting taint on node")
				}
			})

			defer func() {
				var errors []error
				By("restoring nodes")
				for _, nodeSpec := range kvTaintedNodes {
					_, err = virtClient.CoreV1().Nodes().Update(nodeSpec)
					if err != nil {
						errors = append(errors, err)
					}
				}
				Expect(errors).Should(BeEmpty(), "failed restoring one or more nodes")
			}()

			By("watching for terminated kubevirt pods", func() {
				lw, err := virtClient.CoreV1().Pods(tests.KubeVirtInstallNamespace).Watch(metav1.ListOptions{})
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
				Consistently(terminatedPodsCn, 5*time.Second).
					ShouldNot(Receive(), "pods should not terminate")
				stopCn <- true
			})
		})

	})

	Describe("Prometheus scraped metrics", func() {

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

		It("should find VMI namespace on namespace label of the metric", func() {

			/*
				This test is required because in cases of misconfigurations on
				monitoring objects (such for the ServiceMonitor), our rules will
				still be picked up by the monitoring-operator, but Prometheus
				will fail to load it.
			*/

			By("creating a VMI in a user defined namespace")
			vmi := tests.NewRandomVMIWithEphemeralDisk(
				tests.ContainerDiskFor(tests.ContainerDiskAlpine))
			startVMI(vmi)

			By("finding virt-operator pod")
			ops, err := virtClient.CoreV1().Pods("kubevirt").
				List(metav1.ListOptions{LabelSelector: "kubevirt.io=virt-operator"})
			Expect(err).ToNot(HaveOccurred(), "failed to list virt-operators")
			Expect(ops.Size).ToNot(Equal(0), "no virt-operators found")
			op := ops.Items[0]
			Expect(op).ToNot(BeNil(), "virt-operator pod should not be nil")

			By("finding Prometheus endpoint")
			ep, err := virtClient.CoreV1().Endpoints("openshift-monitoring").
				Get("prometheus-k8s", metav1.GetOptions{})
			Expect(err).ToNot(HaveOccurred(), "failed to retrieve Prometheus endpoint")
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

	Describe("[rfe_id:3187][crit:medium][vendor:cnv-qe@redhat.com][level:component] Prometheus Endpoints", func() {
		var preparedVMIs []*v1.VirtualMachineInstance
		var pod *k8sv1.Pod
		var metricsURL string

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
		getKubevirtVMMetrics := func() string {
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
		collectMetrics := func(metricSubstring string) map[string]float64 {
			By("Scraping the Prometheus endpoint")
			var metrics map[string]float64
			var lines []string

			Eventually(func() map[string]float64 {
				out := getKubevirtVMMetrics()
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
			vmi := tests.NewRandomVMIWithEphemeralDisk(tests.ContainerDiskFor(tests.ContainerDiskAlpine))
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
			expecter, err := tests.LoggedInAlpineExpecter(vmi)
			Expect(err).ToNot(HaveOccurred())
			defer expecter.Close()

			By("Writing some data to the disk")
			_, err = expecter.ExpectBatch([]expect.Batcher{
				&expect.BSnd{S: "dd if=/dev/zero of=/dev/vdb bs=1M count=1\n"},
				&expect.BExp{R: "localhost:~#"},
				&expect.BSnd{S: "sync\n"},
				&expect.BExp{R: "localhost:~#"},
			}, 10*time.Second)
			Expect(err).ToNot(HaveOccurred())

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
			pod, err = kubecli.NewVirtHandlerClient(virtClient).Namespace(tests.KubeVirtInstallNamespace).ForNode(nodeName).Pod()
			Expect(err).ToNot(HaveOccurred(), "Should find the virt-handler pod")
			metricsURL = fmt.Sprintf("https://%s:%d/metrics", tests.FormatIPForURL(pod.Status.PodIP), 8443)
		})

		PIt(" [flaky] should find one leading virt-controller and two ready", func() {
			endpoint, err := virtClient.CoreV1().
				Endpoints(tests.KubeVirtInstallNamespace).
				Get("kubevirt-prometheus-metrics", metav1.GetOptions{})
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

		It("should find one leading virt-operator and two ready", func() {
			endpoint, err := virtClient.CoreV1().
				Endpoints(tests.KubeVirtInstallNamespace).
				Get("kubevirt-prometheus-metrics", metav1.GetOptions{})
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

		It("should be exposed and registered on the metrics endpoint", func() {
			endpoint, err := virtClient.CoreV1().Endpoints(tests.KubeVirtInstallNamespace).Get("kubevirt-prometheus-metrics", metav1.GetOptions{})
			Expect(err).ToNot(HaveOccurred())
			l, err := labels.Parse("prometheus.kubevirt.io")
			Expect(err).ToNot(HaveOccurred())
			pods, err := virtClient.CoreV1().Pods(tests.KubeVirtInstallNamespace).List(metav1.ListOptions{LabelSelector: l.String()})
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
		It("should return Prometheus metrics", func() {
			endpoint, err := virtClient.CoreV1().Endpoints(tests.KubeVirtInstallNamespace).Get("kubevirt-prometheus-metrics", metav1.GetOptions{})
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

		It("should throttle the Prometheus metrics access", func() {
			By("Scraping the Prometheus endpoint")
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

			errors := make(chan error)
			for ix := 0; ix < concurrency; ix++ {
				go func() {
					req, _ := http.NewRequest("GET", metricsURL, nil)
					resp, err := client.Do(req)
					if err != nil {
						fmt.Fprintf(GinkgoWriter, "client: request: %v #%d: %v\n", req, ix, err) // troubleshooting helper
					} else {
						resp.Body.Close()
					}
					errors <- err
				}()
			}

			errorCount := 0
			for ix := 0; ix < concurrency; ix++ {
				err := <-errors
				if err != nil {
					errorCount += 1
				}
			}

			fmt.Fprintf(GinkgoWriter, "client: total errors #%d\n", errorCount) // troubleshooting helper
			Expect(errorCount).To(BeNumerically(">", 0))
		})

		It("should include the metrics for a running VM", func() {
			By("Scraping the Prometheus endpoint")
			Eventually(func() string {
				out := getKubevirtVMMetrics()
				lines := takeMetricsWithPrefix(out, "kubevirt")
				return strings.Join(lines, "\n")
			}, 30*time.Second, 2*time.Second).Should(ContainSubstring("kubevirt"))
		})

		It("should include the storage metrics for a running VM", func() {
			metrics := collectMetrics("kubevirt_vmi_storage_")
			By("Checking the collected metrics")
			keys := getKeysFromMetrics(metrics)
			for _, key := range keys {
				if strings.Contains(key, "vdb") {
					value := metrics[key]
					Expect(value).To(BeNumerically(">", float64(0.0)))
				}
			}
		})

		It("should include the network metrics for a running VM", func() {
			metrics := collectMetrics("kubevirt_vmi_network_")
			By("Checking the collected metrics")
			keys := getKeysFromMetrics(metrics)
			for _, key := range keys {
				value := metrics[key]
				Expect(value).To(BeNumerically(">=", float64(0.0)))
			}
		})

		It("should include the memory metrics for a running VM", func() {
			metrics := collectMetrics("kubevirt_vmi_memory")
			By("Checking the collected metrics")
			keys := getKeysFromMetrics(metrics)
			for _, key := range keys {
				value := metrics[key]
				// swap metrics may (and should) be actually zero
				Expect(value).To(BeNumerically(">=", float64(0.0)))
			}
		})

		It("should include VMI infos for a running VM", func() {
			metrics := collectMetrics("kubevirt_vmi_")
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
		})

		It("should include VMI phase metrics for few running VMs", func() {
			// this tests requires at least two running VMis. To ensure this condition,
			// the simplest way is just always run an additional VMI.
			By("Creating another VirtualMachineInstance")

			// `pod` is the pod of the virt-handler of the node on which we run all the VMIs
			// when setting up the tests. So we implicitely run all the VMIs on the same node,
			// so the test works. TODO: make this explicit.
			preferredNodeName := pod.Spec.NodeName
			vmi := pinVMIOnNode(tests.NewRandomVMI(), preferredNodeName)
			nodeName := startVMI(vmi)
			Expect(nodeName).To(Equal(preferredNodeName), "Should run VMIs on the same node")

			metrics := collectMetrics("kubevirt_vmi_")
			By("Checking the collected metrics")
			keys := getKeysFromMetrics(metrics)
			for _, key := range keys {
				if strings.Contains(key, `phase="running"`) {
					value := metrics[key]
					Expect(value).To(Equal(float64(len(preparedVMIs) + 1)))
				}
			}
		})
	})

	Describe("Start a VirtualMachineInstance", func() {
		BeforeEach(func() {
			tests.BeforeTestCleanup()
		})

		Context("when the controller pod is not running and an election happens", func() {
			It("should succeed afterwards", func() {
				newLeaderPod := getNewLeaderPod(virtClient)
				Expect(newLeaderPod).NotTo(BeNil())

				// TODO: It can be race condition when newly deployed pod receive leadership, in this case we will need
				// to reduce Deployment replica before destroying the pod and to restore it after the test
				By("Destroying the leading controller pod")
				Eventually(func() string {
					leaderPodName := getLeader()

					Expect(virtClient.CoreV1().Pods(tests.KubeVirtInstallNamespace).Delete(leaderPodName, &metav1.DeleteOptions{})).To(BeNil())

					Eventually(getLeader, 30*time.Second, 5*time.Second).ShouldNot(Equal(leaderPodName))

					leaderPod, err := virtClient.CoreV1().Pods(tests.KubeVirtInstallNamespace).Get(getLeader(), metav1.GetOptions{})
					Expect(err).To(BeNil())

					return leaderPod.Name
				}, 90*time.Second, 5*time.Second).Should(Equal(newLeaderPod.Name))

				Expect(func() k8sv1.ConditionStatus {
					leaderPod, err := virtClient.CoreV1().Pods(tests.KubeVirtInstallNamespace).Get(newLeaderPod.Name, metav1.GetOptions{})
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
				obj, err := virtClient.RestClient().Post().Resource("virtualmachineinstances").Namespace(tests.NamespaceTestDefault).Body(vmi).Do().Get()
				Expect(err).To(BeNil())
				tests.WaitForSuccessfulVMIStart(obj)
			}, 150)
		})

	})
})

func getLeader() string {
	virtClient, err := kubecli.GetKubevirtClient()
	tests.PanicOnError(err)

	controllerEndpoint, err := virtClient.CoreV1().Endpoints(tests.KubeVirtInstallNamespace).Get(leaderelectionconfig.DefaultEndpointName, metav1.GetOptions{})
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
	controllerPods, err := virtClient.CoreV1().Pods(tests.KubeVirtInstallNamespace).List(
		metav1.ListOptions{LabelSelector: labelSelector.String(), FieldSelector: fieldSelector.String()})
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
