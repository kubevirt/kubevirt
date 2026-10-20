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

package snapshotoverlay

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	k8sv1 "k8s.io/api/core/v1"

	v1 "kubevirt.io/api/core/v1"

	"kubevirt.io/kubevirt/pkg/libvmi"
	"kubevirt.io/kubevirt/pkg/pointer"
	storagetypes "kubevirt.io/kubevirt/pkg/storage/types"
)

var _ = Describe("Overlay recovery", func() {
	DescribeTable("should hold the start of a VMI until the scratch volume it recovers is in its spec", func(annotation string, volume *v1.UtilityVolume, pending bool) {
		vmi := libvmi.New()
		if annotation != "" {
			vmi.Annotations = map[string]string{storagetypes.PendingOverlayRecoveryAnnotation: annotation}
		}
		if volume != nil {
			vmi.Spec.UtilityVolumes = []v1.UtilityVolume{*volume}
		}
		Expect(AwaitingRecoveryScratch(vmi)).To(Equal(pending))
	},
		Entry("nothing to recover", "", nil, false),
		Entry("scratch not attached yet", "snap-scratch-content", nil, true),
		Entry("scratch attached", "snap-scratch-content", &v1.UtilityVolume{
			Name: "snap-scratch-content", PersistentVolumeClaimVolumeSource: k8sv1.PersistentVolumeClaimVolumeSource{ClaimName: "snap-scratch-content"}, Type: pointer.P(v1.SnapshotOverlay),
		}, false),
		Entry("another volume under the scratch name", "snap-scratch-content", &v1.UtilityVolume{
			Name: "snap-scratch-content", PersistentVolumeClaimVolumeSource: k8sv1.PersistentVolumeClaimVolumeSource{ClaimName: "other"}, Type: pointer.P(v1.SnapshotOverlay),
		}, true),
	)
})
