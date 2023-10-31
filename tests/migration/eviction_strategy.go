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

package migration

import (
	"context"
	"fmt"
	"time"

	kvpointer "kubevirt.io/kubevirt/pkg/pointer"

	"kubevirt.io/kubevirt/tests/testsuite"

	"kubevirt.io/kubevirt/tests/clientcmd"
	"kubevirt.io/kubevirt/tests/framework/checks"
	"kubevirt.io/kubevirt/tests/framework/cleanup"
	"kubevirt.io/kubevirt/tests/framework/kubevirt"
	"kubevirt.io/kubevirt/tests/framework/matcher"
	"kubevirt.io/kubevirt/tests/libnet"
	"kubevirt.io/kubevirt/tests/libnode"
	"kubevirt.io/kubevirt/tests/libvmi"
	"kubevirt.io/kubevirt/tests/util"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	k8sv1 "k8s.io/api/core/v1"
	policyv1 "k8s.io/api/policy/v1"
	policyv1beta1 "k8s.io/api/policy/v1beta1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"

	"kubevirt.io/kubevirt/tests/libwait"

	v1 "kubevirt.io/api/core/v1"
	"kubevirt.io/client-go/kubecli"

	"kubevirt.io/kubevirt/tests"
	"kubevirt.io/kubevirt/tests/console"
	cd "kubevirt.io/kubevirt/tests/containerdisk"
)

var _ = SIGMigrationDescribe("Live Migration", func() {
	var (
		virtClient kubecli.KubevirtClient
		err        error
	)

	BeforeEach(func() {
		checks.SkipIfMigrationIsNotPossible()
		virtClient = kubevirt.Client()
	})

	setControlPlaneSchedulability := func(setSchedulable bool) {
		Expect(CurrentSpecReport().IsSerial).To(BeTrue(), "Tests which alter the cluster nodes must not be executed in parallel, see https://onsi.github.io/ginkgo/#serial-specs")
		controlPlaneNodes := libnode.GetControlPlaneNodes(virtClient)
		for _, node := range controlPlaneNodes.Items {
			if setSchedulable {
				libnode.SetNodeSchedulable(node.Name, virtClient)
			} else {
				libnode.SetNodeUnschedulable(node.Name, virtClient)
			}
		}
	}

	// temporaryNodeDrain also sets the `NoSchedule` taint on the node.
	// nodes with this taint will be reset to their original state on each
	// test teardown by the test framework. Check `libnode.CleanNodes`.
	// TODO: move this function in `libnode` package. First resolve cycle in dependency graph
	//  .-> //tests/testsuite:go_default_library
	//  |   //tests/libnode:go_default_library
	//  |   //tests/clientcmd:go_default_library
	//  `-- //tests/testsuite:go_default_library
	temporaryNodeDrain := func(nodeName string) {
		By("taining the node with `NoExecute`, the framework will reset the node's taints and un-schedulable properties on test teardown")
		libnode.Taint(nodeName, libnode.GetNodeDrainKey(), k8sv1.TaintEffectNoSchedule)

		By(fmt.Sprintf("Draining node %s", nodeName))
		// we can't really expect an error during node drain because vms with eviction strategy can be migrated by the
		// time that we call it.
		vmiSelector := v1.AppLabel + "=virt-launcher"
		k8sClient := clientcmd.GetK8sCmdClient()
		if k8sClient == "oc" {
			_, _, err := clientcmd.RunCommandWithNS("", k8sClient, "adm", "drain", nodeName, "--delete-emptydir-data", "--pod-selector", vmiSelector,
				"--ignore-daemonsets=true", "--force", "--timeout=180s")
			Expect(err).ToNot(HaveOccurred())
		} else {
			_, _, err := clientcmd.RunCommandWithNS("", k8sClient, "drain", nodeName, "--delete-emptydir-data", "--pod-selector", vmiSelector,
				"--ignore-daemonsets=true", "--force", "--timeout=180s")
			Expect(err).ToNot(HaveOccurred())
		}
	}

	Context("with a live-migrate eviction strategy set", func() {
		Context("[ref_id:2293] with a VMI running with an eviction strategy set", func() {

			var vmi *v1.VirtualMachineInstance

			BeforeEach(func() {
				vmi = alpineVMIWithEvictionStrategy()
			})

			It("[test_id:3242]should block the eviction api and migrate", func() {
				vmi = tests.RunVMIAndExpectLaunch(vmi, 180)
				vmiNodeOrig := vmi.Status.NodeName
				pod := tests.GetRunningPodByVirtualMachineInstance(vmi, vmi.Namespace)
				err := virtClient.CoreV1().Pods(vmi.Namespace).EvictV1beta1(context.Background(), &policyv1beta1.Eviction{ObjectMeta: metav1.ObjectMeta{Name: pod.Name}})
				Expect(errors.IsTooManyRequests(err)).To(BeTrue())

				By("Ensuring the VMI has migrated and lives on another node")
				Eventually(func() error {
					vmi, err := virtClient.VirtualMachineInstance(vmi.Namespace).Get(context.Background(), vmi.Name, &metav1.GetOptions{})
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
				resVMI, err := virtClient.VirtualMachineInstance(vmi.Namespace).Get(context.Background(), vmi.Name, &metav1.GetOptions{})
				Expect(err).ShouldNot(HaveOccurred())
				Expect(resVMI.Status.EvacuationNodeName).To(Equal(""), "vmi evacuation state should be clean")
			})

			It("[sig-compute][test_id:3243]should recreate the PDB if VMIs with similar names are recreated", func() {
				for x := 0; x < 3; x++ {
					By("creating the VMI")
					_, err := virtClient.VirtualMachineInstance(vmi.Namespace).Create(context.Background(), vmi)
					Expect(err).ToNot(HaveOccurred())

					By("checking that the PDB appeared")
					Eventually(matcher.AllPDBs(vmi.Namespace), 3*time.Second, 500*time.Millisecond).Should(HaveLen(1))

					By("waiting for VMI")
					libwait.WaitForSuccessfulVMIStart(vmi,
						libwait.WithTimeout(60),
					)
					By("deleting the VMI")
					Expect(virtClient.VirtualMachineInstance(vmi.Namespace).Delete(context.Background(), vmi.Name, &metav1.DeleteOptions{})).To(Succeed())
					By("checking that the PDB disappeared")
					Eventually(matcher.AllPDBs(vmi.Namespace), 3*time.Second, 500*time.Millisecond).Should(BeEmpty())
					Eventually(matcher.ThisVMI(vmi), 60*time.Second, 500*time.Millisecond).Should(matcher.BeGone())
				}
			})

			It("should create the PDB if VMI is live-migratable and has the LiveMigrateIfPossible strategy set", func() {
				By("creating the VMI")
				strategy := v1.EvictionStrategyLiveMigrateIfPossible
				vmi.Spec.EvictionStrategy = &strategy
				vmi, err = virtClient.VirtualMachineInstance(vmi.Namespace).Create(context.Background(), vmi)
				Expect(err).ToNot(HaveOccurred())

				By("checking that the PDB appeared, with extra time since schedulability needs to be determined first in the cluster")
				Eventually(matcher.AllPDBs(vmi.Namespace), 60*time.Second, 500*time.Millisecond).Should(HaveLen(1))
				By("waiting for VMI")
				libwait.WaitForSuccessfulVMIStart(vmi,
					libwait.WithTimeout(60),
				)

				By("deleting the VMI")
				Expect(virtClient.VirtualMachineInstance(vmi.Namespace).Delete(context.Background(), vmi.Name, &metav1.DeleteOptions{})).To(Succeed())
				By("checking that the PDB disappeared")
				Eventually(matcher.AllPDBs(vmi.Namespace), 3*time.Second, 500*time.Millisecond).Should(BeEmpty())
			})

			It("[sig-compute][test_id:7680]should delete PDBs created by an old virt-controller", func() {
				By("creating the VMI")
				createdVMI, err := virtClient.VirtualMachineInstance(vmi.Namespace).Create(context.Background(), vmi)
				Expect(err).ToNot(HaveOccurred())
				By("waiting for VMI")
				libwait.WaitForSuccessfulVMIStart(createdVMI,
					libwait.WithTimeout(60),
				)

				By("Adding a fake old virt-controller PDB")
				two := intstr.FromInt(2)
				pdb, err := virtClient.PolicyV1().PodDisruptionBudgets(createdVMI.Namespace).Create(context.Background(), &policyv1.PodDisruptionBudget{
					ObjectMeta: metav1.ObjectMeta{
						OwnerReferences: []metav1.OwnerReference{
							*metav1.NewControllerRef(createdVMI, v1.VirtualMachineInstanceGroupVersionKind),
						},
						GenerateName: "kubevirt-disruption-budget-",
					},
					Spec: policyv1.PodDisruptionBudgetSpec{
						MinAvailable: &two,
						Selector: &metav1.LabelSelector{
							MatchLabels: map[string]string{
								v1.CreatedByLabel: string(createdVMI.UID),
							},
						},
					},
				}, metav1.CreateOptions{})
				Expect(err).ToNot(HaveOccurred())

				By("checking that the PDB disappeared")
				Eventually(func() bool {
					_, err := virtClient.PolicyV1().PodDisruptionBudgets(createdVMI.Namespace).Get(context.Background(), pdb.Name, metav1.GetOptions{})
					return errors.IsNotFound(err)
				}, 60*time.Second, 1*time.Second).Should(BeTrue())
			})

			It("[test_id:3244]should block the eviction api while a slow migration is in progress", func() {
				vmi = fedoraVMIWithEvictionStrategy()

				By("Starting the VirtualMachineInstance")
				vmi = tests.RunVMIAndExpectLaunch(vmi, 240)

				By("Checking that the VirtualMachineInstance console has expected output")
				Expect(console.LoginToFedora(vmi)).To(Succeed())

				Eventually(matcher.ThisVMI(vmi), 12*time.Minute, 2*time.Second).Should(matcher.HaveConditionTrue(v1.VirtualMachineInstanceAgentConnected))

				runStressTest(vmi, stressDefaultVMSize, stressDefaultSleepDuration)

				// execute a migration, wait for finalized state
				By("Starting the Migration")
				migration := tests.NewRandomMigration(vmi.Name, vmi.Namespace)
				migration, err := virtClient.VirtualMachineInstanceMigration(vmi.Namespace).Create(migration, &metav1.CreateOptions{})
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
					err := virtClient.CoreV1().Pods(vmi.Namespace).EvictV1beta1(context.Background(), &policyv1beta1.Eviction{ObjectMeta: metav1.ObjectMeta{Name: pod.Name}})
					Expect(errors.IsTooManyRequests(err)).To(BeTrue(), "expected TooManyRequests error, got: %v", err)
				}
				By("Verifying that both pods are protected by the PodDisruptionBudget for the whole migration")
				getOptions := metav1.GetOptions{}
				Eventually(func() v1.VirtualMachineInstanceMigrationPhase {
					currentMigration, err := virtClient.VirtualMachineInstanceMigration(vmi.Namespace).Get(migration.Name, &getOptions)
					Expect(err).ToNot(HaveOccurred())
					Expect(currentMigration.Status.Phase).NotTo(Equal(v1.MigrationFailed))
					for _, p := range pods.Items {
						pod, err := virtClient.CoreV1().Pods(vmi.Namespace).Get(context.Background(), p.Name, getOptions)
						if err != nil || pod.Status.Phase != k8sv1.PodRunning {
							continue
						}

						deleteOptions := &metav1.DeleteOptions{Preconditions: &metav1.Preconditions{ResourceVersion: &pod.ResourceVersion}}
						eviction := &policyv1beta1.Eviction{ObjectMeta: metav1.ObjectMeta{Name: pod.Name}, DeleteOptions: deleteOptions}
						err = virtClient.CoreV1().Pods(vmi.Namespace).EvictV1beta1(context.Background(), eviction)
						Expect(errors.IsTooManyRequests(err)).To(BeTrue(), "expected TooManyRequests error, got: %v", err)
					}
					return currentMigration.Status.Phase
				}, 180*time.Second, 500*time.Millisecond).Should(Equal(v1.MigrationSucceeded))
			})

			Context("[Serial] with node tainted during node drain", Serial, func() {
				BeforeEach(func() {
					// Taints defined by k8s are special and can't be applied manually.
					// Temporarily configure KubeVirt to use something else for the duration of these tests.
					if libnode.GetNodeDrainKey() == "node.kubernetes.io/unschedulable" {
						drain := "kubevirt.io/drain"
						cfg := getCurrentKvConfig(virtClient)
						cfg.MigrationConfiguration.NodeDrainTaintKey = &drain
						tests.UpdateKubeVirtConfigValueAndWait(cfg)
					}
					setControlPlaneSchedulability(false)
				})

				AfterEach(func() {
					setControlPlaneSchedulability(true)
				})

				It("[test_id:6982]should migrate a VMI only one time", func() {
					vmi = fedoraVMIWithEvictionStrategy()

					By("Starting the VirtualMachineInstance")
					vmi = tests.RunVMIAndExpectLaunch(vmi, 180)

					Eventually(matcher.ThisVMI(vmi), 12*time.Minute, 2*time.Second).Should(matcher.HaveConditionTrue(v1.VirtualMachineInstanceAgentConnected))

					// Mark the control-plane nodes as schedulable so we can migrate there
					setControlPlaneSchedulability(true)

					node := vmi.Status.NodeName
					temporaryNodeDrain(node)

					// verify VMI migrated and lives on another node now.
					Eventually(func() error {
						vmi, err := virtClient.VirtualMachineInstance(vmi.Namespace).Get(context.Background(), vmi.Name, &metav1.GetOptions{})
						if err != nil {
							return err
						} else if vmi.Status.NodeName == node {
							return fmt.Errorf("VMI still exist on the same node")
						} else if vmi.Status.MigrationState == nil || vmi.Status.MigrationState.SourceNode != node {
							return fmt.Errorf("VMI did not migrate yet")
						} else if vmi.Status.EvacuationNodeName != "" {
							return fmt.Errorf("evacuation node name is still set on the VMI")
						}

						// VMI should still be running at this point. If it
						// isn't, then there's nothing to be waiting on.
						Expect(vmi.Status.Phase).To(Equal(v1.Running))

						return nil
					}, 180*time.Second, 1*time.Second).ShouldNot(HaveOccurred())

					Consistently(func() error {
						migrations, err := virtClient.VirtualMachineInstanceMigration(vmi.Namespace).List(&metav1.ListOptions{})
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
					vmi = fedoraVMIWithEvictionStrategy()

					By("Starting the VirtualMachineInstance")
					vmi = tests.RunVMIAndExpectLaunch(vmi, 180)

					By("Checking that the VirtualMachineInstance console has expected output")
					Expect(console.LoginToFedora(vmi)).To(Succeed())

					Eventually(matcher.ThisVMI(vmi), 12*time.Minute, 2*time.Second).Should(matcher.HaveConditionTrue(v1.VirtualMachineInstanceAgentConnected))

					// Put VMI under load
					runStressTest(vmi, stressDefaultVMSize, stressDefaultSleepDuration)

					// Mark the control-plane nodes as schedulable so we can migrate there
					setControlPlaneSchedulability(true)

					node := vmi.Status.NodeName
					temporaryNodeDrain(node)

					// verify VMI migrated and lives on another node now.
					Eventually(func() error {
						vmi, err := virtClient.VirtualMachineInstance(vmi.Namespace).Get(context.Background(), vmi.Name, &metav1.GetOptions{})
						if err != nil {
							return err
						} else if vmi.Status.NodeName == node {
							return fmt.Errorf("VMI still exist on the same node")
						} else if vmi.Status.MigrationState == nil || vmi.Status.MigrationState.SourceNode != node {
							return fmt.Errorf("VMI did not migrate yet")
						}

						// VMI should still be running at this point. If it
						// isn't, then there's nothing to be waiting on.
						Expect(vmi.Status.Phase).To(Equal(v1.Running))

						return nil
					}, 180*time.Second, 1*time.Second).ShouldNot(HaveOccurred())
				})

				It("[test_id:2222] should migrate a VMI when custom taint key is configured", func() {
					vmi = alpineVMIWithEvictionStrategy()

					By("Configuring a custom nodeDrainTaintKey in kubevirt configuration")
					cfg := getCurrentKvConfig(virtClient)
					drainKey := "kubevirt.io/alt-drain"
					cfg.MigrationConfiguration.NodeDrainTaintKey = &drainKey
					tests.UpdateKubeVirtConfigValueAndWait(cfg)

					By("Starting the VirtualMachineInstance")
					vmi = tests.RunVMIAndExpectLaunch(vmi, 180)

					// Mark the control-plane nodes as schedulable so we can migrate there
					setControlPlaneSchedulability(true)

					node := vmi.Status.NodeName
					temporaryNodeDrain(node)

					// verify VMI migrated and lives on another node now.
					Eventually(func() error {
						vmi, err := virtClient.VirtualMachineInstance(vmi.Namespace).Get(context.Background(), vmi.Name, &metav1.GetOptions{})
						if err != nil {
							return err
						} else if vmi.Status.NodeName == node {
							return fmt.Errorf("VMI still exist on the same node")
						} else if vmi.Status.MigrationState == nil || vmi.Status.MigrationState.SourceNode != node {
							return fmt.Errorf("VMI did not migrate yet")
						}
						return nil
					}, 180*time.Second, 1*time.Second).ShouldNot(HaveOccurred())
				})

				It("[test_id:2224] should handle mixture of VMs with different eviction strategies.", func() {
					vmi_evict1 := alpineVMIWithEvictionStrategy()
					vmi_evict2 := alpineVMIWithEvictionStrategy()
					vmi_noevict := tests.NewRandomVMIWithEphemeralDisk(cd.ContainerDiskFor(cd.ContainerDiskAlpine))
					vmi_noevict.Spec.EvictionStrategy = kvpointer.P(v1.EvictionStrategyNone)

					labelKey := "testkey"
					labels := map[string]string{
						labelKey: "",
					}

					// give an affinity rule to ensure the vmi's get placed on the same node.
					affinityRule := &k8sv1.Affinity{
						PodAffinity: &k8sv1.PodAffinity{
							PreferredDuringSchedulingIgnoredDuringExecution: []k8sv1.WeightedPodAffinityTerm{
								{
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
										TopologyKey: "kubernetes.io/hostname",
									},
								},
							},
						},
					}

					vmi_evict1.Labels = labels
					vmi_evict2.Labels = labels
					vmi_noevict.Labels = labels

					vmi_evict1.Spec.Affinity = affinityRule
					vmi_evict2.Spec.Affinity = affinityRule
					vmi_noevict.Spec.Affinity = affinityRule

					By("Starting the VirtualMachineInstance with eviction set to live migration")
					vm_evict1 := tests.NewRandomVirtualMachine(vmi_evict1, false)
					vm_evict2 := tests.NewRandomVirtualMachine(vmi_evict2, false)
					vm_noevict := tests.NewRandomVirtualMachine(vmi_noevict, false)

					// post VMs
					vm_evict1, err = virtClient.VirtualMachine(vm_evict1.Namespace).Create(context.Background(), vm_evict1)
					Expect(err).ToNot(HaveOccurred())
					vm_evict2, err = virtClient.VirtualMachine(vm_evict2.Namespace).Create(context.Background(), vm_evict2)
					Expect(err).ToNot(HaveOccurred())
					vm_noevict, err = virtClient.VirtualMachine(vm_noevict.Namespace).Create(context.Background(), vm_noevict)
					Expect(err).ToNot(HaveOccurred())

					// Start VMs
					tests.StartVirtualMachine(vm_evict1)
					tests.StartVirtualMachine(vm_evict2)
					tests.StartVirtualMachine(vm_noevict)

					// Get VMIs
					vmi_evict1, err = virtClient.VirtualMachineInstance(vmi_evict1.Namespace).Get(context.Background(), vmi_evict1.Name, &metav1.GetOptions{})
					vmi_evict2, err = virtClient.VirtualMachineInstance(vmi_evict1.Namespace).Get(context.Background(), vmi_evict2.Name, &metav1.GetOptions{})
					vmi_noevict, err = virtClient.VirtualMachineInstance(vmi_evict1.Namespace).Get(context.Background(), vmi_noevict.Name, &metav1.GetOptions{})

					By("Verifying all VMIs are collcated on the same node")
					Expect(vmi_evict1.Status.NodeName).To(Equal(vmi_evict2.Status.NodeName))
					Expect(vmi_evict1.Status.NodeName).To(Equal(vmi_noevict.Status.NodeName))

					// Mark the control-plane nodes as schedulable so we can migrate there
					setControlPlaneSchedulability(true)

					node := vmi_evict1.Status.NodeName
					temporaryNodeDrain(node)

					By("Verify expected vmis migrated after node drain completes")
					// verify migrated where expected to migrate.
					Eventually(func() error {
						vmi, err := virtClient.VirtualMachineInstance(vmi_evict1.Namespace).Get(context.Background(), vmi_evict1.Name, &metav1.GetOptions{})
						if err != nil {
							return err
						} else if vmi.Status.NodeName == node {
							return fmt.Errorf("VMI still exist on the same node")
						} else if vmi.Status.MigrationState == nil || vmi.Status.MigrationState.SourceNode != node {
							return fmt.Errorf("VMI did not migrate yet")
						}

						vmi, err = virtClient.VirtualMachineInstance(vmi_evict2.Namespace).Get(context.Background(), vmi_evict2.Name, &metav1.GetOptions{})
						if err != nil {
							return err
						} else if vmi.Status.NodeName == node {
							return fmt.Errorf("VMI still exist on the same node")
						} else if vmi.Status.MigrationState == nil || vmi.Status.MigrationState.SourceNode != node {
							return fmt.Errorf("VMI did not migrate yet")
						}

						// This VMI should be terminated
						vmi, err = virtClient.VirtualMachineInstance(vmi_noevict.Namespace).Get(context.Background(), vmi_noevict.Name, &metav1.GetOptions{})
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
		Context("[Serial]with multiple VMIs with eviction policies set", Serial, func() {

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
					_, err := virtClient.VirtualMachineInstance(vmi.Namespace).Create(context.Background(), vmi)
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
					migrationList, err := virtClient.VirtualMachineInstanceMigration(k8sv1.NamespaceAll).List(&metav1.ListOptions{})
					Expect(err).ToNot(HaveOccurred())

					runningMigrations := filterRunningMigrations(migrationList.Items)

					return len(runningMigrations)
				}, 2*time.Minute, 1*time.Second).Should(BeNumerically(">", 0))

				By("checking that all VMIs were migrated, and we never see more than two running migrations in parallel")
				Eventually(func() []string {
					var nodes []string
					for _, vmi := range vmis {
						vmi, err = virtClient.VirtualMachineInstance(vmi.Namespace).Get(context.Background(), vmi.Name, &metav1.GetOptions{})
						nodes = append(nodes, vmi.Status.NodeName)
					}

					migrationList, err := virtClient.VirtualMachineInstanceMigration(k8sv1.NamespaceAll).List(&metav1.ListOptions{})
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
						newvmi, err := virtClient.VirtualMachineInstance(vmi.Namespace).Get(context.Background(), vmi.Name, &metav1.GetOptions{})
						Expect(err).ToNot(HaveOccurred(), "Should successfully get new VMI")
						vmiPod := tests.GetRunningPodByVirtualMachineInstance(newvmi, newvmi.Namespace)
						return libnet.ValidateVMIandPodIPMatch(newvmi, vmiPod)
					}, time.Minute, time.Second).Should(Succeed(), "Should match PodIP with latest VMI Status after migration")
				}
			})
		})
	})

	Describe("[Serial] with a cluster-wide live-migrate eviction strategy set", Serial, func() {
		var originalKV *v1.KubeVirt

		BeforeEach(func() {
			kv := util.GetCurrentKv(virtClient)
			originalKV = kv.DeepCopy()

			evictionStrategy := v1.EvictionStrategyLiveMigrate
			kv.Spec.Configuration.EvictionStrategy = &evictionStrategy
			tests.UpdateKubeVirtConfigValueAndWait(kv.Spec.Configuration)
		})

		AfterEach(func() {
			tests.UpdateKubeVirtConfigValueAndWait(originalKV.Spec.Configuration)
		})

		Context("with a VMI running", func() {
			Context("with no eviction strategy set", func() {
				It("[test_id:10155]should block the eviction api and migrate", func() {
					// no EvictionStrategy set
					vmi := tests.NewRandomVMIWithEphemeralDisk(cd.ContainerDiskFor(cd.ContainerDiskAlpine))
					vmi = tests.RunVMIAndExpectLaunch(vmi, 180)
					vmiNodeOrig := vmi.Status.NodeName
					pod := tests.GetRunningPodByVirtualMachineInstance(vmi, vmi.Namespace)
					err := virtClient.CoreV1().Pods(vmi.Namespace).EvictV1beta1(context.Background(), &policyv1beta1.Eviction{ObjectMeta: metav1.ObjectMeta{Name: pod.Name}})
					Expect(errors.IsTooManyRequests(err)).To(BeTrue())

					By("Ensuring the VMI has migrated and lives on another node")
					Eventually(func() error {
						vmi, err := virtClient.VirtualMachineInstance(vmi.Namespace).Get(context.Background(), vmi.Name, &metav1.GetOptions{})
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
					resVMI, err := virtClient.VirtualMachineInstance(vmi.Namespace).Get(context.Background(), vmi.Name, &metav1.GetOptions{})
					Expect(err).ShouldNot(HaveOccurred())
					Expect(resVMI.Status.EvacuationNodeName).To(Equal(""), "vmi evacuation state should be clean")
				})
			})

			Context("with eviction strategy set to 'None'", func() {
				It("[test_id:10156]The VMI should get evicted", func() {
					vmi := tests.NewRandomVMIWithEphemeralDisk(cd.ContainerDiskFor(cd.ContainerDiskAlpine))
					evictionStrategy := v1.EvictionStrategyNone
					vmi.Spec.EvictionStrategy = &evictionStrategy
					vmi = tests.RunVMIAndExpectLaunch(vmi, 180)
					pod := tests.GetRunningPodByVirtualMachineInstance(vmi, vmi.Namespace)
					err := virtClient.CoreV1().Pods(vmi.Namespace).EvictV1beta1(context.Background(), &policyv1beta1.Eviction{ObjectMeta: metav1.ObjectMeta{Name: pod.Name}})
					Expect(err).ToNot(HaveOccurred())
				})
			})
		})
	})
})

func fedoraVMIWithEvictionStrategy() *v1.VirtualMachineInstance {
	opts := append(libvmi.WithMasqueradeNetworking(),
		libvmi.WithResourceMemory(fedoraVMSize),
		libvmi.WithEvictionStrategy(v1.EvictionStrategyLiveMigrate),
		libvmi.WithNamespace(testsuite.GetTestNamespace(nil)),
	)
	return libvmi.NewFedora(opts...)
}

func alpineVMIWithEvictionStrategy() *v1.VirtualMachineInstance {
	opts := append(libvmi.WithMasqueradeNetworking(),
		libvmi.WithEvictionStrategy(v1.EvictionStrategyLiveMigrate),
		libvmi.WithNamespace(testsuite.GetTestNamespace(nil)),
	)
	return libvmi.NewAlpine(opts...)
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
