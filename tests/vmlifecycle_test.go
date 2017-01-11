package tests_test

import (
	"flag"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"k8s.io/client-go/pkg/api"
	kubev1 "k8s.io/client-go/pkg/api/v1"
	"kubevirt.io/kubevirt/pkg/api/v1"
	"kubevirt.io/kubevirt/pkg/kubecli"
	"kubevirt.io/kubevirt/tests"
)

var _ = Describe("Vmlifecycle", func() {

	flag.Parse()

	restClient, err := kubecli.GetRESTClient()
	tests.PanicOnError(err)

	BeforeEach(func() {
		tests.MustCleanup()
	})

	Context("New VM given", func() {
		vm := v1.NewMinimalVM("testvm")

		It("Should be accepted on POST", func() {
			err := restClient.Post().Resource("vms").Namespace(api.NamespaceDefault).Body(vm).Do().Error()
			Expect(err).To(BeNil())
		})

		It("Should start the VM on POST", func(done Done) {
			result := restClient.Post().Resource("vms").Namespace(api.NamespaceDefault).Body(vm).Do()
			Expect(result.Error()).To(BeNil())
			obj, _ := result.Get()

			tests.NewObjectEventWatcher(obj, func(event *kubev1.Event) bool {
				Expect(event.Type).NotTo(Equal("Warning"), "Received VM warning event")
				if event.Type == "Normal" && event.Reason == v1.Started.String() {
					result = restClient.Get().Namespace(api.NamespaceDefault).
						Resource("vms").Name("testvm").Do()
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
			vm := v1.NewMinimalVM("testvm")
			vm.Spec.Domain.Devices.Interfaces[0].Source.Network = "nonexistent"

			It("Should retry starting the VM", func(done Done) {
				result := restClient.Post().Resource("vms").Namespace(api.NamespaceDefault).Body(vm).Do()
				Expect(result.Error()).To(BeNil())

				retryCount := 0
				obj, _ := result.Get()
				tests.NewObjectEventWatcher(obj, func(event *kubev1.Event) bool {
					if event.Type == "Warning" && event.Reason == v1.SyncFailed.String() {
						retryCount++
						if retryCount >= 3 {
							// Done, three retries is enough
							return true
						}
					}
					return false
				}).Watch()
				close(done)
			}, 10)

			PIt("Should stop retrying starting the VM on VM delete", func(done Done) {
				result := restClient.Post().Resource("vms").Namespace(api.NamespaceDefault).Body(vm).Do()
				Expect(result.Error()).To(BeNil())

				obj, _ := result.Get()
				tests.NewObjectEventWatcher(obj, func(event *kubev1.Event) bool {
					if event.Type == "Warning" && event.Reason == v1.SyncFailed.String() {
						return true
					}
					return false
				}).Watch()

				result = restClient.Delete().Resource("vms").Namespace(api.NamespaceDefault).Name(vm.GetObjectMeta().GetName()).Do()
				Expect(result.Error()).To(BeNil())

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
