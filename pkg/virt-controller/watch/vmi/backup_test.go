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
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	k8sv1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	backupv1 "kubevirt.io/api/backup/v1alpha1"
	virtv1 "kubevirt.io/api/core/v1"

	"kubevirt.io/kubevirt/pkg/controller"
	"kubevirt.io/kubevirt/pkg/pointer"
	"kubevirt.io/kubevirt/pkg/testutils"
	virtconfig "kubevirt.io/kubevirt/pkg/virt-config"
	"kubevirt.io/kubevirt/pkg/virt-config/featuregate"
)

var _ = Describe("VMI offline backup gating", func() {
	const (
		namespace = "default"
		vmName    = "testvm"
	)

	// offlineBackupEnabledConfig enables the OfflineIncrementalBackup gate and its
	// IncrementalBackup prerequisite so the gating logic is active.
	offlineBackupEnabledConfig := func() *virtconfig.ClusterConfig {
		config, _, _ := testutils.NewFakeClusterConfigUsingKVConfig(&virtv1.KubeVirtConfiguration{
			DeveloperConfiguration: &virtv1.DeveloperConfiguration{
				FeatureGates: []string{
					featuregate.IncrementalBackupGate,
					featuregate.OfflineIncrementalBackupGate,
				},
			},
		})
		return config
	}

	progressingOfflineBackup := func() *backupv1.VirtualMachineBackup {
		b := &backupv1.VirtualMachineBackup{
			ObjectMeta: metav1.ObjectMeta{Name: "backup", Namespace: namespace},
			Spec: backupv1.VirtualMachineBackupSpec{
				Source: k8sv1.TypedLocalObjectReference{Kind: "VirtualMachine", Name: vmName},
			},
			Status: &backupv1.VirtualMachineBackupStatus{
				Offline: pointer.P(true),
				Conditions: []metav1.Condition{
					{Type: string(backupv1.ConditionProgressing), Status: metav1.ConditionTrue},
				},
			},
		}
		return b
	}

	vmi := func() *virtv1.VirtualMachineInstance {
		return &virtv1.VirtualMachineInstance{
			ObjectMeta: metav1.ObjectMeta{Name: vmName, Namespace: namespace},
		}
	}

	Describe("isOfflineBackupProgressing", func() {
		It("is true for a progressing offline backup", func() {
			Expect(isOfflineBackupProgressing(progressingOfflineBackup())).To(BeTrue())
		})

		It("is false when not offline", func() {
			b := progressingOfflineBackup()
			b.Status.Offline = nil
			Expect(isOfflineBackupProgressing(b)).To(BeFalse())
		})

		It("is false when being deleted", func() {
			b := progressingOfflineBackup()
			now := metav1.Now()
			b.DeletionTimestamp = &now
			Expect(isOfflineBackupProgressing(b)).To(BeFalse())
		})

		It("is false when already complete", func() {
			b := progressingOfflineBackup()
			b.Status.Conditions = append(b.Status.Conditions, metav1.Condition{
				Type: string(backupv1.ConditionComplete), Status: metav1.ConditionTrue,
			})
			Expect(isOfflineBackupProgressing(b)).To(BeFalse())
		})
	})

	Describe("offlineBackupInProgress", func() {
		newController := func(backups ...*backupv1.VirtualMachineBackup) *Controller {
			backupInformer, _ := testutils.NewFakeInformerFor(&backupv1.VirtualMachineBackup{})
			trackerInformer, _ := testutils.NewFakeInformerFor(&backupv1.VirtualMachineBackupTracker{})
			for _, b := range backups {
				Expect(backupInformer.GetStore().Add(b)).To(Succeed())
			}
			return &Controller{
				vmBackupStore:        backupInformer.GetStore(),
				vmBackupTrackerStore: trackerInformer.GetStore(),
				clusterConfig:        offlineBackupEnabledConfig(),
			}
		}

		It("is true when an offline backup targets the VMI's VM", func() {
			c := newController(progressingOfflineBackup())
			Expect(c.offlineBackupInProgress(vmi())).To(BeTrue())
		})

		It("is false for a backup of a different VM", func() {
			b := progressingOfflineBackup()
			b.Spec.Source.Name = "othervm"
			c := newController(b)
			Expect(c.offlineBackupInProgress(vmi())).To(BeFalse())
		})

		It("is false with no backups", func() {
			c := newController()
			Expect(c.offlineBackupInProgress(vmi())).To(BeFalse())
		})

		It("is false when the OfflineIncrementalBackup feature gate is disabled", func() {
			c := newController(progressingOfflineBackup())
			config, _, _ := testutils.NewFakeClusterConfigUsingKVConfig(&virtv1.KubeVirtConfiguration{})
			c.clusterConfig = config
			Expect(c.offlineBackupInProgress(vmi())).To(BeFalse())
		})
	})

	Describe("syncBackupInProgressCondition", func() {
		cm := controller.NewVirtualMachineInstanceConditionManager()

		It("adds the condition while a backup is in progress", func() {
			backupInformer, _ := testutils.NewFakeInformerFor(&backupv1.VirtualMachineBackup{})
			trackerInformer, _ := testutils.NewFakeInformerFor(&backupv1.VirtualMachineBackupTracker{})
			Expect(backupInformer.GetStore().Add(progressingOfflineBackup())).To(Succeed())
			c := &Controller{
				vmBackupStore:        backupInformer.GetStore(),
				vmBackupTrackerStore: trackerInformer.GetStore(),
				clusterConfig:        offlineBackupEnabledConfig(),
			}

			v := vmi()
			c.syncBackupInProgressCondition(v)
			Expect(cm.HasCondition(v, virtv1.VirtualMachineInstanceBackupInProgress)).To(BeTrue())
		})

		It("removes the condition when no backup is in progress", func() {
			backupInformer, _ := testutils.NewFakeInformerFor(&backupv1.VirtualMachineBackup{})
			trackerInformer, _ := testutils.NewFakeInformerFor(&backupv1.VirtualMachineBackupTracker{})
			c := &Controller{
				vmBackupStore:        backupInformer.GetStore(),
				vmBackupTrackerStore: trackerInformer.GetStore(),
				clusterConfig:        offlineBackupEnabledConfig(),
			}

			v := vmi()
			v.Status.Conditions = append(v.Status.Conditions, virtv1.VirtualMachineInstanceCondition{
				Type:   virtv1.VirtualMachineInstanceBackupInProgress,
				Status: k8sv1.ConditionTrue,
			})
			c.syncBackupInProgressCondition(v)
			Expect(cm.HasCondition(v, virtv1.VirtualMachineInstanceBackupInProgress)).To(BeFalse())
		})
	})
})
