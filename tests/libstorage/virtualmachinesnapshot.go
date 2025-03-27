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
 * Copyright 2023 Red Hat, Inc.
 *
 */

package libstorage

import (
	"context"
	"time"

	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/gstruct"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	snapshotv1 "kubevirt.io/api/snapshot/v1beta1"
	"kubevirt.io/client-go/kubecli"
)

var groupName = "kubevirt.io"

func WaitSnapshotSucceeded(virtClient kubecli.KubevirtClient, namespace string, snapshotName string) *snapshotv1.VirtualMachineSnapshot {
	var snapshot *snapshotv1.VirtualMachineSnapshot
	Eventually(func() *snapshotv1.VirtualMachineSnapshotStatus {
		var err error
		snapshot, err = virtClient.VirtualMachineSnapshot(namespace).Get(context.Background(), snapshotName, metav1.GetOptions{})
		Expect(err).ToNot(HaveOccurred())
		return snapshot.Status
	}, 180*time.Second, 2*time.Second).Should(gstruct.PointTo(gstruct.MatchFields(gstruct.IgnoreExtras, gstruct.Fields{
		"ReadyToUse": gstruct.PointTo(BeTrue()),
		"Conditions": ContainElements(
			gstruct.MatchFields(gstruct.IgnoreExtras, gstruct.Fields{
				"Type":   Equal(snapshotv1.ConditionReady),
				"Status": Equal(corev1.ConditionTrue),
				"Reason": Equal("Ready"),
			}),
			gstruct.MatchFields(gstruct.IgnoreExtras, gstruct.Fields{
				"Type":   Equal(snapshotv1.ConditionProgressing),
				"Status": Equal(corev1.ConditionFalse),
				"Reason": Equal("Operation complete"),
			}),
		),
	})))

	return snapshot
}

func NewSnapshot(vm, namespace string) *snapshotv1.VirtualMachineSnapshot {
	return &snapshotv1.VirtualMachineSnapshot{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "snapshot-" + vm,
			Namespace: namespace,
		},
		Spec: snapshotv1.VirtualMachineSnapshotSpec{
			Source: corev1.TypedLocalObjectReference{
				APIGroup: &groupName,
				Kind:     "VirtualMachine",
				Name:     vm,
			},
		},
	}
}
