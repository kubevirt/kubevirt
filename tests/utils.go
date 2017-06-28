/*
 * This file is part of the kubevirt project
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 *
 * Copyright 2017 Red Hat, Inc.
 *
 */

package tests

import (
	"fmt"
	"reflect"
	"time"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/fields"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/rand"
	"k8s.io/client-go/pkg/api"
	kubev1 "k8s.io/client-go/pkg/api/v1"

	"kubevirt.io/kubevirt/pkg/api/v1"
	"kubevirt.io/kubevirt/pkg/kubecli"
)

type EventType string

const (
	NormalEvent  EventType = "Normal"
	WarningEvent EventType = "Warning"
)

type startType string

const (
	invalidWatch startType = "invalidWatch"
	// Watch since the moment a long poll connection is established
	watchSinceNow startType = "watchSinceNow"
	// Watch since the resourceVersion of the passed in runtime object
	watchSinceObjectUpdate startType = "watchSinceObjectUpdate"
	// Watch since the resourceVersion of the watched object
	watchSinceWatchedObjectUpdate startType = "watchSinceWatchedObjectUpdate"
	// Watch since the resourceVersion passed in to the builder
	watchSinceResourceVersion startType = "watchSinceResourceVersion"
)

type ProcessFunc func(event *kubev1.Event) (done bool)

type ObjectEventWatcher struct {
	object          runtime.Object
	timeout         *time.Duration
	failOnWarnings  bool
	resourceVersion string
	startType       startType
}

func NewObjectEventWatcher(object runtime.Object) *ObjectEventWatcher {
	return &ObjectEventWatcher{object: object, startType: invalidWatch}
}

func (w *ObjectEventWatcher) Timeout(duration time.Duration) *ObjectEventWatcher {
	w.timeout = &duration
	return w
}

func (w *ObjectEventWatcher) FailOnWarnings() *ObjectEventWatcher {
	w.failOnWarnings = true
	return w
}

/*
SinceNow sets a watch starting point for events, from the moment on the connection to the apiserver
was established.
*/
func (w *ObjectEventWatcher) SinceNow() *ObjectEventWatcher {
	w.startType = watchSinceNow
	return w
}

/*
SinceWatchedObjectResourceVersion takes the resource version of the runtime object which is watched,
and takes it as the starting point for all events to watch for.
*/
func (w *ObjectEventWatcher) SinceWatchedObjectResourceVersion() *ObjectEventWatcher {
	w.startType = watchSinceWatchedObjectUpdate
	return w
}

/*
SinceObjectResourceVersion takes the resource version of the passed in runtime object and takes it
as the starting point for all events to watch for.
*/
func (w *ObjectEventWatcher) SinceObjectResourceVersion(object runtime.Object) *ObjectEventWatcher {
	var err error
	w.startType = watchSinceObjectUpdate
	w.resourceVersion, err = meta.NewAccessor().ResourceVersion(object)
	Expect(err).ToNot(HaveOccurred())
	return w
}

/*
SinceResourceVersion sets the passed in resourceVersion as the starting point for all events to watch for.
*/
func (w *ObjectEventWatcher) SinceResourceVersion(rv string) *ObjectEventWatcher {
	w.resourceVersion = rv
	w.startType = watchSinceResourceVersion
	return w
}

func (w *ObjectEventWatcher) Watch(processFunc ProcessFunc) {
	Expect(w.startType).ToNot(Equal(invalidWatch))
	resourceVersion := ""

	switch w.startType {
	case watchSinceNow:
		resourceVersion = ""
	case watchSinceObjectUpdate, watchSinceResourceVersion:
		resourceVersion = w.resourceVersion
	case watchSinceWatchedObjectUpdate:
		var err error
		w.resourceVersion, err = meta.NewAccessor().ResourceVersion(w.object)
		Expect(err).ToNot(HaveOccurred())
	}

	cli, err := kubecli.Get()
	if err != nil {
		panic(err)
	}

	f := processFunc

	if w.failOnWarnings {
		f = func(event *kubev1.Event) bool {
			Expect(event.Type).NotTo(Equal(string(WarningEvent)), "Unexpected Warning event recieved.")
			return processFunc(event)
		}

	}

	uid := w.object.(metav1.ObjectMetaAccessor).GetObjectMeta().GetName()
	eventWatcher, err := cli.Core().Events(api.NamespaceAll).
		Watch(metav1.ListOptions{
			FieldSelector:   fields.ParseSelectorOrDie("involvedObject.name=" + string(uid)).String(),
			ResourceVersion: resourceVersion,
		})
	if err != nil {
		panic(err)
	}
	defer eventWatcher.Stop()
	timedOut := false
	done := make(chan struct{})

	go func() {
		defer GinkgoRecover()
		for obj := range eventWatcher.ResultChan() {
			if timedOut {
				// If some events are still in the queue, make sure we don't process them anymore
				break
			}
			if f(obj.Object.(*kubev1.Event)) {
				close(done)
				break
			}
		}
	}()

	if w.timeout != nil {
		select {
		case <-done:
		case <-time.After(*w.timeout):
		}
	} else {
		<-done
	}
}

func (w *ObjectEventWatcher) WaitFor(eventType EventType, reason interface{}) (e *kubev1.Event) {
	w.Watch(func(event *kubev1.Event) bool {
		if event.Type == string(eventType) && event.Reason == reflect.ValueOf(reason).String() {
			e = event
			return true
		}
		return false
	})
	return
}

func MustCleanup() {
	coreClient, err := kubecli.Get()
	PanicOnError(err)
	restClient, err := kubecli.GetRESTClient()
	PanicOnError(err)

	kuki := coreClient.Core().Pods(api.NamespaceAll)
	fmt.Println(kuki)
	// Remove all Migrations
	PanicOnError(restClient.Delete().Namespace(api.NamespaceAll).Resource("migrations").Do().Error())

	// Remove all VMs
	PanicOnError(restClient.Delete().Namespace(api.NamespaceAll).Resource("vms").Do().Error())

	// Remove all Jobs
	PanicOnError(coreClient.CoreV1().RESTClient().Delete().AbsPath("/apis/batch/v1/namespaces/default/jobs").Do().Error())

	// Remove all pods associated with a job
	jobPodlabelSelector, err := labels.Parse("job-name")
	PanicOnError(err)
	err = coreClient.Core().Pods(api.NamespaceDefault).
		DeleteCollection(nil, metav1.ListOptions{FieldSelector: fields.Everything().String(), LabelSelector: jobPodlabelSelector.String()})

	PanicOnError(err)
	// Remove VM pods
	vmPodlabelSelector, err := labels.Parse(v1.AppLabel + " in (virt-launcher)")
	PanicOnError(err)
	err = coreClient.Core().Pods(api.NamespaceDefault).
		DeleteCollection(nil, metav1.ListOptions{FieldSelector: fields.Everything().String(), LabelSelector: vmPodlabelSelector.String()})

	PanicOnError(err)
}

func PanicOnError(err error) {
	if err != nil {
		panic(err)
	}
}

func NewRandomVM() *v1.VM {
	return NewRandomVMWithNS(api.NamespaceDefault)
}

func NewRandomVMWithNS(namespace string) *v1.VM {
	return v1.NewMinimalVMWithNS(namespace, "testvm"+rand.String(5))
}

func NewRandomVMWithDirectLun(lun int) *v1.VM {
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

func NewRandomVMWithPVC(claimName string) *v1.VM {
	vm := NewRandomVM()
	vm.Spec.Domain.Memory.Unit = "MB"
	vm.Spec.Domain.Memory.Value = 64
	vm.Spec.Domain.Devices.Disks = []v1.Disk{{
		Type:     "PersistentVolumeClaim",
		Snapshot: "external",
		Device:   "disk",
		Target: v1.DiskTarget{
			Device: "vda",
		},
		Source: v1.DiskSource{
			Name: claimName,
		},
	}}
	return vm
}

func NewRandomMigrationForVm(vm *v1.VM) *v1.Migration {
	ns := vm.GetObjectMeta().GetNamespace()
	return v1.NewMinimalMigrationWithNS(ns, vm.ObjectMeta.Name+"migrate"+rand.String(5), vm.ObjectMeta.Name)
}

func NewRandomVMWithSerialConsole() *v1.VM {
	vm := NewRandomVMWithPVC("disk-cirros")
	vm.Spec.Domain.Devices.Serials = []v1.Serial{
		{
			Type: "pty",
			Target: &v1.SerialTarget{
				Port: newUInt(0),
			},
		},
	}
	vm.Spec.Domain.Devices.Consoles = []v1.Console{
		{
			Type: "pty",
			Target: &v1.ConsoleTarget{
				Type: newString("serial"),
				Port: newUInt(0),
			},
		},
	}
	return vm
}

func NewRandomVMWithSpice() *v1.VM {
	vm := NewRandomVM()
	vm.Spec.Domain.Devices.Video = []v1.Video{
		{
			Type:   "qxl",
			Heads:  newUInt(1),
			Ram:    newUInt(65563),
			VGAMem: newUInt(16384),
			VRam:   newUInt(8192),
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

// Block until the specified VM started and return the target node name.
func WaitForSuccessfulVMStart(vm runtime.Object) (nodeName string) {
	_, ok := vm.(*v1.VM)
	Expect(ok).To(BeTrue(), "Object is not of type *v1.VM")
	restClient, err := kubecli.GetRESTClient()
	Expect(err).ToNot(HaveOccurred())

	// Fetch the VM, to make sure we have a resourceVersion as a starting point for the watch
	vmMeta := vm.(*v1.VM).ObjectMeta
	obj, err := restClient.Get().Resource("vms").Namespace(vmMeta.Namespace).Name(vmMeta.Name).Do().Get()
	NewObjectEventWatcher(obj).SinceWatchedObjectResourceVersion().FailOnWarnings().WaitFor(NormalEvent, v1.Started)

	// FIXME the event order is wrong. First the document should be updated
	Eventually(func() v1.VMPhase {
		obj, err := restClient.Get().Resource("vms").Namespace(vmMeta.Namespace).Name(vmMeta.Name).Do().Get()
		Expect(err).ToNot(HaveOccurred())
		fetchedVM := obj.(*v1.VM)
		nodeName = fetchedVM.Status.NodeName
		return fetchedVM.Status.Phase
	}).Should(Equal(v1.Running))
	return
}

func GetReadyNodes() []kubev1.Node {
	coreClient, err := kubecli.Get()
	PanicOnError(err)
	nodes, err := coreClient.CoreV1().Nodes().List(metav1.ListOptions{})
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

func newUInt(x uint) *uint {
	return &x
}

func newString(x string) *string {
	return &x
}
