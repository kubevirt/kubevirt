/*
Package defaults contains Tier 1 tests for CPU hotplug MaxSockets calculation.

STP Reference: tests/CNV-61263/CNV-61263_test_plan.md
STD Reference: tests/CNV-61263/CNV-61263_test_description.yaml

Jira: CNV-61263 - [CLOSED LOOP for] CPU hotplug logic still going over the limits
Source Bug: CNV-57352, CNV-48124

Related PRs:
  - kubevirt/kubevirt#14338: defaults: Limit MaxSockets based on maximum of vcpus
  - kubevirt/kubevirt#14511: [release-1.5] Cherry-pick

This test suite validates the MaxSockets calculation fix that prevents vCPU count
from exceeding 512 when CPU hotplug is enabled. The fix modifies setupCPUHotplug()
in pkg/defaults/defaults.go to cap MaxSockets appropriately.

The fix formula:
  - If totalVCPUs (maxSockets * cores * threads) > 512:
    - adjustedSockets = 512 / (cores * threads)
    - maxSockets = max(adjustedSockets, configuredSockets)
*/
package defaults

import (
	"context"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	v1 "kubevirt.io/api/core/v1"
	"kubevirt.io/kubevirt/pkg/defaults"
	virtconfig "kubevirt.io/kubevirt/pkg/virt-config"
	"kubevirt.io/kubevirt/tests/decorators"
	"kubevirt.io/kubevirt/tests/framework/kubevirt"
)

// defaultMaxHotplugRatio is the expected default ratio used for CPU hotplug.
// Tests use this constant to ensure deterministic behavior regardless of cluster config.
const defaultMaxHotplugRatio = 4

var _ = Describe("[sig-compute]CPU Hotplug MaxSockets Calculation", decorators.SigCompute, Serial, func() {
	/*
	 * Test suite for CPU hotplug MaxSockets calculation fix.
	 *
	 * The CPU hotplug feature reserves additional sockets for future hotplugging
	 * by multiplying configured sockets by 4 (the default MaxHotplugRatio).
	 * This can cause the total vCPU count to exceed machine type limits.
	 *
	 * The fix (PR #14338) caps MaxSockets so total vCPUs don't exceed 512.
	 *
	 * Note: These tests use the cluster's live configuration. For deterministic
	 * behavior in all environments, consider constructing a dedicated ClusterConfig
	 * with an explicit MaxHotplugRatio if flakiness is observed.
	 */

	var (
		clusterConfig *virtconfig.ClusterConfig
	)

	BeforeEach(func() {
		// Get cluster configuration with default MaxHotplugRatio of 4
		virtClient := kubevirt.Client()
		kv, err := virtClient.KubeVirt(metav1.NamespaceAll).List(context.Background(), metav1.ListOptions{})
		ExpectWithOffset(1, err).ToNot(HaveOccurred())
		ExpectWithOffset(1, kv.Items).ToNot(BeEmpty())

		clusterConfig, err = virtconfig.NewClusterConfig(
			nil, nil, nil, kv.Items[0].Namespace,
		)
		ExpectWithOffset(1, err).ToNot(HaveOccurred())

		// Verify the MaxHotplugRatio matches our expected value for deterministic tests
		// If this assertion fails, the cluster has a non-default ratio configured
		ExpectWithOffset(1, clusterConfig.GetMaxHotplugRatio()).To(Equal(uint32(defaultMaxHotplugRatio)),
			"Tests expect MaxHotplugRatio of %d; cluster has different configuration", defaultMaxHotplugRatio)
	})

	Context("when CPU topology would exceed 512 vCPU limit", Ordered, func() {
		/*
		 * Test ID: TS-CNV61263-003
		 * Tier: 1
		 * Priority: P1
		 *
		 * Test case: VMI with 32 sockets, 2 cores, 3 threads
		 * - Base vCPUs: 32 * 2 * 3 = 192
		 * - Default 4x ratio: 128 sockets * 2 * 3 = 768 vCPUs (exceeds 512)
		 * - Expected MaxSockets: 85 (512 / 6 = 85.33, truncated to 85)
		 * - Total potential vCPUs: 85 * 2 * 3 = 510 (under 512)
		 */
		It("[test_id:TS-CNV61263-003] should calculate MaxSockets with upper bound 512 when topology exceeds limit", func() {
			By("Creating VMI spec with high CPU topology (32 sockets, 2 cores, 3 threads)")
			vmi := &v1.VirtualMachineInstance{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-vmi-high-cpu",
					Namespace: "default",
				},
				Spec: v1.VirtualMachineInstanceSpec{
					Domain: v1.DomainSpec{
						CPU: &v1.CPU{
							Sockets: 32,
							Cores:   2,
							Threads: 3,
						},
					},
				},
			}

			By("Calling SetupCPUHotplug to calculate MaxSockets")
			defaults.SetupCPUHotplug(clusterConfig, vmi)

			By("Verifying MaxSockets is capped to 85")
			// Expected calculation:
			// - Default: 32 * 4 = 128 sockets
			// - Total vCPUs: 128 * 2 * 3 = 768 (exceeds 512)
			// - Adjusted: 512 / (2 * 3) = 85.33 -> 85 sockets
			// - Final: max(85, 32) = 85
			ExpectWithOffset(1, vmi.Spec.Domain.CPU.MaxSockets).To(Equal(uint32(85)),
				"MaxSockets should be capped to 85 (512 / 6 = 85)")

			By("Verifying total potential vCPUs stays under 512")
			totalVCPUs := vmi.Spec.Domain.CPU.MaxSockets * vmi.Spec.Domain.CPU.Cores * vmi.Spec.Domain.CPU.Threads
			ExpectWithOffset(1, totalVCPUs).To(BeNumerically("<=", uint32(512)),
				"Total potential vCPUs (%d) should not exceed 512", totalVCPUs)

			By("Verifying MaxSockets is at least the configured sockets")
			ExpectWithOffset(1, vmi.Spec.Domain.CPU.MaxSockets).To(BeNumerically(">=", uint32(32)),
				"MaxSockets should be at least the configured sockets (32)")
		})
	})

	Context("when CPU topology is within normal limits", Ordered, func() {
		/*
		 * Test ID: TS-CNV61263-006
		 * Tier: 1
		 * Priority: P1
		 *
		 * Regression test: VMI with 4 sockets, 2 cores, 1 thread
		 * - Base vCPUs: 4 * 2 * 1 = 8
		 * - Default 4x ratio: 16 sockets * 2 * 1 = 32 vCPUs (under 512)
		 * - Expected MaxSockets: 16 (standard 4x ratio, no capping needed)
		 *
		 * This ensures the fix doesn't break standard hotplug behavior.
		 */
		It("[test_id:TS-CNV61263-006] should calculate MaxSockets as 4x configured sockets for standard topology", func() {
			By("Creating VMI spec with standard CPU topology (4 sockets, 2 cores, 1 thread)")
			vmi := &v1.VirtualMachineInstance{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-vmi-standard-cpu",
					Namespace: "default",
				},
				Spec: v1.VirtualMachineInstanceSpec{
					Domain: v1.DomainSpec{
						CPU: &v1.CPU{
							Sockets: 4,
							Cores:   2,
							Threads: 1,
						},
					},
				},
			}

			By("Calling setupCPUHotplug to calculate MaxSockets")
			defaults.SetupCPUHotplug(clusterConfig, vmi)

			By("Verifying MaxSockets is 16 (standard 4x ratio)")
			// Expected calculation:
			// - Default: 4 * 4 = 16 sockets
			// - Total vCPUs: 16 * 2 * 1 = 32 (under 512, no capping)
			// - Final: 16 sockets
			ExpectWithOffset(1, vmi.Spec.Domain.CPU.MaxSockets).To(Equal(uint32(16)),
				"MaxSockets should be 16 (4 * 4 = 16, standard ratio)")

			By("Verifying total potential vCPUs is 32")
			totalVCPUs := vmi.Spec.Domain.CPU.MaxSockets * vmi.Spec.Domain.CPU.Cores * vmi.Spec.Domain.CPU.Threads
			ExpectWithOffset(1, totalVCPUs).To(Equal(uint32(32)),
				"Total potential vCPUs should be 32 (16 * 2 * 1)")

			By("Verifying no capping was applied (under 512 limit)")
			ExpectWithOffset(1, totalVCPUs).To(BeNumerically("<", uint32(512)),
				"Standard topology should not trigger capping")
		})
	})

	Context("edge cases for MaxSockets calculation", Ordered, func() {
		/*
		 * Additional edge case tests for completeness.
		 * These verify boundary conditions of the 512 vCPU limit.
		 */

		It("should keep 4x ratio at exactly 512 vCPUs (no additional capping needed)", func() {
			By("Creating VMI spec that results in exactly 512 vCPUs with 4x ratio")
			// Configuration: 32 sockets * 4 = 128 maxSockets
			// 128 sockets * 2 cores * 2 threads = 512 vCPUs (exactly at limit)
			// At this boundary, the 4x ratio naturally produces 512 vCPUs,
			// so no additional capping is required.
			vmi := &v1.VirtualMachineInstance{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-vmi-boundary",
					Namespace: "default",
				},
				Spec: v1.VirtualMachineInstanceSpec{
					Domain: v1.DomainSpec{
						CPU: &v1.CPU{
							Sockets: 32, // 32 * 4 = 128 maxSockets
							Cores:   2,
							Threads: 2,
						},
					},
				},
			}

			By("Calling SetupCPUHotplug")
			defaults.SetupCPUHotplug(clusterConfig, vmi)

			By("Verifying MaxSockets is 128 (full 4x ratio retained at 512 vCPU boundary)")
			// 32 requested sockets * 4x over-provisioning = 128 sockets
			// 128 sockets * 2 cores * 2 threads = 512 vCPUs (exactly at the 512-vCPU limit)
			// At the boundary we still allow the full 4x ratio (no additional capping),
			// so MaxSockets is expected to remain 128.
			ExpectWithOffset(1, vmi.Spec.Domain.CPU.MaxSockets).To(Equal(uint32(128)),
				"MaxSockets should retain the 4x ratio (128) at exactly 512 vCPUs")

			By("Verifying total vCPUs equals exactly 512")
			totalVCPUs := vmi.Spec.Domain.CPU.MaxSockets * vmi.Spec.Domain.CPU.Cores * vmi.Spec.Domain.CPU.Threads
			ExpectWithOffset(1, totalVCPUs).To(Equal(uint32(512)),
				"Total potential vCPUs should be exactly 512")

			By("Verifying MaxSockets is at least the configured sockets")
			ExpectWithOffset(1, vmi.Spec.Domain.CPU.MaxSockets).To(BeNumerically(">=", uint32(32)),
				"MaxSockets should be at least the configured sockets (32)")
		})

		It("should preserve explicitly set MaxSockets", func() {
			By("Creating VMI spec with explicit MaxSockets already set")
			vmi := &v1.VirtualMachineInstance{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-vmi-explicit-max",
					Namespace: "default",
				},
				Spec: v1.VirtualMachineInstanceSpec{
					Domain: v1.DomainSpec{
						CPU: &v1.CPU{
							Sockets:    216,
							Cores:      1,
							Threads:    1,
							MaxSockets: 2, // Explicitly set by user
						},
					},
				},
			}

			By("Calling setupCPUHotplug")
			defaults.SetupCPUHotplug(clusterConfig, vmi)

			By("Verifying explicit MaxSockets is preserved")
			// When MaxSockets is already set (non-zero), setupCPUHotplug should not override
			ExpectWithOffset(1, vmi.Spec.Domain.CPU.MaxSockets).To(Equal(uint32(2)),
				"Explicit MaxSockets should be preserved")
		})
	})
})
