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
 * Copyright The KubeVirt Authors.
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

	. "github.com/onsi/gomega"

	k8sv1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"

	v1 "kubevirt.io/api/core/v1"
	"kubevirt.io/client-go/log"

	"kubevirt.io/kubevirt/pkg/apimachinery/patch"
	"kubevirt.io/kubevirt/tests/flags"
	"kubevirt.io/kubevirt/tests/framework/kubevirt"
	"kubevirt.io/kubevirt/tests/libkubevirt"
	"kubevirt.io/kubevirt/tests/libpod"
	"kubevirt.io/kubevirt/tests/testsuite"
)

type compare func(string, string) bool

type KvChangeOption func(kv *v1.KubeVirt) *patch.PatchSet

func RegisterKubevirtConfigChange(kvChangeOption ...KvChangeOption) error {
	kv := libkubevirt.GetCurrentKv(kubevirt.Client())
	var patches []patch.PatchOperation

	for _, c := range kvChangeOption {
		patches = append(patches, c(kv).GetPatches()...)
	}

	if len(patches) == 0 {
		return nil
	}

	patchData, err := patch.GeneratePatchPayload(patches...)
	if err != nil {
		return err
	}

	_, err = kubevirt.Client().KubeVirt(kv.Namespace).Patch(
		context.Background(),
		kv.Name,
		types.JSONPatchType,
		patchData,
		metav1.PatchOptions{},
	)
	return err
}

// UpdateKubeVirtConfigValueAndWait updates the given configuration in the kubevirt custom resource
// and then waits  to allow the configuration events to be propagated to the consumers.
func UpdateKubeVirtConfigValueAndWait(kvConfig v1.KubeVirtConfiguration) *v1.KubeVirt {
	kv := testsuite.UpdateKubeVirtConfigValue(kvConfig)

	testsuite.EnsureKubevirtReady()
	waitForConfigToBePropagated(kv.ResourceVersion)
	log.DefaultLogger().Infof("system is in sync with kubevirt config resource version %s", kv.ResourceVersion)

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
	waitTimeout := 10 * time.Second
	WaitForConfigToBePropagatedToComponent("kubevirt.io=virt-controller",
		resourceVersion, ExpectResourceVersionToBeLessEqualThanConfigVersion, waitTimeout)
	WaitForConfigToBePropagatedToComponent("kubevirt.io=virt-api",
		resourceVersion, ExpectResourceVersionToBeLessEqualThanConfigVersion, waitTimeout)
	WaitForConfigToBePropagatedToComponent("kubevirt.io=virt-handler",
		resourceVersion, ExpectResourceVersionToBeLessEqualThanConfigVersion, waitTimeout)
}

func WaitForConfigToBePropagatedToComponent(podLabel, resourceVersion string, compareResourceVersions compare, duration time.Duration) {
	virtClient := kubevirt.Client()

	errComponentInfo := fmt.Sprintf("component: %q", strings.TrimPrefix(podLabel, "kubevirt.io="))

	EventuallyWithOffset(3, func() error {
		pods, err := virtClient.CoreV1().Pods(flags.KubeVirtInstallNamespace).List(
			context.Background(), metav1.ListOptions{LabelSelector: podLabel})
		if err != nil {
			return fmt.Errorf("failed to fetch pods: %v, %s", err, errComponentInfo)
		}
		for i := range pods.Items {
			errAdditionalInfo := errComponentInfo + fmt.Sprintf(", pod: %q", pods.Items[i].Name)

			if pods.Items[i].DeletionTimestamp != nil {
				continue
			}

			body, err := callURLOnPod(&pods.Items[i], "8443", "/healthz")
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

func callURLOnPod(pod *k8sv1.Pod, port, url string) ([]byte, error) {
	const minPort = 4321
	const maxPortIncrease = 6000
	//nolint:gosec
	randPort := strconv.Itoa(minPort + rand.Intn(maxPortIncrease))
	stopChan := make(chan struct{})
	defer close(stopChan)
	readyTimeout := 5 * time.Second
	err := libpod.ForwardPorts(pod, []string{fmt.Sprintf("%s:%s", randPort, port)}, stopChan, readyTimeout)
	if err != nil {
		return nil, err
	}
	tr := &http.Transport{
		//nolint:gosec
		TLSClientConfig: &tls.Config{InsecureSkipVerify: true, VerifyPeerCertificate: func(_ [][]byte, _ [][]*x509.Certificate) error {
			return nil
		}},
	}
	client := &http.Client{Transport: tr}
	req, err := http.NewRequestWithContext(
		context.Background(), "GET",
		fmt.Sprintf("https://localhost:%s/%s", randPort, strings.TrimSuffix(url, "/")), http.NoBody)
	if err != nil {
		return nil, err
	}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	return io.ReadAll(resp.Body)
}
