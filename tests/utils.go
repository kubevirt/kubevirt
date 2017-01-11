package tests

import (
	"k8s.io/client-go/pkg/api"
	"k8s.io/client-go/pkg/api/meta"
	kubev1 "k8s.io/client-go/pkg/api/v1"
	"k8s.io/client-go/pkg/fields"
	"k8s.io/client-go/pkg/labels"
	"k8s.io/client-go/pkg/runtime"
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

	uid := w.object.(meta.ObjectMetaAccessor).GetObjectMeta().GetUID()
	eventWatcher, err := cli.Core().Events(api.NamespaceDefault).
		Watch(kubev1.ListOptions{FieldSelector: fields.ParseSelectorOrDie("involvedObject.uid=" + string(uid)).String()})
	if err != nil {
		panic(err)
	}
	defer eventWatcher.Stop()

	for obj := range eventWatcher.ResultChan() {
		if done := w.process(obj.Object.(*kubev1.Event)); done == true {
			break
		}
	}
}

func MustCleanup() {
	coreClient, err := kubecli.Get()
	PanicOnError(err)
	restClient, err := kubecli.GetRESTClient()
	PanicOnError(err)

	// Remove all VMs
	PanicOnError(restClient.Delete().Namespace(api.NamespaceDefault).Resource("vms").Do().Error())

	// Remove VM pods
	labelSelector, err := labels.Parse(v1.AppLabel + " in (virt-launcher)")
	PanicOnError(err)
	err = coreClient.Core().Pods(api.NamespaceDefault).
		DeleteCollection(nil, kubev1.ListOptions{FieldSelector: fields.Everything().String(), LabelSelector: labelSelector.String()})
	PanicOnError(err)
}

func PanicOnError(err error) {
	if err != nil {
		panic(err)
	}
}
