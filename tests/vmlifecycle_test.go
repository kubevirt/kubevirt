package tests_test

import (
	"flag"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"k8s.io/client-go/pkg/api"
	kubev1 "k8s.io/client-go/pkg/api/v1"
	metav1 "k8s.io/client-go/pkg/apis/meta/v1"
	"k8s.io/client-go/pkg/fields"
	"k8s.io/client-go/pkg/labels"
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

	coreCli, err := kubecli.Get()
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
			obj, err := restClient.Post().Resource("vms").Namespace(api.NamespaceDefault).Body(vm).Do().Get()
			Expect(err).To(BeNil())
			tests.WaitForSuccessfulVMStart(obj)

			close(done)
		}, 30)

		It("Should log libvirt start and stop lifecycle events of the domain", func(done Done) {
			// Get the pod name of virt-handler running on the master node to inspect its logs later on
			handlerNodeSelector := fields.ParseSelectorOrDie("spec.nodeName=master")
			labelSelector, err := labels.Parse("daemon in (virt-handler)")
			Expect(err).NotTo(HaveOccurred())
			pods, err := coreCli.CoreV1().Pods(api.NamespaceDefault).List(kubev1.ListOptions{FieldSelector: handlerNodeSelector.String(), LabelSelector: labelSelector.String()})
			Expect(err).NotTo(HaveOccurred())
			Expect(pods.Items).To(HaveLen(1))

			handlerName := pods.Items[0].GetObjectMeta().GetName()
			seconds := int64(30)
			logsQuery := coreCli.Pods(api.NamespaceDefault).GetLogs(handlerName, &kubev1.PodLogOptions{SinceSeconds: &seconds})

			// Make sure we schedule the VM to master
			vm.Spec.NodeSelector = map[string]string{"kubernetes.io/hostname": "master"}

			// Start the VM and wait for the confirmation of the start
			obj, err := restClient.Post().Resource("vms").Namespace(api.NamespaceDefault).Body(vm).Do().Get()
			Expect(err).ToNot(HaveOccurred())
			tests.WaitForSuccessfulVMStart(obj)

			// Check if the start event was logged
			Eventually(func() string {
				data, err := logsQuery.DoRaw()
				Expect(err).ToNot(HaveOccurred())
				return string(data)
			}, 5, 0.1).Should(MatchRegexp("(name=%s)[^\n]+(kind=Domain)[^\n]+(Domain is in state Running)", vm.GetObjectMeta().GetName()))

			// Delete the VM and wait for the confirmation of the delete
			_, err = restClient.Delete().Resource("vms").Namespace(api.NamespaceDefault).Name(vm.GetObjectMeta().GetName()).Do().Get()
			Expect(err).To(BeNil())
			tests.NewObjectEventWatcher(obj, func(event *kubev1.Event) bool {
				if event.Type == "Normal" && event.Reason == v1.Deleted.String() {
					return true
				}
				return false
			}).Watch()

			// Check if the stop event was logged
			Eventually(func() string {
				data, err := logsQuery.DoRaw()
				Expect(err).ToNot(HaveOccurred())
				return string(data)
			}, 5, 0.1).Should(MatchRegexp("(name=%s)[^\n]+(kind=Domain)[^\n]+(Domain deleted)", vm.GetObjectMeta().GetName()))

			close(done)
		}, 30)

		Context("New VM which can't be started", func() {

			It("Should retry starting the VM", func(done Done) {
				vm.Spec.Domain.Devices.Interfaces[0].Source.Network = "nonexistent"
				obj, err := restClient.Post().Resource("vms").Namespace(api.NamespaceDefault).Body(vm).Do().Get()
				Expect(err).To(BeNil())

				retryCount := 0
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
			}, 30)

			It("Should stop retrying invalid VM and go on to latest change request", func(done Done) {
				vm.Spec.Domain.Devices.Interfaces[0].Source.Network = "nonexistent"
				obj, err := restClient.Post().Resource("vms").Namespace(api.NamespaceDefault).Body(vm).Do().Get()
				Expect(err).To(BeNil())

				// Wait until we see that starting the VM is failing
				tests.NewObjectEventWatcher(obj, func(event *kubev1.Event) bool {
					if event.Type == "Warning" && event.Reason == v1.SyncFailed.String() && strings.Contains(event.Message, "nonexistent") {
						return true
					}
					return false
				}).Watch()

				_, err = restClient.Delete().Resource("vms").Namespace(api.NamespaceDefault).Name(vm.GetObjectMeta().GetName()).Do().Get()
				Expect(err).To(BeNil())

				// Check that the definition is deleted from the host
				tests.NewObjectEventWatcher(obj, func(event *kubev1.Event) bool {
					if event.Type == "Normal" && event.Reason == v1.Deleted.String() {
						return true
					}
					return false
				}).Watch()

				close(done)

			}, 30)
		})
	})
	AfterEach(func() {
		tests.MustCleanup()
	})
})
