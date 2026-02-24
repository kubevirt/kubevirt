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

package components

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"

	operatorutil "kubevirt.io/kubevirt/pkg/virt-operator/util"
)

var _ = Describe("DaemonSets", func() {

	Context("virt-handler with custom KubeletRootDir", func() {
		It("should use custom kubelet root path in volumes when KubeletRootDir is set", func() {
			customKubeletRoot := "/custom/var/lib/kubelet"
			config := &operatorutil.KubeVirtDeploymentConfig{
				Registry:        "reg.io/kubevirt",
				KubeVirtVersion: "v1.2.3",
				Namespace:       "kubevirt",
				KubeletRootDir:  customKubeletRoot,
			}

			ds := NewHandlerDaemonSet(config, "", "", "")

			var kubeletVol, kubeletPodsVol *corev1.Volume
			for i := range ds.Spec.Template.Spec.Volumes {
				v := &ds.Spec.Template.Spec.Volumes[i]
				switch v.Name {
				case "kubelet":
					kubeletVol = v
				case "kubelet-pods":
					kubeletPodsVol = v
				}
			}
			Expect(kubeletVol).ToNot(BeNil(), "kubelet volume should exist")
			Expect(kubeletPodsVol).ToNot(BeNil(), "kubelet-pods volume should exist")
			Expect(kubeletVol.VolumeSource.HostPath).ToNot(BeNil())
			Expect(kubeletVol.VolumeSource.HostPath.Path).To(Equal(customKubeletRoot))
			Expect(kubeletPodsVol.VolumeSource.HostPath).ToNot(BeNil())
			Expect(kubeletPodsVol.VolumeSource.HostPath.Path).To(Equal(customKubeletRoot + "/pods"))
		})
	})
})
