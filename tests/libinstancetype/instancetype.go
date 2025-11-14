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
//nolint:lll
package libinstancetype

import (
	"context"

	. "github.com/onsi/gomega"

	"k8s.io/apimachinery/pkg/api/equality"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"

	"kubevirt.io/kubevirt/tests/testsuite"
)

func EnsureControllerRevisionObjectsEqual(crNameA, crNameB string, k8sClient kubernetes.Interface) bool {
	crA, err := k8sClient.AppsV1().ControllerRevisions(testsuite.GetTestNamespace(nil)).Get(context.Background(), crNameA, metav1.GetOptions{})
	Expect(err).ToNot(HaveOccurred())
	crB, err := k8sClient.AppsV1().ControllerRevisions(testsuite.GetTestNamespace(nil)).Get(context.Background(), crNameB, metav1.GetOptions{})
	Expect(err).ToNot(HaveOccurred())
	return equality.Semantic.DeepEqual(crA.Data.Object, crB.Data.Object)
}
