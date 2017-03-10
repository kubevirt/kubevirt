package tests_test

import (
	"flag"

	"fmt"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"k8s.io/client-go/pkg/api"
	batchv1 "k8s.io/client-go/pkg/apis/batch/v1"
	metav1 "k8s.io/client-go/pkg/apis/meta/v1"
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
			err = restClient.Post().Resource("migrations").Namespace(api.NamespaceDefault).Body(migration).Do().Error()
			Expect(err).To(BeNil())
			Eventually(func() v1.MigrationPhase {
				r, err := restClient.Get().Resource("migrations").Namespace(api.NamespaceDefault).Name(migration.ObjectMeta.Name).Do().Get()
				Expect(err).ToNot(HaveOccurred())
				var m *v1.Migration = r.(*v1.Migration)
				return m.Status.Phase
			}, TIMEOUT, POLLING_INTERVAL).Should(Equal(v1.MigrationFailed))
		})

		It("Should go to MigrationInProgress state if the VM exists", func(done Done) {

			vm, err := restClient.Post().Resource("vms").Namespace(api.NamespaceDefault).Body(sourceVM).Do().Get()
			Expect(err).ToNot(HaveOccurred())
			tests.WaitForSuccessfulVMStart(vm)

			err = restClient.Post().Resource("migrations").Namespace(api.NamespaceDefault).Body(migration).Do().Error()
			Expect(err).ToNot(HaveOccurred())

			Eventually(func() v1.MigrationPhase {
				obj, err := restClient.Get().Resource("migrations").Namespace(api.NamespaceDefault).Name(migration.ObjectMeta.Name).Do().Get()
				Expect(err).ToNot(HaveOccurred())
				var m *v1.Migration = obj.(*v1.Migration)
				return m.Status.Phase
			}, TIMEOUT, POLLING_INTERVAL).Should(Equal(v1.MigrationInProgress))
			close(done)
		}, 30)

		It("Should update the Status.MigrationNodeName after the migration target pod was started", func(done Done) {

			// Create the VM
			obj, err := restClient.Post().Resource("vms").Namespace(api.NamespaceDefault).Body(sourceVM).Do().Get()
			Expect(err).ToNot(HaveOccurred())
			tests.WaitForSuccessfulVMStart(obj)

			// Create the Migration
			err = restClient.Post().Resource("migrations").Namespace(api.NamespaceDefault).Body(migration).Do().Error()
			Expect(err).ToNot(HaveOccurred())

			// TODO we need events
			var fetchedVM *v1.VM
			Eventually(func() string {
				obj, err := restClient.Get().Resource("vms").Namespace(api.NamespaceDefault).Name(obj.(*v1.VM).ObjectMeta.Name).Do().Get()
				Expect(err).ToNot(HaveOccurred())
				fetchedVM = obj.(*v1.VM)
				return fetchedVM.Status.MigrationNodeName
			}, TIMEOUT, POLLING_INTERVAL).ShouldNot(BeEmpty())
			Expect(fetchedVM.Status.Phase).To(Equal(v1.Migrating))

			close(done)
		}, 30)

		It("Should migrate the VM", func(done Done) {

			// Create the VM
			obj, err := restClient.Post().Resource("vms").Namespace(api.NamespaceDefault).Body(sourceVM).Do().Get()
			Expect(err).ToNot(HaveOccurred())
			tests.WaitForSuccessfulVMStart(obj)

			// Create the Migration
			err = restClient.Post().Resource("migrations").Namespace(api.NamespaceDefault).Body(migration).Do().Error()
			Expect(err).ToNot(HaveOccurred())

			selector, err := labels.Parse(fmt.Sprintf("%s in (%s)", v1.DomainLabel, sourceVM.GetObjectMeta().GetName()))
			jobsGetter := coreClient.CoreV1().RESTClient().Get().AbsPath("/apis/batch/v1/namespaces/default/jobs").LabelsSelectorParam(selector)
			Expect(err).ToNot(HaveOccurred())

			// Wait for the job
			Eventually(func() int {
				jobs, err := jobsGetter.Do().Get()
				Expect(err).ToNot(HaveOccurred())
				return len(jobs.(*batchv1.JobList).Items)
			}, TIMEOUT, POLLING_INTERVAL).Should(Equal(1))

			// Wait for the successful completion of the job
			Eventually(func() *metav1.Time {
				jobs, err := jobsGetter.Do().Get()
				Expect(err).ToNot(HaveOccurred())
				Expect(err).ToNot(HaveOccurred())
				return jobs.(*batchv1.JobList).Items[0].Status.CompletionTime
			}, TIMEOUT, POLLING_INTERVAL).ShouldNot(BeNil())

			close(done)
		}, 30)

		AfterEach(func() {
			tests.MustCleanup()
		})
	})
})
