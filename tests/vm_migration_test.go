package tests_test

import (
	"flag"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"k8s.io/client-go/pkg/api"
	"kubevirt.io/kubevirt/pkg/api/v1"
	"kubevirt.io/kubevirt/pkg/kubecli"
	"kubevirt.io/kubevirt/tests"
)

var _ = Describe("VmMigration", func() {

	flag.Parse()

	restClient, err := kubecli.GetRESTClient()
	tests.PanicOnError(err)
	var vm *v1.VM
	var migration *v1.Migration

	var TIMEOUT float64 = 10.0
	var POLLING_INTERVAL float64 = 0.1

	BeforeEach(func() {
		vm = tests.NewRandomVM()
		migration = tests.NewMigrationForVm(vm)

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

			vm, err := restClient.Post().Resource("vms").Namespace(api.NamespaceDefault).Body(vm).Do().Get()
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
			vm, err := restClient.Post().Resource("vms").Namespace(api.NamespaceDefault).Body(vm).Do().Get()
			Expect(err).ToNot(HaveOccurred())
			tests.WaitForSuccessfulVMStart(vm)

			// Create the Migration
			err = restClient.Post().Resource("migrations").Namespace(api.NamespaceDefault).Body(migration).Do().Error()
			Expect(err).ToNot(HaveOccurred())

			Eventually(func() string {
				obj, err := restClient.Get().Resource("vms").Namespace(api.NamespaceDefault).Name(vm.(*v1.VM).ObjectMeta.Name).Do().Get()
				Expect(err).ToNot(HaveOccurred())
				var m *v1.VM = obj.(*v1.VM)
				return m.Status.MigrationNodeName
			}, TIMEOUT, POLLING_INTERVAL).ShouldNot(BeEmpty())
			close(done)
		}, 30)

		AfterEach(func() {
			tests.MustCleanup()
		})
	})
})
