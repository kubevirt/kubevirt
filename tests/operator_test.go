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

package tests_test

import (
	"bytes"
	"compress/gzip"
	"context"
	"crypto/x509"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"

	"kubevirt.io/kubevirt/pkg/virt-operator/resource/apply"
	. "kubevirt.io/kubevirt/tests/framework/matcher"
	util2 "kubevirt.io/kubevirt/tests/util"

	"regexp"
	"strings"
	"time"

	jsonpatch "github.com/evanphx/json-patch"
	. "github.com/onsi/ginkgo"
	"github.com/onsi/ginkgo/extensions/table"
	. "github.com/onsi/gomega"
	v12 "k8s.io/api/apps/v1"
	k8sv1 "k8s.io/api/core/v1"
	extv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	extclient "k8s.io/apiextensions-apiserver/pkg/client/clientset/clientset"
	"k8s.io/apimachinery/pkg/api/equality"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/rand"
	aggregatorclient "k8s.io/kube-aggregator/pkg/client/clientset_generated/clientset"
	"k8s.io/utils/pointer"

	v1 "kubevirt.io/api/core/v1"
	snapshotv1 "kubevirt.io/api/snapshot/v1alpha1"
	"kubevirt.io/client-go/kubecli"
	"kubevirt.io/client-go/log"
	cdiv1 "kubevirt.io/containerized-data-importer-api/pkg/apis/core/v1beta1"
	sdkapi "kubevirt.io/controller-lifecycle-operator-sdk/pkg/sdk/api"
	"kubevirt.io/kubevirt/pkg/controller"
	"kubevirt.io/kubevirt/pkg/virt-operator/resource/generate/components"
	"kubevirt.io/kubevirt/pkg/virt-operator/util"
	"kubevirt.io/kubevirt/tests"
	"kubevirt.io/kubevirt/tests/console"
	cd "kubevirt.io/kubevirt/tests/containerdisk"
	"kubevirt.io/kubevirt/tests/flags"
	"kubevirt.io/kubevirt/tests/libnet"
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

var _ = Describe("[Serial][sig-operator]Operator", func() {
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
		parseDaemonset                         func(string) (*v12.DaemonSet, string, string, string, string)
		parseImage                             func(string, string) (string, string, string)
		parseDeployment                        func(string) (*v12.Deployment, string, string, string, string)
		parseOperatorImage                     func() (*v12.Deployment, string, string, string, string)
		patchOperator                          func(*string, *string) bool
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
	)

	tests.BeforeAll(func() {
		virtClient, err = kubecli.GetKubevirtClient()
		util2.PanicOnError(err)
		config, err := kubecli.GetConfig()
		util2.PanicOnError(err)
		aggregatorClient = aggregatorclient.NewForConfigOrDie(config)

		k8sClient = tests.GetK8sCmdClient()

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

			Eventually(func() error {
				for _, deployment := range []string{"virt-api", "virt-controller"} {
					_, err := virtClient.AppsV1().Deployments(flags.KubeVirtInstallNamespace).Get(context.Background(), deployment, metav1.GetOptions{})
					if err != nil && !errors.IsNotFound(err) {
						return err
					}
				}
				return nil
			}, 15*time.Second, 1*time.Second).ShouldNot(HaveOccurred())
		}

		allPodsAreTerminated = func(kv *v1.KubeVirt) {
			Eventually(func() error {
				pods, err := virtClient.CoreV1().Pods(kv.Namespace).List(context.Background(), metav1.ListOptions{LabelSelector: "kubevirt.io"})
				if err != nil {
					return err
				}

				for _, pod := range pods.Items {
					managed, ok := pod.Labels[v1.ManagedByLabel]
					if !ok || managed != v1.ManagedByLabelOperatorValue {
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
					managed, ok := pod.Labels[v1.ManagedByLabel]
					if !ok || managed != v1.ManagedByLabelOperatorValue {
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
			Eventually(func() error {
				kv, err := virtClient.KubeVirt(kv.Namespace).Get(kv.Name, &metav1.GetOptions{})
				if err != nil {
					return err
				}

				available := false
				progressing := false
				degraded := false
				for _, condition := range kv.Status.Conditions {
					if condition.Type == v1.KubeVirtConditionAvailable && condition.Status == k8sv1.ConditionTrue {
						available = true
					} else if condition.Type == v1.KubeVirtConditionProgressing && condition.Status == k8sv1.ConditionTrue {
						progressing = true
					} else if condition.Type == v1.KubeVirtConditionDegraded && condition.Status == k8sv1.ConditionTrue {
						degraded = true
					}
				}

				if !available || !progressing || !degraded {
					return fmt.Errorf("Waiting for conditions to indicate update (conditions: %+v)", kv.Status.Conditions)
				}
				return nil
			}, 120*time.Second, 1*time.Second).ShouldNot(HaveOccurred())

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
					return fmt.Errorf("Waiting for conditions to indicate deployment (conditions: %+v)", kv.Status.Conditions)
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

		parseDaemonset = func(name string) (daemonSet *v12.DaemonSet, image, registry, imagePrefix, version string) {
			var err error
			daemonSet, err = virtClient.AppsV1().DaemonSets(flags.KubeVirtInstallNamespace).Get(context.Background(), name, metav1.GetOptions{})
			Expect(err).ToNot(HaveOccurred())
			image = daemonSet.Spec.Template.Spec.Containers[0].Image
			imageRegEx := regexp.MustCompile(fmt.Sprintf("%s%s%s", `^(.*)/(.*)`, name, `([@:].*)?$`))
			matches := imageRegEx.FindAllStringSubmatch(image, 1)
			Expect(len(matches)).To(Equal(1))
			Expect(len(matches[0])).To(Equal(4))
			registry = matches[0][1]
			imagePrefix = matches[0][2]
			version = matches[0][3]
			return
		}

		parseImage = func(name, image string) (registry, imagePrefix, version string) {
			imageRegEx := regexp.MustCompile(fmt.Sprintf("%s%s%s", `^(.*)/(.*)`, name, `([@:].*)?$`))
			matches := imageRegEx.FindAllStringSubmatch(image, 1)
			Expect(len(matches)).To(Equal(1))
			Expect(len(matches[0])).To(Equal(4))
			registry = matches[0][1]
			imagePrefix = matches[0][2]
			version = matches[0][3]
			return
		}

		parseDeployment = func(name string) (deployment *v12.Deployment, image, registry, imagePrefix, version string) {
			var err error
			deployment, err = virtClient.AppsV1().Deployments(flags.KubeVirtInstallNamespace).Get(context.Background(), name, metav1.GetOptions{})
			Expect(err).ToNot(HaveOccurred())
			image = deployment.Spec.Template.Spec.Containers[0].Image
			registry, imagePrefix, version = parseImage(name, image)
			return
		}

		parseOperatorImage = func() (operator *v12.Deployment, image, registry, imagePrefix, version string) {
			return parseDeployment("virt-operator")
		}

		patchOperator = func(imagePrefix, version *string) bool {

			modified := true

			Eventually(func() error {

				operator, oldImage, registry, oldPrefix, oldVersion := parseOperatorImage()
				if imagePrefix == nil {
					// keep old prefix
					imagePrefix = &oldPrefix
				}
				if version == nil {
					// keep old version
					version = &oldVersion
				} else {
					newVersion := components.AddVersionSeparatorPrefix(*version)
					version = &newVersion
				}
				newImage := fmt.Sprintf("%s/%svirt-operator%s", registry, *imagePrefix, *version)

				if oldImage == newImage {
					modified = false
					return nil
				}

				operator.Spec.Template.Spec.Containers[0].Image = newImage
				for idx, env := range operator.Spec.Template.Spec.Containers[0].Env {
					if env.Name == util.OperatorImageEnvName {
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
		Expect(strings.HasPrefix(version, "@")).To(BeTrue())
		originalOperatorVersion = strings.TrimPrefix(version, "@")

		if tests.HasDataVolumeCRD() {
			cdiList, err := virtClient.CdiClient().CdiV1beta1().CDIs().List(context.Background(), metav1.ListOptions{})
			Expect(err).ToNot(HaveOccurred())
			Expect(len(cdiList.Items)).To(Equal(1))

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

				tests.CreateConfigMap(configMapName, config_data)
				tests.CreateSecret(secretName, secret_data)

				tests.AddUserData(vmi, "cloud-init", "#!/bin/bash\necho 'hello'\n")
				tests.AddConfigMapDisk(vmi, configMapName, configMapName)
				tests.AddSecretDisk(vmi, secretName, secretName)
				tests.AddServiceAccountDisk(vmi, "default")
				// In case there are no existing labels add labels to add some data to the downwardAPI disk
				if vmi.ObjectMeta.Labels == nil {
					vmi.ObjectMeta.Labels = map[string]string{"downwardTestLabelKey": "downwardTestLabelVal"}
				}
				tests.AddLabelDownwardAPIVolume(vmi, downwardAPIName)

				vmis = append(vmis, vmi)
			}

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
				vmi, err := virtClient.VirtualMachineInstance(util2.NamespaceTestDefault).Create(vmi)
				Expect(err).To(BeNil(), "Create VMI successfully")
				tests.WaitForSuccessfulVMIStart(vmi)
			}
		}

		deleteAllVMIs = func(vmis []*v1.VirtualMachineInstance) {
			for _, vmi := range vmis {
				err := virtClient.VirtualMachineInstance(vmi.Namespace).Delete(vmi.Name, &metav1.DeleteOptions{})
				Expect(err).To(BeNil(), "Delete VMI successfully")
			}
		}

		verifyVMIsEvicted = func(vmis []*v1.VirtualMachineInstance) {

			Eventually(func() error {
				for _, vmi := range vmis {
					vmi, err := virtClient.VirtualMachineInstance(vmi.Namespace).Get(vmi.Name, &metav1.GetOptions{})
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
					vmi, err := virtClient.VirtualMachineInstance(vmi.Namespace).Get(vmi.Name, &metav1.GetOptions{})
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
				migrationList, err := virtClient.VirtualMachineInstanceMigration(util2.NamespaceTestDefault).List(&metav1.ListOptions{})
				Expect(err).To(BeNil(), "retrieving migrations")
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
					err = ioutil.WriteFile(snapshotYamlFile, []byte(snapshotYaml), 0644)
					Expect(err).ToNot(HaveOccurred())

					restoreYamlFile := filepath.Join(workDir, fmt.Sprintf("%s.yaml", restoreName))
					err = ioutil.WriteFile(restoreYamlFile, []byte(restoreYaml), 0644)
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

		generatePreviousVersionVmYamls = func(previousImageRegistry string, previousImageTag string) {
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
            memory: 64M
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
`, version, version, version, i, version, i, previousImageRegistry, cd.ContainerDiskCirros, previousImageTag)

				yamlFile := filepath.Join(workDir, fmt.Sprintf("vm-%s.yaml", version))
				err = ioutil.WriteFile(yamlFile, []byte(vmYaml), 0644)
				Expect(err).ToNot(HaveOccurred())

				vmYamls[version] = &vmYamlDefinition{
					apiVersion:    version,
					vmName:        "vm-" + version,
					generatedYaml: vmYaml,
					yamlFile:      yamlFile,
				}
			}

		}
	})

	BeforeEach(func() {
		tests.BeforeTestCleanup()

		workDir, err = ioutil.TempDir("", tests.TempDirPrefix+"-")
		Expect(err).ToNot(HaveOccurred())

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

		if workDir != "" {
			err = os.RemoveAll(workDir)
			workDir = ""
			Expect(err).ToNot(HaveOccurred())
		}

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
	})

	It("[test_id:1746]should have created and available condition", func() {
		kv := util2.GetCurrentKv(virtClient)

		By("verifying that created and available condition is present")
		waitForKv(kv)
	})

	Describe("should reconcile components", func() {

		deploymentName := "virt-controller"
		daemonSetName := "virt-handler"
		envVarDeploymentKeyToUpdate := "USER_ADDED_ENV"

		crdName := "virtualmachines.kubevirt.io"
		shortNameAdded := "new"

		table.DescribeTable("checking updating resource is reverted to original state for ", func(changeResource func(), getResource func() runtime.Object, compareResource func() bool) {
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

			table.Entry("[test_id:6254] deployments",

				func() {

					vc, err := virtClient.AppsV1().Deployments(originalKv.Namespace).Get(context.Background(), deploymentName, metav1.GetOptions{})
					Expect(err).ToNot(HaveOccurred())

					vc.Spec.Template.Spec.Containers[0].Env = []k8sv1.EnvVar{
						{
							Name:  envVarDeploymentKeyToUpdate,
							Value: "value",
						},
					}

					vc, err = virtClient.AppsV1().Deployments(originalKv.Namespace).Update(context.Background(), vc, metav1.UpdateOptions{})
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

			table.Entry("[test_id:6255] customresourcedefinitions",
				func() {
					vmcrd, err := virtClient.ExtensionsClient().ApiextensionsV1().CustomResourceDefinitions().Get(context.Background(), crdName, metav1.GetOptions{})
					Expect(err).ToNot(HaveOccurred())

					vmcrd.Spec.Names.ShortNames = append(vmcrd.Spec.Names.ShortNames, shortNameAdded)

					vmcrd, err = virtClient.ExtensionsClient().ApiextensionsV1().CustomResourceDefinitions().Update(context.Background(), vmcrd, metav1.UpdateOptions{})
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
			table.Entry("[test_id:6256] poddisruptionbudgets",
				func() {
					pdb, err := virtClient.PolicyV1beta1().PodDisruptionBudgets(originalKv.Namespace).Get(context.Background(), "virt-controller-pdb", metav1.GetOptions{})
					Expect(err).ToNot(HaveOccurred())

					pdb.Spec.Selector.MatchLabels = map[string]string{
						"kubevirt.io": "dne",
					}

					pdb, err = virtClient.PolicyV1beta1().PodDisruptionBudgets(originalKv.Namespace).Update(context.Background(), pdb, metav1.UpdateOptions{})
					Expect(err).ToNot(HaveOccurred())
					Expect(pdb.Spec.Selector.MatchLabels["kubevirt.io"]).To(Equal("dne"))
				},

				func() runtime.Object {
					pdb, err := virtClient.PolicyV1beta1().PodDisruptionBudgets(originalKv.Namespace).Get(context.Background(), "virt-controller-pdb", metav1.GetOptions{})
					Expect(err).ToNot(HaveOccurred())
					return pdb
				},

				func() bool {
					pdb, err := virtClient.PolicyV1beta1().PodDisruptionBudgets(originalKv.Namespace).Get(context.Background(), "virt-controller-pdb", metav1.GetOptions{})
					Expect(err).ToNot(HaveOccurred())

					return pdb.Spec.Selector.MatchLabels["kubevirt.io"] != "dne"
				}),
			table.Entry("[test_id:6308] daemonsets",
				func() {
					vc, err := virtClient.AppsV1().DaemonSets(originalKv.Namespace).Get(context.Background(), daemonSetName, metav1.GetOptions{})
					Expect(err).ToNot(HaveOccurred())

					vc.Spec.Template.Spec.Containers[0].Env = []k8sv1.EnvVar{
						{
							Name:  envVarDeploymentKeyToUpdate,
							Value: "value",
						},
					}

					vc, err = virtClient.AppsV1().DaemonSets(originalKv.Namespace).Update(context.Background(), vc, metav1.UpdateOptions{})
					Expect(err).ToNot(HaveOccurred())
					Expect(vc.Spec.Template.Spec.Containers[0].Env[0].Name).To(Equal(envVarDeploymentKeyToUpdate))
				},

				func() runtime.Object {
					vc, err := virtClient.AppsV1().DaemonSets(originalKv.Namespace).Get(context.Background(), daemonSetName, metav1.GetOptions{})
					Expect(err).ToNot(HaveOccurred())
					return vc
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
			vmi, err = virtClient.VirtualMachineInstance(util2.NamespaceTestDefault).Create(vmi)
			Expect(err).To(BeNil())
			tests.WaitForSuccessfulVMIStart(vmi)

			By("getting virt-launcher")
			uid := vmi.GetObjectMeta().GetUID()
			labelSelector := fmt.Sprintf(v1.CreatedByLabel + "=" + string(uid))
			pods, err := virtClient.CoreV1().Pods(util2.NamespaceTestDefault).List(context.Background(), metav1.ListOptions{LabelSelector: labelSelector})
			Expect(err).ToNot(HaveOccurred(), "Should list pods")
			Expect(len(pods.Items)).To(Equal(1))
			Expect(usesSha(pods.Items[0].Spec.Containers[0].Image)).To(BeTrue(), "launcher pod should use shasum")

		})
	})

	Describe("[test_id:4744]should apply component customization", func() {

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

			By("Test that patch was applied to deployment")
			Eventually(func() string {
				vc, err := virtClient.AppsV1().Deployments(originalKv.Namespace).Get(context.Background(), "virt-controller", metav1.GetOptions{})
				Expect(err).ToNot(HaveOccurred())

				return vc.Spec.Template.ObjectMeta.Annotations[annotationPatchKey]
			}, 60*time.Second, 5*time.Second).Should(Equal(annotationPatchValue))

			Consistently(func() string {
				vc, err := virtClient.AppsV1().Deployments(originalKv.Namespace).Get(context.Background(), "virt-controller", metav1.GetOptions{})
				Expect(err).ToNot(HaveOccurred())

				return vc.Spec.Template.ObjectMeta.Annotations[annotationPatchKey]
			}, 30*time.Second, 5*time.Second).Should(Equal(annotationPatchValue))

			By("Deleting patch from KubeVirt object")
			kv, err = virtClient.KubeVirt(originalKv.Namespace).Get(originalKv.Name, &metav1.GetOptions{})
			Expect(err).ToNot(HaveOccurred())

			By("Check that KubeVirt CR generation does not get updated when applying patch")
			Expect(kv.GetGeneration()).To(Equal(generation))

			kv.Spec.CustomizeComponents = v1.CustomizeComponents{}
			kv, err = virtClient.KubeVirt(originalKv.Namespace).Update(kv)
			Expect(err).ToNot(HaveOccurred())

			By("Test that patch was removed from deployment")
			Eventually(func() string {
				vc, err := virtClient.AppsV1().Deployments(originalKv.Namespace).Get(context.Background(), "virt-controller", metav1.GetOptions{})
				Expect(err).ToNot(HaveOccurred())

				return vc.Spec.Template.ObjectMeta.Annotations[annotationPatchKey]
			}, 60*time.Second, 5*time.Second).Should(Equal(""))
		})
	})

	Describe("[rfe_id:2291][crit:high][vendor:cnv-qe@redhat.com][level:component]should update kubevirt", func() {
		runStrategyHalted := v1.RunStrategyHalted

		// This test is installing a previous release of KubeVirt
		// running a VM/VMI using that previous release
		// Updating KubeVirt to the target tested code
		// Ensuring VM/VMI is still operational after the update from previous release.
		It("[release-blocker][sig-compute][test_id:3145]from previous release to target tested release", func() {
			if !tests.HasCDI() {
				Skip("Skip Update test when CDI is not present")
			}

			migratableVMIs := generateMigratableVMIs(2)
			launcherSha := getVirtLauncherSha()
			Expect(launcherSha).ToNot(Equal(""))

			previousImageTag := flags.PreviousReleaseTag
			previousImageRegistry := flags.PreviousReleaseRegistry
			if previousImageTag == "" {
				previousImageTag, err = tests.DetectLatestUpstreamOfficialTag()
				Expect(err).ToNot(HaveOccurred())
				By(fmt.Sprintf("By Using detected tag %s", previousImageTag))
			} else {
				By(fmt.Sprintf("By Using user defined tag %s", previousImageTag))
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

			// Install Previous Release of KubeVirt
			By(fmt.Sprintf("Creating KubeVirt Object with Previous Release: %s using registry %s", previousImageTag, previousImageRegistry))
			kv := copyOriginalKv()
			kv.Name = "kubevirt-release-install"
			kv.Spec.ImageTag = previousImageTag
			kv.Spec.ImageRegistry = previousImageRegistry
			kv.Spec.WorkloadUpdateStrategy.WorkloadUpdateMethods = []v1.WorkloadUpdateMethod{v1.WorkloadUpdateMethodLiveMigrate}
			createKv(kv)

			// Wait for Previous Release to come online
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
			oldClientCacheDir := workDir + "/oldclient"
			err = os.MkdirAll(oldClientCacheDir, 0755)
			Expect(err).ToNot(HaveOccurred())
			newClientCacheDir := workDir + "/newclient"
			err = os.MkdirAll(newClientCacheDir, 0755)
			Expect(err).ToNot(HaveOccurred())

			// Create VM on previous release using a specific API.
			// NOTE: we are testing with yaml here and explicitly _NOT_ generating
			// this vm using the latest api code. We want to guarrantee there are no
			// surprises when it comes to backwards compatibility with previous
			// virt apis.  As we progress our api from v1alpha3 -> v1 there
			// needs to be a VM created for every api. This is how we will ensure
			// our api remains upgradable and supportable from previous release.

			generatePreviousVersionVmYamls(previousImageRegistry, previousImageTag)
			generatePreviousVersionVmsnapshotYamls()
			for _, vmYaml := range vmYamls {
				By(fmt.Sprintf("Creating VM with %s api", vmYaml.vmName))
				// NOTE: using kubectl to post yaml directly
				_, _, err = tests.RunCommand(k8sClient, "create", "-f", vmYaml.yamlFile, "--cache-dir", oldClientCacheDir)
				Expect(err).ToNot(HaveOccurred())

				for _, vmSnapshot := range vmYaml.vmSnapshots {
					By(fmt.Sprintf("Creating VM snapshot %s for vm %s", vmSnapshot.vmSnapshotName, vmYaml.vmName))
					_, _, err = tests.RunCommand(k8sClient, "create", "-f", vmSnapshot.yamlFile, "--cache-dir", oldClientCacheDir)
					Expect(err).ToNot(HaveOccurred())
				}

				// Use Current virtctl to start VM
				// NOTE: we are using virtctl explicitly here because we want to start the VM
				// using the subresource endpoint in the same way virtctl performs this.
				By("Starting VM with virtctl")
				startCommand := tests.NewRepeatableVirtctlCommand("start", "--namespace", util2.NamespaceTestDefault, vmYaml.vmName)
				Expect(startCommand()).To(Succeed())

				By(fmt.Sprintf("Waiting for VM with %s api to become ready", vmYaml.apiVersion))

				Eventually(func() bool {
					virtualMachine, err := virtClient.VirtualMachine(util2.NamespaceTestDefault).Get(vmYaml.vmName, &metav1.GetOptions{})
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
			By("Updating KubeVirtObject With Current Tag")
			patchKvVersionAndRegistry(kv.Name, curVersion, curRegistry)

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
					virtualMachine, err := virtClient.VirtualMachine(util2.NamespaceTestDefault).Get(vmYaml.vmName, &metav1.GetOptions{})
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
						vmSnapshot, err := virtClient.VirtualMachineSnapshot(util2.NamespaceTestDefault).Get(context.Background(), snapshot.vmSnapshotName, metav1.GetOptions{})
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
					vmi, err := virtClient.VirtualMachineInstance(util2.NamespaceTestDefault).Get(vmYaml.vmName, &metav1.GetOptions{})
					Expect(err).ToNot(HaveOccurred())
					if err := libnet.WithIPv6(console.LoginToCirros)(vmi); err != nil {
						return err
					}
					return nil
				}, 60*time.Second, 1*time.Second).Should(BeNil())

				By("Stopping VM with virtctl")
				stopFn := tests.NewRepeatableVirtctlCommand("stop", "--namespace", util2.NamespaceTestDefault, vmYaml.vmName)
				Eventually(func() error {
					return stopFn()
				}, 30*time.Second, 1*time.Second).Should(BeNil())

				By("Waiting for VMI to stop")
				Eventually(func() bool {
					_, err := virtClient.VirtualMachineInstance(util2.NamespaceTestDefault).Get(vmYaml.vmName, &metav1.GetOptions{})
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
					vm, err := virtClient.VirtualMachine(util2.NamespaceTestDefault).Get(vmYaml.vmName, &metav1.GetOptions{})
					if err != nil {
						return err
					}

					// by making a change to the VM, we ensure that writing the object is possible.
					// This ensures VMs created previously before the update are still compatible with our validation webhooks
					vm.Annotations["some-annotation"] = "some-val"
					annotationBytes, err := json.Marshal(vm.Annotations)
					Expect(err).ToNot(HaveOccurred())

					ops := fmt.Sprintf(`[{ "op": "add", "path": "/metadata/annotations", "value": %s }]`, string(annotationBytes))
					_, err = virtClient.VirtualMachine(vm.Namespace).Patch(vm.Name, types.JSONPatchType, []byte(ops), &metav1.PatchOptions{})
					return err
				}, 10*time.Second, 1*time.Second).ShouldNot(HaveOccurred())

				// change run stategy to halted to be able to restore the vm
				Eventually(func() bool {
					vm, err := virtClient.VirtualMachine(util2.NamespaceTestDefault).Get(vmYaml.vmName, &metav1.GetOptions{})
					if err != nil {
						return false
					}
					vm.Spec.RunStrategy = &runStrategyHalted
					_, err = virtClient.VirtualMachine(util2.NamespaceTestDefault).Update(vm)
					if err != nil {
						return false
					}
					updatedVM, err := virtClient.VirtualMachine(util2.NamespaceTestDefault).Get(vmYaml.vmName, &metav1.GetOptions{})
					Expect(err).ToNot(HaveOccurred())
					return updatedVM.Spec.Running == nil && updatedVM.Spec.RunStrategy != nil && *updatedVM.Spec.RunStrategy == runStrategyHalted
				}, 30*time.Second, 3*time.Second).Should(BeTrue())

				By(fmt.Sprintf("Ensure vm %s can be restored from vmsnapshots", vmYaml.vmName))
				for _, snapshot := range vmYaml.vmSnapshots {
					_, _, err = tests.RunCommand(k8sClient, "create", "-f", snapshot.restoreYamlFile, "--cache-dir", newClientCacheDir)
					Expect(err).ToNot(HaveOccurred())
					Eventually(func() bool {
						r, err := virtClient.VirtualMachineRestore(util2.NamespaceTestDefault).Get(context.Background(), snapshot.restoreName, metav1.GetOptions{})
						if err != nil {
							return false
						}
						return r.Status != nil && r.Status.Complete != nil && *r.Status.Complete
					}, 180*time.Second, 3*time.Second).Should(BeTrue())
				}

				By(fmt.Sprintf("Deleting VM with %s api", vmYaml.apiVersion))
				_, _, err = tests.RunCommand(k8sClient, "delete", "-f", vmYaml.yamlFile, "--cache-dir", newClientCacheDir)
				Expect(err).ToNot(HaveOccurred())

				By("Waiting for VM to be removed")
				Eventually(func() bool {
					_, err := virtClient.VirtualMachine(util2.NamespaceTestDefault).Get(vmYaml.vmName, &metav1.GetOptions{})
					if err != nil && errors.IsNotFound(err) {
						return true
					}
					return false
				}, 90*time.Second, 1*time.Second).Should(BeTrue())
			}

			By("Verifying all migratable vmi workloads are updated via live migration")
			verifyVMIsUpdated(migratableVMIs, launcherSha)

			By("Verifying that a once migrated VMI after an update can be migrated again")
			vmi := migratableVMIs[0]
			migration, err := virtClient.VirtualMachineInstanceMigration(vmi.Namespace).Create(tests.NewRandomMigration(vmi.Name, vmi.Namespace))
			Expect(err).ToNot(HaveOccurred())
			Eventually(ThisMigration(migration), 180).Should(HaveSucceeded())

			By("Deleting migratable VMIs")
			deleteAllVMIs(migratableVMIs)

			By("Deleting KubeVirt object")
			deleteAllKvAndWait(false)
		})
	})

	Describe("[rfe_id:2291][crit:high][vendor:cnv-qe@redhat.com][level:component]infrastructure management", func() {
		It("[test_id:3146]should be able to delete and re-create kubevirt install", func() {
			allPodsAreReady(originalKv)
			sanityCheckDeploymentsExist()

			By("Deleting KubeVirt object")
			deleteAllKvAndWait(false)

			// this is just verifying some common known components do in fact get deleted.
			By("Sanity Checking Deployments infrastructure is deleted")
			sanityCheckDeploymentsDeleted()

			By("ensuring that namespaces can be successfully created and deleted")
			_, err := virtClient.CoreV1().Namespaces().Create(context.Background(), &k8sv1.Namespace{ObjectMeta: metav1.ObjectMeta{Name: tests.NamespaceTestOperator}}, metav1.CreateOptions{})
			if err != nil && !errors.IsAlreadyExists(err) {
				Expect(err).ToNot(HaveOccurred())
			}
			err = virtClient.CoreV1().Namespaces().Delete(context.Background(), tests.NamespaceTestOperator, metav1.DeleteOptions{})
			Expect(err).ToNot(HaveOccurred())
			Eventually(func() bool {
				_, err := virtClient.CoreV1().Namespaces().Get(context.Background(), tests.NamespaceTestOperator, metav1.GetOptions{})
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
				kv, err := virtClient.KubeVirt(originalKv.Namespace).Get(originalKv.Name, &metav1.GetOptions{})
				Expect(err).ToNot(HaveOccurred())
				kv.Spec.UninstallStrategy = v1.KubeVirtUninstallStrategyBlockUninstallIfWorkloadsExist
				_, err = virtClient.KubeVirt(kv.Namespace).Update(kv)
				Expect(err).ToNot(HaveOccurred())

				By("creating a simple VMI")
				_, err = virtClient.VirtualMachineInstance(util2.NamespaceTestDefault).Create(tests.NewRandomVMIWithEphemeralDisk(cd.ContainerDiskFor(cd.ContainerDiskCirros)))
				Expect(err).ToNot(HaveOccurred())

				By("Deleting KubeVirt object")
				err = virtClient.KubeVirt(kv.Namespace).Delete(kv.Name, &metav1.DeleteOptions{})
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("there are still Virtual Machine Instances present"))
			})
		})

		It("[test_id:3148]should be able to create kubevirt install with custom image tag", func() {

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
		It("[test_id:3149]should be able to create kubevirt install with image prefix", func() {

			if flags.ImagePrefixAlt == "" {
				Skip("Skip operator imagePrefix test because imagePrefixAlt is not present")
			}

			kv := copyOriginalKv()

			allPodsAreReady(originalKv)
			sanityCheckDeploymentsExist()

			_, _, _, oldPrefix, _ := parseOperatorImage()

			By("Update Operator using imagePrefixAlt")
			patchOperator(&flags.ImagePrefixAlt, nil)

			// should result in kubevirt cr entering updating state
			By("Wait for Updating Condition")
			waitForUpdateCondition(kv)

			By("Waiting for KV to stabilize")
			waitForKv(kv)

			By("Verifying infrastructure Is Updated")
			allPodsAreReady(kv)

			By("Verifying deployments have prefix")
			for _, name := range []string{"virt-operator", "virt-api", "virt-controller"} {
				_, _, _, prefix, _ := parseDeployment(name)
				Expect(prefix).To(Equal(flags.ImagePrefixAlt), fmt.Sprintf("%s should have correct image prefix", name))
			}
			_, _, _, prefix, _ := parseDaemonset("virt-handler")
			Expect(prefix).To(Equal(flags.ImagePrefixAlt), "virt-handler should have correct image prefix")

			By("Verifying VMs are working")
			vmi := tests.NewRandomVMIWithEphemeralDisk(cd.ContainerDiskFor(cd.ContainerDiskAlpine))
			vmi, err := virtClient.VirtualMachineInstance(util2.NamespaceTestDefault).Create(vmi)
			Expect(err).ShouldNot(HaveOccurred(), "Create VMI successfully")
			tests.WaitForSuccessfulVMIStart(vmi)

			By("Verifying virt-launcher image is also prefixed")
			pod := tests.GetRunningPodByVirtualMachineInstance(vmi, vmi.Namespace)
			for _, container := range pod.Spec.Containers {
				if container.Name == "compute" {
					_, prefix, _ := parseImage("virt-launcher", container.Image)
					Expect(prefix).To(Equal(flags.ImagePrefixAlt), "launcher image should have prefix")
				}
			}

			By("Deleting VM")
			err = virtClient.VirtualMachineInstance(util2.NamespaceTestDefault).Delete(vmi.Name, &metav1.DeleteOptions{})
			Expect(err).ShouldNot(HaveOccurred(), "Delete VMI successfully")

			By("Restore Operator using original imagePrefix ")
			patchOperator(&oldPrefix, nil)

			By("Wait for Updating Condition")
			waitForUpdateCondition(kv)

			By("Waiting for KV to stabilize")
			waitForKv(kv)

			By("Verifying infrastructure Is Restored to original version")
			allPodsAreReady(kv)
		})

		It("[test_id:3150]should be able to update kubevirt install with custom image tag", func() {
			if flags.KubeVirtVersionTagAlt == "" {
				Skip("Skip operator custom image tag test because alt tag is not present")
			}

			vmis := generateMigratableVMIs(2)
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
		It("[test_id:3151]should be able to update kubevirt install when operator updates if no custom image tag is set", func() {

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

		It("[test_id:3152]should fail if KV object already exists", func() {

			newKv := copyOriginalKv()
			newKv.Name = "someother-kubevirt"

			By("Creating another KubeVirt object")
			createKv(newKv)
			By("Waiting for duplicate KubeVirt object to fail")
			Eventually(func() error {
				kv, err := virtClient.KubeVirt(flags.KubeVirtInstallNamespace).Get(newKv.Name, &metav1.GetOptions{})
				if err != nil {
					return err
				}

				failed := false
				for _, condition := range kv.Status.Conditions {
					if condition.Type == v1.KubeVirtConditionSynchronized &&
						condition.Status == k8sv1.ConditionFalse &&
						condition.Reason == "ExistingDeployment" {
						failed = true
					}
				}

				if !failed {
					return fmt.Errorf("Waiting for sync failed condition")
				}
				return nil
			}, 30*time.Second, 1*time.Second).ShouldNot(HaveOccurred())

			By("Deleting duplicate KubeVirt Object")
			deleteAllKvAndWait(true)
		})

		It("[test_id:4612]should create non-namespaces resources without owner references", func() {
			crd, err := virtClient.ExtensionsClient().ApiextensionsV1().CustomResourceDefinitions().Get(context.Background(), "virtualmachineinstances.kubevirt.io", metav1.GetOptions{})
			Expect(err).ToNot(HaveOccurred())
			Expect(crd.ObjectMeta.OwnerReferences).To(HaveLen(0))
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
			Expect(crd.ObjectMeta.OwnerReferences).To(HaveLen(0))
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
				if !tests.IsOpenShift() {
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
				vmi, err = virtClient.VirtualMachineInstance(util2.NamespaceTestDefault).Create(vmi)
				Expect(err).To(BeNil())
				tests.WaitForSuccessfulVMIStart(vmi)

				uid := vmi.GetObjectMeta().GetUID()
				labelSelector := fmt.Sprintf(v1.CreatedByLabel + "=" + string(uid))
				pods, err = virtClient.CoreV1().Pods(util2.NamespaceTestDefault).List(context.Background(), metav1.ListOptions{LabelSelector: labelSelector})
				Expect(err).ToNot(HaveOccurred(), "Should get virt-launcher")
				Expect(len(pods.Items)).To(Equal(1))
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
				_, err := virtClient.VirtualMachine(vm.Namespace).Get(vm.Name, &metav1.GetOptions{})
				if err != nil && !errors.IsNotFound(err) {
					Expect(err).ToNot(HaveOccurred())
				}

				if vm != nil && vm.DeletionTimestamp == nil {
					err = virtClient.VirtualMachine(vm.Namespace).Delete(vm.Name, &metav1.DeleteOptions{})
					Expect(err).ToNot(HaveOccurred())
				}
				vm = nil
			}
		})

		It("[test_id:3153]Ensure infra can handle dynamically detecting DataVolume Support", func() {
			if !tests.HasDataVolumeCRD() {
				Skip("Can't test DataVolume support when DataVolume CRD isn't present")
			}
			tests.SkipIfVersionBelow("Skipping dynamic cdi test in versions below 1.13 because crd garbage collection is broken", "1.13")

			// This tests starting infrastructure with and without the DataVolumes feature gate
			vm = tests.NewRandomVMWithDataVolume(cd.DataVolumeImportUrlForContainerDisk(cd.ContainerDiskAlpine), util2.NamespaceTestDefault)
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
			_, err = virtClient.VirtualMachine(util2.NamespaceTestDefault).Create(vm)
			Expect(err).To(HaveOccurred())

			// Enable DataVolumes by reinstalling CDI
			By("Enabling CDI install")
			createCdi()

			// wait for virt-api to pick up the change.
			time.Sleep(30 * time.Second)

			// Verify we can post a VM with DataVolumeTemplates successfully
			By("Expecting Error to not occur when posting VM with DataVolume")
			vm, err = virtClient.VirtualMachine(util2.NamespaceTestDefault).Create(vm)
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
			hasWorkloadUpdates := false
			if len(originalKv.Spec.WorkloadUpdateStrategy.WorkloadUpdateMethods) > 0 {
				hasWorkloadUpdates = true
			}
			expectedPromRuleSpec := components.NewPrometheusRuleSpec(flags.KubeVirtInstallNamespace, hasWorkloadUpdates)
			Expect(prometheusRule.Spec).To(Equal(*expectedPromRuleSpec))
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

					decompressed, err := ioutil.ReadAll(gzreader)
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
			labelKey := "kubevirt-test"
			labelValue := "test-label"
			infra := v1.ComponentConfig{
				NodePlacement: &v1.NodePlacement{
					NodeSelector: map[string]string{labelKey: labelValue},
				},
			}
			patchKvInfra(&infra, false, "")

			Eventually(func() bool {
				for _, name := range []string{"virt-controller", "virt-api"} {
					deployment, err := virtClient.AppsV1().Deployments(flags.KubeVirtInstallNamespace).Get(context.Background(), name, metav1.GetOptions{})
					Expect(err).ToNot(HaveOccurred())
					if deployment.Spec.Template.Spec.NodeSelector == nil || deployment.Spec.Template.Spec.NodeSelector[labelKey] != labelValue {
						return false
					}
				}
				return true
			}, 60*time.Second, 1*time.Second).Should(BeTrue())

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
			errMsg := fmt.Sprintf("Unsupported value: \"%s\"", incorrectOperator)
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
			errMsg := fmt.Sprintf("Unsupported value: \"%s\"", incorrectOperator)
			patchKvWorkloads(&incorrectWorkload, true, errMsg)
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
			for _, replicas := range []uint8{3, 1, 2} { // End with 2 so cluster is back to normal
				By(fmt.Sprintf("Setting the replica count in kvInfra to %d", replicas))
				var infra *v1.ComponentConfig
				if replicas != 2 { // Ensure that nil infra brings us back to 2 replicas
					infra = &v1.ComponentConfig{
						Replicas: &replicas,
					}
				}
				patchKvInfra(infra, false, "")

				By(fmt.Sprintf("Expecting %d replicas of virt-api and virt-controller", replicas))
				Eventually(func() bool {
					for _, name := range []string{"virt-api", "virt-controller"} {
						pods, err := virtClient.CoreV1().Pods(flags.KubeVirtInstallNamespace).List(context.Background(), metav1.ListOptions{LabelSelector: fmt.Sprintf("%s=%s", v1.AppLabel, name)})
						Expect(err).ToNot(HaveOccurred())
						if len(pods.Items) != int(replicas) {
							return false
						}
					}
					return true
				}, 60*time.Second, 1*time.Second).Should(BeTrue())

				if replicas == 1 {
					By(fmt.Sprintf("Expecting PDBs to disppear"))
					Eventually(func() bool {
						for _, name := range []string{"virt-api", "virt-controller"} {
							_, err := virtClient.PolicyV1beta1().PodDisruptionBudgets(flags.KubeVirtInstallNamespace).Get(context.Background(), name+"-pdb", metav1.GetOptions{})
							if err == nil {
								return false
							}
						}
						return true
					}, 60*time.Second, 1*time.Second).Should(BeTrue())
				} else {
					By(fmt.Sprintf("Expecting minAvailable to become %d on the PDBs", replicas-1))
					Eventually(func() bool {
						for _, name := range []string{"virt-api", "virt-controller"} {
							pdb, err := virtClient.PolicyV1beta1().PodDisruptionBudgets(flags.KubeVirtInstallNamespace).Get(context.Background(), name+"-pdb", metav1.GetOptions{})
							Expect(err).ToNot(HaveOccurred())
							if pdb.Spec.MinAvailable.IntValue() != int(replicas-1) {
								return false
							}
						}
						return true
					}, 60*time.Second, 1*time.Second).Should(BeTrue())
				}
			}
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
