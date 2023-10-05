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

package infrastructure

import (
	"context"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	clusterutil "kubevirt.io/kubevirt/pkg/util/cluster"

	"kubevirt.io/kubevirt/tests/libinfra"
	"kubevirt.io/kubevirt/tests/libvmi"

	"kubevirt.io/kubevirt/tests/framework/kubevirt"

	"kubevirt.io/kubevirt/tests/exec"
	"kubevirt.io/kubevirt/tests/testsuite"

	expect "github.com/google/goexpect"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	gomegatypes "github.com/onsi/gomega/types"
	netutils "k8s.io/utils/net"

	"kubevirt.io/kubevirt/tests/libnode"

	"kubevirt.io/kubevirt/tests/libwait"

	k8sv1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	v1 "kubevirt.io/api/core/v1"
	"kubevirt.io/client-go/kubecli"

	"kubevirt.io/kubevirt/tests"
	"kubevirt.io/kubevirt/tests/console"
	"kubevirt.io/kubevirt/tests/flags"
	"kubevirt.io/kubevirt/tests/libnet"
)

const (
	remoteCmdErrPattern = "failed running `%s` with stdout:\n %v \n stderr:\n %v \n err: \n %v \n"
)

var _ = DescribeInfra("[rfe_id:3187][crit:medium][vendor:cnv-qe@redhat.com][level:component]Prometheus scraped metrics", func() {

	var (
		virtClient kubecli.KubevirtClient
	)

	// start a VMI, wait for it to run and return the node it runs on
	startVMI := func(vmi *v1.VirtualMachineInstance) string {
		By("Starting a new VirtualMachineInstance")
		obj, err := virtClient.
			RestClient().
			Post().
			Resource("virtualmachineinstances").
			Namespace(testsuite.GetTestNamespace(vmi)).
			Body(vmi).
			Do(context.Background()).Get()
		Expect(err).ToNot(HaveOccurred(), "Should create VMI")
		vmiObj, ok := obj.(*v1.VirtualMachineInstance)
		Expect(ok).To(BeTrue(), "Object is not of type *v1.VirtualMachineInstance")

		By("Waiting until the VM is ready")
		return libwait.WaitForSuccessfulVMIStart(vmiObj).Status.NodeName
	}

	BeforeEach(func() {
		virtClient = kubevirt.Client()

		onOCP, err := clusterutil.IsOnOpenShift(virtClient)
		Expect(err).ToNot(HaveOccurred(), "failed to detect cluster type")

		if !onOCP {
			Skip("test is verifying integration with OCP's cluster monitoring stack")
		}
	})

	/*
		This test is querying the metrics from Prometheus *after* they were
		scraped and processed by the different components on the way.
	*/

	It("[test_id:4135]should find VMI namespace on namespace label of the metric", func() {

		/*
			This test is required because in cases of misconfigurations on
			monitoring objects (such for the ServiceMonitor), our rules will
			still be picked up by the monitoring-operator, but Prometheus
			will fail to load it.
		*/

		By("creating a VMI in a user defined namespace")
		vmi := libvmi.NewAlpine()
		startVMI(vmi)

		By("finding virt-operator pod")
		ops, err := virtClient.CoreV1().Pods(flags.KubeVirtInstallNamespace).List(context.Background(), metav1.ListOptions{LabelSelector: "kubevirt.io=virt-operator"})
		Expect(err).ToNot(HaveOccurred(), "failed to list virt-operators")
		Expect(ops.Size()).ToNot(Equal(0), "no virt-operators found")
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
		Expect(promIP).ToNot(Equal(""), "could not get Prometheus IP from endpoint")
		var promPort int32
		for _, port := range ep.Subsets[0].Ports {
			if port.Name == "web" {
				promPort = port.Port
			}
		}
		Expect(promPort).ToNot(BeEquivalentTo(0), "could not get Prometheus port from endpoint")

		// We need a token from a service account that can view all namespaces in the cluster
		By("extracting virt-operator sa token")
		cmd := []string{"cat", "/var/run/secrets/kubernetes.io/serviceaccount/token"}
		token, stderr, err := exec.ExecuteCommandOnPodWithResults(virtClient, &op, "virt-operator", cmd)
		Expect(err).ToNot(HaveOccurred(), fmt.Sprintf(remoteCmdErrPattern, strings.Join(cmd, " "), token, stderr, err))
		Expect(token).ToNot(BeEmpty(), "virt-operator sa token returned empty")

		By("querying Prometheus API endpoint for a VMI exported metric")
		cmd = []string{
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
			)}

		stdout, stderr, err := exec.ExecuteCommandOnPodWithResults(virtClient, &op, "virt-operator", cmd)
		Expect(err).ToNot(HaveOccurred(), fmt.Sprintf(remoteCmdErrPattern, strings.Join(cmd, " "), stdout, stderr, err))

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

var _ = DescribeInfra("[rfe_id:3187][crit:medium][vendor:cnv-qe@redhat.com][level:component]Prometheus Endpoints", func() {

	var (
		virtClient kubecli.KubevirtClient
		err        error

		preparedVMIs         []*v1.VirtualMachineInstance
		pod                  *k8sv1.Pod
		handlerMetricIPs     []string
		controllerMetricIPs  []string
		getKubevirtVMMetrics func(string) string
	)

	// collect metrics whose key contains the given string, expects non-empty result
	collectMetrics := func(ip, metricSubstring string) map[string]float64 {
		By("Scraping the Prometheus endpoint")
		var metrics map[string]float64
		var lines []string

		Eventually(func() map[string]float64 {
			out := getKubevirtVMMetrics(ip)
			lines = libinfra.TakeMetricsWithPrefix(out, metricSubstring)
			metrics, err = libinfra.ParseMetricsToMap(lines)
			Expect(err).ToNot(HaveOccurred())
			return metrics
		}, 30*time.Second, 2*time.Second).ShouldNot(BeEmpty())

		// troubleshooting helper
		_, err = fmt.Fprintf(GinkgoWriter, "metrics [%s]:\nlines=%s\n%#v\n", metricSubstring, lines, metrics)
		Expect(err).ToNot(HaveOccurred())
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
		vmi := libvmi.NewAlpine(libvmi.WithEmptyDisk("testdisk", v1.VirtIO, resource.MustParse("1G")))
		if preferredNodeName != "" {
			vmi = libvmi.NewAlpine(libvmi.WithEmptyDisk("testdisk", v1.VirtIO, resource.MustParse("1G")),
				libvmi.WithNodeSelectorFor(&k8sv1.Node{ObjectMeta: metav1.ObjectMeta{Name: preferredNodeName}}))
		}

		vmi = tests.RunVMIAndExpectLaunch(vmi, 30)
		nodeName := vmi.Status.NodeName

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

	BeforeEach(func() {
		virtClient = kubevirt.Client()

		preparedVMIs = []*v1.VirtualMachineInstance{}
		pod = nil
		handlerMetricIPs = []string{}
		controllerMetricIPs = []string{}

		By("Finding the virt-controller prometheus endpoint")
		virtControllerLeaderPodName := libinfra.GetLeader()
		leaderPod, err := virtClient.CoreV1().Pods(flags.KubeVirtInstallNamespace).Get(context.Background(), virtControllerLeaderPodName, metav1.GetOptions{})
		Expect(err).ToNot(HaveOccurred(), "Should find the virt-controller pod")

		for _, ip := range leaderPod.Status.PodIPs {
			controllerMetricIPs = append(controllerMetricIPs, ip.IP)
		}

		// The initial test for the metrics subsystem used only a single VM for the sake of simplicity.
		// However, testing a single entity is a corner case (do we test handling sequences? potential clashes
		// in maps? and so on).
		// Thus, we run now two VMIs per testcase. A more realistic test would use a random number of VMIs >= 3,
		// but we don't do now to make test run quickly and (more important) because lack of resources on CI.

		nodeName := prepareVMIForTests("")
		// any node is fine, we don't really care, as long as we run all VMIs on it.
		prepareVMIForTests(nodeName)

		By("Finding the virt-handler prometheus endpoint")
		pod, err = libnode.GetVirtHandlerPod(virtClient, nodeName)
		Expect(err).ToNot(HaveOccurred(), "Should find the virt-handler pod")
		for _, ip := range pod.Status.PodIPs {
			handlerMetricIPs = append(handlerMetricIPs, ip.IP)
		}

		// returns metrics from the node the VMI(s) runs on
		getKubevirtVMMetrics = tests.GetKubevirtVMMetricsFunc(&virtClient, pod)
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

			cmd := fmt.Sprintf("curl -L -k https://%s:8443/metrics", libnet.FormatIPForURL(ep.IP))
			stdout, stderr, err := exec.ExecuteCommandOnPodWithResults(virtClient, pod, "virt-handler", strings.Fields(cmd))
			Expect(err).ToNot(HaveOccurred(), fmt.Sprintf(remoteCmdErrPattern, cmd, stdout, stderr, err))

			scrapedData := strings.Split(stdout, "\n")
			for _, data := range scrapedData {
				if strings.HasPrefix(data, "#") {
					continue
				}
				switch data {
				case "kubevirt_virt_controller_leading_status 1":
					foundMetrics["leading"]++
				case "kubevirt_virt_controller_ready_status 1":
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

			cmd := fmt.Sprintf("curl -L -k https://%s:8443/metrics", libnet.FormatIPForURL(ep.IP))
			stdout, stderr, err := exec.ExecuteCommandOnPodWithResults(virtClient, pod, "virt-handler", strings.Fields(cmd))
			Expect(err).ToNot(HaveOccurred(), fmt.Sprintf(remoteCmdErrPattern, cmd, stdout, stderr, err))

			scrapedData := strings.Split(stdout, "\n")
			for _, data := range scrapedData {
				if strings.HasPrefix(data, "#") {
					continue
				}
				switch data {
				case "kubevirt_virt_operator_leading_status 1":
					foundMetrics["leading"]++
				case "kubevirt_virt_operator_ready_status 1":
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
		l, err := labels.Parse("prometheus.kubevirt.io=true")
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
			cmd := fmt.Sprintf("curl -L -k https://%s:8443/metrics", libnet.FormatIPForURL(ep.IP))
			stdout, stderr, err := exec.ExecuteCommandOnPodWithResults(virtClient, pod, "virt-handler", strings.Fields(cmd))
			Expect(err).ToNot(HaveOccurred(), fmt.Sprintf(remoteCmdErrPattern, cmd, stdout, stderr, err))
			Expect(stdout).To(ContainSubstring("go_goroutines"))
		}
	})

	DescribeTable("should throttle the Prometheus metrics access", func(family k8sv1.IPFamily) {
		libnet.SkipWhenClusterNotSupportIPFamily(family)

		ip := libinfra.GetSupportedIP(handlerMetricIPs, family)

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
		metricsURL := tests.PrepareMetricsURL(ip, 8443)
		for ix := 0; ix < concurrency; ix++ {
			go func(ix int) {
				req, _ := http.NewRequest("GET", metricsURL, nil)
				resp, err := client.Do(req)
				if err != nil {
					_, fprintfErr := fmt.Fprintf(GinkgoWriter, "client: request: %v #%d: %v\n", req, ix, err) // troubleshooting helper
					Expect(fprintfErr).ToNot(HaveOccurred())
				} else {
					Expect(resp.Body.Close()).To(Succeed())
				}
				errorsChan <- err
			}(ix)
		}

		err := libinfra.ValidatedHTTPResponses(errorsChan, concurrency)
		Expect(err).ToNot(HaveOccurred(), "Should throttle HTTP access without unexpected errors")
	},
		Entry("[test_id:4140] by using IPv4", k8sv1.IPv4Protocol),
		Entry("[test_id:6226] by using IPv6", k8sv1.IPv6Protocol),
	)

	DescribeTable("should include the metrics for a running VM", func(family k8sv1.IPFamily) {
		libnet.SkipWhenClusterNotSupportIPFamily(family)

		ip := libinfra.GetSupportedIP(handlerMetricIPs, family)

		By("Scraping the Prometheus endpoint")
		Eventually(func() string {
			out := getKubevirtVMMetrics(ip)
			lines := libinfra.TakeMetricsWithPrefix(out, "kubevirt")
			return strings.Join(lines, "\n")
		}, 30*time.Second, 2*time.Second).Should(ContainSubstring("kubevirt"))
	},
		Entry("[test_id:4141] by using IPv4", k8sv1.IPv4Protocol),
		Entry("[test_id:6227] by using IPv6", k8sv1.IPv6Protocol),
	)

	DescribeTable("should include the storage metrics for a running VM", func(family k8sv1.IPFamily, metricSubstring, operator string) {
		libnet.SkipWhenClusterNotSupportIPFamily(family)

		ip := libinfra.GetSupportedIP(handlerMetricIPs, family)

		metrics := collectMetrics(ip, metricSubstring)
		By("Checking the collected metrics")
		keys := libinfra.GetKeysFromMetrics(metrics)
		for _, vmi := range preparedVMIs {
			for _, vol := range vmi.Spec.Volumes {
				key := libinfra.GetMetricKeyForVmiDisk(keys, vmi.Name, vol.Name)
				Expect(key).To(Not(BeEmpty()))

				value := metrics[key]
				_, err := fmt.Fprintf(GinkgoWriter, "metric value was %f\n", value)
				Expect(err).ToNot(HaveOccurred())
				Expect(value).To(BeNumerically(operator, float64(0.0)))

			}
		}
	},
		Entry("[test_id:4142] storage flush requests metric by using IPv4", k8sv1.IPv4Protocol, "kubevirt_vmi_storage_flush_requests_total", ">="),
		Entry("[test_id:6228] storage flush requests metric by using IPv6", k8sv1.IPv6Protocol, "kubevirt_vmi_storage_flush_requests_total", ">="),
		Entry("[test_id:4142] time spent on cache flushing metric by using IPv4", k8sv1.IPv4Protocol, "kubevirt_vmi_storage_flush_times_seconds_total", ">="),
		Entry("[test_id:6229] time spent on cache flushing metric by using IPv6", k8sv1.IPv6Protocol, "kubevirt_vmi_storage_flush_times_seconds_total", ">="),
		Entry("[test_id:4142] I/O read operations metric by using IPv4", k8sv1.IPv4Protocol, "kubevirt_vmi_storage_iops_read_total", ">="),
		Entry("[test_id:6230] I/O read operations metric by using IPv6", k8sv1.IPv6Protocol, "kubevirt_vmi_storage_iops_read_total", ">="),
		Entry("[test_id:4142] I/O write operations metric by using IPv4", k8sv1.IPv4Protocol, "kubevirt_vmi_storage_iops_write_total", ">="),
		Entry("[test_id:6231] I/O write operations metric by using IPv6", k8sv1.IPv6Protocol, "kubevirt_vmi_storage_iops_write_total", ">="),
		Entry("[test_id:4142] storage read operation time metric by using IPv4", k8sv1.IPv4Protocol, "kubevirt_vmi_storage_read_times_seconds_total", ">="),
		Entry("[test_id:6232] storage read operation time metric by using IPv6", k8sv1.IPv6Protocol, "kubevirt_vmi_storage_read_times_seconds_total", ">="),
		Entry("[test_id:4142] storage read traffic in bytes metric by using IPv4", k8sv1.IPv4Protocol, "kubevirt_vmi_storage_read_traffic_bytes_total", ">="),
		Entry("[test_id:6233] storage read traffic in bytes metric by using IPv6", k8sv1.IPv6Protocol, "kubevirt_vmi_storage_read_traffic_bytes_total", ">="),
		Entry("[test_id:4142] storage write operation time metric by using IPv4", k8sv1.IPv4Protocol, "kubevirt_vmi_storage_write_times_seconds_total", ">="),
		Entry("[test_id:6234] storage write operation time metric by using IPv6", k8sv1.IPv6Protocol, "kubevirt_vmi_storage_write_times_seconds_total", ">="),
		Entry("[test_id:4142] storage write traffic in bytes metric by using IPv4", k8sv1.IPv4Protocol, "kubevirt_vmi_storage_write_traffic_bytes_total", ">="),
		Entry("[test_id:6235] storage write traffic in bytes metric by using IPv6", k8sv1.IPv6Protocol, "kubevirt_vmi_storage_write_traffic_bytes_total", ">="),
	)

	DescribeTable("should include metrics for a running VM", func(family k8sv1.IPFamily, metricSubstring, operator string) {
		libnet.SkipWhenClusterNotSupportIPFamily(family)

		ip := libinfra.GetSupportedIP(handlerMetricIPs, family)

		metrics := collectMetrics(ip, metricSubstring)
		By("Checking the collected metrics")
		keys := libinfra.GetKeysFromMetrics(metrics)
		for _, key := range keys {
			value := metrics[key]
			_, err := fmt.Fprintf(GinkgoWriter, "metric value was %f\n", value)
			Expect(err).ToNot(HaveOccurred())
			Expect(value).To(BeNumerically(operator, float64(0.0)))
		}
	},
		Entry("[test_id:4143] network metrics by IPv4", k8sv1.IPv4Protocol, "kubevirt_vmi_network_", ">="),
		Entry("[test_id:6236] network metrics by IPv6", k8sv1.IPv6Protocol, "kubevirt_vmi_network_", ">="),
		Entry("[test_id:4144] memory metrics by IPv4", k8sv1.IPv4Protocol, "kubevirt_vmi_memory", ">="),
		Entry("[test_id:6237] memory metrics by IPv6", k8sv1.IPv6Protocol, "kubevirt_vmi_memory", ">="),
		Entry("[test_id:4553] vcpu wait by IPv4", k8sv1.IPv4Protocol, "kubevirt_vmi_vcpu_wait", "=="),
		Entry("[test_id:6238] vcpu wait by IPv6", k8sv1.IPv6Protocol, "kubevirt_vmi_vcpu_wait", "=="),
		Entry("[test_id:4554] vcpu seconds by IPv4", k8sv1.IPv4Protocol, "kubevirt_vmi_vcpu_seconds_total", ">="),
		Entry("[test_id:6239] vcpu seconds by IPv6", k8sv1.IPv6Protocol, "kubevirt_vmi_vcpu_seconds_total", ">="),
		Entry("[test_id:4556] vmi unused memory by IPv4", k8sv1.IPv4Protocol, "kubevirt_vmi_memory_unused_bytes", ">="),
		Entry("[test_id:6240] vmi unused memory by IPv6", k8sv1.IPv6Protocol, "kubevirt_vmi_memory_unused_bytes", ">="),
	)

	DescribeTable("should include VMI infos for a running VM", func(family k8sv1.IPFamily) {
		libnet.SkipWhenClusterNotSupportIPFamily(family)

		ip := libinfra.GetSupportedIP(handlerMetricIPs, family)

		metrics := collectMetrics(ip, "kubevirt_vmi_")
		By("Checking the collected metrics")
		keys := libinfra.GetKeysFromMetrics(metrics)
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
		Entry("[test_id:4145] by IPv4", k8sv1.IPv4Protocol),
		Entry("[test_id:6241] by IPv6", k8sv1.IPv6Protocol),
	)

	DescribeTable("should include VMI phase metrics for all running VMs", func(family k8sv1.IPFamily) {
		libnet.SkipWhenClusterNotSupportIPFamily(family)

		ip := libinfra.GetSupportedIP(handlerMetricIPs, family)

		metrics := collectMetrics(ip, "kubevirt_vmi_")
		By("Checking the collected metrics")
		keys := libinfra.GetKeysFromMetrics(metrics)
		for _, key := range keys {
			if strings.Contains(key, `phase="running"`) {
				value := metrics[key]
				Expect(value).To(Equal(float64(len(preparedVMIs))))
			}
		}
	},
		Entry("[test_id:4146] by IPv4", k8sv1.IPv4Protocol),
		Entry("[test_id:6242] by IPv6", k8sv1.IPv6Protocol),
	)

	DescribeTable("should include VMI eviction blocker status for all running VMs", func(family k8sv1.IPFamily) {
		libnet.SkipWhenClusterNotSupportIPFamily(family)

		ip := libinfra.GetSupportedIP(controllerMetricIPs, family)

		metrics := collectMetrics(ip, "kubevirt_vmi_non_evictable")
		By("Checking the collected metrics")
		keys := libinfra.GetKeysFromMetrics(metrics)
		for _, key := range keys {
			value := metrics[key]
			_, err := fmt.Fprintf(GinkgoWriter, "metric value was %f\n", value)
			Expect(err).ToNot(HaveOccurred())
			Expect(value).To(BeNumerically(">=", float64(0.0)))
		}
	},
		Entry("[test_id:4148] by IPv4", k8sv1.IPv4Protocol),
		Entry("[test_id:6243] by IPv6", k8sv1.IPv6Protocol),
	)

	DescribeTable("should include kubernetes labels to VMI metrics", func(family k8sv1.IPFamily) {
		libnet.SkipWhenClusterNotSupportIPFamily(family)

		ip := libinfra.GetSupportedIP(handlerMetricIPs, family)

		// Every VMI is labeled with kubevirt.io/nodeName, so just creating a VMI should
		// be enough to its metrics to contain a kubernetes label
		metrics := collectMetrics(ip, "kubevirt_vmi_vcpu_seconds_total")
		By("Checking collected metrics")
		keys := libinfra.GetKeysFromMetrics(metrics)
		containK8sLabel := false
		for _, key := range keys {
			if strings.Contains(key, "kubernetes_vmi_label_") {
				containK8sLabel = true
			}
		}
		Expect(containK8sLabel).To(BeTrue())
	},
		Entry("[test_id:4147] by IPv4", k8sv1.IPv4Protocol),
		Entry("[test_id:6244] by IPv6", k8sv1.IPv6Protocol),
	)

	// explicit test fo swap metrics as test_id:4144 doesn't catch if they are missing
	DescribeTable("should include swap metrics", func(family k8sv1.IPFamily) {
		libnet.SkipWhenClusterNotSupportIPFamily(family)

		ip := libinfra.GetSupportedIP(handlerMetricIPs, family)

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
		Entry("[test_id:4555] by IPv4", k8sv1.IPv4Protocol),
		Entry("[test_id:6245] by IPv6", k8sv1.IPv6Protocol),
	)
})
