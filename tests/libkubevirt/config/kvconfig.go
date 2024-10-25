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
 * Copyright 2023 Red Hat, Inc.
 *
 */

package config

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"encoding/json"
	"fmt"
	"io"
	"math/rand"
	"net/http"
	"strconv"
	"strings"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	k8sv1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/equality"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/strategicpatch"

	v1 "kubevirt.io/api/core/v1"
	"kubevirt.io/client-go/kubecli"
	"kubevirt.io/client-go/log"

	"kubevirt.io/kubevirt/pkg/apimachinery/patch"
	"kubevirt.io/kubevirt/tests/flags"
	"kubevirt.io/kubevirt/tests/framework/kubevirt"
	"kubevirt.io/kubevirt/tests/libkubevirt"
	"kubevirt.io/kubevirt/tests/libpod"
)

const (
	virtApiComponentName        = "virt-api"
	virtHandlerComponentName    = "virt-handler"
	virtControllerComponentName = "virt-controller"
)

type compare func(string, string) bool

func RegisterKubevirtConfigChange(change func(c v1.KubeVirtConfiguration) (*patch.PatchSet, error)) error {
	kv := libkubevirt.GetCurrentKv(kubevirt.Client())
	patchSet, err := change(kv.Spec.Configuration)
	if err != nil {
		return fmt.Errorf("failed changing the kubevirt configuration: %v", err)
	}

	if patchSet.IsEmpty() {
		return nil
	}

	return patchKV(kv.Namespace, kv.Name, patchSet)
}

func patchKV(namespace, name string, patchSet *patch.PatchSet) error {
	patchData, err := patchSet.GeneratePayload()
	if err != nil {
		return err
	}
	_, err = kubevirt.Client().KubeVirt(namespace).Patch(context.Background(), name, types.JSONPatchType, patchData, metav1.PatchOptions{})
	return err
}

func PatchWorkloadUpdateMethodAndRolloutStrategy(kvName string, virtClient kubecli.KubevirtClient, updateStrategy *v1.KubeVirtWorkloadUpdateStrategy, rolloutStrategy *v1.VMRolloutStrategy, fgs []string) {
	patch, err := patch.New(
		patch.WithAdd("/spec/workloadUpdateStrategy", updateStrategy),
		patch.WithAdd("/spec/configuration/vmRolloutStrategy", rolloutStrategy),
		patch.WithAdd("/spec/configuration/developerConfiguration/featureGates", fgs),
	).GeneratePayload()
	ExpectWithOffset(1, err).ToNot(HaveOccurred())
	EventuallyWithOffset(1, func() error {
		_, err := virtClient.KubeVirt(flags.KubeVirtInstallNamespace).Patch(context.Background(), kvName, types.JSONPatchType, patch, metav1.PatchOptions{})
		return err
	}, 10*time.Second, 1*time.Second).ShouldNot(HaveOccurred())
}

// UpdateKubeVirtConfigValueAndWait updates the given configuration in the kubevirt custom resource
// and then waits  to allow the configuration events to be propagated to the consumers.
func UpdateKubeVirtConfigValueAndWait(kvConfig v1.KubeVirtConfiguration) *v1.KubeVirt {
	kv := UpdateKubeVirtConfigValue(kvConfig)

	waitForConfigToBePropagated(kv.ResourceVersion)
	log.DefaultLogger().Infof("system is in sync with kubevirt config resource version %s", kv.ResourceVersion)

	return kv
}

// UpdateKubeVirtConfigValue updates the given configuration in the kubevirt custom resource
func UpdateKubeVirtConfigValue(kvConfig v1.KubeVirtConfiguration) *v1.KubeVirt {

	virtClient := kubevirt.Client()

	kv := libkubevirt.GetCurrentKv(virtClient)
	old, err := json.Marshal(kv)
	Expect(err).ToNot(HaveOccurred())

	if equality.Semantic.DeepEqual(kv.Spec.Configuration, kvConfig) {
		return kv
	}

	Expect(CurrentSpecReport().IsSerial).To(BeTrue(), "Tests which alter the global kubevirt configuration must not be executed in parallel, see https://onsi.github.io/ginkgo/#serial-specs")

	updatedKV := kv.DeepCopy()
	updatedKV.Spec.Configuration = kvConfig
	newJson, err := json.Marshal(updatedKV)
	Expect(err).ToNot(HaveOccurred())

	patch, err := strategicpatch.CreateTwoWayMergePatch(old, newJson, kv)
	Expect(err).ToNot(HaveOccurred())

	kv, err = virtClient.KubeVirt(kv.Namespace).Patch(context.Background(), kv.GetName(), types.MergePatchType, patch, metav1.PatchOptions{})
	Expect(err).ToNot(HaveOccurred())

	return kv
}

func ExpectResourceVersionToBeLessEqualThanConfigVersion(resourceVersion, configVersion string) bool {
	rv, err := strconv.ParseInt(resourceVersion, 10, 32)
	if err != nil {
		log.DefaultLogger().Reason(err).Errorf("Resource version is unable to be parsed")
		return false
	}

	crv, err := strconv.ParseInt(configVersion, 10, 32)
	if err != nil {
		log.DefaultLogger().Reason(err).Errorf("Config resource version is unable to be parsed")
		return false
	}

	if rv > crv {
		log.DefaultLogger().Errorf("Config is not in sync. Expected %s or greater, Got %s", resourceVersion, configVersion)
		return false
	}

	return true
}

func waitForConfigToBePropagated(resourceVersion string) {
	checkComponentVersions(virtHandlerComponentName, resourceVersion)
	checkComponentVersions(virtControllerComponentName, resourceVersion)
	checkComponentVersions(virtApiComponentName, resourceVersion)
}

func WaitForConfigToBePropagatedToComponent(podLabel string, resourceVersion string, compareResourceVersions compare, duration time.Duration) {
	virtClient := kubevirt.Client()

	errComponentInfo := fmt.Sprintf("component: \"%s\"", strings.TrimPrefix(podLabel, "kubevirt.io="))

	EventuallyWithOffset(3, func() error {
		pods, err := virtClient.CoreV1().Pods(flags.KubeVirtInstallNamespace).List(context.Background(), metav1.ListOptions{LabelSelector: podLabel})

		if err != nil {
			return fmt.Errorf("failed to fetch pods: %v, %s", err, errComponentInfo)
		}
		for _, pod := range pods.Items {
			errAdditionalInfo := errComponentInfo + fmt.Sprintf(", pod: \"%s\"", pod.Name)

			if pod.DeletionTimestamp != nil {
				continue
			}

			body, err := callUrlOnPod(&pod, "8443", "/healthz")
			if err != nil {
				return fmt.Errorf("failed to call healthz endpoint: %v, %s", err, errAdditionalInfo)
			}
			result := map[string]interface{}{}
			err = json.Unmarshal(body, &result)
			if err != nil {
				return fmt.Errorf("failed to parse response from healthz endpoint: %v, %s", err, errAdditionalInfo)
			}

			if configVersion := result["config-resource-version"].(string); !compareResourceVersions(resourceVersion, configVersion) {
				return fmt.Errorf("resource & config versions (%s and %s respectively) are not as expected. %s ",
					resourceVersion, configVersion, errAdditionalInfo)
			}
		}
		return nil
	}, duration, 1*time.Second).ShouldNot(HaveOccurred())
}

func checkComponentVersions(componentName string, resourceVersion string) {
	rv, err := strconv.ParseInt(resourceVersion, 10, 32)
	Expect(err).ToNot(HaveOccurred())

	virtClient := kubevirt.Client()

	EventuallyWithOffset(3, func(g Gomega) {
		kv := libkubevirt.GetCurrentKv(virtClient)
		g.Expect(kv.Status.ComponentVersions).ToNot(BeNil())
		var componentVersions map[string]string
		switch componentName {
		case virtHandlerComponentName:
			componentVersions = kv.Status.ComponentVersions.VirtHandler
		case virtControllerComponentName:
			componentVersions = kv.Status.ComponentVersions.VirtController
		case virtApiComponentName:
			componentVersions = kv.Status.ComponentVersions.VirtApi
		}
		g.Expect(componentVersions).ToNot(BeNil())

		for _, version := range componentVersions {
			crv, err := strconv.ParseInt(version, 10, 32)
			g.Expect(err).ToNot(HaveOccurred())
			g.Expect(crv).To(BeNumerically(">=", rv))
		}
	}, 60*time.Second, 1*time.Second).Should(Succeed())
}

func callUrlOnPod(pod *k8sv1.Pod, port string, url string) ([]byte, error) {
	randPort := strconv.Itoa(4321 + rand.Intn(6000))
	stopChan := make(chan struct{})
	defer close(stopChan)
	err := libpod.ForwardPorts(pod, []string{fmt.Sprintf("%s:%s", randPort, port)}, stopChan, 5*time.Second)
	if err != nil {
		return nil, err
	}
	tr := &http.Transport{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: true, VerifyPeerCertificate: func(_ [][]byte, _ [][]*x509.Certificate) error {
			return nil
		}},
	}
	client := &http.Client{Transport: tr}
	resp, err := client.Get(fmt.Sprintf("https://localhost:%s/%s", randPort, strings.TrimSuffix(url, "/")))
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	return io.ReadAll(resp.Body)
}
