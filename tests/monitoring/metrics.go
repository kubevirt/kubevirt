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

package monitoring

import (
	"context"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/machadovilaca/operator-observability/pkg/operatormetrics"
	"github.com/onsi/gomega/types"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	v1 "kubevirt.io/api/core/v1"
	"kubevirt.io/client-go/kubecli"

	"kubevirt.io/kubevirt/pkg/libvmi"
	"kubevirt.io/kubevirt/pkg/monitoring/metrics/testing"
	virtapi "kubevirt.io/kubevirt/pkg/monitoring/metrics/virt-api"
	virtcontroller "kubevirt.io/kubevirt/pkg/monitoring/metrics/virt-controller"
	virthandler "kubevirt.io/kubevirt/pkg/monitoring/metrics/virt-handler"
	virtoperator "kubevirt.io/kubevirt/pkg/monitoring/metrics/virt-operator"

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

	BeforeEach(func() {
		virtClient = kubevirt.Client()
		basicVMLifecycle(virtClient)
		metrics = fetchPrometheusMetrics(virtClient)
	})

	Context("Prometheus metrics", func() {
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
			"kubevirt_vmi_migration_disk_transfer_rate_bytes":                    true,
			"kubevirt_vmi_migration_data_total_bytes":                            true,
			"kubevirt_vmi_migration_start_time_seconds":                          true,
			"kubevirt_vmi_migration_end_time_seconds":                            true,
		}

		It("should contain virt components metrics", func() {
			err := virtoperator.SetupMetrics()
			Expect(err).ToNot(HaveOccurred())

			err = virtoperator.RegisterLeaderMetrics()
			Expect(err).ToNot(HaveOccurred())

			err = virtapi.SetupMetrics()
			Expect(err).ToNot(HaveOccurred())

			err = virtcontroller.SetupMetrics(nil, nil, nil, nil)
			Expect(err).ToNot(HaveOccurred())

			err = virtcontroller.RegisterLeaderMetrics()
			Expect(err).ToNot(HaveOccurred())

			err = virthandler.SetupMetrics("", "", 0, nil)
			Expect(err).ToNot(HaveOccurred())

			for _, metric := range operatormetrics.ListMetrics() {
				if excludedMetrics[metric.GetOpts().Name] {
					continue
				}

				Expect(metrics.Data.Result).To(ContainElement(gomegaContainsMetricMatcher(metric, nil)))
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

func basicVMLifecycle(virtClient kubecli.KubevirtClient) {
	By("Creating and running a VM")
	vm := createAndRunVM(virtClient)

	By("Waiting for the VM to be reported")
	libmonitoring.WaitForMetricValue(virtClient, "kubevirt_number_of_vms", 1)

	By("Waiting for the VM domainstats metrics to be reported")
	libmonitoring.WaitForMetricValueWithLabelsToBe(virtClient, "kubevirt_vmi_filesystem_capacity_bytes", map[string]string{"namespace": vm.Namespace, "name": vm.Name}, 0, ">", 0)

	By("Verifying kubevirt_vm_disk_allocated_size_bytes metric")
	libmonitoring.WaitForMetricValueWithLabelsToBe(virtClient, "kubevirt_vm_disk_allocated_size_bytes",
		map[string]string{
			"namespace":             vm.Namespace,
			"name":                  vm.Name,
			"persistentvolumeclaim": "test-vm-pvc",
			"volume_mode":           "Filesystem",
			"device":                "testdisk",
		},
		0, ">", 0)

	By("Verifying kubevirt_vm_vnic_info metric")
	libmonitoring.WaitForMetricValueWithLabels(virtClient, "kubevirt_vm_vnic_info", 1,
		map[string]string{
			"namespace":    vm.Namespace,
			"name":         vm.Name,
			"binding_type": "core",
			"network":      "pod networking",
			"binding_name": "masquerade",
		}, 0)

	By("Verifying kubevirt_vmi_vnic_info metric")
	libmonitoring.WaitForMetricValueWithLabels(virtClient, "kubevirt_vmi_vnic_info", 1,
		map[string]string{
			"namespace":    vm.Namespace,
			"name":         vm.Name,
			"binding_type": "core",
			"network":      "pod networking",
			"binding_name": "masquerade",
		}, 0)

	By("Deleting the VirtualMachine")
	err := virtClient.VirtualMachine(vm.Namespace).Delete(context.Background(), vm.Name, metav1.DeleteOptions{})
	Expect(err).ToNot(HaveOccurred())

	By("Waiting for the VM deletion to be reported")
	libmonitoring.WaitForMetricValue(virtClient, "kubevirt_number_of_vms", -1)
}

func createAndRunVM(virtClient kubecli.KubevirtClient) *v1.VirtualMachine {
	vmDiskPVC := "test-vm-pvc"
	pvc := libstorage.CreateFSPVC(vmDiskPVC, testsuite.GetTestNamespace(nil), "512Mi", nil)
	iface := *v1.DefaultMasqueradeNetworkInterface()

	vmi := libvmifact.NewFedora(
		libvmi.WithNamespace(testsuite.GetTestNamespace(nil)),
		libvmi.WithLimitMemory("512Mi"),
		libvmi.WithPersistentVolumeClaim("testdisk", pvc.Name),
		libvmi.WithInterface(iface),
		libvmi.WithNetwork(v1.DefaultPodNetwork()),
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
