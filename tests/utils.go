package tests

import (
	"fmt"
	. "github.com/onsi/gomega"
	"k8s.io/client-go/pkg/api"
	"k8s.io/client-go/pkg/api/meta"
	kubev1 "k8s.io/client-go/pkg/api/v1"
	"k8s.io/client-go/pkg/fields"
	"k8s.io/client-go/pkg/labels"
	"k8s.io/client-go/pkg/runtime"
	"k8s.io/client-go/pkg/util/rand"
	"kubevirt.io/kubevirt/pkg/api/v1"
	"kubevirt.io/kubevirt/pkg/kubecli"
)

type ProcessFunc func(event *kubev1.Event) (done bool)

type ObjectEventWatcher struct {
	object  runtime.Object
	process ProcessFunc
}

func NewObjectEventWatcher(object runtime.Object, process ProcessFunc) *ObjectEventWatcher {
	return &ObjectEventWatcher{object: object, process: process}
}

func (w *ObjectEventWatcher) Watch() {
	cli, err := kubecli.Get()
	if err != nil {
		panic(err)
	}

	uid := w.object.(meta.ObjectMetaAccessor).GetObjectMeta().GetName()
	eventWatcher, err := cli.Core().Events(api.NamespaceDefault).
		Watch(kubev1.ListOptions{FieldSelector: fields.ParseSelectorOrDie("involvedObject.name=" + string(uid)).String()})
	if err != nil {
		panic(err)
	}
	defer eventWatcher.Stop()

	for obj := range eventWatcher.ResultChan() {
		if w.process(obj.Object.(*kubev1.Event)) {
			break
		}
	}
}

func MustCleanup() {
	coreClient, err := kubecli.Get()
	PanicOnError(err)
	restClient, err := kubecli.GetRESTClient()
	PanicOnError(err)

	// Remove all Migrations
	PanicOnError(restClient.Delete().Namespace(api.NamespaceDefault).Resource("migrations").Do().Error())

	// Remove all VMs
	PanicOnError(restClient.Delete().Namespace(api.NamespaceDefault).Resource("vms").Do().Error())

	// Remove all Jobs
	PanicOnError(coreClient.CoreV1().RESTClient().Delete().AbsPath("/apis/batch/v1/namespaces/default/jobs").Do().Error())

	// Remove all pods associated with a job
	jobPodlabelSelector, err := labels.Parse("job-name")
	PanicOnError(err)
	err = coreClient.Core().Pods(api.NamespaceDefault).
		DeleteCollection(nil, kubev1.ListOptions{FieldSelector: fields.Everything().String(), LabelSelector: jobPodlabelSelector.String()})

	PanicOnError(err)
	// Remove VM pods
	vmPodlabelSelector, err := labels.Parse(v1.AppLabel + " in (virt-launcher)")
	PanicOnError(err)
	err = coreClient.Core().Pods(api.NamespaceDefault).
		DeleteCollection(nil, kubev1.ListOptions{FieldSelector: fields.Everything().String(), LabelSelector: vmPodlabelSelector.String()})

	PanicOnError(err)
}

func PanicOnError(err error) {
	if err != nil {
		panic(err)
	}
}

func NewRandomVM() *v1.VM {
	return v1.NewMinimalVM("testvm" + rand.String(5))
}

func NewRandomVMWithLun(lun int) *v1.VM {
	vm := NewRandomVM()
	vm.Spec.Domain.Memory.Unit = "MB"
	vm.Spec.Domain.Memory.Value = 64
	vm.Spec.Domain.Devices.Disks = []v1.Disk{{
		Type:     "network",
		Snapshot: "external",
		Device:   "disk",
		Driver: &v1.DiskDriver{
			Name: "qemu",
			Type: "raw",
		},
		Target: v1.DiskTarget{
			Device: "vda",
		},
		Source: v1.DiskSource{
			Host: &v1.DiskSourceHost{
				Name: "iscsi-demo-target",
				Port: "3260",
			},
			Protocol: "iscsi",
			Name:     fmt.Sprintf("iqn.2017-01.io.kubevirt:sn.42/%d", lun),
		},
	}}
	return vm
}

func NewMigrationForVm(vm *v1.VM) *v1.Migration {
	return v1.NewMinimalMigration(vm.ObjectMeta.Name+"migrate", vm.ObjectMeta.Name)
}

func NewRandomVMWithSpice() *v1.VM {
	vm := NewRandomVM()
	vm.Spec.Domain.Devices.Video = []v1.Video{
		{
			Model: v1.VideoModel{
				Type:   "qxl",
				Heads:  1,
				Ram:    65563,
				VGAMem: 16384,
				VRam:   8192,
			},
		},
	}
	vm.Spec.Domain.Devices.Graphics = []v1.Graphics{
		{
			DefaultMode: "any",
			Type:        "spice",
		},
	}
	return vm
}

// Block until the specified VM started
func WaitForSuccessfulVMStart(vm runtime.Object) {
	v, ok := vm.(*v1.VM)
	Expect(ok).To(BeTrue(), "Object is not of type *v1.VM")
	restClient, err := kubecli.GetRESTClient()
	Expect(err).ToNot(HaveOccurred())
	NewObjectEventWatcher(vm, func(event *kubev1.Event) bool {
		Expect(event.Type).NotTo(Equal("Warning"), "Received VM warning event")
		if event.Type == "Normal" && event.Reason == v1.Started.String() {
			obj, err := restClient.Get().Namespace(api.NamespaceDefault).
				Resource("vms").Name(v.GetObjectMeta().GetName()).Do().Get()
			Expect(err).ToNot(HaveOccurred())
			Expect(string(obj.(*v1.VM).Status.Phase)).To(Equal(string(v1.Running)))
			return true
		}
		return false
	}).Watch()
}

func GetReadyNodes() []kubev1.Node {
	coreClient, err := kubecli.Get()
	PanicOnError(err)
	nodes, err := coreClient.CoreV1().Nodes().List(kubev1.ListOptions{})
	Expect(err).ToNot(HaveOccurred())

	readyNodes := []kubev1.Node{}
	for _, node := range nodes.Items {
		for _, condition := range node.Status.Conditions {
			if condition.Type == kubev1.NodeReady {
				if condition.Status == kubev1.ConditionTrue {
					readyNodes = append(readyNodes, node)
					break
				}

			}
		}
	}
	return readyNodes
}
