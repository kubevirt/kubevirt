//nolint:dupl,lll
package instancetype

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	appsv1 "k8s.io/api/apps/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"

	virtv1 "kubevirt.io/api/core/v1"
	instancetypeapi "kubevirt.io/api/instancetype"
	instancetypev1alpha1 "kubevirt.io/api/instancetype/v1alpha1"
	instancetypev1alpha2 "kubevirt.io/api/instancetype/v1alpha2"
	instancetypev1beta1 "kubevirt.io/api/instancetype/v1beta1"
	generatedscheme "kubevirt.io/client-go/generated/kubevirt/clientset/versioned/scheme"
	"kubevirt.io/client-go/kubecli"

	"kubevirt.io/kubevirt/pkg/libvmi"
	utils "kubevirt.io/kubevirt/pkg/util"

	instancetypepkg "kubevirt.io/kubevirt/pkg/instancetype"

	"kubevirt.io/kubevirt/tests/decorators"
	"kubevirt.io/kubevirt/tests/framework/kubevirt"
	"kubevirt.io/kubevirt/tests/libinstancetype"
	"kubevirt.io/kubevirt/tests/libinstancetype/builder"
	builderv1alpha1 "kubevirt.io/kubevirt/tests/libinstancetype/builder/v1alpha1"
	builderv1alpha2 "kubevirt.io/kubevirt/tests/libinstancetype/builder/v1alpha2"
	"kubevirt.io/kubevirt/tests/libvmifact"
	"kubevirt.io/kubevirt/tests/testsuite"
)

var _ = Describe("[crit:medium][vendor:cnv-qe@redhat.com][level:component][sig-compute] Instance type and preference ControllerRevision Upgrades", decorators.SigCompute, func() {
	var virtClient kubecli.KubevirtClient

	BeforeEach(func() {
		virtClient = kubevirt.Client()
	})

	var vm *virtv1.VirtualMachine

	createControllerRevision := func(obj runtime.Object) (*appsv1.ControllerRevision, error) {
		cr, err := instancetypepkg.CreateControllerRevision(vm, obj)
		Expect(err).ToNot(HaveOccurred())
		return virtClient.AppsV1().ControllerRevisions(vm.Namespace).Create(context.Background(), cr, metav1.CreateOptions{})
	}

	generateLegacyCRName := func(obj runtime.Object) string {
		obj, err := utils.GenerateKubeVirtGroupVersionKind(obj)
		Expect(err).ToNot(HaveOccurred())
		metaObj, ok := obj.(metav1.Object)
		Expect(ok).To(BeTrue())
		return fmt.Sprintf("%s-%s-%s-%d", vm.Name, metaObj.GetName(), metaObj.GetUID(), metaObj.GetGeneration())
	}

	createLegacyControllerRevision := func(obj runtime.Object) (*appsv1.ControllerRevision, error) {
		cr, err := instancetypepkg.CreateControllerRevision(vm, obj)
		Expect(err).ToNot(HaveOccurred())

		// The legacy naming convention did not include the object version so replace that here
		cr.Name = generateLegacyCRName(obj)

		// The legacy CRs also didn't include a version label so also remove that
		Expect(cr.Labels).To(HaveKey(instancetypeapi.ControllerRevisionObjectVersionLabel))
		delete(cr.Labels, instancetypeapi.ControllerRevisionObjectVersionLabel)

		return virtClient.AppsV1().ControllerRevisions(vm.Namespace).Create(context.Background(), cr, metav1.CreateOptions{})
	}

	updateInstancetypeMatcher := func(revisionName string) {
		Eventually(func(g Gomega) {
			var err error
			vm, err = virtClient.VirtualMachine(testsuite.GetTestNamespace(vm)).Get(context.Background(), vm.Name, metav1.GetOptions{})
			g.Expect(err).ToNot(HaveOccurred())

			vm.Spec.Instancetype.RevisionName = revisionName

			_, err = virtClient.VirtualMachine(vm.Namespace).Update(context.Background(), vm, metav1.UpdateOptions{})
			g.Expect(err).ToNot(HaveOccurred())
		}, 30*time.Second, time.Second).Should(Succeed())
	}

	updatePreferenceMatcher := func(revisionName string) {
		Eventually(func(g Gomega) {
			var err error
			vm, err = virtClient.VirtualMachine(testsuite.GetTestNamespace(vm)).Get(context.Background(), vm.Name, metav1.GetOptions{})
			g.Expect(err).ToNot(HaveOccurred())

			vm.Spec.Preference.RevisionName = revisionName

			_, err = virtClient.VirtualMachine(vm.Namespace).Update(context.Background(), vm, metav1.UpdateOptions{})
			g.Expect(err).ToNot(HaveOccurred())
		}, 30*time.Second, time.Second).Should(Succeed())
	}

	getInstancetypeRevisionName := func() string {
		var err error
		vm, err = virtClient.VirtualMachine(testsuite.GetTestNamespace(vm)).Get(context.Background(), vm.Name, metav1.GetOptions{})
		Expect(err).ToNot(HaveOccurred())
		Expect(vm.Spec.Instancetype).ToNot(BeNil())
		Expect(vm.Spec.Instancetype.RevisionName).ToNot(BeEmpty())
		return vm.Spec.Instancetype.RevisionName
	}

	getPreferenceRevisionName := func() string {
		var err error
		vm, err = virtClient.VirtualMachine(testsuite.GetTestNamespace(vm)).Get(context.Background(), vm.Name, metav1.GetOptions{})
		Expect(err).ToNot(HaveOccurred())
		Expect(vm.Spec.Preference).ToNot(BeNil())
		Expect(vm.Spec.Preference.RevisionName).ToNot(BeEmpty())
		return vm.Spec.Preference.RevisionName
	}

	BeforeEach(func() {
		// We create a fake instance type and preference here just to allow for
		// the creation of the initial VirtualMachine. This then allows the
		// creation of a ControllerRevision later on in the test to use the now
		// created VirtualMachine as an OwnerReference.
		instancetype := builder.NewInstancetype(
			builder.WithCPUs(uint32(1)),
			builder.WithMemory(resource.MustParse("128Mi")),
		)
		instancetype, err := virtClient.VirtualMachineInstancetype(testsuite.GetTestNamespace(instancetype)).Create(context.Background(), instancetype, metav1.CreateOptions{})
		Expect(err).ToNot(HaveOccurred())

		preference := builder.NewPreference()
		preference, err = virtClient.VirtualMachinePreference(testsuite.GetTestNamespace(preference)).Create(context.Background(), preference, metav1.CreateOptions{})
		Expect(err).ToNot(HaveOccurred())

		vm = libvmi.NewVirtualMachine(
			libvmifact.NewGuestless(),
			libvmi.WithInstancetype(instancetype.Name),
			libvmi.WithPreference(preference.Name),
		)
		vm, err = virtClient.VirtualMachine(testsuite.GetTestNamespace(instancetype)).Create(context.Background(), vm, metav1.CreateOptions{})
		Expect(err).ToNot(HaveOccurred())

		// Wait for the initial revisionNames to be populated before we start out tests
		libinstancetype.WaitForVMInstanceTypeRevisionNames(vm.Name, virtClient)
	})

	DescribeTable("should upgrade", func(generateControllerRevision func() (*appsv1.ControllerRevision, error), updateMatcher func(string), getVMRevisionName func() string) {
		// Capture the original RevisionName
		originalRevisionName := getVMRevisionName()

		By("Generating the target ControllerRevision")
		cr, err := generateControllerRevision()
		Expect(err).ToNot(HaveOccurred())

		By("Updating the VirtualMachine to reference the generated ControllerRevision")
		originalTestRevisionName := cr.Name
		updateMatcher(originalTestRevisionName)

		By("Waiting for the ControllerRevision referenced by the VirtualMachine to be upgraded to the latest version")
		var revisionName string
		Eventually(func(g Gomega) {
			By("Waiting for the RevisionName to be updated")
			revisionName = getVMRevisionName()
			g.Expect(revisionName).ToNot(Equal(originalRevisionName))

			cr, err = virtClient.AppsV1().ControllerRevisions(vm.Namespace).Get(context.Background(), revisionName, metav1.GetOptions{})
			g.Expect(err).ToNot(HaveOccurred())

			By("Ensuring the referenced ControllerRevision has the latest version label")
			g.Expect(instancetypepkg.IsObjectLatestVersion(cr)).To(BeTrue())

			By("Ensuring the referenced ControllerRevision contains an object of the latest version")
			decodedObj, err := runtime.Decode(generatedscheme.Codecs.UniversalDeserializer(), cr.Data.Raw)
			Expect(err).ToNot(HaveOccurred())
			Expect(decodedObj.GetObjectKind().GroupVersionKind().Version).To(Equal(instancetypeapi.LatestVersion))
		}, 30*time.Second, time.Second).Should(Succeed())

		// If a new CR has been created assert that the old CR has been deleted
		if originalTestRevisionName != revisionName {
			_, err := virtClient.AppsV1().ControllerRevisions(vm.Namespace).Get(context.Background(), originalTestRevisionName, metav1.GetOptions{})
			Expect(err).Should(HaveOccurred())
			Expect(errors.ReasonForError(err)).Should(Equal(metav1.StatusReasonNotFound))
		}
	},
		Entry("VirtualMachineInstancetype from v1beta1 without labels to latest",
			func() (*appsv1.ControllerRevision, error) {
				instancetype := builder.NewInstancetype(
					builder.WithCPUs(uint32(1)),
					builder.WithMemory(resource.MustParse("128Mi")),
				)
				instancetype, err := virtClient.VirtualMachineInstancetype(instancetype.Namespace).Create(context.Background(), instancetype, metav1.CreateOptions{})
				Expect(err).ToNot(HaveOccurred())
				return createLegacyControllerRevision(instancetype)
			},
			updateInstancetypeMatcher,
			getInstancetypeRevisionName,
		),
		Entry("VirtualMachineInstancetype from v1beta1 with labels to latest",
			func() (*appsv1.ControllerRevision, error) {
				instancetype := builder.NewInstancetype(
					builder.WithCPUs(uint32(1)),
					builder.WithMemory(resource.MustParse("128Mi")),
				)
				instancetype, err := virtClient.VirtualMachineInstancetype(instancetype.Namespace).Create(context.Background(), instancetype, metav1.CreateOptions{})
				Expect(err).ToNot(HaveOccurred())
				return createControllerRevision(instancetype)
			},
			updateInstancetypeMatcher,
			getInstancetypeRevisionName,
		),
		Entry("VirtualMachineInstancetype from v1alpha2 to latest",
			func() (*appsv1.ControllerRevision, error) {
				instancetype := builderv1alpha2.NewInstancetype(
					builderv1alpha2.WithCPUs(uint32(1)),
					builderv1alpha2.WithMemory(resource.MustParse("128Mi")),
				)
				instancetype, err := virtClient.GeneratedKubeVirtClient().InstancetypeV1alpha2().VirtualMachineInstancetypes(instancetype.Namespace).Create(context.Background(), instancetype, metav1.CreateOptions{})
				Expect(err).ToNot(HaveOccurred())
				return createLegacyControllerRevision(instancetype)
			},
			updateInstancetypeMatcher,
			getInstancetypeRevisionName,
		),
		Entry("VirtualMachineInstancetype from v1alpha1 to latest",
			func() (*appsv1.ControllerRevision, error) {
				instancetype := builderv1alpha1.NewInstancetype(
					builderv1alpha1.WithCPUs(uint32(1)),
					builderv1alpha1.WithMemory(resource.MustParse("128Mi")),
				)
				instancetype, err := virtClient.GeneratedKubeVirtClient().InstancetypeV1alpha1().VirtualMachineInstancetypes(instancetype.Namespace).Create(context.Background(), instancetype, metav1.CreateOptions{})
				Expect(err).ToNot(HaveOccurred())
				return createLegacyControllerRevision(instancetype)
			},
			updateInstancetypeMatcher,
			getInstancetypeRevisionName,
		),
		Entry("VirtualMachineInstancetypeSpecRevision v1alpha1 to latest",
			func() (*appsv1.ControllerRevision, error) {
				instancetype := builderv1alpha1.NewInstancetype(
					builderv1alpha1.WithCPUs(uint32(1)),
					builderv1alpha1.WithMemory(resource.MustParse("128Mi")),
				)
				specBytes, err := json.Marshal(&instancetype.Spec)
				Expect(err).ToNot(HaveOccurred())

				specRevision := instancetypev1alpha1.VirtualMachineInstancetypeSpecRevision{
					APIVersion: instancetypev1alpha1.SchemeGroupVersion.String(),
					Spec:       specBytes,
				}
				specRevisionBytes, err := json.Marshal(specRevision)
				Expect(err).ToNot(HaveOccurred())

				cr := &appsv1.ControllerRevision{
					ObjectMeta: metav1.ObjectMeta{
						GenerateName:    "specrevision-",
						OwnerReferences: []metav1.OwnerReference{*metav1.NewControllerRef(vm, virtv1.VirtualMachineGroupVersionKind)},
					},
					Data: runtime.RawExtension{
						Raw: specRevisionBytes,
					},
				}
				return virtClient.AppsV1().ControllerRevisions(vm.Namespace).Create(context.Background(), cr, metav1.CreateOptions{})
			},
			updateInstancetypeMatcher,
			getInstancetypeRevisionName,
		),
		Entry("VirtualMachineClusterInstancetype from v1beta1 without labels to latest",
			func() (*appsv1.ControllerRevision, error) {
				instancetype := builder.NewClusterInstancetype(
					builder.WithCPUs(uint32(1)),
					builder.WithMemory(resource.MustParse("128Mi")),
				)
				instancetype, err := virtClient.VirtualMachineClusterInstancetype().Create(context.Background(), instancetype, metav1.CreateOptions{})
				Expect(err).ToNot(HaveOccurred())
				return createLegacyControllerRevision(instancetype)
			},
			updateInstancetypeMatcher,
			getInstancetypeRevisionName,
		),
		Entry("VirtualMachineClusterInstancetype from v1beta1 with labels to latest",
			func() (*appsv1.ControllerRevision, error) {
				instancetype := builder.NewClusterInstancetype(
					builder.WithCPUs(uint32(1)),
					builder.WithMemory(resource.MustParse("128Mi")),
				)
				instancetype, err := virtClient.VirtualMachineClusterInstancetype().Create(context.Background(), instancetype, metav1.CreateOptions{})
				Expect(err).ToNot(HaveOccurred())
				return createControllerRevision(instancetype)
			},
			updateInstancetypeMatcher,
			getInstancetypeRevisionName,
		),
		Entry("VirtualMachineClusterInstancetype from v1alpha2 to latest",
			func() (*appsv1.ControllerRevision, error) {
				instancetype := builderv1alpha2.NewClusterInstancetype(
					builderv1alpha2.WithCPUs(uint32(1)),
					builderv1alpha2.WithMemory(resource.MustParse("128Mi")),
				)
				instancetype, err := virtClient.GeneratedKubeVirtClient().InstancetypeV1alpha2().VirtualMachineClusterInstancetypes().Create(context.Background(), instancetype, metav1.CreateOptions{})
				Expect(err).ToNot(HaveOccurred())
				return createLegacyControllerRevision(instancetype)
			},
			updateInstancetypeMatcher,
			getInstancetypeRevisionName,
		),
		Entry("VirtualMachineClusterInstancetype from v1alpha1 to latest",
			func() (*appsv1.ControllerRevision, error) {
				instancetype := builderv1alpha1.NewClusterInstancetype(
					builderv1alpha1.WithCPUs(uint32(1)),
					builderv1alpha1.WithMemory(resource.MustParse("128Mi")),
				)
				instancetype, err := virtClient.GeneratedKubeVirtClient().InstancetypeV1alpha1().VirtualMachineClusterInstancetypes().Create(context.Background(), instancetype, metav1.CreateOptions{})
				Expect(err).ToNot(HaveOccurred())
				return createLegacyControllerRevision(instancetype)
			},
			updateInstancetypeMatcher,
			getInstancetypeRevisionName,
		),
		Entry("VirtualMachinePreference from v1beta1 without labels to latest",
			func() (*appsv1.ControllerRevision, error) {
				preference := builder.NewPreference(
					builder.WithPreferredCPUTopology(instancetypev1beta1.Sockets),
				)
				preference, err := virtClient.VirtualMachinePreference(preference.Namespace).Create(context.Background(), preference, metav1.CreateOptions{})
				Expect(err).ToNot(HaveOccurred())
				return createLegacyControllerRevision(preference)
			},
			updatePreferenceMatcher,
			getPreferenceRevisionName,
		),
		Entry("VirtualMachinePreference from v1beta1 with labels to latest",
			func() (*appsv1.ControllerRevision, error) {
				preference := builder.NewPreference(
					builder.WithPreferredCPUTopology(instancetypev1beta1.Sockets),
				)
				preference, err := virtClient.VirtualMachinePreference(preference.Namespace).Create(context.Background(), preference, metav1.CreateOptions{})
				Expect(err).ToNot(HaveOccurred())
				return createControllerRevision(preference)
			},
			updatePreferenceMatcher,
			getPreferenceRevisionName,
		),
		Entry("VirtualMachinePreference from v1alpha2 to latest",
			func() (*appsv1.ControllerRevision, error) {
				preference := builderv1alpha2.NewPreference(
					builderv1alpha2.WithPreferredCPUTopology(instancetypev1alpha2.PreferSockets),
				)
				preference, err := virtClient.GeneratedKubeVirtClient().InstancetypeV1alpha2().VirtualMachinePreferences(preference.Namespace).Create(context.Background(), preference, metav1.CreateOptions{})
				Expect(err).ToNot(HaveOccurred())
				return createLegacyControllerRevision(preference)
			},
			updatePreferenceMatcher,
			getPreferenceRevisionName,
		),
		Entry("VirtualMachinePreference from v1alpha1 to latest",
			func() (*appsv1.ControllerRevision, error) {
				preference := builderv1alpha1.NewPreference(
					builderv1alpha1.WithPreferredCPUTopology(instancetypev1alpha1.PreferSockets),
				)
				preference, err := virtClient.GeneratedKubeVirtClient().InstancetypeV1alpha1().VirtualMachinePreferences(preference.Namespace).Create(context.Background(), preference, metav1.CreateOptions{})
				Expect(err).ToNot(HaveOccurred())
				return createLegacyControllerRevision(preference)
			},
			updatePreferenceMatcher,
			getPreferenceRevisionName,
		),
		Entry("VirtualMachinePreferenceSpecRevision v1alpha1 to latest",
			func() (*appsv1.ControllerRevision, error) {
				preference := builderv1alpha1.NewPreference(
					builderv1alpha1.WithPreferredCPUTopology(instancetypev1alpha1.PreferSockets),
				)
				specBytes, err := json.Marshal(&preference.Spec)
				Expect(err).ToNot(HaveOccurred())

				specRevision := instancetypev1alpha1.VirtualMachinePreferenceSpecRevision{
					APIVersion: instancetypev1alpha1.SchemeGroupVersion.String(),
					Spec:       specBytes,
				}
				specRevisionBytes, err := json.Marshal(specRevision)
				Expect(err).ToNot(HaveOccurred())

				cr := &appsv1.ControllerRevision{
					ObjectMeta: metav1.ObjectMeta{
						GenerateName:    "specrevision-",
						OwnerReferences: []metav1.OwnerReference{*metav1.NewControllerRef(vm, virtv1.VirtualMachineGroupVersionKind)},
					},
					Data: runtime.RawExtension{
						Raw: specRevisionBytes,
					},
				}
				return virtClient.AppsV1().ControllerRevisions(vm.Namespace).Create(context.Background(), cr, metav1.CreateOptions{})
			},
			updatePreferenceMatcher,
			getPreferenceRevisionName,
		),
		Entry("VirtualMachineClusterPreference from v1beta1 without labels to latest",
			func() (*appsv1.ControllerRevision, error) {
				preference := builder.NewClusterPreference(
					builder.WithPreferredCPUTopology(instancetypev1beta1.Sockets),
				)
				preference, err := virtClient.VirtualMachineClusterPreference().Create(context.Background(), preference, metav1.CreateOptions{})
				Expect(err).ToNot(HaveOccurred())
				return createLegacyControllerRevision(preference)
			},
			updatePreferenceMatcher,
			getPreferenceRevisionName,
		),
		Entry("VirtualMachineClusterPreference from v1beta1 with labels to latest",
			func() (*appsv1.ControllerRevision, error) {
				preference := builder.NewClusterPreference(
					builder.WithPreferredCPUTopology(instancetypev1beta1.Sockets),
				)
				preference, err := virtClient.VirtualMachineClusterPreference().Create(context.Background(), preference, metav1.CreateOptions{})
				Expect(err).ToNot(HaveOccurred())
				return createControllerRevision(preference)
			},
			updatePreferenceMatcher,
			getPreferenceRevisionName,
		),
		Entry("VirtualMachineClusterPreference from v1alpha2 to latest",
			func() (*appsv1.ControllerRevision, error) {
				preference := builderv1alpha2.NewClusterPreference(
					builderv1alpha2.WithPreferredCPUTopology(instancetypev1alpha2.PreferSockets),
				)
				preference, err := virtClient.GeneratedKubeVirtClient().InstancetypeV1alpha2().VirtualMachineClusterPreferences().Create(context.Background(), preference, metav1.CreateOptions{})
				Expect(err).ToNot(HaveOccurred())
				return createLegacyControllerRevision(preference)
			},
			updatePreferenceMatcher,
			getPreferenceRevisionName,
		),
		Entry("VirtualMachineClusterPreference from v1alpha1 to latest",
			func() (*appsv1.ControllerRevision, error) {
				preference := builderv1alpha1.NewClusterPreference(
					builderv1alpha1.WithPreferredCPUTopology(instancetypev1alpha1.PreferSockets),
				)
				preference, err := virtClient.GeneratedKubeVirtClient().InstancetypeV1alpha1().VirtualMachineClusterPreferences().Create(context.Background(), preference, metav1.CreateOptions{})
				Expect(err).ToNot(HaveOccurred())
				return createLegacyControllerRevision(preference)
			},
			updatePreferenceMatcher,
			getPreferenceRevisionName,
		),
	)
})
