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

package tests_test

import (
	"flag"

	"fmt"

	. "github.com/onsi/ginkgo"
	"github.com/onsi/ginkgo/extensions/table"
	. "github.com/onsi/gomega"
	k8sv1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"

	"kubevirt.io/kubevirt/pkg/api/v1"
	"kubevirt.io/kubevirt/pkg/kubecli"
	"kubevirt.io/kubevirt/tests"
)

var _ = Describe("VmMigration", func() {

	flag.Parse()

	virtClient, err := kubecli.GetKubevirtClient()
	tests.PanicOnError(err)

	var sourceVM *v1.VirtualMachine

	var TIMEOUT float64 = 60.0
	var POLLING_INTERVAL float64 = 0.1

	BeforeEach(func() {
		Skip("Migration Support is not supported at the moment.")
		if len(tests.GetReadyNodes()) < 2 {
			Skip("To test migrations, at least two nodes need to be active")
		}
		sourceVM = tests.NewRandomVM()

		tests.BeforeTestCleanup()
	})

	Context("New Migration given", func() {

		It("Should fail if the VM does not exist", func() {
			migration := tests.NewRandomMigrationForVm(sourceVM)
			err = virtClient.RestClient().Post().Resource("migrations").Namespace(tests.NamespaceTestDefault).Body(migration).Do().Error()
			Expect(err).To(BeNil())
			Eventually(func() v1.MigrationPhase {
				r, err := virtClient.RestClient().Get().Resource("migrations").Namespace(tests.NamespaceTestDefault).Name(migration.ObjectMeta.Name).Do().Get()
				Expect(err).ToNot(HaveOccurred())
				var m *v1.Migration = r.(*v1.Migration)
				return m.Status.Phase
			}, TIMEOUT, POLLING_INTERVAL).Should(Equal(v1.MigrationFailed))
		})

		It("Should go to MigrationRunning state if the VM exists", func(done Done) {
			vm, err := virtClient.RestClient().Post().Resource("virtualmachines").Namespace(tests.NamespaceTestDefault).Body(sourceVM).Do().Get()
			Expect(err).ToNot(HaveOccurred())
			tests.WaitForSuccessfulVMStart(vm)

			migration := tests.NewRandomMigrationForVm(sourceVM)
			err = virtClient.RestClient().Post().Resource("migrations").Namespace(tests.NamespaceTestDefault).Body(migration).Do().Error()
			Expect(err).ToNot(HaveOccurred())

			Eventually(func() v1.MigrationPhase {
				obj, err := virtClient.RestClient().Get().Resource("migrations").Namespace(tests.NamespaceTestDefault).Name(migration.ObjectMeta.Name).Do().Get()
				Expect(err).ToNot(HaveOccurred())
				var m *v1.Migration = obj.(*v1.Migration)
				return m.Status.Phase
			}, TIMEOUT, POLLING_INTERVAL).Should(Equal(v1.MigrationRunning))
			close(done)
		}, 30)

		It("Should respect and preserve pre-set node affinity on the VM", func(done Done) {
			// Prepare dummy affinity rule
			sourceVM.Spec.Affinity = &v1.Affinity{
				NodeAffinity: &k8sv1.NodeAffinity{
					RequiredDuringSchedulingIgnoredDuringExecution: &k8sv1.NodeSelector{
						NodeSelectorTerms: []k8sv1.NodeSelectorTerm{
							{
								MatchExpressions: []k8sv1.NodeSelectorRequirement{
									{
										Key:      "invalidtag",
										Values:   []string{"nothing"},
										Operator: k8sv1.NodeSelectorOpNotIn,
									},
								},
							},
						},
					},
				},
			}

			vm, err := virtClient.RestClient().Post().Resource("virtualmachines").Namespace(tests.NamespaceTestDefault).Body(sourceVM).Do().Get()
			Expect(err).ToNot(HaveOccurred())
			tests.WaitForSuccessfulVMStart(vm)

			migration := tests.NewRandomMigrationForVm(sourceVM)
			err = virtClient.RestClient().Post().Resource("migrations").Namespace(tests.NamespaceTestDefault).Body(migration).Do().Error()
			Expect(err).ToNot(HaveOccurred())

			Eventually(func() v1.MigrationPhase {
				obj, err := virtClient.RestClient().Get().Resource("migrations").Namespace(tests.NamespaceTestDefault).Name(migration.ObjectMeta.Name).Do().Get()
				Expect(err).ToNot(HaveOccurred())
				var m *v1.Migration = obj.(*v1.Migration)
				return m.Status.Phase
			}, 3*TIMEOUT, POLLING_INTERVAL).Should(Equal(v1.MigrationSucceeded))

			// Check Pod and VM affinity
			labelSelector, err := labels.Parse(v1.DomainLabel + "=" + sourceVM.ObjectMeta.Name + "," + v1.MigrationLabel + "=" + migration.ObjectMeta.Name)
			Expect(err).ToNot(HaveOccurred())

			pods, err := virtClient.CoreV1().Pods(tests.NamespaceTestDefault).List(metav1.ListOptions{LabelSelector: labelSelector.String()})
			Expect(err).ToNot(HaveOccurred())
			Expect(pods.Items[0].Spec.Affinity.NodeAffinity.RequiredDuringSchedulingIgnoredDuringExecution.NodeSelectorTerms[0].MatchExpressions[0]).To(BeEquivalentTo(sourceVM.Spec.Affinity.NodeAffinity.RequiredDuringSchedulingIgnoredDuringExecution.NodeSelectorTerms[0].MatchExpressions[0]))

			close(done)
		}, 90)

		Context("New Migration given", func() {
			table.DescribeTable("Should migrate the VM", func(namespace string, migrateCount int) {

				// Create the VM
				sourceVM = tests.NewRandomVMWithNS(namespace)
				obj, err := virtClient.RestClient().Post().Resource("virtualmachines").Namespace(namespace).Body(sourceVM).Do().Get()
				Expect(err).ToNot(HaveOccurred())
				tests.WaitForSuccessfulVMStart(obj)

				for x := 0; x < migrateCount; x++ {
					vmMeta := obj.(*v1.VirtualMachine).ObjectMeta
					obj, err = virtClient.RestClient().Get().Resource("virtualmachines").Namespace(vmMeta.Namespace).Name(vmMeta.Name).Do().Get()
					Expect(err).ToNot(HaveOccurred())

					sourceNode := obj.(*v1.VirtualMachine).Status.NodeName

					// Create the Migration
					migration := tests.NewRandomMigrationForVm(sourceVM)
					err = virtClient.RestClient().Post().Resource("migrations").Namespace(migration.GetObjectMeta().GetNamespace()).Body(migration).Do().Error()
					Expect(err).ToNot(HaveOccurred())

					selector, err := labels.Parse(fmt.Sprintf("%s in (%s)", v1.MigrationLabel, migration.GetObjectMeta().GetName()) +
						fmt.Sprintf(",%s in (%s)", v1.AppLabel, "migration"))
					Expect(err).ToNot(HaveOccurred())

					// Wait for the job
					Eventually(func() int {
						jobs, err := virtClient.CoreV1().Pods(migration.GetObjectMeta().GetNamespace()).List(metav1.ListOptions{LabelSelector: selector.String()})
						Expect(err).ToNot(HaveOccurred())
						return len(jobs.Items)
					}, TIMEOUT*2, POLLING_INTERVAL).Should(Equal(1))

					// Wait for the successful completion of the job
					Eventually(func() k8sv1.PodPhase {
						jobs, err := virtClient.CoreV1().Pods(migration.GetObjectMeta().GetNamespace()).List(metav1.ListOptions{LabelSelector: selector.String()})
						Expect(err).ToNot(HaveOccurred())
						return jobs.Items[0].Status.Phase
					}, TIMEOUT*2, POLLING_INTERVAL).Should(Equal(k8sv1.PodSucceeded))

					// Give the pod controller some time to update the VM after successful migrations
					Eventually(func() v1.VMPhase {
						obj, err := virtClient.RestClient().Get().Resource("virtualmachines").Namespace(obj.(*v1.VirtualMachine).ObjectMeta.Namespace).Name(obj.(*v1.VirtualMachine).ObjectMeta.Name).Do().Get()
						Expect(err).ToNot(HaveOccurred())
						fetchedVM := obj.(*v1.VirtualMachine)
						return fetchedVM.Status.Phase
					}, TIMEOUT, POLLING_INTERVAL).Should(Equal(v1.Running))

					obj, err = virtClient.RestClient().Get().Resource("virtualmachines").Namespace(obj.(*v1.VirtualMachine).ObjectMeta.Namespace).Name(obj.(*v1.VirtualMachine).ObjectMeta.Name).Do().Get()
					Expect(err).ToNot(HaveOccurred())
					migratedVM := obj.(*v1.VirtualMachine)
					Expect(migratedVM.Status.Phase).To(Equal(v1.Running))
					Expect(migratedVM.Status.NodeName).ToNot(Equal(sourceNode))
				}
			},
				table.Entry("three times in a row in namespace "+tests.NamespaceTestDefault, tests.NamespaceTestDefault, 3),
				table.Entry("once in namespace "+tests.NamespaceTestAlternative, tests.NamespaceTestAlternative, 1),
			)
		})

		It("Should create a pod to execute VM migration", func(done Done) {
			// Create the VM
			vm, err := virtClient.RestClient().Post().Resource("virtualmachines").Namespace(tests.NamespaceTestDefault).Body(sourceVM).Do().Get()
			Expect(err).ToNot(HaveOccurred())
			tests.WaitForSuccessfulVMStart(vm)

			// Create the Migration
			migration := tests.NewRandomMigrationForVm(sourceVM)
			err = virtClient.RestClient().Post().Resource("migrations").Namespace(tests.NamespaceTestDefault).Body(migration).Do().Error()
			Expect(err).ToNot(HaveOccurred())

			obj, err := virtClient.RestClient().Get().Resource("migrations").Namespace(tests.NamespaceTestDefault).Name(migration.ObjectMeta.Name).Do().Get()
			Expect(err).ToNot(HaveOccurred())

			thisMigration := obj.(*v1.Migration)
			labelSelector, err := labels.Parse(v1.DomainLabel + "," + v1.AppLabel + "=migration" + "," + v1.MigrationUIDLabel + "=" + string(thisMigration.GetObjectMeta().GetUID()))
			Expect(err).ToNot(HaveOccurred())

			Eventually(func() int {
				pods, err := virtClient.CoreV1().Pods(tests.NamespaceTestDefault).List(metav1.ListOptions{LabelSelector: labelSelector.String()})
				Expect(err).ToNot(HaveOccurred())
				return len(pods.Items)
			}, TIMEOUT, POLLING_INTERVAL).Should(Equal(1))
			close(done)
		}, 60)
	})
})
