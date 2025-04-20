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
 */

package checks

import (
	"context"
	"fmt"

	"github.com/onsi/ginkgo/v2"
	"github.com/onsi/gomega"

	k8sv1 "k8s.io/api/core/v1"
	"k8s.io/apiextensions-apiserver/pkg/client/clientset/clientset"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"kubevirt.io/client-go/kubecli"
)

// Deprecated: SkipTestIfNoFeatureGate should be converted to check & fail
func SkipTestIfNoFeatureGate(featureGate string) {
	if !HasFeature(featureGate) {
		ginkgo.Skip(fmt.Sprintf("the %v feature gate is not enabled.", featureGate))
	}
}

func RecycleImageOrFail(virtClient kubecli.KubevirtClient, imageName string) {
	windowsPv, err := virtClient.CoreV1().PersistentVolumes().Get(context.Background(), imageName, metav1.GetOptions{})
	if err != nil || windowsPv.Status.Phase == k8sv1.VolumePending || windowsPv.Status.Phase == k8sv1.VolumeFailed {
		ginkgo.Fail(fmt.Sprintf("Skip tests that requires PV %s", imageName))
	} else if windowsPv.Status.Phase == k8sv1.VolumeReleased {
		windowsPv.Spec.ClaimRef = nil
		_, err = virtClient.CoreV1().PersistentVolumes().Update(context.Background(), windowsPv, metav1.UpdateOptions{})
		gomega.Expect(err).ToNot(gomega.HaveOccurred())
	}
}

// Deprecated: SkipIfUseFlannel should be converted to check & fail
func SkipIfUseFlannel(virtClient kubecli.KubevirtClient) {
	labelSelector := "app=flannel"
	flannelpod, err := virtClient.CoreV1().Pods(metav1.NamespaceSystem).List(context.Background(), metav1.ListOptions{LabelSelector: labelSelector})
	gomega.Expect(err).ToNot(gomega.HaveOccurred())
	if len(flannelpod.Items) > 0 {
		ginkgo.Skip("Skip networkpolicy test for flannel network")
	}
}

// Deprecated: SkipIfPrometheusRuleIsNotEnabled should be converted to check & fail
func SkipIfPrometheusRuleIsNotEnabled(virtClient kubecli.KubevirtClient) {
	ext, err := clientset.NewForConfig(virtClient.Config())
	gomega.Expect(err).ToNot(gomega.HaveOccurred())

	_, err = ext.ApiextensionsV1().CustomResourceDefinitions().Get(context.Background(), "prometheusrules.monitoring.coreos.com", metav1.GetOptions{})
	if errors.IsNotFound(err) {
		ginkgo.Skip("Skip monitoring tests when PrometheusRule CRD is not available in the cluster")
	} else if err != nil {
		gomega.Expect(err).ToNot(gomega.HaveOccurred())
	}
}

// Deprecated: SkipIfS390X should be converted to check & fail
func SkipIfS390X(arch string, message string) {
	if IsS390X(arch) {
		ginkgo.Skip("Skip test on s390x: " + message)
	}
}

// Deprecated: SkipIfRunningOnKindInfra should be converted to check & fail
func SkipIfRunningOnKindInfra(message string) {
	if IsRunningOnKindInfra() {
		ginkgo.Skip("Skip test on kind infra: " + message)
	}
}
