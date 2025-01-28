package find_test

import (
	"context"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	appsv1 "k8s.io/api/apps/v1"
	k8sv1 "k8s.io/api/core/v1"
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
	"kubevirt.io/kubevirt/pkg/testutils"
)

const (
	nonExistingResourceName = "non-existing-resource"
)

type preferenceSpecFinder interface {
	FindPreference(vm *v1.VirtualMachine) (*v1beta1.VirtualMachinePreferenceSpec, error)
}

var _ = Describe("Preference SpecFinder", func() {
	var (
		finder preferenceSpecFinder
		vm     *v1.VirtualMachine

		virtClient                      *kubecli.MockKubevirtClient
		fakeClientset                   *fake.Clientset
		preferenceInformerStore         cache.Store
		clusterPreferenceInformerStore  cache.Store
		controllerRevisionInformerStore cache.Store
	)

	BeforeEach(func() {
		ctrl := gomock.NewController(GinkgoT())
		virtClient = kubecli.NewMockKubevirtClient(ctrl)

		virtClient.EXPECT().AppsV1().Return(k8sfake.NewSimpleClientset().AppsV1()).AnyTimes()

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

		vm = kubecli.NewMinimalVM("testvm")
		vm.Spec.Template = &v1.VirtualMachineInstanceTemplateSpec{
			Spec: v1.VirtualMachineInstanceSpec{
				Domain: v1.DomainSpec{},
			},
		}
		vm.Namespace = k8sv1.NamespaceDefault

		_, err := virtClient.VirtualMachine(vm.Namespace).Create(context.Background(), vm, metav1.CreateOptions{})
		Expect(err).ToNot(HaveOccurred())
	})

	It("find returns nil when no preference is specified", func() {
		vm.Spec.Preference = nil
		preference, err := finder.FindPreference(vm)
		Expect(err).ToNot(HaveOccurred())
		Expect(preference).To(BeNil())
	})

	It("find returns error when invalid Preference Kind is specified", func() {
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

			vm.Spec.Preference = &v1.PreferenceMatcher{
				Name: clusterPreference.Name,
				Kind: apiinstancetype.ClusterSingularPreferenceResourceName,
			}
		})

		It("find returns expected preference spec", func() {
			s, err := finder.FindPreference(vm)
			Expect(err).ToNot(HaveOccurred())
			Expect(s).To(HaveValue(Equal(clusterPreference.Spec)))
		})

		It("find returns expected preference spec with no kind provided", func() {
			vm.Spec.Preference.Kind = ""
			s, err := finder.FindPreference(vm)
			Expect(err).ToNot(HaveOccurred())
			Expect(s).To(HaveValue(Equal(clusterPreference.Spec)))
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
			vm.Spec.Preference.Name = nonExistingResourceName
			_, err := finder.FindPreference(vm)
			Expect(err).To(MatchError(errors.IsNotFound, "IsNotFound"))
		})
	})

	Context("Using namespaced Preference", func() {
		var preference *v1beta1.VirtualMachinePreference

		BeforeEach(func() {
			preferredCPUTopology := v1beta1.Cores
			preference = &v1beta1.VirtualMachinePreference{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-preference",
					Namespace: vm.Namespace,
				},
				Spec: v1beta1.VirtualMachinePreferenceSpec{
					CPU: &v1beta1.CPUPreferences{
						PreferredCPUTopology: &preferredCPUTopology,
					},
				},
			}

			_, err := virtClient.VirtualMachinePreference(vm.Namespace).Create(context.Background(), preference, metav1.CreateOptions{})
			Expect(err).ToNot(HaveOccurred())

			err = preferenceInformerStore.Add(preference)
			Expect(err).ToNot(HaveOccurred())

			vm.Spec.Preference = &v1.PreferenceMatcher{
				Name: preference.Name,
				Kind: apiinstancetype.SingularPreferenceResourceName,
			}
		})

		It("find returns expected preference spec", func() {
			preferenceSpec, err := finder.FindPreference(vm)
			Expect(err).ToNot(HaveOccurred())
			Expect(preferenceSpec).To(HaveValue(Equal(preference.Spec)))
		})

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
			vm.Spec.Preference.Name = nonExistingResourceName
			_, err := finder.FindPreference(vm)
			Expect(err).To(MatchError(errors.IsNotFound, "IsNotFound"))
		})
	})
})
