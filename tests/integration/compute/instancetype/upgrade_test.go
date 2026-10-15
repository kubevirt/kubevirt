/*
This file is part of the KubeVirt project

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.

Copyright The KubeVirt Authors.
*/

package instancetype_test

import (
	"context"
	"fmt"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	appsv1 "k8s.io/api/apps/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"

	virtv1 "kubevirt.io/api/core/v1"
	instancetypeapi "kubevirt.io/api/instancetype"
	instancetypev1beta1 "kubevirt.io/api/instancetype/v1beta1"
	generatedscheme "kubevirt.io/client-go/kubevirt/scheme"

	"kubevirt.io/kubevirt/pkg/apimachinery/patch"
	"kubevirt.io/kubevirt/pkg/instancetype/revision"
	"kubevirt.io/kubevirt/pkg/instancetype/upgrade"
	"kubevirt.io/kubevirt/pkg/libvmi"
	"kubevirt.io/kubevirt/pkg/pointer"
	"kubevirt.io/kubevirt/pkg/util/gvk"
	"kubevirt.io/kubevirt/tests/framework/matcher"
)

//nolint:funlen // Register the related Ginkgo scenarios in the shared container.
func (s *instancetypeSuite) describeUpgrades() {
	Context("ControllerRevision upgrades", func() {
		var ctx context.Context
		var vm *virtv1.VirtualMachine

		BeforeEach(func(specCtx SpecContext) {
			ctx = specCtx
			instancetype := &instancetypev1beta1.VirtualMachineInstancetype{
				ObjectMeta: metav1.ObjectMeta{
					GenerateName: "instancetype-",
					Namespace:    s.namespace,
				},
				Spec: instancetypev1beta1.VirtualMachineInstancetypeSpec{
					CPU: instancetypev1beta1.CPUInstancetype{Guest: 1},
					Memory: instancetypev1beta1.MemoryInstancetype{
						Guest: resource.MustParse("128Mi"),
					},
				},
			}
			var err error
			instancetype, err = s.f.VirtClient().VirtualMachineInstancetype(s.namespace).Create(ctx, instancetype, metav1.CreateOptions{})
			Expect(err).ToNot(HaveOccurred())

			preference := &instancetypev1beta1.VirtualMachinePreference{
				ObjectMeta: metav1.ObjectMeta{
					GenerateName: "preference-",
					Namespace:    s.namespace,
				},
			}
			preference, err = s.f.VirtClient().VirtualMachinePreference(s.namespace).Create(ctx, preference, metav1.CreateOptions{})
			Expect(err).ToNot(HaveOccurred())

			vm = libvmi.NewVirtualMachine(
				libvmi.New(libvmi.WithMemoryRequest("128Mi")),
				libvmi.WithInstancetype(instancetype.Name),
				libvmi.WithPreference(preference.Name),
			)
			vm, err = s.f.VirtClient().VirtualMachine(s.namespace).Create(ctx, vm, metav1.CreateOptions{})
			Expect(err).ToNot(HaveOccurred())

			Eventually(ctx, matcher.ThisVM(vm), 30*time.Second, time.Second).Should(matcher.HaveControllerRevisionRefs())
		})

		createControllerRevision := func(obj runtime.Object) (*appsv1.ControllerRevision, error) {
			cr, err := revision.CreateControllerRevision(vm, obj)
			Expect(err).ToNot(HaveOccurred())
			return s.f.VirtClient().AppsV1().ControllerRevisions(s.namespace).Create(ctx, cr, metav1.CreateOptions{})
		}

		createLegacyControllerRevision := func(obj runtime.Object) (*appsv1.ControllerRevision, error) {
			cr, err := revision.CreateControllerRevision(vm, obj)
			Expect(err).ToNot(HaveOccurred())

			obj, err = gvk.GenerateKubeVirtGroupVersionKind(obj)
			Expect(err).ToNot(HaveOccurred())
			metaObj, ok := obj.(metav1.Object)
			Expect(ok).To(BeTrue())
			cr.Name = fmt.Sprintf("%s-%s-%s-%d", vm.Name, metaObj.GetName(), metaObj.GetUID(), metaObj.GetGeneration())

			Expect(cr.Labels).To(HaveKey(instancetypeapi.ControllerRevisionObjectVersionLabel))
			delete(cr.Labels, instancetypeapi.ControllerRevisionObjectVersionLabel)

			return s.f.VirtClient().AppsV1().ControllerRevisions(s.namespace).Create(ctx, cr, metav1.CreateOptions{})
		}

		updateInstancetypeMatcher := func(revisionName string) {
			b, err := patch.New(patch.WithAdd("/status/instancetypeRef/controllerRevisionRef/name", revisionName)).GeneratePayload()
			Expect(err).ToNot(HaveOccurred())
			_, err = s.f.VirtClient().VirtualMachine(s.namespace).PatchStatus(ctx, vm.Name, types.JSONPatchType, b, metav1.PatchOptions{})
			Expect(err).ToNot(HaveOccurred())
		}

		updatePreferenceMatcher := func(revisionName string) {
			b, err := patch.New(patch.WithAdd("/status/preferenceRef/controllerRevisionRef/name", revisionName)).GeneratePayload()
			Expect(err).ToNot(HaveOccurred())
			_, err = s.f.VirtClient().VirtualMachine(s.namespace).PatchStatus(ctx, vm.Name, types.JSONPatchType, b, metav1.PatchOptions{})
			Expect(err).ToNot(HaveOccurred())
		}

		getInstancetypeRevisionName := func() string {
			var err error
			vm, err = s.f.VirtClient().VirtualMachine(s.namespace).Get(ctx, vm.Name, metav1.GetOptions{})
			Expect(err).ToNot(HaveOccurred())
			Expect(revision.HasControllerRevisionRef(vm.Status.InstancetypeRef)).To(BeTrue())
			Expect(vm.Status.InstancetypeRef.ControllerRevisionRef.Name).ToNot(BeEmpty())
			return vm.Status.InstancetypeRef.ControllerRevisionRef.Name
		}

		getPreferenceRevisionName := func() string {
			var err error
			vm, err = s.f.VirtClient().VirtualMachine(s.namespace).Get(ctx, vm.Name, metav1.GetOptions{})
			Expect(err).ToNot(HaveOccurred())
			Expect(revision.HasControllerRevisionRef(vm.Status.PreferenceRef)).To(BeTrue())
			Expect(vm.Status.PreferenceRef.ControllerRevisionRef.Name).ToNot(BeEmpty())
			return vm.Status.PreferenceRef.ControllerRevisionRef.Name
		}

		createInstancetype := func() runtime.Object {
			it := &instancetypev1beta1.VirtualMachineInstancetype{
				ObjectMeta: metav1.ObjectMeta{GenerateName: "instancetype-", Namespace: s.namespace},
				Spec: instancetypev1beta1.VirtualMachineInstancetypeSpec{
					CPU:    instancetypev1beta1.CPUInstancetype{Guest: 1},
					Memory: instancetypev1beta1.MemoryInstancetype{Guest: resource.MustParse("128Mi")},
				},
			}
			it, err := s.f.VirtClient().VirtualMachineInstancetype(s.namespace).Create(ctx, it, metav1.CreateOptions{})
			Expect(err).ToNot(HaveOccurred())
			return it
		}

		createClusterInstancetype := func() runtime.Object {
			it := &instancetypev1beta1.VirtualMachineClusterInstancetype{
				ObjectMeta: metav1.ObjectMeta{GenerateName: "clusterinstancetype-"},
				Spec: instancetypev1beta1.VirtualMachineInstancetypeSpec{
					CPU:    instancetypev1beta1.CPUInstancetype{Guest: 1},
					Memory: instancetypev1beta1.MemoryInstancetype{Guest: resource.MustParse("128Mi")},
				},
			}
			it, err := s.f.VirtClient().VirtualMachineClusterInstancetype().Create(ctx, it, metav1.CreateOptions{})
			Expect(err).ToNot(HaveOccurred())
			DeferCleanup(func(ctx SpecContext) error {
				return s.f.VirtClient().VirtualMachineClusterInstancetype().Delete(ctx, it.Name, metav1.DeleteOptions{})
			})
			return it
		}

		createPreference := func() runtime.Object {
			pref := &instancetypev1beta1.VirtualMachinePreference{
				ObjectMeta: metav1.ObjectMeta{GenerateName: "preference-", Namespace: s.namespace},
				Spec: instancetypev1beta1.VirtualMachinePreferenceSpec{
					CPU: &instancetypev1beta1.CPUPreferences{PreferredCPUTopology: pointer.P(instancetypev1beta1.Sockets)},
				},
			}
			pref, err := s.f.VirtClient().VirtualMachinePreference(s.namespace).Create(ctx, pref, metav1.CreateOptions{})
			Expect(err).ToNot(HaveOccurred())
			return pref
		}

		createClusterPreference := func() runtime.Object {
			pref := &instancetypev1beta1.VirtualMachineClusterPreference{
				ObjectMeta: metav1.ObjectMeta{GenerateName: "clusterpreference-"},
				Spec: instancetypev1beta1.VirtualMachinePreferenceSpec{
					CPU: &instancetypev1beta1.CPUPreferences{PreferredCPUTopology: pointer.P(instancetypev1beta1.Sockets)},
				},
			}
			pref, err := s.f.VirtClient().VirtualMachineClusterPreference().Create(ctx, pref, metav1.CreateOptions{})
			Expect(err).ToNot(HaveOccurred())
			DeferCleanup(func(ctx SpecContext) error {
				return s.f.VirtClient().VirtualMachineClusterPreference().Delete(ctx, pref.Name, metav1.DeleteOptions{})
			})
			return pref
		}

		DescribeTable(
			"should upgrade", func(
				specCtx SpecContext,
				generateObject func() runtime.Object,
				generateControllerRevision func(runtime.Object) (*appsv1.ControllerRevision, error),
				updateMatcher func(string),
				getVMRevisionName func() string,
			) {
				ctx = specCtx
				originalRevisionName := getVMRevisionName()

				By("Generating the target ControllerRevision")
				cr, err := generateControllerRevision(generateObject())
				Expect(err).ToNot(HaveOccurred())

				By("Updating the VirtualMachine to reference the generated ControllerRevision")
				originalTestRevisionName := cr.Name
				updateMatcher(originalTestRevisionName)

				By("Waiting for the ControllerRevision to be upgraded to the latest version")
				var revisionName string
				Eventually(ctx, func(g Gomega) {
					revisionName = getVMRevisionName()
					g.Expect(revisionName).ToNot(Equal(originalRevisionName))

					cr, err = s.f.VirtClient().AppsV1().ControllerRevisions(s.namespace).Get(ctx, revisionName, metav1.GetOptions{})
					g.Expect(err).ToNot(HaveOccurred())

					g.Expect(upgrade.IsObjectLatestVersion(cr)).To(BeTrue())

					decodedObj, err := runtime.Decode(generatedscheme.Codecs.UniversalDeserializer(), cr.Data.Raw)
					g.Expect(err).ToNot(HaveOccurred())
					g.Expect(decodedObj.GetObjectKind().GroupVersionKind().Version).To(Equal(instancetypeapi.LatestVersion))
				}, 30*time.Second, time.Second).Should(Succeed())

				if originalTestRevisionName != revisionName {
					Eventually(ctx, func() error {
						_, err := s.f.VirtClient().AppsV1().ControllerRevisions(s.namespace).Get(ctx, originalTestRevisionName, metav1.GetOptions{})
						return err
					}, 30*time.Second, time.Second).Should(MatchError(errors.IsNotFound, "errors.IsNotFound"))
				}
			},
			Entry(
				"VirtualMachineInstancetype from v1beta1 without labels to latest",
				createInstancetype,
				createLegacyControllerRevision,
				updateInstancetypeMatcher,
				getInstancetypeRevisionName,
			),
			Entry(
				"VirtualMachineInstancetype from v1beta1 with labels to latest",
				createInstancetype,
				createControllerRevision,
				updateInstancetypeMatcher,
				getInstancetypeRevisionName,
			),
			Entry(
				"VirtualMachineClusterInstancetype from v1beta1 without labels to latest",
				createClusterInstancetype,
				createLegacyControllerRevision,
				updateInstancetypeMatcher,
				getInstancetypeRevisionName,
			),
			Entry(
				"VirtualMachineClusterInstancetype from v1beta1 with labels to latest",
				createClusterInstancetype,
				createControllerRevision,
				updateInstancetypeMatcher,
				getInstancetypeRevisionName,
			),
			Entry(
				"VirtualMachinePreference from v1beta1 without labels to latest",
				createPreference,
				createLegacyControllerRevision,
				updatePreferenceMatcher,
				getPreferenceRevisionName,
			),
			Entry(
				"VirtualMachinePreference from v1beta1 with labels to latest",
				createPreference,
				createControllerRevision,
				updatePreferenceMatcher,
				getPreferenceRevisionName,
			),
			Entry(
				"VirtualMachineClusterPreference from v1beta1 without labels to latest",
				createClusterPreference,
				createLegacyControllerRevision,
				updatePreferenceMatcher,
				getPreferenceRevisionName,
			),
			Entry(
				"VirtualMachineClusterPreference from v1beta1 with labels to latest",
				createClusterPreference,
				createControllerRevision,
				updatePreferenceMatcher,
				getPreferenceRevisionName,
			),
		)
	})
}
