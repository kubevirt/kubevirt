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
	cryptorand "crypto/rand"
	"crypto/rsa"
	"crypto/tls"
	"crypto/x509"
	"encoding/base64"
	"encoding/json"
	"encoding/pem"
	"encoding/xml"
	goerrors "errors"
	"flag"
	"fmt"
	"io"
	"io/ioutil"
	"net"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"

	netutils "k8s.io/utils/net"

	expect "github.com/google/goexpect"
	covreport "github.com/mfranczy/crd-rest-coverage/pkg/report"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/spf13/cobra"
	"golang.org/x/crypto/ssh"
	"google.golang.org/grpc/codes"
	autoscalingv1 "k8s.io/api/autoscaling/v1"
	k8sv1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	settingsv1alpha1 "k8s.io/api/settings/v1alpha1"
	storagev1 "k8s.io/api/storage/v1"
	extclient "k8s.io/apiextensions-apiserver/pkg/client/clientset/clientset"
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
	"k8s.io/apimachinery/pkg/util/strategicpatch"
	"k8s.io/apimachinery/pkg/util/yaml"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/tools/portforward"
	"k8s.io/client-go/tools/remotecommand"
	"k8s.io/client-go/transport/spdy"

	"kubevirt.io/kubevirt/pkg/certificates/triple/cert"
	"kubevirt.io/kubevirt/pkg/virt-operator/creation/components"

	"kubevirt.io/kubevirt/pkg/certificates/bootstrap"

	v1 "kubevirt.io/client-go/api/v1"
	"kubevirt.io/client-go/kubecli"
	"kubevirt.io/client-go/log"
	cdiv1 "kubevirt.io/containerized-data-importer/pkg/apis/core/v1alpha1"
	"kubevirt.io/kubevirt/pkg/controller"
	"kubevirt.io/kubevirt/pkg/util/cluster"
	"kubevirt.io/kubevirt/pkg/util/net/dns"
	virtconfig "kubevirt.io/kubevirt/pkg/virt-config"
	"kubevirt.io/kubevirt/pkg/virt-controller/services"
	launcherApi "kubevirt.io/kubevirt/pkg/virt-launcher/virtwrap/api"
	"kubevirt.io/kubevirt/pkg/virt-operator/util"
	"kubevirt.io/kubevirt/pkg/virtctl"
	vmsgen "kubevirt.io/kubevirt/tools/vms-generator/utils"
)

var KubeVirtUtilityVersionTag = ""
var KubeVirtVersionTag = "latest"
var KubeVirtVersionTagAlt = ""
var KubeVirtUtilityRepoPrefix = ""
var KubeVirtRepoPrefix = "kubevirt"
var ImagePrefixAlt = ""
var ContainerizedDataImporterNamespace = "cdi"
var KubeVirtKubectlPath = ""
var KubeVirtOcPath = ""
var KubeVirtVirtctlPath = ""
var KubeVirtGoCliPath = ""
var KubeVirtInstallNamespace string
var PreviousReleaseTag = ""
var PreviousReleaseRegistry = ""
var ConfigFile = ""
var Config *KubeVirtTestsConfiguration
var SkipShasumCheck bool

var DeployTestingInfrastructureFlag = false
var PathToTestingInfrastrucureManifests = ""

func init() {
	kubecli.Init()
	flag.StringVar(&KubeVirtUtilityVersionTag, "utility-container-tag", "", "Set the image tag or digest to use")
	flag.StringVar(&KubeVirtVersionTag, "container-tag", "latest", "Set the image tag or digest to use")
	flag.StringVar(&KubeVirtVersionTagAlt, "container-tag-alt", "", "An alternate tag that can be used to test operator deployments")
	flag.StringVar(&KubeVirtUtilityRepoPrefix, "utility-container-prefix", "", "Set the repository prefix for all images")
	flag.StringVar(&KubeVirtRepoPrefix, "container-prefix", "kubevirt", "Set the repository prefix for all images")
	flag.StringVar(&ImagePrefixAlt, "image-prefix-alt", "", "Optional prefix for virt-* image names for additional imagePrefix operator test")
	flag.StringVar(&ContainerizedDataImporterNamespace, "cdi-namespace", "cdi", "Set the repository prefix for CDI components")
	flag.StringVar(&KubeVirtKubectlPath, "kubectl-path", "", "Set path to kubectl binary")
	flag.StringVar(&KubeVirtOcPath, "oc-path", "", "Set path to oc binary")
	flag.StringVar(&KubeVirtVirtctlPath, "virtctl-path", "", "Set path to virtctl binary")
	flag.StringVar(&KubeVirtGoCliPath, "gocli-path", "", "Set path to gocli binary")
	flag.StringVar(&KubeVirtInstallNamespace, "installed-namespace", "kubevirt", "Set the namespace KubeVirt is installed in")
	flag.BoolVar(&DeployTestingInfrastructureFlag, "deploy-testing-infra", false, "Deploy testing infrastructure if set")
	flag.StringVar(&PathToTestingInfrastrucureManifests, "path-to-testing-infra-manifests", "manifests/testing", "Set path to testing infrastructure manifests")
	flag.StringVar(&PreviousReleaseTag, "previous-release-tag", "", "Set tag of the release to test updating from")
	flag.StringVar(&PreviousReleaseRegistry, "previous-release-registry", "", "Set registry of the release to test updating from")
	flag.StringVar(&ConfigFile, "config", "tests/default-config.json", "Path to a JSON formatted file from which the test suite will load its configuration. The path may be absolute or relative; relative paths start at the current working directory.")
	flag.BoolVar(&SkipShasumCheck, "skip-shasums-check", false, "Skip tests with sha sums.")
}

func FlagParse() {
	flag.Parse()

	// When the flags are not provided, copy the values from normal version tag and prefix
	if KubeVirtUtilityVersionTag == "" {
		KubeVirtUtilityVersionTag = KubeVirtVersionTag
	}

	if KubeVirtUtilityRepoPrefix == "" {
		KubeVirtUtilityRepoPrefix = KubeVirtRepoPrefix
	}
}

type EventType string

const TempDirPrefix = "kubevirt-test"

const (
	defaultEventuallyTimeout         = 5 * time.Second
	defaultEventuallyPollingInterval = 1 * time.Second
)

const (
	AlpineHttpUrl = iota
	GuestAgentHttpUrl
	StressHttpUrl
	DmidecodeHttpUrl
	DummyFileHttpUrl
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
const namespaceKubevirt = "kubevirt"
const kubevirtConfig = "kubevirt-config"

const (
	// tests.NamespaceTestDefault is the default namespace, to test non-infrastructure related KubeVirt objects.
	NamespaceTestDefault = "kubevirt-test-default"
	// NamespaceTestAlternative is used to test controller-namespace independency.
	NamespaceTestAlternative = "kubevirt-test-alternative"
	// NamespaceTestOperator is used to test if namespaces can still be deleted when kubevirt is uninstalled
	NamespaceTestOperator = "kubevirt-test-operator"
)

const (
	NFSTargetName   = "test-nfs-target"
	ISCSITargetName = "test-isci-target"
)

var testNamespaces = []string{NamespaceTestDefault, NamespaceTestAlternative, NamespaceTestOperator}
var schedulableNode = ""

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

const (
	// BlockDiskForTest contains name of the block PV and PVC
	BlockDiskForTest = "block-disk-for-tests"
)

const (
	k8sAuditLogPath = "/var/log/k8s-audit/k8s-audit.log"
	osAuditLogPath  = "/var/lib/origin/audit-ocp.log"
	swaggerPath     = "api/openapi-spec/swagger.json"
	artifactsEnv    = "ARTIFACTS"
	tmpPath         = "/tmp/kubevirt.io/tests"
)

const (
	capNetAdmin k8sv1.Capability = "NET_ADMIN"
	capNetRaw   k8sv1.Capability = "NET_RAW"
	capSysNice  k8sv1.Capability = "SYS_NICE"
)

type ProcessFunc func(event *k8sv1.Event) (done bool)

type ObjectEventWatcher struct {
	object                 runtime.Object
	timeout                *time.Duration
	failOnWarnings         bool
	resourceVersion        string
	startType              startType
	dontFailOnMissingEvent bool
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

func (w *ObjectEventWatcher) Watch(abortChan chan struct{}, processFunc ProcessFunc, watchedDescription string) {
	Expect(w.startType).ToNot(Equal(invalidWatch))
	resourceVersion := ""

	switch w.startType {
	case watchSinceNow:
		resourceVersion = ""
	case watchSinceObjectUpdate, watchSinceResourceVersion:
		resourceVersion = w.resourceVersion
	case watchSinceWatchedObjectUpdate:
		var err error
		resourceVersion, err = meta.NewAccessor().ResourceVersion(w.object)
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
		case <-abortChan:
		case <-time.After(*w.timeout):
			if !w.dontFailOnMissingEvent {
				Fail(fmt.Sprintf("Waited for %v seconds on the event stream to match a specific event: %s", w.timeout.Seconds(), watchedDescription), 1)
			}
		}
	} else {
		select {
		case <-abortChan:
		case <-done:
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
	}, fmt.Sprintf("event type %s, reason = %s", string(eventType), reflect.ValueOf(reason).String()))
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
	}, fmt.Sprintf("not happen event type %s, reason = %s", string(eventType), reflect.ValueOf(reason).String()))
	return
}

// Do scale and retuns error, replicas-before.
func DoScaleDeployment(namespace string, name string, desired int32) (error, int32) {
	virtCli, err := kubecli.GetKubevirtClient()
	PanicOnError(err)

	deployment, err := virtCli.AppsV1().Deployments(namespace).Get(name, metav1.GetOptions{})
	if err != nil {
		return err, -1
	}
	scale := &autoscalingv1.Scale{Spec: autoscalingv1.ScaleSpec{Replicas: desired}, ObjectMeta: metav1.ObjectMeta{Name: name, Namespace: namespace}}
	scale, err = virtCli.AppsV1().Deployments(namespace).UpdateScale(name, scale)
	if err != nil {
		return err, -1
	}
	return nil, *deployment.Spec.Replicas
}

func DoScaleVirtHandler(namespace string, name string, selector map[string]string) (int32, map[string]string, int64, error) {
	virtCli, err := kubecli.GetKubevirtClient()
	PanicOnError(err)

	d, err := virtCli.AppsV1().DaemonSets(namespace).Get(name, metav1.GetOptions{})
	if err != nil {
		return 0, nil, 0, err
	}
	sel := d.Spec.Template.Spec.NodeSelector
	ready := d.Status.DesiredNumberScheduled
	d.Spec.Template.Spec.NodeSelector = selector
	d, err = virtCli.AppsV1().DaemonSets(namespace).Update(d)
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
	Eventually(checkForPodsToBeReady, timeout, 2*time.Second).Should(BeEmpty(), "There are pods in system which are not ready.")
}

func GenerateRESTReport() error {
	virtClient, err := kubecli.GetKubevirtClient()
	if err != nil {
		return err
	}

	auditLogPath := k8sAuditLogPath
	if IsOpenShift() {
		v := cluster.GetOpenShiftMajorVersion(virtClient)
		if v == cluster.OpenShift4Major {
			// audit log for OKD4 is not available yet and the path is unknown
			log.Log.Info("Skipping the REST API report on OKD4")
			return nil
		}
		auditLogPath = osAuditLogPath
	}
	logs, _, err := RunCommandWithNS("", "gocli", "scp", auditLogPath, "-")
	if err != nil {
		return err
	}

	tmpFile, err := ioutil.TempFile(os.TempDir(), "audit-")
	if err != nil {
		return err
	}
	defer os.Remove(tmpFile.Name())

	if _, err = tmpFile.Write([]byte(logs)); err != nil {
		return err
	}
	if err = tmpFile.Close(); err != nil {
		return err
	}

	coverage, err := covreport.Generate(tmpFile.Name(), swaggerPath, "/apis/kubevirt.io")
	if err != nil {
		return err
	}

	artifactsPath := os.Getenv(artifactsEnv)
	if artifactsPath == "" {
		return fmt.Errorf("ARTIFACTS env is empty, unable to save the coverage report")
	}
	if _, err := os.Stat(artifactsPath); os.IsNotExist(err) {
		if err = os.MkdirAll(artifactsPath, 0755); err != nil {
			return err
		}
	} else if err != nil {
		return err
	}

	err = covreport.Dump(filepath.Join(artifactsPath, "rest-coverage.json"), coverage)
	if err != nil {
		return err
	}

	return nil
}

func AfterTestSuitCleanup() {

	RestoreKubeVirtResource()

	// Make sure that the namespaces exist, to not have to check in the cleanup code for existing namespaces
	createNamespaces()
	cleanNamespaces()
	cleanupServiceAccounts()

	DeletePVC(osWindows)
	DeletePVC(osRhel)

	DeletePVC(osAlpineHostPath)
	DeletePV(osAlpineHostPath)

	if Config.ManageStorageClasses {
		deleteStorageClass(Config.StorageClassHostPath)
		deleteStorageClass(Config.StorageClassBlockVolume)
	}

	if DeployTestingInfrastructureFlag {
		WipeTestingInfrastructure()
	}
	removeNamespaces()

	CleanNodes()

	// Generate REST API coverage report
	if err := GenerateRESTReport(); err != nil {
		log.Log.Errorf("Could not generate REST coverage report %v", err)
	}
}

func BeforeTestCleanup() {
	deleteBlockPVAndPVC()
	DeletePVC(CustomHostPath)
	DeletePV(CustomHostPath)
	cleanNamespaces()
	CleanNodes()
}

func CleanNodes() {
	virtCli, err := kubecli.GetKubevirtClient()
	PanicOnError(err)
	nodes := GetAllSchedulableNodes(virtCli).Items

	clusterDrainKey := GetNodeDrainKey()

	for _, node := range nodes {

		old, err := json.Marshal(node)
		Expect(err).ToNot(HaveOccurred())
		new := node.DeepCopy()

		found := false
		taints := []k8sv1.Taint{}
		for _, taint := range node.Spec.Taints {

			if taint.Key == clusterDrainKey && taint.Effect == k8sv1.TaintEffectNoSchedule {
				found = true
			} else if taint.Key == "kubevirt.io/drain" && taint.Effect == k8sv1.TaintEffectNoSchedule {
				// this key is used as a fallback if the original drain key is built-in
				found = true
			} else if taint.Key == "kubevirt.io/alt-drain" && taint.Effect == k8sv1.TaintEffectNoSchedule {
				// this key is used in testing as a custom alternate drain key
				found = true
			} else {
				taints = append(taints, taint)
			}

		}
		new.Spec.Taints = taints

		for k, _ := range node.Labels {
			if strings.HasPrefix(k, "tests.kubevirt.io") {
				found = true
				delete(new.Labels, k)
			}
		}

		if node.Spec.Unschedulable {
			new.Spec.Unschedulable = false
		}

		if !found {
			continue
		}
		newJson, err := json.Marshal(new)
		Expect(err).ToNot(HaveOccurred())

		patch, err := strategicpatch.CreateTwoWayMergePatch(old, newJson, node)
		Expect(err).ToNot(HaveOccurred())

		_, err = virtCli.CoreV1().Nodes().Patch(node.Name, types.StrategicMergePatchType, patch)
		Expect(err).ToNot(HaveOccurred())
	}
}

func AddLabelToNode(nodeName string, key string, value string) {
	virtCli, err := kubecli.GetKubevirtClient()
	PanicOnError(err)
	node, err := virtCli.CoreV1().Nodes().Get(nodeName, metav1.GetOptions{})
	Expect(err).ToNot(HaveOccurred())

	old, err := json.Marshal(node)
	Expect(err).ToNot(HaveOccurred())
	new := node.DeepCopy()
	new.Labels[key] = value

	newJson, err := json.Marshal(new)
	Expect(err).ToNot(HaveOccurred())

	patch, err := strategicpatch.CreateTwoWayMergePatch(old, newJson, node)
	Expect(err).ToNot(HaveOccurred())

	_, err = virtCli.CoreV1().Nodes().Patch(node.Name, types.StrategicMergePatchType, patch)
	Expect(err).ToNot(HaveOccurred())
}

func RemoveLabelFromNode(nodeName string, key string) {
	virtCli, err := kubecli.GetKubevirtClient()
	PanicOnError(err)
	node, err := virtCli.CoreV1().Nodes().Get(nodeName, metav1.GetOptions{})
	Expect(err).ToNot(HaveOccurred())

	if _, exists := node.Labels[key]; !exists {
		return
	}

	old, err := json.Marshal(node)
	Expect(err).ToNot(HaveOccurred())
	new := node.DeepCopy()
	delete(new.Labels, key)

	newJson, err := json.Marshal(new)
	Expect(err).ToNot(HaveOccurred())

	patch, err := strategicpatch.CreateTwoWayMergePatch(old, newJson, node)
	Expect(err).ToNot(HaveOccurred())

	_, err = virtCli.CoreV1().Nodes().Patch(node.Name, types.StrategicMergePatchType, patch)
	Expect(err).ToNot(HaveOccurred())
}

func Taint(nodeName string, key string, effect k8sv1.TaintEffect) {
	virtCli, err := kubecli.GetKubevirtClient()
	PanicOnError(err)
	node, err := virtCli.CoreV1().Nodes().Get(nodeName, metav1.GetOptions{})
	Expect(err).ToNot(HaveOccurred())

	old, err := json.Marshal(node)
	Expect(err).ToNot(HaveOccurred())
	new := node.DeepCopy()
	new.Spec.Taints = append(new.Spec.Taints, k8sv1.Taint{
		Key:    key,
		Effect: effect,
	})

	newJson, err := json.Marshal(new)
	Expect(err).ToNot(HaveOccurred())

	patch, err := strategicpatch.CreateTwoWayMergePatch(old, newJson, node)
	Expect(err).ToNot(HaveOccurred())

	_, err = virtCli.CoreV1().Nodes().Patch(node.Name, types.StrategicMergePatchType, patch)
	Expect(err).ToNot(HaveOccurred())
}

func BeforeTestSuitSetup() {
	log.InitializeLogging("tests")
	log.Log.SetIOWriter(GinkgoWriter)

	var err error
	Config, err = loadConfig()
	Expect(err).ToNot(HaveOccurred())

	createNamespaces()
	createServiceAccounts()
	if DeployTestingInfrastructureFlag {
		WipeTestingInfrastructure()
		DeployTestingInfrastructure()
	}

	// Wait for schedulable nodes
	virtClient, err := kubecli.GetKubevirtClient()
	PanicOnError(err)
	Eventually(func() int {
		nodes := GetAllSchedulableNodes(virtClient)
		if len(nodes.Items) > 0 {
			schedulableNode = nodes.Items[0].Name
		}
		return len(nodes.Items)
	}, 5*time.Minute, 10*time.Second).ShouldNot(BeZero(), "no schedulable nodes found")

	if Config.ManageStorageClasses {
		createStorageClass(Config.StorageClassHostPath)
		createStorageClass(Config.StorageClassBlockVolume)
	}

	CreateHostPathPv(osAlpineHostPath, HostPathAlpine)
	CreateHostPathPVC(osAlpineHostPath, defaultDiskSize)

	CreatePVC(osWindows, defaultWindowsDiskSize, Config.StorageClassWindows)
	CreatePVC(osRhel, defaultRhelDiskSize, Config.StorageClassRhel)

	if IsRunningOnKindInfraIPv6() {
		createPodPreset("fix-node-uuid", "fake-product-uuid", "virt-launcher", "/kind/product_uuid", "/sys/class/dmi/id/product_uuid")
	}

	EnsureKVMPresent()

	SetDefaultEventuallyTimeout(defaultEventuallyTimeout)
	SetDefaultEventuallyPollingInterval(defaultEventuallyPollingInterval)

	AdjustKubeVirtResource()
}

var originalKV *v1.KubeVirt

func AdjustKubeVirtResource() {
	virtClient, err := kubecli.GetKubevirtClient()
	PanicOnError(err)

	kv := GetCurrentKv(virtClient)
	originalKV = kv.DeepCopy()

	// Rotate very often during the tests to ensure that things are working
	kv.Spec.CertificateRotationStrategy = v1.KubeVirtCertificateRotateStrategy{SelfSigned: &v1.KubeVirtSelfSignConfiguration{
		CARotateInterval:   &metav1.Duration{Duration: 10 * time.Minute},
		CertRotateInterval: &metav1.Duration{Duration: 7 * time.Minute},
		CAOverlapInterval:  &metav1.Duration{Duration: 4 * time.Minute},
	}}

	data, err := json.Marshal(kv.Spec)
	Expect(err).ToNot(HaveOccurred())
	patchData := fmt.Sprintf(`[{ "op": "replace", "path": "/spec", "value": %s }]`, string(data))
	_, err = virtClient.KubeVirt(kv.Namespace).Patch(kv.Name, types.JSONPatchType, []byte(patchData))
	PanicOnError(err)
}

func RestoreKubeVirtResource() {
	virtClient, err := kubecli.GetKubevirtClient()
	PanicOnError(err)

	data, err := json.Marshal(originalKV.Spec)
	Expect(err).ToNot(HaveOccurred())
	patchData := fmt.Sprintf(`[{ "op": "replace", "path": "/spec", "value": %s }]`, string(data))
	_, err = virtClient.KubeVirt(originalKV.Namespace).Patch(originalKV.Name, types.JSONPatchType, []byte(patchData))
	PanicOnError(err)
}

func createStorageClass(name string) {
	virtClient, err := kubecli.GetKubevirtClient()
	PanicOnError(err)

	sc := &storagev1.StorageClass{
		ObjectMeta: metav1.ObjectMeta{
			Name: name,
			Labels: map[string]string{
				"kubevirt.io/test": name,
			},
		},
		Provisioner: name,
	}
	_, err = virtClient.StorageV1().StorageClasses().Create(sc)
	PanicOnError(err)
}

func deleteStorageClass(name string) {
	virtClient, err := kubecli.GetKubevirtClient()
	PanicOnError(err)

	_, err = virtClient.StorageV1().StorageClasses().Get(name, metav1.GetOptions{})
	if errors.IsNotFound(err) {
		return
	}
	PanicOnError(err)

	err = virtClient.StorageV1().StorageClasses().Delete(name, &metav1.DeleteOptions{})
	PanicOnError(err)
}

func EnsureKVMPresent() {
	useEmulation := false
	virtClient, err := kubecli.GetKubevirtClient()
	PanicOnError(err)

	options := metav1.GetOptions{}
	cfgMap, err := virtClient.CoreV1().ConfigMaps(KubeVirtInstallNamespace).Get(virtconfig.ConfigMapName, options)
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
	CreatePVC(os, size, Config.StorageClassHostPath)
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
			StorageClassName: Config.StorageClassHostPath,
			NodeAffinity: &k8sv1.VolumeNodeAffinity{
				Required: &k8sv1.NodeSelector{
					NodeSelectorTerms: []k8sv1.NodeSelectorTerm{
						{
							MatchExpressions: []k8sv1.NodeSelectorRequirement{
								{
									Key:      "kubernetes.io/hostname",
									Operator: k8sv1.NodeSelectorOpIn,
									Values:   []string{schedulableNode},
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
		if !info.IsDir() && matchFileName("*.yaml", info.Name()) {
			manifests = append(manifests, path)
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

	isOpenShift, err := cluster.IsOnOpenShift(virtClient)
	if err != nil {
		fmt.Printf("ERROR: Can not determine cluster type %v\n", err)
		panic(err)
	}

	return isOpenShift
}

func ServiceMonitorEnabled() bool {
	virtClient, err := kubecli.GetKubevirtClient()
	PanicOnError(err)

	serviceMonitorEnabled, err := util.IsServiceMonitorEnabled(virtClient)
	if err != nil {
		fmt.Printf("ERROR: Can't verify ServiceMonitor CRD %v\n", err)
		panic(err)
	}

	return serviceMonitorEnabled
}

// PrometheusRuleEnabled returns true if the PrometheusRule CRD is enabled
// and false otherwise.
func PrometheusRuleEnabled() bool {
	virtClient, err := kubecli.GetKubevirtClient()
	PanicOnError(err)

	prometheusRuleEnabled, err := util.IsPrometheusRuleEnabled(virtClient)
	if err != nil {
		fmt.Printf("ERROR: Can't verify PrometheusRule CRD %v\n", err)
		panic(err)
	}

	return prometheusRuleEnabled
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

	policy := metav1.DeletePropagationBackground
	options := &metav1.DeleteOptions{PropagationPolicy: &policy}

	result := virtCli.CoreV1().RESTClient().Delete().RequestURI(uri).Body(options).Do()
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
		d, err := virtCli.AppsV1().Deployments(KubeVirtInstallNamespace).Get("virt-api", metav1.GetOptions{})
		Expect(err).ToNot(HaveOccurred())
		return d.Status.ReadyReplicas
	}, 3*time.Minute, 2*time.Second).Should(Equal(replicasApi), "virt-api is not ready")

	Eventually(func() int32 {
		d, err := virtCli.AppsV1().Deployments(KubeVirtInstallNamespace).Get("virt-controller", metav1.GetOptions{})
		Expect(err).ToNot(HaveOccurred())
		return d.Status.ReadyReplicas
	}, 3*time.Minute, 2*time.Second).Should(Equal(replicasController), "virt-controller is not ready")

	Eventually(func() int64 {
		d, err := virtCli.AppsV1().DaemonSets(KubeVirtInstallNamespace).Get("virt-handler", metav1.GetOptions{})
		Expect(err).ToNot(HaveOccurred())
		return d.Status.ObservedGeneration
	}, 1*time.Minute, 2*time.Second).Should(Equal(newGeneration), "virt-handler did not bump the generation")

	Eventually(func() int32 {
		d, err := virtCli.AppsV1().DaemonSets(KubeVirtInstallNamespace).Get("virt-handler", metav1.GetOptions{})
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

func createPodPreset(podPresetName, volName, label, srcPath, dstPath string) {
	virtCli, err := kubecli.GetKubevirtClient()
	PanicOnError(err)

	hostPathType := k8sv1.HostPathFile
	podPreset := settingsv1alpha1.PodPreset{
		ObjectMeta: metav1.ObjectMeta{
			Name:      podPresetName,
			Namespace: NamespaceTestDefault,
		},
		Spec: settingsv1alpha1.PodPresetSpec{
			Selector: metav1.LabelSelector{
				MatchLabels: map[string]string{
					"kubevirt.io": label,
				},
			},
			Volumes: []k8sv1.Volume{
				{
					Name: volName,
					VolumeSource: k8sv1.VolumeSource{
						HostPath: &k8sv1.HostPathVolumeSource{
							Path: srcPath,
							Type: &hostPathType,
						},
					},
				},
			},
			VolumeMounts: []k8sv1.VolumeMount{
				{
					Name:      volName,
					MountPath: dstPath,
				},
			},
		},
	}

	_, err = virtCli.SettingsV1alpha1().PodPresets(NamespaceTestDefault).Create(&podPreset)
	if !errors.IsAlreadyExists(err) {
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

func RunVMIAndExpectLaunch(vmi *v1.VirtualMachineInstance, timeout int) *v1.VirtualMachineInstance {
	obj := RunVMI(vmi, timeout)
	By("Waiting until the VirtualMachineInstance will start")
	WaitForSuccessfulVMIStartWithTimeout(obj, timeout)
	return obj
}

func RunVMIAndExpectLaunchIgnoreWarnings(vmi *v1.VirtualMachineInstance, timeout int) *v1.VirtualMachineInstance {
	obj := RunVMI(vmi, timeout)
	By("Waiting until the VirtualMachineInstance will start")
	WaitForSuccessfulVMIStartWithTimeoutIgnoreWarnings(obj, timeout)
	return obj
}

func RunVMIAndExpectLaunchWithIgnoreWarningArg(vmi *v1.VirtualMachineInstance, timeout int, ignoreWarnings bool) *v1.VirtualMachineInstance {
	By("Starting a VirtualMachineInstance")
	var obj *v1.VirtualMachineInstance
	var err error
	virtClient, err := kubecli.GetKubevirtClient()
	Expect(err).ToNot(HaveOccurred())
	Eventually(func() error {
		obj, err = virtClient.VirtualMachineInstance(NamespaceTestDefault).Create(vmi)
		return err
	}, timeout, 1*time.Second).ShouldNot(HaveOccurred())
	By("Waiting until the VirtualMachineInstance starts")
	if ignoreWarnings {
		WaitForSuccessfulVMIStartWithTimeoutIgnoreWarnings(obj, timeout)
	} else {
		WaitForSuccessfulVMIStartWithTimeout(obj, timeout)
	}
	vmi, err = virtClient.VirtualMachineInstance(NamespaceTestDefault).Get(vmi.Name, &metav1.GetOptions{})
	Expect(err).ToNot(HaveOccurred())
	return vmi
}

func RunVMIAndExpectScheduling(vmi *v1.VirtualMachineInstance, timeout int) *v1.VirtualMachineInstance {
	obj := RunVMI(vmi, timeout)
	By("Waiting until the VirtualMachineInstance will be scheduled")
	waitForVMIScheduling(obj, timeout, false)
	return obj
}

func getRunningPodByVirtualMachineInstance(vmi *v1.VirtualMachineInstance, namespace string) (*k8sv1.Pod, error) {
	virtCli, err := kubecli.GetKubevirtClient()
	if err != nil {
		return nil, err
	}

	vmi, err = virtCli.VirtualMachineInstance(namespace).Get(vmi.Name, &metav1.GetOptions{})
	if err != nil {
		return nil, err
	}

	return GetRunningPodByLabel(string(vmi.GetUID()), v1.CreatedByLabel, namespace, vmi.Status.NodeName)
}

func GetRunningPodByVirtualMachineInstance(vmi *v1.VirtualMachineInstance, namespace string) *k8sv1.Pod {
	pod, err := getRunningPodByVirtualMachineInstance(vmi, namespace)
	PanicOnError(err)
	return pod
}

func GetRunningPodByLabel(label string, labelType string, namespace string, node string) (*k8sv1.Pod, error) {
	virtCli, err := kubecli.GetKubevirtClient()
	if err != nil {
		return nil, err
	}

	labelSelector := fmt.Sprintf("%s=%s", labelType, label)
	var fieldSelector string
	if node != "" {
		fieldSelector = fmt.Sprintf("status.phase==%s,spec.nodeName==%s", k8sv1.PodRunning, node)
	} else {
		fieldSelector = fmt.Sprintf("status.phase==%s", k8sv1.PodRunning)
	}
	pods, err := virtCli.CoreV1().Pods(namespace).List(
		metav1.ListOptions{LabelSelector: labelSelector, FieldSelector: fieldSelector},
	)
	if err != nil {
		return nil, err
	}

	if len(pods.Items) == 0 {
		return nil, fmt.Errorf("failed to find pod with the label %s", label)
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
		return nil, fmt.Errorf("no ready pods with the label %s", label)
	}

	return readyPod, nil
}

func GetPodByVirtualMachineInstance(vmi *v1.VirtualMachineInstance, namespace string) *k8sv1.Pod {
	virtCli, err := kubecli.GetKubevirtClient()
	PanicOnError(err)

	pods, err := virtCli.CoreV1().Pods(namespace).List(metav1.ListOptions{})
	PanicOnError(err)

	var controlledPod *k8sv1.Pod
	for _, pod := range pods.Items {
		if controller.IsControlledBy(&pod, vmi) {
			controlledPod = &pod
			break
		}
	}

	if controlledPod == nil {
		PanicOnError(fmt.Errorf("no controlled pod was found for VMI"))
	}

	return controlledPod
}

func GetComputeContainerOfPod(pod *k8sv1.Pod) *k8sv1.Container {
	return GetContainerOfPod(pod, "compute")
}

func GetContainerDiskContainerOfPod(pod *k8sv1.Pod, volumeName string) *k8sv1.Container {
	diskContainerName := fmt.Sprintf("volume%s", volumeName)
	return GetContainerOfPod(pod, diskContainerName)
}

func GetContainerOfPod(pod *k8sv1.Pod, containerName string) *k8sv1.Container {
	var computeContainer *k8sv1.Container
	for _, container := range pod.Spec.Containers {
		if container.Name == containerName {
			computeContainer = &container
			break
		}
	}
	if computeContainer == nil {
		PanicOnError(fmt.Errorf("could not find the %s container", containerName))
	}
	return computeContainer
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

		// Remove all Services
		svcList, err := virtCli.CoreV1().Services(namespace).List(metav1.ListOptions{})
		for _, svc := range svcList.Items {
			PanicOnError(virtCli.CoreV1().Services(namespace).Delete(svc.Name, &metav1.DeleteOptions{}))
		}

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
		migrations, err := virtCli.VirtualMachineInstanceMigration(namespace).List(&metav1.ListOptions{})
		PanicOnError(err)
		for _, migration := range migrations.Items {
			if controller.HasFinalizer(&migration, v1.VirtualMachineInstanceMigrationFinalizer) {
				_, err := virtCli.VirtualMachineInstanceMigration(namespace).Patch(migration.Name, types.JSONPatchType, []byte("[{ \"op\": \"remove\", \"path\": \"/metadata/finalizers\" }]"))
				if !errors.IsNotFound(err) {
					PanicOnError(err)
				}
			}
		}

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
		EventuallyWithOffset(1, func() bool { return errors.IsNotFound(virtCli.CoreV1().Namespaces().Delete(namespace, nil)) }, 240*time.Second, 1*time.Second).
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

func NewRandomDataVolumeWithHttpImport(imageUrl, namespace string, accessMode k8sv1.PersistentVolumeAccessMode) *cdiv1.DataVolume {
	return newRandomDataVolumeWithHttpImport(imageUrl, namespace, Config.StorageClassLocal, accessMode)
}

func NewRandomVirtualMachineInstanceWithOCSDisk(imageUrl, namespace string, accessMode k8sv1.PersistentVolumeAccessMode, volMode k8sv1.PersistentVolumeMode) (*v1.VirtualMachineInstance, *cdiv1.DataVolume) {
	if !HasCDI() {
		Skip("Skip DataVolume tests when CDI is not present")
	}
	sc, exists := GetCephStorageClass()
	if !exists {
		Skip("Skip OCS tests when Ceph is not present")
	}
	virtCli, err := kubecli.GetKubevirtClient()
	PanicOnError(err)

	dv := newRandomDataVolumeWithHttpImport(imageUrl, namespace, sc, accessMode)
	dv.Spec.PVC.VolumeMode = &volMode
	_, err = virtCli.CdiClient().CdiV1alpha1().DataVolumes(dv.Namespace).Create(dv)
	Expect(err).ToNot(HaveOccurred())
	WaitForSuccessfulDataVolumeImport(dv, 240)
	return NewRandomVMIWithDataVolume(dv.Name), dv
}

func newRandomDataVolumeWithHttpImport(imageUrl, namespace, storageClass string, accessMode k8sv1.PersistentVolumeAccessMode) *cdiv1.DataVolume {
	name := "test-datavolume-" + rand.String(12)
	quantity, err := resource.ParseQuantity("1Gi")
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
				AccessModes: []k8sv1.PersistentVolumeAccessMode{accessMode},
				Resources: k8sv1.ResourceRequirements{
					Requests: k8sv1.ResourceList{
						"storage": quantity,
					},
				},
				StorageClassName: &storageClass,
			},
		},
	}

	dataVolume.TypeMeta = metav1.TypeMeta{
		APIVersion: "cdi.kubevirt.io/v1alpha1",
		Kind:       "DataVolume",
	}

	return dataVolume
}

func NewRandomDataVolumeWithPVCSource(sourceNamespace, sourceName, targetNamespace string, accessMode k8sv1.PersistentVolumeAccessMode) *cdiv1.DataVolume {
	return NewRandomDataVolumeWithPVCSourceWithStorageClass(sourceNamespace, sourceName, targetNamespace, Config.StorageClassLocal, "1Gi", accessMode)
}

func NewRandomDataVolumeWithPVCSourceWithStorageClass(sourceNamespace, sourceName, targetNamespace, storageClass, size string, accessMode k8sv1.PersistentVolumeAccessMode) *cdiv1.DataVolume {
	name := "test-datavolume-" + rand.String(12)
	quantity, err := resource.ParseQuantity(size)
	PanicOnError(err)
	dataVolume := &cdiv1.DataVolume{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: targetNamespace,
		},
		Spec: cdiv1.DataVolumeSpec{
			Source: cdiv1.DataVolumeSource{
				PVC: &cdiv1.DataVolumeSourcePVC{
					Namespace: sourceNamespace,
					Name:      sourceName,
				},
			},
			PVC: &k8sv1.PersistentVolumeClaimSpec{
				AccessModes: []k8sv1.PersistentVolumeAccessMode{accessMode},
				Resources: k8sv1.ResourceRequirements{
					Requests: k8sv1.ResourceList{
						"storage": quantity,
					},
				},
				StorageClassName: &storageClass,
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

	// To avoid mac address issue in the tests change the pod interface binding to masquerade
	// https://github.com/kubevirt/kubevirt/issues/1494
	vmi.Spec.Domain.Devices = v1.Devices{Interfaces: []v1.Interface{{Name: "default",
		InterfaceBindingMethod: v1.InterfaceBindingMethod{
			Masquerade: &v1.InterfaceMasquerade{}}}}}

	vmi.Spec.Networks = []v1.Network{*v1.DefaultPodNetwork()}

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

func NewRandomVMWithEphemeralDisk(containerImage string) *v1.VirtualMachine {
	vmi := NewRandomVMIWithEphemeralDisk(containerImage)
	vm := NewRandomVirtualMachine(vmi, false)

	return vm
}

func NewRandomVMWithDataVolume(imageUrl string, namespace string) *v1.VirtualMachine {
	dataVolume := NewRandomDataVolumeWithHttpImport(imageUrl, namespace, k8sv1.ReadWriteOnce)
	vmi := NewRandomVMIWithDataVolume(dataVolume.Name)
	vm := NewRandomVirtualMachine(vmi, false)

	vm.Spec.DataVolumeTemplates = append(vm.Spec.DataVolumeTemplates, *dataVolume)
	return vm
}

func NewRandomVMWithCloneDataVolume(sourceNamespace, sourceName, targetNamespace string) *v1.VirtualMachine {
	dataVolume := NewRandomDataVolumeWithPVCSource(sourceNamespace, sourceName, targetNamespace, k8sv1.ReadWriteOnce)
	vmi := NewRandomVMIWithDataVolume(dataVolume.Name)
	vmi.Namespace = targetNamespace
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

func NewRandomVMIWithEphemeralDiskAndConfigDriveUserdataHighMemory(containerImage string, userData string) *v1.VirtualMachineInstance {
	vmi := NewRandomVMIWithEphemeralDiskAndConfigDriveUserdata(containerImage, userData)

	vmi.Spec.Domain.Resources.Requests[k8sv1.ResourceMemory] = resource.MustParse("512M")
	return vmi
}

func NewRandomVMIWithEFIBootloader() *v1.VirtualMachineInstance {
	vmi := NewRandomVMIWithEphemeralDiskHighMemory(ContainerDiskFor(ContainerDiskAlpine))

	// EFI needs more memory than other images
	vmi.Spec.Domain.Resources.Requests[k8sv1.ResourceMemory] = resource.MustParse("1Gi")
	vmi.Spec.Domain.Firmware = &v1.Firmware{
		Bootloader: &v1.Bootloader{
			EFI: &v1.EFI{
				SecureBoot: NewBool(false),
			},
		},
	}

	return vmi

}

func NewRandomVMIWithSecureBoot() *v1.VirtualMachineInstance {
	vmi := NewRandomVMIWithEphemeralDiskHighMemory(ContainerDiskFor(ContainerDiskMicroLiveCD))

	// EFI needs more memory than other images
	vmi.Spec.Domain.Resources.Requests[k8sv1.ResourceMemory] = resource.MustParse("1Gi")
	vmi.Spec.Domain.Features = &v1.Features{
		SMM: &v1.FeatureState{
			Enabled: NewBool(true),
		},
	}
	vmi.Spec.Domain.Firmware = &v1.Firmware{
		Bootloader: &v1.Bootloader{
			EFI: &v1.EFI{}, // SecureBoot should default to true
		},
	}

	return vmi

}

func NewRandomMigration(vmiName string, namespace string) *v1.VirtualMachineInstanceMigration {
	return &v1.VirtualMachineInstanceMigration{
		TypeMeta: metav1.TypeMeta{
			APIVersion: v1.GroupVersion.String(),
			Kind:       "VirtualMachineInstanceMigration",
		},
		ObjectMeta: metav1.ObjectMeta{
			GenerateName: "test-migration-",
			Namespace:    namespace,
		},
		Spec: v1.VirtualMachineInstanceMigrationSpec{
			VMIName: vmiName,
		},
	}
}

func NewRandomVMIWithEphemeralDisk(containerImage string) *v1.VirtualMachineInstance {
	vmi := NewRandomVMI()

	vmi.Spec.Domain.Resources.Requests[k8sv1.ResourceMemory] = resource.MustParse("64M")
	AddEphemeralDisk(vmi, "disk0", "virtio", containerImage)
	if containerImage == ContainerDiskFor(ContainerDiskFedora) {
		vmi.Spec.Domain.Devices.Rng = &v1.Rng{} // newer fedora kernels may require hardware RNG to boot
	}
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

func NewRandomFedora32VMIWithFedoraUser() *v1.VirtualMachineInstance {
	vmi := NewRandomVMIWithEphemeralDiskAndUserdata(ContainerDiskFor(ContainerDiskFedora), `#!/bin/bash
	    echo "fedora" |passwd fedora --stdin
        echo `)
	vmi.Spec.Domain.Resources.Requests[k8sv1.ResourceMemory] = resource.MustParse("512M")
	return vmi
}

func NewRandomFedoraVMIWitGuestAgent() *v1.VirtualMachineInstance {
	agentVMI := NewRandomVMIWithEphemeralDiskAndUserdata(ContainerDiskFor(ContainerDiskFedora), GetGuestAgentUserData())
	agentVMI.Spec.Domain.Resources.Requests[k8sv1.ResourceMemory] = resource.MustParse("512M")
	return agentVMI
}

func NewRandomFedoraVMIWithDmidecode() *v1.VirtualMachineInstance {
	dmidecodeUserData := fmt.Sprintf(`#!/bin/bash
	    echo "fedora" |passwd fedora --stdin
	    mkdir -p /usr/local/bin
	    curl %s > /usr/local/bin/dmidecode
	    chmod +x /usr/local/bin/dmidecode
	`, GetUrl(DmidecodeHttpUrl))
	vmi := NewRandomVMIWithEphemeralDiskAndUserdataHighMemory(ContainerDiskFor(ContainerDiskFedora), dmidecodeUserData)
	return vmi
}

func GetGuestAgentUserData() string {
	virtClient, err := kubecli.GetKubevirtClient()
	PanicOnError(err)

	dnsServerIP, err := getClusterDnsServiceIP(virtClient)
	PanicOnError(err)

	ipv6UserDataString := ""
	if netutils.IsIPv6String(dnsServerIP) {
		ipv6UserDataString = fmt.Sprintf(`sudo ip -6 addr add fd10:0:2::2/120 dev eth0
                             for i in {1..20}; do sudo ip -6 route add default via fd10:0:2::1 src fd10:0:2::2 && break || sleep 0.1; done
                             echo "nameserver %s" >> /etc/resolv.conf`, dnsServerIP)
	}
	return fmt.Sprintf(`#!/bin/bash
                echo "fedora" |passwd fedora --stdin
                systemctl disable NetworkManager
                systemctl stop NetworkManager
                %s
                mkdir -p /usr/local/bin
                curl %s > /usr/local/bin/qemu-ga
                chmod +x /usr/local/bin/qemu-ga
                curl %s > /usr/local/bin/stress
                chmod +x /usr/local/bin/stress
                setenforce 0
                systemd-run --unit=guestagent /usr/local/bin/qemu-ga
                `, ipv6UserDataString, GetUrl(GuestAgentHttpUrl), GetUrl(StressHttpUrl))
}

func NewRandomVMIWithEphemeralDiskAndUserdata(containerImage string, userData string) *v1.VirtualMachineInstance {
	vmi := NewRandomVMIWithEphemeralDisk(containerImage)
	AddUserData(vmi, "disk1", userData)
	return vmi
}

func NewRandomVMIWithEphemeralDiskAndConfigDriveUserdata(containerImage string, userData string) *v1.VirtualMachineInstance {
	vmi := NewRandomVMIWithEphemeralDisk(containerImage)
	AddCloudInitConfigDriveData(vmi, "disk1", userData, "", false)
	return vmi
}

func NewRandomVMIWithEphemeralDiskAndUserdataNetworkData(containerImage, userData, networkData string, b64encode bool) *v1.VirtualMachineInstance {
	vmi := NewRandomVMIWithEphemeralDisk(containerImage)
	AddCloudInitNoCloudData(vmi, "disk1", userData, networkData, b64encode)
	return vmi
}

func NewRandomVMIWithEphemeralDiskAndConfigDriveUserdataNetworkData(containerImage, userData, networkData string, b64encode bool) *v1.VirtualMachineInstance {
	vmi := NewRandomVMIWithEphemeralDisk(containerImage)
	AddCloudInitConfigDriveData(vmi, "disk1", userData, networkData, b64encode)
	return vmi
}

func AddUserData(vmi *v1.VirtualMachineInstance, name string, userData string) {
	AddCloudInitNoCloudData(vmi, name, userData, "", true)
}

func AddCloudInitNoCloudData(vmi *v1.VirtualMachineInstance, name, userData, networkData string, b64encode bool) {
	cloudInitNoCloudSource := v1.CloudInitNoCloudSource{}
	if b64encode {
		cloudInitNoCloudSource.UserDataBase64 = base64.StdEncoding.EncodeToString([]byte(userData))
		if networkData != "" {
			cloudInitNoCloudSource.NetworkDataBase64 = base64.StdEncoding.EncodeToString([]byte(networkData))
		}
	} else {
		cloudInitNoCloudSource.UserData = userData
		if networkData != "" {
			cloudInitNoCloudSource.NetworkData = networkData
		}
	}
	addCloudInitDiskAndVolume(vmi, name, v1.VolumeSource{CloudInitNoCloud: &cloudInitNoCloudSource})
}

func AddCloudInitConfigDriveData(vmi *v1.VirtualMachineInstance, name, userData, networkData string, b64encode bool) {
	cloudInitConfigDriveSource := v1.CloudInitConfigDriveSource{}
	if b64encode {
		cloudInitConfigDriveSource.UserDataBase64 = base64.StdEncoding.EncodeToString([]byte(userData))
		if networkData != "" {
			cloudInitConfigDriveSource.NetworkDataBase64 = base64.StdEncoding.EncodeToString([]byte(networkData))
		}
	} else {
		cloudInitConfigDriveSource.UserData = userData
		if networkData != "" {
			cloudInitConfigDriveSource.NetworkData = networkData
		}
	}
	addCloudInitDiskAndVolume(vmi, name, v1.VolumeSource{CloudInitConfigDrive: &cloudInitConfigDriveSource})
}

func addCloudInitDiskAndVolume(vmi *v1.VirtualMachineInstance, name string, volumeSource v1.VolumeSource) {
	vmi.Spec.Domain.Devices.Disks = append(vmi.Spec.Domain.Devices.Disks, v1.Disk{
		Name: name,
		DiskDevice: v1.DiskDevice{
			Disk: &v1.DiskTarget{
				Bus: "virtio",
			},
		},
	})
	vmi.Spec.Volumes = append(vmi.Spec.Volumes, v1.Volume{
		Name:         name,
		VolumeSource: volumeSource,
	})
}

func NewRandomVMIWithPVC(claimName string) *v1.VirtualMachineInstance {
	vmi := NewRandomVMI()
	vmi.Spec.Domain.Resources.Requests[k8sv1.ResourceMemory] = resource.MustParse("64M")
	vmi = AddPVCDisk(vmi, "disk0", "virtio", claimName)
	return vmi
}

func CreateBlockVolumePvAndPvc(size string) {
	virtCli, err := kubecli.GetKubevirtClient()
	PanicOnError(err)

	labelSelector := make(map[string]string)
	labelSelector["kubevirt-test"] = BlockDiskForTest

	_, err = virtCli.CoreV1().PersistentVolumes().Create(newBlockVolumePV(BlockDiskForTest, labelSelector, size))
	if !errors.IsAlreadyExists(err) {
		PanicOnError(err)
	}

	_, err = virtCli.CoreV1().PersistentVolumeClaims(NamespaceTestDefault).Create(newBlockVolumePVC(BlockDiskForTest, labelSelector, size))
	if !errors.IsAlreadyExists(err) {
		PanicOnError(err)
	}
}

func newBlockVolumePV(name string, labelSelector map[string]string, size string) *k8sv1.PersistentVolume {
	quantity, err := resource.ParseQuantity(size)
	PanicOnError(err)

	storageClass := Config.StorageClassBlockVolume
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
									Values:   []string{schedulableNode},
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

	storageClass := Config.StorageClassBlockVolume
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
	if !errors.IsNotFound(err) {
		PanicOnError(err)
	}

	err = virtCli.CoreV1().PersistentVolumeClaims(NamespaceTestDefault).Delete(name, &metav1.DeleteOptions{})
	if !errors.IsNotFound(err) {
		PanicOnError(err)
	}
}

func deleteBlockPVAndPVC() {
	DeletePvAndPvc(BlockDiskForTest)
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
	AddConfigMapDisk(vmi, configMapName, configMapName)
	return vmi
}

func AddConfigMapDisk(vmi *v1.VirtualMachineInstance, configMapName string, volumeName string) {
	AddConfigMapDiskWithCustomLabel(vmi, configMapName, volumeName, "")

}
func AddConfigMapDiskWithCustomLabel(vmi *v1.VirtualMachineInstance, configMapName string, volumeName string, volumeLabel string) {
	vmi.Spec.Volumes = append(vmi.Spec.Volumes, v1.Volume{
		Name: volumeName,
		VolumeSource: v1.VolumeSource{
			ConfigMap: &v1.ConfigMapVolumeSource{
				LocalObjectReference: k8sv1.LocalObjectReference{
					Name: configMapName,
				},
				VolumeLabel: volumeLabel,
			},
		},
	})
	vmi.Spec.Domain.Devices.Disks = append(vmi.Spec.Domain.Devices.Disks, v1.Disk{
		Name: volumeName,
	})
}

func NewRandomVMIWithSecret(secretName string) *v1.VirtualMachineInstance {
	vmi := NewRandomVMIWithPVC(DiskAlpineHostPath)
	AddSecretDisk(vmi, secretName, secretName)
	return vmi
}

func AddSecretDisk(vmi *v1.VirtualMachineInstance, secretName string, volumeName string) {
	AddSecretDiskWithCustomLabel(vmi, secretName, volumeName, "")
}

func AddSecretDiskWithCustomLabel(vmi *v1.VirtualMachineInstance, secretName string, volumeName string, volumeLabel string) {
	vmi.Spec.Volumes = append(vmi.Spec.Volumes, v1.Volume{
		Name: volumeName,
		VolumeSource: v1.VolumeSource{
			Secret: &v1.SecretVolumeSource{
				SecretName:  secretName,
				VolumeLabel: volumeLabel,
			},
		},
	})
	vmi.Spec.Domain.Devices.Disks = append(vmi.Spec.Domain.Devices.Disks, v1.Disk{
		Name: volumeName,
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

func NewRandomVMIWithMasqueradeInterfaceEphemeralDiskAndUserdata(containerImage string, userData string, Ports []v1.Port) *v1.VirtualMachineInstance {
	vmi := NewRandomVMIWithEphemeralDiskAndUserdata(containerImage, userData)
	vmi.Spec.Domain.Devices.Interfaces = []v1.Interface{{Name: "default", Ports: Ports, InterfaceBindingMethod: v1.InterfaceBindingMethod{Masquerade: &v1.InterfaceMasquerade{}}}}
	vmi.Spec.Networks = []v1.Network{*v1.DefaultPodNetwork()}

	return vmi
}

func AddExplicitPodNetworkInterface(vmi *v1.VirtualMachineInstance) {
	vmi.Spec.Domain.Devices.Interfaces = []v1.Interface{*v1.DefaultMasqueradeNetworkInterface()}
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
func WaitForSuccessfulDataVolumeImportOfVMI(obj runtime.Object, seconds int) {
	vmi, ok := obj.(*v1.VirtualMachineInstance)
	ExpectWithOffset(1, ok).To(BeTrue(), "Object is not of type *v1.VMI")
	waitForSuccessfulDataVolumeImport(vmi.Namespace, vmi.Spec.Volumes[0].DataVolume.Name, seconds)
}

func WaitForSuccessfulDataVolumeImport(dv *cdiv1.DataVolume, seconds int) {
	waitForSuccessfulDataVolumeImport(dv.Namespace, dv.Name, seconds)
}

func waitForSuccessfulDataVolumeImport(namespace, name string, seconds int) {
	By("Checking that the DataVolume has succeeded")
	virtClient, err := kubecli.GetKubevirtClient()
	ExpectWithOffset(2, err).ToNot(HaveOccurred())

	EventuallyWithOffset(2, func() cdiv1.DataVolumePhase {
		dv, err := virtClient.CdiClient().CdiV1alpha1().DataVolumes(namespace).Get(name, metav1.GetOptions{})
		ExpectWithOffset(2, err).ToNot(HaveOccurred())

		return dv.Status.Phase
	}, time.Duration(seconds)*time.Second, 1*time.Second).Should(Equal(cdiv1.Succeeded), "Timed out waiting for DataVolume to enter Succeeded phase")
}

// Block until the specified VirtualMachineInstance started and return the target node name.
func waitForVMIStart(obj runtime.Object, seconds int, ignoreWarnings bool) (nodeName string) {
	return waitForVMIPhase([]v1.VirtualMachineInstancePhase{v1.Running}, obj, seconds, ignoreWarnings)
}

// Block until the specified VirtualMachineInstance scheduled and return the target node name.
func waitForVMIScheduling(obj runtime.Object, seconds int, ignoreWarnings bool) {
	waitForVMIPhase([]v1.VirtualMachineInstancePhase{v1.Scheduling, v1.Scheduled, v1.Running}, obj, seconds, ignoreWarnings)
}

func waitForVMIPhase(phases []v1.VirtualMachineInstancePhase, obj runtime.Object, seconds int, ignoreWarnings bool) (nodeName string) {
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

	timeoutMsg := fmt.Sprintf("Timed out waiting for VMI %s to enter %s phase(s)", vmi.Name, phases)
	// FIXME the event order is wrong. First the document should be updated
	EventuallyWithOffset(1, func() v1.VirtualMachineInstancePhase {
		vmi, err = virtClient.VirtualMachineInstance(vmi.Namespace).Get(vmi.Name, &metav1.GetOptions{})
		ExpectWithOffset(1, err).ToNot(HaveOccurred())

		nodeName = vmi.Status.NodeName
		Expect(vmi.IsFinal()).To(BeFalse(), "VMI %s unexpectedly stopped. State: %s", vmi.Name, vmi.Status.Phase)
		return vmi.Status.Phase
	}, time.Duration(seconds)*time.Second, 1*time.Second).Should(BeElementOf(phases), timeoutMsg)

	return
}

func WaitForSuccessfulVMIStartIgnoreWarnings(vmi runtime.Object) string {
	return waitForVMIStart(vmi, 180, true)
}

func WaitForSuccessfulVMIStartWithTimeout(vmi runtime.Object, seconds int) (nodeName string) {
	return waitForVMIStart(vmi, seconds, false)
}

func WaitForSuccessfulVMIStartWithTimeoutIgnoreWarnings(vmi runtime.Object, seconds int) string {
	return waitForVMIStart(vmi, seconds, true)
}

func WaitForPodToDisappearWithTimeout(podName string, seconds int) {
	virtClient, err := kubecli.GetKubevirtClient()
	ExpectWithOffset(1, err).ToNot(HaveOccurred())
	EventuallyWithOffset(1, func() bool {
		_, err := virtClient.CoreV1().Pods(NamespaceTestDefault).Get(podName, metav1.GetOptions{})
		return errors.IsNotFound(err)
	}, seconds, 1*time.Second).Should(BeTrue())
}

func WaitForVirtualMachineToDisappearWithTimeout(vmi *v1.VirtualMachineInstance, seconds int) {
	virtClient, err := kubecli.GetKubevirtClient()
	ExpectWithOffset(1, err).ToNot(HaveOccurred())
	EventuallyWithOffset(1, func() bool {
		_, err := virtClient.VirtualMachineInstance(vmi.Namespace).Get(vmi.Name, &metav1.GetOptions{})
		return errors.IsNotFound(err)
	}, seconds, 1*time.Second).Should(BeTrue())
}

func WaitForMigrationToDisappearWithTimeout(migration *v1.VirtualMachineInstanceMigration, seconds int) {
	virtClient, err := kubecli.GetKubevirtClient()
	ExpectWithOffset(1, err).ToNot(HaveOccurred())
	EventuallyWithOffset(1, func() bool {
		_, err := virtClient.VirtualMachineInstanceMigration(migration.Namespace).Get(migration.Name, &metav1.GetOptions{})
		return errors.IsNotFound(err)
	}, seconds, 1*time.Second).Should(BeTrue())
}

func WaitForSuccessfulVMIStart(vmi runtime.Object) string {
	return waitForVMIStart(vmi, 360, false)
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
					Image:   fmt.Sprintf("%s/vm-killer:%s", KubeVirtUtilityRepoPrefix, KubeVirtUtilityVersionTag),
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
	con, err := virtCli.VirtualMachineInstance(vmi.Namespace).SerialConsole(vmi.Name, &kubecli.SerialConsoleOptions{ConnectionTimeout: timeout})
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

	opts = append(opts, expect.SendTimeout(timeout))
	opts = append(opts, expect.Verbose(true))
	opts = append(opts, expect.VerboseWriter(GinkgoWriter))
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
	ContainerDiskCirrosCustomLocation ContainerDisk = "cirros-custom"
	ContainerDiskCirros               ContainerDisk = "cirros"
	ContainerDiskAlpine               ContainerDisk = "alpine"
	ContainerDiskFedora               ContainerDisk = "fedora-cloud"
	ContainerDiskMicroLiveCD          ContainerDisk = "microlivecd"
	ContainerDiskVirtio               ContainerDisk = "virtio-container-disk"
	ContainerDiskEmpty                ContainerDisk = "empty"
)

// ContainerDiskFor takes the name of an image and returns the full
// registry diks image path.
// Supported values are: cirros, fedora, alpine, guest-agent
func ContainerDiskFor(name ContainerDisk) string {
	switch name {
	case ContainerDiskCirros, ContainerDiskAlpine, ContainerDiskFedora, ContainerDiskMicroLiveCD, ContainerDiskCirrosCustomLocation:
		return fmt.Sprintf("%s/%s-container-disk-demo:%s", KubeVirtUtilityRepoPrefix, name, KubeVirtUtilityVersionTag)
	case ContainerDiskVirtio:
		return fmt.Sprintf("%s/virtio-container-disk:%s", KubeVirtUtilityRepoPrefix, KubeVirtUtilityVersionTag)
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

func configureIPv6OnVMI(vmi *v1.VirtualMachineInstance, expecter expect.Expecter, virtClient kubecli.KubevirtClient, prompt string) error {
	hasEth0Iface := func(vmi *v1.VirtualMachineInstance) bool {
		hasNetEth0Batch := append([]expect.Batcher{
			&expect.BSnd{S: "\n"},
			&expect.BExp{R: prompt},
			&expect.BSnd{S: "ip a | grep -q eth0; echo $?\n"},
			&expect.BExp{R: retcode("0")}})
		_, err := expecter.ExpectBatch(hasNetEth0Batch, 30*time.Second)
		return err == nil
	}

	hasGlobalIPv6 := func(vmi *v1.VirtualMachineInstance) bool {
		hasGlobalIPv6Batch := append([]expect.Batcher{
			&expect.BSnd{S: "\n"},
			&expect.BExp{R: prompt},
			&expect.BSnd{S: "ip -6 address show dev eth0 scope global | grep -q inet6; echo $?\n"},
			&expect.BExp{R: retcode("0")}})
		_, err := expecter.ExpectBatch(hasGlobalIPv6Batch, 30*time.Second)
		return err == nil
	}

	dnsServerIP, err := getClusterDnsServiceIP(virtClient)
	if err != nil {
		log.DefaultLogger().Object(vmi).Infof("Couldn't get DNS Service IP while configuring ipv6: %v", err)
		expecter.Close()
		return err
	}

	if !netutils.IsIPv6String(dnsServerIP) ||
		(vmi.Spec.Domain.Devices.Interfaces == nil || len(vmi.Spec.Domain.Devices.Interfaces) == 0 || vmi.Spec.Domain.Devices.Interfaces[0].InterfaceBindingMethod.Masquerade == nil) ||
		(vmi.Spec.Domain.Devices.AutoattachPodInterface != nil && !*vmi.Spec.Domain.Devices.AutoattachPodInterface) ||
		(!hasEth0Iface(vmi) || hasGlobalIPv6(vmi)) {
		return nil
	}

	addIPv6Address := append([]expect.Batcher{
		&expect.BSnd{S: "\n"},
		&expect.BExp{R: prompt},
		&expect.BSnd{S: "sudo ip -6 addr add fd10:0:2::2/120 dev eth0; echo $?\n"},
		&expect.BExp{R: retcode("0")}})
	resp, err := expecter.ExpectBatch(addIPv6Address, 30*time.Second)
	if err != nil {
		log.DefaultLogger().Object(vmi).Infof("addIPv6Address failed: %v", resp)
		expecter.Close()
		return err
	}

	time.Sleep(5 * time.Second)
	addIPv6DefaultRoute := append([]expect.Batcher{
		&expect.BSnd{S: "\n"},
		&expect.BExp{R: prompt},
		&expect.BSnd{S: "sudo ip -6 route add default via fd10:0:2::1 src fd10:0:2::2; echo $?\n"},
		&expect.BExp{R: retcode("0")}})
	resp, err = expecter.ExpectBatch(addIPv6DefaultRoute, 30*time.Second)
	if err != nil {
		log.DefaultLogger().Object(vmi).Infof("addIPv6DefaultRoute failed: %v", resp)
		expecter.Close()
		return err
	}

	return nil
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

	return expecter, configureIPv6OnVMI(vmi, expecter, virtClient, "\\$ ")
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
		&expect.BExp{R: "localhost:~\\#"}})
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
		&expect.BSnd{S: "\n"},
		&expect.BCas{C: []expect.Caser{
			&expect.Case{
				// Using only "login: " would match things like "Last failed login: Tue Jun  9 22:25:30 UTC 2020 on ttyS0"
				R:  regexp.MustCompile(vmi.Name + ` login: `),
				S:  "fedora\n",
				T:  expect.Next(),
				Rt: 10,
			},
			&expect.Case{
				R:  regexp.MustCompile(`Password:`),
				S:  "fedora\n",
				T:  expect.Next(),
				Rt: 10,
			},
			&expect.Case{
				R:  regexp.MustCompile(`Login incorrect`),
				T:  expect.LogContinue("Failed to log in", expect.NewStatus(codes.PermissionDenied, "login failed")),
				Rt: 10,
			},
			&expect.Case{
				R: regexp.MustCompile(`\$ `),
				T: expect.OK(),
			},
		}},
		&expect.BSnd{S: "sudo su\n"},
		&expect.BExp{R: "\\#"},
	})
	res, err := expecter.ExpectBatch(b, 3*time.Minute)
	if err != nil {
		log.DefaultLogger().Object(vmi).Infof("Login: %+v", res)
		expecter.Close()
		return expecter, err
	}

	return expecter, configureIPv6OnVMI(vmi, expecter, virtClient, "\\#")
}

// ReLoggedInFedoraExpecter return prepared and ready to use console expecter for
// Fedora test VM, when you are reconnecting (no login needed)
func ReLoggedInFedoraExpecter(vmi *v1.VirtualMachineInstance, timeout int) (expect.Expecter, error) {
	virtClient, err := kubecli.GetKubevirtClient()
	PanicOnError(err)
	expecter, _, err := NewConsoleExpecter(virtClient, vmi, 10*time.Second)
	if err != nil {
		return nil, err
	}
	b := append([]expect.Batcher{
		&expect.BSnd{S: "\n"},
		&expect.BExp{R: "#"}})
	res, err := expecter.ExpectBatch(b, time.Duration(timeout)*time.Second)
	if err != nil {
		log.DefaultLogger().Object(vmi).Infof("Login: %+v", res)
		expecter.Close()
		return expecter, err
	}
	return expecter, err
}

func SecureBootExpecter(vmi *v1.VirtualMachineInstance) (expect.Expecter, error) {
	virtClient, err := kubecli.GetKubevirtClient()
	PanicOnError(err)
	expecter, _, err := NewConsoleExpecter(virtClient, vmi, 10*time.Second)
	if err != nil {
		return nil, err
	}
	b := append([]expect.Batcher{
		&expect.BExp{R: "secureboot: Secure boot enabled"},
	})
	res, err := expecter.ExpectBatch(b, 180*time.Second)
	if err != nil {
		log.DefaultLogger().Object(vmi).Infof("Login: %+v", res)
		expecter.Close()
		return expecter, err
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

	return stdoutBuf.String(), stderrBuf.String(), err
}

func GetRunningVirtualMachineInstanceDomainXML(virtClient kubecli.KubevirtClient, vmi *v1.VirtualMachineInstance) (string, error) {
	vmiPod, err := getRunningPodByVirtualMachineInstance(vmi, NamespaceTestDefault)
	if err != nil {
		return "", err
	}

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

	stdout, stderr, err := ExecuteCommandOnPodV2(
		virtClient,
		vmiPod,
		vmiPod.Spec.Containers[containerIdx].Name,
		[]string{"virsh", "dumpxml", vmi.Namespace + "_" + vmi.Name},
	)
	if err != nil {
		return "", fmt.Errorf("could not dump libvirt domxml (remotely on pod): %v: %s", err, stderr)
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

func GetHighestCPUNumberAmongNodes(virtClient kubecli.KubevirtClient) int {
	var cpus int64

	nodes, err := virtClient.CoreV1().Nodes().List(metav1.ListOptions{})
	ExpectWithOffset(1, err).ToNot(HaveOccurred())

	for _, node := range nodes.Items {
		if v, ok := node.Status.Capacity[k8sv1.ResourceCPU]; ok && v.Value() > cpus {
			cpus = v.Value()
		}
	}

	return int(cpus)
}

func GetK8sCmdClient() string {
	// use oc if it exists, otherwise use kubectl
	if KubeVirtOcPath != "" {
		return "oc"
	}

	return "kubectl"
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
	case "gocli":
		cmdPath = KubeVirtGoCliPath
	}
	if cmdPath == "" {
		Skip(fmt.Sprintf("Skip test that requires %s binary", cmdName))
	}
}

func RunCommand(cmdName string, args ...string) (string, string, error) {
	return RunCommandWithNS(NamespaceTestDefault, cmdName, args...)
}

func RunCommandWithNS(namespace string, cmdName string, args ...string) (string, string, error) {
	return RunCommandWithNSAndInput(namespace, nil, cmdName, args...)
}

func RunCommandWithNSAndInput(namespace string, input io.Reader, cmdName string, args ...string) (string, string, error) {
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

	cmd.Stdin, cmd.Stdout, cmd.Stderr = input, &output, &stderr

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
	case "gocli":
		cmdPath = KubeVirtGoCliPath
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
		args = append([]string{"-n", namespace}, args...)
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

func GenerateVMJson(vm *v1.VirtualMachine, generateDirectory string) (string, error) {
	data, err := json.Marshal(vm)
	if err != nil {
		return "", fmt.Errorf("failed to generate json for vm %s", vm.Name)
	}

	jsonFile := filepath.Join(generateDirectory, fmt.Sprintf("%s.json", vm.Name))
	err = ioutil.WriteFile(jsonFile, data, 0644)
	if err != nil {
		return "", fmt.Errorf("failed to write json file %s", jsonFile)
	}
	return jsonFile, nil
}

func GenerateVMIJson(vmi *v1.VirtualMachineInstance, generateDirectory string) (string, error) {
	data, err := json.Marshal(vmi)
	if err != nil {
		return "", fmt.Errorf("failed to generate json for vmi %s", vmi.Name)
	}

	jsonFile := filepath.Join(generateDirectory, fmt.Sprintf("%s.json", vmi.Name))
	err = ioutil.WriteFile(jsonFile, data, 0644)
	if err != nil {
		return "", fmt.Errorf("failed to write json file %s", jsonFile)
	}
	return jsonFile, nil
}

func GenerateTemplateJson(template *vmsgen.Template, generateDirectory string) (string, error) {
	data, err := json.Marshal(template)
	if err != nil {
		return "", fmt.Errorf("failed to generate json for template %q: %v", template.Name, err)
	}

	jsonFile := filepath.Join(generateDirectory, template.Name+".json")
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

	fieldSelectorStr := "status.phase!=" + string(k8sv1.PodFailed) +
		",status.phase!=" + string(k8sv1.PodSucceeded)

	if vmi.Status.NodeName != "" {
		fieldSelectorStr = fieldSelectorStr +
			",spec.nodeName=" + vmi.Status.NodeName
	}

	fieldSelector := fields.ParseSelectorOrDie(fieldSelectorStr)
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
	image := fmt.Sprintf("%s/cdi-http-import-server:%s", KubeVirtUtilityRepoPrefix, KubeVirtUtilityVersionTag)
	resources := k8sv1.ResourceRequirements{}
	resources.Limits = make(k8sv1.ResourceList)
	resources.Limits[k8sv1.ResourceMemory] = resource.MustParse("256M")
	pod := &k8sv1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			GenerateName: ISCSITargetName,
			Labels: map[string]string{
				v1.AppLabel: ISCSITargetName,
			},
		},
		Spec: k8sv1.PodSpec{
			RestartPolicy: k8sv1.RestartPolicyNever,
			Containers: []k8sv1.Container{
				{
					Name:      ISCSITargetName,
					Image:     image,
					Resources: resources,
				},
			},
		},
	}
	if containerDiskName == ContainerDiskEmpty {
		asEmpty := []k8sv1.EnvVar{
			{
				Name:  "AS_EMPTY",
				Value: "true",
			},
		}
		pod.Spec.Containers[0].Env = asEmpty
	} else {
		imageEnv := []k8sv1.EnvVar{
			{
				Name:  "AS_ISCSI",
				Value: "true",
			},
			{
				Name:  "IMAGE_NAME",
				Value: fmt.Sprintf("%s", containerDiskName),
			},
		}
		pod.Spec.Containers[0].Env = imageEnv
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

func CreateISCSIPvAndPvc(name string, size string, iscsiTargetIP string, accessMode k8sv1.PersistentVolumeAccessMode, volumeMode k8sv1.PersistentVolumeMode) {
	virtCli, err := kubecli.GetKubevirtClient()
	PanicOnError(err)

	_, err = virtCli.CoreV1().PersistentVolumes().Create(NewISCSIPV(name, size, iscsiTargetIP, accessMode, volumeMode))
	if !errors.IsAlreadyExists(err) {
		PanicOnError(err)
	}

	_, err = virtCli.CoreV1().PersistentVolumeClaims(NamespaceTestDefault).Create(newISCSIPVC(name, size, accessMode, volumeMode))
	if !errors.IsAlreadyExists(err) {
		PanicOnError(err)
	}
}

func NewISCSIPV(name, size, iscsiTargetIP string, accessMode k8sv1.PersistentVolumeAccessMode, volumeMode k8sv1.PersistentVolumeMode) *k8sv1.PersistentVolume {
	quantity, err := resource.ParseQuantity(size)
	PanicOnError(err)

	storageClass := Config.StorageClassLocal

	return &k8sv1.PersistentVolume{
		ObjectMeta: metav1.ObjectMeta{
			Name: name,
		},
		Spec: k8sv1.PersistentVolumeSpec{
			AccessModes: []k8sv1.PersistentVolumeAccessMode{accessMode},
			Capacity: k8sv1.ResourceList{
				"storage": quantity,
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

func newISCSIPVC(name string, size string, accessMode k8sv1.PersistentVolumeAccessMode, volumeMode k8sv1.PersistentVolumeMode) *k8sv1.PersistentVolumeClaim {
	quantity, err := resource.ParseQuantity(size)
	PanicOnError(err)

	storageClass := Config.StorageClassLocal

	return &k8sv1.PersistentVolumeClaim{
		ObjectMeta: metav1.ObjectMeta{Name: name},
		Spec: k8sv1.PersistentVolumeClaimSpec{
			AccessModes: []k8sv1.PersistentVolumeAccessMode{accessMode},
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

func CreateNFSTargetPOD(os string) (nfsTargetIP string) {
	virtClient, err := kubecli.GetKubevirtClient()
	PanicOnError(err)
	image := fmt.Sprintf("%s/nfs-server:%s", KubeVirtRepoPrefix, KubeVirtVersionTag)
	resources := k8sv1.ResourceRequirements{}
	resources.Limits = make(k8sv1.ResourceList)
	resources.Limits[k8sv1.ResourceMemory] = resource.MustParse("2048M")
	hostPathType := k8sv1.HostPathDirectory
	pod := &k8sv1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name: NFSTargetName,
			Labels: map[string]string{
				v1.AppLabel: NFSTargetName,
			},
		},
		Spec: k8sv1.PodSpec{
			RestartPolicy: k8sv1.RestartPolicyNever,
			Volumes: []k8sv1.Volume{
				{
					Name: "nfsdata",
					VolumeSource: k8sv1.VolumeSource{
						HostPath: &k8sv1.HostPathVolumeSource{
							Path: HostPathBase + os,
							Type: &hostPathType,
						},
					},
				},
			},
			Containers: []k8sv1.Container{
				{
					Name:            NFSTargetName,
					Image:           image,
					ImagePullPolicy: k8sv1.PullAlways,
					Resources:       resources,
					SecurityContext: &k8sv1.SecurityContext{
						Privileged: NewBool(true),
					},
					VolumeMounts: []k8sv1.VolumeMount{
						{
							Name:      "nfsdata",
							MountPath: "/data/nfs",
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
		nfsTargetIP = pod.Status.PodIP
		return pod.Status.Phase
	}
	Eventually(getStatus, 120, 1).Should(Equal(k8sv1.PodRunning))
	return
}

func CreateNFSPvAndPvc(name string, size string, nfsTargetIP string, os string) {
	virtCli, err := kubecli.GetKubevirtClient()
	PanicOnError(err)

	_, err = virtCli.CoreV1().PersistentVolumes().Create(newNFSPV(name, size, nfsTargetIP, os))
	if !errors.IsAlreadyExists(err) {
		PanicOnError(err)
	}

	_, err = virtCli.CoreV1().PersistentVolumeClaims(NamespaceTestDefault).Create(newNFSPVC(name, size, os))
	if !errors.IsAlreadyExists(err) {
		PanicOnError(err)
	}
}

func newNFSPV(name string, size string, nfsTargetIP string, os string) *k8sv1.PersistentVolume {
	quantity := resource.MustParse(size)

	storageClass := Config.StorageClassLocal
	volumeMode := k8sv1.PersistentVolumeFilesystem

	return &k8sv1.PersistentVolume{
		ObjectMeta: metav1.ObjectMeta{
			Name: name,
			Labels: map[string]string{
				"kubevirt.io/test": os,
			},
		},
		Spec: k8sv1.PersistentVolumeSpec{
			AccessModes: []k8sv1.PersistentVolumeAccessMode{k8sv1.ReadWriteMany},
			Capacity: k8sv1.ResourceList{
				"storage": quantity,
			},
			StorageClassName: storageClass,
			VolumeMode:       &volumeMode,
			PersistentVolumeSource: k8sv1.PersistentVolumeSource{
				NFS: &k8sv1.NFSVolumeSource{
					Server: nfsTargetIP,
					Path:   "/",
				},
			},
		},
	}
}

func newNFSPVC(name string, size string, os string) *k8sv1.PersistentVolumeClaim {
	quantity, err := resource.ParseQuantity(size)
	PanicOnError(err)

	storageClass := Config.StorageClassLocal
	volumeMode := k8sv1.PersistentVolumeFilesystem

	return &k8sv1.PersistentVolumeClaim{
		ObjectMeta: metav1.ObjectMeta{
			Name: name,
		},
		Spec: k8sv1.PersistentVolumeClaimSpec{
			AccessModes: []k8sv1.PersistentVolumeAccessMode{k8sv1.ReadWriteMany},
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

	args := []string{fmt.Sprintf(`rm -rf %s`, diskPath)}
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
	check := []string{fmt.Sprintf(`set -x; x="$(head -n 1 < <(nc %s %s -i 3 -w 3))"; echo "$x" ; if [ "$x" = "Hello World!" ]; then echo "succeeded"; exit 0; else echo "failed"; exit 1; fi`, host, port)}
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
	check := []string{fmt.Sprintf(`set -x; trap "kill 0" EXIT; x="$(head -n 1 < <(echo | nc -up %d %s %s -i 3 -w 3 & nc -ul %d))"; echo "$x" ; if [ "$x" = "Hello UDP World!" ]; then echo "succeeded"; exit 0; else echo "failed"; exit 1; fi`,
		localPort, host, port, localPort)}
	job := RenderJob("netcat", []string{"/bin/bash", "-c"}, check)

	return job
}

// NewHelloWorldJobHttp gets an IP address and a port, which it uses to create a pod.
// This pod tries to contact the host on the provided port, over HTTP.
// On success - it expects to receive "Hello World!".
func NewHelloWorldJobHttp(host string, port string) *k8sv1.Pod {
	check := []string{fmt.Sprintf(`set -x; x="$(head -n 1 < <(curl %s:%s))"; echo "$x" ; if [ "$x" = "Hello World!" ]; then echo "succeeded"; exit 0; else echo "failed"; exit 1; fi`, FormatIPForURL(host), port)}
	job := RenderJob("curl", []string{"/bin/bash", "-c"}, check)

	return job
}

func GetNodeWithHugepages(virtClient kubecli.KubevirtClient, hugepages k8sv1.ResourceName) *k8sv1.Node {
	nodes, err := virtClient.CoreV1().Nodes().List(metav1.ListOptions{})
	ExpectWithOffset(1, err).ToNot(HaveOccurred())

	for _, node := range nodes.Items {
		if v, ok := node.Status.Capacity[hugepages]; ok && !v.IsZero() {
			return &node
		}
	}
	return nil
}

func GetAllSchedulableNodes(virtClient kubecli.KubevirtClient) *k8sv1.NodeList {
	nodes, err := virtClient.CoreV1().Nodes().List(metav1.ListOptions{LabelSelector: v1.NodeSchedulable + "=" + "true"})
	Expect(err).ToNot(HaveOccurred(), "Should list compute nodes")
	return nodes
}

// SkipIfVersionBelow will skip tests if it runs on an environment with k8s version below specified
func SkipIfVersionBelow(message string, expectedVersion string) {
	virtClient, err := kubecli.GetKubevirtClient()
	PanicOnError(err)

	curVersion, err := cluster.GetKubernetesVersion(virtClient)
	Expect(err).NotTo(HaveOccurred())

	if curVersion < expectedVersion {
		Skip(message)
	}
}

func SkipIfVersionAboveOrEqual(message string, expectedVersion string) {
	virtClient, err := kubecli.GetKubevirtClient()
	PanicOnError(err)

	curVersion, err := cluster.GetKubernetesVersion(virtClient)
	Expect(err).NotTo(HaveOccurred())

	if curVersion >= expectedVersion {
		Skip(message)
	}
}

func SkipIfOpenShift(message string) {
	if IsOpenShift() {
		Skip("Openshift detected: " + message)
	}
}

func SkipIfOpenShiftAndBelowOrEqualVersion(message string, version string) {
	virtClient, err := kubecli.GetKubevirtClient()
	PanicOnError(err)

	curVersion, err := cluster.GetKubernetesVersion(virtClient)
	Expect(err).NotTo(HaveOccurred())

	// version is above
	if curVersion > version {
		return
	}

	if IsOpenShift() {
		Skip(message)
	}
}

func SkipIfOpenShift4(message string) {
	virtClient, err := kubecli.GetKubevirtClient()
	PanicOnError(err)

	if t, err := cluster.IsOnOpenShift(virtClient); err != nil {
		PanicOnError(err)
	} else if t && cluster.GetOpenShiftMajorVersion(virtClient) == cluster.OpenShift4Major {
		Skip(message)
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
func GetNodeLibvirtCapabilities(vmi *v1.VirtualMachineInstance) string {
	return RunCommandOnVmiPod(vmi, []string{"virsh", "-r", "capabilities"})
}

// GetNodeCPUInfo returns output of lscpu on the pod that runs on the specified node
func GetNodeCPUInfo(vmi *v1.VirtualMachineInstance) string {
	return RunCommandOnVmiPod(vmi, []string{"lscpu"})
}

func NewRandomVirtualMachine(vmi *v1.VirtualMachineInstance, running bool) *v1.VirtualMachine {
	name := vmi.Name
	namespace := vmi.Namespace
	labels := map[string]string{"name": name}
	for k, v := range vmi.Labels {
		labels[k] = v
	}
	vm := &v1.VirtualMachine{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
		},
		Spec: v1.VirtualMachineSpec{
			Running: &running,
			Template: &v1.VirtualMachineInstanceTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels:    labels,
					Name:      name + "makeitinteresting", // this name should have no effect
					Namespace: namespace,
				},
				Spec: vmi.Spec,
			},
		},
	}
	vm.SetGroupVersionKind(schema.GroupVersionKind{Group: v1.GroupVersion.Group, Kind: "VirtualMachine", Version: v1.GroupVersion.Version})
	return vm
}

func StopVirtualMachine(vm *v1.VirtualMachine) *v1.VirtualMachine {
	By("Stopping the VirtualMachineInstance")
	virtClient, err := kubecli.GetKubevirtClient()
	PanicOnError(err)
	running := false
	Eventually(func() error {
		updatedVM, err := virtClient.VirtualMachine(vm.Namespace).Get(vm.Name, &metav1.GetOptions{})
		Expect(err).ToNot(HaveOccurred())
		updatedVM.Spec.Running = &running
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
	running := true
	Eventually(func() error {
		updatedVM, err := virtClient.VirtualMachine(vm.Namespace).Get(vm.Name, &metav1.GetOptions{})
		Expect(err).ToNot(HaveOccurred())
		updatedVM.Spec.Running = &running
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

func HasFeature(feature string) bool {
	hasFeature := false
	virtClient, err := kubecli.GetKubevirtClient()
	PanicOnError(err)
	options := metav1.GetOptions{}
	cfgMap, err := virtClient.CoreV1().ConfigMaps(KubeVirtInstallNamespace).Get(virtconfig.ConfigMapName, options)
	if err == nil {
		val, ok := cfgMap.Data[virtconfig.FeatureGatesKey]
		if !ok {
			return hasFeature
		}
		hasFeature = strings.Contains(val, feature)
	} else {
		if !errors.IsNotFound(err) {
			PanicOnError(err)
		}
	}
	return hasFeature
}

func DisableFeatureGate(feature string) {
	if !HasFeature(feature) {
		return
	}
	virtClient, err := kubecli.GetKubevirtClient()
	Expect(err).ToNot(HaveOccurred())
	cfg, err := virtClient.CoreV1().ConfigMaps(KubeVirtInstallNamespace).Get(virtconfig.ConfigMapName, metav1.GetOptions{})
	Expect(err).ToNot(HaveOccurred())

	val, _ := cfg.Data["feature-gates"]

	newVal := strings.Replace(val, feature+",", "", 1)
	newVal = strings.Replace(newVal, feature, "", 1)

	UpdateClusterConfigValueAndWait("feature-gates", newVal)
}

func EnableFeatureGate(feature string) {
	if HasFeature(feature) {
		return
	}
	virtClient, err := kubecli.GetKubevirtClient()
	Expect(err).ToNot(HaveOccurred())
	cfg, err := virtClient.CoreV1().ConfigMaps(KubeVirtInstallNamespace).Get(virtconfig.ConfigMapName, metav1.GetOptions{})
	Expect(err).ToNot(HaveOccurred())

	val, _ := cfg.Data["feature-gates"]
	newVal := fmt.Sprintf("%s,%s", val, feature)

	UpdateClusterConfigValueAndWait("feature-gates", newVal)
}

func HasDataVolumeCRD() bool {
	virtClient, err := kubecli.GetKubevirtClient()
	PanicOnError(err)

	ext, err := extclient.NewForConfig(virtClient.Config())
	PanicOnError(err)

	_, err = ext.ApiextensionsV1beta1().CustomResourceDefinitions().Get("datavolumes.cdi.kubevirt.io", metav1.GetOptions{})

	if err != nil {
		return false
	}
	return true
}

func HasCDI() bool {
	return HasFeature("DataVolumes")
}

func GetCephStorageClass() (string, bool) {
	virtClient, err := kubecli.GetKubevirtClient()
	Expect(err).ToNot(HaveOccurred())
	storageClassList, err := virtClient.StorageV1().StorageClasses().List(metav1.ListOptions{})
	Expect(err).ToNot(HaveOccurred())
	for _, storageClass := range storageClassList.Items {
		if storageClass.Provisioner == "csi-rbdplugin" {
			return storageClass.Name, true
		}
	}
	return "", false
}

func HasExperimentalIgnitionSupport() bool {
	return HasFeature("ExperimentalIgnitionSupport")
}

func HasLiveMigration() bool {
	return HasFeature("LiveMigration")
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

func StartHTTPServer(vmi *v1.VirtualMachineInstance, port int, isFedoraVM bool) {
	var err error
	var expecter expect.Expecter
	var httpServerMaker string
	var prompt string

	if isFedoraVM {
		expecter, err = LoggedInFedoraExpecter(vmi)
		Expect(err).NotTo(HaveOccurred())
		httpServerMaker = fmt.Sprintf("python3 -m http.server %d --bind ::0 &\n", port)
		prompt = "\\#"
	} else {
		expecter, err = LoggedInCirrosExpecter(vmi)
		Expect(err).NotTo(HaveOccurred())
		httpServerMaker = fmt.Sprintf("screen -d -m nc -klp %d -e echo -e \"HTTP/1.1 200 OK\\nContent-Length: 11\\n\\nHello World!\"\n", port)
		prompt = "\\$"
	}
	defer expecter.Close()

	resp, err := expecter.ExpectBatch([]expect.Batcher{
		&expect.BSnd{S: "\n"},
		&expect.BExp{R: prompt},
		&expect.BSnd{S: httpServerMaker},
		&expect.BExp{R: prompt},
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

func AppendEmptyDisk(vmi *v1.VirtualMachineInstance, diskName, busName, diskSize string) {
	vmi.Spec.Domain.Devices.Disks = append(vmi.Spec.Domain.Devices.Disks, v1.Disk{
		Name: diskName,
		DiskDevice: v1.DiskDevice{
			Disk: &v1.DiskTarget{
				Bus: busName,
			},
		},
	})
	vmi.Spec.Volumes = append(vmi.Spec.Volumes, v1.Volume{
		Name: diskName,
		VolumeSource: v1.VolumeSource{
			EmptyDisk: &v1.EmptyDiskSource{
				Capacity: resource.MustParse(diskSize),
			},
		},
	})
}

func GetRunningVMIDomainSpec(vmi *v1.VirtualMachineInstance) (*launcherApi.DomainSpec, error) {
	runningVMISpec := launcherApi.DomainSpec{}
	cli, err := kubecli.GetKubevirtClient()
	if err != nil {
		return nil, err
	}

	domXML, err := GetRunningVirtualMachineInstanceDomainXML(cli, vmi)
	if err != nil {
		return nil, err
	}

	err = xml.Unmarshal([]byte(domXML), &runningVMISpec)
	return &runningVMISpec, err
}

func ForwardPorts(pod *k8sv1.Pod, ports []string, stop chan struct{}, readyTimeout time.Duration) error {
	errChan := make(chan error, 1)
	readyChan := make(chan struct{})
	go func() {
		cli, err := kubecli.GetKubevirtClient()
		if err != nil {
			errChan <- err
			return
		}

		req := cli.CoreV1().RESTClient().Post().
			Resource("pods").
			Namespace(pod.Namespace).
			Name(pod.Name).
			SubResource("portforward")

		config, err := kubecli.GetKubevirtClientConfig()
		if err != nil {
			errChan <- err
			return
		}
		transport, upgrader, err := spdy.RoundTripperFor(config)
		if err != nil {
			errChan <- err
			return
		}
		dialer := spdy.NewDialer(upgrader, &http.Client{Transport: transport}, "POST", req.URL())
		forwarder, err := portforward.New(dialer, ports, stop, readyChan, GinkgoWriter, GinkgoWriter)
		if err != nil {
			errChan <- err
			return
		}
		err = forwarder.ForwardPorts()
		if err != nil {
			errChan <- err
		}
	}()

	select {
	case err := <-errChan:
		return err
	case <-readyChan:
		return nil
	case <-time.After(readyTimeout):
		return fmt.Errorf("failed to forward ports, timed out")
	}
}

func GenerateHelloWorldServer(vmi *v1.VirtualMachineInstance, testPort int, protocol string) {
	expecter, err := LoggedInCirrosExpecter(vmi)
	Expect(err).ToNot(HaveOccurred())
	defer expecter.Close()

	serverCommand := fmt.Sprintf("screen -d -m sudo nc -klp %d -e echo -e 'Hello World!'\n", testPort)
	if protocol == "udp" {
		// nc has to be in a while loop in case of UDP, since it exists after one message
		serverCommand = fmt.Sprintf("screen -d -m sh -c \"while true\n do nc -uklp %d -e echo -e 'Hello UDP World!'\ndone\n\"\n", testPort)
	}
	_, err = expecter.ExpectBatch([]expect.Batcher{
		&expect.BSnd{S: serverCommand},
		&expect.BExp{R: "\\$ "},
		&expect.BSnd{S: "echo $?\n"},
		&expect.BExp{R: "0"},
	}, 60*time.Second)
	Expect(err).ToNot(HaveOccurred())
}

// UpdateClusterConfigValueAndWait updates the given configuration in the kubevirt config map and then waits
// to allow the configuration events to be propagated to the consumers.
func UpdateClusterConfigValueAndWait(key string, value string) string {
	virtClient, err := kubecli.GetKubevirtClient()
	PanicOnError(err)
	cfgMap, err := virtClient.CoreV1().ConfigMaps(KubeVirtInstallNamespace).Get(virtconfig.ConfigMapName, metav1.GetOptions{})
	ExpectWithOffset(1, err).NotTo(HaveOccurred())
	oldValue := cfgMap.Data[key]
	cfgMap.Data[key] = value
	_, err = virtClient.CoreV1().ConfigMaps(KubeVirtInstallNamespace).Update(cfgMap)
	ExpectWithOffset(1, err).ToNot(HaveOccurred())

	time.Sleep(2 * time.Second)
	return oldValue
}

func WaitAgentConnected(virtClient kubecli.KubevirtClient, vmi *v1.VirtualMachineInstance) {
	By("Waiting for guest agent connection")
	WaitForVMICondition(virtClient, vmi, v1.VirtualMachineInstanceAgentConnected, 12*60)
}

func WaitForVMICondition(virtClient kubecli.KubevirtClient, vmi *v1.VirtualMachineInstance, conditionType v1.VirtualMachineInstanceConditionType, timeoutSec int) {
	By(fmt.Sprintf("Waiting for %s condition", conditionType))
	EventuallyWithOffset(1, func() bool {
		updatedVmi, err := virtClient.VirtualMachineInstance(NamespaceTestDefault).Get(vmi.Name, &metav1.GetOptions{})
		Expect(err).ToNot(HaveOccurred())
		for _, condition := range updatedVmi.Status.Conditions {
			if condition.Type == conditionType && condition.Status == k8sv1.ConditionTrue {
				return true
			}
		}
		return false
	}, time.Duration(timeoutSec)*time.Second, 2).Should(BeTrue(), fmt.Sprintf("Should have %s condition", conditionType))
}

func WaitForVMIConditionRemovedOrFalse(virtClient kubecli.KubevirtClient, vmi *v1.VirtualMachineInstance, conditionType v1.VirtualMachineInstanceConditionType, timeoutSec int) {
	By(fmt.Sprintf("Waiting for %s condition removed or false", conditionType))
	EventuallyWithOffset(1, func() bool {
		updatedVmi, err := virtClient.VirtualMachineInstance(NamespaceTestDefault).Get(vmi.Name, &metav1.GetOptions{})
		Expect(err).ToNot(HaveOccurred())
		for _, condition := range updatedVmi.Status.Conditions {
			if condition.Type == conditionType && condition.Status == k8sv1.ConditionTrue {
				return true
			}
		}
		return false
	}, time.Duration(timeoutSec)*time.Second, 2).Should(BeFalse(), fmt.Sprintf("Should have no or false %s condition", conditionType))
}

func WaitForVMCondition(virtClient kubecli.KubevirtClient, vm *v1.VirtualMachine, conditionType v1.VirtualMachineConditionType, timeoutSec int) {
	By(fmt.Sprintf("Waiting for %s condition", conditionType))
	EventuallyWithOffset(1, func() bool {
		updatedVm, err := virtClient.VirtualMachine(NamespaceTestDefault).Get(vm.Name, &metav1.GetOptions{})
		Expect(err).ToNot(HaveOccurred())
		for _, condition := range updatedVm.Status.Conditions {
			if condition.Type == conditionType && condition.Status == k8sv1.ConditionTrue {
				return true
			}
		}
		return false
	}, time.Duration(timeoutSec)*time.Second, 2).Should(BeTrue(), fmt.Sprintf("Should have %s condition", conditionType))
}

func WaitForVMConditionRemovedOrFalse(virtClient kubecli.KubevirtClient, vm *v1.VirtualMachine, conditionType v1.VirtualMachineConditionType, timeoutSec int) {
	By(fmt.Sprintf("Waiting for %s condition removed or false", conditionType))
	EventuallyWithOffset(1, func() bool {
		updatedVm, err := virtClient.VirtualMachine(NamespaceTestDefault).Get(vm.Name, &metav1.GetOptions{})
		Expect(err).ToNot(HaveOccurred())
		for _, condition := range updatedVm.Status.Conditions {
			if condition.Type == conditionType && condition.Status == k8sv1.ConditionTrue {
				return true
			}
		}
		return false
	}, time.Duration(timeoutSec)*time.Second, 2).Should(BeFalse(), fmt.Sprintf("Should have no or false %s condition", conditionType))
}

// GeneratePrivateKey creates a RSA Private Key of specified byte size
func GeneratePrivateKey(bitSize int) (*rsa.PrivateKey, error) {
	privateKey, err := rsa.GenerateKey(cryptorand.Reader, bitSize)
	if err != nil {
		return nil, err
	}

	err = privateKey.Validate()
	if err != nil {
		return nil, err
	}

	return privateKey, nil
}

// GeneratePublicKey will return in the format "ssh-rsa ..."
func GeneratePublicKey(privatekey *rsa.PublicKey) ([]byte, error) {
	publicRsaKey, err := ssh.NewPublicKey(privatekey)
	if err != nil {
		return nil, err
	}

	publicKeyBytes := ssh.MarshalAuthorizedKey(publicRsaKey)

	return publicKeyBytes, nil
}

// EncodePrivateKeyToPEM encodes Private Key from RSA to PEM format
func EncodePrivateKeyToPEM(privateKey *rsa.PrivateKey) []byte {
	// Get ASN.1 DER format
	privDER := x509.MarshalPKCS1PrivateKey(privateKey)

	privateBlock := pem.Block{
		Type:    "RSA PRIVATE KEY",
		Headers: nil,
		Bytes:   privDER,
	}

	// Private key in PEM format
	privatePEM := pem.EncodeToMemory(&privateBlock)

	return privatePEM
}

func PodReady(pod *k8sv1.Pod) k8sv1.ConditionStatus {
	for _, cond := range pod.Status.Conditions {
		if cond.Type == k8sv1.PodReady {
			return cond.Status
		}
	}
	return k8sv1.ConditionFalse
}

func RetryWithMetadataIfModified(objectMeta metav1.ObjectMeta, do func(objectMeta metav1.ObjectMeta) error) (err error) {
	return RetryIfModified(func() error {
		return do(objectMeta)
	})
}

func RetryIfModified(do func() error) (err error) {
	retries := 0
	for err = do(); errors.IsConflict(err); err = do() {
		if retries >= 10 {
			return fmt.Errorf("object seems to be permanently modified, failing after 10 retries: %v", err)
		}
		retries++
		log.DefaultLogger().Reason(err).Infof("Object got modified, will retry.")
	}
	return err
}

func GenerateRandomMac() (net.HardwareAddr, error) {
	prefix := []byte{0x02, 0x00, 0x00} // local unicast prefix
	suffix := make([]byte, 3)
	_, err := cryptorand.Read(suffix)
	if err != nil {
		return nil, err
	}
	return net.HardwareAddr(append(prefix, suffix...)), nil
}

func GetUrl(urlIndex int) string {
	var str string

	switch urlIndex {
	case AlpineHttpUrl:
		str = fmt.Sprintf("http://cdi-http-import-server.%s/images/alpine.iso", KubeVirtInstallNamespace)
	case GuestAgentHttpUrl:
		str = fmt.Sprintf("http://cdi-http-import-server.%s/qemu-ga", KubeVirtInstallNamespace)
	case StressHttpUrl:
		str = fmt.Sprintf("http://cdi-http-import-server.%s/stress", KubeVirtInstallNamespace)
	case DmidecodeHttpUrl:
		str = fmt.Sprintf("http://cdi-http-import-server.%s/dmidecode", KubeVirtInstallNamespace)
	case DummyFileHttpUrl:
		str = fmt.Sprintf("http://cdi-http-import-server.%s/dummy.file", KubeVirtInstallNamespace)
	default:
		str = ""
	}

	return str
}

func GetCurrentKv(virtClient kubecli.KubevirtClient) *v1.KubeVirt {
	kvs := GetKvList(virtClient)
	Expect(len(kvs)).To(Equal(1))
	return &kvs[0]
}

func GetKvList(virtClient kubecli.KubevirtClient) []v1.KubeVirt {
	var kvListInstallNS *v1.KubeVirtList
	var kvListDefaultNS *v1.KubeVirtList
	var items []v1.KubeVirt

	var err error

	Eventually(func() error {

		kvListInstallNS, err = virtClient.KubeVirt(KubeVirtInstallNamespace).List(&metav1.ListOptions{})

		return err
	}, 10*time.Second, 1*time.Second).ShouldNot(HaveOccurred())

	Eventually(func() error {

		kvListDefaultNS, err = virtClient.KubeVirt(NamespaceTestDefault).List(&metav1.ListOptions{})

		return err
	}, 10*time.Second, 1*time.Second).ShouldNot(HaveOccurred())

	items = append(items, kvListInstallNS.Items...)
	items = append(items, kvListDefaultNS.Items...)

	return items
}

func getCert(pod *k8sv1.Pod, port string) []byte {
	randPort := strconv.Itoa(int(4321 + rand.Intn(6000)))
	var rawCert []byte
	mutex := &sync.Mutex{}
	conf := &tls.Config{
		InsecureSkipVerify: true,
		VerifyPeerCertificate: func(rawCerts [][]byte, verifiedChains [][]*x509.Certificate) error {
			mutex.Lock()
			defer mutex.Unlock()
			rawCert = rawCerts[0]
			return nil
		},
	}

	var cert []byte
	EventuallyWithOffset(2, func() []byte {
		stopChan := make(chan struct{})
		defer close(stopChan)
		err := ForwardPorts(pod, []string{fmt.Sprintf("%s:%s", randPort, port)}, stopChan, 10*time.Second)
		ExpectWithOffset(2, err).ToNot(HaveOccurred())

		conn, err := tls.Dial("tcp4", fmt.Sprintf("localhost:%s", randPort), conf)
		if err == nil {
			defer conn.Close()
		}
		mutex.Lock()
		defer mutex.Unlock()
		cert = make([]byte, len(rawCert))
		copy(cert, rawCert)
		return cert
	}, 40*time.Second, 1*time.Second).Should(Not(BeEmpty()))

	return cert
}

// GetCertsForPods returns the used certificates for all pods matching  the label selector
func GetCertsForPods(labelSelector string, namespace string, port string) ([][]byte, error) {
	cli, err := kubecli.GetKubevirtClient()
	Expect(err).ToNot(HaveOccurred())
	pods, err := cli.CoreV1().Pods(namespace).List(metav1.ListOptions{LabelSelector: labelSelector})
	Expect(err).ToNot(HaveOccurred())
	Expect(pods.Items).ToNot(BeEmpty())

	var certs [][]byte

	for _, pod := range pods.Items {
		err := func() error {
			certs = append(certs, getCert(&pod, port))
			return nil
		}()
		if err != nil {
			return nil, err
		}
	}
	return certs, nil
}

// EnsurePodsCertIsSynced waits until new certificates are rolled out  to all pods which are matching the specified labelselector.
// Once all certificates are in sync, the final secret is returned
func EnsurePodsCertIsSynced(labelSelector string, namespace string, port string) []byte {
	var certs [][]byte
	EventuallyWithOffset(1, func() bool {
		var err error
		certs, err = GetCertsForPods(labelSelector, namespace, port)
		Expect(err).ToNot(HaveOccurred())
		if len(certs) == 0 {
			return true
		}
		for _, crt := range certs {
			if !reflect.DeepEqual(certs[0], crt) {
				return false
			}
		}
		return true
	}, 90*time.Second, 1*time.Second).Should(BeTrue(), "certificates across '%s' pods are not in sync", labelSelector)
	if len(certs) > 0 {
		return certs[0]
	}
	return nil
}

// GetPodsCertIfSynced returns the certificate for all matching pods once all of them use the same certificate
func GetPodsCertIfSynced(labelSelector string, namespace string, port string) (cert []byte, synced bool, err error) {
	certs, err := GetCertsForPods(labelSelector, namespace, port)
	if err != nil {
		return nil, false, err
	}
	if len(certs) == 0 {
		return nil, true, nil
	}
	for _, crt := range certs {
		if !reflect.DeepEqual(certs[0], crt) {
			return nil, false, nil
		}
	}
	return certs[0], true, nil
}

func GetCertFromSecret(secretName string) []byte {
	virtClient, err := kubecli.GetKubevirtClient()
	Expect(err).ToNot(HaveOccurred())
	secret, err := virtClient.CoreV1().Secrets(KubeVirtInstallNamespace).Get(secretName, metav1.GetOptions{})
	Expect(err).ToNot(HaveOccurred())
	if rawBundle, ok := secret.Data[bootstrap.CertBytesValue]; ok {
		return rawBundle
	}
	return nil
}

func GetBundleFromConfigMap(configMapName string) ([]byte, []*x509.Certificate) {
	virtClient, err := kubecli.GetKubevirtClient()
	Expect(err).ToNot(HaveOccurred())
	configMap, err := virtClient.CoreV1().ConfigMaps(KubeVirtInstallNamespace).Get(components.KubeVirtCASecretName, metav1.GetOptions{})
	Expect(err).ToNot(HaveOccurred())
	if rawBundle, ok := configMap.Data[components.CABundleKey]; ok {
		crts, err := cert.ParseCertsPEM([]byte(rawBundle))
		Expect(err).ToNot(HaveOccurred())
		return []byte(rawBundle), crts
	}
	return nil, nil
}

func ContainsCrt(bundle []byte, containedCrt []byte) bool {
	crts, err := cert.ParseCertsPEM(bundle)
	Expect(err).ToNot(HaveOccurred())
	attached := false
	for _, crt := range crts {
		crtBytes := cert.EncodeCertPEM(crt)
		if reflect.DeepEqual(crtBytes, containedCrt) {
			attached = true
			break
		}
	}
	return attached
}

func FormatIPForURL(ip string) string {
	if netutils.IsIPv6String(ip) {
		return "[" + ip + "]"
	}
	return ip
}

func getClusterDnsServiceIP(virtClient kubecli.KubevirtClient) (string, error) {
	dnsServiceName := "kube-dns"
	dnsNamespace := "kube-system"
	if IsOpenShift() {
		dnsServiceName = "dns-default"
		dnsNamespace = "openshift-dns"
	}
	kubeDNSService, err := virtClient.CoreV1().Services(dnsNamespace).Get(dnsServiceName, metav1.GetOptions{})
	if err != nil {
		return "", err
	}
	return kubeDNSService.Spec.ClusterIP, nil
}

func IsRunningOnKindInfra() bool {
	provider := os.Getenv("KUBEVIRT_PROVIDER")
	return strings.HasPrefix(provider, "kind")
}

func IsRunningOnKindInfraIPv6() bool {
	provider := os.Getenv("KUBEVIRT_PROVIDER")
	return strings.HasPrefix(provider, "kind-k8s-1.17.0-ipv6")
}

func SkipPVCTestIfRunnigOnKindInfra() {
	if IsRunningOnKindInfra() {
		Skip("Skip PVC tests till PR https://github.com/kubevirt/kubevirt/pull/3171 is merged")
	}
}

func SkipNFSTestIfRunnigOnKindInfra() {
	if IsRunningOnKindInfra() {
		Skip("Skip NFS tests till issue https://github.com/kubevirt/kubevirt/issues/3322 is fixed")
	}
}

func IsUsingBuiltinNodeDrainKey() bool {
	return GetNodeDrainKey() == "node.kubernetes.io/unschedulable"
}

func GetNodeDrainKey() string {
	var data map[string]string

	key := virtconfig.NodeDrainTaintDefaultKey
	cfgMap, err := GetKubeVirtConfigMap()
	if err == nil {
		if val, ok := cfgMap.Data[virtconfig.MigrationsConfigKey]; ok {
			json.Unmarshal([]byte(val), &data)
			if val, ok = data["nodeDrainTaintKey"]; ok {
				key = val
			}
		}
	}

	return key
}

func GetKubeVirtConfigMap() (*k8sv1.ConfigMap, error) {
	virtClient, err := kubecli.GetKubevirtClient()
	PanicOnError(err)

	options := metav1.GetOptions{}
	cfgMap, err := virtClient.CoreV1().ConfigMaps(KubeVirtInstallNamespace).Get(virtconfig.ConfigMapName, options)
	if err != nil && !errors.IsNotFound(err) {
		PanicOnError(err)
	}

	return cfgMap, err
}

func ClearKubeVirtConfigMap(key string) error {
	virtClient, err := kubecli.GetKubevirtClient()
	PanicOnError(err)

	cfgMap, err := GetKubeVirtConfigMap()
	if err == nil {
		if _, ok := cfgMap.Data[key]; ok {
			data := fmt.Sprintf(`[{ "op": "remove", "path": "/data/%s"}]`, key)
			_, err = virtClient.CoreV1().ConfigMaps(KubeVirtInstallNamespace).Patch(virtconfig.ConfigMapName, types.JSONPatchType, []byte(data))
		}
	}

	return err
}

func RandTmpDir() string {
	return tmpPath + "/" + rand.String(10)
}

func IsIPv6Cluster(virtClient kubecli.KubevirtClient) bool {
	clusterDnsIP, _ := getClusterDnsServiceIP(virtClient)
	return netutils.IsIPv6String(clusterDnsIP)
}

func retcode(retcode string) string {
	return "\n" + retcode
}

func IsLauncherCapabilityValid(capability k8sv1.Capability) bool {
	switch capability {
	case
		capNetAdmin,
		capNetRaw,
		capSysNice:
		return true
	}
	return false
}
