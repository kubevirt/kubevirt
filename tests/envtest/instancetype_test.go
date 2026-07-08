package envtest_test

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
	utils "kubevirt.io/kubevirt/pkg/util"
	"kubevirt.io/kubevirt/tests/envtest/framework"
	"kubevirt.io/kubevirt/tests/framework/matcher"
)

var _ = Describe("Instancetype", func() {
	var f *framework.Framework
	var ctx context.Context

	Context("ControllerRevision upgrades", Ordered, func() {
		var vm *virtv1.VirtualMachine

		BeforeAll(func() {
			ctx = context.Background()
			f = framework.New()
			f.Start()

			instancetype := &instancetypev1beta1.VirtualMachineInstancetype{
				ObjectMeta: metav1.ObjectMeta{
					GenerateName: "instancetype-",
					Namespace:    "default",
				},
				Spec: instancetypev1beta1.VirtualMachineInstancetypeSpec{
					CPU: instancetypev1beta1.CPUInstancetype{Guest: 1},
					Memory: instancetypev1beta1.MemoryInstancetype{
						Guest: resource.MustParse("128Mi"),
					},
				},
			}
			var err error
			instancetype, err = f.VirtClient().VirtualMachineInstancetype("default").Create(ctx, instancetype, metav1.CreateOptions{})
			Expect(err).ToNot(HaveOccurred())

			preference := &instancetypev1beta1.VirtualMachinePreference{
				ObjectMeta: metav1.ObjectMeta{
					GenerateName: "preference-",
					Namespace:    "default",
				},
			}
			preference, err = f.VirtClient().VirtualMachinePreference("default").Create(ctx, preference, metav1.CreateOptions{})
			Expect(err).ToNot(HaveOccurred())

			vm = libvmi.NewVirtualMachine(
				libvmi.New(libvmi.WithResourceMemory("128Mi")),
				libvmi.WithInstancetype(instancetype.Name),
				libvmi.WithPreference(preference.Name),
			)
			vm, err = f.VirtClient().VirtualMachine("default").Create(ctx, vm, metav1.CreateOptions{})
			Expect(err).ToNot(HaveOccurred())

			Eventually(matcher.ThisVM(vm), 30*time.Second, time.Second).Should(matcher.HaveControllerRevisionRefs())
		})

		AfterAll(func() {
			f.Stop()
		})

		createControllerRevision := func(obj runtime.Object) (*appsv1.ControllerRevision, error) {
			cr, err := revision.CreateControllerRevision(vm, obj)
			Expect(err).ToNot(HaveOccurred())
			return f.VirtClient().AppsV1().ControllerRevisions("default").Create(ctx, cr, metav1.CreateOptions{})
		}

		createLegacyControllerRevision := func(obj runtime.Object) (*appsv1.ControllerRevision, error) {
			cr, err := revision.CreateControllerRevision(vm, obj)
			Expect(err).ToNot(HaveOccurred())

			obj, err = utils.GenerateKubeVirtGroupVersionKind(obj)
			Expect(err).ToNot(HaveOccurred())
			metaObj, ok := obj.(metav1.Object)
			Expect(ok).To(BeTrue())
			cr.Name = fmt.Sprintf("%s-%s-%s-%d", vm.Name, metaObj.GetName(), metaObj.GetUID(), metaObj.GetGeneration())

			Expect(cr.Labels).To(HaveKey(instancetypeapi.ControllerRevisionObjectVersionLabel))
			delete(cr.Labels, instancetypeapi.ControllerRevisionObjectVersionLabel)

			return f.VirtClient().AppsV1().ControllerRevisions("default").Create(ctx, cr, metav1.CreateOptions{})
		}

		updateInstancetypeMatcher := func(revisionName string) {
			b, err := patch.New(patch.WithAdd("/status/instancetypeRef/controllerRevisionRef/name", revisionName)).GeneratePayload()
			Expect(err).ToNot(HaveOccurred())
			_, err = f.VirtClient().VirtualMachine("default").PatchStatus(ctx, vm.Name, types.JSONPatchType, b, metav1.PatchOptions{})
			Expect(err).ToNot(HaveOccurred())
		}

		updatePreferenceMatcher := func(revisionName string) {
			b, err := patch.New(patch.WithAdd("/status/preferenceRef/controllerRevisionRef/name", revisionName)).GeneratePayload()
			Expect(err).ToNot(HaveOccurred())
			_, err = f.VirtClient().VirtualMachine("default").PatchStatus(ctx, vm.Name, types.JSONPatchType, b, metav1.PatchOptions{})
			Expect(err).ToNot(HaveOccurred())
		}

		getInstancetypeRevisionName := func() string {
			var err error
			vm, err = f.VirtClient().VirtualMachine("default").Get(ctx, vm.Name, metav1.GetOptions{})
			Expect(err).ToNot(HaveOccurred())
			Expect(revision.HasControllerRevisionRef(vm.Status.InstancetypeRef)).ToNot(BeNil())
			Expect(vm.Status.InstancetypeRef.ControllerRevisionRef.Name).ToNot(BeEmpty())
			return vm.Status.InstancetypeRef.ControllerRevisionRef.Name
		}

		getPreferenceRevisionName := func() string {
			var err error
			vm, err = f.VirtClient().VirtualMachine("default").Get(ctx, vm.Name, metav1.GetOptions{})
			Expect(err).ToNot(HaveOccurred())
			Expect(revision.HasControllerRevisionRef(vm.Status.PreferenceRef)).ToNot(BeNil())
			Expect(vm.Status.PreferenceRef.ControllerRevisionRef.Name).ToNot(BeEmpty())
			return vm.Status.PreferenceRef.ControllerRevisionRef.Name
		}

		DescribeTable("should upgrade", func(generateControllerRevision func() (*appsv1.ControllerRevision, error), updateMatcher func(string), getVMRevisionName func() string) {
			originalRevisionName := getVMRevisionName()

			By("Generating the target ControllerRevision")
			cr, err := generateControllerRevision()
			Expect(err).ToNot(HaveOccurred())

			By("Updating the VirtualMachine to reference the generated ControllerRevision")
			originalTestRevisionName := cr.Name
			updateMatcher(originalTestRevisionName)

			By("Waiting for the ControllerRevision to be upgraded to the latest version")
			var revisionName string
			Eventually(func(g Gomega) {
				revisionName = getVMRevisionName()
				g.Expect(revisionName).ToNot(Equal(originalRevisionName))

				cr, err = f.VirtClient().AppsV1().ControllerRevisions("default").Get(ctx, revisionName, metav1.GetOptions{})
				g.Expect(err).ToNot(HaveOccurred())

				g.Expect(upgrade.IsObjectLatestVersion(cr)).To(BeTrue())

				decodedObj, err := runtime.Decode(generatedscheme.Codecs.UniversalDeserializer(), cr.Data.Raw)
				Expect(err).ToNot(HaveOccurred())
				Expect(decodedObj.GetObjectKind().GroupVersionKind().Version).To(Equal(instancetypeapi.LatestVersion))
			}, 30*time.Second, time.Second).Should(Succeed())

			if originalTestRevisionName != revisionName {
				Eventually(func() error {
					_, err := f.VirtClient().AppsV1().ControllerRevisions("default").Get(ctx, originalTestRevisionName, metav1.GetOptions{})
					return err
				}, 30*time.Second, time.Second).Should(MatchError(errors.IsNotFound, "errors.IsNotFound"))
			}
		},
			Entry("VirtualMachineInstancetype from v1beta1 without labels to latest",
				func() (*appsv1.ControllerRevision, error) {
					it := &instancetypev1beta1.VirtualMachineInstancetype{
						ObjectMeta: metav1.ObjectMeta{GenerateName: "instancetype-", Namespace: "default"},
						Spec: instancetypev1beta1.VirtualMachineInstancetypeSpec{
							CPU:    instancetypev1beta1.CPUInstancetype{Guest: 1},
							Memory: instancetypev1beta1.MemoryInstancetype{Guest: resource.MustParse("128Mi")},
						},
					}
					it, err := f.VirtClient().VirtualMachineInstancetype("default").Create(ctx, it, metav1.CreateOptions{})
					Expect(err).ToNot(HaveOccurred())
					return createLegacyControllerRevision(it)
				},
				updateInstancetypeMatcher,
				getInstancetypeRevisionName,
			),
			Entry("VirtualMachineInstancetype from v1beta1 with labels to latest",
				func() (*appsv1.ControllerRevision, error) {
					it := &instancetypev1beta1.VirtualMachineInstancetype{
						ObjectMeta: metav1.ObjectMeta{GenerateName: "instancetype-", Namespace: "default"},
						Spec: instancetypev1beta1.VirtualMachineInstancetypeSpec{
							CPU:    instancetypev1beta1.CPUInstancetype{Guest: 1},
							Memory: instancetypev1beta1.MemoryInstancetype{Guest: resource.MustParse("128Mi")},
						},
					}
					it, err := f.VirtClient().VirtualMachineInstancetype("default").Create(ctx, it, metav1.CreateOptions{})
					Expect(err).ToNot(HaveOccurred())
					return createControllerRevision(it)
				},
				updateInstancetypeMatcher,
				getInstancetypeRevisionName,
			),
			Entry("VirtualMachineClusterInstancetype from v1beta1 without labels to latest",
				func() (*appsv1.ControllerRevision, error) {
					it := &instancetypev1beta1.VirtualMachineClusterInstancetype{
						ObjectMeta: metav1.ObjectMeta{GenerateName: "clusterinstancetype-"},
						Spec: instancetypev1beta1.VirtualMachineInstancetypeSpec{
							CPU:    instancetypev1beta1.CPUInstancetype{Guest: 1},
							Memory: instancetypev1beta1.MemoryInstancetype{Guest: resource.MustParse("128Mi")},
						},
					}
					it, err := f.VirtClient().VirtualMachineClusterInstancetype().Create(ctx, it, metav1.CreateOptions{})
					Expect(err).ToNot(HaveOccurred())
					return createLegacyControllerRevision(it)
				},
				updateInstancetypeMatcher,
				getInstancetypeRevisionName,
			),
			Entry("VirtualMachineClusterInstancetype from v1beta1 with labels to latest",
				func() (*appsv1.ControllerRevision, error) {
					it := &instancetypev1beta1.VirtualMachineClusterInstancetype{
						ObjectMeta: metav1.ObjectMeta{GenerateName: "clusterinstancetype-"},
						Spec: instancetypev1beta1.VirtualMachineInstancetypeSpec{
							CPU:    instancetypev1beta1.CPUInstancetype{Guest: 1},
							Memory: instancetypev1beta1.MemoryInstancetype{Guest: resource.MustParse("128Mi")},
						},
					}
					it, err := f.VirtClient().VirtualMachineClusterInstancetype().Create(ctx, it, metav1.CreateOptions{})
					Expect(err).ToNot(HaveOccurred())
					return createControllerRevision(it)
				},
				updateInstancetypeMatcher,
				getInstancetypeRevisionName,
			),
			Entry("VirtualMachinePreference from v1beta1 without labels to latest",
				func() (*appsv1.ControllerRevision, error) {
					pref := &instancetypev1beta1.VirtualMachinePreference{
						ObjectMeta: metav1.ObjectMeta{GenerateName: "preference-", Namespace: "default"},
						Spec: instancetypev1beta1.VirtualMachinePreferenceSpec{
							CPU: &instancetypev1beta1.CPUPreferences{PreferredCPUTopology: toPtr(instancetypev1beta1.Sockets)},
						},
					}
					pref, err := f.VirtClient().VirtualMachinePreference("default").Create(ctx, pref, metav1.CreateOptions{})
					Expect(err).ToNot(HaveOccurred())
					return createLegacyControllerRevision(pref)
				},
				updatePreferenceMatcher,
				getPreferenceRevisionName,
			),
			Entry("VirtualMachinePreference from v1beta1 with labels to latest",
				func() (*appsv1.ControllerRevision, error) {
					pref := &instancetypev1beta1.VirtualMachinePreference{
						ObjectMeta: metav1.ObjectMeta{GenerateName: "preference-", Namespace: "default"},
						Spec: instancetypev1beta1.VirtualMachinePreferenceSpec{
							CPU: &instancetypev1beta1.CPUPreferences{PreferredCPUTopology: toPtr(instancetypev1beta1.Sockets)},
						},
					}
					pref, err := f.VirtClient().VirtualMachinePreference("default").Create(ctx, pref, metav1.CreateOptions{})
					Expect(err).ToNot(HaveOccurred())
					return createControllerRevision(pref)
				},
				updatePreferenceMatcher,
				getPreferenceRevisionName,
			),
			Entry("VirtualMachineClusterPreference from v1beta1 without labels to latest",
				func() (*appsv1.ControllerRevision, error) {
					pref := &instancetypev1beta1.VirtualMachineClusterPreference{
						ObjectMeta: metav1.ObjectMeta{GenerateName: "clusterpreference-"},
						Spec: instancetypev1beta1.VirtualMachinePreferenceSpec{
							CPU: &instancetypev1beta1.CPUPreferences{PreferredCPUTopology: toPtr(instancetypev1beta1.Sockets)},
						},
					}
					pref, err := f.VirtClient().VirtualMachineClusterPreference().Create(ctx, pref, metav1.CreateOptions{})
					Expect(err).ToNot(HaveOccurred())
					return createLegacyControllerRevision(pref)
				},
				updatePreferenceMatcher,
				getPreferenceRevisionName,
			),
			Entry("VirtualMachineClusterPreference from v1beta1 with labels to latest",
				func() (*appsv1.ControllerRevision, error) {
					pref := &instancetypev1beta1.VirtualMachineClusterPreference{
						ObjectMeta: metav1.ObjectMeta{GenerateName: "clusterpreference-"},
						Spec: instancetypev1beta1.VirtualMachinePreferenceSpec{
							CPU: &instancetypev1beta1.CPUPreferences{PreferredCPUTopology: toPtr(instancetypev1beta1.Sockets)},
						},
					}
					pref, err := f.VirtClient().VirtualMachineClusterPreference().Create(ctx, pref, metav1.CreateOptions{})
					Expect(err).ToNot(HaveOccurred())
					return createControllerRevision(pref)
				},
				updatePreferenceMatcher,
				getPreferenceRevisionName,
			),
		)
	})
})

func toPtr[T any](v T) *T {
	return &v
}
