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
 * Copyright 2023 Red Hat, Inc.
 *
 */

package monitoring

import (
	"context"
	"fmt"
	"time"

	"kubevirt.io/kubevirt/tests/libmigration"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/types"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	v1 "kubevirt.io/api/core/v1"
	"kubevirt.io/client-go/kubecli"

	virtctlpause "kubevirt.io/kubevirt/pkg/virtctl/pause"

	"kubevirt.io/kubevirt/tests"
	"kubevirt.io/kubevirt/tests/clientcmd"
	"kubevirt.io/kubevirt/tests/decorators"
	"kubevirt.io/kubevirt/tests/exec"
	"kubevirt.io/kubevirt/tests/flags"
	"kubevirt.io/kubevirt/tests/framework/checks"
	"kubevirt.io/kubevirt/tests/framework/kubevirt"
	"kubevirt.io/kubevirt/tests/framework/matcher"
	"kubevirt.io/kubevirt/tests/libnode"
	"kubevirt.io/kubevirt/tests/libvmi"
	"kubevirt.io/kubevirt/tests/libwait"
	"kubevirt.io/kubevirt/tests/testsuite"
	"kubevirt.io/kubevirt/tests/util"
)

var _ = Describe("[Serial][sig-monitoring]VM Monitoring", Serial, decorators.SigMonitoring, func() {
	var err error
	var virtClient kubecli.KubevirtClient

	BeforeEach(func() {
		virtClient = kubevirt.Client()
	})

	Context("Cluster VM metrics", func() {
		It("kubevirt_number_of_vms should reflect the number of VMs", func() {
			for i := 0; i < 5; i++ {
				vmi := tests.NewRandomVMI()
				vm := tests.NewRandomVirtualMachine(vmi, false)
				_, err = virtClient.VirtualMachine(vm.Namespace).Create(context.Background(), vm)
				Expect(err).ToNot(HaveOccurred())
			}

			waitForMetricValue(virtClient, "kubevirt_number_of_vms", 5)
		})
	})

	Context("VM status metrics", func() {
		var vm *v1.VirtualMachine
		var cpuMetrics = []string{
			"kubevirt_vmi_cpu_system_usage_seconds_total",
			"kubevirt_vmi_cpu_usage_seconds_total",
			"kubevirt_vmi_cpu_user_usage_seconds_total",
		}

		BeforeEach(func() {
			vmi := tests.NewRandomVMI()
			vm = tests.NewRandomVirtualMachine(vmi, false)

			By("Create a VirtualMachine")
			_, err = virtClient.VirtualMachine(vm.Namespace).Create(context.Background(), vm)
			Expect(err).ToNot(HaveOccurred())
		})

		checkMetricTo := func(metric string, labels map[string]string, matcher types.GomegaMatcher, description string) {
			EventuallyWithOffset(1, func() float64 {
				i, err := getMetricValueWithLabels(virtClient, metric, labels)
				if err != nil {
					return -1
				}
				return i
			}, 3*time.Minute, 20*time.Second).Should(matcher, description)
		}

		It("Should be available for a running VM", func() {
			By("Start the VM")
			vm = tests.StartVirtualMachine(vm)

			By("Checking that the VM metrics are available")
			metricLabels := map[string]string{"name": vm.Name, "namespace": vm.Namespace}
			for _, metric := range cpuMetrics {
				checkMetricTo(metric, metricLabels, BeNumerically(">=", 0), "VM metrics should be available for a running VM")
			}
		})

		It("Should be available for a paused VM", func() {
			By("Start the VM")
			vm = tests.StartVirtualMachine(vm)

			By("Pausing the VM")
			command := clientcmd.NewRepeatableVirtctlCommand(virtctlpause.COMMAND_PAUSE, "vm", "--namespace", testsuite.GetTestNamespace(vm), vm.Name)
			Expect(command()).To(Succeed())

			By("Waiting until next Prometheus scrape")
			time.Sleep(35 * time.Second)

			By("Checking that the VM metrics are available")
			metricLabels := map[string]string{"name": vm.Name, "namespace": vm.Namespace}
			for _, metric := range cpuMetrics {
				checkMetricTo(metric, metricLabels, BeNumerically(">=", 0), "VM metrics should be available for a paused VM")
			}
		})

		It("Should not be available for a stopped VM", func() {
			By("Checking that the VM metrics are not available")
			metricLabels := map[string]string{"name": vm.Name, "namespace": vm.Namespace}
			for _, metric := range cpuMetrics {
				checkMetricTo(metric, metricLabels, BeNumerically("==", -1), "VM metrics should not be available for a stopped VM")
			}
		})
	})

	Context("VM migration metrics", func() {
		var nodes *corev1.NodeList

		BeforeEach(func() {
			checks.SkipIfMigrationIsNotPossible()

			Eventually(func() []corev1.Node {
				nodes = libnode.GetAllSchedulableNodes(virtClient)
				return nodes.Items
			}, 60*time.Second, 1*time.Second).ShouldNot(BeEmpty(), "There should be some compute node")
		})

		It("Should correctly update metrics on successful VMIM", func() {
			By("Creating VMIs")
			vmi := tests.NewRandomFedoraVMI()
			vmi = tests.RunVMIAndExpectLaunch(vmi, 240)

			By("Migrating VMIs")
			migration := tests.NewRandomMigration(vmi.Name, vmi.Namespace)
			libmigration.RunMigrationAndExpectToCompleteWithDefaultTimeout(virtClient, migration)

			waitForMetricValue(virtClient, "kubevirt_vmi_migrations_in_pending_phase", 0)
			waitForMetricValue(virtClient, "kubevirt_vmi_migrations_in_scheduling_phase", 0)
			waitForMetricValue(virtClient, "kubevirt_vmi_migrations_in_running_phase", 0)

			labels := map[string]string{
				"vmi": vmi.Name,
			}
			waitForMetricValueWithLabels(virtClient, "kubevirt_vmi_migration_succeeded", 1, labels)

			By("Delete VMIs")
			Expect(virtClient.VirtualMachineInstance(vmi.Namespace).Delete(context.Background(), vmi.Name, &metav1.DeleteOptions{})).To(Succeed())
			libwait.WaitForVirtualMachineToDisappearWithTimeout(vmi, 240)
		})

		It("Should correctly update metrics on failing VMIM", func() {
			By("Creating VMIs")
			vmi := libvmi.NewFedora(
				libvmi.WithInterface(libvmi.InterfaceDeviceWithMasqueradeBinding()),
				libvmi.WithNetwork(v1.DefaultPodNetwork()),
				libvmi.WithNodeAffinityFor(&nodes.Items[0]),
			)
			vmi = tests.RunVMIAndExpectLaunch(vmi, 240)
			labels := map[string]string{
				"vmi": vmi.Name,
			}

			By("Starting the Migration")
			migration := tests.NewRandomMigration(vmi.Name, vmi.Namespace)
			migration.Annotations = map[string]string{v1.MigrationUnschedulablePodTimeoutSecondsAnnotation: "60"}
			migration = libmigration.RunMigration(virtClient, migration)

			waitForMetricValue(virtClient, "kubevirt_vmi_migrations_in_scheduling_phase", 1)

			Eventually(matcher.ThisMigration(migration), 2*time.Minute, 5*time.Second).Should(matcher.BeInPhase(v1.MigrationFailed), "migration creation should fail")

			waitForMetricValue(virtClient, "kubevirt_vmi_migrations_in_scheduling_phase", 0)
			waitForMetricValueWithLabels(virtClient, "kubevirt_vmi_migration_failed", 1, labels)

			By("Deleting the VMI")
			Expect(virtClient.VirtualMachineInstance(vmi.Namespace).Delete(context.Background(), vmi.Name, &metav1.DeleteOptions{})).To(Succeed())
			libwait.WaitForVirtualMachineToDisappearWithTimeout(vmi, 240)
		})
	})

	Context("VM snapshot metrics", func() {
		quantity, _ := resource.ParseQuantity("500Mi")

		createSimplePVCWithRestoreLabels := func(name string) {
			_, err := virtClient.CoreV1().PersistentVolumeClaims(util.NamespaceTestDefault).Create(context.Background(), &corev1.PersistentVolumeClaim{
				ObjectMeta: metav1.ObjectMeta{
					Name: name,
					Labels: map[string]string{
						"restore.kubevirt.io/source-vm-name":      "simple-vm",
						"restore.kubevirt.io/source-vm-namespace": util.NamespaceTestDefault,
					},
				},
				Spec: corev1.PersistentVolumeClaimSpec{
					AccessModes: []corev1.PersistentVolumeAccessMode{corev1.ReadWriteOnce},
					Resources: corev1.ResourceRequirements{
						Requests: corev1.ResourceList{
							"storage": quantity,
						},
					},
				},
			}, metav1.CreateOptions{})
			Expect(err).ToNot(HaveOccurred())
		}

		It("[test_id:8639]Number of disks restored and total restored bytes metric values should be correct", func() {
			totalMetric := fmt.Sprintf("kubevirt_vmsnapshot_disks_restored_from_source{vm_name='simple-vm',vm_namespace='%s'}", util.NamespaceTestDefault)
			bytesMetric := fmt.Sprintf("kubevirt_vmsnapshot_disks_restored_from_source_bytes{vm_name='simple-vm',vm_namespace='%s'}", util.NamespaceTestDefault)
			numPVCs := 2.0

			for i := 1.0; i < numPVCs+1; i++ {
				// Create dummy PVC that is labelled as "restored" from VM snapshot
				createSimplePVCWithRestoreLabels(fmt.Sprintf("vmsnapshot-restored-pvc-%f", i))
				// Metric values increases per restored disk
				waitForMetricValue(virtClient, totalMetric, i)
				waitForMetricValue(virtClient, bytesMetric, float64(quantity.Value())*i)
			}
		})
	})

	Context("VM alerts", func() {
		var scales *Scaling

		BeforeEach(func() {
			scales = NewScaling(virtClient, []string{virtOperator.deploymentName})
			scales.UpdateScale(virtOperator.deploymentName, int32(0))

			reduceAlertPendingTime(virtClient)
		})

		AfterEach(func() {
			scales.RestoreAllScales()
		})

		It("should fire KubevirtVmHighMemoryUsage alert", func() {
			By("starting VMI")
			vmi := tests.NewRandomVMI()
			tests.RunVMIAndExpectLaunch(vmi, 240)

			By("fill up the vmi pod memory")
			vmiPod := tests.GetRunningPodByVirtualMachineInstance(vmi, util.NamespaceTestDefault)
			vmiPodRequestMemory := vmiPod.Spec.Containers[0].Resources.Requests.Memory().Value()
			_, err := exec.ExecuteCommandOnPod(
				virtClient,
				vmiPod,
				vmiPod.Spec.Containers[0].Name,
				[]string{"/usr/bin/bash", "-c", fmt.Sprintf("cat <( </dev/zero head -c %d) <(sleep 150) | tail", vmiPodRequestMemory)},
			)
			Expect(err).ToNot(HaveOccurred())

			By("waiting for KubevirtVmHighMemoryUsage alert")
			verifyAlertExist(virtClient, "KubevirtVmHighMemoryUsage")
		})

		It("should fire OrphanedVirtualMachineInstances alert", func() {
			By("starting VMI")
			vmi := tests.NewRandomVMI()
			tests.RunVMIAndExpectLaunch(vmi, 240)

			By("delete virt-handler daemonset")
			err = virtClient.AppsV1().DaemonSets(flags.KubeVirtInstallNamespace).Delete(context.Background(), virtHandler.deploymentName, metav1.DeleteOptions{})
			Expect(err).ToNot(HaveOccurred())

			By("waiting for OrphanedVirtualMachineInstances alert")
			verifyAlertExist(virtClient, "OrphanedVirtualMachineInstances")
		})

		It("should fire VMCannotBeEvicted alert", func() {
			By("starting non-migratable VMI with eviction strategy set to LiveMigrate ")
			vmi := tests.NewRandomVMI()
			strategy := v1.EvictionStrategyLiveMigrate
			vmi.Spec.EvictionStrategy = &strategy
			vmi = tests.RunVMI(vmi, 240)

			By("waiting for VMCannotBeEvicted alert")
			verifyAlertExist(virtClient, "VMCannotBeEvicted")
		})
	})
})
