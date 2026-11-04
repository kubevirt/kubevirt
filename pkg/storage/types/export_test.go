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

package types

import (
	"fmt"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

var _ = Describe("ExportServiceDialPort", func() {
	DescribeTable("selects the dial port from the Service", func(clusterIP string, expected int32) {
		svc := &corev1.Service{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "virt-export-test",
				Namespace: "ns",
			},
			Spec: corev1.ServiceSpec{
				ClusterIP: clusterIP,
			},
		}
		Expect(ExportServiceDialPort(svc)).To(Equal(expected))
		Expect(ExportServiceHost(svc)).To(Equal(fmt.Sprintf("virt-export-test.ns.svc:%d", expected)))
	},
		Entry("headless Service uses the container port", corev1.ClusterIPNone, int32(ExportServerPort)),
		Entry("ClusterIP Service uses the historical Service port", "10.96.0.10", int32(ExportClusterIPServicePort)),
		Entry("unallocated ClusterIP still uses the Service port", "", int32(ExportClusterIPServicePort)),
	)

	It("treats a nil Service as headless", func() {
		Expect(ExportServiceDialPort(nil)).To(Equal(int32(ExportServerPort)))
	})
})
