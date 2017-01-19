package tests_test

import (
	"flag"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"k8s.io/client-go/pkg/api"
	kubev1 "k8s.io/client-go/pkg/api/v1"
	metav1 "k8s.io/client-go/pkg/apis/meta/v1"
	"k8s.io/client-go/pkg/util/json"
	"kubevirt.io/kubevirt/pkg/api/v1"
	"kubevirt.io/kubevirt/pkg/kubecli"
	"kubevirt.io/kubevirt/tests"
	"net/http"
	"strings"
)

var _ = Describe("Vmlifecycle", func() {

	flag.Parse()

	restClient, err := kubecli.GetRESTClient()
	tests.PanicOnError(err)
	var vm *v1.VM

	BeforeEach(func() {
		vm = tests.NewRandomVM()
		tests.MustCleanup()
	})

	Context("New VM given", func() {

		It("Should be accepted on POST", func() {
			err := restClient.Post().Resource("vms").Namespace(api.NamespaceDefault).Body(vm).Do().Error()
			Expect(err).To(BeNil())
		})

		It("Should reject posting the same VM a second time", func() {
			err := restClient.Post().Resource("vms").Namespace(api.NamespaceDefault).Body(vm).Do().Error()
			Expect(err).To(BeNil())
			b, err := restClient.Post().Resource("vms").Namespace(api.NamespaceDefault).Body(vm).DoRaw()
			Expect(err).ToNot(BeNil())
			status := metav1.Status{}
			err = json.Unmarshal(b, &status)
			Expect(err).To(BeNil())
			Expect(status.Code).To(Equal(int32(http.StatusConflict)))
		})

		It("Should return 404 if VM does not exist", func() {
			b, err := restClient.Get().Resource("vms").Namespace(api.NamespaceDefault).Name("nonexistnt").DoRaw()
			Expect(err).ToNot(BeNil())
			status := metav1.Status{}
			err = json.Unmarshal(b, &status)
			Expect(err).To(BeNil())
			Expect(status.Code).To(Equal(int32(http.StatusNotFound)))
		})

		It("Should start the VM on POST", func(done Done) {
			result := restClient.Post().Resource("vms").Namespace(api.NamespaceDefault).Body(vm).Do()
			Expect(result.Error()).To(BeNil())
			obj, _ := result.Get()

			tests.NewObjectEventWatcher(obj, func(event *kubev1.Event) bool {
				Expect(event.Type).NotTo(Equal("Warning"), "Received VM warning event")
				if event.Type == "Normal" && event.Reason == v1.Started.String() {
					result = restClient.Get().Namespace(api.NamespaceDefault).
						Resource("vms").Name(vm.GetObjectMeta().GetName()).Do()
					obj, err := result.Get()
					Expect(err).To(BeNil())
					Expect(string(obj.(*v1.VM).Status.Phase)).To(Equal(string(v1.Running)))
					close(done)
					return true
				}
				return false
			}).Watch()
		}, 10)

		Context("New VM which can't be started", func() {

			It("Should retry starting the VM", func(done Done) {
				vm.Spec.Domain.Devices.Interfaces[0].Source.Network = "nonexistent"
				result := restClient.Post().Resource("vms").Namespace(api.NamespaceDefault).Body(vm).Do()
				Expect(result.Error()).To(BeNil())

				retryCount := 0
				obj, _ := result.Get()
				tests.NewObjectEventWatcher(obj, func(event *kubev1.Event) bool {
					if event.Type == "Warning" && event.Reason == v1.SyncFailed.String() {
						retryCount++
						if retryCount >= 2 {
							// Done, two retries is enough
							return true
						}
					}
					return false
				}).Watch()
				close(done)
			}, 10)

			It("Should stop retrying invalid VM and go on to latest change request", func(done Done) {
				vm.Spec.Domain.Devices.Interfaces[0].Source.Network = "nonexistent"
				result := restClient.Post().Resource("vms").Namespace(api.NamespaceDefault).Body(vm).Do()
				Expect(result.Error()).To(BeNil())

				// Wait until we see that starting the VM is failing
				obj, _ := result.Get()
				tests.NewObjectEventWatcher(obj, func(event *kubev1.Event) bool {
					if event.Type == "Warning" && event.Reason == v1.SyncFailed.String() && strings.Contains(event.Message, "nonexistent") {
						return true
					}
					return false
				}).Watch()

				result = restClient.Delete().Resource("vms").Namespace(api.NamespaceDefault).Name(vm.GetObjectMeta().GetName()).Do()
				Expect(result.Error()).To(BeNil())

				// Check that the definition is deleted from the host
				tests.NewObjectEventWatcher(obj, func(event *kubev1.Event) bool {
					if event.Type == "Normal" && event.Reason == v1.Deleted.String() {
						return true
					}
					return false
				}).Watch()

				close(done)

			}, 10)
		})
	})
	AfterEach(func() {
		tests.MustCleanup()
	})
})
