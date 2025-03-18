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
 * Copyright 2021 Red Hat, Inc.
 *
 */

package tests_test

import (
	"context"
	"fmt"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	k8sres "k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/util/retry"

	v1 "kubevirt.io/api/core/v1"
	snapshotv1 "kubevirt.io/api/snapshot/v1beta1"
	"kubevirt.io/client-go/kubecli"

	"kubevirt.io/kubevirt/pkg/libvmi"
	"kubevirt.io/kubevirt/pkg/virt-config/featuregate"
	"kubevirt.io/kubevirt/tests/decorators"
	"kubevirt.io/kubevirt/tests/framework/kubevirt"
	"kubevirt.io/kubevirt/tests/libkubevirt"
	"kubevirt.io/kubevirt/tests/libkubevirt/config"
	"kubevirt.io/kubevirt/tests/libmigration"
	"kubevirt.io/kubevirt/tests/libvmifact"
	"kubevirt.io/kubevirt/tests/testsuite"
)

var _ = Describe("[sig-compute]Dry-Run requests", decorators.SigCompute, decorators.WgS390x, func() {
	var err error
	var virtClient kubecli.KubevirtClient
	var restClient *rest.RESTClient

	BeforeEach(func() {
		virtClient = kubevirt.Client()
		restClient = virtClient.RestClient()
	})

	Context("VirtualMachineInstances", func() {
		var (
			vmi *v1.VirtualMachineInstance
		)
		resource := "virtualmachineinstances"

		BeforeEach(func() {
			vmi = libvmifact.NewAlpine(
				libvmi.WithNamespace(testsuite.GetTestNamespace(nil)),
				libvmi.WithInterface(libvmi.InterfaceDeviceWithMasqueradeBinding()),
				libvmi.WithNetwork(v1.DefaultPodNetwork()),
			)
		})

		It("[test_id:7627]create a VirtualMachineInstance", func() {
			By("Make a Dry-Run request to create a Virtual Machine")
			err = dryRunCreate(restClient, resource, vmi.Namespace, vmi, nil)
			Expect(err).ToNot(HaveOccurred())

			By("Check that no Virtual Machine was actually created")
			_, err = virtClient.VirtualMachineInstance(vmi.Namespace).Get(context.Background(), vmi.Name, metav1.GetOptions{})
			Expect(err).To(MatchError(errors.IsNotFound, "k8serrors.IsNotFound"))
		})

		It("[test_id:7628]delete a VirtualMachineInstance", func() {
			By("Create a VirtualMachineInstance")
			_, err := virtClient.VirtualMachineInstance(vmi.Namespace).Create(context.Background(), vmi, metav1.CreateOptions{})
			Expect(err).ToNot(HaveOccurred())

			By("Make a Dry-Run request to delete a Virtual Machine")
			deletePolicy := metav1.DeletePropagationForeground
			opts := metav1.DeleteOptions{
				DryRun:            []string{metav1.DryRunAll},
				PropagationPolicy: &deletePolicy,
			}
			err = virtClient.VirtualMachineInstance(vmi.Namespace).Delete(context.Background(), vmi.Name, opts)
			Expect(err).ToNot(HaveOccurred())

			By("Check that no Virtual Machine was actually deleted")
			_, err = virtClient.VirtualMachineInstance(vmi.Namespace).Get(context.Background(), vmi.Name, metav1.GetOptions{})
			Expect(err).ToNot(HaveOccurred())
		})

		It("[test_id:7629]update a VirtualMachineInstance", func() {
			By("Create a VirtualMachineInstance")
			_, err = virtClient.VirtualMachineInstance(vmi.Namespace).Create(context.Background(), vmi, metav1.CreateOptions{})
			Expect(err).ToNot(HaveOccurred())

			By("Make a Dry-Run request to update a Virtual Machine")
			err = retry.RetryOnConflict(retry.DefaultRetry, func() error {
				vmi, err = virtClient.VirtualMachineInstance(vmi.Namespace).Get(context.Background(), vmi.Name, metav1.GetOptions{})
				if err != nil {
					return err
				}
				vmi.Labels = map[string]string{
					"key": "42",
				}
				return dryRunUpdate(restClient, resource, vmi.Name, vmi.Namespace, vmi, nil)
			})

			By("Check that no update actually took place")
			vmi, err = virtClient.VirtualMachineInstance(vmi.Namespace).Get(context.Background(), vmi.Name, metav1.GetOptions{})
			Expect(err).ToNot(HaveOccurred())
			Expect(vmi.Labels["key"]).ToNot(Equal("42"))
		})

		It("[test_id:7630]patch a VirtualMachineInstance", func() {
			By("Create a VirtualMachineInstance")
			vmi, err := virtClient.VirtualMachineInstance(vmi.Namespace).Create(context.Background(), vmi, metav1.CreateOptions{})
			Expect(err).ToNot(HaveOccurred())

			By("Make a Dry-Run request to patch a Virtual Machine")
			patch := []byte(`{"metadata": {"labels": {"key": "42"}}}`)
			err = dryRunPatch(restClient, resource, vmi.Name, vmi.Namespace, types.MergePatchType, patch, nil)
			Expect(err).ToNot(HaveOccurred())

			By("Check that no update actually took place")
			vmi, err = virtClient.VirtualMachineInstance(vmi.Namespace).Get(context.Background(), vmi.Name, metav1.GetOptions{})
			Expect(err).ToNot(HaveOccurred())
			Expect(vmi.Labels["key"]).ToNot(Equal("42"))
		})
	})

	Context("VirtualMachines", func() {
		var (
			vm        *v1.VirtualMachine
			namespace string
		)
		const resource = "virtualmachines"

		BeforeEach(func() {
			vm = libvmi.NewVirtualMachine(libvmifact.NewAlpine())
			namespace = testsuite.GetTestNamespace(vm)
		})

		It("[test_id:7631]create a VirtualMachine", func() {
			By("Make a Dry-Run request to create a Virtual Machine")
			err = dryRunCreate(restClient, resource, namespace, vm, nil)
			Expect(err).ToNot(HaveOccurred())

			By("Check that no Virtual Machine was actually created")
			_, err = virtClient.VirtualMachine(namespace).Get(context.Background(), vm.Name, metav1.GetOptions{})
			Expect(err).To(MatchError(errors.IsNotFound, "k8serrors.IsNotFound"))
		})

		It("[test_id:7632]delete a VirtualMachine", func() {
			By("Create a VirtualMachine")
			vm, err = virtClient.VirtualMachine(namespace).Create(context.Background(), vm, metav1.CreateOptions{})
			Expect(err).ToNot(HaveOccurred())

			By("Make a Dry-Run request to delete a Virtual Machine")
			deletePolicy := metav1.DeletePropagationForeground
			opts := metav1.DeleteOptions{
				DryRun:            []string{metav1.DryRunAll},
				PropagationPolicy: &deletePolicy,
			}
			err = virtClient.VirtualMachine(vm.Namespace).Delete(context.Background(), vm.Name, opts)
			Expect(err).ToNot(HaveOccurred())

			By("Check that no Virtual Machine was actually deleted")
			_, err = virtClient.VirtualMachine(vm.Namespace).Get(context.Background(), vm.Name, metav1.GetOptions{})
			Expect(err).ToNot(HaveOccurred())
		})

		It("[test_id:7633]update a VirtualMachine", func() {
			By("Create a VirtualMachine")
			vm, err = virtClient.VirtualMachine(namespace).Create(context.Background(), vm, metav1.CreateOptions{})
			Expect(err).ToNot(HaveOccurred())

			By("Make a Dry-Run request to update a Virtual Machine")
			err = retry.RetryOnConflict(retry.DefaultRetry, func() error {
				vm, err = virtClient.VirtualMachine(vm.Namespace).Get(context.Background(), vm.Name, metav1.GetOptions{})
				if err != nil {
					return err
				}
				vm.Labels = map[string]string{
					"key": "42",
				}
				return dryRunUpdate(restClient, resource, vm.Name, vm.Namespace, vm, nil)
			})
			Expect(err).ToNot(HaveOccurred())

			By("Check that no update actually took place")
			vm, err = virtClient.VirtualMachine(namespace).Get(context.Background(), vm.Name, metav1.GetOptions{})
			Expect(err).ToNot(HaveOccurred())
			Expect(vm.Labels["key"]).ToNot(Equal("42"))
		})

		It("[test_id:7634]patch a VirtualMachine", func() {
			By("Create a VirtualMachine")
			vm, err = virtClient.VirtualMachine(namespace).Create(context.Background(), vm, metav1.CreateOptions{})
			Expect(err).ToNot(HaveOccurred())

			By("Make a Dry-Run request to patch a Virtual Machine")
			patch := []byte(`{"metadata": {"labels": {"key": "42"}}}`)
			err = dryRunPatch(restClient, resource, vm.Name, vm.Namespace, types.MergePatchType, patch, nil)
			Expect(err).ToNot(HaveOccurred())

			By("Check that no update actually took place")
			vm, err = virtClient.VirtualMachine(vm.Namespace).Get(context.Background(), vm.Name, metav1.GetOptions{})
			Expect(err).ToNot(HaveOccurred())
			Expect(vm.Labels["key"]).ToNot(Equal("42"))
		})
	})

	Context("Migrations", decorators.SigComputeMigrations, func() {
		var vmim *v1.VirtualMachineInstanceMigration
		resource := "virtualmachineinstancemigrations"

		BeforeEach(func() {
			vmi := libvmifact.NewAlpine(
				libvmi.WithInterface(libvmi.InterfaceDeviceWithMasqueradeBinding()),
				libvmi.WithNetwork(v1.DefaultPodNetwork()),
			)
			vmi, err = virtClient.VirtualMachineInstance(testsuite.GetTestNamespace(vmi)).Create(context.Background(), vmi, metav1.CreateOptions{})
			Expect(err).ToNot(HaveOccurred())
			vmim = libmigration.New(vmi.Name, vmi.Namespace)
		})

		It("[test_id:7635]create a migration", func() {
			By("Make a Dry-Run request to create a Migration")
			err = dryRunCreate(restClient, resource, vmim.Namespace, vmim, vmim)
			Expect(err).ToNot(HaveOccurred())

			By("Check that no migration was actually created")
			_, err = virtClient.VirtualMachineInstanceMigration(vmim.Namespace).Get(context.Background(), vmim.Name, metav1.GetOptions{})
			Expect(err).To(MatchError(errors.IsNotFound, "k8serrors.IsNotFound"))
		})

		It("[test_id:7636]delete a migration", func() {
			By("Create a migration")
			vmim, err = virtClient.VirtualMachineInstanceMigration(vmim.Namespace).Create(context.Background(), vmim, metav1.CreateOptions{})
			Expect(err).ToNot(HaveOccurred())

			By("Make a Dry-Run request to delete a Migration")
			deletePolicy := metav1.DeletePropagationForeground
			opts := metav1.DeleteOptions{
				DryRun:            []string{metav1.DryRunAll},
				PropagationPolicy: &deletePolicy,
			}
			err = virtClient.VirtualMachineInstanceMigration(vmim.Namespace).Delete(context.Background(), vmim.Name, opts)
			Expect(err).ToNot(HaveOccurred())

			By("Check that no migration was actually deleted")
			_, err = virtClient.VirtualMachineInstanceMigration(vmim.Namespace).Get(context.Background(), vmim.Name, metav1.GetOptions{})
			Expect(err).ToNot(HaveOccurred())
		})

		It("[test_id:7637]update a migration", func() {
			By("Create a migration")
			vmim, err := virtClient.VirtualMachineInstanceMigration(vmim.Namespace).Create(context.Background(), vmim, metav1.CreateOptions{})
			Expect(err).ToNot(HaveOccurred())

			By("Make a Dry-Run request to update the migration")
			err = retry.RetryOnConflict(retry.DefaultRetry, func() error {
				vmim, err = virtClient.VirtualMachineInstanceMigration(vmim.Namespace).Get(context.Background(), vmim.Name, metav1.GetOptions{})
				if err != nil {
					return err
				}
				vmim.Annotations = map[string]string{
					"key": "42",
				}
				return dryRunUpdate(restClient, resource, vmim.Name, vmim.Namespace, vmim, nil)

			})

			Expect(err).ToNot(HaveOccurred())

			By("Check that no update actually took place")
			vmim, err = virtClient.VirtualMachineInstanceMigration(vmim.Namespace).Get(context.Background(), vmim.Name, metav1.GetOptions{})
			Expect(err).ToNot(HaveOccurred())
			Expect(vmim.Annotations["key"]).ToNot(Equal("42"))
		})

		It("[test_id:7638]patch a migration", func() {
			By("Create a migration")
			vmim, err = virtClient.VirtualMachineInstanceMigration(vmim.Namespace).Create(context.Background(), vmim, metav1.CreateOptions{})
			Expect(err).ToNot(HaveOccurred())

			By("Make a Dry-Run request to patch the migration")
			patch := []byte(`{"metadata": {"labels": {"key": "42"}}}`)
			err = dryRunPatch(restClient, resource, vmim.Name, vmim.Namespace, types.MergePatchType, patch, nil)
			Expect(err).ToNot(HaveOccurred())

			By("Check that no update actually took place")
			vmim, err = virtClient.VirtualMachineInstanceMigration(vmim.Namespace).Get(context.Background(), vmim.Name, metav1.GetOptions{})
			Expect(err).ToNot(HaveOccurred())
			Expect(vmim.Labels["key"]).ToNot(Equal("42"))
		})
	})

	Context("VMI Presets", func() {
		var preset *v1.VirtualMachineInstancePreset
		resource := "virtualmachineinstancepresets"
		presetLabelKey := "kubevirt.io/vmi-preset-test"
		presetLabelVal := "test"

		BeforeEach(func() {
			preset = newVMIPreset("test-vmi-preset", presetLabelKey, presetLabelVal)
		})

		It("[test_id:7639]create a VMI preset", func() {
			By("Make a Dry-Run request to create a VMI preset")
			err = dryRunCreate(restClient, resource, preset.Namespace, preset, nil)
			Expect(err).ToNot(HaveOccurred())

			By("Check that no VMI preset was actually created")
			_, err = virtClient.VirtualMachineInstancePreset(preset.Namespace).Get(context.Background(), preset.Name, metav1.GetOptions{})
			Expect(err).To(MatchError(errors.IsNotFound, "k8serrors.IsNotFound"))
		})

		It("[test_id:7640]delete a VMI preset", func() {
			By("Create a VMI preset")
			_, err := virtClient.VirtualMachineInstancePreset(preset.Namespace).Create(context.Background(), preset, metav1.CreateOptions{})
			Expect(err).ToNot(HaveOccurred())

			By("Make a Dry-Run request to delete a VMI preset")
			deletePolicy := metav1.DeletePropagationForeground
			opts := metav1.DeleteOptions{
				DryRun:            []string{metav1.DryRunAll},
				PropagationPolicy: &deletePolicy,
			}
			err = virtClient.VirtualMachineInstancePreset(preset.Namespace).Delete(context.Background(), preset.Name, opts)
			Expect(err).ToNot(HaveOccurred())

			By("Check that no VMI preset was actually deleted")
			_, err = virtClient.VirtualMachineInstancePreset(preset.Namespace).Get(context.Background(), preset.Name, metav1.GetOptions{})
			Expect(err).ToNot(HaveOccurred())
		})

		It("[test_id:7641]update a VMI preset", func() {
			By("Create a VMI preset")
			_, err = virtClient.VirtualMachineInstancePreset(preset.Namespace).Create(context.Background(), preset, metav1.CreateOptions{})
			Expect(err).ToNot(HaveOccurred())

			By("Make a Dry-Run request to update a VMI preset")
			err = retry.RetryOnConflict(retry.DefaultRetry, func() error {
				preset, err = virtClient.VirtualMachineInstancePreset(preset.Namespace).Get(context.Background(), preset.Name, metav1.GetOptions{})
				if err != nil {
					return err
				}

				preset.Labels = map[string]string{
					"key": "42",
				}
				return dryRunUpdate(restClient, resource, preset.Name, preset.Namespace, preset, nil)
			})
			Expect(err).ToNot(HaveOccurred())

			By("Check that no update actually took place")
			preset, err = virtClient.VirtualMachineInstancePreset(preset.Namespace).Get(context.Background(), preset.Name, metav1.GetOptions{})
			Expect(err).ToNot(HaveOccurred())
			Expect(preset.Labels["key"]).ToNot(Equal("42"))
		})

		It("[test_id:7642]patch a VMI preset", func() {
			By("Create a VMI preset")
			preset, err = virtClient.VirtualMachineInstancePreset(preset.Namespace).Create(context.Background(), preset, metav1.CreateOptions{})
			Expect(err).ToNot(HaveOccurred())

			By("Make a Dry-Run request to patch a VMI preset")
			patch := []byte(`{"metadata": {"labels": {"key": "42"}}}`)
			err = dryRunPatch(restClient, resource, preset.Name, preset.Namespace, types.MergePatchType, patch, nil)
			Expect(err).ToNot(HaveOccurred())

			By("Check that no update actually took place")
			preset, err = virtClient.VirtualMachineInstancePreset(preset.Namespace).Get(context.Background(), preset.Name, metav1.GetOptions{})
			Expect(err).ToNot(HaveOccurred())
			Expect(preset.Labels["key"]).ToNot(Equal("42"))
		})
	})

	Context("VMI ReplicaSets", func() {
		var vmirs *v1.VirtualMachineInstanceReplicaSet
		resource := "virtualmachineinstancereplicasets"

		BeforeEach(func() {
			vmirs = newVMIReplicaSet("test-vmi-rs")
		})

		It("[test_id:7643]create a VMI replicaset", func() {
			By("Make a Dry-Run request to create a VMI replicaset")
			err = dryRunCreate(restClient, resource, vmirs.Namespace, vmirs, nil)
			Expect(err).ToNot(HaveOccurred())

			By("Check that no VMI replicaset was actually created")
			_, err = virtClient.ReplicaSet(vmirs.Namespace).Get(context.Background(), vmirs.Name, metav1.GetOptions{})
			Expect(err).To(MatchError(errors.IsNotFound, "k8serrors.IsNotFound"))
		})

		It("[test_id:7644]delete a VMI replicaset", func() {
			By("Create a VMI replicaset")
			_, err := virtClient.ReplicaSet(vmirs.Namespace).Create(context.Background(), vmirs, metav1.CreateOptions{})
			Expect(err).ToNot(HaveOccurred())

			By("Make a Dry-Run request to delete a VMI replicaset")
			deletePolicy := metav1.DeletePropagationForeground
			opts := metav1.DeleteOptions{
				DryRun:            []string{metav1.DryRunAll},
				PropagationPolicy: &deletePolicy,
			}
			err = virtClient.ReplicaSet(vmirs.Namespace).Delete(context.Background(), vmirs.Name, opts)
			Expect(err).ToNot(HaveOccurred())

			By("Check that no VMI replicaset was actually deleted")
			_, err = virtClient.ReplicaSet(vmirs.Namespace).Get(context.Background(), vmirs.Name, metav1.GetOptions{})
			Expect(err).ToNot(HaveOccurred())
		})

		It("[test_id:7645]update a VMI replicaset", func() {
			By("Create a VMI replicaset")
			_, err = virtClient.ReplicaSet(vmirs.Namespace).Create(context.Background(), vmirs, metav1.CreateOptions{})
			Expect(err).ToNot(HaveOccurred())

			By("Make a Dry-Run request to update a VMI replicaset")
			err = retry.RetryOnConflict(retry.DefaultRetry, func() error {
				vmirs, err = virtClient.ReplicaSet(vmirs.Namespace).Get(context.Background(), vmirs.Name, metav1.GetOptions{})
				if err != nil {
					return err
				}

				vmirs.Labels = map[string]string{
					"key": "42",
				}
				return dryRunUpdate(restClient, resource, vmirs.Name, vmirs.Namespace, vmirs, nil)
			})
			Expect(err).ToNot(HaveOccurred())

			By("Check that no update actually took place")
			vmirs, err = virtClient.ReplicaSet(vmirs.Namespace).Get(context.Background(), vmirs.Name, metav1.GetOptions{})
			Expect(err).ToNot(HaveOccurred())
			Expect(vmirs.Labels["key"]).ToNot(Equal("42"))
		})

		It("[test_id:7646]patch a VMI replicaset", func() {
			By("Create a VMI replicaset")
			vmirs, err = virtClient.ReplicaSet(vmirs.Namespace).Create(context.Background(), vmirs, metav1.CreateOptions{})
			Expect(err).ToNot(HaveOccurred())

			By("Make a Dry-Run request to patch a VMI replicaset")
			patch := []byte(`{"metadata": {"labels": {"key": "42"}}}`)
			err = dryRunPatch(restClient, resource, vmirs.Name, vmirs.Namespace, types.MergePatchType, patch, nil)
			Expect(err).ToNot(HaveOccurred())

			By("Check that no update actually took place")
			vmirs, err = virtClient.ReplicaSet(vmirs.Namespace).Get(context.Background(), vmirs.Name, metav1.GetOptions{})
			Expect(err).ToNot(HaveOccurred())
			Expect(vmirs.Labels["key"]).ToNot(Equal("42"))
		})
	})

	Context("KubeVirt CR", func() {
		var kv *v1.KubeVirt
		resource := "kubevirts"

		BeforeEach(func() {
			kv = libkubevirt.GetCurrentKv(virtClient)
		})

		It("[test_id:7648]delete a KubeVirt CR", Serial, func() {
			By("Make a Dry-Run request to delete a KubeVirt CR")
			deletePolicy := metav1.DeletePropagationForeground
			opts := metav1.DeleteOptions{
				DryRun:            []string{metav1.DryRunAll},
				PropagationPolicy: &deletePolicy,
			}
			err = virtClient.KubeVirt(kv.Namespace).Delete(context.Background(), kv.Name, opts)
			Expect(err).ToNot(HaveOccurred())

			By("Check that no KubeVirt CR was actually deleted")
			_, err = virtClient.KubeVirt(kv.Namespace).Get(context.Background(), kv.Name, metav1.GetOptions{})
			Expect(err).ToNot(HaveOccurred())
		})

		It("[test_id:7649]update a KubeVirt CR", func() {
			By("Make a Dry-Run request to update a KubeVirt CR")
			err = retry.RetryOnConflict(retry.DefaultRetry, func() error {
				kv, err = virtClient.KubeVirt(kv.Namespace).Get(context.Background(), kv.Name, metav1.GetOptions{})
				if err != nil {
					return err
				}

				kv.Labels = map[string]string{
					"key": "42",
				}
				return dryRunUpdate(restClient, resource, kv.Name, kv.Namespace, kv, nil)
			})
			Expect(err).ToNot(HaveOccurred())

			By("Check that no update actually took place")
			kv, err = virtClient.KubeVirt(kv.Namespace).Get(context.Background(), kv.Name, metav1.GetOptions{})
			Expect(err).ToNot(HaveOccurred())
			Expect(kv.Labels["key"]).ToNot(Equal("42"))
		})

		It("[test_id:7650]patch a KubeVirt CR", func() {
			By("Make a Dry-Run request to patch a KubeVirt CR")
			patch := []byte(`{"metadata": {"labels": {"key": "42"}}}`)
			err = dryRunPatch(restClient, resource, kv.Name, kv.Namespace, types.MergePatchType, patch, nil)
			Expect(err).ToNot(HaveOccurred())

			By("Check that no update actually took place")
			kv, err = virtClient.KubeVirt(kv.Namespace).Get(context.Background(), kv.Name, metav1.GetOptions{})
			Expect(err).ToNot(HaveOccurred())
			Expect(kv.Labels["key"]).ToNot(Equal("42"))
		})
	})

	Context("VM Snapshots", func() {
		var snap *snapshotv1.VirtualMachineSnapshot

		BeforeEach(func() {
			config.EnableFeatureGate(featuregate.SnapshotGate)
			vm := libvmi.NewVirtualMachine(libvmifact.NewAlpine())
			vm, err = virtClient.VirtualMachine(testsuite.GetTestNamespace(nil)).Create(context.Background(), vm, metav1.CreateOptions{})
			Expect(err).ToNot(HaveOccurred())

			snap = newVMSnapshot(vm)
		})

		It("[test_id:7651]create a VM Snapshot", func() {
			By("Make a Dry-Run request to create a VM Snapshot")
			opts := metav1.CreateOptions{DryRun: []string{metav1.DryRunAll}}
			_, err = virtClient.VirtualMachineSnapshot(snap.Namespace).Create(context.Background(), snap, opts)
			Expect(err).ToNot(HaveOccurred())

			By("Check that no VM Snapshot was actually created")
			_, err = virtClient.VirtualMachineSnapshot(snap.Namespace).Get(context.Background(), snap.Name, metav1.GetOptions{})
			Expect(err).To(MatchError(errors.IsNotFound, "k8serrors.IsNotFound"))
		})

		It("[test_id:7652]delete a VM Snapshot", func() {
			By("Create a VM Snapshot")
			_, err = virtClient.VirtualMachineSnapshot(snap.Namespace).Create(context.Background(), snap, metav1.CreateOptions{})
			Expect(err).ToNot(HaveOccurred())

			By("Make a Dry-Run request to delete a VM Snapshot")
			deletePolicy := metav1.DeletePropagationForeground
			opts := metav1.DeleteOptions{
				DryRun:            []string{metav1.DryRunAll},
				PropagationPolicy: &deletePolicy,
			}
			err = virtClient.VirtualMachineSnapshot(snap.Namespace).Delete(context.Background(), snap.Name, opts)
			Expect(err).ToNot(HaveOccurred())

			By("Check that no VM Snapshot was actually deleted")
			_, err = virtClient.VirtualMachineSnapshot(snap.Namespace).Get(context.Background(), snap.Name, metav1.GetOptions{})
			Expect(err).ToNot(HaveOccurred())
		})

		It("[test_id:7653]update a VM Snapshot", func() {
			By("Create a VM Snapshot")
			_, err = virtClient.VirtualMachineSnapshot(snap.Namespace).Create(context.Background(), snap, metav1.CreateOptions{})
			Expect(err).ToNot(HaveOccurred())

			By("Make a Dry-Run request to update a VM Snapshot")
			snap, err := virtClient.VirtualMachineSnapshot(snap.Namespace).Get(context.Background(), snap.Name, metav1.GetOptions{})
			Expect(err).ToNot(HaveOccurred())
			patch := []byte(`{"metadata":{"labels":{"key":"42"}}}`)
			opts := metav1.PatchOptions{DryRun: []string{metav1.DryRunAll}}
			_, err = virtClient.VirtualMachineSnapshot(snap.Namespace).Patch(context.Background(), snap.Name, types.MergePatchType, patch, opts)
			Expect(err).ToNot(HaveOccurred())

			By("Check that no update actually took place")
			snap, err = virtClient.VirtualMachineSnapshot(snap.Namespace).Get(context.Background(), snap.Name, metav1.GetOptions{})
			Expect(err).ToNot(HaveOccurred())
			Expect(snap.Labels["key"]).ToNot(Equal("42"))
		})

		It("[test_id:7654]patch a VM Snapshot", func() {
			By("Create a VM Snapshot")
			_, err = virtClient.VirtualMachineSnapshot(snap.Namespace).Create(context.Background(), snap, metav1.CreateOptions{})
			Expect(err).ToNot(HaveOccurred())

			By("Make a Dry-Run request to patch a VM Snapshot")
			patch := []byte(`{"metadata": {"labels": {"key": "42"}}}`)
			opts := metav1.PatchOptions{DryRun: []string{metav1.DryRunAll}}
			_, err = virtClient.VirtualMachineSnapshot(snap.Namespace).Patch(context.Background(), snap.Name, types.MergePatchType, patch, opts)
			Expect(err).ToNot(HaveOccurred())

			By("Check that no update actually took place")
			snap, err = virtClient.VirtualMachineSnapshot(snap.Namespace).Get(context.Background(), snap.Name, metav1.GetOptions{})
			Expect(err).ToNot(HaveOccurred())
			Expect(snap.Labels["key"]).ToNot(Equal("42"))
		})
	})

	Context("VM Restores", func() {
		var restore *snapshotv1.VirtualMachineRestore

		BeforeEach(func() {
			config.EnableFeatureGate(featuregate.SnapshotGate)

			vm := libvmi.NewVirtualMachine(libvmifact.NewAlpine())
			vm, err = virtClient.VirtualMachine(testsuite.GetTestNamespace(nil)).Create(context.Background(), vm, metav1.CreateOptions{})
			Expect(err).ToNot(HaveOccurred())

			snap := newVMSnapshot(vm)
			_, err = virtClient.VirtualMachineSnapshot(snap.Namespace).Create(context.Background(), snap, metav1.CreateOptions{})
			Expect(err).ToNot(HaveOccurred())

			restore = newVMRestore(vm, snap)
			waitForSnapshotToBeReady(virtClient, snap, 120)
		})

		It("[test_id:7655]create a VM Restore", func() {
			By("Make a Dry-Run request to create a VM Restore")
			opts := metav1.CreateOptions{DryRun: []string{metav1.DryRunAll}}
			_, err = virtClient.VirtualMachineRestore(restore.Namespace).Create(context.Background(), restore, opts)
			Expect(err).ToNot(HaveOccurred())

			By("Check that no VM Restore was actually created")
			_, err = virtClient.VirtualMachineRestore(restore.Namespace).Get(context.Background(), restore.Name, metav1.GetOptions{})
			Expect(err).To(MatchError(errors.IsNotFound, "k8serrors.IsNotFound"))
		})

		It("[test_id:7656]delete a VM Restore", func() {
			By("Create a VM Restore")
			_, err = virtClient.VirtualMachineRestore(restore.Namespace).Create(context.Background(), restore, metav1.CreateOptions{})
			Expect(err).ToNot(HaveOccurred())

			By("Make a Dry-Run request to delete a VM Restore")
			deletePolicy := metav1.DeletePropagationForeground
			opts := metav1.DeleteOptions{
				DryRun:            []string{metav1.DryRunAll},
				PropagationPolicy: &deletePolicy,
			}
			err = virtClient.VirtualMachineRestore(restore.Namespace).Delete(context.Background(), restore.Name, opts)
			Expect(err).ToNot(HaveOccurred())

			By("Check that no VM Restore was actually deleted")
			_, err = virtClient.VirtualMachineRestore(restore.Namespace).Get(context.Background(), restore.Name, metav1.GetOptions{})
			Expect(err).ToNot(HaveOccurred())
		})

		It("[test_id:7657]update a VM Restore", func() {
			By("Create a VM Restore")
			_, err = virtClient.VirtualMachineRestore(restore.Namespace).Create(context.Background(), restore, metav1.CreateOptions{})
			Expect(err).ToNot(HaveOccurred())

			By("Make a Dry-Run request to update a VM Restore")
			restore, err := virtClient.VirtualMachineRestore(restore.Namespace).Get(context.Background(), restore.Name, metav1.GetOptions{})
			Expect(err).ToNot(HaveOccurred())

			patch := []byte(`{"metadata":{"labels":{"key":"42"}}}`)
			opts := metav1.PatchOptions{DryRun: []string{metav1.DryRunAll}}
			_, err = virtClient.VirtualMachineRestore(restore.Namespace).Patch(context.Background(), restore.Name, types.MergePatchType, patch, opts)
			Expect(err).ToNot(HaveOccurred())

			By("Check that no update actually took place")
			restore, err = virtClient.VirtualMachineRestore(restore.Namespace).Get(context.Background(), restore.Name, metav1.GetOptions{})
			Expect(err).ToNot(HaveOccurred())
			Expect(restore.Labels["key"]).ToNot(Equal("42"))
		})

		It("[test_id:7658]patch a VM Restore", func() {
			By("Create a VM Restore")
			_, err = virtClient.VirtualMachineRestore(restore.Namespace).Create(context.Background(), restore, metav1.CreateOptions{})
			Expect(err).ToNot(HaveOccurred())

			By("Make a Dry-Run request to patch a VM Restore")
			patch := []byte(`{"metadata": {"labels": {"key": "42"}}}`)
			opts := metav1.PatchOptions{DryRun: []string{metav1.DryRunAll}}
			_, err = virtClient.VirtualMachineRestore(restore.Namespace).Patch(context.Background(), restore.Name, types.MergePatchType, patch, opts)
			Expect(err).ToNot(HaveOccurred())

			By("Check that no update actually took place")
			restore, err = virtClient.VirtualMachineRestore(restore.Namespace).Get(context.Background(), restore.Name, metav1.GetOptions{})
			Expect(err).ToNot(HaveOccurred())
			Expect(restore.Labels["key"]).ToNot(Equal("42"))
		})
	})
})

func newVMIPreset(name, labelKey, labelValue string) *v1.VirtualMachineInstancePreset {
	return &v1.VirtualMachineInstancePreset{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: testsuite.GetTestNamespace(nil),
		},
		Spec: v1.VirtualMachineInstancePresetSpec{
			Domain: &v1.DomainSpec{
				Resources: v1.ResourceRequirements{
					Requests: corev1.ResourceList{
						corev1.ResourceMemory: k8sres.MustParse("512Mi"),
					},
				},
			},
			Selector: metav1.LabelSelector{
				MatchLabels: map[string]string{
					labelKey: labelValue,
				},
			},
		},
	}
}

func newVMIReplicaSet(name string) *v1.VirtualMachineInstanceReplicaSet {
	vmi := libvmifact.NewAlpine(
		libvmi.WithInterface(libvmi.InterfaceDeviceWithMasqueradeBinding()),
		libvmi.WithNetwork(v1.DefaultPodNetwork()),
	)

	return &v1.VirtualMachineInstanceReplicaSet{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: testsuite.GetTestNamespace(nil),
		},
		Spec: v1.VirtualMachineInstanceReplicaSetSpec{
			Selector: &metav1.LabelSelector{
				MatchLabels: map[string]string{
					"kubevirt.io/testrs": "testrs",
				},
			},
			Template: &v1.VirtualMachineInstanceTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: map[string]string{
						"kubevirt.io/testrs": "testrs",
					},
				},
				Spec: vmi.Spec,
			},
		},
	}
}

func newVMSnapshot(vm *v1.VirtualMachine) *snapshotv1.VirtualMachineSnapshot {
	group := vm.GroupVersionKind().Group

	return &snapshotv1.VirtualMachineSnapshot{
		ObjectMeta: metav1.ObjectMeta{
			Name:      vm.Name + "-snapshot",
			Namespace: testsuite.GetTestNamespace(vm),
		},
		Spec: snapshotv1.VirtualMachineSnapshotSpec{
			Source: corev1.TypedLocalObjectReference{
				APIGroup: &group,
				Kind:     vm.GroupVersionKind().Kind,
				Name:     vm.Name,
			},
		},
	}
}

func newVMRestore(vm *v1.VirtualMachine, snapshot *snapshotv1.VirtualMachineSnapshot) *snapshotv1.VirtualMachineRestore {
	group := vm.GroupVersionKind().Group

	return &snapshotv1.VirtualMachineRestore{
		ObjectMeta: metav1.ObjectMeta{
			Name:      vm.Name + "-restore",
			Namespace: testsuite.GetTestNamespace(vm),
		},
		Spec: snapshotv1.VirtualMachineRestoreSpec{
			Target: corev1.TypedLocalObjectReference{
				APIGroup: &group,
				Kind:     vm.GroupVersionKind().Kind,
				Name:     vm.Name,
			},
			VirtualMachineSnapshotName: snapshot.Name,
		},
	}
}

func waitForSnapshotToBeReady(virtClient kubecli.KubevirtClient, snapshot *snapshotv1.VirtualMachineSnapshot, timeoutSec int) {
	By(fmt.Sprintf("Waiting for snapshot %s to be ready to use", snapshot.Name))
	EventuallyWithOffset(1, func() bool {
		updatedSnap, err := virtClient.VirtualMachineSnapshot(snapshot.Namespace).Get(context.Background(), snapshot.Name, metav1.GetOptions{})
		Expect(err).ToNot(HaveOccurred())
		return updatedSnap.Status != nil && updatedSnap.Status.ReadyToUse != nil && *updatedSnap.Status.ReadyToUse
	}, time.Duration(timeoutSec)*time.Second, 2).Should(BeTrue(), "Should be ready to use")
}

func dryRunCreate(client *rest.RESTClient, resource, namespace string, obj interface{}, result runtime.Object) error {
	opts := metav1.CreateOptions{DryRun: []string{metav1.DryRunAll}}
	return client.Post().
		Namespace(namespace).
		Resource(resource).
		VersionedParams(&opts, scheme.ParameterCodec).
		Body(obj).
		Do(context.Background()).
		Into(result)
}

func dryRunUpdate(client *rest.RESTClient, resource, name, namespace string, obj interface{}, result runtime.Object) error {
	opts := metav1.UpdateOptions{DryRun: []string{metav1.DryRunAll}}
	return client.Put().
		Name(name).
		Namespace(namespace).
		Resource(resource).
		VersionedParams(&opts, scheme.ParameterCodec).
		Body(obj).
		Do(context.Background()).
		Into(result)
}

func dryRunPatch(client *rest.RESTClient, resource, name, namespace string, pt types.PatchType, data []byte, result runtime.Object) error {
	opts := metav1.PatchOptions{DryRun: []string{metav1.DryRunAll}}
	return client.Patch(pt).
		Name(name).
		Namespace(namespace).
		Resource(resource).
		VersionedParams(&opts, scheme.ParameterCodec).
		Body(data).
		Do(context.Background()).
		Into(result)
}
