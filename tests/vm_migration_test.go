/*
 * This file is part of the kubevirt project
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
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	k8sv1 "k8s.io/client-go/pkg/api/v1"

	"kubevirt.io/kubevirt/pkg/api/v1"
	"kubevirt.io/kubevirt/pkg/kubecli"
	"kubevirt.io/kubevirt/tests"
)

var _ = Describe("VmMigration", func() {

	flag.Parse()

	restClient, err := kubecli.GetRESTClient()
	tests.PanicOnError(err)
	coreClient, err := kubecli.Get()
	tests.PanicOnError(err)

	var sourceVM *v1.VM

	var TIMEOUT float64 = 10.0
	var POLLING_INTERVAL float64 = 0.1

	BeforeEach(func() {
		if len(tests.GetReadyNodes()) < 2 {
			Skip("To test migrations, at least two nodes need to be active")
		}
		sourceVM = tests.NewRandomVM()

		tests.BeforeTestCleanup()
	})

	Context("New Migration given", func() {

		It("Should fail if the VM does not exist", func() {
			migration := tests.NewRandomMigrationForVm(sourceVM)
			err = restClient.Post().Resource("migrations").Namespace(tests.NamespaceTestDefault).Body(migration).Do().Error()
			Expect(err).To(BeNil())
			Eventually(func() v1.MigrationPhase {
				r, err := restClient.Get().Resource("migrations").Namespace(tests.NamespaceTestDefault).Name(migration.ObjectMeta.Name).Do().Get()
				Expect(err).ToNot(HaveOccurred())
				var m *v1.Migration = r.(*v1.Migration)
				return m.Status.Phase
			}, TIMEOUT, POLLING_INTERVAL).Should(Equal(v1.MigrationFailed))
		})

		It("Should go to MigrationRunning state if the VM exists", func(done Done) {
			vm, err := restClient.Post().Resource("vms").Namespace(tests.NamespaceTestDefault).Body(sourceVM).Do().Get()
			Expect(err).ToNot(HaveOccurred())
			tests.WaitForSuccessfulVMStart(vm)

			migration := tests.NewRandomMigrationForVm(sourceVM)
			err = restClient.Post().Resource("migrations").Namespace(tests.NamespaceTestDefault).Body(migration).Do().Error()
			Expect(err).ToNot(HaveOccurred())

			Eventually(func() v1.MigrationPhase {
				obj, err := restClient.Get().Resource("migrations").Namespace(tests.NamespaceTestDefault).Name(migration.ObjectMeta.Name).Do().Get()
				Expect(err).ToNot(HaveOccurred())
				var m *v1.Migration = obj.(*v1.Migration)
				return m.Status.Phase
			}, TIMEOUT, POLLING_INTERVAL).Should(Equal(v1.MigrationRunning))
			close(done)
		}, 30)

		Context("New Migration given", func() {
			table.DescribeTable("Should migrate the VM", func(namespace string, migrateCount int) {

				// Create the VM
				sourceVM = tests.NewRandomVMWithNS(namespace)
				obj, err := restClient.Post().Resource("vms").Namespace(namespace).Body(sourceVM).Do().Get()
				Expect(err).ToNot(HaveOccurred())
				tests.WaitForSuccessfulVMStart(obj)

				for x := 0; x < migrateCount; x++ {
					vmMeta := obj.(*v1.VM).ObjectMeta
					obj, err = restClient.Get().Resource("vms").Namespace(vmMeta.Namespace).Name(vmMeta.Name).Do().Get()
					Expect(err).ToNot(HaveOccurred())

					sourceNode := obj.(*v1.VM).Status.NodeName

					// Create the Migration
					migration := tests.NewRandomMigrationForVm(sourceVM)
					err = restClient.Post().Resource("migrations").Namespace(migration.GetObjectMeta().GetNamespace()).Body(migration).Do().Error()
					Expect(err).ToNot(HaveOccurred())

					selector, err := labels.Parse(fmt.Sprintf("%s in (%s)", v1.MigrationLabel, migration.GetObjectMeta().GetName()) +
						fmt.Sprintf(",%s in (%s)", v1.AppLabel, "migration"))
					Expect(err).ToNot(HaveOccurred())

					// Wait for the job
					Eventually(func() int {
						jobs, err := coreClient.CoreV1().Pods(migration.GetObjectMeta().GetNamespace()).List(metav1.ListOptions{LabelSelector: selector.String()})
						Expect(err).ToNot(HaveOccurred())
						return len(jobs.Items)
					}, TIMEOUT*2, POLLING_INTERVAL).Should(Equal(1))

					// Wait for the successful completion of the job
					Eventually(func() k8sv1.PodPhase {
						jobs, err := coreClient.CoreV1().Pods(migration.GetObjectMeta().GetNamespace()).List(metav1.ListOptions{LabelSelector: selector.String()})
						Expect(err).ToNot(HaveOccurred())
						return jobs.Items[0].Status.Phase
					}, TIMEOUT*2, POLLING_INTERVAL).Should(Equal(k8sv1.PodSucceeded))

					// Give the pod controller some time to update the VM after successful migrations
					Eventually(func() v1.VMPhase {
						obj, err := restClient.Get().Resource("vms").Namespace(obj.(*v1.VM).ObjectMeta.Namespace).Name(obj.(*v1.VM).ObjectMeta.Name).Do().Get()
						Expect(err).ToNot(HaveOccurred())
						fetchedVM := obj.(*v1.VM)
						return fetchedVM.Status.Phase
					}, TIMEOUT, POLLING_INTERVAL).Should(Equal(v1.Running))

					obj, err = restClient.Get().Resource("vms").Namespace(obj.(*v1.VM).ObjectMeta.Namespace).Name(obj.(*v1.VM).ObjectMeta.Name).Do().Get()
					Expect(err).ToNot(HaveOccurred())
					migratedVM := obj.(*v1.VM)
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
			vm, err := restClient.Post().Resource("vms").Namespace(tests.NamespaceTestDefault).Body(sourceVM).Do().Get()
			Expect(err).ToNot(HaveOccurred())
			tests.WaitForSuccessfulVMStart(vm)

			// Create the Migration
			migration := tests.NewRandomMigrationForVm(sourceVM)
			err = restClient.Post().Resource("migrations").Namespace(tests.NamespaceTestDefault).Body(migration).Do().Error()
			Expect(err).ToNot(HaveOccurred())

			obj, err := restClient.Get().Resource("migrations").Namespace(tests.NamespaceTestDefault).Name(migration.ObjectMeta.Name).Do().Get()
			Expect(err).ToNot(HaveOccurred())

			thisMigration := obj.(*v1.Migration)
			labelSelector, err := labels.Parse(v1.DomainLabel + "," + v1.AppLabel + "=migration" + "," + v1.MigrationUIDLabel + "=" + string(thisMigration.GetObjectMeta().GetUID()))
			Expect(err).ToNot(HaveOccurred())

			Eventually(func() int {
				pods, err := coreClient.CoreV1().Pods(tests.NamespaceTestDefault).List(metav1.ListOptions{LabelSelector: labelSelector.String()})
				Expect(err).ToNot(HaveOccurred())
				return len(pods.Items)
			}, TIMEOUT, POLLING_INTERVAL).Should(Equal(1))
			close(done)
		}, 60)
	})
})
