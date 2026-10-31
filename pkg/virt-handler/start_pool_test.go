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
	"fmt"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	v1 "kubevirt.io/api/core/v1"

	"kubevirt.io/kubevirt/pkg/virt-launcher/virtwrap/api"
)

var _ = Describe("vm start tracker", func() {
	It("keeps a second begin from starting the same VMI", func() {
		tracker := newVMStartTracker()
		Expect(tracker.begin("uid")).To(BeTrue())
		Expect(tracker.begin("uid")).To(BeFalse())
		Expect(tracker.inProgress("uid")).To(BeTrue())
	})

	It("surfaces a start error once and then allows another begin", func() {
		tracker := newVMStartTracker()
		Expect(tracker.begin("uid")).To(BeTrue())
		tracker.finish("uid", fmt.Errorf("sync failed"), time.Now())

		Expect(tracker.takeError("uid")).To(MatchError("sync failed"))
		Expect(tracker.takeError("uid")).To(Succeed())
		Expect(tracker.begin("uid")).To(BeTrue())
	})

	It("holds a cooldown after a successful start", func() {
		tracker := newVMStartTracker()
		finished := time.Now()
		Expect(tracker.begin("uid")).To(BeTrue())
		tracker.finish("uid", nil, finished)

		Expect(tracker.inProgress("uid")).To(BeFalse())
		Expect(tracker.cooldownRemaining("uid", finished.Add(200*time.Millisecond))).To(BeNumerically(">", 0))
		Expect(tracker.cooldownRemaining("uid", finished.Add(vmStartCooldown))).To(BeZero())
	})

	It("treats a nil or shutoff domain as a cold start and a running domain as worker work", func() {
		vmi := &v1.VirtualMachineInstance{}
		vmi.Status.Phase = v1.Scheduled
		Expect(domainNeedsAsyncStart(vmi, nil)).To(BeTrue())

		shutoff := api.NewMinimalDomain("test")
		shutoff.Status.Status = api.Shutoff
		Expect(domainNeedsAsyncStart(vmi, shutoff)).To(BeTrue())

		running := api.NewMinimalDomain("test")
		running.Status.Status = api.Running
		Expect(domainNeedsAsyncStart(vmi, running)).To(BeFalse())

		vmi.Status.Phase = v1.Running
		Expect(domainNeedsAsyncStart(vmi, nil)).To(BeFalse())

		vmi.Status.Phase = v1.Scheduled
		now := metav1.Now()
		vmi.DeletionTimestamp = &now
		Expect(domainNeedsAsyncStart(vmi, nil)).To(BeFalse())
	})
})
