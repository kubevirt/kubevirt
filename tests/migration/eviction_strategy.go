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

package migration

import (
	"context"
	"fmt"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/gstruct"
	"github.com/onsi/gomega/types"

	k8sv1 "k8s.io/api/core/v1"
	policyv1 "k8s.io/api/policy/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"

	v1 "kubevirt.io/api/core/v1"
	"kubevirt.io/client-go/kubecli"

	"kubevirt.io/kubevirt/pkg/libvmi"
	"kubevirt.io/kubevirt/pkg/pointer"
	"kubevirt.io/kubevirt/tests/console"
	"kubevirt.io/kubevirt/tests/decorators"
	"kubevirt.io/kubevirt/tests/framework/cleanup"
	"kubevirt.io/kubevirt/tests/framework/kubevirt"
	"kubevirt.io/kubevirt/tests/framework/matcher"
	"kubevirt.io/kubevirt/tests/libkubevirt"
	"kubevirt.io/kubevirt/tests/libkubevirt/config"
	"kubevirt.io/kubevirt/tests/libmigration"
	"kubevirt.io/kubevirt/tests/libnet"
	"kubevirt.io/kubevirt/tests/libnode"
	"kubevirt.io/kubevirt/tests/libpod"
	"kubevirt.io/kubevirt/tests/libvmifact"
	"kubevirt.io/kubevirt/tests/libvmops"
	"kubevirt.io/kubevirt/tests/libwait"
	"kubevirt.io/kubevirt/tests/testsuite"
)

func vmiIsMigratedFrom(originalNode string) types.GomegaMatcher {
	return gstruct.PointTo(gstruct.MatchFields(gstruct.IgnoreExtras, gstruct.Fields{
		"Status": gstruct.MatchFields(gstruct.IgnoreExtras, gstruct.Fields{
			"NodeName": Not(Equal(originalNode)),
		}),
	}))
}

func evacuationIsClear() types.GomegaMatcher {
	return gstruct.PointTo(gstruct.MatchFields(gstruct.IgnoreExtras, gstruct.Fields{
		"Status": gstruct.MatchFields(gstruct.IgnoreExtras, gstruct.Fields{
			"EvacuationNodeName": BeEmpty(),
		}),
	}))
}

var _ = Describe(SIG("Live Migration", decorators.RequiresTwoSchedulableNodes, func() {
	var virtClient kubecli.KubevirtClient

	BeforeEach(func() {
		virtClient = kubevirt.Client()
	})

	Context("with a live-migrate eviction strategy set", func() {
		Context("[ref_id:2293] with a VMI running with an eviction strategy set", func() {

			It("[test_id:3242]should block the eviction api and migrate", decorators.Conformance, func() {
				vmi := libvmops.RunVMIAndExpectLaunch(alpineVMIWithEvictionStrategy(), 180)

				originalNode := vmi.Status.NodeName

				pod, err := libpod.GetPodByVirtualMachineInstance(vmi, vmi.Namespace)
				Expect(err).NotTo(HaveOccurred())

				By("Evicting the VMI")
				err = virtClient.CoreV1().Pods(vmi.Namespace).EvictV1(context.Background(), &policyv1.Eviction{ObjectMeta: metav1.ObjectMeta{Name: pod.Name}})
				Expect(errors.IsTooManyRequests(err)).To(BeTrue())

				By("Ensuring the VMI has migrated and lives on another node")
				Eventually(matcher.ThisVMI(vmi)).WithTimeout(time.Minute).WithPolling(time.Second).Should(
					SatisfyAll(
						vmiIsMigratedFrom(originalNode),
						evacuationIsClear(),
						haveMigrationState(gstruct.PointTo(gstruct.MatchFields(
							gstruct.IgnoreExtras, gstruct.Fields{
								"SourceNode": Equal(originalNode),
							},
						))),
					),
				)
			})

			It("[sig-compute][test_id:7680]should delete PDBs created by an old virt-controller", func() {
				By("creating the VMI")
				vmi := alpineVMIWithEvictionStrategy()
				vmi, err := virtClient.VirtualMachineInstance(vmi.Namespace).Create(context.Background(), vmi, metav1.CreateOptions{})
				Expect(err).ToNot(HaveOccurred())
				By("waiting for VMI")
				libwait.WaitForSuccessfulVMIStart(vmi,
					libwait.WithTimeout(60),
				)

				By("Adding a fake old virt-controller PDB")
				pdb, err := virtClient.PolicyV1().PodDisruptionBudgets(vmi.Namespace).Create(context.Background(), &policyv1.PodDisruptionBudget{
					ObjectMeta: metav1.ObjectMeta{
						OwnerReferences: []metav1.OwnerReference{
							*metav1.NewControllerRef(vmi, v1.VirtualMachineInstanceGroupVersionKind),
						},
						GenerateName: "kubevirt-disruption-budget-",
					},
					Spec: policyv1.PodDisruptionBudgetSpec{
						MinAvailable: pointer.P(intstr.FromInt32(2)),
						Selector: &metav1.LabelSelector{
							MatchLabels: map[string]string{
								v1.CreatedByLabel: string(vmi.UID),
							},
						},
					},
				}, metav1.CreateOptions{})
				Expect(err).ToNot(HaveOccurred())

				By("checking that the PDB disappeared")
				Eventually(func() error {
					_, err := virtClient.PolicyV1().PodDisruptionBudgets(vmi.Namespace).Get(context.Background(), pdb.Name, metav1.GetOptions{})
					return err
				}, 60*time.Second, 1*time.Second).Should(MatchError(errors.IsNotFound, "k8serrors.IsNotFound"))
			})

			It("[test_id:3244]should block the eviction api while a slow migration is in progress", func() {
				By("Starting the VirtualMachineInstance")
				vmi := libvmops.RunVMIAndExpectLaunch(fedoraVMIWithEvictionStrategy(), 240)

				By("Checking that the VirtualMachineInstance console has expected output")
				Expect(console.LoginToFedora(vmi)).To(Succeed())

				Eventually(matcher.ThisVMI(vmi), 12*time.Minute, 2*time.Second).Should(matcher.HaveConditionTrue(v1.VirtualMachineInstanceAgentConnected))

				runStressTest(vmi, stressDefaultVMSize)

				// execute a migration, wait for finalized state
				By("Starting the Migration")
				migration := libmigration.New(vmi.Name, vmi.Namespace)
				migration, err := virtClient.VirtualMachineInstanceMigration(vmi.Namespace).Create(context.Background(), migration, metav1.CreateOptions{})
				Expect(err).ToNot(HaveOccurred())

				By("Waiting until we have two available pods")
				var pods *k8sv1.PodList
				Eventually(func() []k8sv1.Pod {
					labelSelector := fmt.Sprintf("%s=%s", v1.CreatedByLabel, vmi.GetUID())
					fieldSelector := fmt.Sprintf("status.phase==%s", k8sv1.PodRunning)
					pods, err = virtClient.CoreV1().Pods(vmi.Namespace).List(context.Background(), metav1.ListOptions{LabelSelector: labelSelector, FieldSelector: fieldSelector})
					Expect(err).ToNot(HaveOccurred())
					return pods.Items
				}, 90*time.Second, 500*time.Millisecond).Should(HaveLen(2))

				By("Verifying at least once that both pods are protected")
				for _, pod := range pods.Items {
					err := virtClient.CoreV1().Pods(vmi.Namespace).EvictV1(context.Background(), &policyv1.Eviction{ObjectMeta: metav1.ObjectMeta{Name: pod.Name}})
					Expect(errors.IsTooManyRequests(err)).To(BeTrue(), "expected TooManyRequests error, got: %v", err)
				}
				By("Verifying that both pods are protected by the PodDisruptionBudget for the whole migration")
				getOptions := metav1.GetOptions{}
				Eventually(func() v1.VirtualMachineInstanceMigrationPhase {
					currentMigration, err := virtClient.VirtualMachineInstanceMigration(vmi.Namespace).Get(context.Background(), migration.Name, getOptions)
					Expect(err).ToNot(HaveOccurred())
					Expect(currentMigration.Status.Phase).NotTo(Equal(v1.MigrationFailed))
					for _, p := range pods.Items {
						pod, err := virtClient.CoreV1().Pods(vmi.Namespace).Get(context.Background(), p.Name, getOptions)
						if err != nil || pod.Status.Phase != k8sv1.PodRunning {
							continue
						}

						deleteOptions := &metav1.DeleteOptions{Preconditions: &metav1.Preconditions{ResourceVersion: &pod.ResourceVersion}}
						eviction := &policyv1.Eviction{ObjectMeta: metav1.ObjectMeta{Name: pod.Name}, DeleteOptions: deleteOptions}
						err = virtClient.CoreV1().Pods(vmi.Namespace).EvictV1(context.Background(), eviction)
						Expect(errors.IsTooManyRequests(err)).To(BeTrue(), "expected TooManyRequests error, got: %v", err)
					}
					return currentMigration.Status.Phase
				}, 180*time.Second, 500*time.Millisecond).Should(Equal(v1.MigrationSucceeded))
			})

			Context(" with node tainted during node drain", Serial, func() {

				var (
					nodeAffinity     *k8sv1.NodeAffinity
					nodeAffinityTerm k8sv1.PreferredSchedulingTerm
				)

				expectVMIMigratedToAnotherNode := func(vmiNamespace, vmiName, sourceNodeName string) {
					EventuallyWithOffset(1, func() error {
						vmi, err := virtClient.VirtualMachineInstance(vmiNamespace).Get(context.Background(), vmiName, metav1.GetOptions{})
						if err != nil {
							return err
						} else if vmi.Status.NodeName == sourceNodeName {
							return fmt.Errorf("VMI still exists on the same node")
						} else if vmi.Status.MigrationState == nil || vmi.Status.MigrationState.SourceNode != sourceNodeName {
							return fmt.Errorf("VMI did not migrate yet")
						} else if vmi.Status.EvacuationNodeName != "" {
							return fmt.Errorf("evacuation node name is still set on the VMI")
						}

						Expect(vmi.Status.Phase).To(Equal(v1.Running))

						return nil
					}, 180*time.Second, 1*time.Second).ShouldNot(HaveOccurred(), "VMI should still be running")
				}

				BeforeEach(func() {
					// Taints defined by k8s are special and can't be applied manually.
					// Temporarily configure KubeVirt to use something else for the duration of these tests.
					if libnode.GetNodeDrainKey() == "node.kubernetes.io/unschedulable" {
						cfg := getCurrentKvConfig(virtClient)
						cfg.MigrationConfiguration.NodeDrainTaintKey = pointer.P("kubevirt.io/drain")
						config.UpdateKubeVirtConfigValueAndWait(cfg)
					}

					controlPlaneNodes := libnode.GetControlPlaneNodes(virtClient)

					// This nodeAffinity will make sure the vmi, initially, will not be scheduled in the control-plane node in those clusters where there is only one.
					// This is mandatory, since later the tests will drain the node where the vmi will be scheduled.
					nodeAffinityTerm = k8sv1.PreferredSchedulingTerm{
						Weight: int32(1),
						Preference: k8sv1.NodeSelectorTerm{
							MatchExpressions: []k8sv1.NodeSelectorRequirement{
								{Key: k8sv1.LabelHostname, Operator: k8sv1.NodeSelectorOpNotIn, Values: []string{controlPlaneNodes.Items[0].Name}},
							},
						},
					}

					nodeAffinity = &k8sv1.NodeAffinity{
						PreferredDuringSchedulingIgnoredDuringExecution: []k8sv1.PreferredSchedulingTerm{nodeAffinityTerm},
					}
				})

				It("[test_id:6982]should migrate a VMI only one time", func() {
					vmi := fedoraVMIWithEvictionStrategy()
					vmi.Spec.Affinity = &k8sv1.Affinity{NodeAffinity: nodeAffinity}

					By("Starting the VirtualMachineInstance")
					vmi = libvmops.RunVMIAndExpectLaunch(vmi, 180)

					Eventually(matcher.ThisVMI(vmi), 12*time.Minute, 2*time.Second).Should(matcher.HaveConditionTrue(v1.VirtualMachineInstanceAgentConnected))

					node := vmi.Status.NodeName
					libnode.TemporaryNodeDrain(node)
					expectVMIMigratedToAnotherNode(vmi.Namespace, vmi.Name, node)

					Consistently(func() error {
						migrations, err := virtClient.VirtualMachineInstanceMigration(vmi.Namespace).List(context.Background(), metav1.ListOptions{})
						if err != nil {
							return err
						}
						if len(migrations.Items) > 1 {
							return fmt.Errorf("should have only 1 migration issued for evacuation of 1 VM")
						}
						return nil
					}, 20*time.Second, 1*time.Second).ShouldNot(HaveOccurred())

				})

				It("[test_id:2221] should migrate a VMI under load to another node", func() {
					vmi := fedoraVMIWithEvictionStrategy()
					vmi.Spec.Affinity = &k8sv1.Affinity{NodeAffinity: nodeAffinity}

					By("Starting the VirtualMachineInstance")
					vmi = libvmops.RunVMIAndExpectLaunch(vmi, 180)

					By("Checking that the VirtualMachineInstance console has expected output")
					Expect(console.LoginToFedora(vmi)).To(Succeed())

					Eventually(matcher.ThisVMI(vmi), 12*time.Minute, 2*time.Second).Should(matcher.HaveConditionTrue(v1.VirtualMachineInstanceAgentConnected))

					// Put VMI under load
					runStressTest(vmi, stressDefaultVMSize)

					node := vmi.Status.NodeName
					libnode.TemporaryNodeDrain(node)
					expectVMIMigratedToAnotherNode(vmi.Namespace, vmi.Name, node)
				})

				It("[test_id:2222] should migrate a VMI when custom taint key is configured", func() {
					vmi := alpineVMIWithEvictionStrategy()
					vmi.Spec.Affinity = &k8sv1.Affinity{NodeAffinity: nodeAffinity}

					By("Configuring a custom nodeDrainTaintKey in kubevirt configuration")
					cfg := getCurrentKvConfig(virtClient)
					cfg.MigrationConfiguration.NodeDrainTaintKey = pointer.P("kubevirt.io/alt-drain")
					config.UpdateKubeVirtConfigValueAndWait(cfg)

					By("Starting the VirtualMachineInstance")
					vmi = libvmops.RunVMIAndExpectLaunch(vmi, 180)

					node := vmi.Status.NodeName
					libnode.TemporaryNodeDrain(node)
					expectVMIMigratedToAnotherNode(vmi.Namespace, vmi.Name, node)
				})

				It("[test_id:2224] should handle mixture of VMs with different eviction strategies.", func() {
					const labelKey = "testkey"

					// give an affinity rule to ensure the vmi's get placed on the same node.
					podAffinityTerm := k8sv1.WeightedPodAffinityTerm{
						Weight: int32(1),
						PodAffinityTerm: k8sv1.PodAffinityTerm{
							LabelSelector: &metav1.LabelSelector{
								MatchExpressions: []metav1.LabelSelectorRequirement{
									{
										Key:      labelKey,
										Operator: metav1.LabelSelectorOpIn,
										Values:   []string{""}},
								},
							},
							TopologyKey: k8sv1.LabelHostname,
						},
					}
					vmi_evict1 := alpineVMIWithEvictionStrategy(
						libvmi.WithNamespace(testsuite.GetTestNamespace(nil)),
						libvmi.WithLabel(labelKey, ""),
						libvmi.WithPreferredPodAffinity(podAffinityTerm),
						libvmi.WithPreferredNodeAffinity(nodeAffinityTerm),
					)
					vmi_evict2 := alpineVMIWithEvictionStrategy(
						libvmi.WithNamespace(testsuite.GetTestNamespace(nil)),
						libvmi.WithLabel(labelKey, ""),
						libvmi.WithPreferredPodAffinity(podAffinityTerm),
						libvmi.WithPreferredNodeAffinity(nodeAffinityTerm),
					)
					vmi_noevict := libvmifact.NewAlpine(
						libvmi.WithNamespace(testsuite.GetTestNamespace(nil)),
						libvmi.WithInterface(libvmi.InterfaceDeviceWithMasqueradeBinding()),
						libvmi.WithNetwork(v1.DefaultPodNetwork()),
						libvmi.WithEvictionStrategy(v1.EvictionStrategyNone),
						libvmi.WithLabel(labelKey, ""),
						libvmi.WithPreferredPodAffinity(podAffinityTerm),
						libvmi.WithPreferredNodeAffinity(nodeAffinityTerm),
					)

					By("Starting the VirtualMachineInstance with eviction set to live migration")
					vm_evict1 := libvmi.NewVirtualMachine(vmi_evict1)
					vm_evict2 := libvmi.NewVirtualMachine(vmi_evict2)
					vm_noevict := libvmi.NewVirtualMachine(vmi_noevict)

					// post VMs
					vm_evict1, err := virtClient.VirtualMachine(vm_evict1.Namespace).Create(context.Background(), vm_evict1, metav1.CreateOptions{})
					Expect(err).ToNot(HaveOccurred())
					vm_evict2, err = virtClient.VirtualMachine(vm_evict2.Namespace).Create(context.Background(), vm_evict2, metav1.CreateOptions{})
					Expect(err).ToNot(HaveOccurred())
					vm_noevict, err = virtClient.VirtualMachine(vm_noevict.Namespace).Create(context.Background(), vm_noevict, metav1.CreateOptions{})
					Expect(err).ToNot(HaveOccurred())

					// Start VMs
					vm_evict1 = libvmops.StartVirtualMachine(vm_evict1)
					vm_evict2 = libvmops.StartVirtualMachine(vm_evict2)
					vm_noevict = libvmops.StartVirtualMachine(vm_noevict)

					// Get VMIs
					vmi_evict1, err = virtClient.VirtualMachineInstance(vmi_evict1.Namespace).Get(context.Background(), vmi_evict1.Name, metav1.GetOptions{})
					vmi_evict2, err = virtClient.VirtualMachineInstance(vmi_evict1.Namespace).Get(context.Background(), vmi_evict2.Name, metav1.GetOptions{})
					vmi_noevict, err = virtClient.VirtualMachineInstance(vmi_evict1.Namespace).Get(context.Background(), vmi_noevict.Name, metav1.GetOptions{})

					By("Verifying all VMIs are collcated on the same node")
					Expect(vmi_evict1.Status.NodeName).To(Equal(vmi_evict2.Status.NodeName))
					Expect(vmi_evict1.Status.NodeName).To(Equal(vmi_noevict.Status.NodeName))

					node := vmi_evict1.Status.NodeName
					libnode.TemporaryNodeDrain(node)

					By("Verify expected vmis migrated after node drain completes")
					// verify migrated where expected to migrate.
					Eventually(func() error {
						vmi, err := virtClient.VirtualMachineInstance(vmi_evict1.Namespace).Get(context.Background(), vmi_evict1.Name, metav1.GetOptions{})
						if err != nil {
							return err
						} else if vmi.Status.NodeName == node {
							return fmt.Errorf("VMI still exist on the same node")
						} else if vmi.Status.MigrationState == nil || vmi.Status.MigrationState.SourceNode != node {
							return fmt.Errorf("VMI did not migrate yet")
						}

						vmi, err = virtClient.VirtualMachineInstance(vmi_evict2.Namespace).Get(context.Background(), vmi_evict2.Name, metav1.GetOptions{})
						if err != nil {
							return err
						} else if vmi.Status.NodeName == node {
							return fmt.Errorf("VMI still exist on the same node")
						} else if vmi.Status.MigrationState == nil || vmi.Status.MigrationState.SourceNode != node {
							return fmt.Errorf("VMI did not migrate yet")
						}

						// This VMI should be terminated
						vmi, err = virtClient.VirtualMachineInstance(vmi_noevict.Namespace).Get(context.Background(), vmi_noevict.Name, metav1.GetOptions{})
						if err != nil {
							return err
						} else if vmi.Status.NodeName == node {
							return fmt.Errorf("VMI still exist on the same node")
						}
						// this VM should not have migrated. Instead it should have been shutdown and started on the other node.
						Expect(vmi.Status.MigrationState).To(BeNil())
						return nil
					}, 180*time.Second, 1*time.Second).ShouldNot(HaveOccurred())

				})
			})
		})
		Context("with multiple VMIs with eviction policies set", Serial, func() {
			It("[release-blocker][test_id:3245]should not migrate more than two VMIs at the same time from a node", func() {
				var vmis []*v1.VirtualMachineInstance
				for i := 0; i < 4; i++ {
					vmi := alpineVMIWithEvictionStrategy()
					vmi.Spec.NodeSelector = map[string]string{cleanup.TestLabelForNamespace(vmi.Namespace): "target"}
					vmis = append(vmis, vmi)
				}

				By("selecting a node as the source")
				sourceNode := libnode.GetAllSchedulableNodes(virtClient).Items[0]
				libnode.AddLabelToNode(sourceNode.Name, cleanup.TestLabelForNamespace(vmis[0].Namespace), "target")

				By("starting four VMIs on that node")
				for _, vmi := range vmis {
					_, err := virtClient.VirtualMachineInstance(vmi.Namespace).Create(context.Background(), vmi, metav1.CreateOptions{})
					Expect(err).ToNot(HaveOccurred())
				}

				By("waiting until the VMIs are ready")
				for _, vmi := range vmis {
					libwait.WaitForSuccessfulVMIStart(vmi,
						libwait.WithTimeout(180),
					)
				}

				By("selecting a node as the target")
				targetNode := libnode.GetAllSchedulableNodes(virtClient).Items[1]
				libnode.AddLabelToNode(targetNode.Name, cleanup.TestLabelForNamespace(vmis[0].Namespace), "target")

				By("tainting the source node as non-schedulabele")
				libnode.Taint(sourceNode.Name, libnode.GetNodeDrainKey(), k8sv1.TaintEffectNoSchedule)

				By("waiting until migration kicks in")
				Eventually(func() int {
					migrationList, err := virtClient.VirtualMachineInstanceMigration(k8sv1.NamespaceAll).List(context.Background(), metav1.ListOptions{})
					Expect(err).ToNot(HaveOccurred())

					runningMigrations := filterRunningMigrations(migrationList.Items)

					return len(runningMigrations)
				}, 2*time.Minute, 1*time.Second).Should(BeNumerically(">", 0))

				By("checking that all VMIs were migrated, and we never see more than two running migrations in parallel")
				Eventually(func() []string {
					var nodes []string
					for _, vmi := range vmis {
						vmi, err := virtClient.VirtualMachineInstance(vmi.Namespace).Get(context.Background(), vmi.Name, metav1.GetOptions{})
						Expect(err).ToNot(HaveOccurred())
						nodes = append(nodes, vmi.Status.NodeName)
					}

					migrationList, err := virtClient.VirtualMachineInstanceMigration(k8sv1.NamespaceAll).List(context.Background(), metav1.ListOptions{})
					Expect(err).ToNot(HaveOccurred())

					runningMigrations := filterRunningMigrations(migrationList.Items)
					Expect(len(runningMigrations)).To(BeNumerically("<=", 2))

					return nodes
				}, 4*time.Minute, 1*time.Second).Should(ConsistOf(
					targetNode.Name,
					targetNode.Name,
					targetNode.Name,
					targetNode.Name,
				))

				By("Checking that all migrated VMIs have the new pod IP address on VMI status")
				for _, vmi := range vmis {
					Eventually(func() error {
						newvmi, err := virtClient.VirtualMachineInstance(vmi.Namespace).Get(context.Background(), vmi.Name, metav1.GetOptions{})
						Expect(err).ToNot(HaveOccurred(), "Should successfully get new VMI")
						vmiPod, err := libpod.GetPodByVirtualMachineInstance(newvmi, newvmi.Namespace)
						Expect(err).NotTo(HaveOccurred())
						return libnet.ValidateVMIandPodIPMatch(newvmi, vmiPod)
					}, time.Minute, time.Second).Should(Succeed(), "Should match PodIP with latest VMI Status after migration")
				}
			})
		})
	})

	Describe(" with a cluster-wide live-migrate eviction strategy set", Serial, func() {
		var originalKV *v1.KubeVirt

		BeforeEach(func() {
			kv := libkubevirt.GetCurrentKv(virtClient)
			originalKV = kv.DeepCopy()

			kv.Spec.Configuration.EvictionStrategy = pointer.P(v1.EvictionStrategyLiveMigrate)
			config.UpdateKubeVirtConfigValueAndWait(kv.Spec.Configuration)
		})

		AfterEach(func() {
			config.UpdateKubeVirtConfigValueAndWait(originalKV.Spec.Configuration)
		})

		Context("with a VMI running", func() {
			Context("with no eviction strategy set", func() {
				It("[test_id:10155]should block the eviction api and migrate", func() {
					// no EvictionStrategy set
					vmi := libvmifact.NewAlpine(
						libvmi.WithInterface(libvmi.InterfaceDeviceWithMasqueradeBinding()),
						libvmi.WithNetwork(v1.DefaultPodNetwork()),
					)

					vmi = libvmops.RunVMIAndExpectLaunch(vmi, 180)
					vmiNodeOrig := vmi.Status.NodeName
					pod, err := libpod.GetPodByVirtualMachineInstance(vmi, vmi.Namespace)
					Expect(err).NotTo(HaveOccurred())
					err = virtClient.CoreV1().Pods(vmi.Namespace).EvictV1(context.Background(), &policyv1.Eviction{ObjectMeta: metav1.ObjectMeta{Name: pod.Name}})
					Expect(errors.IsTooManyRequests(err)).To(BeTrue())

					By("Ensuring the VMI has migrated and lives on another node")
					Eventually(func() error {
						vmi, err := virtClient.VirtualMachineInstance(vmi.Namespace).Get(context.Background(), vmi.Name, metav1.GetOptions{})
						if err != nil {
							return err
						}

						if vmi.Status.NodeName == vmiNodeOrig {
							return fmt.Errorf("VMI is still on the same node")
						}

						if vmi.Status.MigrationState == nil || vmi.Status.MigrationState.SourceNode != vmiNodeOrig {
							return fmt.Errorf("VMI did not migrate yet")
						}

						if vmi.Status.EvacuationNodeName != "" {
							return fmt.Errorf("VMI is still evacuating: %v", vmi.Status.EvacuationNodeName)
						}

						return nil
					}, 360*time.Second, 1*time.Second).ShouldNot(HaveOccurred())
					resVMI, err := virtClient.VirtualMachineInstance(vmi.Namespace).Get(context.Background(), vmi.Name, metav1.GetOptions{})
					Expect(err).ShouldNot(HaveOccurred())
					Expect(resVMI.Status.EvacuationNodeName).To(Equal(""), "vmi evacuation state should be clean")
				})
			})

			Context("with eviction strategy set to 'None'", func() {
				It("[test_id:10156]The VMI should get evicted", func() {
					vmi := libvmifact.NewAlpine(
						libvmi.WithInterface(libvmi.InterfaceDeviceWithMasqueradeBinding()),
						libvmi.WithNetwork(v1.DefaultPodNetwork()),
						libvmi.WithEvictionStrategy(v1.EvictionStrategyNone),
					)
					vmi = libvmops.RunVMIAndExpectLaunch(vmi, 180)
					pod, err := libpod.GetPodByVirtualMachineInstance(vmi, vmi.Namespace)
					Expect(err).NotTo(HaveOccurred())
					err = virtClient.CoreV1().Pods(vmi.Namespace).EvictV1(context.Background(), &policyv1.Eviction{ObjectMeta: metav1.ObjectMeta{Name: pod.Name}})
					Expect(err).ToNot(HaveOccurred())
					Expect(matcher.ThisVMI(vmi)()).To(evacuationIsClear())
				})
			})
		})
	})
}))

func fedoraVMIWithEvictionStrategy() *v1.VirtualMachineInstance {
	return libvmifact.NewFedora(libnet.WithMasqueradeNetworking(),
		libvmi.WithMemoryRequest(fedoraVMSize),
		libvmi.WithEvictionStrategy(v1.EvictionStrategyLiveMigrate),
		libvmi.WithNamespace(testsuite.GetTestNamespace(nil)))
}

func alpineVMIWithEvictionStrategy(additionalOpts ...libvmi.Option) *v1.VirtualMachineInstance {
	opts := []libvmi.Option{libnet.WithMasqueradeNetworking(), libvmi.WithEvictionStrategy(v1.EvictionStrategyLiveMigrate), libvmi.WithNamespace(testsuite.GetTestNamespace(nil))}
	opts = append(opts, additionalOpts...)

	return libvmifact.NewAlpine(opts...)
}

func filterRunningMigrations(migrations []v1.VirtualMachineInstanceMigration) []v1.VirtualMachineInstanceMigration {
	runningMigrations := []v1.VirtualMachineInstanceMigration{}
	for _, migration := range migrations {
		if migration.IsRunning() {
			runningMigrations = append(runningMigrations, migration)
		}
	}
	return runningMigrations
}
