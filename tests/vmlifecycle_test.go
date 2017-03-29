package tests_test

import (
	"flag"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"k8s.io/client-go/kubernetes"
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
	"time"
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
			tests.NewObjectEventWatcher(obj).WaitFor(tests.NormalEvent, v1.Deleted)

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
				tests.NewObjectEventWatcher(obj).Watch(func(event *kubev1.Event) bool {
					if event.Type == "Warning" && event.Reason == v1.SyncFailed.String() {
						retryCount++
						if retryCount >= 2 {
							// Done, two retries is enough
							return true
						}
					}
					return false
				})
				close(done)
			}, 30)

			It("Should stop retrying invalid VM and go on to latest change request", func(done Done) {
				vm.Spec.Domain.Devices.Interfaces[0].Source.Network = "nonexistent"
				obj, err := restClient.Post().Resource("vms").Namespace(api.NamespaceDefault).Body(vm).Do().Get()
				Expect(err).To(BeNil())

				// Wait until we see that starting the VM is failing
				event := tests.NewObjectEventWatcher(obj).WaitFor(tests.WarningEvent, v1.SyncFailed)
				Expect(event.Message).To(ContainSubstring("nonexistent"))

				_, err = restClient.Delete().Resource("vms").Namespace(api.NamespaceDefault).Name(vm.GetObjectMeta().GetName()).Do().Get()
				Expect(err).To(BeNil())

				// Check that the definition is deleted from the host
				tests.NewObjectEventWatcher(obj).WaitFor(tests.NormalEvent, v1.Deleted)

				close(done)

			}, 30)
		})

		Context("New VM that will be killed", func() {
			It("Should be in Failed phase", func(done Done) {
				obj, err := restClient.Post().Resource("vms").Namespace(api.NamespaceDefault).Body(vm).Do().Get()
				Expect(err).To(BeNil())

				tests.WaitForSuccessfulVMStart(obj)
				_, ok := obj.(*v1.VM)
				Expect(ok).To(BeTrue(), "Object is not of type *v1.VM")
				restClient, err := kubecli.GetRESTClient()
				Expect(err).ToNot(HaveOccurred())

				err = pkillAllVms(coreCli)
				Expect(err).To(BeNil())

				Eventually(func() v1.VMPhase {
					object, err := restClient.Get().Resource("vms").Namespace(api.NamespaceDefault).Name(obj.(*v1.VM).ObjectMeta.Name).Do().Get()
					Expect(err).ToNot(HaveOccurred())
					fetchedVM := object.(*v1.VM)
					return fetchedVM.Status.Phase
				}, "10s", "1s").Should(Equal(v1.Failed))

				close(done)
			}, 50)
			It("should be left alone by virt-handler", func(done Done) {
				obj, err := restClient.Post().Resource("vms").Namespace(api.NamespaceDefault).Body(vm).Do().Get()
				Expect(err).To(BeNil())

				tests.WaitForSuccessfulVMStart(obj)
				_, ok := obj.(*v1.VM)
				Expect(ok).To(BeTrue(), "Object is not of type *v1.VM")
				Expect(err).ToNot(HaveOccurred())

				err = pkillAllVms(coreCli)
				Expect(err).To(BeNil())

				triedToSync := false
				tests.NewObjectEventWatcher(obj).Timeout(15*time.Second).WaitFor(tests.WarningEvent, v1.SyncFailed)
				Expect(triedToSync).To(BeFalse(), "virt-handler tried to sync on a VM in final state")

				close(done)
			}, 50)
		})
	})
	AfterEach(func() {
		tests.MustCleanup()
	})
})

func renderPkillAllVmsJob() *kubev1.Pod {
	job := kubev1.Pod{
		ObjectMeta: kubev1.ObjectMeta{
			GenerateName: "vm-killer",
			Labels: map[string]string{
				v1.AppLabel: "test",
			},
		},
		Spec: kubev1.PodSpec{
			RestartPolicy: kubev1.RestartPolicyNever,
			Containers: []kubev1.Container{
				{
					Name:  "vm-killer",
					Image: "kubevirt/vm-killer:devel",
					Command: []string{
						"/vm-killer.sh",
					},
				},
			},
			HostPID: true,
			SecurityContext: &kubev1.PodSecurityContext{
				RunAsUser: new(int64),
			},
		},
	}

	return &job
}

func pkillAllVms(core *kubernetes.Clientset) error {
	job := renderPkillAllVmsJob()
	_, err := core.Pods(kubev1.NamespaceDefault).Create(job)

	return err
}
