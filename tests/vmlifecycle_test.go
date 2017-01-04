package tests_test

import (
	"flag"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"k8s.io/client-go/1.5/pkg/api"
	kubev1 "k8s.io/client-go/1.5/pkg/api/v1"
	"k8s.io/client-go/1.5/pkg/fields"
	"k8s.io/client-go/1.5/pkg/labels"
	"kubevirt.io/kubevirt/pkg/api/v1"
	"kubevirt.io/kubevirt/pkg/kubecli"
)

var _ = Describe("Vmlifecycle", func() {

	flag.Parse()
	coreClient, err := kubecli.Get()
	if err != nil {
		panic(err)
	}
	restClient, err := kubecli.GetRESTClient()
	if err != nil {
		panic(err)
	}

	// TODO: we need this cleanup code for every test file, make a helper for that
	cleanup := func() {
		// Remove all VMs
		err := restClient.Delete().Namespace(api.NamespaceDefault).Resource("vms").Do().Error()
		Expect(err).To(BeNil())

		// Remove VM pods
		Expect(err).To(BeNil())
		labelSelector, err := labels.Parse(v1.AppLabel + " in (virt-launcher)")
		Expect(err).To(BeNil())
		err = coreClient.Core().Pods(api.NamespaceDefault).DeleteCollection(nil, api.ListOptions{FieldSelector: fields.Everything(), LabelSelector: labelSelector})
		Expect(err).To(BeNil())
	}

	BeforeEach(func() {
		cleanup()
	})

	Context("New VM given", func() {
		vm := v1.NewMinimalVM("testvm")

		It("Should be accepted on POST", func() {
			// Create a VM
			err := restClient.Post().Resource("vms").Namespace(api.NamespaceDefault).Body(vm).Do().Error()
			Expect(err).To(BeNil())
		})

		It("Should start the VM on POST", func(done Done) {
			// Create a VM
			result := restClient.Post().Resource("vms").Namespace(api.NamespaceDefault).Body(vm).Do()
			obj, err := result.Get()
			Expect(result.Error()).To(BeNil())

			//TODO: make a helper for the watch and iterate ove object events by uid which takes a callback function
			// Create a watcher for events of this VM and  make sure we stop it at the end of the test
			// TODO: make a helper for that pattern
			uid := obj.(*v1.VM).GetObjectMeta().GetUID()
			eventWatcher, err := coreClient.Core().Events(api.NamespaceDefault).Watch(api.ListOptions{FieldSelector: fields.ParseSelectorOrDie("involvedObject.uid=" + string(uid))})
			Expect(err).To(BeNil())
			defer eventWatcher.Stop()

			for obj := range eventWatcher.ResultChan() {
				event := obj.Object.(*kubev1.Event)
				Expect(event.Type).NotTo(Equal("Warning"), "Received VM warning event")
				if event.Type == "Normal" && event.Reason == v1.Started.String() {
					close(done)
					return
				}
			}
		}, 10)

	})
	AfterEach(func() {
		cleanup()
	})
})
