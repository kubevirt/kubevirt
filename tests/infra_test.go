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
	"sort"
	"strconv"
	"strings"
	"time"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	gomegatypes "github.com/onsi/gomega/types"

	expect "github.com/google/goexpect"

	k8sv1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/fields"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/client-go/tools/leaderelection/resourcelock"

	v1 "kubevirt.io/client-go/api/v1"
	"kubevirt.io/client-go/kubecli"
	"kubevirt.io/kubevirt/pkg/virt-controller/leaderelectionconfig"
	"kubevirt.io/kubevirt/tests"
)

var _ = Describe("Infrastructure", func() {
	tests.FlagParse()

	virtClient, err := kubecli.GetKubevirtClient()
	tests.PanicOnError(err)

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

		// start a VMI, wait for it to run and return the node it runs on
		startVMI := func(vmi *v1.VirtualMachineInstance) string {
			By("Starting a new VirtualMachineInstance")
			obj, err := virtClient.RestClient().Post().Resource("virtualmachineinstances").Namespace(tests.NamespaceTestDefault).Body(vmi).Do().Get()
			Expect(err).ToNot(HaveOccurred(), "Should create VMI")

			By("Waiting until the VM is ready")
			return tests.WaitForSuccessfulVMIStart(obj)
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

		It("should find one leading virt-controller and two ready", func() {
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
