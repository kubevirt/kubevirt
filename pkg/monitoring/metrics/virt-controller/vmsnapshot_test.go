/*
Copyright The KubeVirt Authors.
SPDX-License-Identifier: Apache-2.0
*/

package virtcontroller_test

import (
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	snapshotv1 "kubevirt.io/api/snapshot/v1beta1"

	metrics "kubevirt.io/kubevirt/pkg/monitoring/metrics/virt-controller"
	"kubevirt.io/kubevirt/pkg/pointer"
)

var _ = Describe("VMSnapshot Metrics Collector", func() {
	Context("VMSnapshot status collector", func() {
		It("should set and retrieve the VMSnapshot creation time metric correctly", func() {
			// Create vm snapshot and content objects
			fixedTime := metav1.NewTime(time.Now())
			vmSnapshot := &snapshotv1.VirtualMachineSnapshot{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "snapshot-name",
					Namespace: "namespace",
				},
				Spec: snapshotv1.VirtualMachineSnapshotSpec{
					Source: corev1.TypedLocalObjectReference{
						APIGroup: pointer.P("kubevirt.io"),
						Kind:     "VirtualMachine",
						Name:     "vm-name",
					},
				},
				Status: &snapshotv1.VirtualMachineSnapshotStatus{
					CreationTime: &fixedTime,
					ReadyToUse:   pointer.P(true),
					Phase:        snapshotv1.Succeeded,
				},
			}

			metrics.HandleSucceededVMSnapshot(vmSnapshot)

			metricTime, err := metrics.GetVMSnapshotSucceededTimestamp("vm-name", "snapshot-name", "namespace")
			Expect(err).NotTo(HaveOccurred())
			Expect(metricTime).NotTo(BeNil())

			Expect(metricTime).To(Equal(float64(vmSnapshot.Status.CreationTime.Unix())))
		})
	})
})
