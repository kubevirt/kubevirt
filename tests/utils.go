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
	"bytes"
	"encoding/base64"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"os/exec"
	"reflect"
	"strconv"
	"time"

	"github.com/google/goexpect"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/spf13/cobra"

	k8sv1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/meta"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/fields"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/rand"

	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/tools/remotecommand"

	"k8s.io/apimachinery/pkg/types"

	"k8s.io/apimachinery/pkg/labels"

	"kubevirt.io/kubevirt/pkg/api/v1"
	"kubevirt.io/kubevirt/pkg/controller"
	"kubevirt.io/kubevirt/pkg/kubecli"
	"kubevirt.io/kubevirt/pkg/log"
	"kubevirt.io/kubevirt/pkg/virtctl"
)

var KubeVirtVersionTag = "latest"
var KubeVirtRepoPrefix = "kubevirt"
var KubeVirtKubectlPath = ""
var KubeVirtInstallNamespace = "kube-system"

func init() {
	flag.StringVar(&KubeVirtVersionTag, "tag", "latest", "Set the image tag or digest to use")
	flag.StringVar(&KubeVirtRepoPrefix, "prefix", "kubevirt", "Set the repository prefix for all images")
	flag.StringVar(&KubeVirtKubectlPath, "kubectl-path", "", "Set path to kubectl binary")
	flag.StringVar(&KubeVirtInstallNamespace, "installed-namespace", "kube-system", "Set the namespace KubeVirt is installed in")
}

type EventType string

const (
	NormalEvent  EventType = "Normal"
	WarningEvent EventType = "Warning"
)

const defaultTestGracePeriod int64 = 0

const (
	SubresourceServiceAccountName = "kubevirt-subresource-test-sa"
	AdminServiceAccountName       = "kubevirt-admin-test-sa"
	EditServiceAccountName        = "kubevirt-edit-test-sa"
	ViewServiceAccountName        = "kubevirt-view-test-sa"
)

const SubresourceTestLabel = "subresource-access-test-pod"

const (
	// tests.NamespaceTestDefault is the default namespace, to test non-infrastructure related KubeVirt objects.
	NamespaceTestDefault = "kubevirt-test-default"
	// NamespaceTestAlternative is used to test controller-namespace independency.
	NamespaceTestAlternative = "kubevirt-test-alternative"
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

const (
	osAlpineHostPath = "alpine-host-path"
	osWindows        = "windows"
	CustomHostPath   = "custom-host-path"
)

const (
	HostPathAlpine = "/tmp/hostImages/alpine"
	HostPathCustom = "/tmp/hostImages/custom"
)

const (
	DiskAlpineHostPath = "disk-alpine-host-path"
	DiskWindows        = "disk-windows"
	DiskCustomHostPath = "disk-custom-host-path"
)

const (
	defaultDiskSize        = "1Gi"
	defaultWindowsDiskSize = "30Gi"
)

const VMIResource = "virtualmachineinstances"

const (
	SecretLabel = "kubevirt.io/secret"
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
			if event.Type == string(WarningEvent) {
				log.Log.Reason(fmt.Errorf("unexpected warning event received")).Error(event.Message)
			} else {
				log.Log.Infof(event.Message)
			}
			Expect(event.Type).NotTo(Equal(string(WarningEvent)), "Unexpected Warning event received.")
			return processFunc(event)
		}

	} else {
		f = func(event *k8sv1.Event) bool {
			if event.Type == string(WarningEvent) {
				log.Log.Reason(fmt.Errorf("unexpected warning event received")).Error(event.Message)
			} else {
				log.Log.Infof(event.Message)
			}
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
	cleanupServiceAccounts()

	DeletePVC(osWindows)

	DeletePVC(osAlpineHostPath)
	DeletePV(osAlpineHostPath)

	removeNamespaces()
}

func BeforeTestCleanup() {
	cleanNamespaces()
}

func BeforeTestSuitSetup() {

	log.InitializeLogging("tests")
	log.Log.SetIOWriter(GinkgoWriter)

	createNamespaces()
	createServiceAccounts()

	CreateHostPathPv(osAlpineHostPath, HostPathAlpine)
	CreatePVC(osAlpineHostPath, defaultDiskSize)

	CreatePVC(osWindows, defaultWindowsDiskSize)
}

func CreatePVC(os string, size string) {
	virtCli, err := kubecli.GetKubevirtClient()
	PanicOnError(err)

	_, err = virtCli.CoreV1().PersistentVolumeClaims(NamespaceTestDefault).Create(newPVC(os, size))
	if !errors.IsAlreadyExists(err) {
		PanicOnError(err)
	}
}

func newPVC(os string, size string) *k8sv1.PersistentVolumeClaim {
	quantity, err := resource.ParseQuantity(size)
	PanicOnError(err)

	name := fmt.Sprintf("disk-%s", os)
	return &k8sv1.PersistentVolumeClaim{
		ObjectMeta: metav1.ObjectMeta{Name: name},
		Spec: k8sv1.PersistentVolumeClaimSpec{
			AccessModes: []k8sv1.PersistentVolumeAccessMode{k8sv1.ReadWriteOnce},
			Resources: k8sv1.ResourceRequirements{
				Requests: k8sv1.ResourceList{
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

func CreateHostPathPv(os string, hostPath string) {
	virtCli, err := kubecli.GetKubevirtClient()
	PanicOnError(err)

	quantity, err := resource.ParseQuantity("1Gi")
	PanicOnError(err)

	name := fmt.Sprintf("%s-disk-for-tests", os)
	hostPathType := k8sv1.HostPathDirectoryOrCreate
	pv := &k8sv1.PersistentVolume{
		ObjectMeta: metav1.ObjectMeta{
			Name: name,
			Labels: map[string]string{
				"kubevirt.io/test": os,
			},
		},
		Spec: k8sv1.PersistentVolumeSpec{
			AccessModes: []k8sv1.PersistentVolumeAccessMode{k8sv1.ReadWriteOnce},
			Capacity: k8sv1.ResourceList{
				"storage": quantity,
			},
			PersistentVolumeReclaimPolicy: k8sv1.PersistentVolumeReclaimRetain,
			PersistentVolumeSource: k8sv1.PersistentVolumeSource{
				HostPath: &k8sv1.HostPathVolumeSource{
					Path: hostPath,
					Type: &hostPathType,
				},
			},
		},
	}

	_, err = virtCli.CoreV1().PersistentVolumes().Create(pv)
	if !errors.IsAlreadyExists(err) {
		PanicOnError(err)
	}
}

func cleanupSubresourceServiceAccount() {
	virtCli, err := kubecli.GetKubevirtClient()
	PanicOnError(err)

	err = virtCli.CoreV1().ServiceAccounts(NamespaceTestDefault).Delete(SubresourceServiceAccountName, nil)
	if !errors.IsNotFound(err) {
		PanicOnError(err)
	}

	err = virtCli.RbacV1().ClusterRoles().Delete(SubresourceServiceAccountName, nil)
	if !errors.IsNotFound(err) {
		PanicOnError(err)
	}

	err = virtCli.RbacV1().ClusterRoleBindings().Delete(SubresourceServiceAccountName, nil)
	if !errors.IsNotFound(err) {
		PanicOnError(err)
	}
}

func createServiceAccount(saName string, clusterRole string) {
	virtCli, err := kubecli.GetKubevirtClient()
	PanicOnError(err)

	sa := k8sv1.ServiceAccount{
		ObjectMeta: metav1.ObjectMeta{
			Name:      saName,
			Namespace: NamespaceTestDefault,
			Labels: map[string]string{
				"kubevirt.io/test": saName,
			},
		},
	}

	_, err = virtCli.CoreV1().ServiceAccounts(NamespaceTestDefault).Create(&sa)
	if !errors.IsAlreadyExists(err) {
		PanicOnError(err)
	}

	roleBinding := rbacv1.ClusterRoleBinding{
		ObjectMeta: metav1.ObjectMeta{
			Name:      saName,
			Namespace: NamespaceTestDefault,
			Labels: map[string]string{
				"kubevirt.io/test": saName,
			},
		},
		RoleRef: rbacv1.RoleRef{
			Kind:     "ClusterRole",
			Name:     clusterRole,
			APIGroup: "rbac.authorization.k8s.io",
		},
	}
	roleBinding.Subjects = append(roleBinding.Subjects, rbacv1.Subject{
		Kind:      "ServiceAccount",
		Name:      saName,
		Namespace: NamespaceTestDefault,
	})

	_, err = virtCli.RbacV1().ClusterRoleBindings().Create(&roleBinding)
	if !errors.IsAlreadyExists(err) {
		PanicOnError(err)
	}
}

func cleanupServiceAccount(saName string) {
	virtCli, err := kubecli.GetKubevirtClient()
	PanicOnError(err)

	err = virtCli.RbacV1().ClusterRoleBindings().Delete(saName, nil)
	if !errors.IsNotFound(err) {
		PanicOnError(err)
	}

	err = virtCli.CoreV1().ServiceAccounts(NamespaceTestDefault).Delete(saName, nil)
	if !errors.IsNotFound(err) {
		PanicOnError(err)
	}
}

func createSubresourceServiceAccount() {
	virtCli, err := kubecli.GetKubevirtClient()
	PanicOnError(err)

	sa := k8sv1.ServiceAccount{
		ObjectMeta: metav1.ObjectMeta{
			Name:      SubresourceServiceAccountName,
			Namespace: NamespaceTestDefault,
			Labels: map[string]string{
				"kubevirt.io/test": "sa",
			},
		},
	}

	_, err = virtCli.CoreV1().ServiceAccounts(NamespaceTestDefault).Create(&sa)
	if !errors.IsAlreadyExists(err) {
		PanicOnError(err)
	}

	role := rbacv1.ClusterRole{

		ObjectMeta: metav1.ObjectMeta{
			Name:      SubresourceServiceAccountName,
			Namespace: NamespaceTestDefault,
			Labels: map[string]string{
				"kubevirt.io/test": "sa",
			},
		},
	}
	role.Rules = append(role.Rules, rbacv1.PolicyRule{
		APIGroups: []string{"subresources.kubevirt.io"},
		Resources: []string{"virtualmachineinstances/test"},
		Verbs:     []string{"get"},
	})

	_, err = virtCli.RbacV1().ClusterRoles().Create(&role)
	if !errors.IsAlreadyExists(err) {
		PanicOnError(err)
	}

	roleBinding := rbacv1.ClusterRoleBinding{
		ObjectMeta: metav1.ObjectMeta{
			Name:      SubresourceServiceAccountName,
			Namespace: NamespaceTestDefault,
			Labels: map[string]string{
				"kubevirt.io/test": "sa",
			},
		},
		RoleRef: rbacv1.RoleRef{
			Kind:     "ClusterRole",
			Name:     SubresourceServiceAccountName,
			APIGroup: "rbac.authorization.k8s.io",
		},
	}
	roleBinding.Subjects = append(roleBinding.Subjects, rbacv1.Subject{
		Kind:      "ServiceAccount",
		Name:      SubresourceServiceAccountName,
		Namespace: NamespaceTestDefault,
	})

	_, err = virtCli.RbacV1().ClusterRoleBindings().Create(&roleBinding)
	if !errors.IsAlreadyExists(err) {
		PanicOnError(err)
	}
}

func createServiceAccounts() {
	createSubresourceServiceAccount()

	createServiceAccount(AdminServiceAccountName, "kubevirt.io:admin")
	createServiceAccount(ViewServiceAccountName, "kubevirt.io:view")
	createServiceAccount(EditServiceAccountName, "kubevirt.io:edit")
}

func cleanupServiceAccounts() {
	cleanupSubresourceServiceAccount()

	cleanupServiceAccount(AdminServiceAccountName)
	cleanupServiceAccount(ViewServiceAccountName)
	cleanupServiceAccount(EditServiceAccountName)
}

func DeletePVC(os string) {
	virtCli, err := kubecli.GetKubevirtClient()
	PanicOnError(err)

	name := fmt.Sprintf("disk-%s", os)
	err = virtCli.CoreV1().PersistentVolumeClaims(NamespaceTestDefault).Delete(name, nil)
	if !errors.IsNotFound(err) {
		PanicOnError(err)
	}
}

func DeletePV(os string) {
	virtCli, err := kubecli.GetKubevirtClient()
	PanicOnError(err)

	name := fmt.Sprintf("%s-disk-for-tests", os)
	err = virtCli.CoreV1().PersistentVolumes().Delete(name, nil)
	if !errors.IsNotFound(err) {
		PanicOnError(err)
	}
}

func GetRunningPodByLabel(label string, labelType string, namespace string) *k8sv1.Pod {
	virtCli, err := kubecli.GetKubevirtClient()
	PanicOnError(err)

	labelSelector := fmt.Sprintf("%s=%s", labelType, label)
	fieldSelector := fmt.Sprintf("status.phase==%s", k8sv1.PodRunning)
	pods, err := virtCli.CoreV1().Pods(namespace).List(
		metav1.ListOptions{LabelSelector: labelSelector, FieldSelector: fieldSelector},
	)
	PanicOnError(err)

	if len(pods.Items) == 0 {
		PanicOnError(fmt.Errorf("failed to find pod with the label %s", label))
	}

	var readyPod *k8sv1.Pod
	for _, pod := range pods.Items {
		ready := true
		for _, status := range pod.Status.ContainerStatuses {
			if !status.Ready {
				ready = false
			}
		}
		if ready {
			readyPod = &pod
			break
		}
	}
	if readyPod == nil {
		PanicOnError(fmt.Errorf("no ready pods with the label %s", label))
	}

	return readyPod
}

func cleanNamespaces() {
	virtCli, err := kubecli.GetKubevirtClient()
	PanicOnError(err)

	for _, namespace := range testNamespaces {

		_, err := virtCli.CoreV1().Namespaces().Get(namespace, metav1.GetOptions{})
		if err != nil {
			continue
		}

		// Remove all VirtualMachines
		PanicOnError(virtCli.RestClient().Delete().Namespace(namespace).Resource("virtualmachines").Do().Error())

		// Remove all VirtualMachineReplicaSets
		PanicOnError(virtCli.RestClient().Delete().Namespace(namespace).Resource("virtualmachineinstancereplicasets").Do().Error())

		// Remove all VMIs
		PanicOnError(virtCli.RestClient().Delete().Namespace(namespace).Resource("virtualmachineinstances").Do().Error())
		vmis, err := virtCli.VirtualMachineInstance(namespace).List(&metav1.ListOptions{})
		PanicOnError(err)
		for _, vmi := range vmis.Items {
			if controller.HasFinalizer(&vmi, v1.VirtualMachineInstanceFinalizer) {
				_, err := virtCli.VirtualMachineInstance(vmi.Namespace).Patch(vmi.Name, types.JSONPatchType, []byte("[{ \"op\": \"remove\", \"path\": \"/metadata/finalizers\" }]"))
				if !errors.IsNotFound(err) {
					PanicOnError(err)
				}
			}
		}

		// Remove all Pods
		PanicOnError(virtCli.CoreV1().RESTClient().Delete().Namespace(namespace).Resource("pods").Do().Error())

		// Remove all VirtualMachineInstance Secrets
		labelSelector := fmt.Sprintf("%s", SecretLabel)
		PanicOnError(
			virtCli.CoreV1().Secrets(namespace).DeleteCollection(
				&metav1.DeleteOptions{}, metav1.ListOptions{LabelSelector: labelSelector},
			),
		)

		// Remove all VirtualMachineInstance Presets
		PanicOnError(virtCli.RestClient().Delete().Namespace(namespace).Resource("virtualmachineinstancepresets").Do().Error())
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

func NewRandomVMI() *v1.VirtualMachineInstance {
	return NewRandomVMIWithNS(NamespaceTestDefault)
}

func NewRandomVMIWithNS(namespace string) *v1.VirtualMachineInstance {
	vmi := v1.NewMinimalVMIWithNS(namespace, "testvmi"+rand.String(5))

	t := defaultTestGracePeriod
	vmi.Spec.TerminationGracePeriodSeconds = &t
	return vmi
}

func NewRandomVMIWithEphemeralDiskHighMemory(containerImage string) *v1.VirtualMachineInstance {
	vmi := NewRandomVMIWithEphemeralDisk(containerImage)

	vmi.Spec.Domain.Resources.Requests[k8sv1.ResourceMemory] = resource.MustParse("512M")
	return vmi
}

func NewRandomVMIWithEphemeralDiskAndUserdataHighMemory(containerImage string, userData string) *v1.VirtualMachineInstance {
	vmi := NewRandomVMIWithEphemeralDiskAndUserdata(containerImage, userData)

	vmi.Spec.Domain.Resources.Requests[k8sv1.ResourceMemory] = resource.MustParse("512M")
	return vmi
}

func NewRandomVMIWithEphemeralDisk(containerImage string) *v1.VirtualMachineInstance {
	vmi := NewRandomVMI()

	vmi.Spec.Domain.Resources.Requests[k8sv1.ResourceMemory] = resource.MustParse("64M")
	AddEphemeralDisk(vmi, "disk0", "virtio", containerImage)
	return vmi
}

func AddEphemeralDisk(vmi *v1.VirtualMachineInstance, name string, bus string, image string) *v1.VirtualMachineInstance {
	vmi.Spec.Domain.Devices.Disks = append(vmi.Spec.Domain.Devices.Disks, v1.Disk{
		Name:       name,
		VolumeName: name,
		DiskDevice: v1.DiskDevice{
			Disk: &v1.DiskTarget{
				Bus: bus,
			},
		},
	})
	vmi.Spec.Volumes = append(vmi.Spec.Volumes, v1.Volume{
		Name: name,
		VolumeSource: v1.VolumeSource{
			RegistryDisk: &v1.RegistryDiskSource{
				Image: image,
			},
		},
	})

	return vmi
}

func AddBootOrderToDisk(vmi *v1.VirtualMachineInstance, diskName string, bootorder *uint) *v1.VirtualMachineInstance {
	for i, d := range vmi.Spec.Domain.Devices.Disks {
		if d.Name == diskName {
			vmi.Spec.Domain.Devices.Disks[i].BootOrder = bootorder
			return vmi
		}
	}
	return vmi
}

func AddPVCDisk(vmi *v1.VirtualMachineInstance, name string, bus string, claimName string) *v1.VirtualMachineInstance {
	vmi.Spec.Domain.Devices.Disks = append(vmi.Spec.Domain.Devices.Disks, v1.Disk{
		Name:       name,
		VolumeName: name,
		DiskDevice: v1.DiskDevice{
			Disk: &v1.DiskTarget{
				Bus: bus,
			},
		},
	})
	vmi.Spec.Volumes = append(vmi.Spec.Volumes, v1.Volume{
		Name: name,
		VolumeSource: v1.VolumeSource{
			PersistentVolumeClaim: &k8sv1.PersistentVolumeClaimVolumeSource{
				ClaimName: claimName,
			},
		},
	})

	return vmi
}

func AddEphemeralFloppy(vmi *v1.VirtualMachineInstance, name string, image string) *v1.VirtualMachineInstance {
	vmi.Spec.Domain.Devices.Disks = append(vmi.Spec.Domain.Devices.Disks, v1.Disk{
		Name:       name,
		VolumeName: name,
		DiskDevice: v1.DiskDevice{
			Floppy: &v1.FloppyTarget{},
		},
	})
	vmi.Spec.Volumes = append(vmi.Spec.Volumes, v1.Volume{
		Name: name,
		VolumeSource: v1.VolumeSource{
			RegistryDisk: &v1.RegistryDiskSource{
				Image: image,
			},
		},
	})

	return vmi
}

func NewRandomVMIWithEphemeralDiskAndUserdata(containerImage string, userData string) *v1.VirtualMachineInstance {
	vmi := NewRandomVMIWithEphemeralDisk(containerImage)

	vmi.Spec.Domain.Devices.Disks = append(vmi.Spec.Domain.Devices.Disks, v1.Disk{
		Name:       "disk1",
		VolumeName: "disk1",
		DiskDevice: v1.DiskDevice{
			Disk: &v1.DiskTarget{
				Bus: "virtio",
			},
		},
	})
	vmi.Spec.Volumes = append(vmi.Spec.Volumes, v1.Volume{
		Name: "disk1",
		VolumeSource: v1.VolumeSource{
			CloudInitNoCloud: &v1.CloudInitNoCloudSource{
				UserDataBase64: base64.StdEncoding.EncodeToString([]byte(userData)),
			},
		},
	})
	return vmi
}

func NewRandomVMIWithPVC(claimName string) *v1.VirtualMachineInstance {
	vmi := NewRandomVMI()

	vmi.Spec.Domain.Resources.Requests[k8sv1.ResourceMemory] = resource.MustParse("64M")
	vmi.Spec.Domain.Devices.Disks = append(vmi.Spec.Domain.Devices.Disks, v1.Disk{
		Name:       "disk0",
		VolumeName: "disk0",
		DiskDevice: v1.DiskDevice{
			Disk: &v1.DiskTarget{
				Bus: "virtio",
			},
		},
	})
	vmi.Spec.Volumes = append(vmi.Spec.Volumes, v1.Volume{
		Name: "disk0",
		VolumeSource: v1.VolumeSource{
			PersistentVolumeClaim: &k8sv1.PersistentVolumeClaimVolumeSource{
				ClaimName: claimName,
			},
		},
	})
	return vmi
}

func NewRandomVMIWithCDRom(claimName string) *v1.VirtualMachineInstance {
	vmi := NewRandomVMI()

	vmi.Spec.Domain.Resources.Requests[k8sv1.ResourceMemory] = resource.MustParse("64M")
	vmi.Spec.Domain.Devices.Disks = append(vmi.Spec.Domain.Devices.Disks, v1.Disk{
		Name:       "disk0",
		VolumeName: "disk0",
		DiskDevice: v1.DiskDevice{
			CDRom: &v1.CDRomTarget{
				// Do not specify ReadOnly flag so that
				// default behavior can be tested
				Bus: "sata",
			},
		},
	})
	vmi.Spec.Volumes = append(vmi.Spec.Volumes, v1.Volume{
		Name: "disk0",
		VolumeSource: v1.VolumeSource{
			PersistentVolumeClaim: &k8sv1.PersistentVolumeClaimVolumeSource{
				ClaimName: claimName,
			},
		},
	})
	return vmi
}

func NewRandomVMIWithEphemeralPVC(claimName string) *v1.VirtualMachineInstance {
	vmi := NewRandomVMI()

	vmi.Spec.Domain.Resources.Requests[k8sv1.ResourceMemory] = resource.MustParse("64M")
	vmi.Spec.Domain.Devices.Disks = append(vmi.Spec.Domain.Devices.Disks, v1.Disk{
		Name:       "disk0",
		VolumeName: "disk0",
		DiskDevice: v1.DiskDevice{
			Disk: &v1.DiskTarget{
				Bus: "sata",
			},
		},
	})
	vmi.Spec.Volumes = append(vmi.Spec.Volumes, v1.Volume{
		Name: "disk0",

		VolumeSource: v1.VolumeSource{
			Ephemeral: &v1.EphemeralVolumeSource{
				PersistentVolumeClaim: &k8sv1.PersistentVolumeClaimVolumeSource{
					ClaimName: claimName,
				},
			},
		},
	})
	return vmi
}

func NewRandomVMIWithWatchdog() *v1.VirtualMachineInstance {
	vmi := NewRandomVMIWithEphemeralDisk(RegistryDiskFor(RegistryDiskAlpine))

	vmi.Spec.Domain.Devices.Watchdog = &v1.Watchdog{
		Name: "mywatchdog",
		WatchdogDevice: v1.WatchdogDevice{
			I6300ESB: &v1.I6300ESBWatchdog{
				Action: v1.WatchdogActionPoweroff,
			},
		},
	}
	return vmi
}

func NewRandomVMIWithe1000NetworkInterface() *v1.VirtualMachineInstance {
	// Use alpine because cirros dhcp client starts prematurily before link is ready
	vmi := NewRandomVMIWithEphemeralDisk(RegistryDiskFor(RegistryDiskAlpine))
	vmi.ObjectMeta.Annotations = map[string]string{v1.InterfaceModel: "e1000"}
	return vmi
}

// Block until the specified VirtualMachineInstance started and return the target node name.
func waitForVMIStart(obj runtime.Object, seconds int, ignoreWarnings bool) (nodeName string) {
	vmi, ok := obj.(*v1.VirtualMachineInstance)
	Expect(ok).To(BeTrue(), "Object is not of type *v1.VMI")

	virtClient, err := kubecli.GetKubevirtClient()
	Expect(err).ToNot(HaveOccurred())

	// Fetch the VirtualMachineInstance, to make sure we have a resourceVersion as a starting point for the watch
	vmi, err = virtClient.VirtualMachineInstance(vmi.Namespace).Get(vmi.Name, &metav1.GetOptions{})
	Expect(err).ToNot(HaveOccurred())

	objectEventWatcher := NewObjectEventWatcher(vmi).SinceWatchedObjectResourceVersion().Timeout(time.Duration(seconds) * time.Second)
	if ignoreWarnings != true {
		objectEventWatcher.FailOnWarnings()
	}
	objectEventWatcher.WaitFor(NormalEvent, v1.Started)

	// FIXME the event order is wrong. First the document should be updated
	Eventually(func() bool {
		vmi, err = virtClient.VirtualMachineInstance(vmi.Namespace).Get(vmi.Name, &metav1.GetOptions{})
		Expect(err).ToNot(HaveOccurred())

		nodeName = vmi.Status.NodeName

		// wait on both phase and graphics
		if vmi.Status.Phase == v1.Running {
			return true
		}
		return false
	}, time.Duration(seconds)*time.Second).Should(Equal(true))

	return
}

func WaitForSuccessfulVMIStartIgnoreWarnings(vmi runtime.Object) string {
	return waitForVMIStart(vmi, 30, true)
}

func WaitForSuccessfulVMIStartWithTimeout(vmi runtime.Object, seconds int) (nodeName string) {
	return waitForVMIStart(vmi, seconds, false)
}

func WaitForVirtualMachineToDisappearWithTimeout(vmi *v1.VirtualMachineInstance, seconds int) {
	virtClient, err := kubecli.GetKubevirtClient()
	Expect(err).ToNot(HaveOccurred())
	Eventually(func() bool {
		_, err := virtClient.VirtualMachineInstance(vmi.Namespace).Get(vmi.Name, &metav1.GetOptions{})
		return errors.IsNotFound(err)
	}, seconds, 1*time.Second).Should(BeTrue())
}

func WaitForSuccessfulVMIStart(vmi runtime.Object) string {
	return waitForVMIStart(vmi, 90, false)
}

func NewInt32(x int32) *int32 {
	return &x
}

func NewRandomReplicaSetFromVMI(vmi *v1.VirtualMachineInstance, replicas int32) *v1.VirtualMachineInstanceReplicaSet {
	name := "replicaset" + rand.String(5)
	rs := &v1.VirtualMachineInstanceReplicaSet{
		ObjectMeta: metav1.ObjectMeta{Name: "replicaset" + rand.String(5)},
		Spec: v1.VirtualMachineInstanceReplicaSetSpec{
			Replicas: &replicas,
			Selector: &metav1.LabelSelector{
				MatchLabels: map[string]string{"name": name},
			},
			Template: &v1.VirtualMachineInstanceTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: map[string]string{"name": name},
					Name:   vmi.ObjectMeta.Name,
				},
				Spec: vmi.Spec,
			},
		},
	}
	return rs
}

func NewBool(x bool) *bool {
	return &x
}

func RenderJob(name string, cmd []string, args []string) *k8sv1.Pod {
	job := k8sv1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			GenerateName: name,
			Labels: map[string]string{
				v1.AppLabel: "test",
			},
		},
		Spec: k8sv1.PodSpec{
			RestartPolicy: k8sv1.RestartPolicyNever,
			Containers: []k8sv1.Container{
				{
					Name:    name,
					Image:   fmt.Sprintf("%s/vm-killer:%s", KubeVirtRepoPrefix, KubeVirtVersionTag),
					Command: cmd,
					Args:    args,
					SecurityContext: &k8sv1.SecurityContext{
						Privileged: NewBool(true),
						RunAsUser:  new(int64),
					},
				},
			},
			HostPID: true,
			SecurityContext: &k8sv1.PodSecurityContext{
				RunAsUser: new(int64),
			},
		},
	}

	return &job
}

func NewConsoleExpecter(virtCli kubecli.KubevirtClient, vmi *v1.VirtualMachineInstance, timeout time.Duration, opts ...expect.Option) (expect.Expecter, <-chan error, error) {
	vmiReader, vmiWriter := io.Pipe()
	expecterReader, expecterWriter := io.Pipe()
	resCh := make(chan error)
	stopChan := make(chan struct{})
	go func() {
		con, err := virtCli.VirtualMachineInstance(vmi.ObjectMeta.Namespace).SerialConsole(vmi.ObjectMeta.Name)
		if err != nil {
			resCh <- err
			return
		}

		resCh <- con.Stream(kubecli.StreamOptions{
			In:  vmiReader,
			Out: expecterWriter,
		})
	}()

	return expect.SpawnGeneric(&expect.GenOptions{
		In:  vmiWriter,
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

type RegistryDisk string

const (
	RegistryDiskCirros RegistryDisk = "cirros"
	RegistryDiskAlpine RegistryDisk = "alpine"
	RegistryDiskFedora RegistryDisk = "fedora-cloud"
)

// RegistryDiskFor takes the name of an image and returns the full
// registry diks image path.
// Supported values are: cirros, fedora, alpine
func RegistryDiskFor(name RegistryDisk) string {
	switch name {
	case RegistryDiskCirros, RegistryDiskAlpine, RegistryDiskFedora:
		return fmt.Sprintf("%s/%s-registry-disk-demo:%s", KubeVirtRepoPrefix, name, KubeVirtVersionTag)
	}
	panic(fmt.Sprintf("Unsupported registry disk %s", name))
}

func CheckForTextExpecter(vmi *v1.VirtualMachineInstance, text string, wait int) error {
	virtClient, err := kubecli.GetKubevirtClient()
	PanicOnError(err)
	expecter, _, err := NewConsoleExpecter(virtClient, vmi, 10*time.Second)
	if err != nil {
		return err
	}
	b := append([]expect.Batcher{
		&expect.BSnd{S: "\n"},
		&expect.BSnd{S: "\n"},
		&expect.BExp{R: text},
	})
	_, err = expecter.ExpectBatch(b, time.Second*time.Duration(wait))
	return err
}

func LoggedInCirrosExpecter(vmi *v1.VirtualMachineInstance) (expect.Expecter, error) {
	virtClient, err := kubecli.GetKubevirtClient()
	PanicOnError(err)
	expecter, _, err := NewConsoleExpecter(virtClient, vmi, 10*time.Second)
	if err != nil {
		return nil, err
	}
	vmiName := vmi.Name
	if vmi.Spec.Hostname != "" {
		vmiName = vmi.Spec.Hostname
	}
	b := append([]expect.Batcher{
		&expect.BSnd{S: "\n"},
		&expect.BSnd{S: "\n"},
		&expect.BExp{R: "login as 'cirros' user. default password: 'gocubsgo'. use 'sudo' for root."},
		&expect.BSnd{S: "\n"},
		&expect.BExp{R: vmiName + " login:"},
		&expect.BSnd{S: "cirros\n"},
		&expect.BExp{R: "Password:"},
		&expect.BSnd{S: "gocubsgo\n"},
		&expect.BExp{R: "$"}})
	res, err := expecter.ExpectBatch(b, 180*time.Second)
	log.DefaultLogger().Object(vmi).V(4).Infof("%v", res)
	return expecter, err
}

func LoggedInAlpineExpecter(vmi *v1.VirtualMachineInstance) (expect.Expecter, error) {
	virtClient, err := kubecli.GetKubevirtClient()
	PanicOnError(err)
	expecter, _, err := NewConsoleExpecter(virtClient, vmi, 10*time.Second)
	if err != nil {
		return nil, err
	}
	b := append([]expect.Batcher{
		&expect.BSnd{S: "\n"},
		&expect.BSnd{S: "\n"},
		&expect.BExp{R: "localhost login:"},
		&expect.BSnd{S: "root\n"},
		&expect.BExp{R: "localhost:~#"}})
	res, err := expecter.ExpectBatch(b, 180*time.Second)
	log.DefaultLogger().Object(vmi).V(4).Infof("%v", res)
	return expecter, err
}

type VMIExpecterFactory func(*v1.VirtualMachineInstance) (expect.Expecter, error)

func NewVirtctlCommand(args ...string) *cobra.Command {
	commandline := []string{}
	master := flag.Lookup("master").Value
	if master != nil && master.String() != "" {
		commandline = append(commandline, "--server", master.String())
	}
	kubeconfig := flag.Lookup("kubeconfig").Value
	if kubeconfig != nil && kubeconfig.String() != "" {
		commandline = append(commandline, "--kubeconfig", kubeconfig.String())
	}
	cmd := virtctl.NewVirtctlCommand()
	cmd.SetArgs(append(commandline, args...))
	return cmd
}

func NewRepeatableVirtctlCommand(args ...string) func() error {
	return func() error {
		cmd := NewVirtctlCommand(args...)
		return cmd.Execute()
	}
}

func ExecuteCommandOnPod(virtCli kubecli.KubevirtClient, pod *k8sv1.Pod, containerName string, command []string) (string, error) {
	var (
		stdout bytes.Buffer
		stderr bytes.Buffer
	)

	req := virtCli.CoreV1().RESTClient().Post().
		Resource("pods").
		Name(pod.Name).
		Namespace(pod.Namespace).
		SubResource("exec").
		Param("container", containerName)

	req.VersionedParams(&k8sv1.PodExecOptions{
		Container: containerName,
		Command:   command,
		Stdin:     false,
		Stdout:    true,
		Stderr:    true,
		TTY:       false,
	}, scheme.ParameterCodec)

	config, err := kubecli.GetKubevirtClientConfig()
	if err != nil {
		return "", err
	}

	exec, err := remotecommand.NewSPDYExecutor(config, "POST", req.URL())
	if err != nil {
		return "", err
	}

	err = exec.Stream(remotecommand.StreamOptions{
		Stdout: &stdout,
		Stderr: &stderr,
		Tty:    false,
	})

	if err != nil {
		return "", err
	}

	if stderr.Len() > 0 {
		return "", fmt.Errorf("stderr: %v", stderr.String())
	}

	return stdout.String(), nil
}

func BeforeAll(fn func()) {
	first := true
	BeforeEach(func() {
		if first {
			fn()
			first = false
		}
	})
}

func SkipIfNoWindowsImage(virtClient kubecli.KubevirtClient) {
	windowsPv, err := virtClient.CoreV1().PersistentVolumes().Get(DiskWindows, metav1.GetOptions{})
	if err != nil || (windowsPv.Status.Phase != k8sv1.VolumeAvailable && windowsPv.Status.Phase != k8sv1.VolumeReleased) {
		Skip(fmt.Sprintf("Skip Windows tests that requires PVC %s", DiskWindows))
	} else if windowsPv.Status.Phase == k8sv1.VolumeReleased {
		windowsPv.Spec.ClaimRef = nil
		_, err = virtClient.CoreV1().PersistentVolumes().Update(windowsPv)
		Expect(err).ToNot(HaveOccurred())
	}
}

func SkipIfNoKubectl() {
	if KubeVirtKubectlPath == "" {
		Skip("Skip test that requires kubectl binary")
	}
}

func RunKubectlCommand(args ...string) (string, error) {
	kubeconfig := flag.Lookup("kubeconfig").Value
	if kubeconfig == nil || kubeconfig.String() == "" {
		return "", fmt.Errorf("can not find kubeconfig")
	}

	master := flag.Lookup("master").Value
	if master != nil && master.String() != "" {
		args = append(args, "--server", master.String())
	}

	cmd := exec.Command(KubeVirtKubectlPath, args...)
	kubeconfEnv := fmt.Sprintf("KUBECONFIG=%s", kubeconfig.String())
	cmd.Env = append(os.Environ(), kubeconfEnv)

	stdOutBytes, err := cmd.Output()
	if err != nil {
		return string(stdOutBytes), err
	}
	return string(stdOutBytes), nil
}

func GenerateVMIJson(vmi *v1.VirtualMachineInstance) (string, error) {
	data, err := json.Marshal(vmi)
	if err != nil {
		return "", fmt.Errorf("failed to generate json for vmi %s", vmi.Name)
	}

	jsonFile := fmt.Sprintf("%s.json", vmi.Name)
	err = ioutil.WriteFile(jsonFile, data, 0644)
	if err != nil {
		return "", fmt.Errorf("failed to write json file %s", jsonFile)
	}
	return jsonFile, nil
}

func NotDeleted(vmis *v1.VirtualMachineInstanceList) (notDeleted []v1.VirtualMachineInstance) {
	for _, vmi := range vmis.Items {
		if vmi.DeletionTimestamp == nil {
			notDeleted = append(notDeleted, vmi)
		}
	}
	return
}

func UnfinishedVMIPodSelector(vmi *v1.VirtualMachineInstance) metav1.ListOptions {
	fieldSelector := fields.ParseSelectorOrDie(
		"status.phase!=" + string(k8sv1.PodFailed) +
			",status.phase!=" + string(k8sv1.PodSucceeded))
	labelSelector, err := labels.Parse(fmt.Sprintf(v1.AppLabel+"=virt-launcher,"+v1.DomainLabel+" in (%s)", vmi.GetName()))
	if err != nil {
		panic(err)
	}
	return metav1.ListOptions{FieldSelector: fieldSelector.String(), LabelSelector: labelSelector.String()}
}

// NewHelloWorldJob takes a DNS entry or an IP and a port which it will use create a pod
// which tries to contact the host on the provided port. It expects to receive "Hello World!" to succeed.
func NewHelloWorldJob(host string, port string) *k8sv1.Pod {
	check := []string{fmt.Sprintf(`set -x; x="$(head -n 1 < <(nc %s %s -i 1 -w 1))"; echo "$x" ; if [ "$x" = "Hello World!" ]; then echo "succeeded"; exit 0; else echo "failed"; exit 1; fi`, host, port)}
	job := RenderJob("netcat", []string{"/bin/bash", "-c"}, check)

	return job
}

// NewHelloWorldJobUDP takes a DNS entry or an IP and a port which it will use create a pod
// which tries to contact the host on the provided port. It expects to receive "Hello World!" to succeed.
// Note that in case of UDP, the server will not see the connection unless something is sent over it
// However, netcat does not work well with UDP and closes before the answer arrives, for that another netcat call is needed,
// this time as a UDP listener
func NewHelloWorldJobUDP(host string, port string) *k8sv1.Pod {
	localPort, err := strconv.Atoi(port)
	if err != nil {
		return nil
	}
	// local port is used to catch the reply - any number can be used
	// we make it different than the port to be safe if both are running on the same machine
	localPort--
	check := []string{fmt.Sprintf(`set -x; x="$(head -n 1 < <(echo | nc -up %d %s %s -i 1 -w 1 & nc -ul %d))"; echo "$x" ; if [ "$x" = "Hello UDP World!" ]; then echo "succeeded"; exit 0; else echo "failed"; exit 1; fi`,
		localPort, host, port, localPort)}
	job := RenderJob("netcat", []string{"/bin/bash", "-c"}, check)

	return job
}

func GetNodeWithHugepages(virtClient kubecli.KubevirtClient, hugepages k8sv1.ResourceName) *k8sv1.Node {
	nodes, err := virtClient.Core().Nodes().List(metav1.ListOptions{})
	Expect(err).ToNot(HaveOccurred())

	for _, node := range nodes.Items {
		if v, ok := node.Status.Capacity[hugepages]; ok && !v.IsZero() {
			return &node
		}
	}
	return nil
}
