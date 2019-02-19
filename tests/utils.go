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
	goerrors "errors"
	"flag"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"regexp"
	"strconv"
	"strings"
	"time"

	ghodssyaml "github.com/ghodss/yaml"
	expect "github.com/google/goexpect"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/spf13/cobra"
	k8sv1 "k8s.io/api/core/v1"
	k8sextv1beta1 "k8s.io/api/extensions/v1beta1"
	rbacv1 "k8s.io/api/rbac/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/meta"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/fields"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/rand"
	"k8s.io/apimachinery/pkg/util/yaml"
	k8sversion "k8s.io/apimachinery/pkg/version"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/tools/remotecommand"

	cdiv1 "kubevirt.io/containerized-data-importer/pkg/apis/core/v1alpha1"
	v1 "kubevirt.io/kubevirt/pkg/api/v1"
	"kubevirt.io/kubevirt/pkg/controller"
	"kubevirt.io/kubevirt/pkg/kubecli"
	"kubevirt.io/kubevirt/pkg/log"
	"kubevirt.io/kubevirt/pkg/util/net/dns"
	virtconfig "kubevirt.io/kubevirt/pkg/virt-config"
	"kubevirt.io/kubevirt/pkg/virt-controller/services"
	"kubevirt.io/kubevirt/pkg/virtctl"
	vmsgen "kubevirt.io/kubevirt/tools/vms-generator/utils"
)

var KubeVirtVersionTag = "latest"
var KubeVirtRepoPrefix = "kubevirt"
var ContainerizedDataImporterNamespace = "cdi"
var KubeVirtKubectlPath = ""
var KubeVirtOcPath = ""
var KubeVirtVirtctlPath = ""
var KubeVirtInstallNamespace string

var DeployTestingInfrastructureFlag = false
var PathToTestingInfrastrucureManifests = ""

func init() {
	flag.StringVar(&KubeVirtVersionTag, "container-tag", "latest", "Set the image tag or digest to use")
	flag.StringVar(&KubeVirtRepoPrefix, "container-prefix", "kubevirt", "Set the repository prefix for all images")
	flag.StringVar(&ContainerizedDataImporterNamespace, "cdi-namespace", "cdi", "Set the repository prefix for CDI components")
	flag.StringVar(&KubeVirtKubectlPath, "kubectl-path", "", "Set path to kubectl binary")
	flag.StringVar(&KubeVirtOcPath, "oc-path", "", "Set path to oc binary")
	flag.StringVar(&KubeVirtVirtctlPath, "virtctl-path", "", "Set path to virtctl binary")
	flag.StringVar(&KubeVirtInstallNamespace, "installed-namespace", "kubevirt", "Set the namespace KubeVirt is installed in")
	flag.BoolVar(&DeployTestingInfrastructureFlag, "deploy-testing-infra", false, "Deploy testing infrastructure if set")
	flag.StringVar(&PathToTestingInfrastrucureManifests, "path-to-testing-infra-manifests", "manifests/testing", "Set path to testing infrastructure manifests")
}

type EventType string

const TempDirPrefix = "kubevirt-test"

const (
	defaultEventuallyTimeout         = 5 * time.Second
	defaultEventuallyPollingInterval = 1 * time.Second
)

const (
	AlpineHttpUrl     = "http://cdi-http-import-server.kubevirt/images/alpine.iso"
	GuestAgentHttpUrl = "http://cdi-http-import-server.kubevirt/qemu-ga"
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

const (
	StorageClassLocal       = "local"
	StorageClassHostPath    = "host-path"
	StorageClassBlockVolume = "block-volume"
	StorageClassRhel        = "rhel"
	StorageClassWindows     = "windows"
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
	object                 runtime.Object
	timeout                *time.Duration
	failOnWarnings         bool
	resourceVersion        string
	startType              startType
	dontFailOnMissingEvent bool
	abort                  chan struct{}
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

func (w *ObjectEventWatcher) Watch(abortChan chan struct{}, processFunc ProcessFunc) {
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
			msg := fmt.Sprintf("Event(%#v): type: '%v' reason: '%v' %v", event.InvolvedObject, event.Type, event.Reason, event.Message)
			if event.Type == string(WarningEvent) {
				log.Log.Reason(fmt.Errorf("unexpected warning event received")).ObjectRef(&event.InvolvedObject).Error(msg)
			} else {
				log.Log.ObjectRef(&event.InvolvedObject).Info(msg)
			}
			ExpectWithOffset(1, event.Type).NotTo(Equal(string(WarningEvent)), "Unexpected Warning event received: %s,%s: %s", event.InvolvedObject.Name, event.InvolvedObject.UID, event.Message)
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
		case <-w.abort:
		case <-time.After(*w.timeout):
			if !w.dontFailOnMissingEvent {
				Fail(fmt.Sprintf("Waited for %v seconds on the event stream to match a specific event", w.timeout.Seconds()), 1)
			}
		}
	} else {
		select {
		case <-done:
		case <-w.abort:
		}
	}
}

func (w *ObjectEventWatcher) WaitFor(stopChan chan struct{}, eventType EventType, reason interface{}) (e *k8sv1.Event) {
	w.Watch(stopChan, func(event *k8sv1.Event) bool {
		if event.Type == string(eventType) && event.Reason == reflect.ValueOf(reason).String() {
			e = event
			return true
		}
		return false
	})
	return
}

func (w *ObjectEventWatcher) WaitNotFor(stopChan chan struct{}, eventType EventType, reason interface{}) (e *k8sv1.Event) {
	w.dontFailOnMissingEvent = true
	w.Watch(stopChan, func(event *k8sv1.Event) bool {
		if event.Type == string(eventType) && event.Reason == reflect.ValueOf(reason).String() {
			e = event
			Fail(fmt.Sprintf("Did not expect %s with reason %s", string(eventType), reflect.ValueOf(reason).String()), 1)
			return true
		}
		return false
	})
	return
}

// Do scale and retuns error, replicas-before.
func DoScaleDeployment(namespace string, name string, desired int32) (error, int32) {
	virtCli, err := kubecli.GetKubevirtClient()
	PanicOnError(err)

	deployment, err := virtCli.ExtensionsV1beta1().Deployments(namespace).Get(name, metav1.GetOptions{})
	if err != nil {
		return err, -1
	}
	scale := &k8sextv1beta1.Scale{Spec: k8sextv1beta1.ScaleSpec{Replicas: desired}, ObjectMeta: metav1.ObjectMeta{Name: name, Namespace: namespace}}
	scale, err = virtCli.ExtensionsV1beta1().Deployments(namespace).UpdateScale(name, scale)
	if err != nil {
		return err, -1
	}
	return nil, *deployment.Spec.Replicas
}

func DoScaleVirtHandler(namespace string, name string, selector map[string]string) (int32, map[string]string, int64, error) {
	virtCli, err := kubecli.GetKubevirtClient()
	PanicOnError(err)

	d, err := virtCli.ExtensionsV1beta1().DaemonSets(namespace).Get(name, metav1.GetOptions{})
	if err != nil {
		return 0, nil, 0, err
	}
	sel := d.Spec.Template.Spec.NodeSelector
	ready := d.Status.DesiredNumberScheduled
	d.Spec.Template.Spec.NodeSelector = selector
	d, err = virtCli.ExtensionsV1beta1().DaemonSets(namespace).Update(d)
	if err != nil {
		return 0, nil, 0, err
	}
	return ready, sel, d.ObjectMeta.Generation, nil
}

func WaitForAllPodsReady(timeout time.Duration, listOptions metav1.ListOptions) {
	checkForPodsToBeReady := func() []string {
		podsNotReady := make([]string, 0)
		virtClient, err := kubecli.GetKubevirtClient()
		PanicOnError(err)

		podsList, err := virtClient.CoreV1().Pods(k8sv1.NamespaceAll).List(listOptions)
		PanicOnError(err)
		for _, pod := range podsList.Items {
			for _, status := range pod.Status.ContainerStatuses {
				if status.State.Terminated != nil {
					break // We don't care about terminated pods
				} else if status.State.Running != nil {
					if !status.Ready { // We need to wait for this one
						podsNotReady = append(podsNotReady, pod.Name)
						break
					}
				} else {
					// It is in Waiting state, We need to wait for this one
					podsNotReady = append(podsNotReady, pod.Name)
					break
				}
			}
		}
		return podsNotReady
	}
	Eventually(checkForPodsToBeReady, timeout, 2*time.Second).Should(BeEmpty(), "The are pods in system which are not ready.")
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

	if DeployTestingInfrastructureFlag {
		WipeTestingInfrastructure()
	}
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
	if DeployTestingInfrastructureFlag {
		WipeTestingInfrastructure()
		DeployTestingInfrastructure()
	}

	CreateHostPathPv(osAlpineHostPath, HostPathAlpine)
	CreateHostPathPVC(osAlpineHostPath, defaultDiskSize)

	CreatePVC(osWindows, defaultWindowsDiskSize, StorageClassWindows)
	CreatePVC(osRhel, defaultRhelDiskSize, StorageClassRhel)

	EnsureKVMPresent()

	SetDefaultEventuallyTimeout(defaultEventuallyTimeout)
	SetDefaultEventuallyPollingInterval(defaultEventuallyPollingInterval)
}

func EnsureKVMPresent() {
	useEmulation := false
	virtClient, err := kubecli.GetKubevirtClient()
	PanicOnError(err)

	options := metav1.GetOptions{}
	cfgMap, err := virtClient.CoreV1().ConfigMaps(KubeVirtInstallNamespace).Get("kubevirt-config", options)
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
		virtHandlerPods, err := virtClient.CoreV1().Pods(KubeVirtInstallNamespace).List(listOptions)
		ExpectWithOffset(1, err).ToNot(HaveOccurred())

		EventuallyWithOffset(1, func() bool {
			ready := true
			// cluster is not ready until all nodes are ready.
			for _, pod := range virtHandlerPods.Items {
				virtHandlerNode, err := virtClient.CoreV1().Nodes().Get(pod.Spec.NodeName, metav1.GetOptions{})
				ExpectWithOffset(1, err).ToNot(HaveOccurred())

				kvmAllocatable, ok := virtHandlerNode.Status.Allocatable[services.KvmDevice]
				vhostNetAllocatable, ok := virtHandlerNode.Status.Allocatable[services.VhostNetDevice]
				ready = ready && ok
				ready = ready && (kvmAllocatable.Value() > 0) && (vhostNetAllocatable.Value() > 0)
			}
			return ready
		}, 120*time.Second, 1*time.Second).Should(BeTrue(),
			"Both KVM devices and vhost-net devices are required for testing, but are not present on cluster nodes")
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

func CreateHostPathPVC(os, size string) {
	CreatePVC(os, size, StorageClassHostPath)
}

func CreatePVC(os, size, storageClass string) {
	virtCli, err := kubecli.GetKubevirtClient()
	PanicOnError(err)

	_, err = virtCli.CoreV1().PersistentVolumeClaims(NamespaceTestDefault).Create(newPVC(os, size, storageClass))
	if !errors.IsAlreadyExists(err) {
		PanicOnError(err)
	}
}

func newPVC(os, size, storageClass string) *k8sv1.PersistentVolumeClaim {
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
			StorageClassName: &storageClass,
		},
	}
}

func CreateHostPathPv(osName string, hostPath string) {
	CreateHostPathPvWithSize(osName, hostPath, "1Gi")
}

func CreateHostPathPvWithSize(osName string, hostPath string, size string) {
	virtCli, err := kubecli.GetKubevirtClient()
	PanicOnError(err)

	quantity, err := resource.ParseQuantity(size)
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
			StorageClassName: StorageClassHostPath,
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

func GetListOfManifests(pathToManifestsDir string) []string {
	var manifests []string
	isOpenshift := IsOpenShift()
	matchFileName := func(pattern, filename string) bool {
		match, err := filepath.Match(pattern, filename)
		if err != nil {
			panic(err)
		}
		return match
	}
	err := filepath.Walk(pathToManifestsDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			fmt.Printf("ERROR: Can not access a path %q: %v\n", path, err)
			return err
		}
		if !info.IsDir() {
			if matchFileName("*-for-ocp.yaml", info.Name()) {
				if isOpenshift {
					manifests = append(manifests, path)
				}
			} else if matchFileName("*-for-k8s.yaml", info.Name()) {
				if !isOpenshift {
					manifests = append(manifests, path)
				}
			} else if matchFileName("*.yaml", info.Name()) {
				manifests = append(manifests, path)
			}
		}
		return nil
	})
	if err != nil {
		fmt.Printf("ERROR: Walking the path %q: %v\n", pathToManifestsDir, err)
		panic(err)
	}
	return manifests
}

func ReadManifestYamlFile(pathToManifest string) []unstructured.Unstructured {
	var objects []unstructured.Unstructured
	stream, err := os.Open(pathToManifest)
	PanicOnError(err)

	decoder := yaml.NewYAMLOrJSONDecoder(stream, 1024)
	for {
		obj := map[string]interface{}{}
		err := decoder.Decode(&obj)
		if err == io.EOF {
			break
		} else if err != nil {
			panic(err)
		}
		if len(obj) == 0 {
			continue
		}
		objects = append(objects, unstructured.Unstructured{Object: obj})
	}
	return objects
}

func isNamespaceScoped(kind schema.GroupVersionKind) bool {
	switch kind.Kind {
	case "ClusterRole", "ClusterRoleBinding":
		return false
	}
	return true
}

func IsOpenShift() bool {
	virtClient, err := kubecli.GetKubevirtClient()
	PanicOnError(err)

	result := virtClient.RestClient().Get().AbsPath("/version/openshift").Do()

	var statusCode int
	result.StatusCode(&statusCode)

	if result.Error() == nil {
		// It is OpenShift
		if statusCode == http.StatusOK {
			return true
		}
	} else {
		// Got 404 so this is not Openshift
		if statusCode == http.StatusNotFound {
			return false
		}
	}
	fmt.Printf(fmt.Sprintf("ERROR: Can not determine cluster type %#v\n", result))
	panic(err)
}

func composeResourceURI(object unstructured.Unstructured) string {
	uri := "/api"
	if object.GetAPIVersion() != "v1" {
		uri += "s"
	}
	uri += "/" + object.GetAPIVersion()
	if object.GetNamespace() != "" && isNamespaceScoped(object.GroupVersionKind()) {
		uri += "/namespaces/" + object.GetNamespace()
	}
	uri += "/" + strings.ToLower(object.GetKind())
	if !strings.HasSuffix(object.GetKind(), "s") {
		uri += "s"
	}
	return uri
}

func ApplyRawManifest(object unstructured.Unstructured) error {
	virtCli, err := kubecli.GetKubevirtClient()
	PanicOnError(err)

	uri := composeResourceURI(object)
	jsonbody, err := object.MarshalJSON()
	PanicOnError(err)
	b, err := virtCli.CoreV1().RESTClient().Post().RequestURI(uri).Body(jsonbody).DoRaw()
	if err != nil {
		fmt.Printf(fmt.Sprintf("ERROR: Can not apply %s\n", object))
		panic(err)
	}
	status := unstructured.Unstructured{}
	return json.Unmarshal(b, &status)
}

func DeleteRawManifest(object unstructured.Unstructured) error {
	virtCli, err := kubecli.GetKubevirtClient()
	PanicOnError(err)

	uri := composeResourceURI(object)
	uri = uri + "/" + object.GetName()
	result := virtCli.CoreV1().RESTClient().Delete().RequestURI(uri).Do()
	if result.Error() != nil && !errors.IsNotFound(result.Error()) {
		fmt.Printf(fmt.Sprintf("ERROR: Can not delete %s err: %#v %s\n", object.GetName(), result.Error(), object))
		panic(err)
	}
	return nil
}

func deployOrWipeTestingInfrastrucure(actionOnObject func(unstructured.Unstructured) error) {
	// Scale down KubeVirt
	err, replicasApi := DoScaleDeployment(KubeVirtInstallNamespace, "virt-api", 0)
	PanicOnError(err)
	err, replicasController := DoScaleDeployment(KubeVirtInstallNamespace, "virt-controller", 0)
	PanicOnError(err)
	daemonInstances, selector, _, err := DoScaleVirtHandler(KubeVirtInstallNamespace, "virt-handler", map[string]string{"kubevirt.io": "scaletozero"})
	PanicOnError(err)
	// Deploy / delete test infrastructure / dependencies
	manifests := GetListOfManifests(PathToTestingInfrastrucureManifests)
	for _, manifest := range manifests {
		objects := ReadManifestYamlFile(manifest)
		for _, obj := range objects {
			err := actionOnObject(obj)
			PanicOnError(err)
		}
	}
	// Scale KubeVirt back
	err, _ = DoScaleDeployment(KubeVirtInstallNamespace, "virt-api", replicasApi)
	PanicOnError(err)
	err, _ = DoScaleDeployment(KubeVirtInstallNamespace, "virt-controller", replicasController)
	PanicOnError(err)
	_, _, newGeneration, err := DoScaleVirtHandler(KubeVirtInstallNamespace, "virt-handler", selector)
	virtCli, err := kubecli.GetKubevirtClient()
	PanicOnError(err)

	Eventually(func() int32 {
		d, err := virtCli.ExtensionsV1beta1().Deployments(KubeVirtInstallNamespace).Get("virt-api", metav1.GetOptions{})
		Expect(err).ToNot(HaveOccurred())
		return d.Status.ReadyReplicas
	}, 3*time.Minute, 2*time.Second).Should(Equal(replicasApi), "virt-api is not ready")

	Eventually(func() int32 {
		d, err := virtCli.ExtensionsV1beta1().Deployments(KubeVirtInstallNamespace).Get("virt-controller", metav1.GetOptions{})
		Expect(err).ToNot(HaveOccurred())
		return d.Status.ReadyReplicas
	}, 3*time.Minute, 2*time.Second).Should(Equal(replicasController), "virt-controller is not ready")

	Eventually(func() int64 {
		d, err := virtCli.ExtensionsV1beta1().DaemonSets(KubeVirtInstallNamespace).Get("virt-handler", metav1.GetOptions{})
		Expect(err).ToNot(HaveOccurred())
		return d.Status.ObservedGeneration
	}, 1*time.Minute, 2*time.Second).Should(Equal(newGeneration), "virt-handler did not bump the generation")

	Eventually(func() int32 {
		d, err := virtCli.ExtensionsV1beta1().DaemonSets(KubeVirtInstallNamespace).Get("virt-handler", metav1.GetOptions{})
		Expect(err).ToNot(HaveOccurred())
		return d.Status.NumberAvailable
	}, 1*time.Minute, 2*time.Second).Should(Equal(daemonInstances), "virt-handler is not ready")

	WaitForAllPodsReady(3*time.Minute, metav1.ListOptions{})
}

func DeployTestingInfrastructure() {
	deployOrWipeTestingInfrastrucure(ApplyRawManifest)
}

func WipeTestingInfrastructure() {
	deployOrWipeTestingInfrastrucure(DeleteRawManifest)
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
			if status.Name == "kubevirt-infra" {
				ready = status.Ready
				break
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

		//Remove all HPA
		PanicOnError(virtCli.AutoscalingV1().RESTClient().Delete().Namespace(namespace).Resource("horizontalpodautoscalers").Do().Error())

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

		// Remove all Migration Objects
		PanicOnError(virtCli.RestClient().Delete().Namespace(namespace).Resource("virtualmachineinstancemigrations").Do().Error())

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
		Name: diskName,
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

func NewRandomVMIWithEFIBootloader() *v1.VirtualMachineInstance {
	vmi := NewRandomVMIWithEphemeralDiskHighMemory(ContainerDiskFor(ContainerDiskAlpine))

	// EFI needs more memory than other images
	vmi.Spec.Domain.Resources.Requests[k8sv1.ResourceMemory] = resource.MustParse("1Gi")
	vmi.Spec.Domain.Firmware = &v1.Firmware{
		Bootloader: &v1.Bootloader{
			EFI: &v1.EFI{},
		},
	}

	return vmi

}

func NewRandomMigration(vmiName string, namespace string) *v1.VirtualMachineInstanceMigration {
	migration := &v1.VirtualMachineInstanceMigration{

		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-migration-" + rand.String(30),
			Namespace: namespace,
		},
		Spec: v1.VirtualMachineInstanceMigrationSpec{
			VMIName: vmiName,
		},
	}
	migration.TypeMeta = metav1.TypeMeta{
		APIVersion: v1.GroupVersion.String(),
		Kind:       "VirtualMachineInstanceMigration",
	}

	return migration
}

func NewRandomVMIWithEphemeralDisk(containerImage string) *v1.VirtualMachineInstance {
	vmi := NewRandomVMI()

	vmi.Spec.Domain.Resources.Requests[k8sv1.ResourceMemory] = resource.MustParse("64M")
	AddEphemeralDisk(vmi, "disk0", "virtio", containerImage)
	return vmi
}

func AddEphemeralDisk(vmi *v1.VirtualMachineInstance, name string, bus string, image string) *v1.VirtualMachineInstance {
	vmi.Spec.Domain.Devices.Disks = append(vmi.Spec.Domain.Devices.Disks, v1.Disk{
		Name: name,
		DiskDevice: v1.DiskDevice{
			Disk: &v1.DiskTarget{
				Bus: bus,
			},
		},
	})
	vmi.Spec.Volumes = append(vmi.Spec.Volumes, v1.Volume{
		Name: name,
		VolumeSource: v1.VolumeSource{
			ContainerDisk: &v1.ContainerDiskSource{
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
		Name: name,
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
		Name: name,
		DiskDevice: v1.DiskDevice{
			Floppy: &v1.FloppyTarget{},
		},
	})
	vmi.Spec.Volumes = append(vmi.Spec.Volumes, v1.Volume{
		Name: name,
		VolumeSource: v1.VolumeSource{
			ContainerDisk: &v1.ContainerDiskSource{
				Image: image,
			},
		},
	})

	return vmi
}

func AddEphemeralCdrom(vmi *v1.VirtualMachineInstance, name string, bus string, image string) *v1.VirtualMachineInstance {
	vmi.Spec.Domain.Devices.Disks = append(vmi.Spec.Domain.Devices.Disks, v1.Disk{
		Name: name,
		DiskDevice: v1.DiskDevice{
			CDRom: &v1.CDRomTarget{
				Bus: bus,
			},
		},
	})
	vmi.Spec.Volumes = append(vmi.Spec.Volumes, v1.Volume{
		Name: name,
		VolumeSource: v1.VolumeSource{
			ContainerDisk: &v1.ContainerDiskSource{
				Image: image,
			},
		},
	})

	return vmi
}

func NewRandomVMIWithEphemeralDiskAndUserdata(containerImage string, userData string) *v1.VirtualMachineInstance {
	vmi := NewRandomVMIWithEphemeralDisk(containerImage)
	AddUserData(vmi, "disk1", userData)
	return vmi
}

func NewRandomVMIWithEphemeralDiskAndUserdataNetworkData(containerImage, userData, networkData string, b64encode bool) *v1.VirtualMachineInstance {
	vmi := NewRandomVMIWithEphemeralDisk(containerImage)
	AddCloudInitData(vmi, "disk1", userData, networkData, b64encode)
	return vmi
}

func AddUserData(vmi *v1.VirtualMachineInstance, name string, userData string) {
	vmi.Spec.Domain.Devices.Disks = append(vmi.Spec.Domain.Devices.Disks, v1.Disk{
		Name: name,
		DiskDevice: v1.DiskDevice{
			Disk: &v1.DiskTarget{
				Bus: "virtio",
			},
		},
	})
	vmi.Spec.Volumes = append(vmi.Spec.Volumes, v1.Volume{
		Name: name,
		VolumeSource: v1.VolumeSource{
			CloudInitNoCloud: &v1.CloudInitNoCloudSource{
				UserDataBase64: base64.StdEncoding.EncodeToString([]byte(userData)),
			},
		},
	})
}

func AddCloudInitData(vmi *v1.VirtualMachineInstance, name, userData, networkData string, b64encode bool) {
	vmi.Spec.Domain.Devices.Disks = append(vmi.Spec.Domain.Devices.Disks, v1.Disk{
		Name: name,
		DiskDevice: v1.DiskDevice{
			Disk: &v1.DiskTarget{
				Bus: "virtio",
			},
		},
	})
	if b64encode {
		vmi.Spec.Volumes = append(vmi.Spec.Volumes, v1.Volume{
			Name: name,
			VolumeSource: v1.VolumeSource{
				CloudInitNoCloud: &v1.CloudInitNoCloudSource{
					UserDataBase64:    base64.StdEncoding.EncodeToString([]byte(userData)),
					NetworkDataBase64: base64.StdEncoding.EncodeToString([]byte(networkData)),
				},
			},
		})
	} else {
		vmi.Spec.Volumes = append(vmi.Spec.Volumes, v1.Volume{
			Name: name,
			VolumeSource: v1.VolumeSource{
				CloudInitNoCloud: &v1.CloudInitNoCloudSource{
					UserData:    userData,
					NetworkData: networkData,
				},
			},
		})
	}
}

func NewRandomVMIWithPVC(claimName string) *v1.VirtualMachineInstance {

	vmi := NewRandomVMI()

	vmi.Spec.Domain.Resources.Requests[k8sv1.ResourceMemory] = resource.MustParse("64M")
	vmi.Spec.Domain.Devices.Disks = append(vmi.Spec.Domain.Devices.Disks, v1.Disk{
		Name: "disk0",
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

func CreateBlockVolumePvAndPvc(name string, size string) {
	virtCli, err := kubecli.GetKubevirtClient()
	PanicOnError(err)

	labelSelector := make(map[string]string)
	labelSelector["kubevirt-test"] = name

	_, err = virtCli.CoreV1().PersistentVolumes().Create(newBlockVolumePV(name, labelSelector, size))
	if !errors.IsAlreadyExists(err) {
		PanicOnError(err)
	}

	_, err = virtCli.CoreV1().PersistentVolumeClaims(NamespaceTestDefault).Create(newBlockVolumePVC(name, labelSelector, size))
	if !errors.IsAlreadyExists(err) {
		PanicOnError(err)
	}
}

func newBlockVolumePV(name string, labelSelector map[string]string, size string) *k8sv1.PersistentVolume {
	quantity, err := resource.ParseQuantity(size)
	PanicOnError(err)

	storageClass := StorageClassBlockVolume
	volumeMode := k8sv1.PersistentVolumeBlock

	// Note: the path depends on kubevirtci!
	// It's configured to have a device backed by a cirros image at exactly that place on node01
	// And the local storage provider also has access to it
	return &k8sv1.PersistentVolume{
		ObjectMeta: metav1.ObjectMeta{
			Name:   name,
			Labels: labelSelector,
		},
		Spec: k8sv1.PersistentVolumeSpec{
			AccessModes: []k8sv1.PersistentVolumeAccessMode{k8sv1.ReadWriteOnce},
			Capacity: k8sv1.ResourceList{
				"storage": quantity,
			},
			StorageClassName: storageClass,
			VolumeMode:       &volumeMode,
			PersistentVolumeSource: k8sv1.PersistentVolumeSource{
				Local: &k8sv1.LocalVolumeSource{
					Path: "/mnt/local-storage/cirros-block-device",
				},
			},
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
}

func newBlockVolumePVC(name string, labelSelector map[string]string, size string) *k8sv1.PersistentVolumeClaim {
	quantity, err := resource.ParseQuantity(size)
	PanicOnError(err)

	storageClass := StorageClassBlockVolume
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

func DeletePvAndPvc(name string) {
	virtCli, err := kubecli.GetKubevirtClient()
	PanicOnError(err)

	err = virtCli.CoreV1().PersistentVolumes().Delete(name, &metav1.DeleteOptions{})
	PanicOnError(err)

	err = virtCli.CoreV1().PersistentVolumeClaims(NamespaceTestDefault).Delete(name, &metav1.DeleteOptions{})
	PanicOnError(err)
}

func NewRandomVMIWithCDRom(claimName string) *v1.VirtualMachineInstance {
	vmi := NewRandomVMI()

	vmi.Spec.Domain.Resources.Requests[k8sv1.ResourceMemory] = resource.MustParse("64M")
	vmi.Spec.Domain.Devices.Disks = append(vmi.Spec.Domain.Devices.Disks, v1.Disk{
		Name: "disk0",
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
		Name: "disk0",
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

func AddHostDisk(vmi *v1.VirtualMachineInstance, path string, diskType v1.HostDiskType, name string) {
	hostDisk := v1.HostDisk{
		Path: path,
		Type: diskType,
	}
	if diskType == v1.HostDiskExistsOrCreate {
		hostDisk.Capacity = resource.MustParse(defaultDiskSize)
	}

	vmi.Spec.Domain.Devices.Disks = append(vmi.Spec.Domain.Devices.Disks, v1.Disk{
		Name: name,
		DiskDevice: v1.DiskDevice{
			Disk: &v1.DiskTarget{
				Bus: "virtio",
			},
		},
	})
	vmi.Spec.Volumes = append(vmi.Spec.Volumes, v1.Volume{
		Name: name,
		VolumeSource: v1.VolumeSource{
			HostDisk: &hostDisk,
		},
	})
}

func NewRandomVMIWithHostDisk(diskPath string, diskType v1.HostDiskType, nodeName string) *v1.VirtualMachineInstance {
	vmi := NewRandomVMI()
	AddHostDisk(vmi, diskPath, diskType, "host-disk")
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
	return vmi
}

func NewRandomVMIWithWatchdog() *v1.VirtualMachineInstance {
	vmi := NewRandomVMIWithEphemeralDisk(ContainerDiskFor(ContainerDiskAlpine))

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
	vmi.Spec.Volumes = append(vmi.Spec.Volumes, v1.Volume{
		Name: configMapName + "-disk",
		VolumeSource: v1.VolumeSource{
			ConfigMap: &v1.ConfigMapVolumeSource{
				LocalObjectReference: k8sv1.LocalObjectReference{
					Name: configMapName,
				},
			},
		},
	})
	vmi.Spec.Domain.Devices.Disks = append(vmi.Spec.Domain.Devices.Disks, v1.Disk{
		Name: configMapName + "-disk",
	})
}

func NewRandomVMIWithSecret(secretName string) *v1.VirtualMachineInstance {
	vmi := NewRandomVMIWithPVC(DiskAlpineHostPath)
	AddSecretDisk(vmi, secretName)
	return vmi
}

func AddSecretDisk(vmi *v1.VirtualMachineInstance, secretName string) {
	vmi.Spec.Volumes = append(vmi.Spec.Volumes, v1.Volume{
		Name: secretName + "-disk",
		VolumeSource: v1.VolumeSource{
			Secret: &v1.SecretVolumeSource{
				SecretName: secretName,
			},
		},
	})
	vmi.Spec.Domain.Devices.Disks = append(vmi.Spec.Domain.Devices.Disks, v1.Disk{
		Name: secretName + "-disk",
	})
}

func NewRandomVMIWithServiceAccount(serviceAccountName string) *v1.VirtualMachineInstance {
	vmi := NewRandomVMIWithPVC(DiskAlpineHostPath)
	AddServiceAccountDisk(vmi, serviceAccountName)
	return vmi
}

func AddServiceAccountDisk(vmi *v1.VirtualMachineInstance, serviceAccountName string) {
	volumeName := serviceAccountName + "-disk"
	vmi.Spec.Volumes = append(vmi.Spec.Volumes, v1.Volume{
		Name: volumeName,
		VolumeSource: v1.VolumeSource{
			ServiceAccount: &v1.ServiceAccountVolumeSource{
				ServiceAccountName: serviceAccountName,
			},
		},
	})
	vmi.Spec.Domain.Devices.Disks = append(vmi.Spec.Domain.Devices.Disks, v1.Disk{
		Name: serviceAccountName + "-disk",
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
	vmi := NewRandomVMIWithEphemeralDisk(ContainerDiskFor(ContainerDiskAlpine))
	AddExplicitPodNetworkInterface(vmi)
	vmi.Spec.Domain.Devices.Interfaces[0].Model = "e1000"
	return vmi
}

func NewRandomVMIWithCustomMacAddress() *v1.VirtualMachineInstance {
	vmi := NewRandomVMIWithEphemeralDisk(ContainerDiskFor(ContainerDiskAlpine))
	AddExplicitPodNetworkInterface(vmi)
	vmi.Spec.Domain.Devices.Interfaces[0].MacAddress = "de:ad:00:00:be:af"
	return vmi
}

// Block until DataVolume succeeds.
func WaitForSuccessfulDataVolumeImport(obj runtime.Object, seconds int) {
	vmi, ok := obj.(*v1.VirtualMachineInstance)
	ExpectWithOffset(1, ok).To(BeTrue(), "Object is not of type *v1.VMI")

	virtClient, err := kubecli.GetKubevirtClient()
	ExpectWithOffset(1, err).ToNot(HaveOccurred())

	EventuallyWithOffset(1, func() cdiv1.DataVolumePhase {
		dv, err := virtClient.CdiClient().CdiV1alpha1().DataVolumes(vmi.Namespace).Get(vmi.Spec.Volumes[0].DataVolume.Name, metav1.GetOptions{})
		ExpectWithOffset(1, err).ToNot(HaveOccurred())

		return dv.Status.Phase
	}, time.Duration(seconds)*time.Second, 1*time.Second).Should(Equal(cdiv1.Succeeded), "Timed out waiting for DataVolume to enter Succeeded phase")

	return
}

// Block until the specified VirtualMachineInstance started and return the target node name.
func waitForVMIStart(obj runtime.Object, seconds int, ignoreWarnings bool) (nodeName string) {
	vmi, ok := obj.(*v1.VirtualMachineInstance)
	ExpectWithOffset(1, ok).To(BeTrue(), "Object is not of type *v1.VMI")

	virtClient, err := kubecli.GetKubevirtClient()
	ExpectWithOffset(1, err).ToNot(HaveOccurred())

	// In case we don't want errors, start an event watcher and  check in parallel if we receive some warnings
	if ignoreWarnings != true {

		// Fetch the VirtualMachineInstance, to make sure we have a resourceVersion as a starting point for the watch
		// FIXME: This may start watching too late and we may miss some warnings
		if vmi.ResourceVersion == "" {
			vmi, err = virtClient.VirtualMachineInstance(vmi.Namespace).Get(vmi.Name, &metav1.GetOptions{})
			ExpectWithOffset(1, err).ToNot(HaveOccurred())
		}

		objectEventWatcher := NewObjectEventWatcher(vmi).SinceWatchedObjectResourceVersion().Timeout(time.Duration(seconds+2) * time.Second)
		objectEventWatcher.FailOnWarnings()

		stopChan := make(chan struct{})
		defer close(stopChan)
		go func() {
			defer GinkgoRecover()
			objectEventWatcher.WaitFor(stopChan, NormalEvent, v1.Started)
		}()
	}

	// FIXME the event order is wrong. First the document should be updated
	EventuallyWithOffset(1, func() v1.VirtualMachineInstancePhase {
		vmi, err = virtClient.VirtualMachineInstance(vmi.Namespace).Get(vmi.Name, &metav1.GetOptions{})
		ExpectWithOffset(1, err).ToNot(HaveOccurred())

		nodeName = vmi.Status.NodeName
		Expect(vmi.IsFinal()).To(BeFalse(), "VMI unexpectedly stopped. State: %s", vmi.Status.Phase)
		return vmi.Status.Phase
	}, time.Duration(seconds)*time.Second, 1*time.Second).Should(Equal(v1.Running), "Timed out waiting for VMI to enter Running phase")

	return
}

func WaitForSuccessfulVMIStartIgnoreWarnings(vmi runtime.Object) string {
	return waitForVMIStart(vmi, 180, true)
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
	return waitForVMIStart(vmi, 180, false)
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

type ContainerDisk string

const (
	ContainerDiskCirros ContainerDisk = "cirros"
	ContainerDiskAlpine ContainerDisk = "alpine"
	ContainerDiskFedora ContainerDisk = "fedora-cloud"
	ContainerDiskVirtio ContainerDisk = "virtio-container-disk"
)

// ContainerDiskFor takes the name of an image and returns the full
// registry diks image path.
// Supported values are: cirros, fedora, alpine, guest-agent
func ContainerDiskFor(name ContainerDisk) string {
	switch name {
	case ContainerDiskCirros, ContainerDiskAlpine, ContainerDiskFedora:
		return fmt.Sprintf("%s/%s-container-disk-demo:%s", KubeVirtRepoPrefix, name, KubeVirtVersionTag)
	case ContainerDiskVirtio:
		return fmt.Sprintf("%s/virtio-container-disk:%s", KubeVirtRepoPrefix, KubeVirtVersionTag)
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

// LoggedInFedoraExpecter return prepared and ready to use console expecter for
// Fedora test VM
func LoggedInFedoraExpecter(vmi *v1.VirtualMachineInstance) (expect.Expecter, error) {
	virtClient, err := kubecli.GetKubevirtClient()
	PanicOnError(err)
	expecter, _, err := NewConsoleExpecter(virtClient, vmi, 10*time.Second)
	if err != nil {
		return nil, err
	}
	b := append([]expect.Batcher{
		&expect.BSnd{S: "\n"},
		&expect.BExp{R: "login:"},
		&expect.BSnd{S: "fedora\n"},
		&expect.BExp{R: "Password:"},
		&expect.BSnd{S: "fedora\n"},
		&expect.BExp{R: "$"},
		&expect.BSnd{S: "sudo su\n"},
		&expect.BExp{R: "#"}})
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

func GetRunningVirtualMachineInstanceDomainXML(virtClient kubecli.KubevirtClient, vmi *v1.VirtualMachineInstance) (string, error) {
	vmiPod := GetRunningPodByVirtualMachineInstance(vmi, NamespaceTestDefault)

	found := false
	containerIdx := 0
	for idx, container := range vmiPod.Spec.Containers {
		if container.Name == "compute" {
			containerIdx = idx
			found = true
		}
	}
	if !found {
		return "", fmt.Errorf("could not find compute container for pod")
	}

	stdout, _, err := ExecuteCommandOnPodV2(
		virtClient,
		vmiPod,
		vmiPod.Spec.Containers[containerIdx].Name,
		[]string{"virsh", "dumpxml", vmi.Namespace + "_" + vmi.Name},
	)
	if err != nil {
		return "", fmt.Errorf("could not dump libvirt domxml (remotely on pod): %v", err)
	}
	return stdout, err
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

func SkipIfNoSriovDevicePlugin(virtClient kubecli.KubevirtClient) {
	_, err := virtClient.ExtensionsV1beta1().DaemonSets(metav1.NamespaceSystem).Get("kube-sriov-device-plugin-amd64", metav1.GetOptions{})
	if err != nil {
		Skip("Skip srio tests that required sriov device plugin")
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
	out, _, _ := RunCommand("kubectl", "get", "clusternetwork")
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

func RunCommand(cmdName string, args ...string) (string, string, error) {
	return RunCommandWithNS(NamespaceTestDefault, cmdName, args...)
}

func RunCommandWithNS(namespace string, cmdName string, args ...string) (string, string, error) {
	commandString, cmd, err := CreateCommandWithNS(namespace, cmdName, args...)
	if err != nil {
		return "", "", err
	}

	var output, stderr bytes.Buffer
	captureOutputBuffers := func() (string, string) {
		trimNullChars := func(buf bytes.Buffer) string {
			return string(bytes.Trim(buf.Bytes(), "\x00"))
		}
		return trimNullChars(output), trimNullChars(stderr)
	}

	cmd.Stdout, cmd.Stderr = &output, &stderr

	if err := cmd.Run(); err != nil {
		outputString, stderrString := captureOutputBuffers()
		log.Log.Reason(err).With("command", commandString, "output", outputString, "stderr", stderrString).Error("command failed: cannot run command")
		return outputString, stderrString, fmt.Errorf("command failed: cannot run command %q: %v", commandString, err)
	}

	outputString, stderrString := captureOutputBuffers()
	return outputString, stderrString, nil
}

func CreateCommandWithNS(namespace string, cmdName string, args ...string) (string, *exec.Cmd, error) {
	cmdPath := ""
	commandString := func() string {
		c := cmdPath
		if cmdPath == "" {
			c = cmdName
		}
		return strings.Join(append([]string{c}, args...), " ")
	}

	cmdName = strings.ToLower(cmdName)
	switch cmdName {
	case "oc":
		cmdPath = KubeVirtOcPath
	case "kubectl":
		cmdPath = KubeVirtKubectlPath
	case "virtctl":
		cmdPath = KubeVirtVirtctlPath
	}

	if cmdPath == "" {
		err := fmt.Errorf("no %s binary specified", cmdName)
		log.Log.Reason(err).With("command", commandString()).Error("command failed")
		return "", nil, fmt.Errorf("command failed: %v", err)
	}

	kubeconfig := flag.Lookup("kubeconfig").Value
	if kubeconfig == nil || kubeconfig.String() == "" {
		err := goerrors.New("cannot find kubeconfig")
		log.Log.Reason(err).With("command", commandString()).Error("command failed")
		return "", nil, fmt.Errorf("command failed: %v", err)
	}

	master := flag.Lookup("master").Value
	if master != nil && master.String() != "" {
		args = append(args, "--server", master.String())
	}
	if namespace != "" {
		args = append(args, "-n", namespace)
	}

	cmd := exec.Command(cmdPath, args...)
	kubeconfEnv := fmt.Sprintf("KUBECONFIG=%s", kubeconfig.String())
	cmd.Env = append(os.Environ(), kubeconfEnv)

	return commandString(), cmd, nil
}

func RunCommandPipe(commands ...[]string) (string, string, error) {
	return RunCommandPipeWithNS(NamespaceTestDefault, commands...)
}

func RunCommandPipeWithNS(namespace string, commands ...[]string) (string, string, error) {
	commandPipeString := func() string {
		commandStrings := []string{}
		for _, command := range commands {
			commandStrings = append(commandStrings, strings.Join(command, " "))
		}
		return strings.Join(commandStrings, " | ")
	}

	if len(commands) < 2 {
		err := goerrors.New("requires at least two commands")
		log.Log.Reason(err).With("command", commandPipeString()).Error("command pipe failed")
		return "", "", fmt.Errorf("command pipe failed: %v", err)
	}

	for i, command := range commands {
		cmdPath := ""
		cmdName := strings.ToLower(command[0])
		switch cmdName {
		case "oc":
			cmdPath = KubeVirtOcPath
		case "kubectl":
			cmdPath = KubeVirtKubectlPath
		case "virtctl":
			cmdPath = KubeVirtVirtctlPath
		}
		if cmdPath == "" {
			err := fmt.Errorf("no %s binary specified", cmdName)
			log.Log.Reason(err).With("command", commandPipeString()).Error("command pipe failed")
			return "", "", fmt.Errorf("command pipe failed: %v", err)
		}
		commands[i][0] = cmdPath
	}

	kubeconfig := flag.Lookup("kubeconfig").Value
	if kubeconfig == nil || kubeconfig.String() == "" {
		err := goerrors.New("cannot find kubeconfig")
		log.Log.Reason(err).With("command", commandPipeString()).Error("command pipe failed")
		return "", "", fmt.Errorf("command pipe failed: %v", err)
	}
	kubeconfEnv := fmt.Sprintf("KUBECONFIG=%s", kubeconfig.String())

	master := flag.Lookup("master").Value
	cmds := make([]*exec.Cmd, len(commands))
	for i := range cmds {
		if master != nil && master.String() != "" {
			commands[i] = append(commands[i], "--server", master.String())
		}
		if namespace != "" {
			commands[i] = append(commands[i], "-n", namespace)
		}
		cmds[i] = exec.Command(commands[i][0], commands[i][1:]...)
		cmds[i].Env = append(os.Environ(), kubeconfEnv)
	}

	var output, stderr bytes.Buffer
	captureOutputBuffers := func() (string, string) {
		trimNullChars := func(buf bytes.Buffer) string {
			return string(bytes.Trim(buf.Bytes(), "\x00"))
		}
		return trimNullChars(output), trimNullChars(stderr)
	}

	last := len(cmds) - 1
	for i, cmd := range cmds[:last] {
		var err error
		if cmds[i+1].Stdin, err = cmd.StdoutPipe(); err != nil {
			cmdArgString := strings.Join(cmd.Args, " ")
			log.Log.Reason(err).With("command", commandPipeString()).Errorf("command pipe failed: cannot attach stdout pipe to command %q", cmdArgString)
			return "", "", fmt.Errorf("command pipe failed: cannot attach stdout pipe to command %q: %v", cmdArgString, err)
		}
		cmd.Stderr = &stderr
	}
	cmds[last].Stdout, cmds[last].Stderr = &output, &stderr

	for _, cmd := range cmds {
		if err := cmd.Start(); err != nil {
			outputString, stderrString := captureOutputBuffers()
			cmdArgString := strings.Join(cmd.Args, " ")
			log.Log.Reason(err).With("command", commandPipeString(), "output", outputString, "stderr", stderrString).Errorf("command pipe failed: cannot start command %q", cmdArgString)
			return outputString, stderrString, fmt.Errorf("command pipe failed: cannot start command %q: %v", cmdArgString, err)
		}
	}

	for _, cmd := range cmds {
		if err := cmd.Wait(); err != nil {
			outputString, stderrString := captureOutputBuffers()
			cmdArgString := strings.Join(cmd.Args, " ")
			log.Log.Reason(err).With("command", commandPipeString(), "output", outputString, "stderr", stderrString).Errorf("command pipe failed: error while waiting for command %q", cmdArgString)
			return outputString, stderrString, fmt.Errorf("command pipe failed: error while waiting for command %q: %v", cmdArgString, err)
		}
	}

	outputString, stderrString := captureOutputBuffers()
	return outputString, stderrString, nil
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
		return "", fmt.Errorf("failed to generate json for template %q: %v", template.Name, err)
	}

	dir, err := ioutil.TempDir("", TempDirPrefix+"-")
	if err != nil {
		return "", fmt.Errorf("failed to create a temporary directory in %q: %v", os.TempDir(), err)
	}

	jsonFile := filepath.Join(dir, template.Name+".json")
	if err = ioutil.WriteFile(jsonFile, data, 0644); err != nil {
		return "", fmt.Errorf("failed to write json to file %q: %v", jsonFile, err)
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

func CreateISCSITargetPOD(containerDiskName ContainerDisk) (iscsiTargetIP string) {
	virtClient, err := kubecli.GetKubevirtClient()
	PanicOnError(err)
	image := fmt.Sprintf("%s/cdi-http-import-server:%s", KubeVirtRepoPrefix, KubeVirtVersionTag)
	resources := k8sv1.ResourceRequirements{}
	resources.Limits = make(k8sv1.ResourceList)
	resources.Limits[k8sv1.ResourceMemory] = resource.MustParse("64M")
	pod := &k8sv1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			GenerateName: "test-iscsi-target",
			Labels: map[string]string{
				v1.AppLabel: "test-iscsi-target",
			},
		},
		Spec: k8sv1.PodSpec{
			RestartPolicy: k8sv1.RestartPolicyNever,
			Containers: []k8sv1.Container{
				{
					Name:      "test-iscsi-target",
					Image:     image,
					Resources: resources,
					Env: []k8sv1.EnvVar{
						{
							Name:  "AS_ISCSI",
							Value: "true",
						},
						{
							Name:  "IMAGE_NAME",
							Value: fmt.Sprintf("%s", containerDiskName),
						},
					},
				},
			},
		},
	}
	pod, err = virtClient.CoreV1().Pods(NamespaceTestDefault).Create(pod)
	PanicOnError(err)

	getStatus := func() k8sv1.PodPhase {
		pod, err := virtClient.CoreV1().Pods(NamespaceTestDefault).Get(pod.Name, metav1.GetOptions{})
		Expect(err).ToNot(HaveOccurred())
		iscsiTargetIP = pod.Status.PodIP
		return pod.Status.Phase
	}
	Eventually(getStatus, 120, 1).Should(Equal(k8sv1.PodRunning))
	return
}

func CreateISCSIPvAndPvc(name string, size string, iscsiTargetIP string) {
	virtCli, err := kubecli.GetKubevirtClient()
	PanicOnError(err)

	_, err = virtCli.CoreV1().PersistentVolumes().Create(newISCSIPV(name, size, iscsiTargetIP))
	if !errors.IsAlreadyExists(err) {
		PanicOnError(err)
	}

	_, err = virtCli.CoreV1().PersistentVolumeClaims(NamespaceTestDefault).Create(newISCSIPVC(name, size))
	if !errors.IsAlreadyExists(err) {
		PanicOnError(err)
	}
}

func newISCSIPV(name string, size string, iscsiTargetIP string) *k8sv1.PersistentVolume {
	quantity, err := resource.ParseQuantity(size)
	PanicOnError(err)

	storageClass := StorageClassLocal
	volumeMode := k8sv1.PersistentVolumeBlock

	return &k8sv1.PersistentVolume{
		ObjectMeta: metav1.ObjectMeta{
			Name: name,
		},
		Spec: k8sv1.PersistentVolumeSpec{
			AccessModes: []k8sv1.PersistentVolumeAccessMode{k8sv1.ReadWriteMany},
			Capacity: k8sv1.ResourceList{
				"storage": quantity,
			},
			ClaimRef: &k8sv1.ObjectReference{
				Name:      name,
				Namespace: NamespaceTestDefault,
			},
			StorageClassName: storageClass,
			VolumeMode:       &volumeMode,
			PersistentVolumeSource: k8sv1.PersistentVolumeSource{
				ISCSI: &k8sv1.ISCSIPersistentVolumeSource{
					TargetPortal: iscsiTargetIP,
					IQN:          "iqn.2018-01.io.kubevirt:wrapper",
					Lun:          1,
					ReadOnly:     false,
				},
			},
		},
	}
}

func newISCSIPVC(name string, size string) *k8sv1.PersistentVolumeClaim {
	quantity, err := resource.ParseQuantity(size)
	PanicOnError(err)

	storageClass := StorageClassLocal
	volumeMode := k8sv1.PersistentVolumeBlock

	return &k8sv1.PersistentVolumeClaim{
		ObjectMeta: metav1.ObjectMeta{Name: name},
		Spec: k8sv1.PersistentVolumeClaimSpec{
			AccessModes: []k8sv1.PersistentVolumeAccessMode{k8sv1.ReadWriteMany},
			Resources: k8sv1.ResourceRequirements{
				Requests: k8sv1.ResourceList{
					"storage": quantity,
				},
			},
			StorageClassName: &storageClass,
			VolumeMode:       &volumeMode,
		},
	}
}

func CreateHostDiskImage(diskPath string) *k8sv1.Pod {
	hostPathType := k8sv1.HostPathDirectoryOrCreate
	dir := filepath.Dir(diskPath)

	args := []string{fmt.Sprintf(`dd if=/dev/zero of=%s bs=1 count=0 seek=1G && ls -l %s`, diskPath, dir)}
	job := RenderHostPathJob("hostdisk-create-job", dir, hostPathType, k8sv1.MountPropagationNone, []string{"/bin/bash", "-c"}, args)

	return job
}

func newDeleteHostDisksJob(diskPath string) *k8sv1.Pod {
	hostPathType := k8sv1.HostPathDirectoryOrCreate

	args := []string{fmt.Sprintf(`rm -f %s`, diskPath)}
	job := RenderHostPathJob("hostdisk-delete-job", filepath.Dir(diskPath), hostPathType, k8sv1.MountPropagationNone, []string{"/bin/bash", "-c"}, args)

	return job
}

func RenderHostPathJob(jobName string, dir string, hostPathType k8sv1.HostPathType, mountPropagation k8sv1.MountPropagationMode, cmd []string, args []string) *k8sv1.Pod {
	job := RenderJob(jobName, cmd, args)
	job.Spec.Containers[0].VolumeMounts = append(job.Spec.Containers[0].VolumeMounts, k8sv1.VolumeMount{
		Name:             "hostpath-mount",
		MountPropagation: &mountPropagation,
		MountPath:        dir,
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
	if IsOpenShift() {
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
	vmi := NewRandomVMIWithEphemeralDiskAndUserdata(ContainerDiskFor(ContainerDiskCirros), "#!/bin/bash\necho 'hello'\n")
	StartVmOnNode(vmi, nodeName)

	return RunCommandOnVmiPod(vmi, []string{"virsh", "-r", "capabilities"})
}

// GetNodeCPUInfo returns output of lscpu on the pod that runs on the specified node
func GetNodeCPUInfo(nodeName string) string {
	vmi := NewRandomVMIWithEphemeralDiskAndUserdata(ContainerDiskFor(ContainerDiskCirros), "#!/bin/bash\necho 'hello'\n")
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

	for _, ns := range []string{KubeVirtInstallNamespace, metav1.NamespaceSystem, NamespaceTestDefault} {
		// Get KubeVirt and CDI specific pods information
		labels := []string{"kubevirt.io", "cdi.kubevirt.io"}
		allPods := []k8sv1.Pod{}

		for _, label := range labels {
			pods, err := virtClient.CoreV1().Pods(ns).List(metav1.ListOptions{LabelSelector: label})
			if err != nil {
				fmt.Println(err)
				Fail(message, callerSkip...)
				return
			}
			allPods = append(allPods, pods.Items...)
		}

		for _, pod := range allPods {
			fmt.Printf("\nPod name: %s\t Pod phase: %s\n\n", pod.Name, pod.Status.Phase)
			data, err := ghodssyaml.Marshal(pod)
			if err != nil {
				log.DefaultLogger().Reason(err).Errorf("Failed to marshal pod %s", pod.Name)
				continue
			}
			fmt.Println(string(data))

			var tailLines int64 = 45
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
			data, err := ghodssyaml.Marshal(vmi)
			if err != nil {
				log.DefaultLogger().Reason(err).Errorf("Failed to marshal vmi %s", vmi.Name)
				continue
			}
			fmt.Println(string(data))
		}

		pvcs, err := virtClient.CoreV1().PersistentVolumeClaims(ns).List(metav1.ListOptions{})
		if err != nil {
			fmt.Println(err)
			Fail(message, callerSkip...)
			return
		}

		for _, pvc := range pvcs.Items {
			data, err := ghodssyaml.Marshal(pvc)
			if err != nil {
				log.DefaultLogger().Reason(err).Errorf("Failed to marshal pvc %s", pvc.Name)
				continue
			}
			fmt.Println(string(data))
		}
	}

	pvs, err := virtClient.CoreV1().PersistentVolumes().List(metav1.ListOptions{})
	if err != nil {
		fmt.Println(err)
		Fail(message, callerSkip...)
		return
	}

	for _, pv := range pvs.Items {
		data, err := ghodssyaml.Marshal(pv)
		if err != nil {
			log.DefaultLogger().Reason(err).Errorf("Failed to marshal pvc %s", pv.Name)
			continue
		}
		fmt.Println(string(data))
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
	cfgMap, err := virtClient.CoreV1().ConfigMaps(KubeVirtInstallNamespace).Get("kubevirt-config", options)
	if err == nil {
		val, ok := cfgMap.Data[virtconfig.FeatureGatesKey]
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

func StartTCPServer(vmi *v1.VirtualMachineInstance, port int) {
	expecter, err := LoggedInCirrosExpecter(vmi)
	Expect(err).ToNot(HaveOccurred())
	defer expecter.Close()

	resp, err := expecter.ExpectBatch([]expect.Batcher{
		&expect.BSnd{S: "\n"},
		&expect.BExp{R: "\\$ "},
		&expect.BSnd{S: fmt.Sprintf("screen -d -m nc -klp %d -e echo -e \"Hello World!\"\n", port)},
		&expect.BExp{R: "\\$ "},
		&expect.BSnd{S: "echo $?\n"},
		&expect.BExp{R: "0"},
	}, 60*time.Second)
	log.DefaultLogger().Infof("%v", resp)
	Expect(err).ToNot(HaveOccurred())
}

func StartHTTPServer(vmi *v1.VirtualMachineInstance, port int) {
	expecter, err := LoggedInCirrosExpecter(vmi)
	Expect(err).ToNot(HaveOccurred())
	defer expecter.Close()

	resp, err := expecter.ExpectBatch([]expect.Batcher{
		&expect.BSnd{S: "\n"},
		&expect.BExp{R: "\\$ "},
		&expect.BSnd{S: fmt.Sprintf("screen -d -m nc -klp %d -e echo -e \"HTTP/1.1 200 OK\\n\\nHello World!\"\n", port)},
		&expect.BExp{R: "\\$ "},
		&expect.BSnd{S: "echo $?\n"},
		&expect.BExp{R: "0"},
	}, 60*time.Second)
	log.DefaultLogger().Infof("%v", resp)
	Expect(err).ToNot(HaveOccurred())
}

func GetVmPodName(virtCli kubecli.KubevirtClient, vmi *v1.VirtualMachineInstance) string {
	namespace := vmi.GetObjectMeta().GetNamespace()
	uid := vmi.GetObjectMeta().GetUID()
	labelSelector := fmt.Sprintf(v1.CreatedByLabel + "=" + string(uid))

	pods, err := virtCli.CoreV1().Pods(namespace).List(metav1.ListOptions{LabelSelector: labelSelector})
	Expect(err).ToNot(HaveOccurred())

	podName := ""
	for _, pod := range pods.Items {
		if pod.ObjectMeta.DeletionTimestamp == nil {
			podName = pod.ObjectMeta.Name
			break
		}
	}
	Expect(podName).ToNot(BeEmpty())

	return podName
}
