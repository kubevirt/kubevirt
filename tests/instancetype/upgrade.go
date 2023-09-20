package instancetype

import (
	"context"
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

	instancetypepkg "kubevirt.io/kubevirt/pkg/instancetype"

	"kubevirt.io/kubevirt/tests"
	"kubevirt.io/kubevirt/tests/decorators"
	"kubevirt.io/kubevirt/tests/framework/kubevirt"
	libinstancetypev1alpha1 "kubevirt.io/kubevirt/tests/libinstancetype/v1alpha1"
	libinstancetypev1alpha2 "kubevirt.io/kubevirt/tests/libinstancetype/v1alpha2"
	libinstancetypev1beta1 "kubevirt.io/kubevirt/tests/libinstancetype/v1beta1"
	"kubevirt.io/kubevirt/tests/libvmi"
	"kubevirt.io/kubevirt/tests/testsuite"
	"kubevirt.io/kubevirt/tests/util"
)

var _ = Describe("[crit:medium][vendor:cnv-qe@redhat.com][level:component][sig-compute] Instance type and preference ControllerRevision Upgrades", decorators.SigCompute, func() {

	var (
		virtClient kubecli.KubevirtClient
	)

	BeforeEach(func() {
		virtClient = kubevirt.Client()
	})

	Context("using ControllerRevisionUpgrade", func() {
		var (
			vm *virtv1.VirtualMachine
		)

		createControllerRevision := func(obj runtime.Object) (*appsv1.ControllerRevision, error) {
			cr, err := instancetypepkg.CreateControllerRevision(vm, obj)
			Expect(err).ToNot(HaveOccurred())
			return virtClient.AppsV1().ControllerRevisions(vm.Namespace).Create(context.Background(), cr, metav1.CreateOptions{})
		}

		updateInstancetypeMatcher := func(revisionName string) {
			Eventually(func(g Gomega) {
				vm, err := virtClient.VirtualMachine(testsuite.GetTestNamespace(vm)).Get(context.Background(), vm.Name, &metav1.GetOptions{})
				g.Expect(err).ToNot(HaveOccurred())

				vm.Spec.Instancetype.RevisionName = revisionName

				_, err = virtClient.VirtualMachine(vm.Namespace).Update(context.Background(), vm)
				g.Expect(err).ToNot(HaveOccurred())
			}, 30*time.Second, time.Second).Should(Succeed())
		}

		updatePreferenceMatcher := func(revisionName string) {
			Eventually(func(g Gomega) {
				vm, err := virtClient.VirtualMachine(testsuite.GetTestNamespace(vm)).Get(context.Background(), vm.Name, &metav1.GetOptions{})
				g.Expect(err).ToNot(HaveOccurred())

				vm.Spec.Preference.RevisionName = revisionName

				_, err = virtClient.VirtualMachine(vm.Namespace).Update(context.Background(), vm)
				g.Expect(err).ToNot(HaveOccurred())
			}, 30*time.Second, time.Second).Should(Succeed())
		}

		getInstancetypeRevisionName := func() string {
			vm, err := virtClient.VirtualMachine(testsuite.GetTestNamespace(vm)).Get(context.Background(), vm.Name, &metav1.GetOptions{})
			Expect(err).ToNot(HaveOccurred())
			Expect(vm.Spec.Instancetype).ToNot(BeNil())
			return vm.Spec.Instancetype.RevisionName
		}

		getPreferenceRevisionName := func() string {
			vm, err := virtClient.VirtualMachine(testsuite.GetTestNamespace(vm)).Get(context.Background(), vm.Name, &metav1.GetOptions{})
			Expect(err).ToNot(HaveOccurred())
			Expect(vm.Spec.Preference).ToNot(BeNil())
			return vm.Spec.Preference.RevisionName
		}

		BeforeEach(func() {
			// We create a fake instance type here just to allow for the
			// creation of the initial VirtualMachine. This then allows the
			// creation of a ControllerRevision later on in the test to use the
			// now created VirtualMachine as an OwnerReference.
			instancetype := newVirtualMachineInstancetype(nil)
			instancetype, err := virtClient.VirtualMachineInstancetype(testsuite.GetTestNamespace(instancetype)).Create(context.Background(), instancetype, metav1.CreateOptions{})
			Expect(err).ToNot(HaveOccurred())

			preference := newVirtualMachinePreference()
			preference, err = virtClient.VirtualMachinePreference(testsuite.GetTestNamespace(preference)).Create(context.Background(), preference, metav1.CreateOptions{})
			Expect(err).ToNot(HaveOccurred())

			vmi := libvmi.NewCirros()
			removeResourcesAndPreferencesFromVMI(vmi)

			vm = tests.NewRandomVirtualMachine(vmi, false)
			vm.Spec.Instancetype = &virtv1.InstancetypeMatcher{
				Name: instancetype.Name,
				Kind: instancetypeapi.SingularResourceName,
			}
			vm.Spec.Preference = &virtv1.PreferenceMatcher{
				Name: preference.Name,
				Kind: instancetypeapi.SingularPreferenceResourceName,
			}

			vm, err = virtClient.VirtualMachine(testsuite.GetTestNamespace(instancetype)).Create(context.Background(), vm)
			Expect(err).ToNot(HaveOccurred())

		})

		DescribeTable("should upgrade", func(generateControllerRevision func() (*appsv1.ControllerRevision, error), updateMatcher func(string), getVMRevisionName func() string) {
			cr, err := generateControllerRevision()
			Expect(err).ToNot(HaveOccurred())

			By("Updating the VirtualMachine to reference the generated ControllerRevision")
			originalCRName := cr.Name
			updateMatcher(originalCRName)

			By("Creating a ControllerRevisionUpgrade request")
			crUpgrade := &instancetypev1beta1.ControllerRevisionUpgrade{
				ObjectMeta: metav1.ObjectMeta{
					GenerateName: "upgrade",
				},
				Spec: &instancetypev1beta1.ControllerRevisionUpgradeSpec{
					TargetName: cr.Name,
				},
			}
			crUpgrade, err = virtClient.ControllerRevisionUpgrade(vm.Namespace).Create(context.Background(), crUpgrade, metav1.CreateOptions{})
			Expect(err).ToNot(HaveOccurred())

			By("Waiting for the upgrade to be marked as successful")
			Eventually(func(g Gomega) {
				crUpgrade, err := virtClient.ControllerRevisionUpgrade(vm.Namespace).Get(context.Background(), crUpgrade.Name, metav1.GetOptions{})
				g.Expect(err).ToNot(HaveOccurred())
				g.Expect(crUpgrade.Status).ToNot(BeNil())
				g.Expect(crUpgrade.Status.Phase).ToNot(BeNil())
				g.Expect(*crUpgrade.Status.Phase).To(Equal(instancetypev1beta1.UpgradeSucceeded))
			}, 30*time.Second, time.Second).Should(Succeed())

			By("asserting that the ControllerRevision referenced by the VirtualMachine has been updated to the latest version")
			vmRevisionName := getVMRevisionName()

			cr, err = virtClient.AppsV1().ControllerRevisions(vm.Namespace).Get(context.Background(), vmRevisionName, metav1.GetOptions{})
			Expect(err).ToNot(HaveOccurred())

			// If a new CR has been created assert that the old CR has been deleted
			if originalCRName != vmRevisionName {
				_, err := virtClient.AppsV1().ControllerRevisions(vm.Namespace).Get(context.Background(), originalCRName, metav1.GetOptions{})
				Expect(err).Should(HaveOccurred())
				Expect(errors.ReasonForError(err)).Should(Equal(metav1.StatusReasonNotFound))
			}

			Expect(cr.Labels).To(HaveKeyWithValue(instancetypeapi.ControllerRevisionObjectVersionLabel, instancetypeapi.LatestVersion))

			decodedObj, err := runtime.Decode(generatedscheme.Codecs.UniversalDeserializer(), cr.Data.Raw)
			Expect(err).ToNot(HaveOccurred())
			Expect(decodedObj.GetObjectKind().GroupVersionKind().Version).To(Equal(instancetypeapi.LatestVersion))
		},
			Entry("VirtualMachineInstancetype from v1beta1 to latest",
				func() (*appsv1.ControllerRevision, error) {
					instancetype := libinstancetypev1beta1.NewInstancetype(
						libinstancetypev1beta1.WithInstancetypeSpec(
							libinstancetypev1beta1.NewInstancetypeSpec(
								libinstancetypev1beta1.WithCPUs(1),
								libinstancetypev1beta1.WithMemory(resource.MustParse("128Mi")),
							),
						),
					)
					instancetype, err := virtClient.VirtualMachineInstancetype(util.NamespaceTestDefault).Create(context.Background(), instancetype, metav1.CreateOptions{})
					Expect(err).ToNot(HaveOccurred())
					return createControllerRevision(instancetype)
				},
				updateInstancetypeMatcher,
				getInstancetypeRevisionName,
			),
			Entry("VirtualMachineInstancetype from v1alpha2 to latest",
				func() (*appsv1.ControllerRevision, error) {
					instancetype := libinstancetypev1alpha2.NewInstancetype(
						libinstancetypev1alpha2.WithInstancetypeSpec(
							libinstancetypev1alpha2.NewInstancetypeSpec(
								libinstancetypev1alpha2.WithCPUs(1),
								libinstancetypev1alpha2.WithMemory(resource.MustParse("128Mi")),
							),
						),
					)
					instancetype, err := virtClient.GeneratedKubeVirtClient().InstancetypeV1alpha2().VirtualMachineInstancetypes(util.NamespaceTestDefault).Create(context.Background(), instancetype, metav1.CreateOptions{})
					Expect(err).ToNot(HaveOccurred())
					return createControllerRevision(instancetype)
				},
				updateInstancetypeMatcher,
				getInstancetypeRevisionName,
			),
			Entry("VirtualMachineInstancetype from v1alpha1 to latest",
				func() (*appsv1.ControllerRevision, error) {
					instancetype := libinstancetypev1alpha1.NewInstancetype(
						libinstancetypev1alpha1.WithInstancetypeSpec(
							libinstancetypev1alpha1.NewInstancetypeSpec(
								libinstancetypev1alpha1.WithCPUs(1),
								libinstancetypev1alpha1.WithMemory(resource.MustParse("128Mi")),
							),
						),
					)
					instancetype, err := virtClient.GeneratedKubeVirtClient().InstancetypeV1alpha1().VirtualMachineInstancetypes(util.NamespaceTestDefault).Create(context.Background(), instancetype, metav1.CreateOptions{})
					Expect(err).ToNot(HaveOccurred())
					return createControllerRevision(instancetype)
				},
				updateInstancetypeMatcher,
				getInstancetypeRevisionName,
			),
			Entry("VirtualMachineClusterInstancetype from v1beta1 to latest",
				func() (*appsv1.ControllerRevision, error) {
					instancetype := libinstancetypev1beta1.NewClusterInstancetype(
						libinstancetypev1beta1.WithClusterInstancetypeSpec(
							libinstancetypev1beta1.NewInstancetypeSpec(
								libinstancetypev1beta1.WithCPUs(1),
								libinstancetypev1beta1.WithMemory(resource.MustParse("128Mi")),
							),
						),
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
					instancetype := libinstancetypev1alpha2.NewClusterInstancetype(
						libinstancetypev1alpha2.WithClusterInstancetypeSpec(
							libinstancetypev1alpha2.NewInstancetypeSpec(
								libinstancetypev1alpha2.WithCPUs(1),
								libinstancetypev1alpha2.WithMemory(resource.MustParse("128Mi")),
							),
						),
					)
					instancetype, err := virtClient.GeneratedKubeVirtClient().InstancetypeV1alpha2().VirtualMachineClusterInstancetypes().Create(context.Background(), instancetype, metav1.CreateOptions{})
					Expect(err).ToNot(HaveOccurred())
					return createControllerRevision(instancetype)
				},
				updateInstancetypeMatcher,
				getInstancetypeRevisionName,
			),
			Entry("VirtualMachineClusterInstancetype from v1alpha1 to latest",
				func() (*appsv1.ControllerRevision, error) {
					instancetype := libinstancetypev1alpha1.NewClusterInstancetype(
						libinstancetypev1alpha1.WithClusterInstancetypeSpec(
							libinstancetypev1alpha1.NewInstancetypeSpec(
								libinstancetypev1alpha1.WithCPUs(1),
								libinstancetypev1alpha1.WithMemory(resource.MustParse("128Mi")),
							),
						),
					)
					instancetype, err := virtClient.GeneratedKubeVirtClient().InstancetypeV1alpha1().VirtualMachineClusterInstancetypes().Create(context.Background(), instancetype, metav1.CreateOptions{})
					Expect(err).ToNot(HaveOccurred())
					return createControllerRevision(instancetype)
				},
				updateInstancetypeMatcher,
				getInstancetypeRevisionName,
			),
			Entry("VirtualMachinePreference from v1beta1 to latest",
				func() (*appsv1.ControllerRevision, error) {
					cpuPreference := instancetypev1beta1.PreferSockets
					preference := libinstancetypev1beta1.NewPreference(
						libinstancetypev1beta1.WithPreferenceSpec(
							instancetypev1beta1.VirtualMachinePreferenceSpec{
								CPU: &instancetypev1beta1.CPUPreferences{
									PreferredCPUTopology: &cpuPreference,
								},
							},
						),
					)
					preference, err := virtClient.VirtualMachinePreference(util.NamespaceTestDefault).Create(context.Background(), preference, metav1.CreateOptions{})
					Expect(err).ToNot(HaveOccurred())
					return createControllerRevision(preference)
				},
				updatePreferenceMatcher,
				getPreferenceRevisionName,
			),
			Entry("VirtualMachinePreference from v1alpha2 to latest",
				func() (*appsv1.ControllerRevision, error) {
					cpuPreference := instancetypev1alpha2.PreferSockets
					preference := libinstancetypev1alpha2.NewPreference(
						libinstancetypev1alpha2.WithPreferenceSpec(
							instancetypev1alpha2.VirtualMachinePreferenceSpec{
								CPU: &instancetypev1alpha2.CPUPreferences{
									PreferredCPUTopology: cpuPreference,
								},
							},
						),
					)
					preference, err := virtClient.GeneratedKubeVirtClient().InstancetypeV1alpha2().VirtualMachinePreferences(util.NamespaceTestDefault).Create(context.Background(), preference, metav1.CreateOptions{})
					Expect(err).ToNot(HaveOccurred())
					return createControllerRevision(preference)
				},
				updatePreferenceMatcher,
				getPreferenceRevisionName,
			),
			Entry("VirtualMachinePreference from v1alpha1 to latest",
				func() (*appsv1.ControllerRevision, error) {
					cpuPreference := instancetypev1alpha1.PreferSockets
					preference := libinstancetypev1alpha1.NewPreference(
						libinstancetypev1alpha1.WithPreferenceSpec(
							instancetypev1alpha1.VirtualMachinePreferenceSpec{
								CPU: &instancetypev1alpha1.CPUPreferences{
									PreferredCPUTopology: cpuPreference,
								},
							},
						),
					)
					preference, err := virtClient.GeneratedKubeVirtClient().InstancetypeV1alpha1().VirtualMachinePreferences(util.NamespaceTestDefault).Create(context.Background(), preference, metav1.CreateOptions{})
					Expect(err).ToNot(HaveOccurred())
					return createControllerRevision(preference)
				},
				updatePreferenceMatcher,
				getPreferenceRevisionName,
			),
			Entry("VirtualMachineClusterPreference from v1beta1 to latest",
				func() (*appsv1.ControllerRevision, error) {
					cpuPreference := instancetypev1beta1.PreferSockets
					preference := libinstancetypev1beta1.NewClusterPreference(
						libinstancetypev1beta1.WithClusterPreferenceSpec(
							instancetypev1beta1.VirtualMachinePreferenceSpec{
								CPU: &instancetypev1beta1.CPUPreferences{
									PreferredCPUTopology: &cpuPreference,
								},
							},
						),
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
					cpuPreference := instancetypev1alpha2.PreferSockets
					preference := libinstancetypev1alpha2.NewClusterPreference(
						libinstancetypev1alpha2.WithClusterPreferenceSpec(
							instancetypev1alpha2.VirtualMachinePreferenceSpec{
								CPU: &instancetypev1alpha2.CPUPreferences{
									PreferredCPUTopology: cpuPreference,
								},
							},
						),
					)
					preference, err := virtClient.GeneratedKubeVirtClient().InstancetypeV1alpha2().VirtualMachineClusterPreferences().Create(context.Background(), preference, metav1.CreateOptions{})
					Expect(err).ToNot(HaveOccurred())
					return createControllerRevision(preference)
				},
				updatePreferenceMatcher,
				getPreferenceRevisionName,
			),
			Entry("VirtualMachineClusterPreference from v1alpha1 to latest",
				func() (*appsv1.ControllerRevision, error) {
					cpuPreference := instancetypev1alpha1.PreferSockets
					preference := libinstancetypev1alpha1.NewClusterPreference(
						libinstancetypev1alpha1.WithClusterPreferenceSpec(
							instancetypev1alpha1.VirtualMachinePreferenceSpec{
								CPU: &instancetypev1alpha1.CPUPreferences{
									PreferredCPUTopology: cpuPreference,
								},
							},
						),
					)
					preference, err := virtClient.GeneratedKubeVirtClient().InstancetypeV1alpha1().VirtualMachineClusterPreferences().Create(context.Background(), preference, metav1.CreateOptions{})
					Expect(err).ToNot(HaveOccurred())
					return createControllerRevision(preference)
				},
				updatePreferenceMatcher,
				getPreferenceRevisionName,
			),
		)
	})
})
