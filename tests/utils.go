/*
 * This file is part of the KubeVirt project
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
	"encoding/base64"
	goerrors "errors"
	"fmt"
	"reflect"
	"time"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	k8sv1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/fields"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/rand"

	"k8s.io/apimachinery/pkg/api/errors"

	"k8s.io/apimachinery/pkg/api/resource"

	"io"

	"github.com/google/goexpect"

	"kubevirt.io/kubevirt/pkg/api/v1"
	"kubevirt.io/kubevirt/pkg/kubecli"
)

type EventType string

const (
	NormalEvent  EventType = "Normal"
	WarningEvent EventType = "Warning"
)

const defaultTestGracePeriod int64 = 0

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

type ProcessFunc func(event *k8sv1.Event) (done bool)

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

	cli, err := kubecli.GetKubevirtClient()
	if err != nil {
		panic(err)
	}

	f := processFunc

	if w.failOnWarnings {
		f = func(event *k8sv1.Event) bool {
			Expect(event.Type).NotTo(Equal(string(WarningEvent)), "Unexpected Warning event recieved.")
			return processFunc(event)
		}

	}

	uid := w.object.(metav1.ObjectMetaAccessor).GetObjectMeta().GetName()
	eventWatcher, err := cli.CoreV1().Events(k8sv1.NamespaceAll).
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
			if f(obj.Object.(*k8sv1.Event)) {
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

func (w *ObjectEventWatcher) WaitFor(eventType EventType, reason interface{}) (e *k8sv1.Event) {
	w.Watch(func(event *k8sv1.Event) bool {
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

	deletePVC("alpine", false)
	deletePV("alpine", false)

	deletePVC("alpine", true)
	deletePV("alpine", true)

	deletePVC("cirros", false)
	deletePV("cirros", false)

	deletePVC("cirros", true)
	deletePV("cirros", true)

	deleteIscsiSecrets()
	removeNamespaces()
}

func BeforeTestCleanup() {
	cleanNamespaces()
	createIscsiSecrets()
}

func BeforeTestSuitSetup() {
	createNamespaces()
	createIscsiSecrets()

	createPV("cirros", 3, false)
	createPVC("cirros", false)

	createPV("cirros", 3, true)
	createPVC("cirros", true)

	createPV("alpine", 2, true)
	createPVC("alpine", true)

	createPV("alpine", 2, false)
	createPVC("alpine", false)
}

func createPVC(os string, withAuth bool) {
	virtCli, err := kubecli.GetKubevirtClient()
	PanicOnError(err)

	_, err = virtCli.CoreV1().PersistentVolumeClaims(NamespaceTestDefault).Create(newPVC(os, withAuth))
	if !errors.IsAlreadyExists(err) {
		PanicOnError(err)
	}
}

func createPV(os string, lun int32, withAuth bool) {
	virtCli, err := kubecli.GetKubevirtClient()
	PanicOnError(err)

	_, err = virtCli.CoreV1().PersistentVolumes().Create(newPV(os, lun, withAuth))
	if !errors.IsAlreadyExists(err) {
		PanicOnError(err)
	}
}

func deletePVC(os string, withAuth bool) {
	virtCli, err := kubecli.GetKubevirtClient()
	PanicOnError(err)

	name := fmt.Sprintf("disk-%s", os)
	if withAuth {
		name = fmt.Sprintf("disk-auth-%s", os)
	}
	err = virtCli.CoreV1().PersistentVolumeClaims(NamespaceTestDefault).Delete(name, nil)
	if !errors.IsNotFound(err) {
		PanicOnError(err)
	}
}

func deletePV(os string, withAuth bool) {
	virtCli, err := kubecli.GetKubevirtClient()
	PanicOnError(err)

	name := fmt.Sprintf("iscsi-disk-%s-for-tests", os)
	if withAuth {
		name = fmt.Sprintf("iscsi-auth-disk-%s-for-tests", os)
	}

	err = virtCli.CoreV1().PersistentVolumes().Delete(name, nil)
	if !errors.IsNotFound(err) {
		PanicOnError(err)
	}
}

func newPVC(os string, withAuth bool) *k8sv1.PersistentVolumeClaim {
	quantity, err := resource.ParseQuantity("1Gi")
	PanicOnError(err)

	name := fmt.Sprintf("disk-%s", os)
	label := os
	if withAuth {
		name = fmt.Sprintf("disk-auth-%s", os)
		label = fmt.Sprintf("%s-auth", os)
	}

	return &k8sv1.PersistentVolumeClaim{
		ObjectMeta: metav1.ObjectMeta{
			Name: name,
		},
		Spec: k8sv1.PersistentVolumeClaimSpec{
			AccessModes: []k8sv1.PersistentVolumeAccessMode{k8sv1.ReadWriteOnce},
			Resources: k8sv1.ResourceRequirements{
				Requests: k8sv1.ResourceList{
					"storage": quantity,
				},
			},
			Selector: &metav1.LabelSelector{
				MatchLabels: map[string]string{
					"kubevirt.io/test": label,
				},
			},
		},
	}
}

func newPV(os string, lun int32, withAuth bool) *k8sv1.PersistentVolume {
	quantity, err := resource.ParseQuantity("1Gi")
	PanicOnError(err)

	name := fmt.Sprintf("iscsi-disk-%s-for-tests", os)
	target := "iscsi-demo-target.kube-system.svc.cluster.local"
	label := os
	if withAuth {
		name = fmt.Sprintf("iscsi-auth-disk-%s-for-tests", os)
		target = "iscsi-auth-demo-target.kube-system.svc.cluster.local"
		label = fmt.Sprintf("%s-auth", os)
	}

	pv := &k8sv1.PersistentVolume{
		ObjectMeta: metav1.ObjectMeta{
			Name: name,
			Labels: map[string]string{
				"kubevirt.io/test": label,
			},
		},
		Spec: k8sv1.PersistentVolumeSpec{
			AccessModes: []k8sv1.PersistentVolumeAccessMode{k8sv1.ReadWriteOnce},
			Capacity: k8sv1.ResourceList{
				"storage": quantity,
			},
			PersistentVolumeReclaimPolicy: k8sv1.PersistentVolumeReclaimRetain,
			PersistentVolumeSource: k8sv1.PersistentVolumeSource{
				ISCSI: &k8sv1.ISCSIVolumeSource{
					IQN:          "iqn.2017-01.io.kubevirt:sn.42",
					Lun:          lun,
					TargetPortal: target,
				},
			},
		},
	}

	if withAuth {
		pv.Spec.PersistentVolumeSource.ISCSI.SessionCHAPAuth = true
		pv.Spec.PersistentVolumeSource.ISCSI.SecretRef = &k8sv1.LocalObjectReference{
			Name: "iscsi-demo-secret",
		}
	}
	return pv
}

func cleanNamespaces() {
	virtCli, err := kubecli.GetKubevirtClient()
	PanicOnError(err)

	for _, namespace := range testNamespaces {

		_, err := virtCli.CoreV1().Namespaces().Get(namespace, metav1.GetOptions{})
		if err != nil {
			continue
		}

		// Remove all VirtualMachineReplicaSets
		PanicOnError(virtCli.RestClient().Delete().Namespace(namespace).Resource("virtualmachinereplicasets").Do().Error())

		// Remove all Migrations
		PanicOnError(virtCli.RestClient().Delete().Namespace(namespace).Resource("migrations").Do().Error())

		// Remove all VMs
		PanicOnError(virtCli.RestClient().Delete().Namespace(namespace).Resource("virtualmachines").Do().Error())

		// Remove all Pods
		PanicOnError(virtCli.CoreV1().RESTClient().Delete().Namespace(namespace).Resource("pods").Do().Error())

		// Remove all VM Secrets
		PanicOnError(virtCli.CoreV1().RESTClient().Delete().Namespace(namespace).Resource("secrets").Do().Error())
	}
}

func removeNamespaces() {
	virtCli, err := kubecli.GetKubevirtClient()
	PanicOnError(err)

	// First send an initial delete to every namespace
	for _, namespace := range testNamespaces {
		err := virtCli.CoreV1().Namespaces().Delete(namespace, nil)
		if !errors.IsNotFound(err) {
			PanicOnError(err)
		}
	}

	// Wait until the namespaces are terminated
	fmt.Println("")
	for _, namespace := range testNamespaces {
		fmt.Printf("Waiting for namespace %s to be removed, this can take a while ...\n", namespace)
		Eventually(func() bool { return errors.IsNotFound(virtCli.CoreV1().Namespaces().Delete(namespace, nil)) }, 180*time.Second, 1*time.Second).
			Should(BeTrue())
	}
}

func deleteIscsiSecrets() {
	virtCli, err := kubecli.GetKubevirtClient()
	PanicOnError(err)

	for _, namespace := range testNamespaces {
		err = virtCli.CoreV1().Secrets(namespace).Delete("iscsi-demo-secret", nil)
		if !errors.IsNotFound(err) {
			PanicOnError(err)
		}
	}
}

func createIscsiSecrets() {
	virtCli, err := kubecli.GetKubevirtClient()
	PanicOnError(err)

	// Create a Test Namespaces
	for _, namespace := range testNamespaces {
		secret := k8sv1.Secret{
			ObjectMeta: metav1.ObjectMeta{
				Name: "iscsi-demo-secret",
			},
			Type: "kubernetes.io/iscsi-chap",
			Data: map[string][]byte{
				"node.session.auth.password": []byte("demopassword"),
				"node.session.auth.username": []byte("demouser"),
			},
		}

		_, err := virtCli.Core().Secrets(namespace).Create(&secret)
		if !errors.IsAlreadyExists(err) {
			PanicOnError(err)
		}
	}
}

func createNamespaces() {
	virtCli, err := kubecli.GetKubevirtClient()
	PanicOnError(err)

	// Create a Test Namespaces
	for _, namespace := range testNamespaces {
		ns := &k8sv1.Namespace{
			ObjectMeta: metav1.ObjectMeta{
				Name: namespace,
			},
		}
		_, err = virtCli.CoreV1().Namespaces().Create(ns)
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

func NewRandomVM() *v1.VirtualMachine {
	return NewRandomVMWithNS(NamespaceTestDefault)
}

func NewRandomVMWithNS(namespace string) *v1.VirtualMachine {
	vm := v1.NewMinimalVMWithNS(namespace, "testvm"+rand.String(5))

	t := defaultTestGracePeriod
	vm.Spec.TerminationGracePeriodSeconds = &t
	return vm
}

func NewRandomVMWithEphemeralDisk(containerImage string) *v1.VirtualMachine {
	vm := NewRandomVM()
	vm.Spec.Domain.Memory.Unit = "MB"
	vm.Spec.Domain.Memory.Value = 64
	vm.Spec.Domain.Devices.Disks = []v1.Disk{{
		Type:   "RegistryDisk:v1alpha",
		Device: "disk",
		Source: v1.DiskSource{
			Name: containerImage,
		},
		Target: v1.DiskTarget{
			Device: "vda",
		},
	}}
	return vm
}

func NewRandomVMWithEphemeralDiskAndUserdata(containerImage string, cloudInitDataSource, userData string) (*v1.VirtualMachine, error) {
	vm := NewRandomVMWithEphemeralDisk(containerImage)
	vm.Spec.Domain.Devices.Consoles = []v1.Console{
		{
			Type: "pty",
		},
	}
	switch cloudInitDataSource {
	case "noCloud":
		spec := &v1.CloudInitSpec{
			NoCloudData: &v1.CloudInitDataSourceNoCloud{
				UserDataBase64: base64.StdEncoding.EncodeToString([]byte(userData)),
			},
		}
		newDisk := v1.Disk{}
		newDisk.Type = "file"
		newDisk.Target = v1.DiskTarget{
			Device: "vdb",
		}
		newDisk.CloudInit = spec

		vm.Spec.Domain.Devices.Disks = append(vm.Spec.Domain.Devices.Disks, newDisk)
		return vm, nil
	}

	return nil, goerrors.New("Unknown cloud-init data source")
}

func NewRandomVMWithUserData(cloudInitDataSource string, userData string) (*v1.VirtualMachine, error) {
	switch cloudInitDataSource {
	case "noCloud":
		vm := NewRandomVMWithSerialConsole()
		spec := &v1.CloudInitSpec{
			NoCloudData: &v1.CloudInitDataSourceNoCloud{
				UserDataBase64: base64.StdEncoding.EncodeToString([]byte(userData)),
			},
		}
		newDisk := v1.Disk{}
		newDisk.Type = "file"
		newDisk.Target = v1.DiskTarget{
			Device: "vdb",
		}
		newDisk.CloudInit = spec

		vm.Spec.Domain.Devices.Disks = append(vm.Spec.Domain.Devices.Disks, newDisk)
		return vm, nil
	}

	return nil, goerrors.New("Unknown cloud-init data source")
}

func NewRandomVMWithDirectLun(lun int, withAuth bool) *v1.VirtualMachine {
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
				Name: "iscsi-demo-target.kube-system",
				Port: "3260",
			},
			Protocol: "iscsi",
			Name:     fmt.Sprintf("iqn.2017-01.io.kubevirt:sn.42/%d", lun),
		},
	}}

	if withAuth {
		vm.Spec.Domain.Devices.Disks[0].Auth = &v1.DiskAuth{
			Username: "demouser",
			Secret: &v1.DiskSecret{
				Type:  "iscsi",
				Usage: "iscsi-demo-secret",
			},
		}
		vm.Spec.Domain.Devices.Disks[0].Source.Host.Name = "iscsi-auth-demo-target.kube-system"
	}
	return vm
}

func NewRandomVMWithPVC(claimName string) *v1.VirtualMachine {
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

func NewRandomMigrationForVm(vm *v1.VirtualMachine) *v1.Migration {
	ns := vm.GetObjectMeta().GetNamespace()
	return v1.NewMinimalMigrationWithNS(ns, vm.ObjectMeta.Name+"migrate"+rand.String(5), vm.ObjectMeta.Name)
}

func NewRandomVMWithWatchdog() *v1.VirtualMachine {
	vm := NewRandomVMWithDirectLun(2, false)
	vm.Spec.Domain.Devices.Consoles = []v1.Console{
		{
			Type: "pty",
		},
	}
	vm.Spec.Domain.Devices.Watchdog = &v1.Watchdog{
		Model:  "i6300esb",
		Action: "poweroff",
	}

	return vm
}
func NewRandomVMWithSerialConsole() *v1.VirtualMachine {
	vm := NewRandomVMWithPVC("disk-cirros")
	vm.Spec.Domain.Devices.Consoles = []v1.Console{
		{
			Type: "pty",
		},
	}
	return vm
}

func NewRandomVMWithSpice() *v1.VirtualMachine {
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
func WaitForSuccessfulVMStartWithTimeout(vm runtime.Object, seconds int) (nodeName string) {
	_, ok := vm.(*v1.VirtualMachine)
	Expect(ok).To(BeTrue(), "Object is not of type *v1.VM")
	virtClient, err := kubecli.GetKubevirtClient()
	Expect(err).ToNot(HaveOccurred())

	// Fetch the VM, to make sure we have a resourceVersion as a starting point for the watch
	vmMeta := vm.(*v1.VirtualMachine).ObjectMeta
	obj, err := virtClient.RestClient().Get().Resource("virtualmachines").Namespace(vmMeta.Namespace).Name(vmMeta.Name).Do().Get()
	NewObjectEventWatcher(obj).SinceWatchedObjectResourceVersion().FailOnWarnings().WaitFor(NormalEvent, v1.Started)

	// FIXME the event order is wrong. First the document should be updated
	Eventually(func() bool {
		obj, err := virtClient.RestClient().Get().Resource("virtualmachines").Namespace(vmMeta.Namespace).Name(vmMeta.Name).Do().Get()
		Expect(err).ToNot(HaveOccurred())
		fetchedVM := obj.(*v1.VirtualMachine)
		nodeName = fetchedVM.Status.NodeName

		graphicDeviceCount := 0
		for _, src := range fetchedVM.Spec.Domain.Devices.Graphics {
			if src.Type == "spice" || src.Type == "vnc" {
				graphicDeviceCount++
			}
		}

		// wait on both phase and graphics
		if len(fetchedVM.Status.Graphics) == graphicDeviceCount &&
			fetchedVM.Status.Phase == v1.Running {
			return true
		}
		return false
	}, time.Duration(seconds)*time.Second).Should(Equal(true))

	return
}

func WaitForSuccessfulVMStart(vm runtime.Object) string {
	return WaitForSuccessfulVMStartWithTimeout(vm, 30)
}

func GetReadyNodes() []k8sv1.Node {
	virtCli, err := kubecli.GetKubevirtClient()
	PanicOnError(err)
	nodes, err := virtCli.CoreV1().Nodes().List(metav1.ListOptions{})
	Expect(err).ToNot(HaveOccurred())

	readyNodes := []k8sv1.Node{}
	for _, node := range nodes.Items {
		for _, condition := range node.Status.Conditions {
			if condition.Type == k8sv1.NodeReady {
				if condition.Status == k8sv1.ConditionTrue {
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

func NewInt32(x int32) *int32 {
	return &x
}

func newString(x string) *string {
	return &x
}

func NewRandomReplicaSetFromVM(vm *v1.VirtualMachine, replicas int32) *v1.VirtualMachineReplicaSet {
	name := "replicaset" + rand.String(5)
	rs := &v1.VirtualMachineReplicaSet{
		ObjectMeta: metav1.ObjectMeta{Name: "replicaset" + rand.String(5)},
		Spec: v1.VMReplicaSetSpec{
			Replicas: &replicas,
			Selector: &metav1.LabelSelector{
				MatchLabels: map[string]string{"name": name},
			},
			Template: &v1.VMTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: map[string]string{"name": name},
					Name:   vm.ObjectMeta.Name,
				},
				Spec: vm.Spec,
			},
		},
	}
	return rs
}

func NewConsoleExpecter(virtCli kubecli.KubevirtClient, vm *v1.VirtualMachine, consoleName string, timeout time.Duration, opts ...expect.Option) (expect.Expecter, <-chan error, error) {
	vmReader, vmWriter := io.Pipe()
	expecterReader, expecterWriter := io.Pipe()
	resCh := make(chan error)
	stopChan := make(chan struct{})
	go func() {
		err := virtCli.VM(vm.ObjectMeta.Namespace).SerialConsole(vm.ObjectMeta.Name, consoleName, vmReader, expecterWriter)
		resCh <- err
	}()

	return expect.SpawnGeneric(&expect.GenOptions{
		In:  vmWriter,
		Out: expecterReader,
		Wait: func() error {
			return <-resCh
		},
		Close: func() error {
			close(stopChan)
			return nil
		},
		Check: func() bool { return true },
	}, timeout, opts...)
}
