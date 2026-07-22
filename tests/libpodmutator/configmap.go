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
 * Copyright The KubeVirt Authors.
 *
 */

package libpodmutator

import (
	"context"

	. "github.com/onsi/gomega"
	k8sv1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"kubevirt.io/client-go/kubecli"

	"kubevirt.io/kubevirt/tests/testsuite"
)

// CreateOrUpdateEnvConfigMap creates or updates a ConfigMap whose keys become environment
// variables. The ConfigMap is created in namespace; when empty, testsuite.GetTestNamespace(nil) is used.
func CreateOrUpdateEnvConfigMap(virtClient kubecli.KubevirtClient, name string, data map[string]string, namespace ...string) {
	testNamespace := testsuite.GetTestNamespace(nil)
	if len(namespace) > 0 && namespace[0] != "" {
		testNamespace = namespace[0]
	}
	configMap := &k8sv1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: testNamespace,
		},
		Data: data,
	}
	_, err := virtClient.CoreV1().ConfigMaps(testNamespace).Create(context.Background(), configMap, metav1.CreateOptions{})
	// if already exists, update instead
	if errors.IsAlreadyExists(err) {
		existing, getErr := virtClient.CoreV1().ConfigMaps(testNamespace).Get(context.Background(), name, metav1.GetOptions{})
		Expect(getErr).ToNot(HaveOccurred())
		existing.Data = data
		_, err = virtClient.CoreV1().ConfigMaps(testNamespace).Update(context.Background(), existing, metav1.UpdateOptions{})
	}
	Expect(err).ToNot(HaveOccurred())
}

// DeleteEnvConfigMap removes a ConfigMap used for env injection.
func DeleteEnvConfigMap(virtClient kubecli.KubevirtClient, name string, namespace ...string) {
	testNamespace := testsuite.GetTestNamespace(nil)
	if len(namespace) > 0 && namespace[0] != "" {
		testNamespace = namespace[0]
	}
	err := virtClient.CoreV1().ConfigMaps(testNamespace).Delete(context.Background(), name, metav1.DeleteOptions{})
	if !errors.IsNotFound(err) {
		Expect(err).ToNot(HaveOccurred())
	}
}
