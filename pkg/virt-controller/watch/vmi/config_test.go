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

package vmi

import (
	"context"
	"sync"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"k8s.io/client-go/tools/cache"

	virtv1 "kubevirt.io/api/core/v1"

	"kubevirt.io/kubevirt/pkg/testutils"
)

var _ = Describe("Informer synchronization", func() {
	It("waits for the migration cache even after the other caches have synced", func(ctx SpecContext) {
		shared, _ := testutils.NewFakeInformerFor(&virtv1.VirtualMachineInstance{})
		migration, _ := testutils.NewFakeInformerFor(&virtv1.VirtualMachineInstanceMigration{})
		informers := Informers{
			VMI: shared, VM: shared, Pod: shared, PVC: shared,
			Migration: migration, StorageClass: shared, DataVolume: shared,
			StorageProfile: shared, CDI: shared, CDIConfig: shared, KubeVirt: shared,
		}
		Expect(informers.HasSynced()).To(BeFalse())

		runCtx, cancel := context.WithCancel(ctx)
		var wg sync.WaitGroup
		DeferCleanup(func() {
			cancel()
			wg.Wait()
		})
		wg.Go(func() { shared.RunWithContext(runCtx) })
		Expect(cache.WaitForCacheSync(ctx.Done(), shared.HasSynced)).To(BeTrue())
		Expect(informers.HasSynced()).To(BeFalse())

		wg.Go(func() { migration.RunWithContext(runCtx) })
		Expect(cache.WaitForCacheSync(ctx.Done(), migration.HasSynced)).To(BeTrue())
		Expect(informers.HasSynced()).To(BeTrue())

		informers.Migration = nil
		Expect(informers.HasSynced()).To(BeFalse())
	}, SpecTimeout(5*time.Second))
})
