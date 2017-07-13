package tests_test

import (
	"flag"
	"fmt"
	"reflect"
	"strings"
	"time"

	"github.com/jeevatkm/go-model"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	meta_v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/errors"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/pkg/api"
	kubev1 "k8s.io/client-go/pkg/api/v1"
	"k8s.io/client-go/pkg/apis/extensions/v1beta1"

	"kubevirt.io/kubevirt/pkg/api/v1"
	"kubevirt.io/kubevirt/pkg/kubecli"
	"kubevirt.io/kubevirt/tests"
)

var _ = Describe("Virt-Handler", func() {
	Context("Replace VM", func() {
		flag.Parse()

		var coreClient *kubernetes.Clientset

		var vm *v1.VM

		var virtHandler *v1beta1.DaemonSet
		var namespace string

		BeforeEach(func() {
			tests.MustCleanup()
			var err error
			namespace = api.NamespaceDefault

			coreClient, err = kubecli.Get()
			Expect(err).ToNot(HaveOccurred())

			vm = tests.NewRandomVM()

			virtHandler, err = getVirtHandler(coreClient, namespace)
			Expect(err).ToNot(HaveOccurred())
			Expect(virtHandler).ToNot(BeNil())
		})

		AfterEach(func() {
			ensureVirtHandlerIsRunning(coreClient, namespace, virtHandler)
		})

		It("should replace VM if virt-handler restarts", func() {
			// Start the VM and wait for the confirmation of the start
			restClient, err := kubecli.GetRESTClient()
			Expect(err).ToNot(HaveOccurred())

			var vmCopy = v1.VM{}
			errs := model.Copy(&vmCopy, vm)
			if errs != nil {
				Expect(errors.NewAggregate(errs)).ToNot(HaveOccurred())
			}

			obj, err := restClient.Post().Resource("vms").Namespace(namespace).Body(vm).Do().Get()
			Expect(err).ToNot(HaveOccurred())
			tests.WaitForSuccessfulVMStart(obj)

			uuid := obj.(*v1.VM).ObjectMeta.UID

			Expect(deleteVirtHandler(coreClient, namespace)).To(Succeed())

			// Delete VM
			// Create a new VM
			err = reCreateVM(&vmCopy)
			Expect(err).To(BeNil())

			vmRestarted := make(chan bool, 1)

			// Wait for the event log to show the VM was re-created
			go func() {
				defer GinkgoRecover()
				waitForSuccessfulVMRestart(coreClient, vm)
				vmRestarted <- true
			}()

			// Re-instantiate virt-handler daemonset
			err = createDaemonSet(coreClient, namespace, virtHandler)
			Expect(err).ToNot(HaveOccurred())

			// Wait for virt-handler pod to re-start
			Eventually(func() bool {
				running, err := isVirtHandlerPodRunning(coreClient, namespace, vm.Status.NodeName)
				Expect(err).ToNot(HaveOccurred())
				return running
			}, 120*time.Second).Should(Equal(true), "virt-handler is still not running after 120 seconds")

			// Ensure the event log shows that a VM restart took place
			<- vmRestarted

			// Verify that the VM's UID has changed
			obj, err = restClient.Get().Resource("vms").Namespace(namespace).Name(vm.ObjectMeta.Name).Do().Get()
			Expect(err).ToNot(HaveOccurred())

			Expect(obj.(*v1.VM).ObjectMeta.UID).ShouldNot(Equal(uuid))
		}, 120)
	})
})

func getVirtHandler(coreClient *kubernetes.Clientset, namespace string) (*v1beta1.DaemonSet, error) {
	betaRestClient := coreClient.ExtensionsV1beta1Client.RESTClient()
	obj, err := betaRestClient.Get().Resource("daemonsets").Namespace(namespace).Name("virt-handler").Param("export", "true").Do().Get()
	if err != nil {
		return nil, err
	}
	ds := obj.(*v1beta1.DaemonSet)
	return ds, nil
}

func getVirtHandlerPodLabelSelector() (labels.Selector, error) {
	return labels.Parse("daemon in (virt-handler)")
}

func deleteVirtHandlerPods(coreClient *kubernetes.Clientset, namespace string) error {
	labelSelector, err := getVirtHandlerPodLabelSelector()
	if err != nil {
		return err
	}
	err = coreClient.Core().Pods(api.NamespaceDefault).
		DeleteCollection(nil, meta_v1.ListOptions{LabelSelector: labelSelector.String()})
	if err != nil {
		return err
	}

	// Wait for virt-handler to really terminate
	Eventually(func() bool {
		running, err := isVirtHandlerRunning(coreClient, namespace, "")
		Expect(err).ToNot(HaveOccurred())
		return running
	}, 60*time.Second).Should(Equal(false), "Timed out waiting for virt-handler to stop")

	return nil
}

func deleteVirtHandler(coreClient *kubernetes.Clientset, namespace string) error {
	betaRestClient := coreClient.ExtensionsV1beta1Client.RESTClient()

	err := betaRestClient.Delete().Resource("daemonsets").Namespace(namespace).Name("virt-handler").Do().Error()
	if err != nil {
		return err
	}

	// Work around: https://github.com/kubernetes/kubernetes/issues/33517
	deleteVirtHandlerPods(coreClient, namespace)

	return nil
}

func reCreateVM(vm *v1.VM) error {
	restClient, err := kubecli.GetRESTClient()
	if err != nil {
		return err
	}
	namespace := vm.ObjectMeta.Namespace
	name := vm.ObjectMeta.Name

	Expect(string(vm.ObjectMeta.UID)).To(Equal(""))
	Expect(vm.ObjectMeta.ResourceVersion).To(Equal(""))

	err = restClient.Delete().Resource("vms").Namespace(namespace).Name(name).Do().Error()
	if err != nil {
		return err
	}

	err = restClient.Post().Resource("vms").Namespace(namespace).Body(vm).Do().Error()
	if err != nil {
		return err
	}
	return nil
}

func isVirtHandlerPodRunning(coreClient *kubernetes.Clientset, namespace string, nodeName string) (bool, error) {
	restClient := coreClient.CoreV1().RESTClient()
	labelSelector, err := getVirtHandlerPodLabelSelector()
	if err != nil {
		return false, err
	}

	obj, err := restClient.Get().Resource("pods").Namespace(namespace).LabelsSelectorParam(labelSelector).Do().Get()
	Expect(err).ToNot(HaveOccurred())

	podList := obj.(*kubev1.PodList)
	for _, pod := range podList.Items {
		if nodeName != "" {
			if (pod.Spec.NodeName == nodeName) && (pod.Status.Phase == kubev1.PodRunning) {
				return true, nil
			}
		} else {
			// If nodeName isn't specified, return true if any virt-handler pod is running
			// useful when checking to see if virt-handler is down
			if pod.Status.Phase == kubev1.PodRunning {
				return true, nil
			}
		}
	}
	return false, nil
}

func isVirtHandlerRunning(coreClient *kubernetes.Clientset, namespace string, nodeName string) (bool, error) {
	betaRestClient := coreClient.ExtensionsV1beta1Client.RESTClient()

	running, err := isVirtHandlerPodRunning(coreClient, namespace, nodeName)
	if err != nil {
		return false, err
	}
	if running {
		// Double check that virt-handler daemonset is not present
		obj, err := betaRestClient.Get().Resource("daemonsets").Namespace(namespace).Name("virt-handler").Do().Get()
		if (err == nil) || (obj != nil) {
			return false, fmt.Errorf("Unexpected result: virt-handler daemonset still exists")
		}
	}
	return running, nil
}

func createDaemonSet(coreClient *kubernetes.Clientset, namespace string, ds *v1beta1.DaemonSet) error {
	betaRestClient := coreClient.ExtensionsV1beta1Client.RESTClient()
	err := betaRestClient.Post().Resource("daemonsets").Namespace(namespace).Body(ds).Do().Error()
	if err != nil {
		return err
	}
	return nil
}

//This function is run during AfterEach.
func ensureVirtHandlerIsRunning(coreClient *kubernetes.Clientset, namespace string, ds *v1beta1.DaemonSet) {
	betaRestClient := coreClient.ExtensionsV1beta1Client.RESTClient()

	// check if DaemonSet exists
	// if not, start it.
	name := ds.ObjectMeta.Name
	err := betaRestClient.Get().Resource("daemonsets").Namespace(namespace).Name(name).Do().Error()
	if err != nil {
		if !(strings.Contains(err.Error(), "could not find") || (strings.Contains(err.Error(), "not found"))) {
			panic(fmt.Sprintf("Unable to verify status of virt-handler daemonset. Test suite cannot continue: %v", err))
		}
		// It's possible to delete a daemonset and not the pods due to:
		// https://github.com/kubernetes/kubernetes/issues/33517
		running, err := isVirtHandlerPodRunning(coreClient, namespace, "")
		if err != nil {
			if strings.Contains(err.Error(), "not found") {
				panic(fmt.Sprintf("Unable to verify status of virt-handler pods. Test suite cannot continue: %v", err))
			}
		}
		if running == true {
			deleteVirtHandlerPods(coreClient, namespace)
		}
		tests.PanicOnError(betaRestClient.Post().Resource("daemonsets").Namespace(namespace).Body(ds).Do().Error())
	}
}

func waitForSuccessfulVMRestart(coreClient *kubernetes.Clientset, vm runtime.Object) (nodeName string) {
	_, ok := vm.(*v1.VM)
	vmName := vm.(*v1.VM).ObjectMeta.Name
	Expect(ok).To(BeTrue(), "Object is not of type *v1.VM")
	restClient, err := kubecli.GetRESTClient()
	Expect(err).ToNot(HaveOccurred())

	w := tests.NewObjectEventWatcher(vm).SinceNow().FailOnWarnings()
	func() {
		stopped := false
		started := false
		// watch for both started and deleted events. don't return until both are seen.
		w.Watch(func(event *kubev1.Event) bool {
			if event.Type == string(tests.NormalEvent) && event.Reason == reflect.ValueOf(v1.Started).String() {
				started = true
			}
			if event.Type == string(tests.NormalEvent) && event.Reason == reflect.ValueOf(v1.Deleted).String() {
				stopped = true
			}
			return started && stopped
		})
	}()

	Eventually(func() v1.VMPhase {
		obj, err := restClient.Get().Resource("vms").Namespace(api.NamespaceDefault).Name(vmName).Do().Get()
		Expect(err).ToNot(HaveOccurred())
		fetchedVM := obj.(*v1.VM)
		nodeName = fetchedVM.Status.NodeName
		return fetchedVM.Status.Phase
	}, 60*time.Second).Should(Equal(v1.Running), "timed out waiting for VM to re-start")
	return
}
