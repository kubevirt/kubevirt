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
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/util/workqueue"

	v1 "kubevirt.io/api/core/v1"
	"kubevirt.io/client-go/log"

	"kubevirt.io/kubevirt/pkg/controller"
	"kubevirt.io/kubevirt/pkg/libvmi"
	"kubevirt.io/kubevirt/pkg/virt-launcher/virtwrap/api"
)

// noopController stands in for the controllers a focused test does not exercise,
// so that a key routed away from the controller under test is a visible no-op
// rather than a nil dereference.
type noopController struct{}

func (noopController) execute(*vmiState) error { return nil }

// recordingController records the keys it is handed, so a test can assert which
// controller the Arbiter routed a key to.
type recordingController struct {
	keys []string
}

func (c *recordingController) execute(state *vmiState) error {
	c.keys = append(c.keys, state.key)
	return nil
}

// newTestArbiter wires an Arbiter around a single controller, reusing the stores,
// queue and expectations the controller under test was built with. Keys the role
// decision hands to the other two controllers become no-ops.
func newTestArbiter(base *BaseController, owners map[role]vmiController) *Arbiter {
	a := &Arbiter{
		logger:           log.Log.With("controller", "arbiter"),
		host:             base.host,
		queue:            base.queue,
		vmiExpectations:  base.vmiExpectations,
		vmiSourceStore:   base.vmiStore,
		vmiTargetStore:   base.vmiStore,
		vmiGlobalStore:   base.vmiStore,
		domainStore:      base.domainStore,
		vmController:     noopController{},
		sourceController: noopController{},
		targetController: noopController{},
	}

	for owner, c := range owners {
		switch owner {
		case roleVM:
			a.vmController = c
		case roleSource:
			a.sourceController = c
		case roleTarget:
			a.targetController = c
		}
	}

	return a
}

var _ = Describe("Arbiter restart recovery", func() {
	const (
		host      = "thisnode"
		otherHost = "othernode"
	)

	var (
		arbiter     *Arbiter
		sourceStore cache.Store
		targetStore cache.Store
		globalStore cache.Store
		domainStore cache.Store
		queue       workqueue.TypedRateLimitingInterface[string]
		vmi         *v1.VirtualMachineInstance
		domain      *api.Domain
		expectedKey string
	)

	BeforeEach(func() {
		sourceStore = cache.NewStore(cache.DeletionHandlingMetaNamespaceKeyFunc)
		targetStore = cache.NewStore(cache.DeletionHandlingMetaNamespaceKeyFunc)
		globalStore = cache.NewStore(cache.DeletionHandlingMetaNamespaceKeyFunc)
		domainStore = cache.NewStore(cache.DeletionHandlingMetaNamespaceKeyFunc)
		queue = workqueue.NewTypedRateLimitingQueueWithConfig(
			workqueue.DefaultTypedControllerRateLimiter[string](),
			workqueue.TypedRateLimitingQueueConfig[string]{Name: "arbiter-test"},
		)

		arbiter = &Arbiter{
			logger:         log.Log.With("controller", "arbiter"),
			host:           host,
			queue:          queue,
			vmiSourceStore: sourceStore,
			vmiTargetStore: targetStore,
			vmiGlobalStore: globalStore,
			domainStore:    domainStore,
		}

		vmi = libvmi.New(libvmi.WithName("testvmi"), libvmi.WithNamespace(metav1.NamespaceDefault))
		vmi.UID = "vmi-uid"
		domain = api.NewMinimalDomainWithUUID("testvmi", vmi.UID)
		Expect(domainStore.Add(domain)).To(Succeed())
		expectedKey = metav1.NamespaceDefault + "/testvmi"
	})

	// virt-handler can die on the source while a migration is in flight and come back
	// after it finished. By then the VMI belongs to the target node, so neither
	// node-scoped informer serves it here - but the source domain it left behind
	// still has to be reaped locally.
	It("should queue a domain left behind by a VMI that migrated away while it was down", func() {
		vmi.Status.NodeName = otherHost
		vmi.Labels = map[string]string{v1.NodeNameLabel: otherHost}
		Expect(globalStore.Add(vmi)).To(Succeed())

		arbiter.enqueueUnclaimedDomains()

		Expect(queue.Len()).To(Equal(1))
		key, _ := queue.Get()
		Expect(key).To(Equal(expectedKey))
	})

	It("should queue a domain whose VMI is gone entirely", func() {
		arbiter.enqueueUnclaimedDomains()

		Expect(queue.Len()).To(Equal(1))
		key, _ := queue.Get()
		Expect(key).To(Equal(expectedKey))
	})

	It("should leave domains of VMIs this node still serves to the informers", func() {
		vmi.Status.NodeName = host
		vmi.Labels = map[string]string{v1.NodeNameLabel: host}
		Expect(sourceStore.Add(vmi)).To(Succeed())
		Expect(globalStore.Add(vmi)).To(Succeed())

		arbiter.enqueueUnclaimedDomains()

		Expect(queue.Len()).To(BeZero())
	})

	It("should leave domains of VMIs this node is the migration target of", func() {
		vmi.Labels = map[string]string{v1.MigrationTargetNodeNameLabel: host}
		Expect(targetStore.Add(vmi)).To(Succeed())
		Expect(globalStore.Add(vmi)).To(Succeed())

		arbiter.enqueueUnclaimedDomains()

		Expect(queue.Len()).To(BeZero())
	})
})

var _ = Describe("Arbiter dispatch", func() {
	const (
		host      = "thisnode"
		otherHost = "othernode"
	)

	var (
		arbiter     *Arbiter
		vmCtrl      *recordingController
		sourceCtrl  *recordingController
		targetCtrl  *recordingController
		vmiStore    cache.Store
		domainStore cache.Store
	)

	newVMI := func() *v1.VirtualMachineInstance {
		vmi := libvmi.New(libvmi.WithName("testvmi"), libvmi.WithNamespace(metav1.NamespaceDefault))
		vmi.UID = "vmi-uid"
		vmi.Status.Phase = v1.Running
		vmi.Status.NodeName = host
		vmi.Labels = map[string]string{v1.NodeNameLabel: host}
		return vmi
	}

	BeforeEach(func() {
		vmiStore = cache.NewStore(cache.DeletionHandlingMetaNamespaceKeyFunc)
		domainStore = cache.NewStore(cache.DeletionHandlingMetaNamespaceKeyFunc)
		vmCtrl = &recordingController{}
		sourceCtrl = &recordingController{}
		targetCtrl = &recordingController{}
		arbiter = &Arbiter{
			logger:          log.Log.With("controller", "arbiter"),
			host:            host,
			queue:           workqueue.NewTypedRateLimitingQueueWithConfig(workqueue.DefaultTypedControllerRateLimiter[string](), workqueue.TypedRateLimitingQueueConfig[string]{Name: "arbiter-test"}),
			vmiExpectations: controller.NewUIDTrackingControllerExpectations(controller.NewControllerExpectations()),
			// The role decision hinges on the VMI's content, not on which store
			// served it, so one store stands in for all three.
			vmiSourceStore:   vmiStore,
			vmiTargetStore:   vmiStore,
			vmiGlobalStore:   vmiStore,
			domainStore:      domainStore,
			vmController:     vmCtrl,
			sourceController: sourceCtrl,
			targetController: targetCtrl,
		}
	})

	dispatch := func(vmi *v1.VirtualMachineInstance, domain *api.Domain) {
		if vmi != nil {
			Expect(vmiStore.Add(vmi)).To(Succeed())
		}
		if domain != nil {
			Expect(domainStore.Add(domain)).To(Succeed())
		}
		Expect(arbiter.execute(metav1.NamespaceDefault + "/testvmi")).To(Succeed())
	}

	expectDispatchedTo := func(owner *recordingController) {
		ExpectWithOffset(1, owner.keys).To(ConsistOf(metav1.NamespaceDefault + "/testvmi"))
		for _, idle := range []*recordingController{vmCtrl, sourceCtrl, targetCtrl} {
			if idle != owner {
				ExpectWithOffset(1, idle.keys).To(BeEmpty())
			}
		}
	}

	It("should route a local VMI to the vm controller", func() {
		dispatch(newVMI(), nil)
		expectDispatchedTo(vmCtrl)
	})

	It("should route a vanished VMI to the vm controller", func() {
		dispatch(nil, nil)
		expectDispatchedTo(vmCtrl)
	})

	It("should route a VMI this node is the migration target of to the target controller", func() {
		vmi := newVMI()
		vmi.Status.NodeName = otherHost
		vmi.Labels = map[string]string{
			v1.NodeNameLabel:                otherHost,
			v1.MigrationTargetNodeNameLabel: host,
		}
		vmi.Status.MigrationState = &v1.VirtualMachineInstanceMigrationState{
			SourceNode: otherHost,
			TargetNode: host,
		}
		dispatch(vmi, nil)
		expectDispatchedTo(targetCtrl)
	})

	It("should route an outgoing migration to the source controller", func() {
		vmi := newVMI()
		vmi.Status.MigrationState = &v1.VirtualMachineInstanceMigrationState{
			SourceNode:     host,
			TargetNode:     otherHost,
			StartTimestamp: new(metav1.Now()),
		}
		dispatch(vmi, nil)
		expectDispatchedTo(sourceCtrl)
	})

	It("should route a VMI that went terminal mid-migration back to the vm controller", func() {
		vmi := newVMI()
		vmi.Status.NodeName = otherHost
		vmi.Labels = map[string]string{
			v1.NodeNameLabel:                otherHost,
			v1.MigrationTargetNodeNameLabel: host,
		}
		vmi.Status.MigrationState = &v1.VirtualMachineInstanceMigrationState{
			SourceNode:     otherHost,
			TargetNode:     host,
			StartTimestamp: new(metav1.Now()),
		}
		vmi.Status.Phase = v1.Failed
		dispatch(vmi, nil)
		expectDispatchedTo(vmCtrl)
	})

	It("should route nobody while only the domain knows about the migration", func() {
		domain := api.NewMinimalDomainWithUUID("testvmi", "vmi-uid")
		domain.Spec.Metadata.KubeVirt.Migration = &api.MigrationMetadata{
			StartTimestamp: new(metav1.Now()),
		}
		dispatch(newVMI(), domain)
		for _, ctrl := range []*recordingController{vmCtrl, sourceCtrl, targetCtrl} {
			Expect(ctrl.keys).To(BeEmpty())
		}
	})
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

	incomingMigration := func(vmi *v1.VirtualMachineInstance) {
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
	}

	DescribeTable("should grant exactly one role",
		func(vmiExists bool, mutate func(*v1.VirtualMachineInstance), expected role) {
			vmi := newVMI()
			mutate(vmi)
			Expect(migrationRole(vmi, vmiExists, nil, host)).To(Equal(expected))
		},
		Entry("a plain local VMI to the vm controller", true, func(*v1.VirtualMachineInstance) {}, roleVM),
		Entry("a vanished VMI to the vm controller for local cleanup", false, func(*v1.VirtualMachineInstance) {}, roleVM),
		Entry("an outgoing migration to the source controller", true, runningMigration, roleSource),
		Entry("an outgoing migration to the source controller before it has started", true, func(vmi *v1.VirtualMachineInstance) {
			runningMigration(vmi)
			vmi.Status.MigrationState.StartTimestamp = nil
		}, roleSource),
		Entry("an outgoing migration to the source controller while it is being aborted", true, func(vmi *v1.VirtualMachineInstance) {
			runningMigration(vmi)
			vmi.Status.MigrationState.AbortRequested = true
		}, roleSource),
		Entry("a finished migration back to the vm controller on the source", true, func(vmi *v1.VirtualMachineInstance) {
			runningMigration(vmi)
			vmi.Status.MigrationState.EndTimestamp = new(metav1.Now())
			vmi.Status.MigrationState.Completed = true
		}, roleVM),
		Entry("a failed migration back to the vm controller on the source", true, func(vmi *v1.VirtualMachineInstance) {
			runningMigration(vmi)
			vmi.Status.MigrationState.Failed = true
			vmi.Status.MigrationState.EndTimestamp = new(metav1.Now())
		}, roleVM),
		Entry("a VMI deleted mid-migration to the vm controller", true, func(vmi *v1.VirtualMachineInstance) {
			runningMigration(vmi)
			vmi.DeletionTimestamp = new(metav1.Now())
		}, roleVM),
		Entry("a VMI that failed mid-migration to the vm controller", true, func(vmi *v1.VirtualMachineInstance) {
			runningMigration(vmi)
			vmi.Status.Phase = v1.Failed
		}, roleVM),
		Entry("an incoming migration to the target controller", true, incomingMigration, roleTarget),
		Entry("an incoming migration to the target controller while it finalizes completion", true, func(vmi *v1.VirtualMachineInstance) {
			// ackMigrationCompletion already moved the VMI onto this node, but the
			// target has not dropped its own label yet.
			vmi.Labels[v1.MigrationTargetNodeNameLabel] = host
			vmi.Status.MigrationState = &v1.VirtualMachineInstanceMigrationState{
				MigrationUID:   "abcd",
				SourceNode:     otherHost,
				TargetNode:     host,
				StartTimestamp: new(metav1.Now()),
				EndTimestamp:   new(metav1.Now()),
				Completed:      true,
			}
		}, roleTarget),
		Entry("an incoming migration back to the vm controller once the target drops its label", true, func(vmi *v1.VirtualMachineInstance) {
			vmi.Status.MigrationState = &v1.VirtualMachineInstanceMigrationState{
				MigrationUID:   "abcd",
				SourceNode:     otherHost,
				TargetNode:     host,
				StartTimestamp: new(metav1.Now()),
				EndTimestamp:   new(metav1.Now()),
				Completed:      true,
			}
		}, roleVM),
		// The target-side twin of the entry above: the guest died on the source
		// before ackMigrationCompletion, so the VMI is terminal yet still claimed by
		// the target. The target controller short-circuits terminal VMIs into its
		// best-effort cleanup and can never reach finalCleanup to drop its own
		// label, so it must not keep the claim: the VM controller's
		// processVmCleanup is what reaps this node's leftover domain and ghost
		// record.
		Entry("a VMI that went terminal mid-migration back to the vm controller", true, func(vmi *v1.VirtualMachineInstance) {
			incomingMigration(vmi)
			vmi.Status.MigrationState.StartTimestamp = new(metav1.Now())
			vmi.Status.Phase = v1.Failed
		}, roleVM),
	)

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

	It("should keep the target after a failure until it has released the VMI", func() {
		vmi := newVMI()
		// A failed migration leaves the VMI on the source node, so the target's claim
		// rests entirely on its own label.
		incomingMigration(vmi)
		vmi.Status.MigrationState.Failed = true
		vmi.Status.MigrationState.Completed = true
		vmi.Status.MigrationState.EndTimestamp = new(metav1.Now())
		Expect(migrationRole(vmi, true, nil, host)).To(Equal(roleTarget))

		// finalCleanup drops the label; the claim goes with it.
		delete(vmi.Labels, v1.MigrationTargetNodeNameLabel)
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
		Entry("incoming migration", incomingMigration),
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
