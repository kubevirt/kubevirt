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
 * Copyright The Kubevirt Authors
 *
 */

package monitoring

import (
	"context"
	"strconv"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/machadovilaca/operator-observability/pkg/operatormetrics"
	"github.com/onsi/gomega/types"
	"github.com/prometheus/client_golang/prometheus"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	v1 "kubevirt.io/api/core/v1"
	"kubevirt.io/client-go/kubecli"

	"kubevirt.io/kubevirt/pkg/libvmi"
	virtapi "kubevirt.io/kubevirt/pkg/monitoring/metrics/virt-api"
	virtcontroller "kubevirt.io/kubevirt/pkg/monitoring/metrics/virt-controller"
	virtoperator "kubevirt.io/kubevirt/pkg/monitoring/metrics/virt-operator"

	"kubevirt.io/kubevirt/tests/decorators"
	"kubevirt.io/kubevirt/tests/framework/kubevirt"
	"kubevirt.io/kubevirt/tests/libmonitoring"
	"kubevirt.io/kubevirt/tests/libvmifact"
	"kubevirt.io/kubevirt/tests/libwait"
	"kubevirt.io/kubevirt/tests/testsuite"
)

var _ = Describe("[sig-monitoring]Metrics", decorators.SigMonitoring, func() {
	var virtClient kubecli.KubevirtClient
	var metrics *libmonitoring.QueryRequestResult

	BeforeEach(func() {
		virtClient = kubevirt.Client()
		setupVM(virtClient)
		metrics = fetchPrometheusMetrics(virtClient)
	})

	Context("Prometheus metrics", func() {
		var excludedMetrics = map[string]bool{
			// virt-api
			// can later be added in pre-existing feature tests
			"kubevirt_portforward_active_tunnels":  true,
			"kubevirt_usbredir_active_connections": true,
			"kubevirt_vnc_active_connections":      true,
			"kubevirt_console_active_connections":  true,

			// virt-controller
			// needs a migration - ignoring since already tested in - VM Monitoring, VM migration metrics
			"kubevirt_vmi_migration_phase_transition_time_from_creation_seconds": true,
			"kubevirt_vmi_migrations_in_pending_phase":                           true,
			"kubevirt_vmi_migrations_in_scheduling_phase":                        true,
			"kubevirt_vmi_migrations_in_running_phase":                           true,
			"kubevirt_vmi_migration_succeeded":                                   true,
			"kubevirt_vmi_migration_failed":                                      true,

			// name do not follow the convention to be prefixed with 'kubevirt_'
			// TODO: @machadovilaca - refactor the metric names
			"rest_client_request_latency_seconds":       true,
			"rest_client_rate_limiter_duration_seconds": true,
			"rest_client_requests_total":                true,
		}

		It("should contain virt components metrics", func() {
			err := virtoperator.SetupMetrics()
			Expect(err).ToNot(HaveOccurred())

			err = virtapi.SetupMetrics()
			Expect(err).ToNot(HaveOccurred())

			err = virtcontroller.SetupMetrics(nil, nil, nil, nil, nil, nil, nil, nil)
			Expect(err).ToNot(HaveOccurred())

			for _, metric := range operatormetrics.ListMetrics() {
				if excludedMetrics[metric.GetOpts().Name] {
					continue
				}

				Expect(metrics.Data.Result).To(ContainElement(gomegaContainsMetricMatcher(metric, nil)))
			}
		})

		It("should have kubevirt_vmi_phase_transition_time_seconds buckets correctly configured", func() {
			buckets := virtcontroller.PhaseTransitionTimeBuckets()

			for _, bucket := range buckets {
				labels := map[string]string{"le": strconv.FormatFloat(bucket, 'f', -1, 64)}

				metric := operatormetrics.NewHistogram(
					operatormetrics.MetricOpts{Name: "kubevirt_vmi_phase_transition_time_from_deletion_seconds"},
					prometheus.HistogramOpts{},
				)

				Expect(metrics.Data.Result).To(ContainElement(gomegaContainsMetricMatcher(metric, labels)))
			}
		})
	})
})

func fetchPrometheusMetrics(virtClient kubecli.KubevirtClient) *libmonitoring.QueryRequestResult {
	metrics, err := libmonitoring.QueryRange(virtClient, "{__name__=~\"kubevirt_.*\"}", time.Now().Add(-1*time.Minute), time.Now(), 15*time.Second)
	Expect(err).ToNot(HaveOccurred())

	Expect(metrics.Status).To(Equal("success"))
	Expect(metrics.Data.ResultType).To(Equal("matrix"))
	Expect(metrics.Data.Result).ToNot(BeEmpty(), "No metrics found")

	return metrics
}

func setupVM(virtClient kubecli.KubevirtClient) {
	vm := createRunningVM(virtClient)
	libmonitoring.WaitForMetricValue(virtClient, "kubevirt_number_of_vms", 1)

	By("Deleting the VirtualMachine")
	err := virtClient.VirtualMachine(vm.Namespace).Delete(context.Background(), vm.Name, &metav1.DeleteOptions{})
	Expect(err).ToNot(HaveOccurred())

	libmonitoring.WaitForMetricValue(virtClient, "kubevirt_number_of_vms", -1)
}

func createRunningVM(virtClient kubecli.KubevirtClient) *v1.VirtualMachine {
	vmi := libvmifact.NewGuestless(libvmi.WithNamespace(testsuite.GetTestNamespace(nil)))
	vm := libvmi.NewVirtualMachine(vmi, libvmi.WithRunning())
	vm, err := virtClient.VirtualMachine(testsuite.GetTestNamespace(vm)).Create(context.Background(), vm)
	Expect(err).ToNot(HaveOccurred())

	Eventually(func() bool {
		vm, err := virtClient.VirtualMachine(testsuite.GetTestNamespace(vm)).Get(context.Background(), vm.Name, &metav1.GetOptions{})
		Expect(err).ToNot(HaveOccurred())
		return vm.Status.Ready
	}, 300*time.Second, 1*time.Second).Should(BeTrue())
	libwait.WaitForSuccessfulVMIStart(vmi)

	return vm
}

func gomegaContainsMetricMatcher(metric operatormetrics.Metric, labels map[string]string) types.GomegaMatcher {
	return &libmonitoring.MetricMatcher{Metric: metric, Labels: labels}
}
