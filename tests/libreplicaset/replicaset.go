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

package libreplicaset

import (
	"context"
	"fmt"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	autov1 "k8s.io/api/autoscaling/v1"
	v12 "k8s.io/apimachinery/pkg/apis/meta/v1"

	v1 "kubevirt.io/api/core/v1"
	"kubevirt.io/client-go/kubecli"

	"kubevirt.io/kubevirt/tests/testsuite"
)

func DoScaleWithScaleSubresource(virtClient kubecli.KubevirtClient, name string, scale int32) {
	// Status updates can conflict with our desire to change the spec
	By(fmt.Sprintf("Scaling to %d", scale))
	var s *autov1.Scale
	s, err := virtClient.ReplicaSet(testsuite.GetTestNamespace(nil)).GetScale(context.Background(), name, v12.GetOptions{})
	ExpectWithOffset(1, err).ToNot(HaveOccurred())
	s.ResourceVersion = "" // Indicate the scale update should be unconditional
	s.Spec.Replicas = scale
	s, err = virtClient.ReplicaSet(testsuite.GetTestNamespace(nil)).UpdateScale(context.Background(), name, s)
	Expect(err).ToNot(HaveOccurred())

	ExpectWithOffset(1, err).ToNot(HaveOccurred())

	By("Checking the number of replicas")
	EventuallyWithOffset(1, func() int32 {
		s, err = virtClient.ReplicaSet(testsuite.GetTestNamespace(nil)).GetScale(context.Background(), name, v12.GetOptions{})
		Expect(err).ToNot(HaveOccurred())
		return s.Status.Replicas
	}, 90*time.Second, time.Second).Should(Equal(scale))

	vmis, err := virtClient.VirtualMachineInstance(testsuite.GetTestNamespace(nil)).List(context.Background(), v12.ListOptions{})
	ExpectWithOffset(1, err).ToNot(HaveOccurred())
	ExpectWithOffset(1, FilterNotDeletedVMIs(vmis)).To(HaveLen(int(scale)))
}

func FilterNotDeletedVMIs(vmis *v1.VirtualMachineInstanceList) []v1.VirtualMachineInstance {
	var notDeleted []v1.VirtualMachineInstance
	for _, vmi := range vmis.Items {
		if vmi.DeletionTimestamp == nil {
			notDeleted = append(notDeleted, vmi)
		}
	}
	return notDeleted
}
