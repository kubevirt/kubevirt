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
	golog "log"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"regexp"
	"strconv"
	"strings"
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
	k8sversion "k8s.io/apimachinery/pkg/version"

	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/tools/remotecommand"

	"k8s.io/apimachinery/pkg/types"

	"k8s.io/apimachinery/pkg/labels"

	"kubevirt.io/kubevirt/pkg/api/v1"
	"kubevirt.io/kubevirt/pkg/controller"
	"kubevirt.io/kubevirt/pkg/kubecli"
	"kubevirt.io/kubevirt/pkg/log"
	"kubevirt.io/kubevirt/pkg/util/net/dns"
	"kubevirt.io/kubevirt/pkg/virt-controller/services"
	"kubevirt.io/kubevirt/pkg/virtctl"
	vmsgen "kubevirt.io/kubevirt/tools/vms-generator/utils"

	cdiv1 "kubevirt.io/containerized-data-importer/pkg/apis/datavolumecontroller/v1alpha1"
)

var KubeVirtVersionTag = "latest"
var KubeVirtRepoPrefix = "kubevirt"
var KubeVirtKubectlPath = ""
var KubeVirtOcPath = ""
var KubeVirtVirtctlPath = ""
var KubeVirtInstallNamespace = "kube-system"

func init() {
	flag.StringVar(&KubeVirtVersionTag, "tag", "latest", "Set the image tag or digest to use")
	flag.StringVar(&KubeVirtRepoPrefix, "prefix", "kubevirt", "Set the repository prefix for all images")
	flag.StringVar(&KubeVirtKubectlPath, "kubectl-path", "", "Set path to kubectl binary")
	flag.StringVar(&KubeVirtOcPath, "oc-path", "", "Set path to oc binary")
	flag.StringVar(&KubeVirtVirtctlPath, "virtctl-path", "", "Set path to virtctl binary")
	flag.StringVar(&KubeVirtInstallNamespace, "installed-namespace", "kube-system", "Set the namespace KubeVirt is installed in")
}

type EventType string

const (
	AlpineHttpUrl = "http://cdi-http-import-server.kube-system/images/alpine.iso"
)

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

const LocalStorageClass = "local"

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
	osRhel           = "rhel"
	CustomHostPath   = "custom-host-path"
)

const (
	HostPathBase   = "/tmp/hostImages/"
	HostPathAlpine = HostPathBase + "alpine"
	HostPathCustom = HostPathBase + "custom"
)

const (
	DiskAlpineHostPath = "disk-alpine-host-path"
	DiskWindows        = "disk-windows"
	DiskRhel           = "disk-rhel"
	DiskCustomHostPath = "disk-custom-host-path"
	BlockPVCCirros     = "local-block-storage-cirros"
)

const (
	defaultDiskSize        = "1Gi"
	defaultWindowsDiskSize = "30Gi"
	defaultRhelDiskSize    = "15Gi"
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
				log.Log.Reason(fmt.Errorf("unexpected warning event received")).ObjectRef(&event.InvolvedObject).Error(event.Message)
			} else {
				log.Log.ObjectRef(&event.InvolvedObject).Infof(event.Message)
			}
			Expect(event.Type).NotTo(Equal(string(WarningEvent)), "Unexpected Warning event received: %s,%s: %s", event.InvolvedObject.Name, event.InvolvedObject.UID, event.Message)
			return processFunc(event)
		}

	} else {
		f = func(event *k8sv1.Event) bool {
			if event.Type == string(WarningEvent) {
				log.Log.ObjectRef(&event.InvolvedObject).Reason(fmt.Errorf("Warning event received")).Error(event.Message)
			} else {
				log.Log.ObjectRef(&event.InvolvedObject).Infof(event.Message)
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
	done := make(chan struct{})

	go func() {
		defer GinkgoRecover()
		for obj := range eventWatcher.ResultChan() {
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
	DeletePVC(osRhel)

	DeletePVC(osAlpineHostPath)
	DeletePV(osAlpineHostPath)

	removeNamespaces()
}

func BeforeTestCleanup() {
	cleanNamespaces()
}

func BeforeTestSuitSetup() {
	// Forward expecter library output to GinkgoWriter
	golog.SetOutput(GinkgoWriter)

	log.InitializeLogging("tests")
	log.Log.SetIOWriter(GinkgoWriter)

	createNamespaces()
	createServiceAccounts()

	CreateHostPathPv(osAlpineHostPath, HostPathAlpine)
	CreatePVC(osAlpineHostPath, defaultDiskSize)

	CreatePVC(osWindows, defaultWindowsDiskSize)
	CreatePVC(osRhel, defaultRhelDiskSize)

	// create PVC for cirros block device PV, which is provided by local volume provider
	selector := make(map[string]string)
	selector["blockstorage"] = "cirros"
	CreateBlockVolumePVC(BlockPVCCirros, selector, "1Gi")

	EnsureKVMPresent()
}

func EnsureKVMPresent() {
	useEmulation := false
	virtClient, err := kubecli.GetKubevirtClient()
	PanicOnError(err)

	options := metav1.GetOptions{}
	cfgMap, err := virtClient.CoreV1().ConfigMaps("kube-system").Get("kubevirt-config", options)
	if err == nil {
		val, ok := cfgMap.Data["debug.useEmulation"]
		useEmulation = ok && (val == "true")
	} else {
		// If the cfgMap is missing, default to useEmulation=false
		// no other error is expected
		if !errors.IsNotFound(err) {
			ExpectWithOffset(1, err).ToNot(HaveOccurred())
		}
	}
	if !useEmulation {
		listOptions := metav1.ListOptions{LabelSelector: v1.AppLabel + "=virt-handler"}
		virtHandlerPods, err := virtClient.CoreV1().Pods(metav1.NamespaceSystem).List(listOptions)
		ExpectWithOffset(1, err).ToNot(HaveOccurred())

		EventuallyWithOffset(1, func() bool {
			ready := true
			// cluster is not ready until all nodes are ready.
			for _, pod := range virtHandlerPods.Items {
				virtHandlerNode, err := virtClient.CoreV1().Nodes().Get(pod.Spec.NodeName, metav1.GetOptions{})
				ExpectWithOffset(1, err).ToNot(HaveOccurred())

				allocatable, ok := virtHandlerNode.Status.Allocatable[services.KvmDevice]
				ready = ready && ok
				ready = ready && (allocatable.Value() > 0)
			}
			return ready
		}, 120*time.Second, 1*time.Second).Should(BeTrue(),
			"KVM devices are required for testing, but are not present on cluster nodes")
	}
}

func CreateConfigMap(name string, data map[string]string) {
	virtCli, err := kubecli.GetKubevirtClient()
	PanicOnError(err)
	_, err = virtCli.CoreV1().ConfigMaps(NamespaceTestDefault).Create(&k8sv1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{Name: name},
		Data:       data,
	})

	if !errors.IsAlreadyExists(err) {
		PanicOnError(err)
	}
}

func CreateSecret(name string, data map[string]string) {
	virtCli, err := kubecli.GetKubevirtClient()
	PanicOnError(err)

	_, err = virtCli.CoreV1().Secrets(NamespaceTestDefault).Create(&k8sv1.Secret{
		ObjectMeta: metav1.ObjectMeta{Name: name},
		StringData: data,
	})
	if !errors.IsAlreadyExists(err) {
		PanicOnError(err)
	}
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
	storageClass := LocalStorageClass

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
			StorageClassName: &storageClass,
		},
	}
}

func CreateBlockVolumePVC(name string, labelSelector map[string]string, size string) {
	virtCli, err := kubecli.GetKubevirtClient()
	PanicOnError(err)

	_, err = virtCli.CoreV1().PersistentVolumeClaims(NamespaceTestDefault).Create(newBlockVolumePVC(name, labelSelector, size))
	if !errors.IsAlreadyExists(err) {
		PanicOnError(err)
	}
}

func newBlockVolumePVC(name string, labelSelector map[string]string, size string) *k8sv1.PersistentVolumeClaim {
	quantity, err := resource.ParseQuantity(size)
	PanicOnError(err)

	storageClass := LocalStorageClass
	volumeMode := k8sv1.PersistentVolumeBlock

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
				MatchLabels: labelSelector,
			},
			StorageClassName: &storageClass,
			VolumeMode:       &volumeMode,
		},
	}
}

func CreateHostPathPv(osName string, hostPath string) {
	virtCli, err := kubecli.GetKubevirtClient()
	PanicOnError(err)

	quantity, err := resource.ParseQuantity("1Gi")
	PanicOnError(err)

	hostPathType := k8sv1.HostPathDirectoryOrCreate

	name := fmt.Sprintf("%s-disk-for-tests", osName)
	pv := &k8sv1.PersistentVolume{
		ObjectMeta: metav1.ObjectMeta{
			Name: name,
			Labels: map[string]string{
				"kubevirt.io/test": osName,
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
			StorageClassName: LocalStorageClass,
			NodeAffinity: &k8sv1.VolumeNodeAffinity{
				Required: &k8sv1.NodeSelector{
					NodeSelectorTerms: []k8sv1.NodeSelectorTerm{
						{
							MatchExpressions: []k8sv1.NodeSelectorRequirement{
								{
									Key:      "kubernetes.io/hostname",
									Operator: k8sv1.NodeSelectorOpIn,
									Values:   []string{"node01"},
								},
							},
						},
					},
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

func DeleteConfigMap(name string) {
	virtCli, err := kubecli.GetKubevirtClient()
	PanicOnError(err)

	err = virtCli.CoreV1().ConfigMaps(NamespaceTestDefault).Delete(name, nil)
	if !errors.IsNotFound(err) {
		PanicOnError(err)
	}
}

func DeleteSecret(name string) {
	virtCli, err := kubecli.GetKubevirtClient()
	PanicOnError(err)

	err = virtCli.CoreV1().Secrets(NamespaceTestDefault).Delete(name, nil)
	if !errors.IsNotFound(err) {
		PanicOnError(err)
	}
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

func RunVMI(vmi *v1.VirtualMachineInstance, timeout int) *v1.VirtualMachineInstance {
	By("Starting a VirtualMachineInstance")
	virtCli, err := kubecli.GetKubevirtClient()
	PanicOnError(err)

	var obj *v1.VirtualMachineInstance
	Eventually(func() error {
		obj, err = virtCli.VirtualMachineInstance(NamespaceTestDefault).Create(vmi)
		return err
	}, timeout, 1*time.Second).ShouldNot(HaveOccurred())
	return obj
}

func RunVMIAndExpectLaunch(vmi *v1.VirtualMachineInstance, withAuth bool, timeout int) *v1.VirtualMachineInstance {
	obj := RunVMI(vmi, timeout)
	By("Waiting until the VirtualMachineInstance will start")
	WaitForSuccessfulVMIStartWithTimeout(obj, timeout)
	return obj
}

func GetRunningPodByVirtualMachineInstance(vmi *v1.VirtualMachineInstance, namespace string) *k8sv1.Pod {
	virtCli, err := kubecli.GetKubevirtClient()
	PanicOnError(err)

	vmi, err = virtCli.VirtualMachineInstance(namespace).Get(vmi.Name, &metav1.GetOptions{})
	Expect(err).ToNot(HaveOccurred())

	return GetRunningPodByLabel(string(vmi.GetUID()), v1.CreatedByLabel, namespace)
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
		// Remove all limit ranges
		PanicOnError(virtCli.CoreV1().RESTClient().Delete().Namespace(namespace).Resource("limitranges").Do().Error())

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
		EventuallyWithOffset(1, func() bool { return errors.IsNotFound(virtCli.CoreV1().Namespaces().Delete(namespace, nil)) }, 180*time.Second, 1*time.Second).
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

func NewRandomDataVolumeWithHttpImport(imageUrl string, namespace string) *cdiv1.DataVolume {

	name := "test-datavolume-" + rand.String(12)
	storageClassName := "local"
	quantity, err := resource.ParseQuantity("2Gi")
	PanicOnError(err)
	dataVolume := &cdiv1.DataVolume{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
		},
		Spec: cdiv1.DataVolumeSpec{
			Source: cdiv1.DataVolumeSource{
				HTTP: &cdiv1.DataVolumeSourceHTTP{
					URL: imageUrl,
				},
			},
			PVC: &k8sv1.PersistentVolumeClaimSpec{
				AccessModes: []k8sv1.PersistentVolumeAccessMode{k8sv1.ReadWriteOnce},
				Resources: k8sv1.ResourceRequirements{
					Requests: k8sv1.ResourceList{
						"storage": quantity,
					},
				},
				StorageClassName: &storageClassName,
			},
		},
	}

	dataVolume.TypeMeta = metav1.TypeMeta{
		APIVersion: "cdi.kubevirt.io/v1alpha1",
		Kind:       "DataVolume",
	}
	return dataVolume
}

func NewRandomVMI() *v1.VirtualMachineInstance {
	return NewRandomVMIWithNS(NamespaceTestDefault)
}

func NewRandomVMIWithNS(namespace string) *v1.VirtualMachineInstance {
	vmi := v1.NewMinimalVMIWithNS(namespace, "testvmi"+rand.String(48))

	t := defaultTestGracePeriod
	vmi.Spec.TerminationGracePeriodSeconds = &t
	return vmi
}

func NewRandomVMIWithDataVolume(dataVolumeName string) *v1.VirtualMachineInstance {
	vmi := NewRandomVMI()

	vmi.Spec.Domain.Resources.Requests[k8sv1.ResourceMemory] = resource.MustParse("64M")

	diskName := "disk0"
	bus := "virtio"
	vmi.Spec.Domain.Devices.Disks = append(vmi.Spec.Domain.Devices.Disks, v1.Disk{
		Name:       diskName,
		VolumeName: diskName,
		DiskDevice: v1.DiskDevice{
			Disk: &v1.DiskTarget{
				Bus: bus,
			},
		},
	})
	vmi.Spec.Volumes = append(vmi.Spec.Volumes, v1.Volume{
		Name: diskName,
		VolumeSource: v1.VolumeSource{
			DataVolume: &v1.DataVolumeSource{
				Name: dataVolumeName,
			},
		},
	})

	vmi.Spec.Domain.Resources.Requests[k8sv1.ResourceMemory] = resource.MustParse("512M")
	return vmi
}

func NewRandomVMWithDataVolume(imageUrl string, namespace string) *v1.VirtualMachine {
	dataVolume := NewRandomDataVolumeWithHttpImport(imageUrl, namespace)
	vmi := NewRandomVMIWithDataVolume(dataVolume.Name)
	vm := NewRandomVirtualMachine(vmi, false)

	vm.Spec.DataVolumeTemplates = append(vm.Spec.DataVolumeTemplates, *dataVolume)
	return vm
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
	AddUserData(vmi, userData)
	return vmi
}

func AddUserData(vmi *v1.VirtualMachineInstance, userData string) {
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

func NewRandomVMIWithHostDisk(diskPath string, diskType v1.HostDiskType, nodeName string) *v1.VirtualMachineInstance {
	vmi := NewRandomVMI()

	vmi.Spec.Domain.Devices.Disks = append(vmi.Spec.Domain.Devices.Disks, v1.Disk{
		Name:       "host-disk",
		VolumeName: "host-disk",
		DiskDevice: v1.DiskDevice{
			Disk: &v1.DiskTarget{
				Bus: "virtio",
			},
		},
	})
	vmi.Spec.Volumes = append(vmi.Spec.Volumes, v1.Volume{
		Name: "host-disk",
		VolumeSource: v1.VolumeSource{
			HostDisk: &v1.HostDisk{
				Path: diskPath,
				Type: diskType,
			},
		},
	})
	if nodeName != "" {
		vmi.Spec.Affinity = &k8sv1.Affinity{
			NodeAffinity: &k8sv1.NodeAffinity{
				RequiredDuringSchedulingIgnoredDuringExecution: &k8sv1.NodeSelector{
					NodeSelectorTerms: []k8sv1.NodeSelectorTerm{
						{
							MatchExpressions: []k8sv1.NodeSelectorRequirement{
								{
									Key:      "kubernetes.io/hostname",
									Operator: k8sv1.NodeSelectorOpIn,
									Values:   []string{nodeName},
								},
							},
						},
					},
				},
			},
		}
	}

	if diskType == v1.HostDiskExistsOrCreate {
		vmi.Spec.Volumes[0].HostDisk.Capacity = resource.MustParse(defaultDiskSize)
	}

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

func NewRandomVMIWithConfigMap(configMapName string) *v1.VirtualMachineInstance {
	vmi := NewRandomVMIWithPVC(DiskAlpineHostPath)
	AddConfigMapDisk(vmi, configMapName)
	return vmi
}

func AddConfigMapDisk(vmi *v1.VirtualMachineInstance, configMapName string) {
	volumeName := configMapName + "-vol"
	vmi.Spec.Volumes = append(vmi.Spec.Volumes, v1.Volume{
		Name: volumeName,
		VolumeSource: v1.VolumeSource{
			ConfigMap: &v1.ConfigMapVolumeSource{
				LocalObjectReference: k8sv1.LocalObjectReference{
					Name: configMapName,
				},
			},
		},
	})
	vmi.Spec.Domain.Devices.Disks = append(vmi.Spec.Domain.Devices.Disks, v1.Disk{
		Name:       configMapName + "-disk",
		VolumeName: volumeName,
	})
}

func NewRandomVMIWithSecret(secretName string) *v1.VirtualMachineInstance {
	vmi := NewRandomVMIWithPVC(DiskAlpineHostPath)
	AddSecretDisk(vmi, secretName)
	return vmi
}

func AddSecretDisk(vmi *v1.VirtualMachineInstance, secretName string) {
	volumeName := secretName + "-vol"
	vmi.Spec.Volumes = append(vmi.Spec.Volumes, v1.Volume{
		Name: volumeName,
		VolumeSource: v1.VolumeSource{
			Secret: &v1.SecretVolumeSource{
				SecretName: secretName,
			},
		},
	})
	vmi.Spec.Domain.Devices.Disks = append(vmi.Spec.Domain.Devices.Disks, v1.Disk{
		Name:       secretName + "-disk",
		VolumeName: volumeName,
	})
}

func NewRandomVMIWithSlirpInterfaceEphemeralDiskAndUserdata(containerImage string, userData string, Ports []v1.Port) *v1.VirtualMachineInstance {
	vmi := NewRandomVMIWithEphemeralDiskAndUserdata(containerImage, userData)
	vmi.Spec.Domain.Devices.Interfaces = []v1.Interface{{Name: "default", Ports: Ports, InterfaceBindingMethod: v1.InterfaceBindingMethod{Slirp: &v1.InterfaceSlirp{}}}}
	vmi.Spec.Networks = []v1.Network{*v1.DefaultPodNetwork()}

	return vmi
}

func NewRandomVMIWithBridgeInterfaceEphemeralDiskAndUserdata(containerImage string, userData string, Ports []v1.Port) *v1.VirtualMachineInstance {
	vmi := NewRandomVMIWithEphemeralDiskAndUserdata(containerImage, userData)
	vmi.Spec.Domain.Devices.Interfaces = []v1.Interface{{Name: "default", Ports: Ports, InterfaceBindingMethod: v1.InterfaceBindingMethod{Bridge: &v1.InterfaceBridge{}}}}
	vmi.Spec.Networks = []v1.Network{*v1.DefaultPodNetwork()}

	return vmi
}

func AddExplicitPodNetworkInterface(vmi *v1.VirtualMachineInstance) {
	vmi.Spec.Domain.Devices.Interfaces = []v1.Interface{*v1.DefaultNetworkInterface()}
	vmi.Spec.Networks = []v1.Network{*v1.DefaultPodNetwork()}
}

func NewRandomVMIWithe1000NetworkInterface() *v1.VirtualMachineInstance {
	// Use alpine because cirros dhcp client starts prematurily before link is ready
	vmi := NewRandomVMIWithEphemeralDisk(RegistryDiskFor(RegistryDiskAlpine))
	AddExplicitPodNetworkInterface(vmi)
	vmi.Spec.Domain.Devices.Interfaces[0].Model = "e1000"
	return vmi
}

func NewRandomVMIWithCustomMacAddress() *v1.VirtualMachineInstance {
	vmi := NewRandomVMIWithEphemeralDisk(RegistryDiskFor(RegistryDiskAlpine))
	AddExplicitPodNetworkInterface(vmi)
	vmi.Spec.Domain.Devices.Interfaces[0].MacAddress = "de:ad:00:00:be:af"
	return vmi
}

// Block until the specified VirtualMachineInstance started and return the target node name.
func waitForVMIStart(obj runtime.Object, seconds int, ignoreWarnings bool) (nodeName string) {
	vmi, ok := obj.(*v1.VirtualMachineInstance)
	ExpectWithOffset(1, ok).To(BeTrue(), "Object is not of type *v1.VMI")

	virtClient, err := kubecli.GetKubevirtClient()
	ExpectWithOffset(1, err).ToNot(HaveOccurred())

	// Fetch the VirtualMachineInstance, to make sure we have a resourceVersion as a starting point for the watch
	vmi, err = virtClient.VirtualMachineInstance(vmi.Namespace).Get(vmi.Name, &metav1.GetOptions{})
	ExpectWithOffset(1, err).ToNot(HaveOccurred())

	objectEventWatcher := NewObjectEventWatcher(vmi).SinceWatchedObjectResourceVersion().Timeout(time.Duration(seconds) * time.Second)
	if ignoreWarnings != true {
		objectEventWatcher.FailOnWarnings()
	}
	objectEventWatcher.WaitFor(NormalEvent, v1.Started)

	// FIXME the event order is wrong. First the document should be updated
	EventuallyWithOffset(1, func() bool {
		vmi, err = virtClient.VirtualMachineInstance(vmi.Namespace).Get(vmi.Name, &metav1.GetOptions{})
		ExpectWithOffset(1, err).ToNot(HaveOccurred())

		nodeName = vmi.Status.NodeName

		// wait on both phase and graphics
		if vmi.Status.Phase == v1.Running {
			return true
		}
		return false
	}, time.Duration(seconds)*time.Second).Should(Equal(true), "Timed out waiting for VMI to enter Running phase")

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
	ExpectWithOffset(1, err).ToNot(HaveOccurred())
	EventuallyWithOffset(1, func() bool {
		_, err := virtClient.VirtualMachineInstance(vmi.Namespace).Get(vmi.Name, &metav1.GetOptions{})
		return errors.IsNotFound(err)
	}, seconds, 1*time.Second).Should(BeTrue())
}

func WaitForSuccessfulVMIStart(vmi runtime.Object) string {
	return waitForVMIStart(vmi, 90, false)
}

func WaitUntilVMIReady(vmi *v1.VirtualMachineInstance, expecterFactory VMIExpecterFactory) *v1.VirtualMachineInstance {
	// Wait for VirtualMachineInstance start
	WaitForSuccessfulVMIStart(vmi)

	// Fetch the new VirtualMachineInstance with updated status
	virtClient, err := kubecli.GetKubevirtClient()
	vmi, err = virtClient.VirtualMachineInstance(NamespaceTestDefault).Get(vmi.Name, &metav1.GetOptions{})
	ExpectWithOffset(1, err).ToNot(HaveOccurred())

	// Lets make sure that the OS is up by waiting until we can login
	expecter, err := expecterFactory(vmi)
	ExpectWithOffset(1, err).ToNot(HaveOccurred())
	expecter.Close()
	return vmi
}

func WaitUntilVMIReadyWithNamespace(namespace string, vmi *v1.VirtualMachineInstance, expecterFactory VMIExpecterFactory) *v1.VirtualMachineInstance {
	// Wait for VirtualMachineInstance start
	WaitForSuccessfulVMIStart(vmi)

	// Fetch the new VirtualMachineInstance with updated status
	virtClient, err := kubecli.GetKubevirtClient()
	vmi, err = virtClient.VirtualMachineInstance(namespace).Get(vmi.Name, &metav1.GetOptions{})
	ExpectWithOffset(1, err).ToNot(HaveOccurred())

	// Lets make sure that the OS is up by waiting until we can login
	expecter, err := expecterFactory(vmi)
	ExpectWithOffset(1, err).ToNot(HaveOccurred())
	expecter.Close()
	return vmi
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

	startTime := time.Now()
	con, err := virtCli.VirtualMachineInstance(vmi.Namespace).SerialConsole(vmi.Name, timeout)
	if err != nil {
		return nil, nil, err
	}
	timeout = timeout - time.Now().Sub(startTime)

	go func() {
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
			expecterWriter.Close()
			vmiReader.Close()
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

func CheckForTextExpecter(vmi *v1.VirtualMachineInstance, expected []expect.Batcher, wait int) error {
	virtClient, err := kubecli.GetKubevirtClient()
	PanicOnError(err)
	expecter, _, err := NewConsoleExpecter(virtClient, vmi, 30*time.Second)
	if err != nil {
		return err
	}
	defer expecter.Close()

	resp, err := expecter.ExpectBatch(expected, time.Second*time.Duration(wait))
	if err != nil {
		log.DefaultLogger().Object(vmi).Infof("%v", resp)
	}
	return err
}

func LoggedInCirrosExpecter(vmi *v1.VirtualMachineInstance) (expect.Expecter, error) {
	virtClient, err := kubecli.GetKubevirtClient()
	PanicOnError(err)
	expecter, _, err := NewConsoleExpecter(virtClient, vmi, 10*time.Second)
	if err != nil {
		return nil, err
	}
	hostName := dns.SanitizeHostname(vmi)

	// Do not login, if we already logged in
	err = expecter.Send("\n")
	if err != nil {
		expecter.Close()
		return nil, err
	}
	_, _, err = expecter.Expect(regexp.MustCompile(`\$`), 10*time.Second)
	if err == nil {
		return expecter, nil
	}

	b := append([]expect.Batcher{
		&expect.BSnd{S: "\n"},
		&expect.BSnd{S: "\n"},
		&expect.BExp{R: "login as 'cirros' user. default password: 'gocubsgo'. use 'sudo' for root."},
		&expect.BSnd{S: "\n"},
		&expect.BExp{R: hostName + " login:"},
		&expect.BSnd{S: "cirros\n"},
		&expect.BExp{R: "Password:"},
		&expect.BSnd{S: "gocubsgo\n"},
		&expect.BExp{R: "\\$"}})
	resp, err := expecter.ExpectBatch(b, 180*time.Second)
	if err != nil {
		log.DefaultLogger().Object(vmi).Infof("Login: %v", resp)
		expecter.Close()
		return nil, err
	}
	return expecter, nil
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
	if err != nil {
		log.DefaultLogger().Object(vmi).Infof("Login: %v", res)
		expecter.Close()
		return nil, err
	}
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
	stdout, stderr, err := ExecuteCommandOnPodV2(virtCli, pod, containerName, command)

	if err != nil {
		return "", err
	}

	if len(stderr) > 0 {
		return "", fmt.Errorf("stderr: %v", stderr)
	}

	return stdout, nil
}

func ExecuteCommandOnPodV2(virtCli kubecli.KubevirtClient, pod *k8sv1.Pod, containerName string, command []string) (stdout, stderr string, err error) {
	var (
		stdoutBuf bytes.Buffer
		stderrBuf bytes.Buffer
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
		return "", "", err
	}

	exec, err := remotecommand.NewSPDYExecutor(config, "POST", req.URL())
	if err != nil {
		return "", "", err
	}

	err = exec.Stream(remotecommand.StreamOptions{
		Stdout: &stdoutBuf,
		Stderr: &stderrBuf,
		Tty:    false,
	})

	if err != nil {
		return "", "", err
	}

	return stdoutBuf.String(), stderrBuf.String(), nil
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
	if err != nil || windowsPv.Status.Phase == k8sv1.VolumePending || windowsPv.Status.Phase == k8sv1.VolumeFailed {
		Skip(fmt.Sprintf("Skip Windows tests that requires PVC %s", DiskWindows))
	} else if windowsPv.Status.Phase == k8sv1.VolumeReleased {
		windowsPv.Spec.ClaimRef = nil
		_, err = virtClient.CoreV1().PersistentVolumes().Update(windowsPv)
		Expect(err).ToNot(HaveOccurred())
	}
}

func SkipIfNoRhelImage(virtClient kubecli.KubevirtClient) {
	rhelPv, err := virtClient.CoreV1().PersistentVolumes().Get(DiskRhel, metav1.GetOptions{})
	if err != nil || rhelPv.Status.Phase == k8sv1.VolumePending || rhelPv.Status.Phase == k8sv1.VolumeFailed {
		Skip(fmt.Sprintf("Skip RHEL tests that requires PVC %s", DiskRhel))
	} else if rhelPv.Status.Phase == k8sv1.VolumeReleased {
		rhelPv.Spec.ClaimRef = nil
		_, err = virtClient.CoreV1().PersistentVolumes().Update(rhelPv)
		Expect(err).ToNot(HaveOccurred())
	}
}

func SkipIfNoMultusProvider(virtClient kubecli.KubevirtClient) {
	_, err := virtClient.ExtensionsV1beta1().DaemonSets("kube-system").Get("kube-multus-ds-amd64", metav1.GetOptions{})
	if err != nil {
		Skip("Skip multus tests that required multus cni plugin")
	}
}

func SkipIfUseFlannel(virtClient kubecli.KubevirtClient) {
	labelSelector := "app=flannel"
	flannelpod, err := virtClient.CoreV1().Pods(metav1.NamespaceSystem).List(metav1.ListOptions{LabelSelector: labelSelector})
	Expect(err).ToNot(HaveOccurred())
	if len(flannelpod.Items) > 0 {
		Skip("Skip networkpolicy test for flannel network")
	}
}

func SkipIfNotUseNetworkPolicy(virtClient kubecli.KubevirtClient) {
	expectedRes := "openshift-ovs-networkpolicy"
	out, _ := RunCommand("kubectl", "get", "clusternetwork")
	//we don't check the result here, because this cmd is openshift only and will be failed on k8s cluster
	if !strings.Contains(out, expectedRes) {
		Skip("Skip networkpolicy test that require openshift-ovs-networkpolicy plugin")
	}
}

func SkipIfNoCmd(cmdName string) {
	var cmdPath string
	switch strings.ToLower(cmdName) {
	case "oc":
		cmdPath = KubeVirtOcPath
	case "kubectl":
		cmdPath = KubeVirtKubectlPath
	case "virtctl":
		cmdPath = KubeVirtVirtctlPath
	}
	if cmdPath == "" {
		Skip(fmt.Sprintf("Skip test that requires %s binary", cmdName))
	}
}

func RunCommand(cmdName string, args ...string) (string, error) {
	var cmdPath string
	var err error
	var stdOutErrBytes []byte
	switch cmdName = strings.ToLower(cmdName); cmdName {
	case "oc":
		cmdPath = KubeVirtOcPath
	case "kubectl":
		cmdPath = KubeVirtKubectlPath
	case "virtctl":
		cmdPath = KubeVirtVirtctlPath
	}

	if cmdPath == "" {
		return "", fmt.Errorf("no %s binary specified", cmdName)
	}

	kubeconfig := flag.Lookup("kubeconfig").Value
	if kubeconfig == nil || kubeconfig.String() == "" {
		return "", fmt.Errorf("can not find kubeconfig")
	}

	master := flag.Lookup("master").Value
	if master != nil && master.String() != "" {
		args = append(args, "--server", master.String())
	}

	cmd := exec.Command(cmdPath, args...)
	kubeconfEnv := fmt.Sprintf("KUBECONFIG=%s", kubeconfig.String())
	cmd.Env = append(os.Environ(), kubeconfEnv)

	switch cmdName {
	case "oc", "virtctl":
		stdOutErrBytes, err = cmd.CombinedOutput()
	case "kubectl":
		stdOutErrBytes, err = cmd.Output()
	}

	if err != nil {
		log.Log.Reason(err).With("output", string(stdOutErrBytes)).Errorf("%s command failed: %s %s,", cmdName, cmdPath, strings.Join(args, " "))
	}
	return string(stdOutErrBytes), err
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

func GenerateTemplateJson(template *vmsgen.Template) (string, error) {
	data, err := json.Marshal(template)
	if err != nil {
		return "", fmt.Errorf("failed to generate json for vm template %s", template.Name)
	}

	jsonFile := fmt.Sprintf("%s.json", template.Name)
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
	virtClient, err := kubecli.GetKubevirtClient()
	PanicOnError(err)

	vmi, err = virtClient.VirtualMachineInstance(vmi.Namespace).Get(vmi.Name, &metav1.GetOptions{})
	Expect(err).ToNot(HaveOccurred())

	fieldSelector := fields.ParseSelectorOrDie(
		"status.phase!=" + string(k8sv1.PodFailed) +
			",status.phase!=" + string(k8sv1.PodSucceeded))
	labelSelector, err := labels.Parse(fmt.Sprintf(v1.AppLabel + "=virt-launcher," + v1.CreatedByLabel + "=" + string(vmi.GetUID())))
	if err != nil {
		panic(err)
	}
	return metav1.ListOptions{FieldSelector: fieldSelector.String(), LabelSelector: labelSelector.String()}
}

func RemoveHostDiskImage(diskPath string, nodeName string) {
	virtClient, err := kubecli.GetKubevirtClient()
	PanicOnError(err)

	job := newDeleteHostDisksJob(diskPath)
	// remove a disk image from a specific node
	job.Spec.Affinity = &k8sv1.Affinity{
		NodeAffinity: &k8sv1.NodeAffinity{
			RequiredDuringSchedulingIgnoredDuringExecution: &k8sv1.NodeSelector{
				NodeSelectorTerms: []k8sv1.NodeSelectorTerm{
					{
						MatchExpressions: []k8sv1.NodeSelectorRequirement{
							{
								Key:      "kubernetes.io/hostname",
								Operator: k8sv1.NodeSelectorOpIn,
								Values:   []string{nodeName},
							},
						},
					},
				},
			},
		},
	}
	job, err = virtClient.CoreV1().Pods(NamespaceTestDefault).Create(job)
	PanicOnError(err)

	getStatus := func() k8sv1.PodPhase {
		pod, err := virtClient.CoreV1().Pods(NamespaceTestDefault).Get(job.Name, metav1.GetOptions{})
		Expect(err).ToNot(HaveOccurred())
		return pod.Status.Phase
	}
	Eventually(getStatus, 30, 1).Should(Equal(k8sv1.PodSucceeded))
}

func CreateHostDiskImage(diskPath string) *k8sv1.Pod {
	hostPathType := k8sv1.HostPathDirectoryOrCreate
	dir := filepath.Dir(diskPath)

	args := []string{fmt.Sprintf(`dd if=/dev/zero of=%s bs=1 count=0 seek=1G && ls -l %s`, diskPath, dir)}
	job := renderHostPathJob("hostdisk-create-job", dir, hostPathType, []string{"/bin/bash", "-c"}, args)

	return job
}

func newDeleteHostDisksJob(diskPath string) *k8sv1.Pod {
	hostPathType := k8sv1.HostPathDirectoryOrCreate

	args := []string{fmt.Sprintf(`rm -f %s`, diskPath)}
	job := renderHostPathJob("hostdisk-delete-job", filepath.Dir(diskPath), hostPathType, []string{"/bin/bash", "-c"}, args)

	return job
}

func renderHostPathJob(jobName string, dir string, hostPathType k8sv1.HostPathType, cmd []string, args []string) *k8sv1.Pod {
	job := RenderJob(jobName, cmd, args)
	job.Spec.Containers[0].VolumeMounts = append(job.Spec.Containers[0].VolumeMounts, k8sv1.VolumeMount{
		Name:      "hostpath-mount",
		MountPath: dir,
	})
	job.Spec.Volumes = append(job.Spec.Volumes, k8sv1.Volume{
		Name: "hostpath-mount",
		VolumeSource: k8sv1.VolumeSource{
			HostPath: &k8sv1.HostPathVolumeSource{
				Path: dir,
				Type: &hostPathType,
			},
		},
	})

	return job
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
	ExpectWithOffset(1, err).ToNot(HaveOccurred())

	for _, node := range nodes.Items {
		if v, ok := node.Status.Capacity[hugepages]; ok && !v.IsZero() {
			return &node
		}
	}
	return nil
}

func GetAllSchedulableNodes(virtClient kubecli.KubevirtClient) *k8sv1.NodeList {
	nodes, err := virtClient.CoreV1().Nodes().List(metav1.ListOptions{LabelSelector: "kubevirt.io/schedulable=true"})
	Expect(err).ToNot(HaveOccurred(), "Should list compute nodes")
	return nodes
}

// SkipIfVersionBelow will skip tests if it runs on an environment with k8s version below specified
func SkipIfVersionBelow(message string, expectedVersion string) {
	virtClient, err := kubecli.GetKubevirtClient()
	PanicOnError(err)

	response, err := virtClient.RestClient().Get().AbsPath("/version").DoRaw()
	ExpectWithOffset(1, err).ToNot(HaveOccurred())

	var info k8sversion.Info

	err = json.Unmarshal(response, &info)
	ExpectWithOffset(1, err).ToNot(HaveOccurred())

	curVersion := strings.Split(info.GitVersion, "+")[0]
	curVersion = strings.Trim(curVersion, "v")

	if curVersion < expectedVersion {
		Skip(message)
	}
}

func SkipIfOpenShift(message string) {
	virtClient, err := kubecli.GetKubevirtClient()
	PanicOnError(err)

	result := virtClient.RestClient().Get().AbsPath("/version/openshift").Do()

	if result.Error() == nil {
		Skip("Openshift detected: " + message)
	}
}

// StartVmOnNode starts a VMI on the specified node
func StartVmOnNode(vmi *v1.VirtualMachineInstance, nodeName string) {
	virtClient, err := kubecli.GetKubevirtClient()
	PanicOnError(err)

	vmi.Spec.Affinity = &k8sv1.Affinity{
		NodeAffinity: &k8sv1.NodeAffinity{
			RequiredDuringSchedulingIgnoredDuringExecution: &k8sv1.NodeSelector{
				NodeSelectorTerms: []k8sv1.NodeSelectorTerm{
					{
						MatchExpressions: []k8sv1.NodeSelectorRequirement{
							{Key: "kubernetes.io/hostname", Operator: k8sv1.NodeSelectorOpIn, Values: []string{nodeName}},
						},
					},
				},
			},
		},
	}

	_, err = virtClient.VirtualMachineInstance(NamespaceTestDefault).Create(vmi)
	ExpectWithOffset(1, err).ToNot(HaveOccurred())
	WaitForSuccessfulVMIStart(vmi)
}

// RunCommandOnVmiPod runs specified command on the virt-launcher pod
func RunCommandOnVmiPod(vmi *v1.VirtualMachineInstance, command []string) string {
	virtClient, err := kubecli.GetKubevirtClient()
	PanicOnError(err)

	pods, err := virtClient.CoreV1().Pods(NamespaceTestDefault).List(UnfinishedVMIPodSelector(vmi))
	ExpectWithOffset(1, err).ToNot(HaveOccurred())
	ExpectWithOffset(1, pods.Items).NotTo(BeEmpty())
	vmiPod := pods.Items[0]

	output, err := ExecuteCommandOnPod(
		virtClient,
		&vmiPod,
		"compute",
		command,
	)
	ExpectWithOffset(1, err).ToNot(HaveOccurred())

	return output
}

// GetNodeLibvirtCapabilities returns node libvirt capabilities
func GetNodeLibvirtCapabilities(nodeName string) string {
	// Create a virt-launcher pod to fetch virsh capabilities
	vmi := NewRandomVMIWithEphemeralDiskAndUserdata(RegistryDiskFor(RegistryDiskCirros), "#!/bin/bash\necho 'hello'\n")
	StartVmOnNode(vmi, nodeName)

	return RunCommandOnVmiPod(vmi, []string{"virsh", "-r", "capabilities"})
}

// GetNodeCPUInfo returns output of lscpu on the pod that runs on the specified node
func GetNodeCPUInfo(nodeName string) string {
	vmi := NewRandomVMIWithEphemeralDiskAndUserdata(RegistryDiskFor(RegistryDiskCirros), "#!/bin/bash\necho 'hello'\n")
	StartVmOnNode(vmi, nodeName)

	return RunCommandOnVmiPod(vmi, []string{"lscpu"})
}

// KubevirtFailHandler call ginkgo.Fail with printing the additional information
func KubevirtFailHandler(message string, callerSkip ...int) {
	if len(callerSkip) > 0 {
		callerSkip[0]++
	}

	virtClient, err := kubecli.GetKubevirtClient()
	if err != nil {
		fmt.Println(err)
		Fail(message, callerSkip...)
		return
	}

	for _, ns := range []string{metav1.NamespaceSystem, NamespaceTestDefault} {
		// Get KubeVirt specific pods information
		pods, err := virtClient.CoreV1().Pods(ns).List(metav1.ListOptions{LabelSelector: "kubevirt.io"})
		if err != nil {
			fmt.Println(err)
			Fail(message, callerSkip...)
			return
		}

		for _, pod := range pods.Items {
			fmt.Printf("\nPod name: %s\t Pod phase: %s\n\n", pod.Name, pod.Status.Phase)
			var tailLines int64 = 15
			var containerName = ""
			if strings.HasPrefix(pod.Name, "virt-launcher") {
				containerName = "compute"
			}
			logsRaw, err := virtClient.CoreV1().Pods(ns).GetLogs(
				pod.Name, &k8sv1.PodLogOptions{
					TailLines: &tailLines,
					Container: containerName,
				},
			).DoRaw()
			if err == nil {
				fmt.Printf(string(logsRaw))
			}
		}

		vmis, err := virtClient.VirtualMachineInstance(ns).List(&metav1.ListOptions{})
		if err != nil {
			fmt.Println(err)
			Fail(message, callerSkip...)
			return
		}

		for _, vmi := range vmis.Items {
			fmt.Printf("%v\n", vmi)
		}
	}
	Fail(message, callerSkip...)
}

func NewRandomVirtualMachine(vmi *v1.VirtualMachineInstance, running bool) *v1.VirtualMachine {
	name := vmi.Name
	namespace := vmi.Namespace
	vm := &v1.VirtualMachine{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
		},
		Spec: v1.VirtualMachineSpec{
			Running: running,
			Template: &v1.VirtualMachineInstanceTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels:    map[string]string{"name": name},
					Name:      name,
					Namespace: namespace,
				},
				Spec: vmi.Spec,
			},
		},
	}
	return vm
}

func StopVirtualMachine(vm *v1.VirtualMachine) *v1.VirtualMachine {
	By("Stopping the VirtualMachineInstance")
	virtClient, err := kubecli.GetKubevirtClient()
	PanicOnError(err)
	Eventually(func() error {
		updatedVM, err := virtClient.VirtualMachine(vm.Namespace).Get(vm.Name, &metav1.GetOptions{})
		Expect(err).ToNot(HaveOccurred())
		updatedVM.Spec.Running = false
		_, err = virtClient.VirtualMachine(updatedVM.Namespace).Update(updatedVM)
		return err
	}, 300*time.Second, 1*time.Second).ShouldNot(HaveOccurred())
	updatedVM, err := virtClient.VirtualMachine(vm.Namespace).Get(vm.Name, &metav1.GetOptions{})
	Expect(err).ToNot(HaveOccurred())
	// Observe the VirtualMachineInstance deleted
	Eventually(func() bool {
		_, err = virtClient.VirtualMachineInstance(updatedVM.Namespace).Get(updatedVM.Name, &metav1.GetOptions{})
		if errors.IsNotFound(err) {
			return true
		}
		return false
	}, 300*time.Second, 1*time.Second).Should(BeTrue(), "The vmi did not disappear")
	By("VM has not the running condition")
	Eventually(func() bool {
		vm, err := virtClient.VirtualMachine(updatedVM.Namespace).Get(updatedVM.Name, &metav1.GetOptions{})
		Expect(err).ToNot(HaveOccurred())
		return vm.Status.Ready
	}, 300*time.Second, 1*time.Second).Should(BeFalse())
	return updatedVM
}
func StartVirtualMachine(vm *v1.VirtualMachine) *v1.VirtualMachine {
	By("Starting the VirtualMachineInstance")
	virtClient, err := kubecli.GetKubevirtClient()
	PanicOnError(err)
	Eventually(func() error {
		updatedVM, err := virtClient.VirtualMachine(vm.Namespace).Get(vm.Name, &metav1.GetOptions{})
		Expect(err).ToNot(HaveOccurred())
		updatedVM.Spec.Running = true
		_, err = virtClient.VirtualMachine(updatedVM.Namespace).Update(updatedVM)
		return err
	}, 300*time.Second, 1*time.Second).ShouldNot(HaveOccurred())
	updatedVM, err := virtClient.VirtualMachine(vm.Namespace).Get(vm.Name, &metav1.GetOptions{})
	Expect(err).ToNot(HaveOccurred())
	// Observe the VirtualMachineInstance created
	Eventually(func() error {
		_, err := virtClient.VirtualMachineInstance(updatedVM.Namespace).Get(updatedVM.Name, &metav1.GetOptions{})
		return err
	}, 300*time.Second, 1*time.Second).Should(Succeed())
	By("VMI has the running condition")
	Eventually(func() bool {
		vm, err := virtClient.VirtualMachine(updatedVM.Namespace).Get(updatedVM.Name, &metav1.GetOptions{})
		Expect(err).ToNot(HaveOccurred())
		return vm.Status.Ready
	}, 300*time.Second, 1*time.Second).Should(BeTrue())
	return updatedVM
}

func HasCDI() bool {
	hasCDI := false
	virtClient, err := kubecli.GetKubevirtClient()
	PanicOnError(err)
	options := metav1.GetOptions{}
	cfgMap, err := virtClient.CoreV1().ConfigMaps("kube-system").Get("kubevirt-config", options)
	if err == nil {
		val, ok := cfgMap.Data["feature-gates"]
		if !ok {
			return hasCDI
		}
		hasCDI = strings.Contains(val, "DataVolumes")
	} else {
		if !errors.IsNotFound(err) {
			PanicOnError(err)
		}
	}
	return hasCDI
}
