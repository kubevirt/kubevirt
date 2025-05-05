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
	"fmt"
	"strconv"
	"time"

	"kubevirt.io/kubevirt/tests/console"
	"kubevirt.io/kubevirt/tests/libmigration"
	"kubevirt.io/kubevirt/tests/libstorage"
	"kubevirt.io/kubevirt/tests/libvmops"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	. "github.com/onsi/gomega/gstruct"
	"github.com/onsi/gomega/types"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	v1 "kubevirt.io/api/core/v1"
	"kubevirt.io/client-go/kubecli"

	"kubevirt.io/kubevirt/pkg/libvmi"
	virtcontroller "kubevirt.io/kubevirt/pkg/monitoring/metrics/virt-controller"
	"kubevirt.io/kubevirt/tests/decorators"
	"kubevirt.io/kubevirt/tests/flags"
	"kubevirt.io/kubevirt/tests/framework/kubevirt"
	"kubevirt.io/kubevirt/tests/framework/matcher"
	"kubevirt.io/kubevirt/tests/libmonitoring"
	"kubevirt.io/kubevirt/tests/libnet"
	"kubevirt.io/kubevirt/tests/libnode"
	"kubevirt.io/kubevirt/tests/libvmifact"
	"kubevirt.io/kubevirt/tests/libwait"
	"kubevirt.io/kubevirt/tests/testsuite"
)

var _ = Describe("[sig-monitoring]VM Monitoring", Serial, decorators.SigMonitoring, func() {
	var (
		err        error
		virtClient kubecli.KubevirtClient
		vm         *v1.VirtualMachine
	)

	BeforeEach(func() {
		virtClient = kubevirt.Client()
	})

	Context("Cluster VM metrics", func() {
		It("kubevirt_number_of_vms should reflect the number of VMs", func() {
			for i := 0; i < 5; i++ {
				vmi := libvmifact.NewGuestless()
				vm := libvmi.NewVirtualMachine(vmi)
				_, err = virtClient.VirtualMachine(testsuite.GetTestNamespace(nil)).Create(context.Background(), vm, metav1.CreateOptions{})
				Expect(err).ToNot(HaveOccurred())
			}

			libmonitoring.WaitForMetricValue(virtClient, "kubevirt_number_of_vms", 5)
		})
	})

	Context("VMI metrics", func() {
		It("should have kubevirt_vmi_phase_transition_time_seconds buckets correctly configured", func() {
			vmi := libvmifact.NewGuestless()
			libvmops.RunVMIAndExpectLaunch(vmi, 240)

			for _, bucket := range virtcontroller.PhaseTransitionTimeBuckets() {
				labels := map[string]string{"le": strconv.FormatFloat(bucket, 'f', -1, 64)}

				GinkgoLogr.Info("Checking bucket", "labels", labels)
				libmonitoring.WaitForMetricValueWithLabelsToBe(virtClient, "kubevirt_vmi_phase_transition_time_seconds_bucket", labels, 0, ">=", 0)
			}
		})

		It("should have kubevirt_rest_client_requests_total for the 'virtualmachineinstances' resource", func() {
			vmi := libvmifact.NewGuestless()
			libvmops.RunVMIAndExpectLaunch(vmi, 240)

			labels := map[string]string{"resource": "virtualmachineinstances"}
			libmonitoring.WaitForMetricValueWithLabelsToBe(virtClient, "kubevirt_rest_client_requests_total", labels, 0, ">", 0)
		})
	})

	Context("VM status metrics", func() {
		var cpuMetrics = []string{
			"kubevirt_vmi_cpu_system_usage_seconds_total",
			"kubevirt_vmi_cpu_usage_seconds_total",
			"kubevirt_vmi_cpu_user_usage_seconds_total",
		}

		checkMetricTo := func(metric string, labels map[string]string, matcher types.GomegaMatcher, description string) {
			EventuallyWithOffset(1, func() float64 {
				i, err := libmonitoring.GetMetricValueWithLabels(virtClient, metric, labels)
				if err != nil {
					return -1
				}
				return i
			}, 3*time.Minute, 20*time.Second).Should(matcher, description)
		}

		It("Should be available for a running VM", func() {

			By("Create a running VirtualMachine")
			vm = libvmi.NewVirtualMachine(libvmifact.NewGuestless(), libvmi.WithRunStrategy(v1.RunStrategyAlways))
			vm, err = virtClient.VirtualMachine(testsuite.GetTestNamespace(vm)).Create(context.Background(), vm, metav1.CreateOptions{})
			Expect(err).ToNot(HaveOccurred())
			Eventually(matcher.ThisVM(vm)).WithTimeout(300 * time.Second).WithPolling(time.Second).Should(matcher.BeReady())

			By("Checking that the VM metrics are available")
			metricLabels := map[string]string{"name": vm.Name, "namespace": vm.Namespace}
			for _, metric := range cpuMetrics {
				checkMetricTo(metric, metricLabels, BeNumerically(">=", 0), "VM metrics should be available for a running VM")
			}
		})

		It("Should be available for a paused VM", func() {

			By("Create a running VirtualMachine")
			vm = libvmi.NewVirtualMachine(libvmifact.NewGuestless(), libvmi.WithRunStrategy(v1.RunStrategyAlways))
			vm, err = virtClient.VirtualMachine(testsuite.GetTestNamespace(vm)).Create(context.Background(), vm, metav1.CreateOptions{})
			Expect(err).ToNot(HaveOccurred())
			Eventually(matcher.ThisVM(vm)).WithTimeout(300 * time.Second).WithPolling(time.Second).Should(matcher.BeReady())

			By("Pausing the VM")
			err := virtClient.VirtualMachineInstance(vm.Namespace).Pause(context.Background(), vm.Name, &v1.PauseOptions{})
			Expect(err).ToNot(HaveOccurred())

			By("Waiting until next Prometheus scrape")
			time.Sleep(35 * time.Second)

			By("Checking that the VM metrics are available")
			metricLabels := map[string]string{"name": vm.Name, "namespace": vm.Namespace}
			for _, metric := range cpuMetrics {
				checkMetricTo(metric, metricLabels, BeNumerically(">=", 0), "VM metrics should be available for a paused VM")
			}
		})

		It("Should not be available for a stopped VM", func() {
			By("Create a stopped VirtualMachine")
			vm = libvmi.NewVirtualMachine(libvmifact.NewGuestless())
			vm, err = virtClient.VirtualMachine(testsuite.GetTestNamespace(vm)).Create(context.Background(), vm, metav1.CreateOptions{})
			Expect(err).ToNot(HaveOccurred())

			By("Checking that the VM metrics are not available")
			metricLabels := map[string]string{"name": vm.Name, "namespace": vm.Namespace}
			for _, metric := range cpuMetrics {
				checkMetricTo(metric, metricLabels, BeNumerically("==", -1), "VM metrics should not be available for a stopped VM")
			}
		})
	})

	Context("VM migration metrics", decorators.RequiresTwoSchedulableNodes, func() {
		var nodes *corev1.NodeList

		BeforeEach(func() {
			Eventually(func() []corev1.Node {
				nodes = libnode.GetAllSchedulableNodes(virtClient)
				return nodes.Items
			}, 60*time.Second, 1*time.Second).ShouldNot(BeEmpty(), "There should be some compute node")
		})

		It("Should correctly update metrics on successful VMIM", func() {
			By("Creating VMIs")
			vmi := libvmifact.NewFedora(libnet.WithMasqueradeNetworking())
			vmi = libvmops.RunVMIAndExpectLaunch(vmi, 240)

			By("Migrating VMIs")
			migration := libmigration.New(vmi.Name, vmi.Namespace)
			libmigration.RunMigrationAndExpectToCompleteWithDefaultTimeout(virtClient, migration)

			libmonitoring.WaitForMetricValue(virtClient, "kubevirt_vmi_migrations_in_pending_phase", 0)
			libmonitoring.WaitForMetricValue(virtClient, "kubevirt_vmi_migrations_in_scheduling_phase", 0)
			libmonitoring.WaitForMetricValue(virtClient, "kubevirt_vmi_migrations_in_unset_phase", 0)
			libmonitoring.WaitForMetricValue(virtClient, "kubevirt_vmi_migrations_in_running_phase", 0)

			labels := map[string]string{
				"vmi":       vmi.Name,
				"namespace": vmi.Namespace,
			}
			libmonitoring.WaitForMetricValueWithLabels(virtClient, "kubevirt_vmi_migration_succeeded", 1, labels, 1)

			By("Delete VMIs")
			Expect(virtClient.VirtualMachineInstance(vmi.Namespace).Delete(context.Background(), vmi.Name, metav1.DeleteOptions{})).To(Succeed())
			libwait.WaitForVirtualMachineToDisappearWithTimeout(vmi, 240)
		})

		It("Should correctly update metrics on failing VMIM", func() {
			By("Creating VMIs")
			vmi := libvmifact.NewFedora(
				libvmi.WithInterface(libvmi.InterfaceDeviceWithMasqueradeBinding()),
				libvmi.WithNetwork(v1.DefaultPodNetwork()),
				libvmi.WithNodeAffinityFor(nodes.Items[0].Name),
			)
			vmi = libvmops.RunVMIAndExpectLaunch(vmi, 240)
			labels := map[string]string{
				"vmi":       vmi.Name,
				"namespace": vmi.Namespace,
			}

			By("Starting the Migration")
			migration := libmigration.New(vmi.Name, vmi.Namespace)
			migration.Annotations = map[string]string{v1.MigrationUnschedulablePodTimeoutSecondsAnnotation: "60"}
			migration = libmigration.RunMigration(virtClient, migration)

			Eventually(matcher.ThisMigration(migration)).WithTimeout(2*time.Minute).WithPolling(5*time.Second).Should(matcher.BeInPhase(v1.MigrationFailed), "migration creation should fail")

			libmonitoring.WaitForMetricValue(virtClient, "kubevirt_vmi_migrations_in_scheduling_phase", 0)
			libmonitoring.WaitForMetricValue(virtClient, "kubevirt_vmi_migrations_in_unset_phase", 0)
			libmonitoring.WaitForMetricValueWithLabels(virtClient, "kubevirt_vmi_migration_failed", 1, labels, 1)

			By("Deleting the VMI")
			Expect(virtClient.VirtualMachineInstance(vmi.Namespace).Delete(context.Background(), vmi.Name, metav1.DeleteOptions{})).To(Succeed())
			libwait.WaitForVirtualMachineToDisappearWithTimeout(vmi, 240)
		})
	})

	Context("VM snapshot metrics", func() {
		quantity, _ := resource.ParseQuantity("500Mi")

		createSimplePVCWithRestoreLabels := func(name string) {
			_, err := virtClient.CoreV1().PersistentVolumeClaims(testsuite.NamespaceTestDefault).Create(context.Background(), &corev1.PersistentVolumeClaim{
				ObjectMeta: metav1.ObjectMeta{
					Name: name,
					Labels: map[string]string{
						"restore.kubevirt.io/source-vm-name":      "simple-vm",
						"restore.kubevirt.io/source-vm-namespace": testsuite.NamespaceTestDefault,
					},
				},
				Spec: corev1.PersistentVolumeClaimSpec{
					AccessModes: []corev1.PersistentVolumeAccessMode{corev1.ReadWriteOnce},
					Resources: corev1.VolumeResourceRequirements{
						Requests: corev1.ResourceList{
							"storage": quantity,
						},
					},
				},
			}, metav1.CreateOptions{})
			Expect(err).ToNot(HaveOccurred())
		}

		It("[test_id:8639]Number of disks restored and total restored bytes metric values should be correct", func() {
			totalMetric := fmt.Sprintf("kubevirt_vmsnapshot_disks_restored_from_source{vm_name='simple-vm',vm_namespace='%s'}", testsuite.NamespaceTestDefault)
			bytesMetric := fmt.Sprintf("kubevirt_vmsnapshot_disks_restored_from_source_bytes{vm_name='simple-vm',vm_namespace='%s'}", testsuite.NamespaceTestDefault)
			numPVCs := 2.0

			for i := 1.0; i < numPVCs+1; i++ {
				// Create dummy PVC that is labelled as "restored" from VM snapshot
				createSimplePVCWithRestoreLabels(fmt.Sprintf("vmsnapshot-restored-pvc-%f", i))
				// Metric values increases per restored disk
				libmonitoring.WaitForMetricValue(virtClient, totalMetric, i)
				libmonitoring.WaitForMetricValue(virtClient, bytesMetric, float64(quantity.Value())*i)
			}
		})

		It("Snapshot succeeded timestamp metric values should be correct", func() {
			virtClient = kubevirt.Client()
			By("Creating a Virtual Machine")
			vmi := libvmifact.NewGuestless()
			vm = libvmi.NewVirtualMachine(vmi)
			vm, err := virtClient.VirtualMachine(testsuite.GetTestNamespace(nil)).Create(context.Background(), vm, metav1.CreateOptions{})
			Expect(err).ToNot(HaveOccurred())

			By("Starting the Virtual Machine")
			libvmops.StartVirtualMachine(vm)

			By("Creating a snapshot of the Virtual Machine")
			snapshot := libstorage.NewSnapshot(vm.Name, vm.Namespace)
			_, err = virtClient.VirtualMachineSnapshot(vm.Namespace).Create(context.Background(), snapshot, metav1.CreateOptions{})
			Expect(err).ToNot(HaveOccurred())
			libstorage.WaitSnapshotSucceeded(virtClient, vm.Namespace, snapshot.Name)

			labels := map[string]string{
				"name":          snapshot.Spec.Source.Name,
				"snapshot_name": snapshot.Name,
				"namespace":     snapshot.Namespace,
			}
			libmonitoring.WaitForMetricValueWithLabelsToBe(virtClient, "kubevirt_vmsnapshot_succeeded_timestamp_seconds", labels, 0, ">", 0)
		})
	})

	Context("VM metrics that are based on the guest agent", func() {
		It("[test_id:11267]should have kubevirt_vmi_info correctly configured with guest OS labels", func() {
			agentVMI := createAgentVMI()
			Expect(agentVMI.Status.GuestOSInfo.KernelRelease).ToNot(BeEmpty())
			Expect(agentVMI.Status.GuestOSInfo.Machine).ToNot(BeEmpty())
			Expect(agentVMI.Status.GuestOSInfo.Name).ToNot(BeEmpty())
			Expect(agentVMI.Status.GuestOSInfo.VersionID).ToNot(BeEmpty())
			Expect(agentVMI.Status.Machine.Type).ToNot(BeEmpty())

			labels := map[string]string{
				"guest_os_kernel_release": agentVMI.Status.GuestOSInfo.KernelRelease,
				"guest_os_arch":           agentVMI.Status.GuestOSInfo.Machine,
				"guest_os_machine":        agentVMI.Status.Machine.Type,
				"guest_os_name":           agentVMI.Status.GuestOSInfo.Name,
				"guest_os_version_id":     agentVMI.Status.GuestOSInfo.VersionID,
			}

			libmonitoring.WaitForMetricValueWithLabels(virtClient, "kubevirt_vmi_info", 1, labels, 1)
		})
	})

	Context("Metrics that are based on VMI connections", func() {
		It("should have kubevirt_vmi_last_api_connection_timestamp_seconds correctly configured", func() {
			By("Starting a VirtualMachineInstance")
			vmi := libvmifact.NewAlpine()
			vmi, err = virtClient.VirtualMachineInstance(testsuite.GetTestNamespace(vmi)).Create(context.Background(), vmi, metav1.CreateOptions{})
			Expect(err).ToNot(HaveOccurred())
			vmi = libwait.WaitForSuccessfulVMIStart(vmi)

			By("Validating the metric gets updated with the first connection timestamp")
			Expect(console.LoginToAlpine(vmi)).To(Succeed())
			initialMetricValue := validateLastConnectionMetricValue(vmi, 0)

			By("Deleting the VirtualMachineInstance")
			Expect(virtClient.VirtualMachineInstance(vmi.Namespace).Delete(context.Background(), vmi.Name, metav1.DeleteOptions{})).To(Succeed())
			libwait.WaitForVirtualMachineToDisappearWithTimeout(vmi, 240)

			By("Starting the same VirtualMachineInstance")
			vmi = libvmifact.NewAlpine()
			vmi, err = virtClient.VirtualMachineInstance(testsuite.GetTestNamespace(vmi)).Create(context.Background(), vmi, metav1.CreateOptions{})
			Expect(err).ToNot(HaveOccurred())
			vmi = libwait.WaitForSuccessfulVMIStart(vmi)

			By("Validating the metric gets updated with the last connection timestamp")
			Expect(console.LoginToAlpine(vmi)).To(Succeed())
			validateLastConnectionMetricValue(vmi, initialMetricValue)
		})
	})

	Context("VM alerts", func() {
		var scales *libmonitoring.Scaling

		BeforeEach(func() {
			scales = libmonitoring.NewScaling(virtClient, []string{virtOperator.deploymentName})
			scales.UpdateScale(virtOperator.deploymentName, int32(0))

			libmonitoring.ReduceAlertPendingTime(virtClient)
		})

		AfterEach(func() {
			scales.RestoreAllScales()
		})

		It("[test_id:9260] should fire OrphanedVirtualMachineInstances alert", func() {
			By("starting VMI")
			vmi := libvmifact.NewGuestless()
			libvmops.RunVMIAndExpectLaunch(vmi, 240)

			By("delete virt-handler daemonset")
			err = virtClient.AppsV1().DaemonSets(flags.KubeVirtInstallNamespace).Delete(context.Background(), virtHandler.deploymentName, metav1.DeleteOptions{})
			Expect(err).ToNot(HaveOccurred())

			By("waiting for OrphanedVirtualMachineInstances alert")
			libmonitoring.VerifyAlertExist(virtClient, "OrphanedVirtualMachineInstances")
		})

		It("should fire VMCannotBeEvicted alert", func() {
			By("starting non-migratable VMI with eviction strategy set to LiveMigrate ")
			vmi := libvmifact.NewAlpine(libvmi.WithEvictionStrategy(v1.EvictionStrategyLiveMigrate))

			vmi, err := kubevirt.Client().VirtualMachineInstance(testsuite.GetTestNamespace(vmi)).Create(context.Background(), vmi, metav1.CreateOptions{})
			Expect(err).ToNot(HaveOccurred())

			Eventually(func() v1.VirtualMachineInstancePhase {
				vmi, err = virtClient.VirtualMachineInstance(vmi.Namespace).Get(context.Background(), vmi.Name, metav1.GetOptions{})
				Expect(err).ToNot(HaveOccurred())

				return vmi.Status.Phase
			}, 5*time.Minute, 30*time.Second).Should(Equal(v1.Running))

			By("waiting for VMCannotBeEvicted alert")
			libmonitoring.VerifyAlertExist(virtClient, "VMCannotBeEvicted")
		})
	})
})

func createAgentVMI() *v1.VirtualMachineInstance {
	virtClient := kubevirt.Client()
	vmiAgentConnectedConditionMatcher := MatchFields(IgnoreExtras, Fields{"Type": Equal(v1.VirtualMachineInstanceAgentConnected)})
	vmi := libvmops.RunVMIAndExpectLaunch(libvmifact.NewFedora(libnet.WithMasqueradeNetworking()), 180)

	var err error
	var agentVMI *v1.VirtualMachineInstance

	By("VMI has the guest agent connected condition")
	Eventually(func() []v1.VirtualMachineInstanceCondition {
		agentVMI, err = virtClient.VirtualMachineInstance(testsuite.GetTestNamespace(vmi)).Get(context.Background(), vmi.Name, metav1.GetOptions{})
		Expect(err).ToNot(HaveOccurred())
		return agentVMI.Status.Conditions
	}, 240*time.Second, 1*time.Second).Should(ContainElement(vmiAgentConnectedConditionMatcher), "Should have agent connected condition")

	return agentVMI
}

func validateLastConnectionMetricValue(vmi *v1.VirtualMachineInstance, formerValue float64) float64 {
	var err error
	var metricValue float64
	virtClient := kubevirt.Client()
	labels := map[string]string{"vmi": vmi.Name, "namespace": vmi.Namespace}

	EventuallyWithOffset(1, func() float64 {
		metricValue, err = libmonitoring.GetMetricValueWithLabels(virtClient, "kubevirt_vmi_last_api_connection_timestamp_seconds", labels)
		if err != nil {
			return -1
		}
		return metricValue
	}, 3*time.Minute, 20*time.Second).Should(BeNumerically(">", formerValue))

	return metricValue
}
