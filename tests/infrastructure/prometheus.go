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
 * Copyright The KubeVirt Authors.
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

	expect "github.com/google/goexpect"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	gomegatypes "github.com/onsi/gomega/types"
	metricsutil "github.com/rhobs/operator-observability-toolkit/pkg/testutil"
	authenticationv1 "k8s.io/api/authentication/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	netutils "k8s.io/utils/net"

	"kubevirt.io/kubevirt/tests/framework/k8s"

	"kubevirt.io/kubevirt/tests/decorators"
	"kubevirt.io/kubevirt/tests/libnode"

	"kubevirt.io/kubevirt/tests/libwait"

	k8sv1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	v1 "kubevirt.io/api/core/v1"
	"kubevirt.io/client-go/kubecli"

	"kubevirt.io/kubevirt/pkg/libvmi"
	"kubevirt.io/kubevirt/tests/console"
	"kubevirt.io/kubevirt/tests/exec"
	"kubevirt.io/kubevirt/tests/flags"
	"kubevirt.io/kubevirt/tests/framework/kubevirt"
	"kubevirt.io/kubevirt/tests/libinfra"
	"kubevirt.io/kubevirt/tests/libmonitoring"
	"kubevirt.io/kubevirt/tests/libnet"
	"kubevirt.io/kubevirt/tests/libvmifact"
	"kubevirt.io/kubevirt/tests/libvmops"
	"kubevirt.io/kubevirt/tests/testsuite"
)

const (
	remoteCmdErrPattern = "failed running `%s` with stdout:\n %v \n stderr:\n %v \n err: \n %v "
)

var _ = Describe("[sig-monitoring][rfe_id:3187][crit:medium][vendor:cnv-qe@redhat.com][level:component]Prometheus scraped metrics", decorators.SigMonitoring, func() { //nolint:lll
	var virtClient kubecli.KubevirtClient

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
	})

	/*
		This test is querying the metrics from Prometheus *after* they were
		scraped and processed by the different components on the way.
	*/

	It("[QUARANTINE][test_id:4135]should find VMI namespace on namespace label of the metric", decorators.Quarantine, func() {
		/*
			This test is required because in cases of misconfigurations on
			monitoring objects (such for the ServiceMonitor), our rules will
			still be picked up by the monitoring-operator, but Prometheus
			will fail to load it.
		*/

		By("creating a VMI in a user defined namespace")
		vmi := libvmifact.NewAlpine()
		vmi.Namespace = testsuite.GetTestNamespace(vmi)
		startVMI(vmi)

		By("finding virt-handler pod")
		ops, err := k8s.Client().CoreV1().Pods(flags.KubeVirtInstallNamespace).List(
			context.Background(),
			metav1.ListOptions{LabelSelector: "kubevirt.io=virt-handler"})
		Expect(err).ToNot(HaveOccurred(), "failed to list virt-handlers")
		Expect(ops.Size()).ToNot(Equal(0), "no virt-handlers found")
		op := ops.Items[0]
		Expect(op).ToNot(BeNil(), "virt-handler pod should not be nil")

		urlSchema := "https"
		promPort := 9091
		if flags.PrometheusNamespace == "monitoring" {
			urlSchema = "http"
			promPort = 9090
		}
		promServiceURL := fmt.Sprintf("prometheus-k8s.%s.svc.cluster.local", flags.PrometheusNamespace)

		// the Service Account needs to have access to the Prometheus subresource api
		token, err := generateTokenForPrometheusAPI(vmi.Namespace)
		Expect(err).ToNot(HaveOccurred(), "failed to generate token for Prometheus API")
		DeferCleanup(cleanupClusterRoleAndBinding, vmi.Namespace)

		By("querying Prometheus API endpoint for a VMI exported metric")
		cmd := []string{
			"curl",
			"-L",
			"-k",
			fmt.Sprintf("%s://%s:%d/api/v1/query", urlSchema, promServiceURL, promPort),
			"-H",
			fmt.Sprintf("Authorization: Bearer %s", token),
			"--data-urlencode",
			fmt.Sprintf(
				`query=kubevirt_vmi_memory_resident_bytes{namespace=%q,name=%q}`,
				vmi.Namespace,
				vmi.Name,
			),
		}

		stdout, stderr, err := exec.ExecuteCommandOnPodWithResults(&op, "virt-handler", cmd)
		Expect(err).ToNot(HaveOccurred(), fmt.Sprintf(remoteCmdErrPattern, strings.Join(cmd, " "), stdout, stderr, err))

		// the Prometheus go-client does not export queryResult, and
		// using an HTTP client for queries would require a port-forwarding
		// since the cluster is running in a different network.
		var queryResult map[string]json.RawMessage

		err = json.Unmarshal([]byte(stdout), &queryResult)
		Expect(err).ToNot(HaveOccurred(), "failed to unmarshal query result: %s", stdout)

		var status string
		err = json.Unmarshal(queryResult["status"], &status)
		Expect(err).ToNot(HaveOccurred(), "failed to unmarshal query status")
		Expect(status).To(Equal("success"))
	})
})

var _ = Describe(SIGSerial("[rfe_id:3187][crit:medium][vendor:cnv-qe@redhat.com][level:component]Prometheus Endpoints", func() {
	var (
		preparedVMIs        []*v1.VirtualMachineInstance
		pod                 *k8sv1.Pod
		handlerMetricIPs    []string
		controllerMetricIPs []string
	)

	prepareVMIForTests := func(preferredNodeName string) string {
		By("Creating the VirtualMachineInstance")

		// WARNING: we assume the VM will have a VirtIO disk (vda)
		// and we add our own vdb on which we do our test.
		// but if the default disk is not vda, the test will break
		// TODO: introspect the VMI and get the device name of this
		// block device?
		options := []libvmi.Option{
			libvmi.WithEmptyDisk("testdisk", v1.VirtIO, resource.MustParse("1G")),
		}

		if preferredNodeName != "" {
			options = append(options, libvmi.WithNodeSelectorFor(preferredNodeName))
		}

		vmi := libvmifact.NewAlpine(options...)

		vmi = libvmops.RunVMIAndExpectLaunch(vmi, libvmops.StartupTimeoutSecondsTiny)
		nodeName := vmi.Status.NodeName

		By("Expecting the VirtualMachineInstance console")
		// This also serves as a sync point to make sure the VM completed the boot
		// (and reduce the risk of false negatives)
		Expect(console.LoginToAlpine(vmi)).To(Succeed())
		const expectTimeout = 10
		By("Writing some data to the disk")
		Expect(console.SafeExpectBatch(vmi, []expect.Batcher{
			&expect.BSnd{S: "dd if=/dev/zero of=/dev/vdb bs=1M count=1\n"},
			&expect.BExp{R: ""},
			&expect.BSnd{S: "sync\n"},
			&expect.BExp{R: ""},
		}, expectTimeout)).To(Succeed())

		preparedVMIs = append(preparedVMIs, vmi)
		return nodeName
	}

	BeforeEach(func() {
		preparedVMIs = []*v1.VirtualMachineInstance{}
		pod = nil
		handlerMetricIPs = []string{}
		controllerMetricIPs = []string{}

		By("Finding the virt-controller prometheus endpoint")
		virtControllerLeaderPodName := libinfra.GetLeader()
		leaderPod, err := k8s.Client().CoreV1().Pods(flags.KubeVirtInstallNamespace).Get(
			context.Background(), virtControllerLeaderPodName, metav1.GetOptions{})
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
		pod, err = libnode.GetVirtHandlerPod(k8s.Client(), nodeName)
		Expect(err).ToNot(HaveOccurred(), "Should find the virt-handler pod")
		for _, ip := range pod.Status.PodIPs {
			handlerMetricIPs = append(handlerMetricIPs, ip.IP)
		}
	})

	It("[test_id:4136] should find one leading virt-controller and two ready", func() {
		By("scraping the metrics endpoint on virt-controller pods")
		results, err := countReadyAndLeaderPods(pod, "controller")
		const expectedReady = 2
		Expect(err).ToNot(HaveOccurred())
		Expect(results["ready"]).To(Equal(expectedReady), "expected 2 ready virt-controllers")
		Expect(results["leading"]).To(Equal(1), "expected 1 leading virt-controller")
	})

	It("[test_id:4137]should find one leading virt-operator and two ready", func() {
		By("scraping the metrics endpoint on virt-operator pods")
		results, err := countReadyAndLeaderPods(pod, "operator")
		const expectedReady = 2
		Expect(err).ToNot(HaveOccurred())
		Expect(results["ready"]).To(Equal(expectedReady), "expected 2 ready virt-operators")
		Expect(results["leading"]).To(Equal(1), "expected 1 leading virt-operator")
	})

	It("[test_id:4138]should be exposed and registered on the metrics endpoint", func() {
		epsList, err := k8s.Client().DiscoveryV1().EndpointSlices(flags.KubeVirtInstallNamespace).List(
			context.Background(), metav1.ListOptions{
				LabelSelector: "kubernetes.io/service-name=kubevirt-prometheus-metrics",
			})
		Expect(err).ToNot(HaveOccurred())
		// should only have one EndpointSlice
		Expect(epsList.Items).To(HaveLen(1), "Expected exactly one EndpointSlice")
		epSlice := epsList.Items[0]

		l, err := labels.Parse("prometheus.kubevirt.io=true")
		Expect(err).ToNot(HaveOccurred())
		pods, err := k8s.Client().CoreV1().Pods(flags.KubeVirtInstallNamespace).List(
			context.Background(), metav1.ListOptions{LabelSelector: l.String()})
		Expect(err).ToNot(HaveOccurred())

		By("checking if the endpoint contains the metrics port")
		const metricsPort = 8443
		// we don't need to check the length, the endpointSlice Obj only contains one port
		Expect(epSlice.Ports[0].Name).ToNot(BeNil())
		Expect(*epSlice.Ports[0].Name).To(Equal("metrics"))
		Expect(epSlice.Ports[0].Port).ToNot(BeNil())
		Expect(*epSlice.Ports[0].Port).To(Equal(int32(metricsPort)))

		By("checking if the IPs in the endpoint slice match the KubeVirt system Pod count")
		const minVirtPods = 3
		Expect(len(pods.Items)).To(BeNumerically(">=", minVirtPods), "At least one api, controller and handler need to be present")
		Expect(epSlice.Endpoints).To(HaveLen(len(pods.Items)))

		ips := map[string]string{}
		for _, ep := range epSlice.Endpoints {
			for _, addr := range ep.Addresses {
				ips[addr] = ""
			}
		}
		for _, pod := range pods.Items {
			Expect(ips).To(HaveKey(pod.Status.PodIP), fmt.Sprintf("IP of Pod %s not found in metrics endpoint", pod.Name))
		}
	})
	It("[test_id:4139]should return Prometheus metrics", func() {
		endpointSliceList, err := k8s.Client().DiscoveryV1().EndpointSlices(flags.KubeVirtInstallNamespace).List(
			context.Background(), metav1.ListOptions{
				LabelSelector: "kubernetes.io/service-name=kubevirt-prometheus-metrics",
			})
		Expect(err).ToNot(HaveOccurred())
		Expect(endpointSliceList.Items).ToNot(BeEmpty(), "Expected at least one EndpointSlice")

		for _, epSlice := range endpointSliceList.Items {
			for _, ep := range epSlice.Endpoints {
				for _, addr := range ep.Addresses {
					cmd := fmt.Sprintf("curl -L -k https://%s:8443/metrics", libnet.FormatIPForURL(addr))
					stdout, stderr, err := exec.ExecuteCommandOnPodWithResults(pod, "virt-handler", strings.Fields(cmd))
					Expect(err).ToNot(HaveOccurred(), fmt.Sprintf(remoteCmdErrPattern, cmd, stdout, stderr, err))
					Expect(stdout).To(ContainSubstring("go_goroutines"))
				}
			}
		}
	})

	DescribeTable("should throttle the Prometheus metrics access", func(family k8sv1.IPFamily) {
		libnet.SkipWhenClusterNotSupportIPFamily(family)

		ip := libnet.GetIP(handlerMetricIPs, family)

		if netutils.IsIPv6String(ip) {
			Skip("Skip testing with IPv6 until https://github.com/kubevirt/kubevirt/issues/4145 is fixed")
		}

		concurrency := 100 // random value "much higher" than maxRequestsInFlight

		tr := &http.Transport{
			MaxIdleConnsPerHost: concurrency,
			TLSClientConfig: &tls.Config{
				//nolint:gosec
				InsecureSkipVerify: true,
			},
		}

		client := http.Client{
			Timeout:   time.Duration(1),
			Transport: tr,
		}

		errorsChan := make(chan error)
		By("Scraping the Prometheus endpoint")
		const metricsPort = 8443
		metricsURL := libmonitoring.PrepareMetricsURL(ip, metricsPort)
		for ix := 0; ix < concurrency; ix++ {
			go func(ix int) {
				req, _ := http.NewRequest("GET", metricsURL, http.NoBody)
				resp, err := client.Do(req)
				if err != nil {
					GinkgoLogr.Info("client request", "request", req, "index", ix, "error", err)
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

		ip := libnet.GetIP(handlerMetricIPs, family)

		By("Scraping the Prometheus endpoint")
		Eventually(func() string {
			out := libmonitoring.GetKubevirtVMMetrics(pod, ip)
			lines := libinfra.TakeMetricsWithPrefix(out, "kubevirt")
			return strings.Join(lines, "\n")
		}, 30*time.Second, 2*time.Second).Should(ContainSubstring("kubevirt"))
	},
		Entry("[test_id:4141] by using IPv4", k8sv1.IPv4Protocol),
		Entry("[test_id:6227] by using IPv6", k8sv1.IPv6Protocol),
	)

	It("should expose kubevirt_node_deprecated_machine_types metric", func() {
		ip := libnet.GetIP(handlerMetricIPs, k8sv1.IPv4Protocol)

		metricsPayload := libmonitoring.GetKubevirtVMMetrics(pod, ip)

		fetcher := metricsutil.NewMetricsFetcher("")
		fetcher.AddNameFilter("kubevirt_node_deprecated_machine_types")

		metrics, err := fetcher.LoadMetrics(metricsPayload)
		Expect(err).ToNot(HaveOccurred())

		results := metrics["kubevirt_node_deprecated_machine_types"]
		Expect(results).ToNot(BeEmpty(), "Expected to find deprecated machine types metrics")

		for _, result := range results {
			Expect(result.Labels).To(HaveKey("machine_type"), fmt.Sprintf("Expected machine_type label in metric: %+v", result))
		}
	})

	DescribeTable("should include the storage metrics for a running VM", func(family k8sv1.IPFamily, metricName, operator string) {
		libnet.SkipWhenClusterNotSupportIPFamily(family)

		ip := libnet.GetIP(handlerMetricIPs, family)
		metricsPayload := libmonitoring.GetKubevirtVMMetrics(pod, ip)

		By("Checking the collected metrics")
		for _, vmi := range preparedVMIs {
			for _, vol := range vmi.Spec.Volumes {
				fetcher := metricsutil.NewMetricsFetcher("")
				fetcher.AddNameFilter(metricName)
				fetcher.AddLabelFilter("name", vmi.Name, "drive", vol.Name)

				metrics, err := fetcher.LoadMetrics(metricsPayload)
				Expect(err).ToNot(HaveOccurred(), "should load metrics without error")

				results := metrics[metricName]
				Expect(results).ToNot(BeEmpty(), "Expected to find metric for VMI %s and disk %s", vmi.Name, vol.Name)

				GinkgoLogr.Info("Metric value", "value", results[0].Value)
				Expect(results[0].Value).To(BeNumerically(operator, 0.0))
			}
		}
	},
		Entry("[test_id:4142] storage flush requests metric by using IPv4",
			k8sv1.IPv4Protocol, "kubevirt_vmi_storage_flush_requests_total", ">="),
		Entry("[test_id:6228] storage flush requests metric by using IPv6",
			k8sv1.IPv6Protocol, "kubevirt_vmi_storage_flush_requests_total", ">="),
		Entry("[test_id:4142] time spent on cache flushing metric by using IPv4",
			k8sv1.IPv4Protocol, "kubevirt_vmi_storage_flush_times_seconds_total", ">="),
		Entry("[test_id:6229] time spent on cache flushing metric by using IPv6",
			k8sv1.IPv6Protocol, "kubevirt_vmi_storage_flush_times_seconds_total", ">="),
		Entry("[test_id:4142] I/O read operations metric by using IPv4", k8sv1.IPv4Protocol, "kubevirt_vmi_storage_iops_read_total", ">="),
		Entry("[test_id:6230] I/O read operations metric by using IPv6", k8sv1.IPv6Protocol, "kubevirt_vmi_storage_iops_read_total", ">="),
		Entry("[test_id:4142] I/O write operations metric by using IPv4", k8sv1.IPv4Protocol, "kubevirt_vmi_storage_iops_write_total", ">="),
		Entry("[test_id:6231] I/O write operations metric by using IPv6", k8sv1.IPv6Protocol, "kubevirt_vmi_storage_iops_write_total", ">="),
		Entry("[test_id:4142] storage read operation time metric by using IPv4",
			k8sv1.IPv4Protocol, "kubevirt_vmi_storage_read_times_seconds_total", ">="),
		Entry("[test_id:6232] storage read operation time metric by using IPv6",
			k8sv1.IPv6Protocol, "kubevirt_vmi_storage_read_times_seconds_total", ">="),
		Entry("[test_id:4142] storage read traffic in bytes metric by using IPv4",
			k8sv1.IPv4Protocol, "kubevirt_vmi_storage_read_traffic_bytes_total", ">="),
		Entry("[test_id:6233] storage read traffic in bytes metric by using IPv6",
			k8sv1.IPv6Protocol, "kubevirt_vmi_storage_read_traffic_bytes_total", ">="),
		Entry("[test_id:4142] storage write operation time metric by using IPv4",
			k8sv1.IPv4Protocol, "kubevirt_vmi_storage_write_times_seconds_total", ">="),
		Entry("[test_id:6234] storage write operation time metric by using IPv6",
			k8sv1.IPv6Protocol, "kubevirt_vmi_storage_write_times_seconds_total", ">="),
		Entry("[test_id:4142] storage write traffic in bytes metric by using IPv4",
			k8sv1.IPv4Protocol, "kubevirt_vmi_storage_write_traffic_bytes_total", ">="),
		Entry("[test_id:6235] storage write traffic in bytes metric by using IPv6",
			k8sv1.IPv6Protocol, "kubevirt_vmi_storage_write_traffic_bytes_total", ">="),
	)

	DescribeTable("should include metrics for a running VM", func(family k8sv1.IPFamily, metricSubstring, operator string) {
		libnet.SkipWhenClusterNotSupportIPFamily(family)

		ip := libnet.GetIP(handlerMetricIPs, family)
		metricsPayload := libmonitoring.GetKubevirtVMMetrics(pod, ip)

		fetcher := metricsutil.NewMetricsFetcher("")
		fetcher.AddNameFilter(metricSubstring)

		metrics, err := fetcher.LoadMetrics(metricsPayload)
		Expect(err).ToNot(HaveOccurred(), "should load metrics without error")
		Expect(metrics).ToNot(BeEmpty(), "Expected at least one metric to be collected for %s", metricSubstring)

		for _, results := range metrics {
			for _, result := range results {
				Expect(result.Value).To(BeNumerically(operator, float64(0.0)))
			}
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

	DescribeTable("[QUARANTINE]should include VMI infos for a running VM", decorators.Quarantine, func(family k8sv1.IPFamily) {
		libnet.SkipWhenClusterNotSupportIPFamily(family)

		ip := libnet.GetIP(handlerMetricIPs, family)
		metricsPayload := libmonitoring.GetKubevirtVMMetrics(pod, ip)

		fetcher := metricsutil.NewMetricsFetcher("")
		fetcher.AddNameFilter("kubevirt_vmi_")

		metrics, err := fetcher.LoadMetrics(metricsPayload)
		Expect(err).ToNot(HaveOccurred())

		By("Checking the collected metrics")
		nodeName := pod.Spec.NodeName

		var nameMatchers []gomegatypes.GomegaMatcher
		for _, vmi := range preparedVMIs {
			nameMatchers = append(nameMatchers, HaveKeyWithValue("name", vmi.Name))
		}

		for metricName, results := range metrics {
			// we don't care about the ordering of the labels
			if strings.HasPrefix(metricName, "kubevirt_vmi_info") {
				// special case: namespace and name don't make sense for this metric
				for _, metricResult := range results {
					Expect(metricResult.Labels).To(HaveKeyWithValue("node", nodeName))
				}
				continue
			}

			for _, metricResult := range results {
				Expect(metricResult.Labels).To(SatisfyAll(
					HaveKeyWithValue("node", nodeName),
					// all testing VMIs are on the same node and namespace,
					// so checking the namespace of any random VMI is fine
					HaveKeyWithValue("namespace", preparedVMIs[0].Namespace),
					// otherwise, each result must refer to exactly one the prepared VMIs.
					SatisfyAny(nameMatchers...),
				))
			}
		}
	},
		Entry("[test_id:4145] by IPv4", k8sv1.IPv4Protocol),
		Entry("[test_id:6241] by IPv6", k8sv1.IPv6Protocol),
	)

	DescribeTable("should include VMI phase metrics for all running VMs", func(family k8sv1.IPFamily) {
		libnet.SkipWhenClusterNotSupportIPFamily(family)

		ip := libnet.GetIP(handlerMetricIPs, family)
		metricsPayload := libmonitoring.GetKubevirtVMMetrics(pod, ip)

		fetcher := metricsutil.NewMetricsFetcher("")
		fetcher.AddNameFilter("kubevirt_vmi_")
		fetcher.AddLabelFilter("phase", "Running")

		metrics, err := fetcher.LoadMetrics(metricsPayload)
		Expect(err).ToNot(HaveOccurred())

		By("Checking the collected metrics")
		for _, results := range metrics {
			for _, metricResult := range results {
				Expect(metricResult.Value).To(Equal(float64(len(preparedVMIs))))
			}
		}
	},
		Entry("[test_id:4146] by IPv4", k8sv1.IPv4Protocol),
		Entry("[test_id:6242] by IPv6", k8sv1.IPv6Protocol),
	)

	DescribeTable("should include VMI eviction blocker status for all running VMs", func(family k8sv1.IPFamily) {
		libnet.SkipWhenClusterNotSupportIPFamily(family)

		ip := libnet.GetIP(controllerMetricIPs, family)

		metricsPayload := libmonitoring.GetKubevirtVMMetrics(pod, ip)

		fetcher := metricsutil.NewMetricsFetcher("")
		fetcher.AddNameFilter("kubevirt_vmi_non_evictable")

		metrics, err := fetcher.LoadMetrics(metricsPayload)
		Expect(err).ToNot(HaveOccurred())
		Expect(metrics).ToNot(BeEmpty(), "Expected at least one metric to be collected")

		results := metrics["kubevirt_vmi_non_evictable"]
		Expect(results).ToNot(BeEmpty())
		Expect(results[0].Value).To(BeNumerically(">=", float64(0.0)))
	},
		Entry("[test_id:4148] by IPv4", k8sv1.IPv4Protocol),
		Entry("[test_id:6243] by IPv6", k8sv1.IPv6Protocol),
	)

	DescribeTable("should include kubernetes labels to VMI metrics", func(family k8sv1.IPFamily) {
		libnet.SkipWhenClusterNotSupportIPFamily(family)

		ip := libnet.GetIP(handlerMetricIPs, family)

		metricsPayload := libmonitoring.GetKubevirtVMMetrics(pod, ip)

		fetcher := metricsutil.NewMetricsFetcher("")
		fetcher.AddNameFilter("kubevirt_vmi_vcpu_seconds_total")

		metrics, err := fetcher.LoadMetrics(metricsPayload)
		Expect(err).ToNot(HaveOccurred())
		Expect(metrics).ToNot(BeEmpty(), "Expected at least one metric to be collected")

		// Every VMI is labeled with kubevirt.io/nodeName, so just creating a VMI should
		// be enough to its metrics to contain a kubernetes label
		containK8sLabel := false
		for _, results := range metrics {
			for _, metricResult := range results {
				for label := range metricResult.Labels {
					if strings.Contains(label, "kubernetes_vmi_label_") {
						containK8sLabel = true
					}
				}
			}
		}
		Expect(containK8sLabel).To(BeTrue())
	},
		Entry("[test_id:4147] by IPv4", k8sv1.IPv4Protocol),
		Entry("[test_id:6244] by IPv6", k8sv1.IPv6Protocol),
	)

	// explicit test swap metrics as test_id:4144 doesn't catch if they are missing
	DescribeTable("should include swap metrics", func(family k8sv1.IPFamily) {
		libnet.SkipWhenClusterNotSupportIPFamily(family)

		ip := libnet.GetIP(handlerMetricIPs, family)

		metricsPayload := libmonitoring.GetKubevirtVMMetrics(pod, ip)

		fetcher := metricsutil.NewMetricsFetcher("")
		fetcher.AddNameFilter("kubevirt_vmi_memory_swap_")

		metrics, err := fetcher.LoadMetrics(metricsPayload)
		Expect(err).ToNot(HaveOccurred())
		Expect(metrics).ToNot(BeEmpty(), "Expected at least one metric to be collected")

		var in, out bool
		for metricName := range metrics {
			if in && out {
				break
			}
			if strings.Contains(metricName, "swap_in") {
				in = true
			}
			if strings.Contains(metricName, "swap_out") {
				out = true
			}
		}

		Expect(in).To(BeTrue())
		Expect(out).To(BeTrue())
	},
		Entry("[test_id:4555] by IPv4", k8sv1.IPv4Protocol),
		Entry("[test_id:6245] by IPv6", k8sv1.IPv6Protocol),
	)
}))

func countReadyAndLeaderPods(pod *k8sv1.Pod, component string) (foundMetrics map[string]int, err error) {
	target := fmt.Sprintf("virt-%s", component)
	leadingMetric := fmt.Sprintf("kubevirt_virt_%s_leading_status 1", component)
	readyMetric := fmt.Sprintf("kubevirt_virt_%s_ready_status 1", component)
	endpointSliceList, err := k8s.Client().DiscoveryV1().EndpointSlices(flags.KubeVirtInstallNamespace).List(
		context.Background(), metav1.ListOptions{
			LabelSelector: "kubernetes.io/service-name=kubevirt-prometheus-metrics",
		})
	if err != nil {
		return nil, err
	}
	foundMetrics = map[string]int{
		"ready":   0,
		"leading": 0,
	}
	for _, epSlice := range endpointSliceList.Items {
		for _, ep := range epSlice.Endpoints {
			if ep.TargetRef == nil || !strings.HasPrefix(ep.TargetRef.Name, target) {
				continue
			}

			for _, addr := range ep.Addresses {
				cmd := fmt.Sprintf("curl -L -k https://%s:8443/metrics", libnet.FormatIPForURL(addr))
				var stdout, stderr string
				stdout, stderr, err = exec.ExecuteCommandOnPodWithResults(pod, "virt-handler", strings.Fields(cmd))
				if err != nil {
					return nil, fmt.Errorf(remoteCmdErrPattern, cmd, stdout, stderr, err)
				}
				foundMetrics["leading"] += strings.Count(stdout, leadingMetric)
				foundMetrics["ready"] += strings.Count(stdout, readyMetric)
			}
		}
	}

	return foundMetrics, err
}

func generateTokenForPrometheusAPI(namespace string) (string, error) {
	// Define resource names
	serviceAccountName := "prometheus-access-sa"
	clusterRoleName := "prometheus-access-cluster-role"
	clusterRoleBindingName := "prometheus-access-cluster-rolebinding"

	// Create ServiceAccount
	sa := &k8sv1.ServiceAccount{
		ObjectMeta: metav1.ObjectMeta{
			Name:      serviceAccountName,
			Namespace: namespace,
		},
	}
	_, err := k8s.Client().CoreV1().ServiceAccounts(namespace).Create(context.Background(), sa, metav1.CreateOptions{})
	if err != nil {
		return "", fmt.Errorf("failed to create ServiceAccount: %w", err)
	}

	// Create ClusterRole
	clusterRole := &rbacv1.ClusterRole{
		ObjectMeta: metav1.ObjectMeta{
			Name: clusterRoleName + "-" + namespace, // Namespaced suffix for uniqueness
		},
		Rules: []rbacv1.PolicyRule{
			{
				APIGroups: []string{"monitoring.coreos.com"},
				Resources: []string{"prometheuses/api"},
				Verbs:     []string{"create"},
			},
		},
	}
	_, err = k8s.Client().RbacV1().ClusterRoles().Create(context.Background(), clusterRole, metav1.CreateOptions{})
	if err != nil {
		return "", fmt.Errorf("failed to create ClusterRole: %w", err)
	}

	// Create ClusterRoleBinding
	clusterRoleBinding := &rbacv1.ClusterRoleBinding{
		ObjectMeta: metav1.ObjectMeta{
			Name: clusterRoleBindingName + "-" + namespace, // Namespaced suffix for uniqueness
		},
		Subjects: []rbacv1.Subject{
			{
				Kind:      "ServiceAccount",
				Name:      serviceAccountName,
				Namespace: namespace,
			},
		},
		RoleRef: rbacv1.RoleRef{
			Kind:     "ClusterRole",
			Name:     clusterRoleName + "-" + namespace, // Match the ClusterRole name
			APIGroup: "rbac.authorization.k8s.io",
		},
	}
	_, err = k8s.Client().RbacV1().ClusterRoleBindings().Create(context.Background(), clusterRoleBinding, metav1.CreateOptions{})
	if err != nil {
		return "", fmt.Errorf("failed to create ClusterRoleBinding: %w", err)
	}

	// Retrieve token for the ServiceAccount
	tokenRequest := &authenticationv1.TokenRequest{
		Spec: authenticationv1.TokenRequestSpec{},
	}
	token, err := k8s.Client().CoreV1().
		ServiceAccounts(namespace).
		CreateToken(
			context.Background(),
			serviceAccountName,
			tokenRequest,
			metav1.CreateOptions{},
		)
	if err != nil {
		return "", fmt.Errorf("failed to retrieve ServiceAccount token: %w", err)
	}

	// Return the token
	return token.Status.Token, nil
}

func cleanupClusterRoleAndBinding(namespace string) {
	clusterRoleName := "prometheus-access-cluster-role-" + namespace
	clusterRoleBindingName := "prometheus-access-cluster-rolebinding-" + namespace

	// Delete ClusterRole
	err := k8s.Client().RbacV1().ClusterRoles().Delete(context.Background(), clusterRoleName, metav1.DeleteOptions{})
	Expect(err).ToNot(HaveOccurred(), "Failed to delete ClusterRole: %s", clusterRoleName)

	// Delete ClusterRoleBinding
	err = k8s.Client().RbacV1().ClusterRoleBindings().Delete(context.Background(), clusterRoleBindingName, metav1.DeleteOptions{})
	Expect(err).ToNot(HaveOccurred(), "Failed to delete ClusterRoleBinding: %s", clusterRoleBindingName)
}
