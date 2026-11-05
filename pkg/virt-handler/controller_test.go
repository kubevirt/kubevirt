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

package virthandler

import (
	"os"
	"path/filepath"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	v1 "kubevirt.io/api/core/v1"

	diskutils "kubevirt.io/kubevirt/pkg/ephemeral-disk-utils"
	"kubevirt.io/kubevirt/pkg/hypervisor"
	"kubevirt.io/kubevirt/pkg/libvmi"
	"kubevirt.io/kubevirt/pkg/safepath"
	"kubevirt.io/kubevirt/pkg/testutils"
	"kubevirt.io/kubevirt/pkg/virt-launcher/virtwrap/api"
)

var _ = Describe("claimDeviceOwnership", func() {
	var controller *BaseController
	var path string

	BeforeEach(func() {
		diskutils.MockDefaultOwnershipManager()

		config, _, _ := testutils.NewFakeClusterConfigUsingKVConfig(&v1.KubeVirtConfiguration{})
		controller = &BaseController{
			clusterConfig:      config,
			hypervisorNodeInfo: hypervisor.NewHypervisorNodeInformation(""),
		}

		path = GinkgoT().TempDir()
	})

	Context("with a generic device", func() {
		const device = "device"
		BeforeEach(func() {
			Expect(os.Mkdir(filepath.Join(path, "dev"), 0o755)).To(Succeed())
			f, err := os.Create(filepath.Join(path, "dev", device))
			Expect(err).ToNot(HaveOccurred())
			Expect(f.Close()).To(Succeed())
		})

		It("should succeed", func() {
			p, err := safepath.JoinAndResolveWithRelativeRoot("/", path)
			Expect(err).ToNot(HaveOccurred())
			Expect(controller.claimDeviceOwnership(p, device)).To(Succeed())
		})

		DescribeTable("should return an error if the device doesn't exist", func(softEmulation bool) {
			kv := &v1.KubeVirtConfiguration{
				DeveloperConfiguration: &v1.DeveloperConfiguration{
					UseEmulation: softEmulation,
				},
			}
			config, _, _ := testutils.NewFakeClusterConfigUsingKVConfig(kv)
			controller.clusterConfig = config

			p, err := safepath.JoinAndResolveWithRelativeRoot("/", path)
			Expect(err).ToNot(HaveOccurred())
			Expect(controller.claimDeviceOwnership(p, "noexist")).To(HaveOccurred())
		},
			Entry("with software emulation enabled", true),
			Entry("with software emulation disable", false),
		)
	})

	DescribeTable("iwith kvm device not existing should", func(softEmulation bool) {
		kv := &v1.KubeVirtConfiguration{
			DeveloperConfiguration: &v1.DeveloperConfiguration{
				UseEmulation: softEmulation,
			},
		}
		config, _, _ := testutils.NewFakeClusterConfigUsingKVConfig(kv)
		controller.clusterConfig = config

		p, err := safepath.JoinAndResolveWithRelativeRoot("/", path)
		Expect(err).ToNot(HaveOccurred())
		err = controller.claimDeviceOwnership(p, "kvm")
		if softEmulation {
			Expect(err).NotTo(HaveOccurred())
		} else {
			Expect(err).To(HaveOccurred())
		}
	},
		Entry("failed with software emulation enabled", true),
		Entry("succeed with software emulation disable", false),
	)
})

var _ = Describe("migrationRole", func() {
	const (
		host       = "thisnode"
		otherHost  = "othernode"
		testVMName = "testvmi"
	)

	newVMI := func(options ...libvmi.Option) *v1.VirtualMachineInstance {
		vmi := libvmi.New(append([]libvmi.Option{
			libvmi.WithName(testVMName),
			libvmi.WithNamespace(metav1.NamespaceDefault),
		}, options...)...)
		vmi.Status.Phase = v1.Running
		vmi.Status.NodeName = host
		vmi.Labels = map[string]string{v1.NodeNameLabel: host}
		return vmi
	}

	runningMigration := func(vmi *v1.VirtualMachineInstance) {
		vmi.Status.MigrationState = &v1.VirtualMachineInstanceMigrationState{
			MigrationUID:   "abcd",
			SourceNode:     host,
			TargetNode:     otherHost,
			StartTimestamp: new(metav1.Now()),
		}
	}

	It("should give a plain local VMI to the vm controller", func() {
		Expect(migrationRole(newVMI(), true, nil, host)).To(Equal(roleVM))
	})

	It("should give a vanished VMI to the vm controller for local cleanup", func() {
		Expect(migrationRole(newVMI(), false, nil, host)).To(Equal(roleVM))
	})

	It("should give an outgoing migration to the source controller", func() {
		vmi := newVMI()
		runningMigration(vmi)
		Expect(migrationRole(vmi, true, nil, host)).To(Equal(roleSource))
	})

	It("should keep the source controller before the migration has started", func() {
		vmi := newVMI()
		runningMigration(vmi)
		vmi.Status.MigrationState.StartTimestamp = nil
		Expect(migrationRole(vmi, true, nil, host)).To(Equal(roleSource))
	})

	It("should hand a finished migration back to the vm controller on the source", func() {
		vmi := newVMI()
		runningMigration(vmi)
		vmi.Status.MigrationState.EndTimestamp = new(metav1.Now())
		vmi.Status.MigrationState.Completed = true
		Expect(migrationRole(vmi, true, nil, host)).To(Equal(roleVM))
	})

	It("should give a VMI deleted mid-migration to the vm controller on the source", func() {
		vmi := newVMI()
		runningMigration(vmi)
		vmi.DeletionTimestamp = new(metav1.Now())
		Expect(migrationRole(vmi, true, nil, host)).To(Equal(roleVM))
	})

	It("should give an incoming migration to the target controller", func() {
		vmi := newVMI()
		vmi.Status.NodeName = otherHost
		vmi.Labels = map[string]string{
			v1.NodeNameLabel:                otherHost,
			v1.MigrationTargetNodeNameLabel: host,
		}
		vmi.Status.MigrationState = &v1.VirtualMachineInstanceMigrationState{
			MigrationUID: "abcd",
			SourceNode:   otherHost,
			TargetNode:   host,
		}
		Expect(migrationRole(vmi, true, nil, host)).To(Equal(roleTarget))
	})

	It("should keep the target controller while it finalizes a completed migration", func() {
		vmi := newVMI()
		vmi.Labels = map[string]string{
			// ackMigrationCompletion already moved the VMI onto this node, but the
			// target has not dropped its own label yet.
			v1.NodeNameLabel:                host,
			v1.MigrationTargetNodeNameLabel: host,
		}
		vmi.Status.MigrationState = &v1.VirtualMachineInstanceMigrationState{
			MigrationUID:   "abcd",
			SourceNode:     otherHost,
			TargetNode:     host,
			StartTimestamp: new(metav1.Now()),
			EndTimestamp:   new(metav1.Now()),
			Completed:      true,
		}
		Expect(migrationRole(vmi, true, nil, host)).To(Equal(roleTarget))
	})

	// virt-handler can die on the target between ackMigrationCompletion and
	// finalCleanup. The claim is a label on the VMI rather than anything in memory,
	// so it survives the restart and the target still gets to finish - but only
	// while it can: once the guest here is terminal, holding a claim it can no
	// longer release would lock the VM controller out of its own node.
	It("should release a claim it can no longer finish once the guest here is terminal", func() {
		vmi := newVMI()
		vmi.Labels = map[string]string{
			v1.NodeNameLabel:                host,
			v1.MigrationTargetNodeNameLabel: host,
		}
		vmi.Status.MigrationState = &v1.VirtualMachineInstanceMigrationState{
			MigrationUID:   "abcd",
			SourceNode:     otherHost,
			TargetNode:     host,
			StartTimestamp: new(metav1.Now()),
			EndTimestamp:   new(metav1.Now()),
			Completed:      true,
		}

		By("the target still owning the running guest it has to finalize")
		Expect(migrationRole(vmi, true, nil, host)).To(Equal(roleTarget))

		By("handing over as soon as that guest is terminal")
		vmi.Status.Phase = v1.Failed
		Expect(migrationRole(vmi, true, nil, host)).To(Equal(roleVM))
	})

	It("should hand over to the vm controller once the target drops its label", func() {
		vmi := newVMI()
		vmi.Status.MigrationState = &v1.VirtualMachineInstanceMigrationState{
			MigrationUID:   "abcd",
			SourceNode:     otherHost,
			TargetNode:     host,
			StartTimestamp: new(metav1.Now()),
			EndTimestamp:   new(metav1.Now()),
			Completed:      true,
		}
		Expect(migrationRole(vmi, true, nil, host)).To(Equal(roleVM))
	})

	It("should give the source back to the vm controller once a migration failed", func() {
		vmi := newVMI()
		runningMigration(vmi)
		vmi.Status.MigrationState.Failed = true
		vmi.Status.MigrationState.EndTimestamp = new(metav1.Now())
		Expect(migrationRole(vmi, true, nil, host)).To(Equal(roleVM))
	})

	It("should keep the source while a migration is being aborted", func() {
		vmi := newVMI()
		runningMigration(vmi)
		vmi.Status.MigrationState.AbortRequested = true
		Expect(migrationRole(vmi, true, nil, host)).To(Equal(roleSource))
	})

	It("should keep the target after a failure until it has released the VMI", func() {
		vmi := newVMI()
		// A failed migration leaves the VMI on the source node, so the target's claim
		// rests entirely on its own label.
		vmi.Status.NodeName = otherHost
		vmi.Labels = map[string]string{
			v1.NodeNameLabel:                otherHost,
			v1.MigrationTargetNodeNameLabel: host,
		}
		vmi.Status.MigrationState = &v1.VirtualMachineInstanceMigrationState{
			MigrationUID: "abcd",
			SourceNode:   otherHost,
			TargetNode:   host,
			Failed:       true,
			Completed:    true,
			EndTimestamp: new(metav1.Now()),
		}
		Expect(migrationRole(vmi, true, nil, host)).To(Equal(roleTarget))

		// finalCleanup drops the label; the claim goes with it.
		delete(vmi.Labels, v1.MigrationTargetNodeNameLabel)
		Expect(migrationRole(vmi, true, nil, host)).To(Equal(roleVM))
	})

	It("should give a VMI that failed mid-migration to the vm controller", func() {
		vmi := newVMI()
		runningMigration(vmi)
		vmi.Status.Phase = v1.Failed
		Expect(migrationRole(vmi, true, nil, host)).To(Equal(roleVM))
	})

	It("should let nobody act while only the domain knows about a migration", func() {
		vmi := newVMI()
		domain := api.NewMinimalDomainWithNS(metav1.NamespaceDefault, testVMName)
		domain.Spec.Metadata.KubeVirt.Migration = &api.MigrationMetadata{
			StartTimestamp: new(metav1.Now()),
		}
		Expect(migrationRole(vmi, true, domain, host)).To(Equal(roleNone))
	})

	DescribeTable("should grant exactly one role", func(mutate func(*v1.VirtualMachineInstance), expected role) {
		vmi := newVMI()
		mutate(vmi)
		Expect(migrationRole(vmi, true, nil, host)).To(Equal(expected))
	},
		Entry("plain VMI", func(*v1.VirtualMachineInstance) {}, roleVM),
		Entry("outgoing migration", runningMigration, roleSource),
		Entry("incoming migration", func(vmi *v1.VirtualMachineInstance) {
			vmi.Labels[v1.MigrationTargetNodeNameLabel] = host
			vmi.Status.MigrationState = &v1.VirtualMachineInstanceMigrationState{
				SourceNode: otherHost,
				TargetNode: host,
			}
		}, roleTarget),
	)

	// The source and the target of a migration are never the same node for a VMI, so
	// the two migration predicates can never both claim it here. The role decision
	// leans on that: it is what makes their order a matter of precedence against the
	// VM controller only.
	DescribeTable("should never have both migration controllers claim a VMI", func(mutate func(*v1.VirtualMachineInstance)) {
		vmi := newVMI()
		mutate(vmi)
		Expect(migrationTargetOwns(vmi, host) && migrationSourceOwns(vmi, host)).To(BeFalse())
	},
		Entry("plain VMI", func(*v1.VirtualMachineInstance) {}),
		Entry("outgoing migration", runningMigration),
		Entry("outgoing migration, finished", func(vmi *v1.VirtualMachineInstance) {
			runningMigration(vmi)
			vmi.Status.MigrationState.Completed = true
			vmi.Status.MigrationState.EndTimestamp = new(metav1.Now())
		}),
		Entry("incoming migration", func(vmi *v1.VirtualMachineInstance) {
			vmi.Status.NodeName = otherHost
			vmi.Labels = map[string]string{
				v1.NodeNameLabel:                otherHost,
				v1.MigrationTargetNodeNameLabel: host,
			}
			vmi.Status.MigrationState = &v1.VirtualMachineInstanceMigrationState{
				SourceNode: otherHost,
				TargetNode: host,
			}
		}),
		Entry("incoming migration, handover acknowledged", func(vmi *v1.VirtualMachineInstance) {
			vmi.Labels[v1.MigrationTargetNodeNameLabel] = host
			vmi.Status.MigrationState = &v1.VirtualMachineInstanceMigrationState{
				SourceNode:   otherHost,
				TargetNode:   host,
				Completed:    true,
				EndTimestamp: new(metav1.Now()),
			}
		}),
	)
})
