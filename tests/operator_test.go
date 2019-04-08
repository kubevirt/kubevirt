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
	"flag"
	"fmt"
	"time"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	k8sv1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"

	v1 "kubevirt.io/kubevirt/pkg/api/v1"
	"kubevirt.io/kubevirt/pkg/kubecli"
	"kubevirt.io/kubevirt/tests"
)

var _ = Describe("Operator", func() {
	flag.Parse()
	var originalKv *v1.KubeVirt

	virtClient, err := kubecli.GetKubevirtClient()
	tests.PanicOnError(err)

	getKvList := func() []v1.KubeVirt {
		var kvList *v1.KubeVirtList
		var err error

		Eventually(func() error {

			kvList, err = virtClient.KubeVirt(tests.KubeVirtInstallNamespace).List(&metav1.ListOptions{})

			return err
		}, 10*time.Second, 1*time.Second).ShouldNot(HaveOccurred())

		return kvList.Items
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
			_, err = virtClient.KubeVirt(tests.KubeVirtInstallNamespace).Create(newKv)
			return err
		}, 10*time.Second, 1*time.Second).ShouldNot(HaveOccurred())
	}

	sanityCheckDeploymentsExist := func() {
		Eventually(func() error {
			_, err := virtClient.ExtensionsV1beta1().Deployments(tests.KubeVirtInstallNamespace).Get("virt-api", metav1.GetOptions{})
			if err != nil {
				return err
			}

			_, err = virtClient.ExtensionsV1beta1().Deployments(tests.KubeVirtInstallNamespace).Get("virt-controller", metav1.GetOptions{})
			if err != nil {
				return err
			}
			return nil
		}, 10*time.Second, 1*time.Second).ShouldNot(HaveOccurred())
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

	allPodsAreReady := func(expectedVersion string) {
		Eventually(func() error {
			podsReadyAndOwned := 0

			pods, err := virtClient.CoreV1().Pods(tests.KubeVirtInstallNamespace).List(metav1.ListOptions{LabelSelector: "kubevirt.io"})
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

	waitForUpdateCondition := func(kv *v1.KubeVirt) {
		Eventually(func() error {
			kv, err := virtClient.KubeVirt(tests.KubeVirtInstallNamespace).Get(kv.Name, &metav1.GetOptions{})
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
		}, 30*time.Second, 1*time.Second).ShouldNot(HaveOccurred())

	}

	waitForKv := func(newKv *v1.KubeVirt) {
		Eventually(func() error {
			kv, err := virtClient.KubeVirt(tests.KubeVirtInstallNamespace).Get(newKv.Name, &metav1.GetOptions{})
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
		}, 160*time.Second, 1*time.Second).ShouldNot(HaveOccurred())
	}

	patchKvVersion := func(name string, version string) {
		data := []byte(fmt.Sprintf(`[{ "op": "add", "path": "/spec/imageTag", "value": "%s"}]`, version))
		Eventually(func() error {
			_, err := virtClient.KubeVirt(tests.KubeVirtInstallNamespace).Patch(name, types.JSONPatchType, data)

			return err
		}, 10*time.Second, 1*time.Second).ShouldNot(HaveOccurred())
	}

	deleteAllKvAndWait := func(ignoreOriginal bool) {
		Eventually(func() error {
			kvList, err := virtClient.KubeVirt(tests.KubeVirtInstallNamespace).List(&metav1.ListOptions{})
			if err != nil {
				return err
			}

			deleteCount := 0
			for _, kv := range kvList.Items {

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
	})

	BeforeEach(func() {
		tests.BeforeTestCleanup()
	})

	AfterEach(func() {
		deleteAllKvAndWait(true)

		kvs := getKvList()

		if len(kvs) == 0 {
			createKv(copyOriginalKv())
		}
		waitForKv(originalKv)
		allPodsAreReady(tests.KubeVirtVersionTag)
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
})
