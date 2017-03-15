package services_test

import (
	"flag"
	"github.com/facebookgo/inject"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/ghttp"
	"k8s.io/client-go/kubernetes"
	corev1 "k8s.io/client-go/pkg/api/v1"
	"k8s.io/client-go/pkg/util/uuid"
	"k8s.io/client-go/rest"
	"kubevirt.io/kubevirt/pkg/api/v1"
	"kubevirt.io/kubevirt/pkg/kubecli"
	. "kubevirt.io/kubevirt/pkg/virt-controller/services"
	"net/http"
)

var _ = Describe("VM", func() {

	var vmService VMService
	var server *ghttp.Server
	var restClient *rest.RESTClient

	BeforeEach(func() {
		var g inject.Graph

		flag.Parse()
		vmService = NewVMService()
		server = ghttp.NewServer()
		config := rest.Config{}
		config.Host = server.URL()
		clientSet, _ := kubernetes.NewForConfig(&config)
		templateService, _ := NewTemplateService("kubevirt/virt-launcher")
		restClient, _ = kubecli.GetRESTClientFromFlags(server.URL(), "")

		g.Provide(
			&inject.Object{Value: restClient},
			&inject.Object{Value: clientSet},
			&inject.Object{Value: vmService},
			&inject.Object{Value: templateService},
		)
		g.Populate()

	})
	Context("calling Setup Migration ", func() {
		It("should work", func() {

			vm := v1.NewMinimalVM("test-vm")
			var migration, expected_migration *v1.Migration
			migration = v1.NewMinimalMigration(vm.ObjectMeta.Name+"-migration", vm.ObjectMeta.Name)
			expected_migration = &v1.Migration{}
			*expected_migration = *migration
			expected_migration.Status.Phase = v1.MigrationInProgress

			vm.ObjectMeta.UID = "testUID"
			vm.ObjectMeta.SetUID(uuid.NewUUID())

			pod := corev1.Pod{}

			server.AppendHandlers(

				ghttp.CombineHandlers(
					ghttp.VerifyRequest("POST", "/api/v1/namespaces/default/pods"),
					ghttp.RespondWithJSONEncoded(http.StatusOK, pod),
				),
			)
			err := vmService.SetupMigration(migration, vm)
			Expect(err).ToNot(HaveOccurred())
			Expect(len(server.ReceivedRequests())).To(Equal(1))

		})
	})

	Context("calling StartVM Pod for a pod that does not exists", func() {
		It("should create the pod", func() {
			vm := v1.NewMinimalVM("test-vm")
			vm.ObjectMeta.UID = "testUID"
			pod := corev1.Pod{}

			server.AppendHandlers(
				ghttp.CombineHandlers(
					ghttp.VerifyRequest("GET", "/api/v1/namespaces/default/pods"),
					ghttp.RespondWithJSONEncoded(http.StatusOK, pod),
				),
				ghttp.CombineHandlers(
					ghttp.VerifyRequest("POST", "/api/v1/namespaces/default/pods"),
					ghttp.RespondWithJSONEncoded(http.StatusOK, pod),
				),
			)
			err := vmService.StartVMPod(vm)
			Expect(err).ToNot(HaveOccurred())
			Expect(len(server.ReceivedRequests())).To(Equal(2))
		})
	})

})
