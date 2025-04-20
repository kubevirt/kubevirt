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

package services

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	v1 "kubevirt.io/api/core/v1"
	"kubevirt.io/client-go/api"

	"kubevirt.io/kubevirt/pkg/testutils"
	"kubevirt.io/kubevirt/pkg/virt-config/featuregate"
)

var _ = Describe("virtiofs container", func() {

	kv := &v1.KubeVirt{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "kubevirt",
			Namespace: "kubevirt",
		},
		Spec: v1.KubeVirtSpec{
			Configuration: v1.KubeVirtConfiguration{
				DeveloperConfiguration: &v1.DeveloperConfiguration{},
			},
		},
		Status: v1.KubeVirtStatus{
			Phase:               v1.KubeVirtPhaseDeploying,
			DefaultArchitecture: "amd64",
		},
	}
	config, _, kvStore := testutils.NewFakeClusterConfigUsingKV(kv)

	enableFeatureGate := func(featureGate string) {
		testutils.UpdateFakeKubeVirtClusterConfig(kvStore, &v1.KubeVirt{
			Spec: v1.KubeVirtSpec{
				Configuration: v1.KubeVirtConfiguration{
					DeveloperConfiguration: &v1.DeveloperConfiguration{
						FeatureGates: []string{featureGate},
					},
				},
			},
		})
	}

	disableFeatureGates := func() {
		testutils.UpdateFakeKubeVirtClusterConfig(kvStore, kv)
	}

	BeforeEach(func() {
		enableFeatureGate(featuregate.VirtIOFSStorageVolumeGate)
		enableFeatureGate(featuregate.VirtIOFSConfigVolumesGate)
	})

	AfterEach(func() {
		disableFeatureGates()
	})

	It("should create unprivileged containers only", func() {
		vmi := api.NewMinimalVMI("testvm")

		vmi.Spec.Volumes = append(vmi.Spec.Volumes, v1.Volume{
			Name: "sharedtestdisk",
			VolumeSource: v1.VolumeSource{
				PersistentVolumeClaim: testutils.NewFakePersistentVolumeSource(),
			},
		})
		vmi.Spec.Domain.Devices.Filesystems = append(vmi.Spec.Domain.Devices.Filesystems, v1.Filesystem{
			Name:     "sharedtestdisk",
			Virtiofs: &v1.FilesystemVirtiofs{},
		})

		vmi.Spec.Volumes = append(vmi.Spec.Volumes, v1.Volume{
			Name: "secret-volume",
			VolumeSource: v1.VolumeSource{
				Secret: &v1.SecretVolumeSource{
					SecretName: "test-secret",
				},
			},
		})
		vmi.Spec.Domain.Devices.Filesystems = append(vmi.Spec.Domain.Devices.Filesystems, v1.Filesystem{
			Name:     "secret-volume",
			Virtiofs: &v1.FilesystemVirtiofs{},
		})

		container := generateVirtioFSContainers(vmi, "virtiofs-container", config)
		Expect(container).To(HaveLen(2))

		// PV
		Expect(container[0].SecurityContext.RunAsNonRoot).To(HaveValue(BeTrue()))
		Expect(container[0].SecurityContext.AllowPrivilegeEscalation).To(HaveValue(BeFalse()))
		// Secret
		Expect(container[1].SecurityContext.RunAsNonRoot).To(HaveValue(BeTrue()))
		Expect(container[1].SecurityContext.AllowPrivilegeEscalation).To(HaveValue(BeFalse()))
	})
})
