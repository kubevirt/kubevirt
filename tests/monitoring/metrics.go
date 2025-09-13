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
 * Copyright The KubeVirt Authors
 *
 */

package monitoring

import (
	"context"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/onsi/gomega/types"
	"github.com/rhobs/operator-observability-toolkit/pkg/operatormetrics"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	v1 "kubevirt.io/api/core/v1"
	"kubevirt.io/client-go/kubecli"

	"kubevirt.io/kubevirt/pkg/libvmi"
	"kubevirt.io/kubevirt/pkg/monitoring/metrics/testing"

	"kubevirt.io/kubevirt/tests/decorators"
	"kubevirt.io/kubevirt/tests/framework/kubevirt"
	. "kubevirt.io/kubevirt/tests/framework/matcher"
	"kubevirt.io/kubevirt/tests/libmonitoring"
	"kubevirt.io/kubevirt/tests/libstorage"
	"kubevirt.io/kubevirt/tests/libvmifact"
	"kubevirt.io/kubevirt/tests/libwait"
	"kubevirt.io/kubevirt/tests/testsuite"
)

var _ = Describe("[sig-monitoring]Metrics", decorators.SigMonitoring, func() {
	var virtClient kubecli.KubevirtClient
	var metrics *libmonitoring.QueryRequestResult
	var vm *v1.VirtualMachine

	BeforeEach(func() {
		virtClient = kubevirt.Client()
	})

	Context("Prometheus metrics", Ordered, func() {
		var excludedMetrics = map[string]bool{
			// virt-api
			// can later be added in pre-existing feature tests
			"kubevirt_portforward_active_tunnels":                true,
			"kubevirt_usbredir_active_connections":               true,
			"kubevirt_vnc_active_connections":                    true,
			"kubevirt_console_active_connections":                true,
			"kubevirt_vmi_last_api_connection_timestamp_seconds": true,

			// needs a snapshot - ignoring since already tested in - VM Monitoring, VM snapshot metrics
			"kubevirt_vmsnapshot_succeeded_timestamp_seconds": true,

			// needs a machines variable - ignoring since already tested in - tests/infrastructure/prometheus
			"kubevirt_node_deprecated_machine_types": true,

			// migration metrics
			// needs a migration - ignoring since already tested in - VM Monitoring, VM migration metrics
			"kubevirt_vmi_migration_phase_transition_time_from_creation_seconds": true,
			"kubevirt_vmi_migrations_in_pending_phase":                           true,
			"kubevirt_vmi_migrations_in_scheduling_phase":                        true,
			"kubevirt_vmi_migrations_in_unset_phase":                             true,
			"kubevirt_vmi_migrations_in_running_phase":                           true,
			"kubevirt_vmi_migration_succeeded":                                   true,
			"kubevirt_vmi_migration_failed":                                      true,
			"kubevirt_vmi_migration_data_remaining_bytes":                        true,
			"kubevirt_vmi_migration_data_processed_bytes":                        true,
			"kubevirt_vmi_migration_dirty_memory_rate_bytes":                     true,
			"kubevirt_vmi_migration_memory_transfer_rate_bytes":                  true,
			"kubevirt_vmi_migration_data_total_bytes":                            true,
			"kubevirt_vmi_migration_start_time_seconds":                          true,
			"kubevirt_vmi_migration_end_time_seconds":                            true,

			// This metric is using a dedicated collector and is being tested separately
			"kubevirt_vmi_dirty_rate_bytes_per_second": true,

			// CPU load metrics need an updated libvirt version running on the nodes
			// that exposes the CPU load information
			"kubevirt_vmi_guest_load_1m":  true,
			"kubevirt_vmi_guest_load_5m":  true,
			"kubevirt_vmi_guest_load_15m": true,
		}

		BeforeAll(func() {
			vm = basicVMLifecycle(virtClient)
			metrics = fetchPrometheusKubevirtMetrics(virtClient)
			Expect(metrics.Data.Result).ToNot(BeEmpty(), "No metrics found")
		})

		It("should contain virt components metrics", func() {
			err := libmonitoring.RegisterAllMetrics()
			Expect(err).ToNot(HaveOccurred(), "Failed to register all metrics")

			for _, metric := range operatormetrics.ListMetrics() {
				if excludedMetrics[metric.GetOpts().Name] {
					continue
				}

				Expect(metrics.Data.Result).To(ContainElement(gomegaContainsMetricMatcher(metric, nil)))
			}
		})

		It("should contain VNIC metrics", func() {
			labels := map[string]string{
				"namespace":    vm.Namespace,
				"name":         vm.Name,
				"binding_type": "core",
				"network":      "pod networking",
				"binding_name": "masquerade",
			}

			By("Verifying VM vnic info metric")
			metric := operatormetrics.NewCounter(operatormetrics.MetricOpts{
				Name: "kubevirt_vm_vnic_info",
			})
			Expect(metrics.Data.Result).To(ContainElement(gomegaContainsMetricMatcher(metric, labels)))

			By("Verifying VMI vnic info metric")
			metric = operatormetrics.NewCounter(operatormetrics.MetricOpts{
				Name: "kubevirt_vmi_vnic_info",
			})
			Expect(metrics.Data.Result).To(ContainElement(gomegaContainsMetricMatcher(metric, labels)))
		})

		It("should contain disk metrics", func() {
			By("Verifying kubevirt_vm_disk_allocated_size_bytes metric")
			metric := operatormetrics.NewGauge(operatormetrics.MetricOpts{
				Name: "kubevirt_vm_disk_allocated_size_bytes",
			})
			labels := map[string]string{
				"namespace":             vm.Namespace,
				"name":                  vm.Name,
				"persistentvolumeclaim": "test-vm-pvc",
				"volume_mode":           "Filesystem",
				"device":                "testdisk",
			}
			Expect(metrics.Data.Result).To(ContainElement(gomegaContainsMetricMatcher(metric, labels)))
		})

		It("should contain label metrics", func() {
			By("Verifying kubevirt_vm_labels metric")
			metric := operatormetrics.NewGauge(operatormetrics.MetricOpts{
				Name: "kubevirt_vm_labels",
			})
			labels := map[string]string{
				"namespace":                 vm.Namespace,
				"name":                      vm.Name,
				"label_vm_kubevirt_io_test": "test-vm-labels",
			}
			Expect(metrics.Data.Result).To(ContainElement(gomegaContainsMetricMatcher(metric, labels)))
		})
	})

	Context("Workqueue metrics", func() {
		It("should not contain controller-runtime workqueue metrics for virt workloads", func() {
			By("Checking workqueue_depth{container=~\"virt*\"} is not present")
			query := "{__name__=\"workqueue_depth\",container=~\"virt.*\"}"
			metrics := fetchPrometheusMetrics(virtClient, query)
			Expect(metrics.Data.Result).To(BeEmpty(), "Expected no workqueue_depth metrics for virt workloads")
		})

		It("kubevirt workqueue metrics should include controllers names", func() {
			names := []string{
				"virt-operator",
				"virt-handler-node-labeller",
				"virt-handler-source",
				"virt-handler-target",
				"virt-handler-vm",
				"virt-controller-disruption-budget",
				"virt-controller-evacuation",
				"virt-controller-export-vmexport",
				"virt-controller-migration",
				"virt-controller-node",
				"virt-controller-pool",
				"virt-controller-replicaset",
				"virt-controller-restore-vmrestore",
				"virt-controller-snapshot-crd",
				"virt-controller-snapshot-vm",
				"virt-controller-snapshot-vmsnapshot",
				"virt-controller-snapshot-vmsnapshotcontent",
				"virt-controller-snapshot-vmsnashotstatus",
				"virt-controller-vm",
				"virt-controller-vmclone",
				"virt-controller-vmi",
				"virt-controller-workload-update",
			}

			for _, name := range names {
				By("Checking workqueue metrics for " + name)
				query := "{__name__=\"kubevirt_workqueue_adds_total\",name=\"" + name + "\"}"
				metrics := fetchPrometheusMetrics(virtClient, query)
				Expect(metrics.Data.Result).ToNot(BeEmpty(), "Expected workqueue metrics for "+name)
			}
		})
	})
})

func fetchPrometheusKubevirtMetrics(virtClient kubecli.KubevirtClient) *libmonitoring.QueryRequestResult {
	return fetchPrometheusMetrics(virtClient, "{__name__=~\"kubevirt_.*\"}")
}

func fetchPrometheusMetrics(virtClient kubecli.KubevirtClient, query string) *libmonitoring.QueryRequestResult {
	metrics, err := libmonitoring.QueryRange(virtClient, query, time.Now().Add(-1*time.Minute), time.Now(), 15*time.Second)
	Expect(err).ToNot(HaveOccurred())

	Expect(metrics.Status).To(Equal("success"))
	Expect(metrics.Data.ResultType).To(Equal("matrix"))

	return metrics
}

func basicVMLifecycle(virtClient kubecli.KubevirtClient) *v1.VirtualMachine {
	By("Creating and running a VM")
	vm := createAndRunVM(virtClient)

	By("Waiting for the VM to be reported")
	libmonitoring.WaitForMetricValue(virtClient, "kubevirt_number_of_vms", 1)

	By("Waiting for the VMI to be reported")
	labels := map[string]string{
		"namespace": vm.Namespace,
		"name":      vm.Name,
	}
	libmonitoring.WaitForMetricValueWithLabels(virtClient, "kubevirt_vmi_info", 1, labels, 1)

	By("Waiting for the VM domainstats metrics to be reported")
	libmonitoring.WaitForMetricValueWithLabelsToBe(virtClient, "kubevirt_vmi_filesystem_capacity_bytes", map[string]string{"namespace": vm.Namespace, "name": vm.Name}, 0, ">", 0)

	By("Deleting the VirtualMachine")
	err := virtClient.VirtualMachine(vm.Namespace).Delete(context.Background(), vm.Name, metav1.DeleteOptions{})
	Expect(err).ToNot(HaveOccurred())

	By("Waiting for the VM deletion to be reported")
	libmonitoring.WaitForMetricValue(virtClient, "kubevirt_number_of_vms", -1)

	return vm
}

func createAndRunVM(virtClient kubecli.KubevirtClient) *v1.VirtualMachine {
	vmDiskPVC := "test-vm-pvc"
	pvc := libstorage.CreateFSPVC(vmDiskPVC, testsuite.GetTestNamespace(nil), "512Mi", nil)
	iface := *v1.DefaultMasqueradeNetworkInterface()

	vmi := libvmifact.NewFedora(
		libvmi.WithNamespace(testsuite.GetTestNamespace(nil)),
		libvmi.WithMemoryLimit("512Mi"),
		libvmi.WithPersistentVolumeClaim("testdisk", pvc.Name),
		libvmi.WithInterface(iface),
		libvmi.WithNetwork(v1.DefaultPodNetwork()),
		libvmi.WithLabel("vm.kubevirt.io/test", "test-vm-labels"),
	)

	vm := libvmi.NewVirtualMachine(vmi, libvmi.WithRunStrategy(v1.RunStrategyAlways))
	vm, err := virtClient.VirtualMachine(testsuite.GetTestNamespace(vm)).Create(context.Background(), vm, metav1.CreateOptions{})
	Expect(err).ToNot(HaveOccurred())

	Eventually(ThisVM(vm)).WithTimeout(300 * time.Second).WithPolling(time.Second).Should(BeReady())
	libwait.WaitForSuccessfulVMIStart(vmi)

	return vm
}

func gomegaContainsMetricMatcher(metric operatormetrics.Metric, labels map[string]string) types.GomegaMatcher {
	return &testing.MetricMatcher{Metric: metric, Labels: labels}
}
