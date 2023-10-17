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

package operator

import (
	"bytes"
	"compress/gzip"
	"context"
	"crypto/x509"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"

	"kubevirt.io/kubevirt/tests/libnode"

	"kubevirt.io/kubevirt/tests/decorators"

	"github.com/Masterminds/semver"
	jsonpatch "github.com/evanphx/json-patch"
	"github.com/google/go-github/v32/github"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	v12 "k8s.io/api/apps/v1"
	k8sv1 "k8s.io/api/core/v1"
	extv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	extclient "k8s.io/apiextensions-apiserver/pkg/client/clientset/clientset"
	"k8s.io/apimachinery/pkg/api/equality"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/fields"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/selection"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/rand"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/util/retry"
	aggregatorclient "k8s.io/kube-aggregator/pkg/client/clientset_generated/clientset"
	"k8s.io/utils/pointer"

	v1 "kubevirt.io/api/core/v1"
	"kubevirt.io/api/instancetype/v1beta1"
	snapshotv1 "kubevirt.io/api/snapshot/v1alpha1"
	"kubevirt.io/client-go/kubecli"
	"kubevirt.io/client-go/log"
	cdiv1 "kubevirt.io/containerized-data-importer-api/pkg/apis/core/v1beta1"
	sdkapi "kubevirt.io/controller-lifecycle-operator-sdk/api"

	"kubevirt.io/kubevirt/pkg/controller"
	virtconfig "kubevirt.io/kubevirt/pkg/virt-config"
	"kubevirt.io/kubevirt/pkg/virt-operator/resource/apply"
	"kubevirt.io/kubevirt/pkg/virt-operator/resource/generate/components"
	"kubevirt.io/kubevirt/pkg/virt-operator/util"
	"kubevirt.io/kubevirt/tests"
	"kubevirt.io/kubevirt/tests/clientcmd"
	"kubevirt.io/kubevirt/tests/console"
	cd "kubevirt.io/kubevirt/tests/containerdisk"
	"kubevirt.io/kubevirt/tests/flags"
	"kubevirt.io/kubevirt/tests/framework/checks"
	"kubevirt.io/kubevirt/tests/framework/kubevirt"
	"kubevirt.io/kubevirt/tests/framework/matcher"
	. "kubevirt.io/kubevirt/tests/framework/matcher"
	"kubevirt.io/kubevirt/tests/libnet"
	"kubevirt.io/kubevirt/tests/libstorage"
	"kubevirt.io/kubevirt/tests/libvmi"
	"kubevirt.io/kubevirt/tests/libwait"
	"kubevirt.io/kubevirt/tests/testsuite"
	util2 "kubevirt.io/kubevirt/tests/util"
)

type vmSnapshotDef struct {
	vmSnapshotName  string
	yamlFile        string
	restoreName     string
	restoreYamlFile string
}

type vmYamlDefinition struct {
	apiVersion    string
	vmName        string
	generatedYaml string
	yamlFile      string
	vmSnapshots   []vmSnapshotDef
}

var _ = Describe("[Serial][sig-operator]Operator", Serial, decorators.SigOperator, func() {
	var originalKv *v1.KubeVirt
	var originalCDI *cdiv1.CDI
	var originalOperatorVersion string
	var err error
	var workDir string

	var virtClient kubecli.KubevirtClient
	var aggregatorClient *aggregatorclient.Clientset
	var k8sClient string
	var vmYamls map[string]*vmYamlDefinition

	var (
		copyOriginalCDI                        func() *cdiv1.CDI
		copyOriginalKv                         func() *v1.KubeVirt
		createKv                               func(*v1.KubeVirt)
		createCdi                              func()
		sanityCheckDeploymentsExistWithNS      func(string)
		sanityCheckDeploymentsExist            func()
		sanityCheckDeploymentsDeleted          func()
		allPodsAreReady                        func(*v1.KubeVirt)
		allPodsAreTerminated                   func(*v1.KubeVirt)
		waitForUpdateCondition                 func(*v1.KubeVirt)
		waitForKvWithTimeout                   func(*v1.KubeVirt, int)
		waitForKv                              func(*v1.KubeVirt)
		patchKvProductNameVersionAndComponent  func(string, string, string, string)
		patchKvVersionAndRegistry              func(string, string, string)
		patchKvVersion                         func(string, string)
		patchKvNodePlacement                   func(string, string, string, *v1.ComponentConfig)
		patchKvNodePlacementExpectError        func(string, string, string, *v1.ComponentConfig, string)
		patchKvInfra                           func(*v1.ComponentConfig, bool, string)
		patchKvWorkloads                       func(*v1.ComponentConfig, bool, string)
		patchKvCertConfig                      func(name string, certConfig *v1.KubeVirtSelfSignConfiguration)
		patchKvCertConfigExpectError           func(name string, certConfig *v1.KubeVirtSelfSignConfiguration)
		parseDaemonset                         func(string) string
		parseImage                             func(string) (string, string, string)
		parseDeployment                        func(string) (*v12.Deployment, string, string, string, string)
		parseOperatorImage                     func() (*v12.Deployment, string, string, string, string)
		patchOperator                          func(*string, *string) bool
		installOperator                        func(string)
		installTestingManifests                func(string)
		deleteOperator                         func(string)
		deleteTestingManifests                 func(string)
		deleteAllKvAndWait                     func(bool)
		usesSha                                func(string) bool
		ensureShasums                          func()
		getVirtLauncherSha                     func() string
		generatePreviousVersionVmYamls         func(string, string)
		generatePreviousVersionVmsnapshotYamls func()
		generateMigratableVMIs                 func(int) []*v1.VirtualMachineInstance
		generateNonMigratableVMIs              func(int) []*v1.VirtualMachineInstance
		startAllVMIs                           func([]*v1.VirtualMachineInstance)
		deleteAllVMIs                          func([]*v1.VirtualMachineInstance)
		verifyVMIsUpdated                      func([]*v1.VirtualMachineInstance, string)
		verifyVMIsEvicted                      func([]*v1.VirtualMachineInstance)
		fetchVirtHandlerCommand                func() string
	)

	tests.DeprecatedBeforeAll(func() {
		virtClient = kubevirt.Client()
		config, err := kubecli.GetKubevirtClientConfig()
		util2.PanicOnError(err)
		aggregatorClient = aggregatorclient.NewForConfigOrDie(config)

		k8sClient = clientcmd.GetK8sCmdClient()

		copyOriginalCDI = func() *cdiv1.CDI {
			newCDI := &cdiv1.CDI{
				Spec: *originalCDI.Spec.DeepCopy(),
			}
			newCDI.Name = originalCDI.Name
			newCDI.Namespace = originalCDI.Namespace
			newCDI.ObjectMeta.Labels = originalCDI.ObjectMeta.Labels
			newCDI.ObjectMeta.Annotations = originalCDI.ObjectMeta.Annotations

			return newCDI

		}

		copyOriginalKv = func() *v1.KubeVirt {
			newKv := &v1.KubeVirt{
				Spec: *originalKv.Spec.DeepCopy(),
			}
			newKv.Name = originalKv.Name
			newKv.Namespace = originalKv.Namespace
			newKv.ObjectMeta.Labels = originalKv.ObjectMeta.Labels
			newKv.ObjectMeta.Annotations = originalKv.ObjectMeta.Annotations

			return newKv

		}

		createKv = func(newKv *v1.KubeVirt) {
			Eventually(func() error {
				_, err = virtClient.KubeVirt(newKv.Namespace).Create(newKv)
				return err
			}, 10*time.Second, 1*time.Second).ShouldNot(HaveOccurred())
		}

		createCdi = func() {
			_, err = virtClient.CdiClient().CdiV1beta1().CDIs().Create(context.Background(), copyOriginalCDI(), metav1.CreateOptions{})
			Expect(err).ToNot(HaveOccurred())

			Eventually(func() bool {
				cdi, err := virtClient.CdiClient().CdiV1beta1().CDIs().Get(context.Background(), originalCDI.Name, metav1.GetOptions{})
				if err != nil {
					return false
				} else if cdi.Status.Phase != sdkapi.PhaseDeployed {
					return false
				}
				return true
			}, 240*time.Second, 1*time.Second).Should(BeTrue())
		}

		sanityCheckDeploymentsExistWithNS = func(namespace string) {
			Eventually(func() error {
				for _, deployment := range []string{"virt-api", "virt-controller"} {
					_, err := virtClient.AppsV1().Deployments(namespace).Get(context.Background(), deployment, metav1.GetOptions{})
					if err != nil {
						return err
					}
				}
				return nil
			}, 15*time.Second, 1*time.Second).ShouldNot(HaveOccurred())
		}

		sanityCheckDeploymentsExist = func() {
			sanityCheckDeploymentsExistWithNS(flags.KubeVirtInstallNamespace)
		}

		sanityCheckDeploymentsDeleted = func() {
			Eventually(func() int {
				deploymentCount := 2
				for _, deployment := range []string{"virt-api", "virt-controller"} {
					_, err := virtClient.AppsV1().Deployments(flags.KubeVirtInstallNamespace).Get(context.Background(), deployment, metav1.GetOptions{})
					if err != nil && errors.IsNotFound(err) {
						deploymentCount--
					}
				}
				return deploymentCount
			}, 15*time.Second, 1*time.Second).Should(Equal(0))
		}

		allPodsAreTerminated = func(kv *v1.KubeVirt) {
			Eventually(func() error {
				pods, err := virtClient.CoreV1().Pods(kv.Namespace).List(context.Background(), metav1.ListOptions{LabelSelector: "kubevirt.io"})
				if err != nil {
					return err
				}

				for _, pod := range pods.Items {
					manager, managed := pod.Labels[v1.ManagedByLabel]
					if !managed || manager != v1.ManagedByLabelOperatorValue {
						continue
					}

					if pod.Status.Phase != k8sv1.PodFailed && pod.Status.Phase != k8sv1.PodSucceeded {
						return fmt.Errorf("Waiting for pod %s with phase %s to reach final phase", pod.Name, pod.Status.Phase)
					}
				}
				return nil
			}, 120*time.Second, 1*time.Second).ShouldNot(HaveOccurred())
		}

		allPodsAreReady = func(kv *v1.KubeVirt) {
			Eventually(func() error {

				curKv, err := virtClient.KubeVirt(kv.Namespace).Get(kv.Name, &metav1.GetOptions{})
				if err != nil {
					return err
				}
				if curKv.Status.TargetDeploymentID != curKv.Status.ObservedDeploymentID {
					return fmt.Errorf("Target and obeserved id don't match")
				}

				podsReadyAndOwned := 0

				pods, err := virtClient.CoreV1().Pods(curKv.Namespace).List(context.Background(), metav1.ListOptions{LabelSelector: "kubevirt.io"})
				if err != nil {
					return err
				}

				for _, pod := range pods.Items {
					if !util.IsManagedByOperator(pod.Labels) {
						continue
					}

					if pod.Status.Phase != k8sv1.PodRunning {
						return fmt.Errorf("Waiting for pod %s with phase %s to reach Running phase", pod.Name, pod.Status.Phase)
					}

					for _, containerStatus := range pod.Status.ContainerStatuses {
						if !containerStatus.Ready {
							return fmt.Errorf("Waiting for pod %s to have all containers in Ready state", pod.Name)
						}
					}

					id, ok := pod.Annotations[v1.InstallStrategyIdentifierAnnotation]
					if !ok {
						return fmt.Errorf("Pod %s is owned by operator but has no id annotation", pod.Name)
					}

					expectedID := curKv.Status.ObservedDeploymentID
					if id != expectedID {
						return fmt.Errorf("Pod %s is of version %s when we expected id %s", pod.Name, id, expectedID)
					}
					podsReadyAndOwned++
				}

				// this just sanity checks that at least one pod was found and verified.
				// 0 would indicate our labeling was incorrect.
				Expect(podsReadyAndOwned).ToNot(Equal(0))

				return nil
			}, 300*time.Second, 1*time.Second).ShouldNot(HaveOccurred())
		}

		waitForUpdateCondition = func(kv *v1.KubeVirt) {
			Eventually(func() *v1.KubeVirt {
				kv, err := virtClient.KubeVirt(kv.Namespace).Get(kv.Name, &metav1.GetOptions{})
				Expect(err).ToNot(HaveOccurred())

				return kv
			}, 120*time.Second, 1*time.Second).Should(
				SatisfyAll(
					matcher.HaveConditionTrue(v1.KubeVirtConditionAvailable),
					matcher.HaveConditionTrue(v1.KubeVirtConditionProgressing),
					matcher.HaveConditionTrue(v1.KubeVirtConditionDegraded),
				))

		}

		waitForKvWithTimeout = func(newKv *v1.KubeVirt, timeoutSeconds int) {
			Eventually(func() error {
				kv, err := virtClient.KubeVirt(newKv.Namespace).Get(newKv.Name, &metav1.GetOptions{})
				if err != nil {
					return err
				}

				if kv.Status.Phase != v1.KubeVirtPhaseDeployed {
					return fmt.Errorf("Waiting for phase to be deployed (current phase: %+v)", kv.Status.Phase)
				}

				available := false
				progressing := true
				degraded := true
				created := false
				for _, condition := range kv.Status.Conditions {
					if condition.Type == v1.KubeVirtConditionAvailable && condition.Status == k8sv1.ConditionTrue {
						available = true
					} else if condition.Type == v1.KubeVirtConditionProgressing && condition.Status == k8sv1.ConditionFalse {
						progressing = false
					} else if condition.Type == v1.KubeVirtConditionDegraded && condition.Status == k8sv1.ConditionFalse {
						degraded = false
					} else if condition.Type == v1.KubeVirtConditionCreated && condition.Status == k8sv1.ConditionTrue {
						created = true
					}
				}

				if !available || progressing || degraded || !created {
					if kv.Status.ObservedGeneration != nil {
						if *kv.Status.ObservedGeneration == kv.ObjectMeta.Generation {
							return fmt.Errorf("observed generation must not match the current configuration")
						}
					}
					return fmt.Errorf("Waiting for conditions to indicate deployment (conditions: %+v)", kv.Status.Conditions)
				}

				if kv.Status.ObservedGeneration != nil {
					if *kv.Status.ObservedGeneration != kv.ObjectMeta.Generation {
						return fmt.Errorf("the observed generation must match the current generation")
					}
				}

				return nil
			}, time.Duration(timeoutSeconds)*time.Second, 1*time.Second).ShouldNot(HaveOccurred())
		}

		waitForKv = func(newKv *v1.KubeVirt) {
			waitForKvWithTimeout(newKv, 300)
		}

		patchKvProductNameVersionAndComponent = func(name, productName string, productVersion string, productComponent string) {

			format := `{ "op": "replace", "path": "%s", "value": "%s"}`
			data := []byte("[" + fmt.Sprintf(format, "/spec/productName", productName) + "," +
				fmt.Sprintf(format, "/spec/productVersion", productVersion) + "," +
				fmt.Sprintf(format, "/spec/productComponent", productComponent) +
				"]")

			Eventually(func() error {
				_, err := virtClient.KubeVirt(flags.KubeVirtInstallNamespace).Patch(name, types.JSONPatchType, data, &metav1.PatchOptions{})

				return err
			}, 10*time.Second, 1*time.Second).ShouldNot(HaveOccurred())
		}

		patchKvVersionAndRegistry = func(name string, version string, registry string) {
			data := []byte(fmt.Sprintf(`[{ "op": "replace", "path": "/spec/imageTag", "value": "%s"},{ "op": "replace", "path": "/spec/imageRegistry", "value": "%s"}]`, version, registry))
			Eventually(func() error {
				_, err := virtClient.KubeVirt(flags.KubeVirtInstallNamespace).Patch(name, types.JSONPatchType, data, &metav1.PatchOptions{})

				return err
			}, 10*time.Second, 1*time.Second).ShouldNot(HaveOccurred())
		}

		patchKvVersion = func(name string, version string) {
			data := []byte(fmt.Sprintf(`[{ "op": "add", "path": "/spec/imageTag", "value": "%s"}]`, version))
			Eventually(func() error {
				_, err := virtClient.KubeVirt(flags.KubeVirtInstallNamespace).Patch(name, types.JSONPatchType, data, &metav1.PatchOptions{})

				return err
			}, 10*time.Second, 1*time.Second).ShouldNot(HaveOccurred())
		}

		patchKvNodePlacement = func(name string, path string, verb string, componentConfig *v1.ComponentConfig) {
			var data []byte

			componentConfigData, _ := json.Marshal(componentConfig)

			data = []byte(fmt.Sprintf(`[{"op": "%s", "path": "/spec/%s", "value": %s}]`, verb, path, string(componentConfigData)))
			By(fmt.Sprintf("sending JSON patch: '%s'", string(data)))
			Eventually(func() error {
				_, err := virtClient.KubeVirt(flags.KubeVirtInstallNamespace).Patch(name, types.JSONPatchType, data, &metav1.PatchOptions{})

				return err
			}, 10*time.Second, 1*time.Second).ShouldNot(HaveOccurred())

		}

		patchKvNodePlacementExpectError = func(name string, path string, verb string, componentConfig *v1.ComponentConfig, errMsg string) {
			var data []byte

			componentConfigData, _ := json.Marshal(componentConfig)

			data = []byte(fmt.Sprintf(`[{"op": "%s", "path": "/spec/%s", "value": %s}]`, verb, path, string(componentConfigData)))
			By(fmt.Sprintf("sending JSON patch: '%s'", string(data)))
			_, err := virtClient.KubeVirt(flags.KubeVirtInstallNamespace).Patch(name, types.JSONPatchType, data, &metav1.PatchOptions{})
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring(errMsg))

		}

		patchKvInfra = func(infra *v1.ComponentConfig, expectError bool, errMsg string) {
			kv := copyOriginalKv()
			verb := "add"

			if kv.Spec.Infra != nil {
				verb = "replace"
			}
			if infra == nil {
				verb = "remove"
			}

			if !expectError {
				patchKvNodePlacement(kv.Name, "infra", verb, infra)
			} else {
				patchKvNodePlacementExpectError(kv.Name, "infra", verb, infra, errMsg)
			}
		}

		patchKvWorkloads = func(workloads *v1.ComponentConfig, expectError bool, errMsg string) {
			kv := copyOriginalKv()
			verb := "add"

			if kv.Spec.Workloads != nil {
				verb = "replace"
			}
			if workloads == nil {
				verb = "remove"
			}
			if !expectError {
				patchKvNodePlacement(kv.Name, "workloads", verb, workloads)
			} else {
				patchKvNodePlacementExpectError(kv.Name, "workloads", verb, workloads, errMsg)
			}
		}

		patchKvCertConfig = func(name string, certConfig *v1.KubeVirtSelfSignConfiguration) {
			var data []byte

			certRotationStrategy := v1.KubeVirtCertificateRotateStrategy{}
			certRotationStrategy.SelfSigned = certConfig
			certConfigData, _ := json.Marshal(certRotationStrategy)

			data = []byte(fmt.Sprintf(`[{"op": "%s", "path": "/spec/certificateRotateStrategy", "value": %s}]`, "replace", string(certConfigData)))
			By(fmt.Sprintf("sending JSON patch: '%s'", string(data)))
			Eventually(func() error {
				_, err := virtClient.KubeVirt(flags.KubeVirtInstallNamespace).Patch(name, types.JSONPatchType, data, &metav1.PatchOptions{})

				return err
			}, 10*time.Second, 1*time.Second).ShouldNot(HaveOccurred())

		}

		patchKvCertConfigExpectError = func(name string, certConfig *v1.KubeVirtSelfSignConfiguration) {
			var data []byte

			certRotationStrategy := v1.KubeVirtCertificateRotateStrategy{}
			certRotationStrategy.SelfSigned = certConfig
			certConfigData, _ := json.Marshal(certRotationStrategy)

			data = []byte(fmt.Sprintf(`[{"op": "%s", "path": "/spec/certificateRotateStrategy", "value": %s}]`, "replace", string(certConfigData)))
			By(fmt.Sprintf("sending JSON patch: '%s'", string(data)))
			Eventually(func() error {
				_, err := virtClient.KubeVirt(flags.KubeVirtInstallNamespace).Patch(name, types.JSONPatchType, data, &metav1.PatchOptions{})

				return err
			}, 10*time.Second, 1*time.Second).Should(HaveOccurred())

		}

		parseDaemonset = func(name string) (imagePrefix string) {
			var err error
			daemonSet, err := virtClient.AppsV1().DaemonSets(flags.KubeVirtInstallNamespace).Get(context.Background(), name, metav1.GetOptions{})
			Expect(err).ToNot(HaveOccurred())
			image := daemonSet.Spec.Template.Spec.Containers[0].Image
			imageRegEx := regexp.MustCompile(fmt.Sprintf("%s%s%s", `^(.*)/(.*)`, name, `([@:].*)?$`))
			matches := imageRegEx.FindAllStringSubmatch(image, 1)
			Expect(matches).To(HaveLen(1))
			Expect(matches[0]).To(HaveLen(4))
			imagePrefix = matches[0][2]
			return
		}

		parseImage = func(image string) (registry, imageName, version string) {
			var getVersion func(matches [][]string) string
			var imageRegEx *regexp.Regexp

			if strings.Contains(image, "@sha") {
				imageRegEx = regexp.MustCompile(`^(.+)/(.+)(@sha\d+:)([\da-fA-F]+)$`)
				getVersion = func(matches [][]string) string { return matches[0][3] + matches[0][4] }
			} else {
				imageRegEx = regexp.MustCompile(`^(.+)/(.+)(:.+)$`)
				getVersion = func(matches [][]string) string { return matches[0][3] }
			}

			matches := imageRegEx.FindAllStringSubmatch(image, 1)
			Expect(matches).To(HaveLen(1))
			registry = matches[0][1]
			imageName = matches[0][2]
			version = getVersion(matches)

			return
		}

		parseDeployment = func(name string) (deployment *v12.Deployment, image, registry, imageName, version string) {
			var err error
			deployment, err = virtClient.AppsV1().Deployments(flags.KubeVirtInstallNamespace).Get(context.Background(), name, metav1.GetOptions{})
			Expect(err).ToNot(HaveOccurred())
			image = deployment.Spec.Template.Spec.Containers[0].Image
			registry, imageName, version = parseImage(image)
			return
		}

		parseOperatorImage = func() (operator *v12.Deployment, image, registry, imageName, version string) {
			return parseDeployment("virt-operator")
		}

		patchOperator = func(newImageName, version *string) bool {

			modified := true

			Eventually(func() error {

				operator, oldImage, registry, oldImageName, oldVersion := parseOperatorImage()
				if newImageName == nil {
					// keep old prefix
					newImageName = &oldImageName
				}
				if version == nil {
					// keep old version
					version = &oldVersion
				} else {
					newVersion := components.AddVersionSeparatorPrefix(*version)
					version = &newVersion
				}
				newImage := fmt.Sprintf("%s/%s%s", registry, *newImageName, *version)

				if oldImage == newImage {
					modified = false
					return nil
				}

				operator.Spec.Template.Spec.Containers[0].Image = newImage
				for idx, env := range operator.Spec.Template.Spec.Containers[0].Env {
					if env.Name == util.VirtOperatorImageEnvName {
						env.Value = newImage
						operator.Spec.Template.Spec.Containers[0].Env[idx] = env
						break
					}
				}

				newTemplate, _ := json.Marshal(operator.Spec.Template)

				op := fmt.Sprintf(`[{ "op": "replace", "path": "/spec/template", "value": %s }]`, string(newTemplate))

				_, err = virtClient.AppsV1().Deployments(flags.KubeVirtInstallNamespace).Patch(context.Background(), "virt-operator", types.JSONPatchType, []byte(op), metav1.PatchOptions{})

				return err
			}, 15*time.Second, 1*time.Second).ShouldNot(HaveOccurred())

			return modified
		}

		installOperator = func(manifestPath string) {
			// namespace is already hardcoded within the manifests
			_, _, err = clientcmd.RunCommandWithNS(metav1.NamespaceNone, k8sClient, "apply", "-f", manifestPath)
			Expect(err).ToNot(HaveOccurred())

			By("Waiting for KubeVirt CRD to be created")
			ext, err := extclient.NewForConfig(virtClient.Config())
			Expect(err).ToNot(HaveOccurred())

			Eventually(func() error {
				_, err := ext.ApiextensionsV1().CustomResourceDefinitions().Get(context.Background(), "kubevirts.kubevirt.io", metav1.GetOptions{})
				return err
			}, 60*time.Second, 1*time.Second).ShouldNot(HaveOccurred())
		}

		installTestingManifests = func(manifestPath string) {
			_, _, err = clientcmd.RunCommandWithNS(metav1.NamespaceNone, k8sClient, "apply", "-f", manifestPath)
			Expect(err).ToNot(HaveOccurred())
		}

		deleteOperator = func(manifestPath string) {
			_, _, err = clientcmd.RunCommandWithNS(metav1.NamespaceNone, k8sClient, "delete", "-f", manifestPath)
			Expect(err).ToNot(HaveOccurred())
		}

		deleteTestingManifests = func(manifestPath string) {
			_, _, err = clientcmd.RunCommandWithNS(metav1.NamespaceNone, k8sClient, "delete", "-f", manifestPath)
			Expect(err).ToNot(HaveOccurred())
		}

		deleteAllKvAndWait = func(ignoreOriginal bool) {
			Eventually(func() error {

				kvs := util2.GetKvList(virtClient)

				deleteCount := 0
				for _, kv := range kvs {

					if ignoreOriginal && kv.Name == originalKv.Name {
						continue
					}
					deleteCount++
					if kv.DeletionTimestamp == nil {

						By("deleting the kv object")
						err := virtClient.KubeVirt(kv.Namespace).Delete(kv.Name, &metav1.DeleteOptions{})
						if err != nil {
							return err
						}
					}
				}
				if deleteCount != 0 {
					return fmt.Errorf("still waiting on %d kvs to delete", deleteCount)
				}
				return nil

			}, 240*time.Second, 1*time.Second).ShouldNot(HaveOccurred())

		}

		usesSha = func(image string) bool {
			return strings.Contains(image, "@sha256:")
		}

		ensureShasums = func() {
			if flags.SkipShasumCheck {
				log.Log.Warning("Cannot use shasums, skipping")
				return
			}

			for _, name := range []string{"virt-operator", "virt-api", "virt-controller"} {
				deployment, err := virtClient.AppsV1().Deployments(flags.KubeVirtInstallNamespace).Get(context.Background(), name, metav1.GetOptions{})
				Expect(err).ToNot(HaveOccurred())
				Expect(usesSha(deployment.Spec.Template.Spec.Containers[0].Image)).To(BeTrue(), fmt.Sprintf("%s should use sha", name))
			}

			handler, err := virtClient.AppsV1().DaemonSets(flags.KubeVirtInstallNamespace).Get(context.Background(), "virt-handler", metav1.GetOptions{})
			Expect(err).ToNot(HaveOccurred())
			Expect(usesSha(handler.Spec.Template.Spec.Containers[0].Image)).To(BeTrue(), "virt-handler should use sha")
		}

		// make sure virt deployments use shasums before we start
		ensureShasums()

		originalKv = util2.GetCurrentKv(virtClient)

		// save the operator sha
		_, _, _, _, version := parseOperatorImage()
		const errFmt = "version %s is expected to end with %s suffix"
		if !flags.SkipShasumCheck {
			const prefix = "@"
			Expect(strings.HasPrefix(version, "@")).To(BeTrue(), fmt.Sprintf(errFmt, version, prefix))
			originalOperatorVersion = strings.TrimPrefix(version, prefix)
		} else {
			const prefix = ":"
			Expect(strings.HasPrefix(version, ":")).To(BeTrue(), fmt.Sprintf(errFmt, version, prefix))
			originalOperatorVersion = strings.TrimPrefix(version, prefix)
		}

		if libstorage.HasDataVolumeCRD() {
			cdiList, err := virtClient.CdiClient().CdiV1beta1().CDIs().List(context.Background(), metav1.ListOptions{})
			Expect(err).ToNot(HaveOccurred())
			Expect(cdiList.Items).To(HaveLen(1))

			originalCDI = &cdiList.Items[0]
		}

		generateMigratableVMIs = func(num int) []*v1.VirtualMachineInstance {
			vmis := []*v1.VirtualMachineInstance{}
			for i := 0; i < num; i++ {
				vmi := tests.NewRandomVMIWithEphemeralDisk(cd.ContainerDiskFor(cd.ContainerDiskCirros))
				configMapName := "configmap-" + rand.String(5)
				secretName := "secret-" + rand.String(5)
				downwardAPIName := "downwardapi-" + rand.String(5)

				config_data := map[string]string{
					"config1": "value1",
					"config2": "value2",
				}

				secret_data := map[string]string{
					"user":     "admin",
					"password": "community",
				}

				tests.CreateConfigMap(configMapName, vmi.Namespace, config_data)
				tests.CreateSecret(secretName, vmi.Namespace, secret_data)

				tests.AddUserData(vmi, "cloud-init", "#!/bin/bash\necho 'hello'\n")
				tests.AddConfigMapDisk(vmi, configMapName, configMapName)
				tests.AddSecretDisk(vmi, secretName, secretName)
				tests.AddServiceAccountDisk(vmi, "default")
				// In case there are no existing labels add labels to add some data to the downwardAPI disk
				if vmi.ObjectMeta.Labels == nil {
					vmi.ObjectMeta.Labels = map[string]string{"downwardTestLabelKey": "downwardTestLabelVal"}
				}
				tests.AddLabelDownwardAPIVolume(vmi, downwardAPIName)
				tests.AddWatchdog(vmi, v1.WatchdogActionPoweroff)

				vmis = append(vmis, vmi)
			}

			lastVMIIndex := len(vmis) - 1
			vmi := vmis[lastVMIIndex]
			const nadName = "secondarynet"

			Expect(libnet.CreateNAD(vmi.GetNamespace(), nadName)).To(Succeed())

			const networkName = "tenant-blue"
			vmi.Spec.Domain.Devices.Interfaces = append(
				vmi.Spec.Domain.Devices.Interfaces,
				v1.Interface{
					Name:                   networkName,
					InterfaceBindingMethod: v1.InterfaceBindingMethod{Bridge: &v1.InterfaceBridge{}},
				},
			)
			vmi.Spec.Networks = append(
				vmi.Spec.Networks,
				v1.Network{
					Name: networkName,
					NetworkSource: v1.NetworkSource{Multus: &v1.MultusNetwork{
						NetworkName: nadName,
					}},
				},
			)

			return vmis
		}

		generateNonMigratableVMIs = func(num int) []*v1.VirtualMachineInstance {

			vmis := []*v1.VirtualMachineInstance{}
			for i := 0; i < num; i++ {
				vmi := tests.NewRandomVMIWithEphemeralDisk(cd.ContainerDiskFor(cd.ContainerDiskCirros))
				// Remove the masquerade interface to use the default bridge one
				// bridge interface isn't allowed to migrate
				vmi.Spec.Domain.Devices.Interfaces = nil
				vmi.Spec.Networks = nil
				vmis = append(vmis, vmi)
			}

			return vmis
		}

		startAllVMIs = func(vmis []*v1.VirtualMachineInstance) {
			for _, vmi := range vmis {
				vmi, err := virtClient.VirtualMachineInstance(testsuite.GetTestNamespace(vmi)).Create(context.Background(), vmi)
				Expect(err).ToNot(HaveOccurred(), "Create VMI successfully")
				libwait.WaitForSuccessfulVMIStart(vmi)
			}
		}

		deleteAllVMIs = func(vmis []*v1.VirtualMachineInstance) {
			for _, vmi := range vmis {
				err := virtClient.VirtualMachineInstance(vmi.Namespace).Delete(context.Background(), vmi.Name, &metav1.DeleteOptions{})
				Expect(err).ToNot(HaveOccurred(), "Delete VMI successfully")
			}
		}

		verifyVMIsEvicted = func(vmis []*v1.VirtualMachineInstance) {

			Eventually(func() error {
				for _, vmi := range vmis {
					vmi, err := virtClient.VirtualMachineInstance(vmi.Namespace).Get(context.Background(), vmi.Name, &metav1.GetOptions{})
					if err == nil && !vmi.IsFinal() {
						return fmt.Errorf("waiting for vmi %s/%s to shutdown as part of update", vmi.Namespace, vmi.Name)
					} else if !errors.IsNotFound(err) {
						return err
					}
				}
				return nil
			}, 320, 1).Should(BeNil(), "All VMIs should delete automatically")

		}

		verifyVMIsUpdated = func(vmis []*v1.VirtualMachineInstance, imageTag string) {

			Eventually(func() error {
				for _, vmi := range vmis {
					vmi, err := virtClient.VirtualMachineInstance(vmi.Namespace).Get(context.Background(), vmi.Name, &metav1.GetOptions{})
					if err != nil {
						return err
					}

					if vmi.Status.MigrationState == nil {
						return fmt.Errorf("waiting for vmi %s/%s to migrate as part of update", vmi.Namespace, vmi.Name)
					} else if !vmi.Status.MigrationState.Completed {

						var startTime time.Time
						var endTime time.Time
						now := time.Now()

						if vmi.Status.MigrationState.StartTimestamp != nil {
							startTime = vmi.Status.MigrationState.StartTimestamp.Time
						}
						if vmi.Status.MigrationState.EndTimestamp != nil {
							endTime = vmi.Status.MigrationState.EndTimestamp.Time
						}

						return fmt.Errorf("waiting for migration %s to complete for vmi %s/%s. Source Node [%s], Target Node [%s], Start Time [%s], End Time [%s], Now [%s], Failed: %t",
							string(vmi.Status.MigrationState.MigrationUID),
							vmi.Namespace,
							vmi.Name,
							vmi.Status.MigrationState.SourceNode,
							vmi.Status.MigrationState.TargetNode,
							startTime.String(),
							endTime.String(),
							now.String(),
							vmi.Status.MigrationState.Failed,
						)
					}

					_, hasOutdatedLabel := vmi.Labels[v1.OutdatedLauncherImageLabel]
					if hasOutdatedLabel {
						return fmt.Errorf("waiting for vmi %s/%s to have update launcher image in status", vmi.Namespace, vmi.Name)
					}
				}
				return nil
			}, 500, 1).Should(BeNil(), "All VMIs should update via live migration")

			// this is put in an eventually loop because it's possible for the VMI to complete
			// migrating and for the migration object to briefly lag behind in reporting
			// the results
			Eventually(func() error {
				By("Verifying only a single successful migration took place for each vmi")
				migrationList, err := virtClient.VirtualMachineInstanceMigration(testsuite.GetTestNamespace(nil)).List(&metav1.ListOptions{})
				Expect(err).ToNot(HaveOccurred(), "retrieving migrations")
				for _, vmi := range vmis {
					count := 0
					for _, migration := range migrationList.Items {
						if migration.Spec.VMIName == vmi.Name && migration.Status.Phase == v1.MigrationSucceeded {
							count++
						}
					}
					if count != 1 {
						return fmt.Errorf("vmi [%s] returned %d successful migrations", vmi.Name, count)
					}
				}
				return nil
			}, 10, 1).Should(BeNil(), "Expects only a single successful migration per workload update")
		}

		getVirtLauncherSha = func() string {
			str := originalKv.Status.ObservedDeploymentConfig
			config := &util.KubeVirtDeploymentConfig{}
			err := json.Unmarshal([]byte(str), config)
			Expect(err).ToNot(HaveOccurred())

			return config.VirtLauncherSha
		}

		generatePreviousVersionVmsnapshotYamls = func() {
			ext, err := extclient.NewForConfig(virtClient.Config())
			Expect(err).ToNot(HaveOccurred())

			crd, err := ext.ApiextensionsV1().CustomResourceDefinitions().Get(context.Background(), "virtualmachinesnapshots.snapshot.kubevirt.io", metav1.GetOptions{})
			Expect(err).ToNot(HaveOccurred())

			// Generate a vmsnapshot Yaml for every version
			// supported in the currently deployed KubeVirt
			// For every vm version
			supportedVersions := []string{}
			for _, version := range crd.Spec.Versions {
				supportedVersions = append(supportedVersions, version.Name)
			}

			for _, vmYaml := range vmYamls {
				vmSnapshots := []vmSnapshotDef{}
				for _, version := range supportedVersions {
					snapshotName := fmt.Sprintf("vm-%s-snapshot-%s", vmYaml.apiVersion, version)
					snapshotYaml := fmt.Sprintf(`apiVersion: snapshot.kubevirt.io/%s
kind: VirtualMachineSnapshot
metadata:
  name: %s
spec:
  source:
    apiGroup: kubevirt.io
    kind: VirtualMachine
    name: %s
`, version, snapshotName, vmYaml.vmName)
					restoreName := fmt.Sprintf("vm-%s-restore-%s", vmYaml.apiVersion, version)
					restoreYaml := fmt.Sprintf(`apiVersion: snapshot.kubevirt.io/%s
kind: VirtualMachineRestore
metadata:
  name: %s
spec:
  target:
    apiGroup: kubevirt.io
    kind: VirtualMachine
    name: %s
  virtualMachineSnapshotName: %s
`, version, restoreName, vmYaml.vmName, snapshotName)

					snapshotYamlFile := filepath.Join(workDir, fmt.Sprintf("%s.yaml", snapshotName))
					err = os.WriteFile(snapshotYamlFile, []byte(snapshotYaml), 0644)
					Expect(err).ToNot(HaveOccurred())

					restoreYamlFile := filepath.Join(workDir, fmt.Sprintf("%s.yaml", restoreName))
					err = os.WriteFile(restoreYamlFile, []byte(restoreYaml), 0644)
					Expect(err).ToNot(HaveOccurred())

					vmSnapshots = append(vmSnapshots, vmSnapshotDef{
						vmSnapshotName:  snapshotName,
						yamlFile:        snapshotYamlFile,
						restoreName:     restoreName,
						restoreYamlFile: restoreYamlFile,
					})
				}
				vmYamlTmp, _ := vmYamls[vmYaml.apiVersion]
				vmYamlTmp.vmSnapshots = vmSnapshots
			}
		}

		generatePreviousVersionVmYamls = func(previousUtilityRegistry string, previousUtilityTag string) {
			ext, err := extclient.NewForConfig(virtClient.Config())
			Expect(err).ToNot(HaveOccurred())

			crd, err := ext.ApiextensionsV1().CustomResourceDefinitions().Get(context.Background(), "virtualmachines.kubevirt.io", metav1.GetOptions{})
			Expect(err).ToNot(HaveOccurred())

			// Generate a vm Yaml for every version supported in the currently deployed KubeVirt

			supportedVersions := []string{}
			for _, version := range crd.Spec.Versions {
				supportedVersions = append(supportedVersions, version.Name)
			}

			for i, version := range supportedVersions {
				vmYaml := fmt.Sprintf(`apiVersion: kubevirt.io/%s
kind: VirtualMachine
metadata:
  labels:
    kubevirt.io/vm: vm-%s
  name: vm-%s
spec:
  dataVolumeTemplates:
  - metadata:
      name: test-dv%v
    spec:
      pvc:
        accessModes:
        - ReadWriteOnce
        resources:
          requests:
            storage: 1Gi
      source:
        blank: {}
  runStrategy: Manual
  template:
    metadata:
      labels:
        kubevirt.io/vm: vm-%s
    spec:
      domain:
        devices:
          disks:
          - disk:
              bus: virtio
            name: containerdisk
          - disk:
              bus: virtio
            name: cloudinitdisk
          - disk:
              bus: virtio
            name: datavolumedisk1
        machine:
          type: ""
        resources:
          requests:
            memory: 128M
      terminationGracePeriodSeconds: 0
      volumes:
      - dataVolume:
          name: test-dv%v
        name: datavolumedisk1
      - containerDisk:
          image: %s/%s-container-disk-demo:%s
        name: containerdisk
      - cloudInitNoCloud:
          userData: |
            #!/bin/sh

            echo 'printed from cloud-init userdata'
        name: cloudinitdisk
`, version, version, version, i, version, i, previousUtilityRegistry, cd.ContainerDiskCirros, previousUtilityTag)

				yamlFile := filepath.Join(workDir, fmt.Sprintf("vm-%s.yaml", version))
				err = os.WriteFile(yamlFile, []byte(vmYaml), 0644)
				Expect(err).ToNot(HaveOccurred())

				vmYamls[version] = &vmYamlDefinition{
					apiVersion:    version,
					vmName:        "vm-" + version,
					generatedYaml: vmYaml,
					yamlFile:      yamlFile,
				}
			}

		}

		fetchVirtHandlerCommand = func() string {
			virtHandler, err := virtClient.AppsV1().DaemonSets(flags.KubeVirtInstallNamespace).Get(context.Background(), "virt-handler", metav1.GetOptions{})
			Expect(err).ToNot(HaveOccurred())

			containers := virtHandler.Spec.Template.Spec.Containers
			Expect(containers).ToNot(BeEmpty())

			container := containers[0]
			return strings.Join(container.Command, " ")
		}
	})

	BeforeEach(func() {
		workDir = GinkgoT().TempDir()

		vmYamls = make(map[string]*vmYamlDefinition)

		verifyOperatorWebhookCertificate()
	})

	AfterEach(func() {
		deleteAllKvAndWait(true)

		kvs := util2.GetKvList(virtClient)
		if len(kvs) == 0 {
			By("Re-creating the original KV to stabilize")
			createKv(copyOriginalKv())
		}

		modified := patchOperator(nil, &originalOperatorVersion)
		if modified {
			// make sure we wait until redeploymemt started
			waitForUpdateCondition(originalKv)
		}

		By("Waiting for original KV to stabilize")
		waitForKvWithTimeout(originalKv, 420)
		allPodsAreReady(originalKv)

		// repost original CDI object if it doesn't still exist
		// in order to restore original environment
		if originalCDI != nil {
			cdiExists := false

			// ensure we wait for cdi to finish deleting before restoring it
			// in the event that cdi has the deletionTimestamp set.
			Eventually(func() bool {
				cdi, err := virtClient.CdiClient().CdiV1beta1().CDIs().Get(context.Background(), originalCDI.Name, metav1.GetOptions{})
				if err != nil && errors.IsNotFound(err) {
					// cdi isn't deleting and doesn't exist.
					return true
				} else {
					Expect(err).ToNot(HaveOccurred())
				}

				// wait for cdi to delete if deletionTimestamp is set
				if cdi.DeletionTimestamp != nil {
					return false
				}

				cdiExists = true
				return true
			}, 240*time.Second, 1*time.Second).Should(BeTrue())

			if cdiExists {
				cdi, err := virtClient.CdiClient().CdiV1beta1().CDIs().Get(context.Background(), originalCDI.Name, metav1.GetOptions{})
				Expect(err).ToNot(HaveOccurred())
				if !equality.Semantic.DeepEqual(cdi.Spec, originalCDI.Spec) {
					cdi.Spec = originalCDI.Spec
					_, err := virtClient.CdiClient().CdiV1beta1().CDIs().Update(context.Background(), cdi, metav1.UpdateOptions{})
					Expect(err).ToNot(HaveOccurred())
				}
			} else if !cdiExists {
				createCdi()
			}
		}

		// make sure virt deployments use shasums again after each test
		ensureShasums()

		// ensure that the state is fully restored after destructive tests
		verifyOperatorWebhookCertificate()

		_, err = virtClient.AppsV1().DaemonSets(flags.KubeVirtInstallNamespace).Get(context.Background(), "disks-images-provider", metav1.GetOptions{})
		Expect(err).ToNot(HaveOccurred(), "")
	})

	It("[test_id:1746]should have created and available condition", func() {
		kv := util2.GetCurrentKv(virtClient)

		By("verifying that created and available condition is present")
		waitForKv(kv)
	})

	Describe("[Serial]should reconcile components", Serial, func() {

		deploymentName := "virt-controller"
		daemonSetName := "virt-handler"
		envVarDeploymentKeyToUpdate := "USER_ADDED_ENV"

		crdName := "virtualmachines.kubevirt.io"
		shortNameAdded := "new"

		DescribeTable("checking updating resource is reverted to original state for ", func(changeResource func(), getResource func() runtime.Object, compareResource func() bool) {
			resource := getResource()
			By("Updating KubeVirt Object")
			changeResource()

			var generation int64
			By("Test that the added envvar was removed")
			Eventually(func() bool {
				equal := compareResource()
				if equal {
					r := getResource()
					o := r.(metav1.Object)
					generation = o.GetGeneration()
				}

				return equal
			}, 120*time.Second, 5*time.Second).Should(BeTrue(), "waiting for deployment to revert to original state")

			Eventually(func() int64 {
				currentKV := util2.GetCurrentKv(virtClient)
				return apply.GetExpectedGeneration(resource, currentKV.Status.Generations)
			}, 60*time.Second, 5*time.Second).Should(Equal(generation), "reverted deployment generation should be set on KV resource")

			By("Test that the expected generation is unchanged")
			Consistently(func() int64 {
				currentKV := util2.GetCurrentKv(virtClient)
				return apply.GetExpectedGeneration(resource, currentKV.Status.Generations)
			}, 30*time.Second, 5*time.Second).Should(Equal(generation))
		},

			Entry("[test_id:6254] deployments",

				func() {

					vc, err := virtClient.AppsV1().Deployments(originalKv.Namespace).Get(context.Background(), deploymentName, metav1.GetOptions{})
					Expect(err).ToNot(HaveOccurred())

					vc.Spec.Template.Spec.Containers[0].Env = []k8sv1.EnvVar{
						{
							Name:  envVarDeploymentKeyToUpdate,
							Value: "value",
						},
					}

					err = retry.RetryOnConflict(retry.DefaultRetry, func() error {
						vc, err = virtClient.AppsV1().Deployments(originalKv.Namespace).Update(context.Background(), vc, metav1.UpdateOptions{})
						return err
					})
					Expect(err).ToNot(HaveOccurred())
					Expect(vc.Spec.Template.Spec.Containers[0].Env[0].Name).To(Equal(envVarDeploymentKeyToUpdate))
				},

				func() runtime.Object {
					vc, err := virtClient.AppsV1().Deployments(originalKv.Namespace).Get(context.Background(), deploymentName, metav1.GetOptions{})
					Expect(err).ToNot(HaveOccurred())
					return vc
				},

				func() bool {
					vc, err := virtClient.AppsV1().Deployments(originalKv.Namespace).Get(context.Background(), deploymentName, metav1.GetOptions{})
					Expect(err).ToNot(HaveOccurred())

					for _, env := range vc.Spec.Template.Spec.Containers[0].Env {
						if env.Name == envVarDeploymentKeyToUpdate {
							return false
						}
					}

					return true
				}),

			Entry("[test_id:6255] customresourcedefinitions",
				func() {
					vmcrd, err := virtClient.ExtensionsClient().ApiextensionsV1().CustomResourceDefinitions().Get(context.Background(), crdName, metav1.GetOptions{})
					Expect(err).ToNot(HaveOccurred())

					vmcrd.Spec.Names.ShortNames = append(vmcrd.Spec.Names.ShortNames, shortNameAdded)

					err = retry.RetryOnConflict(retry.DefaultRetry, func() error {
						vmcrd, err = virtClient.ExtensionsClient().ApiextensionsV1().CustomResourceDefinitions().Update(context.Background(), vmcrd, metav1.UpdateOptions{})
						return err
					})
					Expect(err).ToNot(HaveOccurred())
					Expect(vmcrd.Spec.Names.ShortNames).To(ContainElement(shortNameAdded))
				},

				func() runtime.Object {
					vmcrd, err := virtClient.ExtensionsClient().ApiextensionsV1().CustomResourceDefinitions().Get(context.Background(), crdName, metav1.GetOptions{})
					Expect(err).ToNot(HaveOccurred())
					return vmcrd
				},

				func() bool {
					vmcrd, err := virtClient.ExtensionsClient().ApiextensionsV1().CustomResourceDefinitions().Get(context.Background(), crdName, metav1.GetOptions{})
					Expect(err).ToNot(HaveOccurred())

					for _, sn := range vmcrd.Spec.Names.ShortNames {
						if sn == shortNameAdded {
							return false
						}
					}

					return true
				}),
			Entry("[test_id:6256] poddisruptionbudgets",
				func() {
					pdb, err := virtClient.PolicyV1().PodDisruptionBudgets(originalKv.Namespace).Get(context.Background(), "virt-controller-pdb", metav1.GetOptions{})
					Expect(err).ToNot(HaveOccurred())

					pdb.Spec.Selector.MatchLabels = map[string]string{
						"kubevirt.io": "dne",
					}

					err = retry.RetryOnConflict(retry.DefaultRetry, func() error {
						pdb, err = virtClient.PolicyV1().PodDisruptionBudgets(originalKv.Namespace).Update(context.Background(), pdb, metav1.UpdateOptions{})
						return err
					})
					Expect(err).ToNot(HaveOccurred())
					Expect(pdb.Spec.Selector.MatchLabels["kubevirt.io"]).To(Equal("dne"))
				},

				func() runtime.Object {
					// No virt-controller PDB on single-replica deployments
					checks.SkipIfSingleReplica(virtClient)

					pdb, err := virtClient.PolicyV1().PodDisruptionBudgets(originalKv.Namespace).Get(context.Background(), "virt-controller-pdb", metav1.GetOptions{})
					Expect(err).ToNot(HaveOccurred())
					return pdb
				},

				func() bool {
					pdb, err := virtClient.PolicyV1().PodDisruptionBudgets(originalKv.Namespace).Get(context.Background(), "virt-controller-pdb", metav1.GetOptions{})
					Expect(err).ToNot(HaveOccurred())

					return pdb.Spec.Selector.MatchLabels["kubevirt.io"] != "dne"
				}),
			Entry("[test_id:6308] daemonsets",
				func() {
					vc, err := virtClient.AppsV1().DaemonSets(originalKv.Namespace).Get(context.Background(), daemonSetName, metav1.GetOptions{})
					Expect(err).ToNot(HaveOccurred())

					vc.Spec.Template.Spec.Containers[0].Env = append(vc.Spec.Template.Spec.Containers[0].Env, k8sv1.EnvVar{
						Name:  envVarDeploymentKeyToUpdate,
						Value: "value",
					})

					err = retry.RetryOnConflict(retry.DefaultRetry, func() error {
						vc, err = virtClient.AppsV1().DaemonSets(originalKv.Namespace).Update(context.Background(), vc, metav1.UpdateOptions{})
						return err
					})
					Expect(err).ToNot(HaveOccurred())
					Expect(vc.Spec.Template.Spec.Containers[0].Env).To(ContainElement(k8sv1.EnvVar{
						Name:  envVarDeploymentKeyToUpdate,
						Value: "value",
					}))
				},

				func() runtime.Object {
					var ds *v12.DaemonSet

					// wait for virt-handler readiness
					Eventually(func() bool {
						var err error
						ds, err = virtClient.AppsV1().DaemonSets(originalKv.Namespace).Get(context.Background(), daemonSetName, metav1.GetOptions{})
						Expect(err).ToNot(HaveOccurred())
						return ds.Status.DesiredNumberScheduled == ds.Status.NumberReady && ds.Spec.UpdateStrategy.RollingUpdate.MaxUnavailable.IntValue() == 1
					}, 60*time.Second, 1*time.Second).Should(BeTrue(), "waiting for daemonSet to be ready")
					return ds
				},

				func() bool {
					vc, err := virtClient.AppsV1().DaemonSets(originalKv.Namespace).Get(context.Background(), daemonSetName, metav1.GetOptions{})
					Expect(err).ToNot(HaveOccurred())

					for _, env := range vc.Spec.Template.Spec.Containers[0].Env {
						if env.Name == envVarDeploymentKeyToUpdate {
							return false
						}
					}

					return true
				}),
		)

		It("[test_id:6309] checking updating service is reverted to original state", func() {
			service, err := virtClient.CoreV1().Services(originalKv.Namespace).Get(context.Background(), "virt-api", metav1.GetOptions{})
			Expect(err).ToNot(HaveOccurred())

			originalPort := service.Spec.Ports[0].Port
			service.Spec.Ports[0].Port = 123

			By("Update service with undesired port")
			service, err = virtClient.CoreV1().Services(originalKv.Namespace).Update(context.Background(), service, metav1.UpdateOptions{})
			Expect(err).ToNot(HaveOccurred())
			Expect(service.Spec.Ports[0].Port).To(Equal(int32(123)))

			By("Test that the port is revered to the original")
			Eventually(func() bool {
				service, err := virtClient.CoreV1().Services(originalKv.Namespace).Get(context.Background(), "virt-api", metav1.GetOptions{})
				Expect(err).ToNot(HaveOccurred())

				return service.Spec.Ports[0].Port == originalPort
			}, 120*time.Second, 5*time.Second).Should(BeTrue(), "waiting for service to revert to original state")

			By("Test that the revert of the service stays consistent")
			Consistently(func() int32 {
				service, err = virtClient.CoreV1().Services(originalKv.Namespace).Get(context.Background(), "virt-api", metav1.GetOptions{})
				Expect(err).ToNot(HaveOccurred())

				return service.Spec.Ports[0].Port
			}, 20*time.Second, 5*time.Second).Should(Equal(originalPort))
		})
	})

	Describe("[rfe_id:2291][crit:high][vendor:cnv-qe@redhat.com][level:component]should start a VM", func() {
		It("[test_id:3144]using virt-launcher with a shasum", func() {

			if flags.SkipShasumCheck {
				Skip("Cannot currently test shasums, skipping")
			}

			By("starting a VM")
			vmi := tests.NewRandomVMIWithEphemeralDisk(cd.ContainerDiskFor(cd.ContainerDiskCirros))
			vmi, err = virtClient.VirtualMachineInstance(testsuite.GetTestNamespace(vmi)).Create(context.Background(), vmi)
			Expect(err).ToNot(HaveOccurred())
			libwait.WaitForSuccessfulVMIStart(vmi)

			By("getting virt-launcher")
			uid := vmi.GetObjectMeta().GetUID()
			labelSelector := fmt.Sprintf(v1.CreatedByLabel + "=" + string(uid))
			pods, err := virtClient.CoreV1().Pods(testsuite.GetTestNamespace(vmi)).List(context.Background(), metav1.ListOptions{LabelSelector: labelSelector})
			Expect(err).ToNot(HaveOccurred(), "Should list pods")
			Expect(pods.Items).To(HaveLen(1))
			Expect(usesSha(pods.Items[0].Spec.Containers[0].Image)).To(BeTrue(), "launcher pod should use shasum")

		})
	})

	Describe("[test_id:6987]should apply component configuration", func() {

		It("test VirtualMachineInstancesPerNode", func() {
			newVirtualMachineInstancesPerNode := 10
			maxDevicesCommandArgument := fmt.Sprintf("--max-devices %d", newVirtualMachineInstancesPerNode)

			By("Updating KubeVirt Object")
			kv, err := virtClient.KubeVirt(flags.KubeVirtInstallNamespace).Get(originalKv.Name, &metav1.GetOptions{})
			Expect(err).ToNot(HaveOccurred())
			Expect(kv.Spec.Configuration.VirtualMachineInstancesPerNode).ToNot(Equal(&newVirtualMachineInstancesPerNode))
			kv.Spec.Configuration.VirtualMachineInstancesPerNode = &newVirtualMachineInstancesPerNode

			kv, err = virtClient.KubeVirt(flags.KubeVirtInstallNamespace).Update(kv)
			Expect(err).ToNot(HaveOccurred())

			By("Waiting for virt-operator to apply changes to component")
			waitForKvWithTimeout(kv, 120)

			By("Test that patch was applied to DaemonSet")
			Eventually(fetchVirtHandlerCommand, 60*time.Second, 5*time.Second).Should(ContainSubstring(maxDevicesCommandArgument))

			By("Deleting patch from KubeVirt object")
			kv, err = virtClient.KubeVirt(flags.KubeVirtInstallNamespace).Get(originalKv.Name, &metav1.GetOptions{})
			Expect(err).ToNot(HaveOccurred())

			kv.Spec.Configuration.VirtualMachineInstancesPerNode = nil
			kv, err = virtClient.KubeVirt(flags.KubeVirtInstallNamespace).Update(kv)
			Expect(err).ToNot(HaveOccurred())

			By("Waiting for virt-operator to apply changes to component")
			waitForKvWithTimeout(kv, 120)

			By("Test that patch was removed from DaemonSet")
			Eventually(fetchVirtHandlerCommand, 60*time.Second, 5*time.Second).ShouldNot(ContainSubstring(maxDevicesCommandArgument))
		})
	})

	Describe("[test_id:4744][Serial]should apply component customization", Serial, func() {

		It("test applying and removing a patch", func() {
			annotationPatchValue := "new-annotation-value"
			annotationPatchKey := "applied-patch"

			By("Updating KubeVirt Object")
			kv, err := virtClient.KubeVirt(originalKv.Namespace).Get(originalKv.Name, &metav1.GetOptions{})
			Expect(err).ToNot(HaveOccurred())
			kv.Spec.CustomizeComponents = v1.CustomizeComponents{
				Patches: []v1.CustomizeComponentsPatch{
					{
						ResourceName: "virt-controller",
						ResourceType: "Deployment",
						Patch:        fmt.Sprintf(`{"spec":{"template": {"metadata": { "annotations": {"%s":"%s"}}}}}`, annotationPatchKey, annotationPatchValue),
						Type:         v1.StrategicMergePatchType,
					},
				},
			}

			kv, err = virtClient.KubeVirt(originalKv.Namespace).Update(kv)
			Expect(err).ToNot(HaveOccurred())
			generation := kv.GetGeneration()

			By("waiting for operator to patch the virt-controller component")
			Eventually(func() string {
				vc, err := virtClient.AppsV1().Deployments(originalKv.Namespace).Get(context.Background(), "virt-controller", metav1.GetOptions{})
				Expect(err).ToNot(HaveOccurred())
				return vc.Annotations[v1.KubeVirtGenerationAnnotation]
			}, 90*time.Second, 5*time.Second).Should(Equal(strconv.FormatInt(generation, 10)),
				"Rsource generation numbers should be identical on both the Kubevirt CR and the virt-controller resource")

			By("Test that patch was applied to deployment")
			vc, err := virtClient.AppsV1().Deployments(originalKv.Namespace).Get(context.Background(), "virt-controller", metav1.GetOptions{})
			Expect(err).ToNot(HaveOccurred())
			Expect(vc.Spec.Template.ObjectMeta.Annotations[annotationPatchKey]).To(Equal(annotationPatchValue))

			By("Waiting for virt-operator to apply changes to component")
			waitForKvWithTimeout(kv, 120)

			By("Check that KubeVirt CR generation does not get updated when applying patch")
			kv, err = virtClient.KubeVirt(originalKv.Namespace).Get(originalKv.Name, &metav1.GetOptions{})
			Expect(err).ToNot(HaveOccurred())
			Expect(kv.GetGeneration()).To(Equal(generation))

			By("Deleting patch from KubeVirt object")
			kv.Spec.CustomizeComponents = v1.CustomizeComponents{}
			kv, err = virtClient.KubeVirt(originalKv.Namespace).Update(kv)
			Expect(err).ToNot(HaveOccurred())
			generation = kv.GetGeneration()

			By("waiting for operator to patch the virt-controller component")
			Eventually(func() string {
				vc, err := virtClient.AppsV1().Deployments(originalKv.Namespace).Get(context.Background(), "virt-controller", metav1.GetOptions{})
				Expect(err).ToNot(HaveOccurred())
				return vc.Annotations[v1.KubeVirtGenerationAnnotation]
			}, 90*time.Second, 5*time.Second).Should(Equal(strconv.FormatInt(generation, 10)),
				"Rsource generation numbers should be identical on both the Kubevirt CR and the virt-controller resource")

			By("Test that patch was removed from deployment")
			vc, err = virtClient.AppsV1().Deployments(originalKv.Namespace).Get(context.Background(), "virt-controller", metav1.GetOptions{})
			Expect(err).ToNot(HaveOccurred())
			Expect(vc.Spec.Template.ObjectMeta.Annotations[annotationPatchKey]).To(BeEmpty())

			By("Waiting for virt-operator to apply changes to component")
			waitForKvWithTimeout(kv, 120)

		})
	})

	Describe("imagePullSecrets", func() {
		checkVirtComponents := func(imagePullSecrets []k8sv1.LocalObjectReference) {
			vc, err := virtClient.AppsV1().Deployments(originalKv.Namespace).Get(context.Background(), "virt-controller", metav1.GetOptions{})
			Expect(err).ToNot(HaveOccurred())
			Expect(vc.Spec.Template.Spec.ImagePullSecrets).To(Equal(imagePullSecrets))

			va, err := virtClient.AppsV1().Deployments(originalKv.Namespace).Get(context.Background(), "virt-api", metav1.GetOptions{})
			Expect(err).ToNot(HaveOccurred())
			Expect(va.Spec.Template.Spec.ImagePullSecrets).To(Equal(imagePullSecrets))

			vh, err := virtClient.AppsV1().DaemonSets(originalKv.Namespace).Get(context.Background(), "virt-handler", metav1.GetOptions{})
			Expect(err).ToNot(HaveOccurred())
			Expect(vh.Spec.Template.Spec.ImagePullSecrets).To(Equal(imagePullSecrets))

			if len(imagePullSecrets) == 0 {
				Expect(vh.Spec.Template.Spec.Containers).To(HaveLen(1))
			} else {
				Expect(vh.Spec.Template.Spec.Containers).To(HaveLen(2))
				Expect(vh.Spec.Template.Spec.Containers[1].Name).To(Equal("virt-launcher-image-holder"))
			}
		}

		checkVirtLauncherPod := func(vmi *v1.VirtualMachineInstance) {
			virtLauncherPod := tests.GetRunningPodByVirtualMachineInstance(vmi, vmi.Namespace)
			serviceAccount, err := virtClient.CoreV1().ServiceAccounts(vmi.Namespace).Get(context.Background(), virtLauncherPod.Spec.ServiceAccountName, metav1.GetOptions{})
			Expect(err).NotTo(HaveOccurred())
			Expect(equality.Semantic.DeepEqual(virtLauncherPod.Spec.ImagePullSecrets, serviceAccount.ImagePullSecrets)).To(BeTrue())
		}

		It("should not be present if not specified on the KubeVirt CR", func() {

			By("Check that KubeVirt CR has empty imagePullSecrets")
			kv, err := virtClient.KubeVirt(originalKv.Namespace).Get(originalKv.Name, &metav1.GetOptions{})
			Expect(err).NotTo(HaveOccurred())
			Expect(kv.Spec.ImagePullSecrets).To(BeEmpty())

			By("Ensuring that all virt components have empty image pull secrets")
			checkVirtComponents(nil)

			By("Starting a VMI")
			vmi := tests.NewRandomVMI()
			vmi, err = virtClient.VirtualMachineInstance(testsuite.GetTestNamespace(vmi)).Create(context.Background(), vmi)
			Expect(err).NotTo(HaveOccurred())
			libwait.WaitForSuccessfulVMIStart(vmi)

			By("Ensuring that virt-launcher pod does not have additional image pull secrets")
			checkVirtLauncherPod(vmi)

		})

		It("should be propagated if applied on the KubeVirt CR", func() {

			const imagePullSecretName = "testmyregistrykey"
			var imagePullSecrets = []k8sv1.LocalObjectReference{{Name: imagePullSecretName}}

			By("Delete existing image pull secret")
			_ = virtClient.CoreV1().Secrets(originalKv.Namespace).Delete(context.Background(), imagePullSecretName, metav1.DeleteOptions{})
			By("Create image pull secret")
			_, err := virtClient.CoreV1().Secrets(originalKv.Namespace).Create(context.Background(), &k8sv1.Secret{
				ObjectMeta: metav1.ObjectMeta{
					Name:      imagePullSecretName,
					Namespace: originalKv.Namespace,
				},
				Type: k8sv1.SecretTypeDockerConfigJson,
				Data: map[string][]byte{
					".dockerconfigjson": []byte(`{"auths":{"http://foo.example.com":{"username":"foo","password":"bar","email":"foo@example.com"}}}`),
				},
			}, metav1.CreateOptions{})
			Expect(err).NotTo(HaveOccurred())
			DeferCleanup(func() {
				By("Cleaning up image pull secret")
				err = virtClient.CoreV1().Secrets(originalKv.Namespace).Delete(context.Background(), imagePullSecretName, metav1.DeleteOptions{})
				Expect(err).NotTo(HaveOccurred())
			})

			By("Updating KubeVirt Object")
			kv, err := virtClient.KubeVirt(originalKv.Namespace).Get(originalKv.Name, &metav1.GetOptions{})
			Expect(err).NotTo(HaveOccurred())
			kv.Spec.ImagePullSecrets = imagePullSecrets
			kv, err = virtClient.KubeVirt(originalKv.Namespace).Update(kv)
			Expect(err).NotTo(HaveOccurred())

			By("Waiting for virt-operator to apply changes to component")
			waitForKv(kv)

			By("Ensuring that all virt components have expected image pull secrets")
			checkVirtComponents(imagePullSecrets)

			By("Starting a VMI")
			vmi := tests.NewRandomVMI()
			vmi, err = virtClient.VirtualMachineInstance(testsuite.GetTestNamespace(vmi)).Create(context.Background(), vmi)
			Expect(err).NotTo(HaveOccurred())
			libwait.WaitForSuccessfulVMIStart(vmi)

			By("Ensuring that virt-launcher pod does not have additional image pull secrets")
			checkVirtLauncherPod(vmi)

			By("Deleting imagePullSecrets from KubeVirt object")
			kv, err = virtClient.KubeVirt(originalKv.Namespace).Get(originalKv.Name, &metav1.GetOptions{})
			Expect(err).NotTo(HaveOccurred())
			kv.Spec.ImagePullSecrets = []k8sv1.LocalObjectReference{}
			kv, err = virtClient.KubeVirt(originalKv.Namespace).Update(kv)
			Expect(err).NotTo(HaveOccurred())

			By("Waiting for virt-operator to apply changes to component")
			waitForKv(kv)

			By("Ensuring that all virt components have empty image pull secrets")
			checkVirtComponents(nil)

		})

	})

	Describe("[rfe_id:2291][crit:high][vendor:cnv-qe@redhat.com][level:component]should update kubevirt", decorators.Upgrade, func() {
		runStrategyHalted := v1.RunStrategyHalted

		// This test is installing a previous release of KubeVirt
		// running a VM/VMI using that previous release
		// Updating KubeVirt to the target tested code
		// Ensuring VM/VMI is still operational after the update from previous release.
		DescribeTable("[release-blocker][test_id:3145]from previous release to target tested release", func(updateOperator bool) {
			if !libstorage.HasCDI() {
				Skip("Skip update test when CDI is not present")
			}

			if updateOperator && flags.OperatorManifestPath == "" {
				Skip("Skip operator update test when operator manifest path isn't configured")
			}

			// This test should run fine on single-node setups as long as no VM is created pre-update
			createVMs := true
			if !checks.HasAtLeastTwoNodes() {
				createVMs = false
			}

			var migratableVMIs []*v1.VirtualMachineInstance
			if createVMs {
				migratableVMIs = generateMigratableVMIs(2)
			}
			launcherSha := getVirtLauncherSha()
			if !flags.SkipShasumCheck {
				Expect(launcherSha).ToNot(Equal(""))
			}

			previousImageTag := flags.PreviousReleaseTag
			previousImageRegistry := flags.PreviousReleaseRegistry
			if previousImageTag == "" {
				previousImageTag, err = detectLatestUpstreamOfficialTag()
				Expect(err).ToNot(HaveOccurred())
				By(fmt.Sprintf("By Using detected tag %s for previous kubevirt", previousImageTag))
			} else {
				By(fmt.Sprintf("By Using user defined tag %s for previous kubevirt", previousImageTag))
			}

			previousUtilityTag := flags.PreviousUtilityTag
			previousUtilityRegistry := flags.PreviousUtilityRegistry
			if previousUtilityTag == "" {
				previousUtilityTag = previousImageTag
				By(fmt.Sprintf("By Using detected tag %s for previous utility containers", previousUtilityTag))
			} else {
				By(fmt.Sprintf("By Using user defined tag %s for previous utility containers", previousUtilityTag))
			}

			curVersion := originalKv.Status.ObservedKubeVirtVersion
			curRegistry := originalKv.Status.ObservedKubeVirtRegistry

			allPodsAreReady(originalKv)
			sanityCheckDeploymentsExist()

			// Delete current KubeVirt install so we can install previous release.
			By("Deleting KubeVirt object")
			deleteAllKvAndWait(false)

			By("Verifying all infra pods have terminated")
			allPodsAreTerminated(originalKv)

			By("Sanity Checking Deployments infrastructure is deleted")
			sanityCheckDeploymentsDeleted()

			if updateOperator {
				By("Deleting testing manifests")
				deleteTestingManifests(flags.TestingManifestPath)

				By("Deleting virt-operator installation")
				deleteOperator(flags.OperatorManifestPath)

				By("Installing previous release of virt-operator")
				manifestURL := getUpstreamReleaseAssetURL(previousImageTag, "kubevirt-operator.yaml")
				installOperator(manifestURL)
			}

			// Install previous release of KubeVirt
			By("Creating KubeVirt object")
			kv := copyOriginalKv()
			kv.Name = "kubevirt-release-install"
			kv.Spec.WorkloadUpdateStrategy.WorkloadUpdateMethods = []v1.WorkloadUpdateMethod{v1.WorkloadUpdateMethodLiveMigrate}

			// If updating via the KubeVirt CR, explicitly specify the desired release.
			if !updateOperator {
				kv.Spec.ImageTag = previousImageTag
				kv.Spec.ImageRegistry = previousImageRegistry
			}

			createKv(kv)

			// Wait for previous release to come online
			// wait 7 minutes because this test involves pulling containers
			// over the internet related to the latest kubevirt release
			By("Waiting for KV to stabilize")
			waitForKvWithTimeout(kv, 420)

			By("Verifying infrastructure is Ready")
			allPodsAreReady(kv)
			sanityCheckDeploymentsExist()

			// kubectl API discovery cache only refreshes every 10 minutes
			// Since we're likely dealing with api additions/removals here, we
			// need to ensure we're using a different cache directory after
			// the update from the previous release occurs.
			oldClientCacheDir := filepath.Join(workDir, "oldclient")
			err = os.MkdirAll(oldClientCacheDir, 0755)
			Expect(err).ToNot(HaveOccurred())
			newClientCacheDir := filepath.Join(workDir, "newclient")
			err = os.MkdirAll(newClientCacheDir, 0755)
			Expect(err).ToNot(HaveOccurred())

			// Create VM on previous release using a specific API.
			// NOTE: we are testing with yaml here and explicitly _NOT_ generating
			// this vm using the latest api code. We want to guarrantee there are no
			// surprises when it comes to backwards compatibility with previous
			// virt apis.  As we progress our api from v1alpha3 -> v1 there
			// needs to be a VM created for every api. This is how we will ensure
			// our api remains upgradable and supportable from previous release.

			if createVMs {
				generatePreviousVersionVmYamls(previousUtilityRegistry, previousUtilityTag)
				generatePreviousVersionVmsnapshotYamls()
			} else {
				Expect(vmYamls).To(BeEmpty())
			}
			for _, vmYaml := range vmYamls {
				By(fmt.Sprintf("Creating VM with %s api", vmYaml.vmName))
				// NOTE: using kubectl to post yaml directly
				_, _, err = clientcmd.RunCommand(k8sClient, "create", "-f", vmYaml.yamlFile, "--cache-dir", oldClientCacheDir)
				Expect(err).ToNot(HaveOccurred())

				for _, vmSnapshot := range vmYaml.vmSnapshots {
					By(fmt.Sprintf("Creating VM snapshot %s for vm %s", vmSnapshot.vmSnapshotName, vmYaml.vmName))
					_, _, err = clientcmd.RunCommand(k8sClient, "create", "-f", vmSnapshot.yamlFile, "--cache-dir", oldClientCacheDir)
					Expect(err).ToNot(HaveOccurred())
				}

				// Use Current virtctl to start VM
				// NOTE: we are using virtctl explicitly here because we want to start the VM
				// using the subresource endpoint in the same way virtctl performs this.
				By("Starting VM with virtctl")
				startCommand := clientcmd.NewRepeatableVirtctlCommand("start", "--namespace", testsuite.GetTestNamespace(nil), vmYaml.vmName)
				Expect(startCommand()).To(Succeed())

				By(fmt.Sprintf("Waiting for VM with %s api to become ready", vmYaml.apiVersion))

				Eventually(func() bool {
					virtualMachine, err := virtClient.VirtualMachine(testsuite.GetTestNamespace(nil)).Get(context.Background(), vmYaml.vmName, &metav1.GetOptions{})
					Expect(err).ToNot(HaveOccurred())
					if virtualMachine.Status.Ready {
						return true
					}
					return false
				}, 180*time.Second, 1*time.Second).Should(BeTrue())
			}

			By("Starting multiple migratable VMIs before performing update")
			startAllVMIs(migratableVMIs)

			// Update KubeVirt from the previous release to the testing target release.
			if updateOperator {
				By("Updating virt-operator installation")
				installOperator(flags.OperatorManifestPath)

				By("Re-installing testing manifests")
				installTestingManifests(flags.TestingManifestPath)
			} else {
				By("Updating KubeVirt object With current tag")
				patchKvVersionAndRegistry(kv.Name, curVersion, curRegistry)
			}

			By("Wait for Updating Condition")
			waitForUpdateCondition(kv)

			By("Waiting for KV to stabilize")
			waitForKvWithTimeout(kv, 420)

			By("Verifying infrastructure Is Updated")
			allPodsAreReady(kv)

			// Verify console connectivity to VMI still works and stop VM
			for _, vmYaml := range vmYamls {
				By(fmt.Sprintf("Ensuring vm %s is ready and latest API annotation is set", vmYaml.apiVersion))
				Eventually(func() bool {
					// We are using our internal client here on purpose to ensure we can interact
					// with previously created objects that may have been created using a different
					// api version from the latest one our client uses.
					virtualMachine, err := virtClient.VirtualMachine(testsuite.GetTestNamespace(nil)).Get(context.Background(), vmYaml.vmName, &metav1.GetOptions{})
					Expect(err).ToNot(HaveOccurred())
					if !virtualMachine.Status.Ready {
						return false
					}

					if !controller.ObservedLatestApiVersionAnnotation(virtualMachine) {
						return false
					}

					return true
				}, 180*time.Second, 1*time.Second).Should(BeTrue())

				By(fmt.Sprintf("Ensure vm %s vmsnapshots exist and ready ", vmYaml.vmName))
				for _, snapshot := range vmYaml.vmSnapshots {
					Eventually(func() bool {
						vmSnapshot, err := virtClient.VirtualMachineSnapshot(testsuite.GetTestNamespace(nil)).Get(context.Background(), snapshot.vmSnapshotName, metav1.GetOptions{})
						Expect(err).ToNot(HaveOccurred())
						if !(vmSnapshot.Status != nil && vmSnapshot.Status.ReadyToUse != nil && *vmSnapshot.Status.ReadyToUse) {
							return false
						}

						if vmSnapshot.Status.Phase != snapshotv1.Succeeded {
							return false
						}

						return true
					}, 120*time.Second, 3*time.Second).Should(BeTrue())
				}

				By(fmt.Sprintf("Connecting to %s's console", vmYaml.vmName))
				// This is in an eventually loop because it's possible for the
				// subresource endpoint routing to fail temporarily right after a deployment
				// completes while we wait for the kubernetes apiserver to detect our
				// subresource api server is online and ready to serve requests.
				Eventually(func() error {
					vmi, err := virtClient.VirtualMachineInstance(testsuite.GetTestNamespace(nil)).Get(context.Background(), vmYaml.vmName, &metav1.GetOptions{})
					Expect(err).ToNot(HaveOccurred())
					if err := console.LoginToCirros(vmi); err != nil {
						return err
					}
					return nil
				}, 60*time.Second, 1*time.Second).Should(BeNil())

				By("Stopping VM with virtctl")
				stopFn := clientcmd.NewRepeatableVirtctlCommand("stop", "--namespace", testsuite.GetTestNamespace(nil), vmYaml.vmName)
				Eventually(func() error {
					return stopFn()
				}, 30*time.Second, 1*time.Second).Should(BeNil())

				By("Waiting for VMI to stop")
				Eventually(func() bool {
					_, err := virtClient.VirtualMachineInstance(testsuite.GetTestNamespace(nil)).Get(context.Background(), vmYaml.vmName, &metav1.GetOptions{})
					if err != nil && errors.IsNotFound(err) {
						return true
					} else if err != nil {
						Expect(err).ToNot(HaveOccurred())
					}
					return false
					// #3610 - this timeout needs to be reduced back to 60 seconds.
					// there's an issue occurring after update where sometimes virt-launcher
					// can't dial the event notify socket. This impacts the timing for when
					// the vmi is shutdown. Once that is resolved, reduce the timeout
				}, 160*time.Second, 1*time.Second).Should(BeTrue())

				By("Ensuring we can Modify the VM Spec")
				Eventually(func() error {
					vm, err := virtClient.VirtualMachine(testsuite.GetTestNamespace(nil)).Get(context.Background(), vmYaml.vmName, &metav1.GetOptions{})
					if err != nil {
						return err
					}

					// by making a change to the VM, we ensure that writing the object is possible.
					// This ensures VMs created previously before the update are still compatible with our validation webhooks
					vm.Annotations["some-annotation"] = "some-val"
					annotationBytes, err := json.Marshal(vm.Annotations)
					Expect(err).ToNot(HaveOccurred())

					ops := fmt.Sprintf(`[{ "op": "add", "path": "/metadata/annotations", "value": %s }]`, string(annotationBytes))
					_, err = virtClient.VirtualMachine(vm.Namespace).Patch(context.Background(), vm.Name, types.JSONPatchType, []byte(ops), &metav1.PatchOptions{})
					return err
				}, 10*time.Second, 1*time.Second).ShouldNot(HaveOccurred())

				// change run stategy to halted to be able to restore the vm
				Eventually(func() bool {
					vm, err := virtClient.VirtualMachine(testsuite.GetTestNamespace(nil)).Get(context.Background(), vmYaml.vmName, &metav1.GetOptions{})
					if err != nil {
						return false
					}
					vm.Spec.RunStrategy = &runStrategyHalted
					_, err = virtClient.VirtualMachine(testsuite.GetTestNamespace(vm)).Update(context.Background(), vm)
					if err != nil {
						return false
					}
					updatedVM, err := virtClient.VirtualMachine(testsuite.GetTestNamespace(vm)).Get(context.Background(), vmYaml.vmName, &metav1.GetOptions{})
					Expect(err).ToNot(HaveOccurred())
					return updatedVM.Spec.Running == nil && updatedVM.Spec.RunStrategy != nil && *updatedVM.Spec.RunStrategy == runStrategyHalted
				}, 30*time.Second, 3*time.Second).Should(BeTrue())

				By(fmt.Sprintf("Ensure vm %s can be restored from vmsnapshots", vmYaml.vmName))
				for _, snapshot := range vmYaml.vmSnapshots {
					_, _, err = clientcmd.RunCommand(k8sClient, "create", "-f", snapshot.restoreYamlFile, "--cache-dir", newClientCacheDir)
					Expect(err).ToNot(HaveOccurred())
					Eventually(func() bool {
						r, err := virtClient.VirtualMachineRestore(testsuite.GetTestNamespace(nil)).Get(context.Background(), snapshot.restoreName, metav1.GetOptions{})
						if err != nil {
							return false
						}
						return r.Status != nil && r.Status.Complete != nil && *r.Status.Complete
					}, 180*time.Second, 3*time.Second).Should(BeTrue())
				}

				By(fmt.Sprintf("Deleting VM with %s api", vmYaml.apiVersion))
				_, _, err = clientcmd.RunCommand(k8sClient, "delete", "-f", vmYaml.yamlFile, "--cache-dir", newClientCacheDir)
				Expect(err).ToNot(HaveOccurred())

				By("Waiting for VM to be removed")
				Eventually(func() bool {
					_, err := virtClient.VirtualMachine(testsuite.GetTestNamespace(nil)).Get(context.Background(), vmYaml.vmName, &metav1.GetOptions{})
					if err != nil && errors.IsNotFound(err) {
						return true
					}
					return false
				}, 90*time.Second, 1*time.Second).Should(BeTrue())
			}

			By("Verifying all migratable vmi workloads are updated via live migration")
			verifyVMIsUpdated(migratableVMIs, launcherSha)

			if len(migratableVMIs) > 0 {
				By("Verifying that a once migrated VMI after an update can be migrated again")
				vmi := migratableVMIs[0]
				migration, err := virtClient.VirtualMachineInstanceMigration(vmi.Namespace).Create(tests.NewRandomMigration(vmi.Name, vmi.Namespace), &metav1.CreateOptions{})
				Expect(err).ToNot(HaveOccurred())
				Eventually(ThisMigration(migration), 180).Should(HaveSucceeded())
			}

			By("Deleting migratable VMIs")
			deleteAllVMIs(migratableVMIs)

			By("Deleting KubeVirt object")
			deleteAllKvAndWait(false)
		},
			Entry("by patching KubeVirt CR", false),
			Entry("by updating virt-operator", true),
		)
	})

	Describe("[rfe_id:2291][crit:high][vendor:cnv-qe@redhat.com][level:component]infrastructure management", func() {
		It("[test_id:3146]should be able to delete and re-create kubevirt install", decorators.Upgrade, func() {
			allPodsAreReady(originalKv)
			sanityCheckDeploymentsExist()

			// This ensures that we can remove kubevirt while workloads are running
			By("Starting some vmis")
			var vmis []*v1.VirtualMachineInstance
			if checks.HasAtLeastTwoNodes() {
				vmis = generateMigratableVMIs(2)
				startAllVMIs(vmis)
			}

			By("Deleting KubeVirt object")
			deleteAllKvAndWait(false)

			// this is just verifying some common known components do in fact get deleted.
			By("Sanity Checking Deployments infrastructure is deleted")
			sanityCheckDeploymentsDeleted()

			By("ensuring that namespaces can be successfully created and deleted")
			_, err := virtClient.CoreV1().Namespaces().Create(context.Background(), &k8sv1.Namespace{ObjectMeta: metav1.ObjectMeta{Name: testsuite.NamespaceTestOperator}}, metav1.CreateOptions{})
			if err != nil && !errors.IsAlreadyExists(err) {
				Expect(err).ToNot(HaveOccurred())
			}
			err = virtClient.CoreV1().Namespaces().Delete(context.Background(), testsuite.NamespaceTestOperator, metav1.DeleteOptions{})
			Expect(err).ToNot(HaveOccurred())
			Eventually(func() bool {
				_, err := virtClient.CoreV1().Namespaces().Get(context.Background(), testsuite.NamespaceTestOperator, metav1.GetOptions{})
				return errors.IsNotFound(err)
			}, 60*time.Second, 1*time.Second).Should(BeTrue())

			By("Creating KubeVirt Object")
			createKv(copyOriginalKv())

			By("Creating KubeVirt Object Created and Ready Condition")
			waitForKv(originalKv)

			By("Verifying infrastructure is Ready")
			allPodsAreReady(originalKv)
			// We're just verifying that a few common components that
			// should always exist get re-deployed.
			sanityCheckDeploymentsExist()
		})

		Describe("[rfe_id:3578][crit:high][vendor:cnv-qe@redhat.com][level:component] deleting with BlockUninstallIfWorkloadsExist", func() {
			It("[test_id:3683]should be blocked if a workload exists", func() {
				allPodsAreReady(originalKv)
				sanityCheckDeploymentsExist()

				By("setting the right uninstall strategy")
				Eventually(func() error {
					kv, err := virtClient.KubeVirt(originalKv.Namespace).Get(originalKv.Name, &metav1.GetOptions{})
					Expect(err).ToNot(HaveOccurred())
					kv.Spec.UninstallStrategy = v1.KubeVirtUninstallStrategyBlockUninstallIfWorkloadsExist
					_, err = virtClient.KubeVirt(kv.Namespace).Update(kv)
					return err
				}, 60*time.Second, time.Second).ShouldNot(HaveOccurred())

				By("creating a simple VMI")
				_, err = virtClient.VirtualMachineInstance(testsuite.GetTestNamespace(nil)).Create(context.Background(), tests.NewRandomVMIWithEphemeralDisk(cd.ContainerDiskFor(cd.ContainerDiskCirros)))
				Expect(err).ToNot(HaveOccurred())

				By("Deleting KubeVirt object")
				err = virtClient.KubeVirt(originalKv.Namespace).Delete(originalKv.Name, &metav1.DeleteOptions{})
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("there are still Virtual Machine Instances present"))
			})
		})

		It("[test_id:3148]should be able to create kubevirt install with custom image tag", decorators.Upgrade, func() {

			if flags.KubeVirtVersionTagAlt == "" {
				Skip("Skip operator custom image tag test because alt tag is not present")
			}

			allPodsAreReady(originalKv)
			sanityCheckDeploymentsExist()

			By("Deleting KubeVirt object")
			deleteAllKvAndWait(false)

			// this is just verifying some common known components do in fact get deleted.
			By("Sanity Checking Deployments infrastructure is deleted")
			sanityCheckDeploymentsDeleted()

			By("Creating KubeVirt Object")
			kv := copyOriginalKv()
			kv.Name = "kubevirt-alt-install"
			kv.Spec = v1.KubeVirtSpec{
				ImageTag:      flags.KubeVirtVersionTagAlt,
				ImageRegistry: flags.KubeVirtRepoPrefix,
			}
			createKv(kv)

			By("Creating KubeVirt Object Created and Ready Condition")
			waitForKv(kv)

			By("Verifying infrastructure is Ready")
			allPodsAreReady(kv)
			// We're just verifying that a few common components that
			// should always exist get re-deployed.
			sanityCheckDeploymentsExist()

			By("Deleting KubeVirt object")
			deleteAllKvAndWait(false)
		})

		// this test ensures that we can deal with image prefixes in case they are not used for tests already
		It("[test_id:3149]should be able to create kubevirt install with image prefix", decorators.Upgrade, func() {

			if flags.ImagePrefixAlt == "" {
				Skip("Skip operator imagePrefix test because imagePrefixAlt is not present")
			}

			kv := copyOriginalKv()

			allPodsAreReady(originalKv)
			sanityCheckDeploymentsExist()

			_, _, _, oldImageName, _ := parseOperatorImage()

			By("Update Operator using imagePrefixAlt")
			newImageName := flags.ImagePrefixAlt + oldImageName
			patchOperator(&newImageName, nil)

			// should result in kubevirt cr entering updating state
			By("Wait for Updating Condition")
			waitForUpdateCondition(kv)

			By("Waiting for KV to stabilize")
			waitForKv(kv)

			By("Verifying infrastructure Is Updated")
			allPodsAreReady(kv)

			By("Verifying deployments have correct image name")
			for _, name := range []string{"virt-operator", "virt-api", "virt-controller"} {
				_, _, _, actualImageName, _ := parseDeployment(name)
				Expect(actualImageName).To(ContainSubstring(flags.ImagePrefixAlt), fmt.Sprintf("%s should have correct image prefix", name))
			}
			handlerImageName := parseDaemonset("virt-handler")
			Expect(handlerImageName).To(ContainSubstring(flags.ImagePrefixAlt), "virt-handler should have correct image prefix")

			By("Verifying VMs are working")
			vmi := tests.NewRandomVMIWithEphemeralDisk(cd.ContainerDiskFor(cd.ContainerDiskAlpine))
			vmi, err := virtClient.VirtualMachineInstance(testsuite.GetTestNamespace(vmi)).Create(context.Background(), vmi)
			Expect(err).ShouldNot(HaveOccurred(), "Create VMI successfully")
			libwait.WaitForSuccessfulVMIStart(vmi)

			By("Verifying virt-launcher image is also prefixed")
			pod := tests.GetRunningPodByVirtualMachineInstance(vmi, vmi.Namespace)
			for _, container := range pod.Spec.Containers {
				if container.Name == "compute" {
					_, imageName, _ := parseImage(container.Image)
					Expect(imageName).To(ContainSubstring(flags.ImagePrefixAlt), "launcher image should have prefix")
				}
			}

			By("Deleting VM")
			err = virtClient.VirtualMachineInstance(testsuite.GetTestNamespace(vmi)).Delete(context.Background(), vmi.Name, &metav1.DeleteOptions{})
			Expect(err).ShouldNot(HaveOccurred(), "Delete VMI successfully")

			By("Restore Operator using original imagePrefix ")
			patchOperator(&oldImageName, nil)

			By("Wait for Updating Condition")
			waitForUpdateCondition(kv)

			By("Waiting for KV to stabilize")
			waitForKv(kv)

			By("Verifying infrastructure Is Restored to original version")
			allPodsAreReady(kv)
		})

		It("[test_id:3150]should be able to update kubevirt install with custom image tag", decorators.Upgrade, func() {
			if flags.KubeVirtVersionTagAlt == "" {
				Skip("Skip operator custom image tag test because alt tag is not present")
			}

			var vmis []*v1.VirtualMachineInstance
			if checks.HasAtLeastTwoNodes() {
				vmis = generateMigratableVMIs(2)
			}
			vmisNonMigratable := generateNonMigratableVMIs(2)

			allPodsAreReady(originalKv)
			sanityCheckDeploymentsExist()

			By("Deleting KubeVirt object")
			deleteAllKvAndWait(false)

			// this is just verifying some common known components do in fact get deleted.
			By("Sanity Checking Deployments infrastructure is deleted")
			sanityCheckDeploymentsDeleted()

			By("Creating KubeVirt Object")
			kv := copyOriginalKv()
			kv.Name = "kubevirt-alt-install"
			kv.Spec.Configuration.NetworkConfiguration = &v1.NetworkConfiguration{
				PermitBridgeInterfaceOnPodNetwork: pointer.BoolPtr(true),
			}
			kv.Spec.WorkloadUpdateStrategy.WorkloadUpdateMethods = []v1.WorkloadUpdateMethod{v1.WorkloadUpdateMethodLiveMigrate, v1.WorkloadUpdateMethodEvict}

			createKv(kv)

			By("Creating KubeVirt Object Created and Ready Condition")
			waitForKv(kv)

			By("Verifying infrastructure is Ready")
			allPodsAreReady(kv)
			// We're just verifying that a few common components that
			// should always exist get re-deployed.
			sanityCheckDeploymentsExist()

			By("Starting multiple migratable VMIs before performing update")
			startAllVMIs(vmis)
			startAllVMIs(vmisNonMigratable)

			By("Updating KubeVirtObject With Alt Tag")
			patchKvVersion(kv.Name, flags.KubeVirtVersionTagAlt)

			By("Wait for Updating Condition")
			waitForUpdateCondition(kv)

			By("Waiting for KV to stabilize")
			waitForKv(kv)

			By("Verifying infrastructure Is Updated")
			allPodsAreReady(kv)

			By("Verifying all non-migratable vmi workloads are shutdown")
			verifyVMIsEvicted(vmisNonMigratable)

			By("Verifying all migratable vmi workloads are updated via live migration")
			verifyVMIsUpdated(vmis, flags.KubeVirtVersionTagAlt)

			By("Deleting VMIs")
			deleteAllVMIs(vmis)
			deleteAllVMIs(vmisNonMigratable)

			By("Deleting KubeVirt object")
			deleteAllKvAndWait(false)

		})

		// NOTE - this test verifies new operators can grab the leader election lease
		// during operator updates. The only way the new infrastructure is deployed
		// is if the update operator is capable of getting the lease.
		It("[test_id:3151]should be able to update kubevirt install when operator updates if no custom image tag is set", decorators.Upgrade, func() {

			if flags.KubeVirtVersionTagAlt == "" {
				Skip("Skip operator custom image tag test because alt tag is not present")
			}

			kv := copyOriginalKv()

			allPodsAreReady(originalKv)
			sanityCheckDeploymentsExist()

			By("Update Virt-Operator using  Alt Tag")
			patchOperator(nil, &flags.KubeVirtVersionTagAlt)

			// should result in kubevirt cr entering updating state
			By("Wait for Updating Condition")
			waitForUpdateCondition(kv)

			By("Waiting for KV to stabilize")
			waitForKv(kv)

			By("Verifying infrastructure Is Updated")
			allPodsAreReady(kv)

			// by using the tag, we also test if resetting (in AfterEach) from tag to sha for the same "version" works
			By("Restore Operator Version using original tag. ")
			patchOperator(nil, &flags.KubeVirtVersionTag)

			By("Wait for Updating Condition")
			waitForUpdateCondition(kv)

			By("Waiting for KV to stabilize")
			waitForKv(kv)

			By("Verifying infrastructure Is Restored to original version")
			allPodsAreReady(kv)
		})

		// TODO: not Serial
		It("[test_id:3152]should fail if KV object already exists", func() {

			newKv := copyOriginalKv()
			newKv.Name = "someother-kubevirt"

			By("Creating another KubeVirt object")
			_, err = virtClient.KubeVirt(newKv.Namespace).Create(newKv)
			Expect(err).To(MatchError(ContainSubstring("Kubevirt is already created")))
		})

		It("[test_id:4612]should create non-namespaces resources without owner references", func() {
			crd, err := virtClient.ExtensionsClient().ApiextensionsV1().CustomResourceDefinitions().Get(context.Background(), "virtualmachineinstances.kubevirt.io", metav1.GetOptions{})
			Expect(err).ToNot(HaveOccurred())
			Expect(crd.ObjectMeta.OwnerReferences).To(BeEmpty())
		})

		It("[test_id:4613]should remove owner references on non-namespaces resources when updating a resource", func() {
			By("getting existing resource to reference")
			cm, err := virtClient.CoreV1().ConfigMaps(originalKv.Namespace).Get(context.Background(), "kubevirt-ca", metav1.GetOptions{})
			Expect(err).ToNot(HaveOccurred())
			ownerRef := []metav1.OwnerReference{
				*metav1.NewControllerRef(&k8sv1.ConfigMap{
					ObjectMeta: metav1.ObjectMeta{
						Name: cm.Name,
						UID:  cm.UID,
					},
				}, schema.GroupVersionKind{Version: "v1", Kind: "ConfigMap", Group: ""}),
			}

			By("adding an owner reference")
			origCRD, err := virtClient.ExtensionsClient().ApiextensionsV1().CustomResourceDefinitions().Get(context.Background(), "virtualmachineinstances.kubevirt.io", metav1.GetOptions{})
			Expect(err).ToNot(HaveOccurred())
			crd := origCRD.DeepCopy()
			crd.OwnerReferences = ownerRef
			patch := patchCRD(origCRD, crd)
			_, err = virtClient.ExtensionsClient().ApiextensionsV1().CustomResourceDefinitions().Patch(context.Background(), "virtualmachineinstances.kubevirt.io", types.MergePatchType, patch, metav1.PatchOptions{})
			Expect(err).ToNot(HaveOccurred())
			By("verifying that the owner reference is there")
			origCRD, err = virtClient.ExtensionsClient().ApiextensionsV1().CustomResourceDefinitions().Get(context.Background(), "virtualmachineinstances.kubevirt.io", metav1.GetOptions{})
			Expect(err).ToNot(HaveOccurred())
			Expect(origCRD.OwnerReferences).ToNot(BeEmpty())

			By("changing the install version to force an update")
			crd = origCRD.DeepCopy()
			crd.Annotations[v1.InstallStrategyVersionAnnotation] = "outdated"
			patch = patchCRD(origCRD, crd)
			_, err = virtClient.ExtensionsClient().ApiextensionsV1().CustomResourceDefinitions().Patch(context.Background(), "virtualmachineinstances.kubevirt.io", types.MergePatchType, patch, metav1.PatchOptions{})
			Expect(err).ToNot(HaveOccurred())

			By("waiting until the owner reference disappears again")
			Eventually(func() []metav1.OwnerReference {
				crd, err = virtClient.ExtensionsClient().ApiextensionsV1().CustomResourceDefinitions().Get(context.Background(), "virtualmachineinstances.kubevirt.io", metav1.GetOptions{})
				Expect(err).ToNot(HaveOccurred())
				return crd.OwnerReferences
			}, 20*time.Second, 1*time.Second).Should(BeEmpty())
			Expect(crd.ObjectMeta.OwnerReferences).To(BeEmpty())
		})

		It("[test_id:5010]should be able to update product related labels of kubevirt install", func() {
			productName := "kubevirt-test"
			productVersion := "0.0.0"
			productComponent := "kubevirt-component"
			allPodsAreReady(originalKv)
			sanityCheckDeploymentsExist()

			kv := copyOriginalKv()

			By("Patching kubevirt resource with productName , productVersion  and productComponent")
			patchKvProductNameVersionAndComponent(kv.Name, productName, productVersion, productComponent)

			for _, deployment := range []string{"virt-api", "virt-controller"} {
				By(fmt.Sprintf("Ensuring that the %s deployment is updated", deployment))
				Eventually(func() bool {
					dep, err := virtClient.AppsV1().Deployments(flags.KubeVirtInstallNamespace).Get(context.Background(), deployment, metav1.GetOptions{})
					Expect(err).ToNot(HaveOccurred())
					return dep.ObjectMeta.Labels[v1.AppVersionLabel] == productVersion && dep.ObjectMeta.Labels[v1.AppPartOfLabel] == productName && dep.ObjectMeta.Labels[v1.AppComponentLabel] == productComponent
				}, 240*time.Second, 1*time.Second).Should(BeTrue(), fmt.Sprintf("Expected labels to be updated for %s deployment", deployment))
			}

			By("Ensuring that the virt-handler daemonset is updated")
			Eventually(func() bool {
				dms, err := virtClient.AppsV1().DaemonSets(flags.KubeVirtInstallNamespace).Get(context.Background(), "virt-handler", metav1.GetOptions{})
				Expect(err).ToNot(HaveOccurred())
				return dms.ObjectMeta.Labels[v1.AppVersionLabel] == productVersion && dms.ObjectMeta.Labels[v1.AppPartOfLabel] == productName && dms.ObjectMeta.Labels[v1.AppComponentLabel] == productComponent
			}, 240*time.Second, 1*time.Second).Should(BeTrue(), "Expected labels to be updated for virt-handler daemonset")

			By("Deleting KubeVirt object")
			deleteAllKvAndWait(false)
		})

		Context("[rfe_id:2897][crit:medium][vendor:cnv-qe@redhat.com][level:component]With OpenShift cluster", func() {

			BeforeEach(func() {
				if !checks.IsOpenShift() {
					Skip("OpenShift operator tests should not be started on k8s")
				}
			})

			It("[test_id:2910]Should have kubevirt SCCs created", func() {
				const OpenShiftSCCLabel = "openshift.io/scc"
				var expectedSCCs, sccs []string

				By("Checking if kubevirt SCCs have been created")
				secClient := virtClient.SecClient()
				operatorSCCs := components.GetAllSCC(flags.KubeVirtInstallNamespace)
				for _, scc := range operatorSCCs {
					expectedSCCs = append(expectedSCCs, scc.GetName())
				}

				createdSCCs, err := secClient.SecurityContextConstraints().List(context.Background(), metav1.ListOptions{LabelSelector: controller.OperatorLabel})
				Expect(err).NotTo(HaveOccurred())
				for _, scc := range createdSCCs.Items {
					sccs = append(sccs, scc.GetName())
				}
				Expect(sccs).To(ConsistOf(expectedSCCs))

				By("Checking if virt-handler is assigned to kubevirt-handler SCC")
				l, err := labels.Parse("kubevirt.io=virt-handler")
				Expect(err).ToNot(HaveOccurred())

				pods, err := virtClient.CoreV1().Pods(flags.KubeVirtInstallNamespace).List(context.Background(), metav1.ListOptions{LabelSelector: l.String()})
				Expect(err).ToNot(HaveOccurred(), "Should get virt-handler")
				Expect(pods.Items).ToNot(BeEmpty())
				Expect(pods.Items[0].Annotations[OpenShiftSCCLabel]).To(
					Equal("kubevirt-handler"), "Should virt-handler be assigned to kubevirt-handler SCC",
				)

				By("Checking if virt-launcher is assigned to kubevirt-controller SCC")
				vmi := tests.NewRandomVMIWithEphemeralDisk(cd.ContainerDiskFor(cd.ContainerDiskCirros))
				vmi, err = virtClient.VirtualMachineInstance(testsuite.GetTestNamespace(vmi)).Create(context.Background(), vmi)
				Expect(err).ToNot(HaveOccurred())
				libwait.WaitForSuccessfulVMIStart(vmi)

				uid := vmi.GetObjectMeta().GetUID()
				labelSelector := fmt.Sprintf(v1.CreatedByLabel + "=" + string(uid))
				pods, err = virtClient.CoreV1().Pods(testsuite.GetTestNamespace(vmi)).List(context.Background(), metav1.ListOptions{LabelSelector: labelSelector})
				Expect(err).ToNot(HaveOccurred(), "Should get virt-launcher")
				Expect(pods.Items).To(HaveLen(1))
				Expect(pods.Items[0].Annotations[OpenShiftSCCLabel]).To(
					Equal("kubevirt-controller"), "Should virt-launcher be assigned to kubevirt-controller SCC",
				)
			})
		})
	})

	Describe("[rfe_id:2897][crit:medium][vendor:cnv-qe@redhat.com][level:component]Dynamic feature detection", func() {

		var vm *v1.VirtualMachine

		AfterEach(func() {

			if vm != nil {
				_, err := virtClient.VirtualMachine(vm.Namespace).Get(context.Background(), vm.Name, &metav1.GetOptions{})
				if err != nil && !errors.IsNotFound(err) {
					Expect(err).ToNot(HaveOccurred())
				}

				if vm != nil && vm.DeletionTimestamp == nil {
					err = virtClient.VirtualMachine(vm.Namespace).Delete(context.Background(), vm.Name, &metav1.DeleteOptions{})
					Expect(err).ToNot(HaveOccurred())
				}
				vm = nil
			}
		})

		It("[test_id:3153][QUARANTINE] Ensure infra can handle dynamically detecting DataVolume Support", decorators.Quarantine, func() {
			if !libstorage.HasDataVolumeCRD() {
				Skip("Can't test DataVolume support when DataVolume CRD isn't present")
			}
			// This tests starting infrastructure with and without the DataVolumes feature gate
			var foundSC bool
			vm, foundSC = tests.NewRandomVMWithDataVolume(cd.DataVolumeImportUrlForContainerDisk(cd.ContainerDiskAlpine), testsuite.GetTestNamespace(nil))
			if !foundSC {
				Skip("Skip test when Filesystem storage is not present")
			}

			running := false
			vm.Spec.Running = &running

			// Delete CDI object
			By("Deleting CDI install")
			Eventually(func() error {
				cdi, err := virtClient.CdiClient().CdiV1beta1().CDIs().Get(context.Background(), originalCDI.Name, metav1.GetOptions{})
				if err != nil && errors.IsNotFound(err) {
					// cdi is deleted
					return nil
				} else if err != nil {
					return err
				}

				if cdi.DeletionTimestamp == nil {
					err := virtClient.CdiClient().CdiV1beta1().CDIs().Delete(context.Background(), originalCDI.Name, metav1.DeleteOptions{})
					if err != nil {
						return err
					}
				}

				return fmt.Errorf("still waiting on cdi to delete")

			}, 240*time.Second, 1*time.Second).ShouldNot(HaveOccurred())

			// wait for virt-api and virt-controller to pick up the change that CDI no longer exists.
			time.Sleep(30 * time.Second)

			// Verify posting a VM with DataVolumeTemplate fails when DataVolumes
			// feature gate is disabled
			By("Expecting Error to Occur when posting VM with DataVolume")
			_, err = virtClient.VirtualMachine(testsuite.GetTestNamespace(nil)).Create(context.Background(), vm)
			Expect(err).To(HaveOccurred())

			// Enable DataVolumes by reinstalling CDI
			By("Enabling CDI install")
			createCdi()

			// wait for virt-api to pick up the change.
			time.Sleep(30 * time.Second)

			// Verify we can post a VM with DataVolumeTemplates successfully
			By("Expecting Error to not occur when posting VM with DataVolume")
			vm, err = virtClient.VirtualMachine(testsuite.GetTestNamespace(nil)).Create(context.Background(), vm)
			Expect(err).ToNot(HaveOccurred())

			By("Expecting VM to start successfully")
			tests.StartVirtualMachine(vm)
		})
	})

	Context("[rfe_id:2897][crit:medium][vendor:cnv-qe@redhat.com][level:component]With ServiceMonitor Disabled", func() {

		BeforeEach(func() {
			if tests.ServiceMonitorEnabled() {
				Skip("Test applies on when ServiceMonitor is not defined")
			}
		})

		It("[test_id:3154]Should not create RBAC Role or RoleBinding for ServiceMonitor", func() {
			rbacClient := virtClient.RbacV1()

			By("Checking that Role for ServiceMonitor doesn't exist")
			roleName := "kubevirt-service-monitor"
			_, err := rbacClient.Roles(flags.KubeVirtInstallNamespace).Get(context.Background(), roleName, metav1.GetOptions{})
			Expect(err).To(HaveOccurred())
			Expect(errors.IsNotFound(err)).To(BeTrue(), "Role 'kubevirt-service-monitor' should not have been created")

			By("Checking that RoleBinding for ServiceMonitor doesn't exist")
			_, err = rbacClient.RoleBindings(flags.KubeVirtInstallNamespace).Get(context.Background(), roleName, metav1.GetOptions{})
			Expect(err).To(HaveOccurred())
			Expect(errors.IsNotFound(err)).To(BeTrue(), "RoleBinding 'kubevirt-service-monitor' should not have been created")
		})
	})

	Context("With PrometheusRule Enabled", func() {

		BeforeEach(func() {
			if !tests.PrometheusRuleEnabled() {
				Skip("Test applies on when PrometheusRule is defined")
			}
		})

		It("[test_id:4614]Checks if the kubevirt PrometheusRule cr exists and verify it's spec", func() {
			monv1 := virtClient.PrometheusClient().MonitoringV1()
			prometheusRule, err := monv1.PrometheusRules(flags.KubeVirtInstallNamespace).Get(context.Background(), components.KUBEVIRT_PROMETHEUS_RULE_NAME, metav1.GetOptions{})
			Expect(err).ShouldNot(HaveOccurred())
			Expect(prometheusRule.Spec).ToNot(BeNil())
			Expect(prometheusRule.Spec.Groups).ToNot(BeEmpty())
			Expect(prometheusRule.Spec.Groups[0].Rules).ToNot(BeEmpty())

		})
	})

	Context("With PrometheusRule Disabled", func() {

		BeforeEach(func() {
			if tests.PrometheusRuleEnabled() {
				Skip("Test applies on when PrometheusRule is not defined")
			}
		})

		It("[test_id:4615]Checks that we do not deploy a PrometheusRule cr when not needed", func() {
			monv1 := virtClient.PrometheusClient().MonitoringV1()
			_, err := monv1.PrometheusRules(flags.KubeVirtInstallNamespace).Get(context.Background(), components.KUBEVIRT_PROMETHEUS_RULE_NAME, metav1.GetOptions{})
			Expect(err).To(HaveOccurred())
		})
	})

	Context("[rfe_id:2937][crit:medium][vendor:cnv-qe@redhat.com][level:component]With ServiceMonitor Enabled", func() {

		BeforeEach(func() {
			if !tests.ServiceMonitorEnabled() {
				Skip("Test requires ServiceMonitor to be valid")
			}
		})

		It("[test_id:2936]Should allow Prometheus to scrape KubeVirt endpoints", func() {
			coreClient := virtClient.CoreV1()

			// we don't know when the prometheus toolchain will pick up our config, so we retry plenty of times
			// before to give up. TODO: there is a smarter way to wait?
			Eventually(func() string {
				By("Obtaining Prometheus' configuration data")
				secret, err := coreClient.Secrets("openshift-monitoring").Get(context.Background(), "prometheus-k8s", metav1.GetOptions{})
				Expect(err).ToNot(HaveOccurred())

				data, ok := secret.Data["prometheus.yaml"]
				// In new versions of prometheus-operator, the configuration file is compressed in the secret
				if !ok {
					data, ok = secret.Data["prometheus.yaml.gz"]
					Expect(ok).To(BeTrue())

					By("Decompressing Prometheus' configuration data")
					gzreader, err := gzip.NewReader(bytes.NewReader(data))
					Expect(err).ToNot(HaveOccurred())

					decompressed, err := io.ReadAll(gzreader)
					Expect(err).ToNot(HaveOccurred())

					data = decompressed
				}

				Expect(data).ToNot(BeNil())

				By("Verifying that Prometheus is watching KubeVirt")
				return string(data)
			}, 90*time.Second, 3*time.Second).Should(ContainSubstring(flags.KubeVirtInstallNamespace), "Prometheus should be monitoring KubeVirt")
		})

		It("[test_id:4616]Should patch our namespace labels with openshift.io/cluster-monitoring=true", func() {
			By("Inspecting the labels on our namespace")
			namespace, err := virtClient.CoreV1().Namespaces().Get(context.Background(), flags.KubeVirtInstallNamespace, metav1.GetOptions{})
			Expect(err).ToNot(HaveOccurred())
			monitoringLabel, exists := namespace.ObjectMeta.Labels["openshift.io/cluster-monitoring"]
			Expect(exists).To(BeTrue())
			Expect(monitoringLabel).To(Equal("true"))
		})
	})

	It("[test_id:4617]should adopt previously unmanaged entities by updating its metadata", func() {
		By("removing registration metadata")
		patchData := []byte(fmt.Sprint(`[{ "op": "replace", "path": "/metadata/labels", "value": {} }]`))
		_, err = virtClient.CoreV1().Secrets(flags.KubeVirtInstallNamespace).Patch(context.Background(), components.VirtApiCertSecretName, types.JSONPatchType, patchData, metav1.PatchOptions{})
		Expect(err).ToNot(HaveOccurred())
		_, err = aggregatorClient.ApiregistrationV1().APIServices().Patch(context.Background(), fmt.Sprintf("%s.subresources.kubevirt.io", v1.ApiLatestVersion), types.JSONPatchType, patchData, metav1.PatchOptions{})
		Expect(err).ToNot(HaveOccurred())
		_, err = virtClient.AdmissionregistrationV1().ValidatingWebhookConfigurations().Patch(context.Background(), components.VirtAPIValidatingWebhookName, types.JSONPatchType, patchData, metav1.PatchOptions{})
		Expect(err).ToNot(HaveOccurred())
		_, err = virtClient.AdmissionregistrationV1().MutatingWebhookConfigurations().Patch(context.Background(), components.VirtAPIMutatingWebhookName, types.JSONPatchType, patchData, metav1.PatchOptions{})
		Expect(err).ToNot(HaveOccurred())

		By("checking that it gets added again")
		Eventually(func() map[string]string {
			secret, err := virtClient.CoreV1().Secrets(flags.KubeVirtInstallNamespace).Get(context.Background(), components.VirtApiCertSecretName, metav1.GetOptions{})
			Expect(err).ToNot(HaveOccurred())
			return secret.Labels
		}, 20*time.Second, 1*time.Second).Should(HaveKeyWithValue(v1.ManagedByLabel, v1.ManagedByLabelOperatorValue))
		Eventually(func() map[string]string {
			apiService, err := aggregatorClient.ApiregistrationV1().APIServices().Get(context.Background(), fmt.Sprintf("%s.subresources.kubevirt.io", v1.ApiLatestVersion), metav1.GetOptions{})
			Expect(err).ToNot(HaveOccurred())
			return apiService.Labels
		}, 20*time.Second, 1*time.Second).Should(HaveKeyWithValue(v1.ManagedByLabel, v1.ManagedByLabelOperatorValue))
		Eventually(func() map[string]string {
			validatingWebhook, err := virtClient.AdmissionregistrationV1().ValidatingWebhookConfigurations().Get(context.Background(), components.VirtAPIValidatingWebhookName, metav1.GetOptions{})
			Expect(err).ToNot(HaveOccurred())
			return validatingWebhook.Labels
		}, 20*time.Second, 1*time.Second).Should(HaveKeyWithValue(v1.ManagedByLabel, v1.ManagedByLabelOperatorValue))
		Eventually(func() map[string]string {
			mutatingWebhook, err := virtClient.AdmissionregistrationV1().MutatingWebhookConfigurations().Get(context.Background(), components.VirtAPIMutatingWebhookName, metav1.GetOptions{})
			Expect(err).ToNot(HaveOccurred())
			return mutatingWebhook.Labels
		}, 20*time.Second, 1*time.Second).Should(HaveKeyWithValue(v1.ManagedByLabel, v1.ManagedByLabelOperatorValue))
	})

	Context("[rfe_id:4356]Node Placement", func() {
		It("[test_id:4927]should dynamically update infra config", func() {
			// This label shouldn't exist, but this isn't harmful
			// existing/running deployments will not be torn down until
			// new ones are stood up (and the new ones will get stuck in scheduling)
			fakeLabelKey := "kubevirt-test"
			fakeLabelValue := "test-label"
			infra := v1.ComponentConfig{
				NodePlacement: &v1.NodePlacement{
					NodeSelector: map[string]string{fakeLabelKey: fakeLabelValue},
				},
			}
			By("Adding fake label to Virt components")
			patchKvInfra(&infra, false, "")
			for _, deploymentName := range []string{"virt-controller", "virt-api"} {
				errMsg := "NodeSelector should be propegated to the deployment eventually"
				Eventually(func() bool {
					return nodeSelectorExistInDeployment(virtClient, deploymentName, fakeLabelKey, fakeLabelValue)
				}, 60*time.Second, 1*time.Second).Should(BeTrue(), errMsg)
				//The reason we check this is that sometime it takes a while until the pod is created and
				//if the pod is created after the call to allPodsAreReady in the AfterEach scope
				//than we will run the next test with side effect of pending pods of virt-api and virt-controller
				//and increase flakiness
				errMsg = "the deployment should try to rollup the pods with the new selector and fail to schedule pods because the nodes don't have the fake label"
				Eventually(func() bool {
					return atLeastOnePendingPodExistInDeployment(virtClient, deploymentName)
				}, 60*time.Second, 1*time.Second).Should(BeTrue(), errMsg)
			}
			patchKvInfra(nil, false, "")
		})

		It("[test_id:4928]should dynamically update workloads config", func() {
			labelKey := "kubevirt-test"
			labelValue := "test-label"
			workloads := v1.ComponentConfig{
				NodePlacement: &v1.NodePlacement{
					NodeSelector: map[string]string{labelKey: labelValue},
				},
			}
			patchKvWorkloads(&workloads, false, "")

			Eventually(func() bool {
				daemonset, err := virtClient.AppsV1().DaemonSets(flags.KubeVirtInstallNamespace).Get(context.Background(), "virt-handler", metav1.GetOptions{})
				Expect(err).ToNot(HaveOccurred())
				if daemonset.Spec.Template.Spec.NodeSelector == nil || daemonset.Spec.Template.Spec.NodeSelector[labelKey] != labelValue {
					return false
				}
				return true
			}, 60*time.Second, 1*time.Second).Should(BeTrue())

			patchKvWorkloads(nil, false, "")
		})

		It("should reject infra placement configuration with incorrect toleration operator", func() {
			incorrectOperator := "foo"
			incorrectWorkload := v1.ComponentConfig{
				NodePlacement: &v1.NodePlacement{
					Tolerations: []k8sv1.Toleration{{
						Key:      "someKey",
						Operator: k8sv1.TolerationOperator(incorrectOperator),
						Value:    "someValue",
					}},
				},
			}
			errMsg := "spec.infra.nodePlacement.tolerations.operator in body should be one of"
			patchKvInfra(&incorrectWorkload, true, errMsg)
		})

		It("should reject workload placement configuration with incorrect toleraion operator", func() {
			incorrectOperator := "foo"
			incorrectWorkload := v1.ComponentConfig{
				NodePlacement: &v1.NodePlacement{
					Tolerations: []k8sv1.Toleration{{
						Key:      "someKey",
						Operator: k8sv1.TolerationOperator(incorrectOperator),
						Value:    "someValue",
					}},
				},
			}
			errMsg := "spec.workloads.nodePlacement.tolerations.operator in body should be one of"
			patchKvWorkloads(&incorrectWorkload, true, errMsg)
		})

		It("[test_id:8235]should check if kubevirt components have linux node selector", func() {
			By("Listing only kubevirt components")

			kv := util2.GetCurrentKv(virtClient)
			productComponent := kv.Spec.ProductComponent
			if productComponent == "" {
				productComponent = "kubevirt"
			}

			labelReq, err := labels.NewRequirement("app.kubernetes.io/component", selection.In, []string{productComponent})

			if err != nil {
				panic(err)
			}

			By("Looking for pods with " + productComponent + " component")

			pods, err := virtClient.CoreV1().Pods(flags.KubeVirtInstallNamespace).List(context.Background(), metav1.ListOptions{
				LabelSelector: labels.NewSelector().Add(
					*labelReq,
				).String(),
			})
			Expect(err).NotTo(HaveOccurred())
			Expect(pods.Items).NotTo(BeEmpty())

			By("Checking nodeselector")
			for _, pod := range pods.Items {
				Expect(pod.Spec.NodeSelector).To(HaveKeyWithValue(k8sv1.LabelOSStable, "linux"), fmt.Sprintf("pod %s does not have linux node selector", pod.Name))
			}
		})
	})

	Context("Replicas", func() {
		It("should fail to set replicas to 0", func() {
			var replicas uint8 = 0
			infra := &v1.ComponentConfig{
				Replicas: &replicas,
			}
			patchKvInfra(infra, true, "infra replica count can't be 0")
		})
		It("should dynamically adjust virt- pod count and PDBs", func() {
			for _, replicas := range []uint8{3, 1, 2} {
				By(fmt.Sprintf("Setting the replica count in kvInfra to %d", replicas))
				var infra *v1.ComponentConfig
				infra = &v1.ComponentConfig{
					Replicas: &replicas,
				}
				patchKvInfra(infra, false, "")

				By(fmt.Sprintf("Expecting %d replicas of virt-api and virt-controller", replicas))
				Eventually(func() []k8sv1.Pod {
					for _, name := range []string{"virt-api", "virt-controller"} {
						pods, err := virtClient.CoreV1().Pods(flags.KubeVirtInstallNamespace).List(context.Background(), metav1.ListOptions{LabelSelector: fmt.Sprintf("%s=%s", v1.AppLabel, name)})
						Expect(err).ToNot(HaveOccurred())
						return pods.Items
					}
					return nil
				}, 120*time.Second, 1*time.Second).Should(HaveLen(int(replicas)), fmt.Sprintf("Replicas of virt-api and virt-controller are not %d", replicas))

				if replicas == 1 {
					By(fmt.Sprintf("Expecting PDBs to disppear"))
					Eventually(func() bool {
						for _, name := range []string{"virt-api", "virt-controller"} {
							_, err := virtClient.PolicyV1().PodDisruptionBudgets(flags.KubeVirtInstallNamespace).Get(context.Background(), name+"-pdb", metav1.GetOptions{})
							if err == nil {
								return false
							}
						}
						return true
					}, 60*time.Second, 1*time.Second).Should(BeTrue(), "PDBs have not disappeared")
				} else {
					By(fmt.Sprintf("Expecting minAvailable to become %d on the PDBs", replicas-1))
					Eventually(func() int {
						for _, name := range []string{"virt-api", "virt-controller"} {
							pdb, err := virtClient.PolicyV1().PodDisruptionBudgets(flags.KubeVirtInstallNamespace).Get(context.Background(), name+"-pdb", metav1.GetOptions{})
							Expect(err).ToNot(HaveOccurred())
							return pdb.Spec.MinAvailable.IntValue()
						}
						return -1
					}, 60*time.Second, 1*time.Second).Should(BeEquivalentTo(int(replicas-1)), fmt.Sprintf("minAvailable is not become %d on the PDBs", replicas-1))
				}
			}
		})
		It("should update new single-replica CRs with a finalizer and be stable", func() {
			By("copying the original kv CR")
			kvOrig := copyOriginalKv()
			kv := copyOriginalKv()

			By("storing the actual replica counts for the cluster")
			originalReplicaCounts := make(map[string]int)
			for _, name := range []string{"virt-api", "virt-controller"} {
				pods, err := virtClient.CoreV1().Pods(flags.KubeVirtInstallNamespace).List(context.Background(), metav1.ListOptions{LabelSelector: fmt.Sprintf("%s=%s", v1.AppLabel, name)})
				Expect(err).ToNot(HaveOccurred())
				originalReplicaCounts[name] = len(pods.Items)
			}

			By("deleting the kv CR")
			err = virtClient.KubeVirt(kv.Namespace).Delete(kv.Name, &metav1.DeleteOptions{})
			Expect(err).ToNot(HaveOccurred())

			By("waiting for virt-api and virt-controller to be gone")
			Eventually(func() bool {
				for _, name := range []string{"virt-api", "virt-controller"} {
					pods, err := virtClient.CoreV1().Pods(flags.KubeVirtInstallNamespace).List(context.Background(), metav1.ListOptions{LabelSelector: fmt.Sprintf("%s=%s", v1.AppLabel, name)})
					Expect(err).ToNot(HaveOccurred())
					if len(pods.Items) != 0 {
						return false
					}
				}
				return true
			}, 120*time.Second, 4*time.Second).Should(BeTrue())

			By("waiting for the kv CR to be gone")
			Eventually(func() bool {
				_, err := virtClient.KubeVirt(kv.Namespace).Get(kv.Name, &metav1.GetOptions{})
				return errors.IsNotFound(err)
			}, 120*time.Second, 4*time.Second).Should(BeTrue())

			By("creating a new single-replica kv CR")
			if kv.Spec.Infra == nil {
				kv.Spec.Infra = &v1.ComponentConfig{}
			}
			var one uint8 = 1
			kv.Spec.Infra.Replicas = &one
			kv, err = virtClient.KubeVirt(kv.Namespace).Create(kv)
			Expect(err).ToNot(HaveOccurred())

			By("waiting for the kv CR to get a finalizer")
			Eventually(func() bool {
				kv, err = virtClient.KubeVirt(kv.Namespace).Get(kv.Name, &metav1.GetOptions{})
				Expect(err).ToNot(HaveOccurred())
				return len(kv.Finalizers) > 0
			}, 120*time.Second, 4*time.Second).Should(BeTrue())

			By("ensuring the CR generation is stable")
			Expect(err).ToNot(HaveOccurred())
			Consistently(func() int64 {
				kv2, err := virtClient.KubeVirt(kv.Namespace).Get(kv.Name, &metav1.GetOptions{})
				Expect(err).ToNot(HaveOccurred())
				return kv2.GetGeneration()
			}, 30*time.Second, 2*time.Second).Should(Equal(kv.GetGeneration()))

			By("restoring the original replica count")
			patchKvInfra(kvOrig.Spec.Infra, false, "")

			By("waiting for virt-api and virt-controller replicas to respawn")
			Eventually(func() error {
				for _, name := range []string{"virt-api", "virt-controller"} {
					pods, err := virtClient.CoreV1().Pods(flags.KubeVirtInstallNamespace).List(context.Background(), metav1.ListOptions{LabelSelector: fmt.Sprintf("%s=%s", v1.AppLabel, name)})
					Expect(err).ToNot(HaveOccurred())
					if len(pods.Items) != originalReplicaCounts[name] {
						return fmt.Errorf("expected %d replicas for %s, got %d", originalReplicaCounts[name], name, len(pods.Items))
					}
				}
				return nil
			}, 120*time.Second, 4*time.Second).ShouldNot(HaveOccurred())
		})
	})

	Context("Certificate Rotation", func() {
		var certConfig *v1.KubeVirtSelfSignConfiguration
		BeforeEach(func() {
			certConfig = &v1.KubeVirtSelfSignConfiguration{
				CA: &v1.CertConfig{
					Duration:    &metav1.Duration{Duration: 24 * time.Hour},
					RenewBefore: &metav1.Duration{Duration: 16 * time.Hour},
				},
				Server: &v1.CertConfig{
					Duration:    &metav1.Duration{Duration: 12 * time.Hour},
					RenewBefore: &metav1.Duration{Duration: 10 * time.Hour},
				},
			}
		})

		It("[test_id:6257]should accept valid cert rotation parameters", func() {
			kv := copyOriginalKv()
			patchKvCertConfig(kv.Name, certConfig)
		})

		It("[test_id:6258]should reject combining deprecated and new cert rotation parameters", func() {
			kv := copyOriginalKv()
			certConfig.CAOverlapInterval = &metav1.Duration{Duration: 8 * time.Hour}
			patchKvCertConfigExpectError(kv.Name, certConfig)
		})

		It("[test_id:6259]should reject CA expires before rotation", func() {
			kv := copyOriginalKv()
			certConfig.CA.Duration = &metav1.Duration{Duration: 14 * time.Hour}
			patchKvCertConfigExpectError(kv.Name, certConfig)
		})

		It("[test_id:6260]should reject Cert expires before rotation", func() {
			kv := copyOriginalKv()
			certConfig.Server.Duration = &metav1.Duration{Duration: 8 * time.Hour}
			patchKvCertConfigExpectError(kv.Name, certConfig)
		})

		It("[test_id:6261]should reject Cert rotates after CA expires", func() {
			kv := copyOriginalKv()
			certConfig.Server.Duration = &metav1.Duration{Duration: 48 * time.Hour}
			certConfig.Server.RenewBefore = &metav1.Duration{Duration: 36 * time.Hour}
			patchKvCertConfigExpectError(kv.Name, certConfig)
		})
	})

	Context("with VMExport feature gate toggled", func() {

		AfterEach(func() {
			tests.EnableFeatureGate(virtconfig.VMExportGate)
			testsuite.WaitExportProxyReady()
		})

		It("should delete and recreate virt-exportproxy", func() {
			testsuite.WaitExportProxyReady()
			tests.DisableFeatureGate(virtconfig.VMExportGate)

			Eventually(func() bool {
				_, err := virtClient.AppsV1().Deployments(originalKv.Namespace).Get(context.TODO(), "virt-exportproxy", metav1.GetOptions{})
				return errors.IsNotFound(err)
			}, time.Minute*5, time.Second*2).Should(BeTrue())
		})
	})

	Context("[Serial] Seccomp configuration", Serial, func() {

		Context("Kubevirt profile", func() {
			var nodeName string

			const expectedSeccompProfilePath = "/proc/1/root/var/lib/kubelet/seccomp/kubevirt/kubevirt.json"

			enableSeccompFeature := func() {
				//Disable feature first to simulate addition
				tests.DisableFeatureGate(virtconfig.KubevirtSeccompProfile)
				tests.EnableFeatureGate(virtconfig.KubevirtSeccompProfile)
			}

			disableSeccompFeature := func() {
				//Enable feature first to simulate removal
				tests.EnableFeatureGate(virtconfig.KubevirtSeccompProfile)
				tests.DisableFeatureGate(virtconfig.KubevirtSeccompProfile)
			}

			enableKubevirtProfile := func(enable bool) {
				nodeName = libnode.GetAllSchedulableNodes(virtClient).Items[0].Name

				By("Removing profile if present")
				_, err := tests.ExecuteCommandInVirtHandlerPod(nodeName, []string{"/usr/bin/rm", "-f", expectedSeccompProfilePath})
				Expect(err).NotTo(HaveOccurred())

				By(fmt.Sprintf("Configuring KubevirtSeccompProfile feature gate to %t", enable))
				if enable {
					enableSeccompFeature()
				} else {
					disableSeccompFeature()
				}

				vmProfile := &v1.VirtualMachineInstanceProfile{
					CustomProfile: &v1.CustomProfile{
						LocalhostProfile: pointer.String("kubevirt/kubevirt.json"),
					},
				}
				if !enable {
					vmProfile = nil

				}

				kv := util2.GetCurrentKv(virtClient)
				kv.Spec.Configuration.SeccompConfiguration = &v1.SeccompConfiguration{
					VirtualMachineInstanceProfile: vmProfile,
				}

				tests.UpdateKubeVirtConfigValueAndWait(kv.Spec.Configuration)
			}

			It("should install Kubevirt policy", func() {
				enableKubevirtProfile(true)

				By("Expecting to see the profile")
				Eventually(func() error {
					_, err = tests.ExecuteCommandInVirtHandlerPod(nodeName, []string{"/usr/bin/cat", expectedSeccompProfilePath})
					return err
				}, 1*time.Minute, 1*time.Second).Should(Not(HaveOccurred()))
			})

			It("should not install Kubevirt policy", func() {
				enableKubevirtProfile(false)

				By("Expecting to not see the profile")
				Consistently(func() error {
					_, err = tests.ExecuteCommandInVirtHandlerPod(nodeName, []string{"/usr/bin/cat", expectedSeccompProfilePath})
					return err
				}, 1*time.Minute, 1*time.Second).Should(MatchError(Or(ContainSubstring("No such file"), ContainSubstring("container not found"))))
				Expect(err).To(MatchError(ContainSubstring("No such file")))
			})
		})

		Context("VirtualMachineInstance Profile", func() {
			DescribeTable("with VirtualMachineInstance Profile set to", func(virtualMachineProfile *v1.VirtualMachineInstanceProfile, expectedProfile *k8sv1.SeccompProfile) {
				By("Configuring VirtualMachineInstance Profile")
				kv := util2.GetCurrentKv(virtClient)
				if kv.Spec.Configuration.SeccompConfiguration == nil {
					kv.Spec.Configuration.SeccompConfiguration = &v1.SeccompConfiguration{}
				}
				kv.Spec.Configuration.SeccompConfiguration.VirtualMachineInstanceProfile = virtualMachineProfile
				tests.UpdateKubeVirtConfigValueAndWait(kv.Spec.Configuration)

				By("Checking launcher seccomp policy")
				vmi := libvmi.NewCirros()
				vmi = tests.RunVMI(vmi, 60)
				fetchVMI := matcher.ThisVMI(vmi)
				psaRelatedErrorDetected := false
				err := wait.PollImmediate(time.Second, 30*time.Second, func() (done bool, err error) {
					vmi, err := fetchVMI()
					if err != nil {
						return done, err
					}

					if vmi.Status.Phase != v1.Pending {
						return true, nil
					}

					for _, condition := range vmi.Status.Conditions {
						if condition.Type == v1.VirtualMachineInstanceSynchronized {
							if condition.Status == k8sv1.ConditionFalse && strings.Contains(condition.Message, "needs a privileged namespace") {
								psaRelatedErrorDetected = true
								return true, nil
							}
						}
					}
					return
				})
				Expect(err).NotTo(HaveOccurred())
				// In case we are running on PSA cluster, the case were we don't specify seccomp will violate the policy.
				// Therefore the VMIs Pod will fail to be created and we can't check its configuration.
				// In that case the loop above needs to see that VMI contains PSA related error.
				// This is enough and we declare this test as passed.
				if psaRelatedErrorDetected && virtualMachineProfile == nil {
					return
				}
				Eventually(matcher.ThisVMI(vmi), 30*time.Second, time.Second).Should(BeInPhase(v1.Scheduled))

				pod, err := libvmi.GetPodByVirtualMachineInstance(vmi, vmi.Namespace)
				Expect(err).NotTo(HaveOccurred())
				var podProfile *k8sv1.SeccompProfile
				if pod.Spec.SecurityContext != nil {
					podProfile = pod.Spec.SecurityContext.SeccompProfile
				}

				Expect(podProfile).To(Equal(expectedProfile))
			},
				Entry("default should not set profile", nil, nil),
				Entry("custom should use localhost", &v1.VirtualMachineInstanceProfile{
					CustomProfile: &v1.CustomProfile{
						LocalhostProfile: pointer.String("kubevirt/kubevirt.json"),
					},
				},
					&k8sv1.SeccompProfile{Type: k8sv1.SeccompProfileTypeLocalhost, LocalhostProfile: pointer.String("kubevirt/kubevirt.json")}),
			)
		})
	})

	Context("[Serial] Deployment of common-instancetypes", Serial, func() {
		var labelSelector string

		BeforeEach(func() {
			labelSelector = labels.Set{
				v1.AppComponentLabel: v1.AppComponent,
				v1.ManagedByLabel:    v1.ManagedByLabelOperatorValue,
			}.String()
		})

		AfterEach(func() {
			tests.DisableFeatureGate(virtconfig.CommonInstancetypesDeploymentGate)
			testsuite.EnsureKubevirtReady()
		})

		expectResourcesToNotExist := func() {
			instancetypes, err := virtClient.VirtualMachineClusterInstancetype().List(context.Background(), metav1.ListOptions{LabelSelector: labelSelector})
			Expect(err).ToNot(HaveOccurred())
			Expect(instancetypes.Items).To(BeEmpty())

			preferences, err := virtClient.VirtualMachineClusterPreference().List(context.Background(), metav1.ListOptions{LabelSelector: labelSelector})
			Expect(err).ToNot(HaveOccurred())
			Expect(preferences.Items).To(BeEmpty())
		}

		expectResourcesToExist := func() {
			instancetypes, err := virtClient.VirtualMachineClusterInstancetype().List(context.Background(), metav1.ListOptions{LabelSelector: labelSelector})
			Expect(err).ToNot(HaveOccurred())
			Expect(instancetypes.Items).ToNot(BeEmpty())

			preferences, err := virtClient.VirtualMachineClusterPreference().List(context.Background(), metav1.ListOptions{LabelSelector: labelSelector})
			Expect(err).ToNot(HaveOccurred())
			Expect(preferences.Items).ToNot(BeEmpty())
		}

		It("Should not deploy common-instancetypes when feature gate is disabled", func() {
			expectResourcesToNotExist()
		})

		It("Should deploy common-instancetypes when feature gate is enabled", func() {
			tests.EnableFeatureGate(virtconfig.CommonInstancetypesDeploymentGate)
			testsuite.EnsureKubevirtReady()
			expectResourcesToExist()
		})

		It("Should remove common-instancetypes when feature gate is disabled", func() {
			expectResourcesToNotExist()

			tests.EnableFeatureGate(virtconfig.CommonInstancetypesDeploymentGate)
			testsuite.EnsureKubevirtReady()

			expectResourcesToExist()

			tests.DisableFeatureGate(virtconfig.CommonInstancetypesDeploymentGate)
			testsuite.EnsureKubevirtReady()

			expectResourcesToNotExist()
		})

		Context("Should take ownership", func() {
			const appComponent = "something"
			const managedBy = "someone"

			It("of instancetypes", func() {
				By("Getting instancetypes to be deployed by virt-operator")
				instancetypes, err := components.NewClusterInstancetypes()
				Expect(err).ToNot(HaveOccurred())
				Expect(instancetypes).ToNot(BeEmpty())

				By("Picking the first instancetype and changing its labels")
				instancetype := instancetypes[0]
				instancetype.Labels = map[string]string{
					v1.AppComponentLabel: appComponent,
					v1.ManagedByLabel:    managedBy,
				}

				By("Creating the instancetype")
				instancetype, err = virtClient.VirtualMachineClusterInstancetype().Create(context.Background(), instancetype, metav1.CreateOptions{})
				Expect(err).ToNot(HaveOccurred())
				Expect(instancetype.Labels).To(HaveKeyWithValue(v1.AppComponentLabel, appComponent))
				Expect(instancetype.Labels).To(HaveKeyWithValue(v1.ManagedByLabel, managedBy))

				By("Enabling the feature gate and waiting for KubeVirt to be ready")
				tests.EnableFeatureGate(virtconfig.CommonInstancetypesDeploymentGate)
				testsuite.EnsureKubevirtReady()

				By("Verifying virt-operator took ownership of the instancetype")
				instancetype, err = virtClient.VirtualMachineClusterInstancetype().Get(context.Background(), instancetype.Name, metav1.GetOptions{})
				Expect(err).ToNot(HaveOccurred())
				Expect(instancetype.Labels).To(HaveKeyWithValue(v1.AppComponentLabel, v1.AppComponent))
				Expect(instancetype.Labels).To(HaveKeyWithValue(v1.ManagedByLabel, v1.ManagedByLabelOperatorValue))
			})

			It("of preferences", func() {
				By("Getting preferences to be deployed by virt-operator")
				preferences, err := components.NewClusterPreferences()
				Expect(err).ToNot(HaveOccurred())
				Expect(preferences).ToNot(BeEmpty())

				By("Picking the first preference and changing its labels")
				preference := preferences[0]
				preference.Labels = map[string]string{
					v1.AppComponentLabel: appComponent,
					v1.ManagedByLabel:    managedBy,
				}

				By("Creating the preference")
				preference, err = virtClient.VirtualMachineClusterPreference().Create(context.Background(), preference, metav1.CreateOptions{})
				Expect(err).ToNot(HaveOccurred())
				Expect(preference.Labels).To(HaveKeyWithValue(v1.AppComponentLabel, appComponent))
				Expect(preference.Labels).To(HaveKeyWithValue(v1.ManagedByLabel, managedBy))

				By("Enabling the feature gate and waiting for KubeVirt to be ready")
				tests.EnableFeatureGate(virtconfig.CommonInstancetypesDeploymentGate)
				testsuite.EnsureKubevirtReady()

				By("Verifying virt-operator took ownership of the preference")
				preference, err = virtClient.VirtualMachineClusterPreference().Get(context.Background(), preference.Name, metav1.GetOptions{})
				Expect(err).ToNot(HaveOccurred())
				Expect(preference.Labels).To(HaveKeyWithValue(v1.AppComponentLabel, v1.AppComponent))
				Expect(preference.Labels).To(HaveKeyWithValue(v1.ManagedByLabel, v1.ManagedByLabelOperatorValue))
			})
		})

		Context("Should delete resources not in install strategy", func() {
			BeforeEach(func() {
				By("Enabling the feature gate and waiting for KubeVirt to be ready")
				tests.EnableFeatureGate(virtconfig.CommonInstancetypesDeploymentGate)
				testsuite.EnsureKubevirtReady()
			})

			It("with instancetypes", func() {
				instancetype := &v1beta1.VirtualMachineClusterInstancetype{
					ObjectMeta: metav1.ObjectMeta{
						GenerateName: "clusterinstancetype-",
						Labels: map[string]string{
							v1.AppComponentLabel: v1.AppComponent,
							v1.ManagedByLabel:    v1.ManagedByLabelOperatorValue,
						},
					},
				}

				By("Creating the instancetype")
				instancetype, err = virtClient.VirtualMachineClusterInstancetype().Create(context.Background(), instancetype, metav1.CreateOptions{})
				Expect(err).ToNot(HaveOccurred())

				By("Verifying virt-operator deleted the instancetype")
				Eventually(func(g Gomega) {
					_, err = virtClient.VirtualMachineClusterInstancetype().Get(context.Background(), instancetype.Name, metav1.GetOptions{})
					g.Expect(err).Should(HaveOccurred())
					g.Expect(errors.ReasonForError(err)).Should(Equal(metav1.StatusReasonNotFound))
				}, 1*time.Minute, 5*time.Second).Should(Succeed())
			})

			It("with preferences", func() {
				preference := &v1beta1.VirtualMachineClusterPreference{
					ObjectMeta: metav1.ObjectMeta{
						GenerateName: "clusterpreference-",
						Labels: map[string]string{
							v1.AppComponentLabel: v1.AppComponent,
							v1.ManagedByLabel:    v1.ManagedByLabelOperatorValue,
						},
					},
				}

				By("Creating the preference")
				preference, err = virtClient.VirtualMachineClusterPreference().Create(context.Background(), preference, metav1.CreateOptions{})
				Expect(err).ToNot(HaveOccurred())

				By("Verifying virt-operator deleted the preference")
				Eventually(func(g Gomega) {
					_, err = virtClient.VirtualMachineClusterPreference().Get(context.Background(), preference.Name, metav1.GetOptions{})
					g.Expect(err).Should(HaveOccurred())
					g.Expect(errors.ReasonForError(err)).Should(Equal(metav1.StatusReasonNotFound))
				}, 1*time.Minute, 5*time.Second).Should(Succeed())
			})
		})

		Context("Should revert changes", func() {
			const keyTest = "test"
			const valModified = "modified"
			const cpu = uint32(1024)

			var preferredTopology = v1beta1.PreferThreads

			It("to instancetypes", func() {
				By("Enabling the feature gate and waiting for KubeVirt to be ready")
				tests.EnableFeatureGate(virtconfig.CommonInstancetypesDeploymentGate)
				testsuite.EnsureKubevirtReady()

				By("Getting the deployed instancetypes")
				instancetypes, err := virtClient.VirtualMachineClusterInstancetype().List(context.Background(), metav1.ListOptions{LabelSelector: labelSelector})
				Expect(err).ToNot(HaveOccurred())
				Expect(instancetypes.Items).ToNot(BeEmpty())

				By("Modifying an instancetype")
				originalInstancetype := instancetypes.Items[0].DeepCopy()
				instancetype := originalInstancetype.DeepCopy()
				instancetype.Annotations[keyTest] = valModified
				instancetype.Labels[keyTest] = valModified
				instancetype.Spec = v1beta1.VirtualMachineInstancetypeSpec{
					CPU: v1beta1.CPUInstancetype{
						Guest: cpu,
					},
				}

				instancetype, err = virtClient.VirtualMachineClusterInstancetype().Update(context.Background(), instancetype, metav1.UpdateOptions{})
				Expect(err).ToNot(HaveOccurred())
				Expect(instancetype.Annotations).To(HaveKeyWithValue(keyTest, valModified))
				Expect(instancetype.Labels).To(HaveKeyWithValue(keyTest, valModified))
				Expect(instancetype.Spec.CPU.Guest).To(Equal(cpu))

				By("Verifying virt-operator reverts the changes")
				Eventually(func(g Gomega) {
					instancetype, err := virtClient.VirtualMachineClusterInstancetype().Get(context.Background(), instancetype.Name, metav1.GetOptions{})
					g.Expect(err).ToNot(HaveOccurred())
					g.Expect(instancetype.Annotations).To(Equal(originalInstancetype.Annotations))
					g.Expect(instancetype.Labels).To(Equal(originalInstancetype.Labels))
					g.Expect(instancetype.Spec).To(Equal(originalInstancetype.Spec))
				}, 1*time.Minute, 5*time.Second).Should(Succeed())
			})

			It("to preferences", func() {
				By("Enabling the feature gate and waiting for KubeVirt to be ready")
				tests.EnableFeatureGate(virtconfig.CommonInstancetypesDeploymentGate)
				testsuite.EnsureKubevirtReady()

				By("Getting the deployed preferencess")
				preferences, err := virtClient.VirtualMachineClusterPreference().List(context.Background(), metav1.ListOptions{LabelSelector: labelSelector})
				Expect(err).ToNot(HaveOccurred())
				Expect(preferences.Items).ToNot(BeEmpty())

				By("Modifying a preference")
				originalPreference := preferences.Items[0].DeepCopy()
				preference := originalPreference.DeepCopy()
				preference.Annotations[keyTest] = valModified
				preference.Labels[keyTest] = valModified
				preference.Spec = v1beta1.VirtualMachinePreferenceSpec{
					CPU: &v1beta1.CPUPreferences{
						PreferredCPUTopology: &preferredTopology,
					},
				}

				preference, err = virtClient.VirtualMachineClusterPreference().Update(context.Background(), preference, metav1.UpdateOptions{})
				Expect(err).ToNot(HaveOccurred())
				Expect(preference.Annotations).To(HaveKeyWithValue(keyTest, valModified))
				Expect(preference.Labels).To(HaveKeyWithValue(keyTest, valModified))
				Expect(preference.Spec.CPU).ToNot(BeNil())
				Expect(preference.Spec.CPU.PreferredCPUTopology).ToNot(BeNil())
				Expect(*preference.Spec.CPU.PreferredCPUTopology).To(Equal(preferredTopology))

				By("Verifying virt-operator reverts the changes")
				Eventually(func(g Gomega) {
					preference, err := virtClient.VirtualMachineClusterPreference().Get(context.Background(), preference.Name, metav1.GetOptions{})
					g.Expect(err).ToNot(HaveOccurred())
					g.Expect(preference.Annotations).To(Equal(originalPreference.Annotations))
					g.Expect(preference.Labels).To(Equal(originalPreference.Labels))
					g.Expect(preference.Spec).To(Equal(originalPreference.Spec))
				}, 1*time.Minute, 5*time.Second).Should(Succeed())
			})
		})
	})
})

func patchCRD(orig *extv1.CustomResourceDefinition, modified *extv1.CustomResourceDefinition) []byte {
	origCRDByte, err := json.Marshal(orig)
	ExpectWithOffset(1, err).ToNot(HaveOccurred())
	crdByte, err := json.Marshal(modified)
	ExpectWithOffset(1, err).ToNot(HaveOccurred())
	patch, err := jsonpatch.CreateMergePatch(origCRDByte, crdByte)
	ExpectWithOffset(1, err).ToNot(HaveOccurred())
	return patch
}

// verifyOperatorWebhookCertificate can be used when inside tests doing reinstalls of kubevirt, to ensure that virt-operator already got the new certificate.
// This is necessary, since it can take up to a minute to get the fresh certificates when secrets are updated.
func verifyOperatorWebhookCertificate() {
	caBundle, _ := tests.GetBundleFromConfigMap(components.KubeVirtCASecretName)
	certPool := x509.NewCertPool()
	certPool.AppendCertsFromPEM(caBundle)
	// ensure that the state is fully restored before each test
	Eventually(func() error {
		currentCert, err := tests.GetCertsForPods(fmt.Sprintf("%s=%s", v1.AppLabel, "virt-operator"), flags.KubeVirtInstallNamespace, "8444")
		Expect(err).ToNot(HaveOccurred())
		crt, err := x509.ParseCertificate(currentCert[0])
		Expect(err).ToNot(HaveOccurred())
		_, err = crt.Verify(x509.VerifyOptions{
			Roots: certPool,
		})
		return err
	}, 90*time.Second, 1*time.Second).Should(Not(HaveOccurred()), "bundle and certificate are still not in sync after 90 seconds")
	// we got the first pod with the new certificate, now let's wait until every pod sees it
	// this can take additional time since nodes are not synchronizing at the same moment
	tests.EnsurePodsCertIsSynced(fmt.Sprintf("%s=%s", v1.AppLabel, "virt-operator"), flags.KubeVirtInstallNamespace, "8444")
}

func getUpstreamReleaseAssetURL(tag string, assetName string) string {
	client := github.NewClient(nil)

	var err error
	var release *github.RepositoryRelease

	Eventually(func() error {
		release, _, err = client.Repositories.GetReleaseByTag(context.Background(), "kubevirt", "kubevirt", tag)

		return err
	}, 10*time.Second, 1*time.Second).ShouldNot(HaveOccurred())

	for _, asset := range release.Assets {
		if asset.GetName() == assetName {
			return asset.GetBrowserDownloadURL()
		}
	}

	Fail(fmt.Sprintf("Asset %s not found in release %s of kubevirt upstream repo", assetName, tag))
	return ""
}

func detectLatestUpstreamOfficialTag() (string, error) {
	client := github.NewClient(nil)

	var err error
	var releases []*github.RepositoryRelease

	Eventually(func() error {
		releases, _, err = client.Repositories.ListReleases(context.Background(), "kubevirt", "kubevirt", &github.ListOptions{PerPage: 10000})

		return err
	}, 10*time.Second, 1*time.Second).ShouldNot(HaveOccurred())

	var vs []*semver.Version

	for _, release := range releases {
		if *release.Draft ||
			*release.Prerelease ||
			len(release.Assets) == 0 {

			continue
		}
		v, err := semver.NewVersion(*release.TagName)
		if err != nil {
			panic(err)
		}
		vs = append(vs, v)
	}

	if len(vs) == 0 {
		return "", fmt.Errorf("no kubevirt releases found")
	}

	// decending order from most recent.
	sort.Sort(sort.Reverse(semver.Collection(vs)))

	// most recent tag
	tag := fmt.Sprintf("v%v", vs[0])

	// tag hint gives us information about the most recent tag in the current branch
	// this is executing in. We want to make sure we are using the previous most
	// recent official release from the branch we're in if possible. Note that this is
	// all best effort. If a tag hint can't be detected, we move on with the most
	// recent release from master.
	tagHint := getTagHint()
	hint, err := semver.NewVersion(tagHint)

	if tagHint != "" && err == nil {
		for _, v := range vs {
			if v.LessThan(hint) || v.Equal(hint) {
				tag = fmt.Sprintf("v%v", v)
				By(fmt.Sprintf("Choosing tag %s influenced by tag hint %s", tag, tagHint))
				break
			}
		}
	}

	By(fmt.Sprintf("By detecting latest upstream official tag %s for current branch", tag))
	return tag, nil
}

func getTagHint() string {
	//git describe --tags --abbrev=0 "$(git rev-parse HEAD)"
	cmd := exec.Command("git", "rev-parse", "HEAD")
	cmdOutput, err := cmd.Output()
	if err != nil {
		return ""
	}

	cmd = exec.Command("git", "describe", "--tags", "--abbrev=0", strings.TrimSpace(string(cmdOutput)))
	cmdOutput, err = cmd.Output()
	if err != nil {
		return ""
	}

	return strings.TrimSpace(strings.Split(string(cmdOutput), "-rc")[0])
}

func atLeastOnePendingPodExistInDeployment(virtClient kubecli.KubevirtClient, deploymentName string) bool {
	pods, err := virtClient.CoreV1().Pods(flags.KubeVirtInstallNamespace).List(context.Background(),
		metav1.ListOptions{
			LabelSelector: fmt.Sprintf("kubevirt.io=%s", deploymentName),
			FieldSelector: fields.ParseSelectorOrDie("status.phase=Pending").String(),
		})
	Expect(err).ShouldNot(HaveOccurred())
	if len(pods.Items) == 0 {
		return false
	}
	return true
}

func nodeSelectorExistInDeployment(virtClient kubecli.KubevirtClient, deploymentName string, labelKey string, labelValue string) bool {
	deployment, err := virtClient.AppsV1().Deployments(flags.KubeVirtInstallNamespace).Get(context.Background(), deploymentName, metav1.GetOptions{})
	Expect(err).ToNot(HaveOccurred())
	if deployment.Spec.Template.Spec.NodeSelector == nil || deployment.Spec.Template.Spec.NodeSelector[labelKey] != labelValue {
		return false
	}
	return true
}
