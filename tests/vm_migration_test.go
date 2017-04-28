package tests_test

import (
	"flag"

	"fmt"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	k8sv1 "k8s.io/client-go/pkg/api/v1"
	"k8s.io/client-go/pkg/labels"
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
	var migration *v1.Migration

	var TIMEOUT float64 = 10.0
	var POLLING_INTERVAL float64 = 0.1

	BeforeEach(func() {
		if len(tests.GetReadyNodes()) < 2 {
			Skip("To test migrations, at least two nodes need to be active")
		}
		sourceVM = tests.NewRandomVM()
		migration = tests.NewMigrationForVm(sourceVM)

		tests.MustCleanup()
	})

	Context("New Migration given", func() {

		It("Should fail if the VM does not exist", func() {
			err = restClient.Post().Resource("migrations").Namespace(k8sv1.NamespaceDefault).Body(migration).Do().Error()
			Expect(err).To(BeNil())
			Eventually(func() v1.MigrationPhase {
				r, err := restClient.Get().Resource("migrations").Namespace(k8sv1.NamespaceDefault).Name(migration.ObjectMeta.Name).Do().Get()
				Expect(err).ToNot(HaveOccurred())
				var m *v1.Migration = r.(*v1.Migration)
				return m.Status.Phase
			}, TIMEOUT, POLLING_INTERVAL).Should(Equal(v1.MigrationFailed))
		})

		It("Should go to MigrationScheduled state if the VM exists", func(done Done) {

			vm, err := restClient.Post().Resource("vms").Namespace(k8sv1.NamespaceDefault).Body(sourceVM).Do().Get()
			Expect(err).ToNot(HaveOccurred())
			tests.WaitForSuccessfulVMStart(vm)

			err = restClient.Post().Resource("migrations").Namespace(k8sv1.NamespaceDefault).Body(migration).Do().Error()
			Expect(err).ToNot(HaveOccurred())

			Eventually(func() v1.MigrationPhase {
				obj, err := restClient.Get().Resource("migrations").Namespace(k8sv1.NamespaceDefault).Name(migration.ObjectMeta.Name).Do().Get()
				Expect(err).ToNot(HaveOccurred())
				var m *v1.Migration = obj.(*v1.Migration)
				return m.Status.Phase
			}, TIMEOUT, POLLING_INTERVAL).Should(Equal(v1.MigrationScheduled))
			close(done)
		}, 30)

		It("Should migrate the VM", func(done Done) {

			// Create the VM
			obj, err := restClient.Post().Resource("vms").Namespace(k8sv1.NamespaceDefault).Body(sourceVM).Do().Get()
			Expect(err).ToNot(HaveOccurred())
			tests.WaitForSuccessfulVMStart(obj)

			obj, err = restClient.Get().Resource("vms").Namespace(k8sv1.NamespaceDefault).Name(obj.(*v1.VM).ObjectMeta.Name).Do().Get()
			Expect(err).ToNot(HaveOccurred())

			sourceNode := obj.(*v1.VM).Status.NodeName
			// Create the Migration
			err = restClient.Post().Resource("migrations").Namespace(k8sv1.NamespaceDefault).Body(migration).Do().Error()
			Expect(err).ToNot(HaveOccurred())

			selector, err := labels.Parse(fmt.Sprintf("%s in (%s)", v1.MigrationLabel, migration.GetObjectMeta().GetName()) +
				fmt.Sprintf(",%s in (%s)", v1.AppLabel, "migration"))
			Expect(err).ToNot(HaveOccurred())

			// Wait for the job
			Eventually(func() int {
				jobs, err := coreClient.CoreV1().Pods(k8sv1.NamespaceDefault).List(k8sv1.ListOptions{LabelSelector: selector.String()})
				Expect(err).ToNot(HaveOccurred())
				return len(jobs.Items)
			}, TIMEOUT*2, POLLING_INTERVAL).Should(Equal(1))

			// Wait for the successful completion of the job
			Eventually(func() k8sv1.PodPhase {
				jobs, err := coreClient.CoreV1().Pods(k8sv1.NamespaceDefault).List(k8sv1.ListOptions{LabelSelector: selector.String()})
				Expect(err).ToNot(HaveOccurred())
				return jobs.Items[0].Status.Phase
			}, TIMEOUT*2, POLLING_INTERVAL).Should(Equal(k8sv1.PodSucceeded))

			// Give the pod controller some time to update the VM after successful migrations
			Eventually(func() v1.VMPhase {
				obj, err := restClient.Get().Resource("vms").Namespace(k8sv1.NamespaceDefault).Name(obj.(*v1.VM).ObjectMeta.Name).Do().Get()
				Expect(err).ToNot(HaveOccurred())
				fetchedVM := obj.(*v1.VM)
				return fetchedVM.Status.Phase
			}, TIMEOUT, POLLING_INTERVAL).Should(Equal(v1.Running))

			obj, err = restClient.Get().Resource("vms").Namespace(k8sv1.NamespaceDefault).Name(obj.(*v1.VM).ObjectMeta.Name).Do().Get()
			Expect(err).ToNot(HaveOccurred())
			migratedVM := obj.(*v1.VM)
			Expect(migratedVM.Status.Phase).To(Equal(v1.Running))
			Expect(migratedVM.Status.NodeName).ToNot(Equal(sourceNode))

			close(done)
		}, 60)

		It("Should create a pod to execute VM migration", func(done Done) {
			// Create the VM
			vm, err := restClient.Post().Resource("vms").Namespace(k8sv1.NamespaceDefault).Body(sourceVM).Do().Get()
			Expect(err).ToNot(HaveOccurred())
			tests.WaitForSuccessfulVMStart(vm)

			// Create the Migration
			err = restClient.Post().Resource("migrations").Namespace(k8sv1.NamespaceDefault).Body(migration).Do().Error()
			Expect(err).ToNot(HaveOccurred())

			obj, err := restClient.Get().Resource("migrations").Namespace(k8sv1.NamespaceDefault).Name(migration.ObjectMeta.Name).Do().Get()
			Expect(err).ToNot(HaveOccurred())

			thisMigration := obj.(*v1.Migration)
			labelSelector, err := labels.Parse(v1.DomainLabel + "," + v1.AppLabel + "=migration" + "," + v1.MigrationUIDLabel + "=" + string(thisMigration.GetObjectMeta().GetUID()))
			Expect(err).ToNot(HaveOccurred())

			Eventually(func() int {
				pods, err := coreClient.CoreV1().Pods(k8sv1.NamespaceDefault).List(k8sv1.ListOptions{LabelSelector: labelSelector.String()})
				Expect(err).ToNot(HaveOccurred())
				return len(pods.Items)
			}, TIMEOUT, POLLING_INTERVAL).Should(Equal(1))
			close(done)
		}, 60)

		AfterEach(func() {
			tests.MustCleanup()
		})
	})
})
