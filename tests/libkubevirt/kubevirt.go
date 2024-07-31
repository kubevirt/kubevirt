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

package libkubevirt

import (
	"context"
	"time"

	"github.com/onsi/gomega"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	v1 "kubevirt.io/api/core/v1"
	"kubevirt.io/client-go/kubecli"
)

func GetCurrentKv(virtClient kubecli.KubevirtClient) *v1.KubeVirt {
	kvs := GetKvList(virtClient)
	gomega.Expect(kvs).To(gomega.HaveLen(1))
	return &kvs[0]
}

func GetKvList(virtClient kubecli.KubevirtClient) []v1.KubeVirt {
	var kvList *v1.KubeVirtList
	var err error

	gomega.Eventually(func() error {

		kvList, err = virtClient.KubeVirt(metav1.NamespaceAll).List(context.Background(), metav1.ListOptions{})

		return err
	}, 10*time.Second, 1*time.Second).ShouldNot(gomega.HaveOccurred())

	return kvList.Items
}
