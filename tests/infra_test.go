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
	"crypto/x509"
	"encoding/json"
	"flag"
	"fmt"
	"io/ioutil"
	"net/http"
	"sort"
	"strconv"
	"strings"
	"time"

	expect "github.com/google/goexpect"
	. "github.com/onsi/ginkgo"
	"github.com/onsi/ginkgo/extensions/table"
	. "github.com/onsi/gomega"
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

		var stopChan chan struct{}

		BeforeEach(func() {
			stopChan = make(chan struct{})
		})

		AfterEach(func() {
			close(stopChan)
		})

		table.DescribeTable("should provide a valid certificate", func(component string) {

			var pod *k8sv1.Pod
			switch component {
			case "virt-handler":
				pod, err = tests.GetRandomVirtHandler(tests.KubeVirtInstallNamespace)
				Expect(err).NotTo(HaveOccurred())
			case "virt-controller":
				pod, err = tests.GetRandomVirtController(tests.KubeVirtInstallNamespace)
				Expect(err).NotTo(HaveOccurred())
			default:
				Expect(true).To(BeFalse())
			}

			// XXX use random ports, but our client-go version does not support retrieving the random port
			Expect(tests.ForwardPorts(pod, []string{"4321:8443"}, stopChan, 10*time.Second)).To(Succeed())

			transport := &http.Transport{TLSClientConfig: &tls.Config{
				RootCAs:            tests.ClusterCA,
				ClientCAs:          tests.ClusterCA,
				InsecureSkipVerify: true,
				VerifyPeerCertificate: func(rawCerts [][]byte, verifiedChains [][]*x509.Certificate) error {
					c, err := x509.ParseCertificate(rawCerts[0])
					if err != nil {
						return fmt.Errorf("failed to parse certificate: %v", err)
					}
					// It should be verifyable against the k8s ca.crt
					_, err = c.Verify(x509.VerifyOptions{
						Roots:   tests.ClusterCA,
						DNSName: pod.Name,
					})
					if err != nil {
						return fmt.Errorf("failed to verify certificate against k8s CA: %v", err)
					}

					if len(c.IPAddresses) == 0 {
						return fmt.Errorf("certificate should contain the pod IP but is empty")
					}
					if c.IPAddresses[0].String() != pod.Status.PodIP {
						return fmt.Errorf("expected IP %s but got %s", pod.Status.PodIP, c.IPAddresses[0])
					}

					// It should not be verifyable against default CAs
					systemPool, err := x509.SystemCertPool()
					_, err = c.Verify(x509.VerifyOptions{
						DNSName: pod.Name,
						Roots:   systemPool,
					})
					if err == nil {
						return fmt.Errorf("expected to fail verification against system CAs, but it passed")
					}
					return nil
				},
			}}

			client := http.Client{Transport: transport}
			resp, err := client.Get(fmt.Sprintf("https://localhost:%s/metrics", "4321"))
			Expect(err).ToNot(HaveOccurred())
			metrics, err := ioutil.ReadAll(resp.Body)
			Expect(err).ToNot(HaveOccurred())
			Expect(string(metrics)).To(ContainSubstring("go_goroutines"))
		},
			table.Entry("on virt-handler", "virt-handler"),
			table.Entry("on virt-controller", "virt-controller"),
		)

		It("should return Prometheus metrics and be registerd on the endpoint if the have the prometheus.kubevirt.io label", func() {
			endpoint, err := virtClient.CoreV1().Endpoints(tests.KubeVirtInstallNamespace).Get("kubevirt-prometheus-metrics", metav1.GetOptions{})
			Expect(err).ToNot(HaveOccurred())
			l, err := labels.Parse(fmt.Sprintf("prometheus.kubevirt.io"))
			Expect(err).ToNot(HaveOccurred())
			pods, err := virtClient.CoreV1().Pods(tests.KubeVirtInstallNamespace).List(metav1.ListOptions{LabelSelector: l.String()})
			Expect(err).NotTo(HaveOccurred())

			addresses := map[string]bool{}
			for _, subset := range endpoint.Subsets {
				for _, ep := range subset.Addresses {
					addresses[ep.IP] = true
					Expect(err).ToNot(HaveOccurred())
				}
			}
			Expect(addresses).To(HaveLen(len(pods.Items)))

			for _, pod := range pods.Items {
				Expect(addresses).To(HaveKey(pod.Status.PodIP))
				metrics := getKubevirtVMMetrics(&pod)
				Expect(metrics).To(ContainSubstring("go_goroutines"))
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
				out := getKubevirtVMMetrics(pod)
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
			out := getKubevirtVMMetrics(pod)
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
				out := getKubevirtVMMetrics(pod)
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

func getKubevirtVMMetrics(pod *k8sv1.Pod) (metrics string) {
	stopChan := make(chan struct{})
	var metricsPort *k8sv1.ContainerPort

loop:
	for _, container := range pod.Spec.Containers {
		for _, port := range container.Ports {
			if port.Name == "metrics" {
				metricsPort = &port
				break loop
			}
		}
	}
	Expect(metricsPort).ToNot(BeNil(), "Pod does not contain a port named 'metrics'")

	defer close(stopChan)
	Expect(tests.ForwardPorts(pod, []string{fmt.Sprintf("4321:%v", metricsPort.ContainerPort)}, stopChan, 10*time.Second)).To(Succeed())
	resp, err := tests.GetK8sHTTPClient().Get(fmt.Sprintf("https://localhost:%s/metrics", "4321"))
	Expect(err).ToNot(HaveOccurred())
	data, err := ioutil.ReadAll(resp.Body)
	Expect(err).ToNot(HaveOccurred())
	return string(data)
}
