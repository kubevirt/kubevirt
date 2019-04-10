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
	"encoding/json"
	"flag"
	"fmt"
	"sort"
	"strconv"
	"strings"
	"time"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	expect "github.com/google/goexpect"

	k8sv1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/fields"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/client-go/tools/leaderelection/resourcelock"

	v1 "kubevirt.io/kubevirt/pkg/api/v1"
	"kubevirt.io/kubevirt/pkg/kubecli"
	"kubevirt.io/kubevirt/pkg/virt-controller/leaderelectionconfig"
	"kubevirt.io/kubevirt/tests"
)

var _ = Describe("Infrastructure", func() {
	flag.Parse()

	virtClient, err := kubecli.GetKubevirtClient()
	tests.PanicOnError(err)

	BeforeEach(func() {
		tests.BeforeTestCleanup()
	})

	Describe("Prometheus Endpoints", func() {
		It("should be exposed and and registered on the metrics endpoint", func() {
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
			l, err := labels.Parse("kubevirt.io=virt-handler")
			Expect(err).ToNot(HaveOccurred())
			pods, err := virtClient.CoreV1().Pods(tests.KubeVirtInstallNamespace).List(metav1.ListOptions{LabelSelector: l.String()})
			Expect(err).ToNot(HaveOccurred())
			Expect(pods.Items).ToNot(BeEmpty())

			for _, ep := range endpoint.Subsets[0].Addresses {
				stdout, _, err := tests.ExecuteCommandOnPodV2(virtClient,
					&pods.Items[0], "virt-handler",
					[]string{
						"curl",
						"-L",
						"-k",
						fmt.Sprintf("https://%s:%s/metrics", ep.IP, "8443"),
					})
				Expect(err).ToNot(HaveOccurred())
				Expect(stdout).To(ContainSubstring("go_goroutines"))
			}
		})
		It("should include the metrics for a running VM", func() {
			By("Creating the VirtualMachineInstance")
			vmi := tests.NewRandomVMI()

			By("Starting a new VirtualMachineInstance")
			obj, err := virtClient.RestClient().Post().Resource("virtualmachineinstances").Namespace(tests.NamespaceTestDefault).Body(vmi).Do().Get()
			Expect(err).ToNot(HaveOccurred(), "Should create VMI")

			By("Waiting until the VM is ready")
			nodeName := tests.WaitForSuccessfulVMIStart(obj)

			By("Finding the prometheus endpoint")
			pod, err := kubecli.NewVirtHandlerClient(virtClient).ForNode(nodeName).Pod()
			Expect(err).ToNot(HaveOccurred(), "Should find the virt-handler pod")

			By("Scraping the Prometheus endpoint")
			Eventually(func() string {
				out := getKubevirtVMMetrics(virtClient, pod, "virt-handler")
				lines := takeMetricsWithPrefix(out, "kubevirt")
				return strings.Join(lines, "\n")
			}, 30*time.Second, 2*time.Second).Should(ContainSubstring("kubevirt"))
		}, 300)

		It("should include the storage metrics for a running VM", func() {
			By("Creating the VirtualMachineInstance")

			// WARNING: we assume the VM will have a VirtIO disk (vda)
			// and we add our own vdb on which we do our test.
			// but if the default disk is not vda, the test will break
			// TODO: introspect the VMI and get the device name of this
			// block device?
			vmi := tests.NewRandomVMIWithEphemeralDisk(tests.ContainerDiskFor(tests.ContainerDiskAlpine))
			tests.AppendEmptyDisk(vmi, "testdisk", "virtio", "1Gi")

			By("Starting a new VirtualMachineInstance")
			obj, err := virtClient.RestClient().Post().Resource("virtualmachineinstances").Namespace(tests.NamespaceTestDefault).Body(vmi).Do().Get()
			Expect(err).ToNot(HaveOccurred(), "Should create VMI")

			By("Waiting until the VM is ready")
			nodeName := tests.WaitForSuccessfulVMIStart(obj)

			By("Expecting the VirtualMachineInstance console")
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
			// we wrote data to the disk, so from now on the VM *is* running

			By("Finding the prometheus endpoint")
			pod, err := kubecli.NewVirtHandlerClient(virtClient).ForNode(nodeName).Pod()
			Expect(err).ToNot(HaveOccurred(), "Should find the virt-handler pod")

			By("Scraping the Prometheus endpoint")
			// the VM *is* running, so we must have metrics promptly reported
			out := getKubevirtVMMetrics(virtClient, pod, "virt-handler")
			lines := takeMetricsWithPrefix(out, "kubevirt")
			metrics, err := parseMetricsToMap(lines)
			Expect(err).ToNot(HaveOccurred())
			Expect(metrics).To(HaveKey(ContainSubstring("kubevirt_vm_storage_")))

			By("Checking the collected metrics")
			var keys []string
			for metric := range metrics {
				keys = append(keys, metric)
			}
			// we sort keys only to make debug of test failures easier
			sort.Strings(keys)
			for _, key := range keys {
				if strings.HasPrefix(key, "kubevirt_vm_storage_") && strings.Contains(key, "vdb") {
					value := metrics[key]
					Expect(value).To(BeNumerically(">", float64(0.0)))
				}
			}
		}, 300)

		It("should include the memory metrics for a running VM", func() {
			By("Creating the VirtualMachineInstance")
			vmi := tests.NewRandomVMI()

			By("Starting a new VirtualMachineInstance")
			obj, err := virtClient.RestClient().Post().Resource("virtualmachineinstances").Namespace(tests.NamespaceTestDefault).Body(vmi).Do().Get()
			Expect(err).ToNot(HaveOccurred(), "Should create VMI")

			By("Waiting until the VM is ready")
			nodeName := tests.WaitForSuccessfulVMIStart(obj)

			By("Finding the prometheus endpoint")
			pod, err := kubecli.NewVirtHandlerClient(virtClient).ForNode(nodeName).Pod()
			Expect(err).ToNot(HaveOccurred(), "Should find the virt-handler pod")

			By("Scraping the Prometheus endpoint")
			var metrics map[string]float64
			Eventually(func() map[string]float64 {
				out := getKubevirtVMMetrics(virtClient, pod, "virt-handler")
				lines := takeMetricsWithPrefix(out, "kubevirt")
				metrics, err := parseMetricsToMap(lines)
				Expect(err).ToNot(HaveOccurred())
				return metrics
			}, 30*time.Second, 2*time.Second).Should(HaveKey(ContainSubstring("kubevirt_vm_memory_")))

			By("Checking the collected metrics")
			var keys []string
			for metric := range metrics {
				keys = append(keys, metric)
			}
			// we sort keys only to make debug of test failures easier
			sort.Strings(keys)
			for _, key := range keys {
				if strings.HasPrefix(key, "kubevirt_vm_metrics_") {
					value := metrics[key]
					// swap metrics may (and should) be actually zero
					Expect(value).To(BeNumerically(">=", float64(0.0)))
				}
			}
		}, 300)
	})

	Describe("Start a VirtualMachineInstance", func() {
		Context("when the controller pod is not running and an election happens", func() {
			It("should succeed afterwards", func() {
				newLeaderPod := getNewLeaderPod(virtClient)
				Expect(newLeaderPod).NotTo(BeNil())

				// TODO: It can be race condition when newly deployed pod receive leadership, in this case we will need
				// to reduce Deployment replica before destroy the pod and restore it after the test
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

func getKubevirtVMMetrics(virtClient kubecli.KubevirtClient, pod *k8sv1.Pod, containerName string) string {
	stdout, _, err := tests.ExecuteCommandOnPodV2(virtClient,
		pod, containerName,
		[]string{
			"curl",
			"-L",
			"-k",
			fmt.Sprintf("https://%s:%s/metrics", pod.Status.PodIP, "8443"),
		})
	Expect(err).ToNot(HaveOccurred())
	return stdout
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
