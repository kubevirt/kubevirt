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

	"kubevirt.io/kubevirt/pkg/libvmi"
	"kubevirt.io/kubevirt/pkg/virt-launcher/virtwrap/api"
)

// noopController stands in for the controllers a focused test does not exercise,
// so that a key routed away from the controller under test is a visible no-op
// rather than a nil dereference.
type noopController struct{}

func (noopController) execute(*vmiState) error { return nil }

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
