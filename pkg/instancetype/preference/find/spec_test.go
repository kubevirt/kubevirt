//nolint:dupl
package find_test

import (
	"context"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	appsv1 "k8s.io/api/apps/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	k8sfake "k8s.io/client-go/kubernetes/fake"
	"k8s.io/client-go/testing"
	"k8s.io/client-go/tools/cache"

	"github.com/golang/mock/gomock"

	v1 "kubevirt.io/api/core/v1"
	apiinstancetype "kubevirt.io/api/instancetype"
	"kubevirt.io/api/instancetype/v1beta1"
	"kubevirt.io/client-go/kubecli"
	"kubevirt.io/client-go/kubevirt/fake"

	"kubevirt.io/kubevirt/pkg/instancetype/preference/find"
	"kubevirt.io/kubevirt/pkg/instancetype/revision"
	"kubevirt.io/kubevirt/pkg/libvmi"
	"kubevirt.io/kubevirt/pkg/pointer"
	"kubevirt.io/kubevirt/pkg/testutils"
)

var _ = Describe("Preference SpecFinder", func() {
	const (
		nonExistingResourceName = "non-existing-resource"
	)

	type preferenceSpecFinder interface {
		FindPreference(vm *v1.VirtualMachine) (*v1beta1.VirtualMachinePreferenceSpec, error)
	}

	var (
		finder preferenceSpecFinder
		vm     *v1.VirtualMachine

		virtClient                      *kubecli.MockKubevirtClient
		fakeClientset                   *fake.Clientset
		fakeK8sClientSet                *k8sfake.Clientset
		preferenceInformerStore         cache.Store
		clusterPreferenceInformerStore  cache.Store
		controllerRevisionInformerStore cache.Store
	)

	BeforeEach(func() {
		ctrl := gomock.NewController(GinkgoT())
		virtClient = kubecli.NewMockKubevirtClient(ctrl)

		fakeK8sClientSet = k8sfake.NewSimpleClientset()
		virtClient.EXPECT().AppsV1().Return(fakeK8sClientSet.AppsV1()).AnyTimes()

		fakeClientset = fake.NewSimpleClientset()

		virtClient.EXPECT().VirtualMachine(metav1.NamespaceDefault).Return(
			fakeClientset.KubevirtV1().VirtualMachines(metav1.NamespaceDefault)).AnyTimes()

		virtClient.EXPECT().VirtualMachineClusterPreference().Return(
			fakeClientset.InstancetypeV1beta1().VirtualMachineClusterPreferences()).AnyTimes()

		virtClient.EXPECT().VirtualMachinePreference(metav1.NamespaceDefault).Return(
			fakeClientset.InstancetypeV1beta1().VirtualMachinePreferences(metav1.NamespaceDefault)).AnyTimes()

		preferenceInformer, _ := testutils.NewFakeInformerFor(&v1beta1.VirtualMachinePreference{})
		preferenceInformerStore = preferenceInformer.GetStore()

		clusterPreferenceInformer, _ := testutils.NewFakeInformerFor(&v1beta1.VirtualMachineClusterPreference{})
		clusterPreferenceInformerStore = clusterPreferenceInformer.GetStore()

		controllerRevisionInformer, _ := testutils.NewFakeInformerFor(&appsv1.ControllerRevision{})
		controllerRevisionInformerStore = controllerRevisionInformer.GetStore()

		finder = find.NewSpecFinder(preferenceInformerStore, clusterPreferenceInformerStore, controllerRevisionInformerStore, virtClient)
	})

	It("find returns nil when no preference is specified", func() {
		vm = libvmi.NewVirtualMachine(libvmi.New())
		preference, err := finder.FindPreference(vm)
		Expect(err).ToNot(HaveOccurred())
		Expect(preference).To(BeNil())
	})

	It("find returns error when invalid Preference Kind is specified", func() {
		vm = libvmi.NewVirtualMachine(libvmi.New())
		vm.Spec.Preference = &v1.PreferenceMatcher{
			Name: "foo",
			Kind: "bar",
		}
		spec, err := finder.FindPreference(vm)
		Expect(err).To(MatchError(ContainSubstring("got unexpected kind in PreferenceMatcher")))
		Expect(spec).To(BeNil())
	})

	Context("Using global ClusterPreference", func() {
		var clusterPreference *v1beta1.VirtualMachineClusterPreference

		BeforeEach(func() {
			preferredCPUTopology := v1beta1.Cores
			clusterPreference = &v1beta1.VirtualMachineClusterPreference{
				ObjectMeta: metav1.ObjectMeta{
					Name: "test-cluster-preference",
				},
				Spec: v1beta1.VirtualMachinePreferenceSpec{
					CPU: &v1beta1.CPUPreferences{
						PreferredCPUTopology: &preferredCPUTopology,
					},
				},
			}

			_, err := virtClient.VirtualMachineClusterPreference().Create(context.Background(), clusterPreference, metav1.CreateOptions{})
			Expect(err).ToNot(HaveOccurred())

			err = clusterPreferenceInformerStore.Add(clusterPreference)
			Expect(err).ToNot(HaveOccurred())

			vm = libvmi.NewVirtualMachine(libvmi.New(), libvmi.WithClusterPreference(clusterPreference.Name))
		})

		It("find returns expected preference spec", func() {
			preferenceSpec, err := finder.FindPreference(vm)
			Expect(err).ToNot(HaveOccurred())
			Expect(preferenceSpec).To(HaveValue(Equal(clusterPreference.Spec)))
		})

		DescribeTable("returns expected preference referenced by", func(updateVM func(*v1.VirtualMachine, string)) {
			cr, err := revision.CreateControllerRevision(vm, clusterPreference)
			Expect(err).ToNot(HaveOccurred())

			_, err = virtClient.AppsV1().ControllerRevisions(vm.Namespace).Create(context.Background(), cr, metav1.CreateOptions{})
			Expect(err).ToNot(HaveOccurred())

			updateVM(vm, cr.Name)

			preferenceSpec, err := finder.FindPreference(vm)
			Expect(err).ToNot(HaveOccurred())
			Expect(preferenceSpec).To(HaveValue(Equal(clusterPreference.Spec)))
			Expect(fakeK8sClientSet.Actions()).To(
				ContainElement(
					testing.NewGetAction(
						appsv1.SchemeGroupVersion.WithResource("controllerrevisions"),
						vm.Namespace,
						cr.Name,
					),
				),
			)
			Expect(fakeClientset.Actions()).ToNot(
				ContainElement(
					testing.NewGetAction(
						v1beta1.SchemeGroupVersion.WithResource(apiinstancetype.SingularPreferenceResourceName),
						vm.Namespace,
						vm.Spec.Preference.Name,
					),
				),
			)
		},
			Entry("ControllerRevisionRef",
				func(vm *v1.VirtualMachine, crName string) {
					vm.Status.PreferenceRef = &v1.InstancetypeStatusRef{
						ControllerRevisionRef: &v1.ControllerRevisionRef{
							Name: crName,
						},
					}
				},
			),
			Entry("RevisionName",
				func(vm *v1.VirtualMachine, crName string) {
					vm.Spec.Preference = &v1.PreferenceMatcher{
						RevisionName: crName,
					}
				},
			),
			Entry("RevisionName over ControllerRevisionRef",
				func(vm *v1.VirtualMachine, crName string) {
					vm.Status.PreferenceRef = &v1.InstancetypeStatusRef{
						ControllerRevisionRef: &v1.ControllerRevisionRef{
							Name: "foobar",
						},
					}
					vm.Spec.Preference = &v1.PreferenceMatcher{
						RevisionName: crName,
					}
				},
			),
		)

		It("find returns expected preference spec with no kind provided", func() {
			vm.Spec.Preference.Kind = ""
			preferenceSpec, err := finder.FindPreference(vm)
			Expect(err).ToNot(HaveOccurred())
			Expect(preferenceSpec).To(HaveValue(Equal(clusterPreference.Spec)))
		})

		It("uses client when preference not found within informer", func() {
			err := clusterPreferenceInformerStore.Delete(clusterPreference)
			Expect(err).ToNot(HaveOccurred())
			preferenceSpec, err := finder.FindPreference(vm)
			Expect(err).ToNot(HaveOccurred())
			Expect(preferenceSpec).To(HaveValue(Equal(clusterPreference.Spec)))
			Expect(fakeClientset.Actions()).To(
				ContainElement(
					testing.NewGetAction(
						v1beta1.SchemeGroupVersion.WithResource(apiinstancetype.ClusterPluralPreferenceResourceName),
						"",
						vm.Spec.Preference.Name,
					),
				),
			)
		})

		It("returns expected preference using only the client", func() {
			finder = find.NewSpecFinder(nil, nil, nil, virtClient)
			preferenceSpec, err := finder.FindPreference(vm)
			Expect(err).ToNot(HaveOccurred())
			Expect(preferenceSpec).To(HaveValue(Equal(clusterPreference.Spec)))
			Expect(fakeClientset.Actions()).To(
				ContainElement(
					testing.NewGetAction(
						v1beta1.SchemeGroupVersion.WithResource(apiinstancetype.ClusterPluralPreferenceResourceName),
						"",
						vm.Spec.Preference.Name,
					),
				),
			)
		})

		It("find fails when preference does not exist", func() {
			vm = libvmi.NewVirtualMachine(libvmi.New(), libvmi.WithClusterPreference(nonExistingResourceName))
			_, err := finder.FindPreference(vm)
			Expect(err).To(MatchError(errors.IsNotFound, "IsNotFound"))
		})

		It("find returns only referenced object - bug #14595", func() {
			// Make a slightly altered copy of the object already present in the client and store it in a CR
			stored := clusterPreference.DeepCopy()
			stored.ObjectMeta.Name = "stored"
			stored.Spec.CPU.PreferredCPUTopology = pointer.P(v1beta1.Threads)

			controllerRevision, err := revision.CreateControllerRevision(vm, stored)
			Expect(err).ToNot(HaveOccurred())

			_, err = virtClient.AppsV1().ControllerRevisions(vm.Namespace).Create(
				context.Background(), controllerRevision, metav1.CreateOptions{})
			Expect(err).ToNot(HaveOccurred())

			// Assert that the spec points to the original clusterInstancetype
			Expect(vm.Spec.Preference.Name).To(Equal(clusterPreference.Name))

			// Reference this stored version from the VM status
			vm.Status.PreferenceRef = &v1.InstancetypeStatusRef{
				Name: stored.Name,
				Kind: stored.Kind,
				ControllerRevisionRef: &v1.ControllerRevisionRef{
					Name: controllerRevision.Name,
				},
			}

			foundPreferenceSpec, err := finder.FindPreference(vm)
			Expect(err).ToNot(HaveOccurred())
			// FIXME(lyarwood): This is bug #14595, should return clusterPreference.Spec
			Expect(foundPreferenceSpec).To(HaveValue(Equal(stored.Spec)))
		})
	})

	Context("Using namespaced Preference", func() {
		var preference *v1beta1.VirtualMachinePreference

		BeforeEach(func() {
			preferredCPUTopology := v1beta1.Cores
			preference = &v1beta1.VirtualMachinePreference{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-preference",
					Namespace: metav1.NamespaceDefault,
				},
				Spec: v1beta1.VirtualMachinePreferenceSpec{
					CPU: &v1beta1.CPUPreferences{
						PreferredCPUTopology: &preferredCPUTopology,
					},
				},
			}

			_, err := virtClient.VirtualMachinePreference(metav1.NamespaceDefault).Create(
				context.Background(), preference, metav1.CreateOptions{})
			Expect(err).ToNot(HaveOccurred())

			err = preferenceInformerStore.Add(preference)
			Expect(err).ToNot(HaveOccurred())

			vm = libvmi.NewVirtualMachine(libvmi.New(libvmi.WithNamespace(metav1.NamespaceDefault)), libvmi.WithPreference(preference.Name))
		})

		It("find returns expected preference spec", func() {
			preferenceSpec, err := finder.FindPreference(vm)
			Expect(err).ToNot(HaveOccurred())
			Expect(preferenceSpec).To(HaveValue(Equal(preference.Spec)))
		})

		DescribeTable("returns expected preference referenced by", func(updateVM func(*v1.VirtualMachine, string)) {
			cr, err := revision.CreateControllerRevision(vm, preference)
			Expect(err).ToNot(HaveOccurred())

			_, err = virtClient.AppsV1().ControllerRevisions(vm.Namespace).Create(context.Background(), cr, metav1.CreateOptions{})
			Expect(err).ToNot(HaveOccurred())

			updateVM(vm, cr.Name)

			preferenceSpec, err := finder.FindPreference(vm)
			Expect(err).ToNot(HaveOccurred())
			Expect(preferenceSpec).To(HaveValue(Equal(preference.Spec)))
			Expect(fakeK8sClientSet.Actions()).To(
				ContainElement(
					testing.NewGetAction(
						appsv1.SchemeGroupVersion.WithResource("controllerrevisions"),
						vm.Namespace,
						cr.Name,
					),
				),
			)
			Expect(fakeClientset.Actions()).ToNot(
				ContainElement(
					testing.NewGetAction(
						v1beta1.SchemeGroupVersion.WithResource(apiinstancetype.SingularPreferenceResourceName),
						vm.Namespace,
						vm.Spec.Preference.Name,
					),
				),
			)
		},
			Entry("ControllerRevisionRef",
				func(vm *v1.VirtualMachine, crName string) {
					vm.Status.PreferenceRef = &v1.InstancetypeStatusRef{
						ControllerRevisionRef: &v1.ControllerRevisionRef{
							Name: crName,
						},
					}
				},
			),
			Entry("RevisionName",
				func(vm *v1.VirtualMachine, crName string) {
					vm.Spec.Preference = &v1.PreferenceMatcher{
						RevisionName: crName,
					}
				},
			),
			Entry("RevisionName over ControllerRevisionRef",
				func(vm *v1.VirtualMachine, crName string) {
					vm.Status.PreferenceRef = &v1.InstancetypeStatusRef{
						ControllerRevisionRef: &v1.ControllerRevisionRef{
							Name: "foobar",
						},
					}
					vm.Spec.Preference = &v1.PreferenceMatcher{
						RevisionName: crName,
					}
				},
			),
		)

		It("uses client when preference not found within informer", func() {
			err := preferenceInformerStore.Delete(preference)
			Expect(err).ToNot(HaveOccurred())

			preferenceSpec, err := finder.FindPreference(vm)
			Expect(err).ToNot(HaveOccurred())
			Expect(preferenceSpec).To(HaveValue(Equal(preference.Spec)))
			Expect(fakeClientset.Actions()).To(
				ContainElement(
					testing.NewGetAction(
						v1beta1.SchemeGroupVersion.WithResource(apiinstancetype.PluralPreferenceResourceName),
						vm.Namespace,
						vm.Spec.Preference.Name,
					),
				),
			)
		})

		It("returns expected preference using only the client", func() {
			finder = find.NewSpecFinder(nil, nil, nil, virtClient)
			preferenceSpec, err := finder.FindPreference(vm)
			Expect(err).ToNot(HaveOccurred())
			Expect(preferenceSpec).To(HaveValue(Equal(preference.Spec)))
			Expect(fakeClientset.Actions()).To(
				ContainElement(
					testing.NewGetAction(
						v1beta1.SchemeGroupVersion.WithResource(apiinstancetype.PluralPreferenceResourceName),
						vm.Namespace,
						vm.Spec.Preference.Name,
					),
				),
			)
		})

		It("find fails when preference does not exist", func() {
			vm = libvmi.NewVirtualMachine(
				libvmi.New(libvmi.WithNamespace(metav1.NamespaceDefault)),
				libvmi.WithPreference(nonExistingResourceName),
			)
			_, err := finder.FindPreference(vm)
			Expect(err).To(MatchError(errors.IsNotFound, "IsNotFound"))
		})

		It("find returns only referenced object - bug #14595", func() {
			// Make a slightly altered copy of the object already present in the client and store it in a CR
			stored := preference.DeepCopy()
			stored.ObjectMeta.Name = "stored"
			stored.Spec.CPU.PreferredCPUTopology = pointer.P(v1beta1.Threads)

			controllerRevision, err := revision.CreateControllerRevision(vm, stored)
			Expect(err).ToNot(HaveOccurred())

			_, err = virtClient.AppsV1().ControllerRevisions(vm.Namespace).Create(
				context.Background(), controllerRevision, metav1.CreateOptions{})
			Expect(err).ToNot(HaveOccurred())

			// Assert that the spec points to the original clusterInstancetype
			Expect(vm.Spec.Preference.Name).To(Equal(preference.Name))

			// Reference this stored version from the VM status
			vm.Status.PreferenceRef = &v1.InstancetypeStatusRef{
				Name: stored.Name,
				Kind: stored.Kind,
				ControllerRevisionRef: &v1.ControllerRevisionRef{
					Name: controllerRevision.Name,
				},
			}

			foundPreferenceSpec, err := finder.FindPreference(vm)
			Expect(err).ToNot(HaveOccurred())
			// FIXME(lyarwood): This is bug #14595, should return preference.Spec
			Expect(foundPreferenceSpec).To(HaveValue(Equal(stored.Spec)))
		})
	})
})
