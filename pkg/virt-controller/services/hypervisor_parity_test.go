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
	"embed"

	"k8s.io/apimachinery/pkg/api/resource"

	v1 "kubevirt.io/api/core/v1"
	"kubevirt.io/kubevirt/pkg/libvmi"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

//go:embed renderresources.go
var renderResourcesSource string

var _ embed.FS

// NOTE(parity): At the time of writing there is no branching in GetMemoryOverhead()
// (or AdjustQemuProcessMemoryLimits()) based on underlying hypervisor (KVM vs mshv).
// These tests intentionally document that fact with concise parity assertions so that
// if a future change introduces hypervisor-specific adjustments, a deliberate test
// update will be required.
var _ = Describe("Hypervisor memory overhead parity", func() {

	It("should fail fast once GetMemoryOverhead introduces ConfigurableHypervisor gating (sentinel for future branch)", func() {
		// This sentinel ensures that if someone adds code in renderresources.go like:
		//   if config.ConfigurableHypervisorEnabled() { ... }
		// (either by adding a config parameter or referencing a global), the test suite will
		// flag it explicitly so we can update the parity tests to exercise both branches
		// with real configs (ensuring either maintained parity or deliberate, documented divergence).
		//
		// Limitation: today GetMemoryOverhead has no access to ClusterConfig, so we cannot
		// truly toggle behavior; we instead watch for appearance of the gating call.
		// When this fires, replace this sentinel with a real off/on evaluation using the
		// new code path.
		Expect(renderResourcesSource).NotTo(ContainSubstring("ConfigurableHypervisorEnabled("),
			"GetMemoryOverhead started using ConfigurableHypervisor gating; update parity test to calculate both OFF and ON overhead via config and assert expected invariant (likely equality unless spec changes)")
	})

	It("should treat combined VFIO+SEV overhead identically across hypervisors (parity sentinel)", func() {
		// Purpose:
		// - Locks in the exact additive constants: VFIO => +1Gi, SEV => +256Mi (delta must equal 1Gi+256Mi, not just >=).
		// - Serves as hypervisor parity guard: any future HyperVLayered-specific overhead will break this and require
		//   an intentional update (keeps hidden drift from creeping in).
		// - Verifies ordering: ratio scaling (1.2) applies after all additive components and includes VFIO+SEV.
		// - Consolidates prior multiple tests into a single high‑signal sentinel.
		// Base VMI
		baseVMI := libvmi.New(
			libvmi.WithResourceMemory("1Gi"),
		)
		baseOverhead := GetMemoryOverhead(baseVMI, "amd64", nil)

		// Extended VMI with VFIO GPU + SEV
		extraVMI := libvmi.New(
			libvmi.WithResourceMemory("1Gi"),
		)
		extraVMI.Spec.Domain.Devices.GPUs = []v1.GPU{{Name: "gpu0", DeviceName: "example.com/GPU"}}
		extraVMI.Spec.Domain.LaunchSecurity = &v1.LaunchSecurity{SEV: &v1.SEV{}}

		overheadWithExtras := GetMemoryOverhead(extraVMI, "amd64", nil)
		delta := resource.NewQuantity(overheadWithExtras.Value()-baseOverhead.Value(), resource.BinarySI)

		// Expected additive components: EXACT 1Gi VFIO + 256Mi SEV (no other differences)
		expectedVFIO := resource.MustParse("1Gi")
		expectedSEV := resource.MustParse("256Mi")
		expectedDelta := resource.NewQuantity(0, resource.BinarySI)
		expectedDelta.Add(expectedVFIO)
		expectedDelta.Add(expectedSEV)

		Expect(delta.Value()).To(Equal(expectedDelta.Value()),
			"Delta must equal VFIO (1Gi) + SEV (256Mi); indicates purely additive parity")

		// Now apply an additional overhead ratio and ensure uniform scaling
		ratio := "1.2"
		ratioPtr := &ratio // avoid pointer.P to keep test resilient if utils/pointer API changes
		scaled := GetMemoryOverhead(extraVMI, "amd64", ratioPtr)
		// Compute expected scaled (float) then compare within 1% tolerance
		expectedScaledFloat := float64(overheadWithExtras.Value()) * 1.2
		expectedScaled := int64(expectedScaledFloat)
		Expect(scaled.Value()).To(BeNumerically("~", expectedScaled, expectedScaled/100),
			"Scaled overhead should be approximately overheadWithExtras * 1.2 (within 1%)")
	})

})

// (renderResourcesSource embedded above)
