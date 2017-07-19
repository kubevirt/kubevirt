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
	"reflect"
	"time"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/fields"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/rand"
	"k8s.io/client-go/pkg/api"
	kubev1 "k8s.io/client-go/pkg/api/v1"

	"k8s.io/apimachinery/pkg/api/errors"

	"k8s.io/apimachinery/pkg/api/resource"

	"kubevirt.io/kubevirt/pkg/api/v1"
	"kubevirt.io/kubevirt/pkg/kubecli"
)

type EventType string

const (
	NormalEvent  EventType = "Normal"
	WarningEvent EventType = "Warning"
)

const (
	// tests.NamespaceTestDefault is the default namespace, to test non-infrastructure related KubeVirt objects.
	NamespaceTestDefault string = "kubevirt-test-default"
	// NamespaceTestAlternative is used to test controller-namespace independency.
	NamespaceTestAlternative string = "kubevirt-test-alternative"
)

var testNamespaces = []string{NamespaceTestDefault, NamespaceTestAlternative}

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

func AfterTestSuitCleanup() {
	// Make sure that the namespaces exist, to not have to check in the cleanup code for existing namespaces
	createNamespaces()
	cleanNamespaces()
	deletePVC("alpine")
	deletePVC("cirros")
	deletePV("alpine")
	deletePV("cirros")
	removeNamespaces()
}

func BeforeTestCleanup() {
	cleanNamespaces()
}

func BeforeTestSuitSetup() {
	createNamespaces()
	createPV("cirros", 3)
	createPV("alpine", 2)
	createPVC("alpine")
	createPVC("cirros")
}

func createPVC(os string) {
	coreClient, err := kubecli.Get()
	PanicOnError(err)

	_, err = coreClient.PersistentVolumeClaims(NamespaceTestDefault).Create(newPVC(os))
	if !errors.IsAlreadyExists(err) {
		PanicOnError(err)
	}
}

func createPV(os string, lun int32) {
	coreClient, err := kubecli.Get()
	PanicOnError(err)

	_, err = coreClient.PersistentVolumes().Create(newPV(os, lun))
	if !errors.IsAlreadyExists(err) {
		PanicOnError(err)
	}
}

func deletePVC(os string) {
	coreClient, err := kubecli.Get()
	PanicOnError(err)

	err = coreClient.PersistentVolumeClaims(NamespaceTestDefault).Delete("disk-"+os, nil)
	if !errors.IsNotFound(err) {
		PanicOnError(err)
	}
}

func deletePV(os string) {
	coreClient, err := kubecli.Get()
	PanicOnError(err)

	err = coreClient.PersistentVolumes().Delete("iscsi-disk-"+os+"-for-tests", nil)
	if !errors.IsNotFound(err) {
		PanicOnError(err)
	}
}

func newPVC(os string) *kubev1.PersistentVolumeClaim {
	quantity, err := resource.ParseQuantity("1Gi")
	PanicOnError(err)
	return &kubev1.PersistentVolumeClaim{
		ObjectMeta: metav1.ObjectMeta{
			Name: "disk-" + os,
		},
		Spec: kubev1.PersistentVolumeClaimSpec{
			AccessModes: []kubev1.PersistentVolumeAccessMode{kubev1.ReadWriteOnce},
			Resources: kubev1.ResourceRequirements{
				Requests: kubev1.ResourceList{
					"storage": quantity,
				},
			},
			Selector: &metav1.LabelSelector{
				MatchLabels: map[string]string{
					"kubevirt.io/test": os,
				},
			},
		},
	}
}

func newPV(os string, lun int32) *kubev1.PersistentVolume {
	quantity, err := resource.ParseQuantity("1Gi")
	PanicOnError(err)
	return &kubev1.PersistentVolume{
		ObjectMeta: metav1.ObjectMeta{
			Name: "iscsi-disk-" + os + "-for-tests",
			Labels: map[string]string{
				"kubevirt.io/test": os,
			},
		},
		Spec: kubev1.PersistentVolumeSpec{
			AccessModes: []kubev1.PersistentVolumeAccessMode{kubev1.ReadWriteOnce},
			Capacity: kubev1.ResourceList{
				"storage": quantity,
			},
			PersistentVolumeReclaimPolicy: kubev1.PersistentVolumeReclaimRetain,
			PersistentVolumeSource: kubev1.PersistentVolumeSource{
				ISCSI: &kubev1.ISCSIVolumeSource{
					IQN:          "iqn.2017-01.io.kubevirt:sn.42",
					Lun:          lun,
					TargetPortal: "iscsi-demo-target.default.svc.cluster.local",
				},
			},
		},
	}
}

func cleanNamespaces() {
	coreClient, err := kubecli.Get()
	PanicOnError(err)
	restClient, err := kubecli.GetRESTClient()
	PanicOnError(err)

	for _, namespace := range testNamespaces {

		_, err := coreClient.Core().Namespaces().Get(namespace, metav1.GetOptions{})
		if err != nil {
			continue
		}

		// Remove all Migrations
		PanicOnError(restClient.Delete().Namespace(namespace).Resource("migrations").Do().Error())

		// Remove all VMs
		PanicOnError(restClient.Delete().Namespace(namespace).Resource("vms").Do().Error())

		// Remove all Pods
		PanicOnError(coreClient.CoreV1().RESTClient().Delete().Namespace(namespace).Resource("pods").Do().Error())
	}
}

func removeNamespaces() {
	coreClient, err := kubecli.Get()
	PanicOnError(err)

	// First send an initial delete to every namespace
	for _, namespace := range testNamespaces {
		err := coreClient.Namespaces().Delete(namespace, nil)
		if !errors.IsNotFound(err) {
			PanicOnError(err)
		}
	}

	// Wait until the namespaces are terminated
	for _, namespace := range testNamespaces {
		Eventually(func() bool { return errors.IsNotFound(coreClient.Namespaces().Delete(namespace, nil)) }, 30*time.Second, 1*time.Second).
			Should(BeTrue())
	}
}

func createNamespaces() {
	coreClient, err := kubecli.Get()
	PanicOnError(err)

	// Create a Test Namespaces
	for _, namespace := range testNamespaces {
		ns := &kubev1.Namespace{
			ObjectMeta: metav1.ObjectMeta{
				Name: namespace,
			},
		}
		_, err = coreClient.Namespaces().Create(ns)
		if !errors.IsAlreadyExists(err) {
			PanicOnError(err)
		}
	}
}

func PanicOnError(err error) {
	if err != nil {
		panic(err)
	}
}

func NewRandomVM() *v1.VM {
	return NewRandomVMWithNS(NamespaceTestDefault)
}

func NewRandomVMWithNS(namespace string) *v1.VM {
	return v1.NewMinimalVMWithNS(namespace, "testvm"+rand.String(5))
}

func NewRandomVMWithDirectLun(lun int32) *v1.VM {
	vm := NewRandomVM()
	vm.Spec.Domain.Memory.Value = 64
	vm.Spec.Domain.Devices.Disks = []v1.Disk{{
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
			ISCSI: &v1.DiskSourceISCSI{
				TargetPortal: "iscsi-demo-target.default:3260",
				IQN:          "iqn.2017-01.io.kubevirt:sn.42",
				Lun:          lun,
			},
		},
	}}
	return vm
}

func NewRandomVMWithPVC(claimName string) *v1.VM {
	vm := NewRandomVM()
	vm.Spec.Domain.Memory.Value = 64
	vm.Spec.Domain.Devices.Disks = []v1.Disk{{
		Snapshot: "external",
		Device:   "disk",
		Target: v1.DiskTarget{
			Device: "vda",
		},
		Source: v1.DiskSource{
			PersistentVolumeClaim: &v1.DiskSourcePersistentVolumeClaim{
				ClaimName: claimName,
			},
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
			Target: &v1.SerialTarget{
				Port: newUInt(0),
			},
		},
	}
	vm.Spec.Domain.Devices.Consoles = []v1.Console{
		{
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
			Type: "spice",
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
