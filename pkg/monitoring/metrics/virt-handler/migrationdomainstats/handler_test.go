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
package migrationdomainstats

import (
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	v1 "kubevirt.io/api/core/v1"
	"kubevirt.io/client-go/api"

	"kubevirt.io/kubevirt/pkg/controller"
	"kubevirt.io/kubevirt/pkg/pointer"
	"kubevirt.io/kubevirt/pkg/testutils"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

var _ = Describe("Handler", func() {
	It("should clean up queue when migration is finished", func() {
		vmiInformer, _ := testutils.NewFakeInformerFor(&v1.VirtualMachineInstance{})
		handler, err := newHandler(vmiInformer)
		Expect(err).ToNot(HaveOccurred())

		vmi := api.NewMinimalVMI("testvmi")
		vmi.Status.MigrationState = &v1.VirtualMachineInstanceMigrationState{
			StartTimestamp: pointer.P(metav1.Now()),
		}
		key := controller.NamespacedKey(vmi.Namespace, vmi.Name)
		vmiInformer.GetStore().Add(vmi.DeepCopy())
		handler.addMigration(vmi.DeepCopy())

		queue := handler.vmiStats[key]
		Expect(handler.vmiStats).To(HaveKey(key))

		// trigger collection
		handler.Collect()

		Consistently(func() bool { return queue.isActive.Load() }).Should(BeTrue())
		Expect(handler.vmiStats).To(HaveKey(key))

		vmi.Status.EvacuationNodeName = "node"
		vmiInformer.GetStore().Add(vmi.DeepCopy())
		handler.addMigration(vmi.DeepCopy())

		Expect(handler.vmiStats).To(HaveKeyWithValue(key, queue))

		vmi.Status.MigrationState.Completed = true
		vmiInformer.GetStore().Add(vmi.DeepCopy())

		// TODO: Make the test go brrr
		Eventually(func() bool { return queue.isActive.Load() }).WithTimeout(6 * time.Second).Should(BeFalse())

		handler.Collect()

		Expect(handler.vmiStats).ToNot(HaveKey(key))
	})
})
