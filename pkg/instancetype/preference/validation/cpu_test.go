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

package validation_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	v1beta1 "kubevirt.io/api/instancetype/v1beta1"

	"kubevirt.io/kubevirt/pkg/instancetype/preference/validation"
	"kubevirt.io/kubevirt/pkg/pointer"
)

var _ = Describe("CheckSpreadCPUTopology", func() {
	It("should allow 1 vCPU with spread topology regardless of ratio", func() {
		instancetypeSpec := &v1beta1.VirtualMachineInstancetypeSpec{
			CPU: v1beta1.CPUInstancetype{Guest: 1},
		}
		preferenceSpec := &v1beta1.VirtualMachinePreferenceSpec{
			CPU: &v1beta1.CPUPreferences{
				PreferredCPUTopology: pointer.P(v1beta1.Spread),
			},
		}
		Expect(validation.CheckSpreadCPUTopology(instancetypeSpec, preferenceSpec)).To(Succeed())
	})
})
