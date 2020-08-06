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
 * Copyright 2018 Red Hat, Inc.
 *
 */

package tests_test

import (
	"fmt"
	"path/filepath"
	"time"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/pborman/uuid"
	k8sv1 "k8s.io/api/core/v1"
	"k8s.io/api/policy/v1beta1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	v1 "kubevirt.io/client-go/api/v1"
	"kubevirt.io/client-go/kubecli"
	virtconfig "kubevirt.io/kubevirt/pkg/virt-config"
	"kubevirt.io/kubevirt/tests"
	"kubevirt.io/kubevirt/tests/flags"
)

var _ = FDescribe("Launcher eviction", func() {

	var virtClient kubecli.KubevirtClient
	var err error

	tests.BeforeAll(func() {

		virtClient, err = kubecli.GetKubevirtClient()
		tests.PanicOnError(err)

		_, err := virtClient.
			CoreV1().
			ConfigMaps(flags.KubeVirtInstallNamespace).
			Get(virtconfig.ConfigMapName, metav1.GetOptions{})

		if err != nil && !errors.IsNotFound(err) {
			Expect(err).ToNot(HaveOccurred())
		}

		if errors.IsNotFound(err) {
			// create an empty kubevirt-config configmap if none exists.
			cfgMap := &k8sv1.ConfigMap{
				ObjectMeta: metav1.ObjectMeta{Name: virtconfig.ConfigMapName},
				Data: map[string]string{
					"feature-gates": "",
				},
			}

			_, err = virtClient.CoreV1().ConfigMaps(flags.KubeVirtInstallNamespace).Create(cfgMap)
			if err != nil {
				Expect(err).ToNot(HaveOccurred())
			}

			tests.EnableFeatureGate(virtconfig.LiveMigrationGate)
		}
	})

	BeforeEach(func() {
		tests.BeforeTestCleanup()
		if !tests.HasLiveMigration() {
			Skip("LiveMigration feature gate is not enabled in kubevirt-config")
		}

		nodes := tests.GetAllSchedulableNodes(virtClient)
		Expect(nodes.Items).ToNot(BeEmpty(), "There should be some compute node")

		if len(nodes.Items) < 2 {
			Skip("Launcher eviction tests require at least 2 nodes")
		}
	})

	runVMIAndExpectLaunch := func(vmi *v1.VirtualMachineInstance, timeout int) *v1.VirtualMachineInstance {
		return tests.RunVMIAndExpectLaunchWithIgnoreWarningArg(vmi, timeout, false)
	}

	Context("Evicted virt-launcher pod", func() {

		It("Should migrate the VMI", func() {
			vmi := cirrosVMIWithEvictionStrategy()
			runVMIAndExpectLaunch(vmi, 180)
			pod := tests.GetRunningPodByVirtualMachineInstance(vmi, vmi.Namespace)
			err := virtClient.CoreV1().Pods(pod.Namespace).Evict(&v1beta1.Eviction{ObjectMeta: metav1.ObjectMeta{Name: pod.Name}})
			Expect(err).Should(HaveOccurred())
			Expect(err.Error()).To(Equal(`admission webhook "virt-launcher-eviction-interceptor.kubevirt.io" denied the request: virt-launcher eviction will be handled by KubeVirt`))

			Eventually(func() error {
				migrations, err := virtClient.VirtualMachineInstanceMigration(pod.Namespace).List(&metav1.ListOptions{
					LabelSelector: fmt.Sprintf("%s=%s", v1.MigrationForEvictedPodLabel, pod.Name),
				})
				if err != nil {
					return err
				}
				for _, migration := range migrations.Items {
					Expect(migration.Status.Phase).To(Equal(v1.Succeeded))
				}
				return nil
			}, 180*time.Second, 1*time.Second).ShouldNot(HaveOccurred())
		})

		Context("Not migratable VMI", func() {

			BeforeEach(func() {
				tests.EnableFeatureGate(virtconfig.HostDiskGate)
			})

			AfterEach(func() {
				tests.DisableFeatureGate(virtconfig.HostDiskGate)
			})

			It("Should shut down the VM", func() {
				By("Finding a worker node to bind the VM to")
				nodes, err := virtClient.CoreV1().Nodes().List(metav1.ListOptions{
					LabelSelector: fmt.Sprintf("%s=%s", "node-role.kubernetes.io/worker", ""),
				})
				Expect(err).ShouldNot(HaveOccurred())
				Expect(len(nodes.Items)).Should(BeNumerically(">=", 1))

				By("Creating a non-migratable VM")
				diskName := "disk-" + uuid.NewRandom().String() + ".img"
				diskPath := filepath.Join(tests.RandTmpDir(), diskName)
				template := tests.NewRandomVMIWithHostDisk(diskPath, v1.HostDiskExistsOrCreate, nodes.Items[0].GetName())
				vm := tests.NewRandomVirtualMachine(template, false)
				_, err = virtClient.VirtualMachine(tests.NamespaceTestDefault).Create(vm)
				Expect(err).ShouldNot(HaveOccurred())
				Expect(virtClient.VirtualMachine(tests.NamespaceTestDefault).Start(vm.Name)).Should(Succeed())

				By("Waiting for VMI to start")
				EventuallyWithOffset(5, func() bool {
					vmi, err := virtClient.VirtualMachineInstance(tests.NamespaceTestDefault).Get(vm.Name, &metav1.GetOptions{})
					if err != nil {
						if errors.IsNotFound(err) {
							return false
						}
						Fail("Failed getting VMI")
					}
					return vmi.Status.Phase == v1.Running
				}, 180*time.Second, 1*time.Second).Should(BeTrue())

				By("Finding virt-launcher for the running VM")
				vmi, err := virtClient.VirtualMachineInstance(tests.NamespaceTestDefault).Get(vm.Name, &metav1.GetOptions{})
				Expect(err).ShouldNot(HaveOccurred())
				pod := tests.GetRunningPodByVirtualMachineInstance(vmi, tests.NamespaceTestDefault)

				By("Evicting virt-launcher")
				err = virtClient.CoreV1().Pods(pod.Namespace).Evict(&v1beta1.Eviction{ObjectMeta: metav1.ObjectMeta{Name: pod.Name}})
				Expect(err).Should(HaveOccurred())
				Expect(err.Error()).To(Equal(`admission webhook "virt-launcher-eviction-interceptor.kubevirt.io" denied the request: virt-launcher eviction will be handled by KubeVirt`))

				By("Expecting the VM to stop")
				Eventually(func() bool {
					targetVm, err := virtClient.VirtualMachine(tests.NamespaceTestDefault).Get(vm.Name, &metav1.GetOptions{})
					Expect(err).ShouldNot(HaveOccurred())
					running := targetVm.Spec.Running
					return *running
				}, 180*time.Second, 1*time.Second).Should(BeFalse())
			})
		})

	})
})
