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
	"encoding/json"

	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	k8sv1 "k8s.io/api/core/v1"
	extclient "k8s.io/apiextensions-apiserver/pkg/client/clientset/clientset"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"

	v1 "kubevirt.io/client-go/api/v1"
	"kubevirt.io/client-go/kubecli"
	"kubevirt.io/kubevirt/tests"
)

var _ = Describe("Operator", func() {
	tests.FlagParse()
	var originalKv *v1.KubeVirt
	var originalKubeVirtConfig *k8sv1.ConfigMap
	var err error
	var workDir string

	virtClient, err := kubecli.GetKubevirtClient()
	tests.PanicOnError(err)

	k8sClient := tests.GetK8sCmdClient()

	type vmYamlDefinition struct {
		apiVersion    string
		vmName        string
		generatedYaml string
		yamlFile      string
	}
	var vmYamls []vmYamlDefinition

	getKvList := func() []v1.KubeVirt {
		var kvListInstallNS *v1.KubeVirtList
		var kvListDefaultNS *v1.KubeVirtList
		var items []v1.KubeVirt

		var err error

		Eventually(func() error {

			kvListInstallNS, err = virtClient.KubeVirt(tests.KubeVirtInstallNamespace).List(&metav1.ListOptions{})

			return err
		}, 10*time.Second, 1*time.Second).ShouldNot(HaveOccurred())

		Eventually(func() error {

			kvListDefaultNS, err = virtClient.KubeVirt(tests.NamespaceTestDefault).List(&metav1.ListOptions{})

			return err
		}, 10*time.Second, 1*time.Second).ShouldNot(HaveOccurred())

		items = append(items, kvListInstallNS.Items...)
		items = append(items, kvListDefaultNS.Items...)

		return items
	}

	getCurrentKv := func() *v1.KubeVirt {
		kvs := getKvList()
		Expect(len(kvs)).To(Equal(1))
		return &kvs[0]
	}

	copyOriginalKv := func() *v1.KubeVirt {
		newKv := &v1.KubeVirt{
			Spec: *originalKv.Spec.DeepCopy(),
		}
		newKv.Name = originalKv.Name
		newKv.Namespace = originalKv.Namespace
		newKv.ObjectMeta.Labels = originalKv.ObjectMeta.Labels
		newKv.ObjectMeta.Annotations = originalKv.ObjectMeta.Annotations

		return newKv

	}

	createKv := func(newKv *v1.KubeVirt) {
		Eventually(func() error {
			_, err = virtClient.KubeVirt(newKv.Namespace).Create(newKv)
			return err
		}, 10*time.Second, 1*time.Second).ShouldNot(HaveOccurred())
	}

	sanityCheckDeploymentsExistWithNS := func(namespace string) {
		Eventually(func() error {
			_, err := virtClient.ExtensionsV1beta1().Deployments(namespace).Get("virt-api", metav1.GetOptions{})
			if err != nil {
				return err
			}

			_, err = virtClient.ExtensionsV1beta1().Deployments(namespace).Get("virt-controller", metav1.GetOptions{})
			if err != nil {
				return err
			}
			return nil
		}, 10*time.Second, 1*time.Second).ShouldNot(HaveOccurred())
	}

	sanityCheckDeploymentsExist := func() {
		sanityCheckDeploymentsExistWithNS(tests.KubeVirtInstallNamespace)
	}

	sanityCheckDeploymentsDeleted := func() {

		Eventually(func() error {
			_, err := virtClient.ExtensionsV1beta1().Deployments(tests.KubeVirtInstallNamespace).Get("virt-api", metav1.GetOptions{})
			if err != nil && !errors.IsNotFound(err) {
				return err
			}

			_, err = virtClient.ExtensionsV1beta1().Deployments(tests.KubeVirtInstallNamespace).Get("virt-controller", metav1.GetOptions{})
			if err != nil && !errors.IsNotFound(err) {
				return err
			}
			return nil
		}, 10*time.Second, 1*time.Second).ShouldNot(HaveOccurred())
	}

	allPodsAreReadyWithNS := func(expectedVersion string, namespace string) {
		Eventually(func() error {
			podsReadyAndOwned := 0

			pods, err := virtClient.CoreV1().Pods(namespace).List(metav1.ListOptions{LabelSelector: "kubevirt.io"})
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

				version, ok := pod.Annotations[v1.InstallStrategyVersionAnnotation]
				if !ok {
					return fmt.Errorf("Pod %s is owned by operator but has no version annotation", pod.Name)
				}

				if version != expectedVersion {
					return fmt.Errorf("Pod %s is of version %s when we expected version %s", pod.Name, version, expectedVersion)
				}
				podsReadyAndOwned++
			}

			// this just sanity checks that at least one pod was found and verified.
			// 0 would indicate our labeling was incorrect.
			Expect(podsReadyAndOwned).ToNot(Equal(0))

			return nil
		}, 120*time.Second, 1*time.Second).ShouldNot(HaveOccurred())
	}

	allPodsAreReady := func(expectedVersion string) {
		allPodsAreReadyWithNS(expectedVersion, tests.KubeVirtInstallNamespace)
	}

	waitForUpdateCondition := func(kv *v1.KubeVirt) {
		Eventually(func() error {
			kv, err := virtClient.KubeVirt(kv.Namespace).Get(kv.Name, &metav1.GetOptions{})
			if err != nil {
				return err
			}

			updating := false
			for _, condition := range kv.Status.Conditions {
				if condition.Type == v1.KubeVirtConditionUpdating {
					updating = true
				}
			}
			if !updating {
				return fmt.Errorf("Waiting for updating condition")
			}
			return nil
		}, 120*time.Second, 1*time.Second).ShouldNot(HaveOccurred())

	}

	disableFeatureGate := func(feature string) {
		if !tests.HasFeature(feature) {
			return
		}

		cfg, err := virtClient.CoreV1().ConfigMaps(tests.KubeVirtInstallNamespace).Get("kubevirt-config", metav1.GetOptions{})
		Expect(err).ToNot(HaveOccurred())

		val, _ := cfg.Data["feature-gates"]

		newVal := strings.Replace(val, feature+",", "", 1)
		newVal = strings.Replace(newVal, feature, "", 1)

		cfg.Data["feature-gates"] = newVal

		newData, err := json.Marshal(cfg.Data)
		Expect(err).ToNot(HaveOccurred())

		data := fmt.Sprintf(`[{ "op": "replace", "path": "/data", "value": %s }]`, string(newData))
		_, err = virtClient.CoreV1().ConfigMaps(tests.KubeVirtInstallNamespace).Patch("kubevirt-config", types.JSONPatchType, []byte(data))
		Expect(err).ToNot(HaveOccurred())
	}

	enableFeatureGate := func(feature string) {
		if tests.HasFeature(feature) {
			return
		}

		cfg, err := virtClient.CoreV1().ConfigMaps(tests.KubeVirtInstallNamespace).Get("kubevirt-config", metav1.GetOptions{})
		Expect(err).ToNot(HaveOccurred())

		val, _ := cfg.Data["feature-gates"]
		newVal := fmt.Sprintf("%s,%s", val, feature)

		cfg.Data["feature-gates"] = newVal

		newData, err := json.Marshal(cfg.Data)
		Expect(err).ToNot(HaveOccurred())

		data := fmt.Sprintf(`[{ "op": "replace", "path": "/data", "value": %s }]`, string(newData))

		_, err = virtClient.CoreV1().ConfigMaps(tests.KubeVirtInstallNamespace).Patch("kubevirt-config", types.JSONPatchType, []byte(data))
		Expect(err).ToNot(HaveOccurred())
	}

	waitForKvWithTimeout := func(newKv *v1.KubeVirt, timeoutSeconds int) {
		Eventually(func() error {
			kv, err := virtClient.KubeVirt(newKv.Namespace).Get(newKv.Name, &metav1.GetOptions{})
			if err != nil {
				return err
			}

			if kv.Status.Phase != v1.KubeVirtPhaseDeployed {
				return fmt.Errorf("Waiting for phase to be deployed")
			}

			ready := false
			created := false
			for _, condition := range kv.Status.Conditions {
				if condition.Type == v1.KubeVirtConditionReady && condition.Status == k8sv1.ConditionTrue {
					ready = true
				} else if condition.Type == v1.KubeVirtConditionCreated && condition.Status == k8sv1.ConditionTrue {
					created = true
				}
			}

			if !ready || !created {
				return fmt.Errorf("Waiting for phase to be deployed")
			}
			return nil
		}, time.Duration(timeoutSeconds)*time.Second, 1*time.Second).ShouldNot(HaveOccurred())
	}

	waitForKv := func(newKv *v1.KubeVirt) {
		waitForKvWithTimeout(newKv, 300)
	}

	patchKvVersionAndRegistry := func(name string, version string, registry string) {
		data := []byte(fmt.Sprintf(`[{ "op": "replace", "path": "/spec/imageTag", "value": "%s"},{ "op": "replace", "path": "/spec/imageRegistry", "value": "%s"}]`, version, registry))
		Eventually(func() error {
			_, err := virtClient.KubeVirt(tests.KubeVirtInstallNamespace).Patch(name, types.JSONPatchType, data)

			return err
		}, 10*time.Second, 1*time.Second).ShouldNot(HaveOccurred())
	}

	patchKvVersion := func(name string, version string) {
		data := []byte(fmt.Sprintf(`[{ "op": "add", "path": "/spec/imageTag", "value": "%s"}]`, version))
		Eventually(func() error {
			_, err := virtClient.KubeVirt(tests.KubeVirtInstallNamespace).Patch(name, types.JSONPatchType, data)

			return err
		}, 10*time.Second, 1*time.Second).ShouldNot(HaveOccurred())
	}

	patchOperatorVersion := func(imageTag string) {
		Eventually(func() error {

			operator, err := virtClient.AppsV1().Deployments(tests.KubeVirtInstallNamespace).Get("virt-operator", metav1.GetOptions{})

			imageRegEx := regexp.MustCompile(`^(.*)/virt-operator(:.*)?$`)
			matches := imageRegEx.FindAllStringSubmatch(operator.Spec.Template.Spec.Containers[0].Image, 1)
			registry := matches[0][1]
			newImage := fmt.Sprintf("%s/virt-operator:%s", registry, imageTag)

			operator.Spec.Template.Spec.Containers[0].Image = newImage
			for idx, env := range operator.Spec.Template.Spec.Containers[0].Env {
				if env.Name == "OPERATOR_IMAGE" {
					env.Value = newImage
					operator.Spec.Template.Spec.Containers[0].Env[idx] = env
					break
				}
			}

			newTemplate, _ := json.Marshal(operator.Spec.Template)

			op := fmt.Sprintf(`[{ "op": "replace", "path": "/spec/template", "value": %s }]`, string(newTemplate))

			_, err = virtClient.AppsV1().Deployments(tests.KubeVirtInstallNamespace).Patch("virt-operator", types.JSONPatchType, []byte(op))

			return err
		}, 10*time.Second, 1*time.Second).ShouldNot(HaveOccurred())
	}

	deleteAllKvAndWait := func(ignoreOriginal bool) {
		Eventually(func() error {

			kvs := getKvList()

			deleteCount := 0
			for _, kv := range kvs {

				if ignoreOriginal && kv.Name == originalKv.Name {
					continue
				}
				deleteCount++
				if kv.DeletionTimestamp == nil {
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

		}, 120*time.Second, 1*time.Second).ShouldNot(HaveOccurred())

	}

	tests.BeforeAll(func() {
		originalKv = getCurrentKv()

		originalKubeVirtConfig, err = virtClient.CoreV1().ConfigMaps(tests.KubeVirtInstallNamespace).Get("kubevirt-config", metav1.GetOptions{})
		if err != nil && !errors.IsNotFound(err) {
			Expect(err).ToNot(HaveOccurred())
		}

		if errors.IsNotFound(err) {
			// create an empty kubevirt-config configmap if none exists.
			cfgMap := &k8sv1.ConfigMap{
				ObjectMeta: metav1.ObjectMeta{Name: "kubevirt-config"},
				Data: map[string]string{
					"feature-gates": "",
				},
			}

			originalKubeVirtConfig, err = virtClient.CoreV1().ConfigMaps(tests.KubeVirtInstallNamespace).Create(cfgMap)
			Expect(err).ToNot(HaveOccurred())

		}

	})

	generateVmYamls := func() {
		ext, err := extclient.NewForConfig(virtClient.Config())
		Expect(err).ToNot(HaveOccurred())

		crd, err := ext.ApiextensionsV1beta1().CustomResourceDefinitions().Get("virtualmachines.kubevirt.io", metav1.GetOptions{})
		Expect(err).ToNot(HaveOccurred())

		// Generate a vm Yaml for every version supported in the currently deployed KubeVirt

		supportedVersions := []string{}

		if len(crd.Spec.Versions) > 0 {
			for _, version := range crd.Spec.Versions {
				supportedVersions = append(supportedVersions, version.Name)
			}
		} else {
			supportedVersions = append(supportedVersions, crd.Spec.Version)
		}

		for _, version := range supportedVersions {
			vmYaml := fmt.Sprintf(`apiVersion: kubevirt.io/%s
kind: VirtualMachine
metadata:
  labels:
    kubevirt.io/vm: vm-%s
  name: vm-%s
spec:
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
        machine:
          type: ""
        resources:
          requests:
            memory: 64M
      terminationGracePeriodSeconds: 0
      volumes:
      - containerDisk:
          image: %s/%s-container-disk-demo:%s
        name: containerdisk
      - cloudInitNoCloud:
          userData: |
            #!/bin/sh

            echo 'printed from cloud-init userdata'
        name: cloudinitdisk
`, version, version, version, version, tests.KubeVirtUtilityRepoPrefix, tests.ContainerDiskCirros, tests.KubeVirtUtilityVersionTag)

			yamlFile := filepath.Join(workDir, fmt.Sprintf("vm-%s.yaml", version))
			err = ioutil.WriteFile(yamlFile, []byte(vmYaml), 0644)

			Expect(err).ToNot(HaveOccurred())

			vmYamls = append(vmYamls, vmYamlDefinition{
				apiVersion:    version,
				vmName:        "vm-" + version,
				generatedYaml: vmYaml,
				yamlFile:      yamlFile,
			})
		}

	}

	BeforeEach(func() {
		tests.BeforeTestCleanup()

		workDir, err = ioutil.TempDir("", tests.TempDirPrefix+"-")
		Expect(err).ToNot(HaveOccurred())

		vmYamls = []vmYamlDefinition{}
	})

	AfterEach(func() {
		ignoreDeleteOriginalKV := true

		curKubeVirtConfig, err := virtClient.CoreV1().ConfigMaps(tests.KubeVirtInstallNamespace).Get("kubevirt-config", metav1.GetOptions{})
		Expect(err).ToNot(HaveOccurred())

		// if revision changed, patch data and reload everything
		if curKubeVirtConfig.ResourceVersion != originalKubeVirtConfig.ResourceVersion {
			ignoreDeleteOriginalKV = false

			// Add Spec Patch
			newData, err := json.Marshal(originalKubeVirtConfig.Data)
			Expect(err).ToNot(HaveOccurred())
			data := fmt.Sprintf(`[{ "op": "replace", "path": "/data", "value": %s }]`, string(newData))

			originalKubeVirtConfig, err = virtClient.CoreV1().ConfigMaps(tests.KubeVirtInstallNamespace).Patch("kubevirt-config", types.JSONPatchType, []byte(data))
			Expect(err).ToNot(HaveOccurred())
		}

		deleteAllKvAndWait(ignoreDeleteOriginalKV)

		kvs := getKvList()

		if len(kvs) == 0 {
			createKv(copyOriginalKv())
		}
		patchOperatorVersion(tests.KubeVirtVersionTag)
		waitForKv(originalKv)
		allPodsAreReady(tests.KubeVirtVersionTag)

		if workDir != "" {
			err = os.RemoveAll(workDir)
			workDir = ""
			Expect(err).ToNot(HaveOccurred())
		}

	})

	Describe("should update kubevirt", func() {

		// This test is installing a previous release of KubeVirt
		// running a VM/VMI using that previous release
		// Updating KubeVirt to the target tested code
		// Ensuring VM/VMI is still operational after the update from previous release.
		It("from previous release to target tested release", func() {
			previousImageTag := tests.PreviousReleaseTag
			previousImageRegistry := tests.PreviousReleaseRegistry

			if previousImageTag == "" {
				Skip("--previous-release-tag not provided")
			}

			curTag := originalKv.Status.ObservedKubeVirtVersion
			curRegistry := originalKv.Status.ObservedKubeVirtRegistry

			allPodsAreReady(tests.KubeVirtVersionTag)
			sanityCheckDeploymentsExist()

			// Delete current KubeVirt install so we can install previous release.
			By("Deleting KubeVirt object")
			deleteAllKvAndWait(false)

			By("Sanity Checking Deployments infrastructure is deleted")
			sanityCheckDeploymentsDeleted()

			// Install Previous Release of KubeVirt
			By(fmt.Sprintf("Creating KubeVirt Object with Previous Release: %s using registry %s", previousImageTag, previousImageRegistry))
			kv := copyOriginalKv()
			kv.Name = "kubevirt-release-install"
			kv.Spec.ImageTag = previousImageTag
			kv.Spec.ImageRegistry = previousImageRegistry
			createKv(kv)

			// Wait for Previous Release to come online
			// wait 7 minutes because this test involves pulling containers
			// over the internet related to the latest kubevirt release
			By("Waiting for KV to stabilize")
			waitForKvWithTimeout(kv, 420)

			By("Verifying infrastructure is Ready")
			allPodsAreReady(previousImageTag)
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
			// NOTE: we are testing with yaml here and explicilty _NOT_ generating
			// this vm using the latest api code. We want to guarrantee there are no
			// surprises when it comes to backwards compatiblity with previous
			// virt apis.  As we progress our api from v1alpha3 -> v1 there
			// needs to be a VM created for every api. This is how we will ensure
			// our api remains upgradable and supportable from previous release.

			generateVmYamls()
			for _, vmYaml := range vmYamls {
				By(fmt.Sprintf("Creating VM with %s api", vmYaml.vmName))
				// NOTE: using kubectl to post yaml directly
				_, _, err = tests.RunCommand(k8sClient, "create", "-f", vmYaml.yamlFile, "--cache-dir", oldClientCacheDir)
				Expect(err).ToNot(HaveOccurred())

				// Use Current virtctl to start VM
				// NOTE: we are using virtctl explicitly here because we want to start the VM
				// using the subresource endpoint in the same way virtctl performs this.
				By("Starting VM with virtctl")
				startFn := tests.NewRepeatableVirtctlCommand("start", "--namespace", tests.NamespaceTestDefault, vmYaml.vmName)
				err = startFn()
				Expect(err).ToNot(HaveOccurred())

				By(fmt.Sprintf("Waiting for VM with %s api to become ready", vmYaml.apiVersion))

				Eventually(func() bool {
					virtualMachine, err := virtClient.VirtualMachine(tests.NamespaceTestDefault).Get(vmYaml.vmName, &metav1.GetOptions{})
					Expect(err).ToNot(HaveOccurred())
					if virtualMachine.Status.Ready {
						return true
					}
					return false
				}, 90*time.Second, 1*time.Second).Should(BeTrue())
			}

			// Update KubeVirt from the previous release to the testing target release.
			By("Updating KubeVirtObject With Current Tag")
			patchKvVersionAndRegistry(kv.Name, curTag, curRegistry)

			By("Wait for Updating Condition")
			waitForUpdateCondition(kv)

			By("Waiting for KV to stabilize")
			waitForKv(kv)

			By("Verifying infrastructure Is Updated")
			allPodsAreReady(curTag)

			// Verify console connectivity to VMI still works and stop VM
			for _, vmYaml := range vmYamls {
				By(fmt.Sprintf("Ensuring vm %s is ready and latest API annotation is set", vmYaml.apiVersion))
				Eventually(func() bool {
					// We are using our internal client here on purpose to ensure we can interact
					// with previously created objects that may have been created using a different
					// api version from the latest one our client uses.
					virtualMachine, err := virtClient.VirtualMachine(tests.NamespaceTestDefault).Get(vmYaml.vmName, &metav1.GetOptions{})
					Expect(err).ToNot(HaveOccurred())
					if !virtualMachine.Status.Ready {
						return false
					}

					return true
				}, 90*time.Second, 1*time.Second).Should(BeTrue())

				By(fmt.Sprintf("Connecting to %s's console", vmYaml.vmName))
				// This is in an eventually loop because it's possible for the
				// subresource endpoint routing to fail temporarily right after a deployment
				// completes while we wait for the kubernetes apiserver to detect our
				// subresource api server is online and ready to serve requests.
				Eventually(func() error {
					vmi, err := virtClient.VirtualMachineInstance(tests.NamespaceTestDefault).Get(vmYaml.vmName, &metav1.GetOptions{})
					Expect(err).ToNot(HaveOccurred())
					expecter, err := tests.LoggedInCirrosExpecter(vmi)
					if err != nil {
						return err
					}
					expecter.Close()
					return nil
				}, 60*time.Second, 1*time.Second).Should(BeNil())

				By("Stopping VM with virtctl")
				stopFn := tests.NewRepeatableVirtctlCommand("stop", "--namespace", tests.NamespaceTestDefault, vmYaml.vmName)
				Eventually(func() error {
					return stopFn()
				}, 30*time.Second, 1*time.Second).Should(BeNil())

				By("Waiting for VMI to stop")
				Eventually(func() bool {
					_, err := virtClient.VirtualMachineInstance(tests.NamespaceTestDefault).Get(vmYaml.vmName, &metav1.GetOptions{})
					if err != nil && errors.IsNotFound(err) {
						return true
					} else if err != nil {
						Expect(err).ToNot(HaveOccurred())
					}
					return false
				}, 90*time.Second, 1*time.Second).Should(BeTrue())

				By(fmt.Sprintf("Deleting VM with %s api", vmYaml.apiVersion))
				_, _, err = tests.RunCommand(k8sClient, "delete", "-f", vmYaml.yamlFile, "--cache-dir", newClientCacheDir)
				Expect(err).ToNot(HaveOccurred())

				By("Waiting for VM to be removed")
				Eventually(func() bool {
					_, err := virtClient.VirtualMachine(tests.NamespaceTestDefault).Get(vmYaml.vmName, &metav1.GetOptions{})
					if err != nil && errors.IsNotFound(err) {
						return true
					}
					return false
				}, 90*time.Second, 1*time.Second).Should(BeTrue())
			}

			By("Deleting KubeVirt object")
			deleteAllKvAndWait(false)
		})
	})

	Describe("infrastructure management", func() {
		It("should be able to delete and re-create kubevirt install", func() {
			allPodsAreReady(tests.KubeVirtVersionTag)
			sanityCheckDeploymentsExist()

			By("Deleting KubeVirt object")
			deleteAllKvAndWait(false)

			// this is just verifying some common known components do in fact get deleted.
			By("Sanity Checking Deployments infrastructure is deleted")
			sanityCheckDeploymentsDeleted()

			By("Creating KubeVirt Object")
			createKv(copyOriginalKv())

			By("Creating KubeVirt Object Created and Ready Condition")
			waitForKv(originalKv)

			By("Verifying infrastructure is Ready")
			allPodsAreReady(tests.KubeVirtVersionTag)
			// We're just verifying that a few common components that
			// should always exist get re-deployed.
			sanityCheckDeploymentsExist()
		})

		It("should be able to delete and re-create kubevirt install in a different namespace", func() {
			allPodsAreReady(tests.KubeVirtVersionTag)
			sanityCheckDeploymentsExist()

			By("Deleting KubeVirt object")
			deleteAllKvAndWait(false)

			// this is just verifying some common known components do in fact get deleted.
			By("Sanity Checking Deployments infrastructure is deleted")
			sanityCheckDeploymentsDeleted()

			By("Creating KubeVirt Object")
			newKV := copyOriginalKv()
			newKV.Name = "kubevirt-test-default-namespace"
			newKV.Namespace = tests.NamespaceTestDefault

			createKv(newKV)

			By("Creating KubeVirt Object Created and Ready Condition")
			waitForKv(newKV)

			By("Verifying infrastructure is Ready")
			allPodsAreReadyWithNS(tests.KubeVirtVersionTag, newKV.Namespace)
			// We're just verifying that a few common components that
			// should always exist get re-deployed.
			sanityCheckDeploymentsExistWithNS(newKV.Namespace)
		})

		It("should be able to create kubevirt install with custom image tag", func() {

			if tests.KubeVirtVersionTagAlt == "" {
				Skip("Skip operator custom image tag test because alt tag is not present")
			}

			allPodsAreReady(tests.KubeVirtVersionTag)
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
				ImageTag:      tests.KubeVirtVersionTagAlt,
				ImageRegistry: tests.KubeVirtRepoPrefix,
			}
			createKv(kv)

			By("Creating KubeVirt Object Created and Ready Condition")
			waitForKv(kv)

			By("Verifying infrastructure is Ready")
			allPodsAreReady(tests.KubeVirtVersionTagAlt)
			// We're just verifying that a few common components that
			// should always exist get re-deployed.
			sanityCheckDeploymentsExist()

			By("Deleting KubeVirt object")
			deleteAllKvAndWait(false)
		})

		It("should be able to update kubevirt install with custom image tag", func() {

			if tests.KubeVirtVersionTagAlt == "" {
				Skip("Skip operator custom image tag test because alt tag is not present")
			}

			allPodsAreReady(tests.KubeVirtVersionTag)
			sanityCheckDeploymentsExist()

			By("Deleting KubeVirt object")
			deleteAllKvAndWait(false)

			// this is just verifying some common known components do in fact get deleted.
			By("Sanity Checking Deployments infrastructure is deleted")
			sanityCheckDeploymentsDeleted()

			By("Creating KubeVirt Object")
			kv := copyOriginalKv()
			kv.Name = "kubevirt-alt-install"
			createKv(kv)

			By("Creating KubeVirt Object Created and Ready Condition")
			waitForKv(kv)

			By("Verifying infrastructure is Ready")
			allPodsAreReady(tests.KubeVirtVersionTag)
			// We're just verifying that a few common components that
			// should always exist get re-deployed.
			sanityCheckDeploymentsExist()

			By("Updating KubeVirtObject With Alt Tag")
			patchKvVersion(kv.Name, tests.KubeVirtVersionTagAlt)

			By("Wait for Updating Condition")
			waitForUpdateCondition(kv)

			By("Waiting for KV to stabilize")
			waitForKv(kv)

			By("Verifying infrastructure Is Updated")
			allPodsAreReady(tests.KubeVirtVersionTagAlt)

			By("Deleting KubeVirt object")
			deleteAllKvAndWait(false)

		})

		// NOTE - this test verifies new operators can grab the leader election lease
		// during operator updates. The only way the new infrastructure is deployed
		// is if the update operator is capable of getting the lease.
		It("should be able to update kubevirt install when operator updates if no custom image tag is set", func() {

			if tests.KubeVirtVersionTagAlt == "" {
				Skip("Skip operator custom image tag test because alt tag is not present")
			}

			kv := copyOriginalKv()

			allPodsAreReady(tests.KubeVirtVersionTag)
			sanityCheckDeploymentsExist()

			By("Update Virt-Operator using  Alt Tag")
			patchOperatorVersion(tests.KubeVirtVersionTagAlt)

			// should result in kubevirt cr entering updating state
			By("Wait for Updating Condition")
			waitForUpdateCondition(kv)

			By("Waiting for KV to stabilize")
			waitForKv(kv)

			By("Verifying infrastructure Is Updated")
			allPodsAreReady(tests.KubeVirtVersionTagAlt)

			By("Restore Operator Version using original image tag. ")
			patchOperatorVersion(tests.KubeVirtVersionTag)

			By("Wait for Updating Condition")
			waitForUpdateCondition(kv)

			By("Waiting for KV to stabilize")
			waitForKv(kv)

			By("Verifying infrastructure Is Restored to original version")
			allPodsAreReady(tests.KubeVirtVersionTag)
		})

		It("should fail if KV object already exists", func() {

			newKv := copyOriginalKv()
			newKv.Name = "someother-kubevirt"

			By("Creating another KubeVirt object")
			createKv(newKv)
			By("Waiting for duplicate KubeVirt object to fail")
			Eventually(func() error {
				kv, err := virtClient.KubeVirt(tests.KubeVirtInstallNamespace).Get(newKv.Name, &metav1.GetOptions{})
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
	})

	Describe("feature flag enabling/disabling", func() {

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

		It("[test_id:835]Ensure infra can start with DataVolume feature gate enabled.", func() {
			if !tests.HasDataVolumeCRD() {
				Skip("Can't test DataVolume support when DataVolume CRD isn't present")
			}

			// This tests starting infrastructure with and without the DataVolumes feature gate
			running := false
			vm = tests.NewRandomVMWithDataVolume(tests.AlpineHttpUrl, tests.NamespaceTestDefault)
			vm.Spec.Running = &running

			// Disable the gate
			By("DisablingFeatureGate")
			disableFeatureGate("DataVolumes")

			// Cycle the infrastructure to pick up the change.
			allPodsAreReady(tests.KubeVirtVersionTag)
			sanityCheckDeploymentsExist()

			By("Deleting KubeVirt object")
			deleteAllKvAndWait(false)

			By("Sanity Checking Deployments infrastructure is deleted")
			sanityCheckDeploymentsDeleted()

			By("Creating KubeVirt Object")
			createKv(copyOriginalKv())

			By("Creating KubeVirt Object Created and Ready Condition")
			waitForKv(originalKv)

			By("Verifying infrastructure is Ready")
			allPodsAreReady(tests.KubeVirtVersionTag)
			sanityCheckDeploymentsExist()

			// Verify posting a VM with DataVolumeTemplate fails when DataVolumes
			// feature gate is disabled
			By("Expecting Error to Occur when posting VM with DataVolume")
			_, err = virtClient.VirtualMachine(tests.NamespaceTestDefault).Create(vm)
			Expect(err).To(HaveOccurred())

			// Enable DataVolumes feature gate
			By("EnablingFeatureGate")
			enableFeatureGate("DataVolumes")

			// Cycle the infrastructure so we know the feature gate is picked up.
			By("Deleting KubeVirt object")
			deleteAllKvAndWait(false)

			By("Sanity Checking Deployments infrastructure is deleted")
			sanityCheckDeploymentsDeleted()

			By("Creating KubeVirt Object")
			createKv(copyOriginalKv())

			By("Creating KubeVirt Object Created and Ready Condition")
			waitForKv(originalKv)

			By("Verifying infrastructure is Ready")
			allPodsAreReady(tests.KubeVirtVersionTag)

			// Verify we can post a VM with DataVolumeTemplates successfully
			By("Expecting Error to not occur when posting VM with DataVolume")
			_, err = virtClient.VirtualMachine(tests.NamespaceTestDefault).Create(vm)
			Expect(err).ToNot(HaveOccurred())
		})
	})
})
