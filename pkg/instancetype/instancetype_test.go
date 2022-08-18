package instancetype_test

import (
	"context"

	"github.com/golang/mock/gomock"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	appsv1 "k8s.io/api/apps/v1"
	k8sv1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/validation/field"
	"k8s.io/client-go/testing"
	"k8s.io/utils/pointer"

	k8sfield "k8s.io/apimachinery/pkg/util/validation/field"
	k8sfake "k8s.io/client-go/kubernetes/fake"

	"kubevirt.io/client-go/api"
	"kubevirt.io/client-go/kubecli"

	"kubevirt.io/kubevirt/pkg/instancetype"

	v1 "kubevirt.io/api/core/v1"
	apiinstancetype "kubevirt.io/api/instancetype"
	instancetypev1alpha2 "kubevirt.io/api/instancetype/v1alpha2"
	fakeclientset "kubevirt.io/client-go/generated/kubevirt/clientset/versioned/fake"
	"kubevirt.io/client-go/generated/kubevirt/clientset/versioned/typed/instancetype/v1alpha2"
)

const resourceUID types.UID = "9160e5de-2540-476a-86d9-af0081aee68a"
const resourceGeneration int64 = 1

var _ = Describe("Instancetype and Preferences", func() {
	var (
		ctrl                    *gomock.Controller
		instancetypeMethods     instancetype.Methods
		vm                      *v1.VirtualMachine
		vmi                     *v1.VirtualMachineInstance
		virtClient              *kubecli.MockKubevirtClient
		vmInterface             *kubecli.MockVirtualMachineInterface
		fakeInstancetypeClients v1alpha2.InstancetypeV1alpha2Interface
		k8sClient               *k8sfake.Clientset
	)

	expectControllerRevisionCreation := func(instancetypeSpecRevision *appsv1.ControllerRevision) {
		k8sClient.Fake.PrependReactor("create", "controllerrevisions", func(action testing.Action) (handled bool, obj runtime.Object, err error) {
			created, ok := action.(testing.CreateAction)
			Expect(ok).To(BeTrue())

			createObj := created.GetObject().(*appsv1.ControllerRevision)
			Expect(createObj).To(Equal(instancetypeSpecRevision))

			return true, created.GetObject(), nil
		})
	}

	BeforeEach(func() {

		k8sClient = k8sfake.NewSimpleClientset()
		ctrl = gomock.NewController(GinkgoT())
		virtClient = kubecli.NewMockKubevirtClient(ctrl)
		vmInterface = kubecli.NewMockVirtualMachineInterface(ctrl)
		virtClient.EXPECT().VirtualMachine(metav1.NamespaceDefault).Return(vmInterface).AnyTimes()
		virtClient.EXPECT().AppsV1().Return(k8sClient.AppsV1()).AnyTimes()
		fakeInstancetypeClients = fakeclientset.NewSimpleClientset().InstancetypeV1alpha2()

		instancetypeMethods = instancetype.NewMethods(virtClient)

		vm = kubecli.NewMinimalVM("testvm")
		vm.Spec.Template = &v1.VirtualMachineInstanceTemplateSpec{
			Spec: v1.VirtualMachineInstanceSpec{
				Domain: v1.DomainSpec{},
			},
		}
		vm.Namespace = k8sv1.NamespaceDefault

	})

	Context("Find and store Instancetype Spec", func() {

		It("find returns nil when no instancetype is specified", func() {
			vm.Spec.Instancetype = nil
			spec, err := instancetypeMethods.FindInstancetypeSpec(vm)
			Expect(err).ToNot(HaveOccurred())
			Expect(spec).To(BeNil())
		})

		It("find returns error when invalid Instancetype Kind is specified", func() {
			vm.Spec.Instancetype = &v1.InstancetypeMatcher{
				Name: "foo",
				Kind: "bar",
			}
			spec, err := instancetypeMethods.FindInstancetypeSpec(vm)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("got unexpected kind in InstancetypeMatcher"))
			Expect(spec).To(BeNil())
		})

		It("store returns error when instancetypeMatcher kind is invalid", func() {
			vm.Spec.Instancetype = &v1.InstancetypeMatcher{
				Kind: "foobar",
			}
			Expect(instancetypeMethods.StoreControllerRevisions(vm)).To(MatchError(ContainSubstring("got unexpected kind in InstancetypeMatcher")))
		})

		It("store returns nil when no instancetypeMatcher is specified", func() {
			vm.Spec.Instancetype = nil
			Expect(instancetypeMethods.StoreControllerRevisions(vm)).To(Succeed())
		})

		Context("Using global ClusterInstancetype", func() {
			var clusterInstancetype *instancetypev1alpha2.VirtualMachineClusterInstancetype
			var fakeClusterInstancetypeClient v1alpha2.VirtualMachineClusterInstancetypeInterface

			BeforeEach(func() {

				fakeClusterInstancetypeClient = fakeInstancetypeClients.VirtualMachineClusterInstancetypes()
				virtClient.EXPECT().VirtualMachineClusterInstancetype().Return(fakeClusterInstancetypeClient).AnyTimes()

				clusterInstancetype = &instancetypev1alpha2.VirtualMachineClusterInstancetype{
					TypeMeta: metav1.TypeMeta{
						Kind:       "VirtualMachineClusterInstancetype",
						APIVersion: instancetypev1alpha2.SchemeGroupVersion.String(),
					},
					ObjectMeta: metav1.ObjectMeta{
						Name:       "test-cluster-instancetype",
						UID:        resourceUID,
						Generation: resourceGeneration,
					},
					Spec: instancetypev1alpha2.VirtualMachineInstancetypeSpec{
						CPU: instancetypev1alpha2.CPUInstancetype{
							Guest: uint32(2),
						},
					},
				}

				_, err := virtClient.VirtualMachineClusterInstancetype().Create(context.Background(), clusterInstancetype, metav1.CreateOptions{})
				Expect(err).ToNot(HaveOccurred())

				vm.Spec.Instancetype = &v1.InstancetypeMatcher{
					Name: clusterInstancetype.Name,
					Kind: apiinstancetype.ClusterSingularResourceName,
				}
			})

			It("returns expected instancetype", func() {

				f, err := instancetypeMethods.FindInstancetypeSpec(vm)
				Expect(err).ToNot(HaveOccurred())
				Expect(*f).To(Equal(clusterInstancetype.Spec))
			})

			It("find fails when instancetype does not exist", func() {
				vm.Spec.Instancetype.Name = "non-existing-instancetype"
				_, err := instancetypeMethods.FindInstancetypeSpec(vm)
				Expect(err).To(HaveOccurred())
				Expect(errors.IsNotFound(err)).To(BeTrue())
			})

			It("store VirtualMachineClusterInstancetype ControllerRevision", func() {

				clusterInstancetypeControllerRevision, err := instancetype.CreateControllerRevision(vm, clusterInstancetype)
				Expect(err).ToNot(HaveOccurred())

				expectedRevisionNamePatch, err := instancetype.GenerateRevisionNamePatch(clusterInstancetypeControllerRevision, nil)
				Expect(err).ToNot(HaveOccurred())

				vmInterface.EXPECT().Patch(vm.Name, types.JSONPatchType, expectedRevisionNamePatch, &metav1.PatchOptions{})

				expectControllerRevisionCreation(clusterInstancetypeControllerRevision)

				Expect(instancetypeMethods.StoreControllerRevisions(vm)).To(Succeed())
				Expect(vm.Spec.Instancetype.RevisionName).To(Equal(clusterInstancetypeControllerRevision.Name))

			})

			It("store returns a nil revision when RevisionName already populated", func() {
				clusterInstancetypeControllerRevision, err := instancetype.CreateControllerRevision(vm, clusterInstancetype)
				Expect(err).ToNot(HaveOccurred())

				_, err = virtClient.AppsV1().ControllerRevisions(vm.Namespace).Create(context.Background(), clusterInstancetypeControllerRevision, metav1.CreateOptions{})
				Expect(err).ToNot(HaveOccurred())

				vm.Spec.Instancetype = &v1.InstancetypeMatcher{
					Name:         clusterInstancetype.Name,
					RevisionName: clusterInstancetypeControllerRevision.Name,
					Kind:         apiinstancetype.ClusterSingularResourceName,
				}

				Expect(instancetypeMethods.StoreControllerRevisions(vm)).To(Succeed())
				Expect(vm.Spec.Instancetype.RevisionName).To(Equal(clusterInstancetypeControllerRevision.Name))
			})

			It("store fails when instancetype does not exist", func() {
				vm.Spec.Instancetype.Name = "non-existing-instancetype"

				err := instancetypeMethods.StoreControllerRevisions(vm)
				Expect(err).To(HaveOccurred())
				Expect(errors.IsNotFound(err)).To(BeTrue())
			})

			It("store ControllerRevision succeeds if a revision exists with expected data", func() {

				instancetypeControllerRevision, err := instancetype.CreateControllerRevision(vm, clusterInstancetype)
				Expect(err).ToNot(HaveOccurred())

				_, err = virtClient.AppsV1().ControllerRevisions(vm.Namespace).Create(context.Background(), instancetypeControllerRevision, metav1.CreateOptions{})
				Expect(err).ToNot(HaveOccurred())

				expectedRevisionNamePatch, err := instancetype.GenerateRevisionNamePatch(instancetypeControllerRevision, nil)
				Expect(err).ToNot(HaveOccurred())

				vmInterface.EXPECT().Patch(vm.Name, types.JSONPatchType, expectedRevisionNamePatch, &metav1.PatchOptions{})

				Expect(instancetypeMethods.StoreControllerRevisions(vm)).To(Succeed())
				Expect(vm.Spec.Instancetype.RevisionName).To(Equal(instancetypeControllerRevision.Name))

			})

			It("store ControllerRevision fails if a revision exists with unexpected data", func() {
				unexpectedInstancetype := clusterInstancetype.DeepCopy()
				unexpectedInstancetype.Spec.CPU.Guest = 15

				instancetypeControllerRevision, err := instancetype.CreateControllerRevision(vm, unexpectedInstancetype)
				Expect(err).ToNot(HaveOccurred())

				_, err = virtClient.AppsV1().ControllerRevisions(vm.Namespace).Create(context.Background(), instancetypeControllerRevision, metav1.CreateOptions{})
				Expect(err).ToNot(HaveOccurred())

				Expect(instancetypeMethods.StoreControllerRevisions(vm)).To(MatchError(ContainSubstring("found existing ControllerRevision with unexpected data")))
			})

			It("store ControllerRevision fails if instancetype conflicts with vm", func() {
				vm.Spec.Template.Spec.Domain.CPU = &v1.CPU{
					Cores: 1,
				}

				Expect(instancetypeMethods.StoreControllerRevisions(vm)).To(MatchError(ContainSubstring("VM field conflicts with selected Instancetype")))
			})

		})

		Context("Using namespaced Instancetype", func() {
			var f *instancetypev1alpha2.VirtualMachineInstancetype
			var fakeInstancetypeClient v1alpha2.VirtualMachineInstancetypeInterface

			BeforeEach(func() {

				fakeInstancetypeClient = fakeInstancetypeClients.VirtualMachineInstancetypes(vm.Namespace)
				virtClient.EXPECT().VirtualMachineInstancetype(gomock.Any()).Return(fakeInstancetypeClient).AnyTimes()

				f = &instancetypev1alpha2.VirtualMachineInstancetype{
					TypeMeta: metav1.TypeMeta{
						Kind:       "VirtualMachineInstancetype",
						APIVersion: instancetypev1alpha2.SchemeGroupVersion.String(),
					},
					ObjectMeta: metav1.ObjectMeta{
						Name:       "test-instancetype",
						Namespace:  vm.Namespace,
						UID:        resourceUID,
						Generation: resourceGeneration,
					},
					Spec: instancetypev1alpha2.VirtualMachineInstancetypeSpec{
						CPU: instancetypev1alpha2.CPUInstancetype{
							Guest: uint32(2),
						},
					},
				}

				_, err := virtClient.VirtualMachineInstancetype(vm.Namespace).Create(context.Background(), f, metav1.CreateOptions{})
				Expect(err).ToNot(HaveOccurred())

				vm.Spec.Instancetype = &v1.InstancetypeMatcher{
					Name: f.Name,
					Kind: apiinstancetype.SingularResourceName,
				}
			})

			It("find returns expected instancetype", func() {
				instancetypeSpec, err := instancetypeMethods.FindInstancetypeSpec(vm)
				Expect(err).ToNot(HaveOccurred())
				Expect(*instancetypeSpec).To(Equal(f.Spec))
			})

			It("find fails when instancetype does not exist", func() {
				vm.Spec.Instancetype.Name = "non-existing-instancetype"
				_, err := instancetypeMethods.FindInstancetypeSpec(vm)
				Expect(err).To(HaveOccurred())
				Expect(errors.IsNotFound(err)).To(BeTrue())
			})

			It("store VirtualMachineInstancetype ControllerRevision", func() {

				instancetypeControllerRevision, err := instancetype.CreateControllerRevision(vm, f)
				Expect(err).ToNot(HaveOccurred())

				expectedRevisionNamePatch, err := instancetype.GenerateRevisionNamePatch(instancetypeControllerRevision, nil)
				Expect(err).ToNot(HaveOccurred())

				vmInterface.EXPECT().Patch(vm.Name, types.JSONPatchType, expectedRevisionNamePatch, &metav1.PatchOptions{})

				expectControllerRevisionCreation(instancetypeControllerRevision)

				Expect(instancetypeMethods.StoreControllerRevisions(vm)).To(Succeed())
				Expect(vm.Spec.Instancetype.RevisionName).To(Equal(instancetypeControllerRevision.Name))
			})

			It("store fails when instancetype does not exist", func() {
				vm.Spec.Instancetype.Name = "non-existing-instancetype"

				err := instancetypeMethods.StoreControllerRevisions(vm)
				Expect(err).To(HaveOccurred())
				Expect(errors.IsNotFound(err)).To(BeTrue())
			})

			It("store returns a nil revision when RevisionName already populated", func() {
				instancetypeControllerRevision, err := instancetype.CreateControllerRevision(vm, f)
				Expect(err).ToNot(HaveOccurred())

				_, err = virtClient.AppsV1().ControllerRevisions(vm.Namespace).Create(context.Background(), instancetypeControllerRevision, metav1.CreateOptions{})
				Expect(err).ToNot(HaveOccurred())

				vm.Spec.Instancetype = &v1.InstancetypeMatcher{
					Name:         f.Name,
					RevisionName: instancetypeControllerRevision.Name,
					Kind:         apiinstancetype.SingularResourceName,
				}

				Expect(instancetypeMethods.StoreControllerRevisions(vm)).To(Succeed())
				Expect(vm.Spec.Instancetype.RevisionName).To(Equal(instancetypeControllerRevision.Name))
			})

			It("store ControllerRevision succeeds if a revision exists with expected data", func() {

				instancetypeControllerRevision, err := instancetype.CreateControllerRevision(vm, f)
				Expect(err).ToNot(HaveOccurred())

				_, err = virtClient.AppsV1().ControllerRevisions(vm.Namespace).Create(context.Background(), instancetypeControllerRevision, metav1.CreateOptions{})
				Expect(err).ToNot(HaveOccurred())

				expectedRevisionNamePatch, err := instancetype.GenerateRevisionNamePatch(instancetypeControllerRevision, nil)
				Expect(err).ToNot(HaveOccurred())

				vmInterface.EXPECT().Patch(vm.Name, types.JSONPatchType, expectedRevisionNamePatch, &metav1.PatchOptions{})

				Expect(instancetypeMethods.StoreControllerRevisions(vm)).To(Succeed())
				Expect(vm.Spec.Instancetype.RevisionName).To(Equal(instancetypeControllerRevision.Name))

			})

			It("store ControllerRevision fails if a revision exists with unexpected data", func() {
				unexpectedInstancetype := f.DeepCopy()
				unexpectedInstancetype.Spec.CPU.Guest = 15

				instancetypeControllerRevision, err := instancetype.CreateControllerRevision(vm, unexpectedInstancetype)
				Expect(err).ToNot(HaveOccurred())

				_, err = virtClient.AppsV1().ControllerRevisions(vm.Namespace).Create(context.Background(), instancetypeControllerRevision, metav1.CreateOptions{})
				Expect(err).ToNot(HaveOccurred())

				Expect(instancetypeMethods.StoreControllerRevisions(vm)).To(MatchError(ContainSubstring("found existing ControllerRevision with unexpected data")))
			})

			It("store ControllerRevision fails if instancetype conflicts with vm", func() {
				vm.Spec.Template.Spec.Domain.CPU = &v1.CPU{
					Cores: 1,
				}

				Expect(instancetypeMethods.StoreControllerRevisions(vm)).To(MatchError(ContainSubstring("VM field conflicts with selected Instancetype")))
			})
		})
	})

	Context("Add instancetype name annotations", func() {
		const instancetypeName = "instancetype-name"

		BeforeEach(func() {
			vm = kubecli.NewMinimalVM("testvm")
			vm.Spec.Instancetype = &v1.InstancetypeMatcher{Name: instancetypeName}
		})

		It("should add instancetype name annotation", func() {
			vm.Spec.Instancetype.Kind = apiinstancetype.SingularResourceName

			meta := &metav1.ObjectMeta{}
			instancetype.AddInstancetypeNameAnnotations(vm, meta)

			Expect(meta.Annotations[v1.InstancetypeAnnotation]).To(Equal(instancetypeName))
			Expect(meta.Annotations[v1.ClusterInstancetypeAnnotation]).To(Equal(""))
		})

		It("should add cluster instancetype name annotation", func() {
			vm.Spec.Instancetype.Kind = apiinstancetype.ClusterSingularResourceName

			meta := &metav1.ObjectMeta{}
			instancetype.AddInstancetypeNameAnnotations(vm, meta)

			Expect(meta.Annotations[v1.InstancetypeAnnotation]).To(Equal(""))
			Expect(meta.Annotations[v1.ClusterInstancetypeAnnotation]).To(Equal(instancetypeName))
		})

		It("should add cluster name annotation, if instancetype.kind is empty", func() {
			vm.Spec.Instancetype.Kind = ""

			meta := &metav1.ObjectMeta{}
			instancetype.AddInstancetypeNameAnnotations(vm, meta)

			Expect(meta.Annotations[v1.InstancetypeAnnotation]).To(Equal(""))
			Expect(meta.Annotations[v1.ClusterInstancetypeAnnotation]).To(Equal(instancetypeName))
		})
	})

	Context("Find and store VirtualMachinePreferenceSpec", func() {

		It("find returns nil when no preference is specified", func() {
			vm.Spec.Preference = nil
			preference, err := instancetypeMethods.FindPreferenceSpec(vm)
			Expect(err).ToNot(HaveOccurred())
			Expect(preference).To(BeNil())
		})

		It("find returns error when invalid Preference Kind is specified", func() {
			vm.Spec.Preference = &v1.PreferenceMatcher{
				Name: "foo",
				Kind: "bar",
			}
			spec, err := instancetypeMethods.FindPreferenceSpec(vm)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("got unexpected kind in PreferenceMatcher"))
			Expect(spec).To(BeNil())
		})

		It("store returns error when preferenceMatcher kind is invalid", func() {
			vm.Spec.Preference = &v1.PreferenceMatcher{
				Kind: "foobar",
			}
			Expect(instancetypeMethods.StoreControllerRevisions(vm)).To(MatchError(ContainSubstring("got unexpected kind in PreferenceMatcher")))
		})

		It("store returns nil when no preference is specified", func() {
			vm.Spec.Preference = nil
			Expect(instancetypeMethods.StoreControllerRevisions(vm)).To(Succeed())
		})

		Context("Using global ClusterPreference", func() {
			var clusterPreference *instancetypev1alpha2.VirtualMachineClusterPreference
			var fakeClusterPreferenceClient v1alpha2.VirtualMachineClusterPreferenceInterface

			BeforeEach(func() {

				fakeClusterPreferenceClient = fakeInstancetypeClients.VirtualMachineClusterPreferences()
				virtClient.EXPECT().VirtualMachineClusterPreference().Return(fakeClusterPreferenceClient).AnyTimes()

				clusterPreference = &instancetypev1alpha2.VirtualMachineClusterPreference{
					ObjectMeta: metav1.ObjectMeta{
						Name:       "test-cluster-preference",
						UID:        resourceUID,
						Generation: resourceGeneration,
					},
					Spec: instancetypev1alpha2.VirtualMachinePreferenceSpec{
						CPU: &instancetypev1alpha2.CPUPreferences{
							PreferredCPUTopology: instancetypev1alpha2.PreferCores,
						},
					},
				}

				_, err := virtClient.VirtualMachineClusterPreference().Create(context.Background(), clusterPreference, metav1.CreateOptions{})
				Expect(err).ToNot(HaveOccurred())

				vm.Spec.Preference = &v1.PreferenceMatcher{
					Name: clusterPreference.Name,
					Kind: apiinstancetype.ClusterSingularPreferenceResourceName,
				}
			})

			It("find returns expected preference spec", func() {
				s, err := instancetypeMethods.FindPreferenceSpec(vm)
				Expect(err).ToNot(HaveOccurred())
				Expect(*s).To(Equal(clusterPreference.Spec))
			})

			It("find fails when preference does not exist", func() {
				vm.Spec.Preference.Name = "non-existing-preference"
				_, err := instancetypeMethods.FindPreferenceSpec(vm)
				Expect(err).To(HaveOccurred())
				Expect(errors.IsNotFound(err)).To(BeTrue())
			})

			It("store VirtualMachineClusterPreference ControllerRevision", func() {

				clusterPreferenceControllerRevision, err := instancetype.CreateControllerRevision(vm, clusterPreference)
				Expect(err).ToNot(HaveOccurred())

				expectedRevisionNamePatch, err := instancetype.GenerateRevisionNamePatch(nil, clusterPreferenceControllerRevision)
				Expect(err).ToNot(HaveOccurred())

				vmInterface.EXPECT().Patch(vm.Name, types.JSONPatchType, expectedRevisionNamePatch, &metav1.PatchOptions{})

				expectControllerRevisionCreation(clusterPreferenceControllerRevision)

				Expect(instancetypeMethods.StoreControllerRevisions(vm)).To(Succeed())
				Expect(vm.Spec.Preference.RevisionName).To(Equal(clusterPreferenceControllerRevision.Name))

			})

			It("store fails when VirtualMachineClusterPreference doesn't exist", func() {
				vm.Spec.Preference.Name = "non-existing-preference"

				err := instancetypeMethods.StoreControllerRevisions(vm)
				Expect(err).To(HaveOccurred())
				Expect(errors.IsNotFound(err)).To(BeTrue())

			})

			It("store returns nil revision when RevisionName already populated", func() {
				clusterPreferenceControllerRevision, err := instancetype.CreateControllerRevision(vm, clusterPreference)
				Expect(err).ToNot(HaveOccurred())

				_, err = virtClient.AppsV1().ControllerRevisions(vm.Namespace).Create(context.Background(), clusterPreferenceControllerRevision, metav1.CreateOptions{})
				Expect(err).ToNot(HaveOccurred())

				vm.Spec.Preference.RevisionName = clusterPreferenceControllerRevision.Name

				Expect(instancetypeMethods.StoreControllerRevisions(vm)).To(Succeed())
				Expect(vm.Spec.Preference.RevisionName).To(Equal(clusterPreferenceControllerRevision.Name))

			})

			It("store ControllerRevision succeeds if a revision exists with expected data", func() {

				clusterPreferenceControllerRevision, err := instancetype.CreateControllerRevision(vm, clusterPreference)
				Expect(err).ToNot(HaveOccurred())

				_, err = virtClient.AppsV1().ControllerRevisions(vm.Namespace).Create(context.Background(), clusterPreferenceControllerRevision, metav1.CreateOptions{})
				Expect(err).ToNot(HaveOccurred())

				expectedRevisionNamePatch, err := instancetype.GenerateRevisionNamePatch(nil, clusterPreferenceControllerRevision)
				Expect(err).ToNot(HaveOccurred())

				vmInterface.EXPECT().Patch(vm.Name, types.JSONPatchType, expectedRevisionNamePatch, &metav1.PatchOptions{})

				Expect(instancetypeMethods.StoreControllerRevisions(vm)).To(Succeed())
				Expect(vm.Spec.Preference.RevisionName).To(Equal(clusterPreferenceControllerRevision.Name))

			})

			It("store ControllerRevision fails if a revision exists with unexpected data", func() {
				unexpectedPreference := clusterPreference.DeepCopy()
				unexpectedPreference.Spec.CPU.PreferredCPUTopology = instancetypev1alpha2.PreferThreads

				clusterPreferenceControllerRevision, err := instancetype.CreateControllerRevision(vm, unexpectedPreference)
				Expect(err).ToNot(HaveOccurred())

				_, err = virtClient.AppsV1().ControllerRevisions(vm.Namespace).Create(context.Background(), clusterPreferenceControllerRevision, metav1.CreateOptions{})
				Expect(err).ToNot(HaveOccurred())

				Expect(instancetypeMethods.StoreControllerRevisions(vm)).To(MatchError(ContainSubstring("found existing ControllerRevision with unexpected data")))
			})
		})

		Context("Using namespaced Preference", func() {
			var preference *instancetypev1alpha2.VirtualMachinePreference
			var fakePreferenceClient v1alpha2.VirtualMachinePreferenceInterface

			BeforeEach(func() {

				fakePreferenceClient = fakeInstancetypeClients.VirtualMachinePreferences(vm.Namespace)
				virtClient.EXPECT().VirtualMachinePreference(gomock.Any()).Return(fakePreferenceClient).AnyTimes()

				preference = &instancetypev1alpha2.VirtualMachinePreference{
					ObjectMeta: metav1.ObjectMeta{
						Name:       "test-preference",
						Namespace:  vm.Namespace,
						UID:        resourceUID,
						Generation: resourceGeneration,
					},
					Spec: instancetypev1alpha2.VirtualMachinePreferenceSpec{
						CPU: &instancetypev1alpha2.CPUPreferences{
							PreferredCPUTopology: instancetypev1alpha2.PreferCores,
						},
					},
				}

				_, err := virtClient.VirtualMachinePreference(vm.Namespace).Create(context.Background(), preference, metav1.CreateOptions{})
				Expect(err).ToNot(HaveOccurred())

				vm.Spec.Preference = &v1.PreferenceMatcher{
					Name: preference.Name,
					Kind: apiinstancetype.SingularPreferenceResourceName,
				}
			})

			It("find returns expected preference spec", func() {
				s, err := instancetypeMethods.FindPreferenceSpec(vm)
				Expect(err).ToNot(HaveOccurred())
				Expect(*s).To(Equal(preference.Spec))
			})

			It("find fails when preference does not exist", func() {
				vm.Spec.Preference.Name = "non-existing-preference"
				_, err := instancetypeMethods.FindPreferenceSpec(vm)
				Expect(err).To(HaveOccurred())
				Expect(errors.IsNotFound(err)).To(BeTrue())
			})

			It("store VirtualMachinePreference ControllerRevision", func() {
				preferenceControllerRevision, err := instancetype.CreateControllerRevision(vm, preference)
				Expect(err).ToNot(HaveOccurred())

				expectedRevisionNamePatch, err := instancetype.GenerateRevisionNamePatch(nil, preferenceControllerRevision)
				Expect(err).ToNot(HaveOccurred())

				vmInterface.EXPECT().Patch(vm.Name, types.JSONPatchType, expectedRevisionNamePatch, &metav1.PatchOptions{})

				expectControllerRevisionCreation(preferenceControllerRevision)

				Expect(instancetypeMethods.StoreControllerRevisions(vm)).To(Succeed())
				Expect(vm.Spec.Preference.RevisionName).To(Equal(preferenceControllerRevision.Name))

			})

			It("store fails when VirtualMachinePreference doesn't exist", func() {
				vm.Spec.Preference.Name = "non-existing-preference"

				err := instancetypeMethods.StoreControllerRevisions(vm)
				Expect(err).To(HaveOccurred())
				Expect(errors.IsNotFound(err)).To(BeTrue())

			})

			It("store returns nil revision when RevisionName already populated", func() {
				preferenceControllerRevision, err := instancetype.CreateControllerRevision(vm, preference)
				Expect(err).ToNot(HaveOccurred())

				_, err = virtClient.AppsV1().ControllerRevisions(vm.Namespace).Create(context.Background(), preferenceControllerRevision, metav1.CreateOptions{})
				Expect(err).ToNot(HaveOccurred())

				vm.Spec.Preference.RevisionName = preferenceControllerRevision.Name

				Expect(instancetypeMethods.StoreControllerRevisions(vm)).To(Succeed())
				Expect(vm.Spec.Preference.RevisionName).To(Equal(preferenceControllerRevision.Name))

			})

			It("store ControllerRevision succeeds if a revision exists with expected data", func() {

				preferenceControllerRevision, err := instancetype.CreateControllerRevision(vm, preference)
				Expect(err).ToNot(HaveOccurred())

				_, err = virtClient.AppsV1().ControllerRevisions(vm.Namespace).Create(context.Background(), preferenceControllerRevision, metav1.CreateOptions{})
				Expect(err).ToNot(HaveOccurred())

				expectedRevisionNamePatch, err := instancetype.GenerateRevisionNamePatch(nil, preferenceControllerRevision)
				Expect(err).ToNot(HaveOccurred())

				vmInterface.EXPECT().Patch(vm.Name, types.JSONPatchType, expectedRevisionNamePatch, &metav1.PatchOptions{})

				Expect(instancetypeMethods.StoreControllerRevisions(vm)).To(Succeed())
				Expect(vm.Spec.Preference.RevisionName).To(Equal(preferenceControllerRevision.Name))

			})

			It("store ControllerRevision fails if a revision exists with unexpected data", func() {
				unexpectedPreference := preference.DeepCopy()
				unexpectedPreference.Spec.CPU.PreferredCPUTopology = instancetypev1alpha2.PreferThreads

				preferenceControllerRevision, err := instancetype.CreateControllerRevision(vm, unexpectedPreference)
				Expect(err).ToNot(HaveOccurred())

				_, err = virtClient.AppsV1().ControllerRevisions(vm.Namespace).Create(context.Background(), preferenceControllerRevision, metav1.CreateOptions{})
				Expect(err).ToNot(HaveOccurred())

				Expect(instancetypeMethods.StoreControllerRevisions(vm)).To(MatchError(ContainSubstring("found existing ControllerRevision with unexpected data")))
			})
		})
	})

	Context("Add preference name annotations", func() {
		const preferenceName = "preference-name"

		BeforeEach(func() {
			vm = kubecli.NewMinimalVM("testvm")
			vm.Spec.Preference = &v1.PreferenceMatcher{Name: preferenceName}
		})

		It("should add preference name annotation", func() {
			vm.Spec.Preference.Kind = apiinstancetype.SingularPreferenceResourceName

			meta := &metav1.ObjectMeta{}
			instancetype.AddPreferenceNameAnnotations(vm, meta)

			Expect(meta.Annotations[v1.PreferenceAnnotation]).To(Equal(preferenceName))
			Expect(meta.Annotations[v1.ClusterPreferenceAnnotation]).To(Equal(""))
		})

		It("should add cluster preference name annotation", func() {
			vm.Spec.Preference.Kind = apiinstancetype.ClusterSingularPreferenceResourceName

			meta := &metav1.ObjectMeta{}
			instancetype.AddPreferenceNameAnnotations(vm, meta)

			Expect(meta.Annotations[v1.PreferenceAnnotation]).To(Equal(""))
			Expect(meta.Annotations[v1.ClusterPreferenceAnnotation]).To(Equal(preferenceName))
		})

		It("should add cluster name annotation, if preference.kind is empty", func() {
			vm.Spec.Preference.Kind = ""

			meta := &metav1.ObjectMeta{}
			instancetype.AddPreferenceNameAnnotations(vm, meta)

			Expect(meta.Annotations[v1.PreferenceAnnotation]).To(Equal(""))
			Expect(meta.Annotations[v1.ClusterPreferenceAnnotation]).To(Equal(preferenceName))
		})
	})

	Context("Apply", func() {

		var (
			instancetypeSpec *instancetypev1alpha2.VirtualMachineInstancetypeSpec
			preferenceSpec   *instancetypev1alpha2.VirtualMachinePreferenceSpec
			field            *field.Path
		)

		BeforeEach(func() {
			vmi = api.NewMinimalVMI("testvmi")

			vmi.Spec = v1.VirtualMachineInstanceSpec{
				Domain: v1.DomainSpec{},
			}
			field = k8sfield.NewPath("spec", "template", "spec")
		})

		Context("instancetype.spec.CPU and preference.spec.CPU", func() {

			BeforeEach(func() {

				instancetypeSpec = &instancetypev1alpha2.VirtualMachineInstancetypeSpec{
					CPU: instancetypev1alpha2.CPUInstancetype{
						Guest:                 uint32(2),
						Model:                 "host-passthrough",
						DedicatedCPUPlacement: true,
						IsolateEmulatorThread: true,
						NUMA: &v1.NUMA{
							GuestMappingPassthrough: &v1.NUMAGuestMappingPassthrough{},
						},
						Realtime: &v1.Realtime{
							Mask: "0-3,^1",
						},
					},
				}
				preferenceSpec = &instancetypev1alpha2.VirtualMachinePreferenceSpec{
					CPU: &instancetypev1alpha2.CPUPreferences{},
				}
			})

			It("should default to PreferSockets", func() {

				conflicts := instancetypeMethods.ApplyToVmi(field, instancetypeSpec, preferenceSpec, &vmi.Spec)
				Expect(conflicts).To(BeEmpty())

				Expect(vmi.Spec.Domain.CPU.Sockets).To(Equal(instancetypeSpec.CPU.Guest))
				Expect(vmi.Spec.Domain.CPU.Cores).To(Equal(uint32(1)))
				Expect(vmi.Spec.Domain.CPU.Threads).To(Equal(uint32(1)))
				Expect(vmi.Spec.Domain.CPU.Model).To(Equal(instancetypeSpec.CPU.Model))
				Expect(vmi.Spec.Domain.CPU.DedicatedCPUPlacement).To(Equal(instancetypeSpec.CPU.DedicatedCPUPlacement))
				Expect(vmi.Spec.Domain.CPU.IsolateEmulatorThread).To(Equal(instancetypeSpec.CPU.IsolateEmulatorThread))
				Expect(*vmi.Spec.Domain.CPU.NUMA).To(Equal(*instancetypeSpec.CPU.NUMA))
				Expect(*vmi.Spec.Domain.CPU.Realtime).To(Equal(*instancetypeSpec.CPU.Realtime))

			})

			It("should apply in full with PreferCores selected", func() {

				preferenceSpec.CPU.PreferredCPUTopology = instancetypev1alpha2.PreferCores

				conflicts := instancetypeMethods.ApplyToVmi(field, instancetypeSpec, preferenceSpec, &vmi.Spec)
				Expect(conflicts).To(BeEmpty())

				Expect(vmi.Spec.Domain.CPU.Sockets).To(Equal(uint32(1)))
				Expect(vmi.Spec.Domain.CPU.Cores).To(Equal(instancetypeSpec.CPU.Guest))
				Expect(vmi.Spec.Domain.CPU.Threads).To(Equal(uint32(1)))
				Expect(vmi.Spec.Domain.CPU.Model).To(Equal(instancetypeSpec.CPU.Model))
				Expect(vmi.Spec.Domain.CPU.DedicatedCPUPlacement).To(Equal(instancetypeSpec.CPU.DedicatedCPUPlacement))
				Expect(vmi.Spec.Domain.CPU.IsolateEmulatorThread).To(Equal(instancetypeSpec.CPU.IsolateEmulatorThread))
				Expect(*vmi.Spec.Domain.CPU.NUMA).To(Equal(*instancetypeSpec.CPU.NUMA))
				Expect(*vmi.Spec.Domain.CPU.Realtime).To(Equal(*instancetypeSpec.CPU.Realtime))

			})

			It("should apply in full with PreferThreads selected", func() {

				preferenceSpec.CPU.PreferredCPUTopology = instancetypev1alpha2.PreferThreads

				conflicts := instancetypeMethods.ApplyToVmi(field, instancetypeSpec, preferenceSpec, &vmi.Spec)
				Expect(conflicts).To(BeEmpty())

				Expect(vmi.Spec.Domain.CPU.Sockets).To(Equal(uint32(1)))
				Expect(vmi.Spec.Domain.CPU.Cores).To(Equal(uint32(1)))
				Expect(vmi.Spec.Domain.CPU.Threads).To(Equal(instancetypeSpec.CPU.Guest))
				Expect(vmi.Spec.Domain.CPU.Model).To(Equal(instancetypeSpec.CPU.Model))
				Expect(vmi.Spec.Domain.CPU.DedicatedCPUPlacement).To(Equal(instancetypeSpec.CPU.DedicatedCPUPlacement))
				Expect(vmi.Spec.Domain.CPU.IsolateEmulatorThread).To(Equal(instancetypeSpec.CPU.IsolateEmulatorThread))
				Expect(*vmi.Spec.Domain.CPU.NUMA).To(Equal(*instancetypeSpec.CPU.NUMA))
				Expect(*vmi.Spec.Domain.CPU.Realtime).To(Equal(*instancetypeSpec.CPU.Realtime))

			})

			It("should apply in full with PreferSockets selected", func() {

				preferenceSpec.CPU.PreferredCPUTopology = instancetypev1alpha2.PreferSockets

				conflicts := instancetypeMethods.ApplyToVmi(field, instancetypeSpec, preferenceSpec, &vmi.Spec)
				Expect(conflicts).To(BeEmpty())

				Expect(vmi.Spec.Domain.CPU.Sockets).To(Equal(instancetypeSpec.CPU.Guest))
				Expect(vmi.Spec.Domain.CPU.Cores).To(Equal(uint32(1)))
				Expect(vmi.Spec.Domain.CPU.Threads).To(Equal(uint32(1)))
				Expect(vmi.Spec.Domain.CPU.Model).To(Equal(instancetypeSpec.CPU.Model))
				Expect(vmi.Spec.Domain.CPU.DedicatedCPUPlacement).To(Equal(instancetypeSpec.CPU.DedicatedCPUPlacement))
				Expect(vmi.Spec.Domain.CPU.IsolateEmulatorThread).To(Equal(instancetypeSpec.CPU.IsolateEmulatorThread))
				Expect(*vmi.Spec.Domain.CPU.NUMA).To(Equal(*instancetypeSpec.CPU.NUMA))
				Expect(*vmi.Spec.Domain.CPU.Realtime).To(Equal(*instancetypeSpec.CPU.Realtime))

			})

			It("should return a conflict if vmi.Spec.Domain.CPU already defined", func() {

				instancetypeSpec = &instancetypev1alpha2.VirtualMachineInstancetypeSpec{
					CPU: instancetypev1alpha2.CPUInstancetype{
						Guest: uint32(2),
					},
				}

				vmi.Spec.Domain.CPU = &v1.CPU{
					Cores:   4,
					Sockets: 1,
					Threads: 1,
				}

				conflicts := instancetypeMethods.ApplyToVmi(field, instancetypeSpec, preferenceSpec, &vmi.Spec)
				Expect(conflicts).To(HaveLen(1))
				Expect(conflicts[0].String()).To(Equal("spec.template.spec.domain.cpu"))

			})

			It("should return a conflict if vmi.Spec.Domain.Resources.Requests[k8sv1.ResourceCPU] already defined", func() {

				instancetypeSpec = &instancetypev1alpha2.VirtualMachineInstancetypeSpec{
					CPU: instancetypev1alpha2.CPUInstancetype{
						Guest: uint32(2),
					},
				}

				vmi.Spec.Domain.Resources = v1.ResourceRequirements{
					Requests: k8sv1.ResourceList{
						k8sv1.ResourceCPU: resource.MustParse("1"),
					},
				}

				conflicts := instancetypeMethods.ApplyToVmi(field, instancetypeSpec, preferenceSpec, &vmi.Spec)
				Expect(conflicts).To(HaveLen(1))
				Expect(conflicts[0].String()).To(Equal("spec.template.spec.domain.resources.requests.cpu"))

			})

			It("should return a conflict if vmi.Spec.Domain.Resources.Limits[k8sv1.ResourceCPU] already defined", func() {

				instancetypeSpec = &instancetypev1alpha2.VirtualMachineInstancetypeSpec{
					CPU: instancetypev1alpha2.CPUInstancetype{
						Guest: uint32(2),
					},
				}

				vmi.Spec.Domain.Resources = v1.ResourceRequirements{
					Limits: k8sv1.ResourceList{
						k8sv1.ResourceCPU: resource.MustParse("1"),
					},
				}

				conflicts := instancetypeMethods.ApplyToVmi(field, instancetypeSpec, preferenceSpec, &vmi.Spec)
				Expect(conflicts).To(HaveLen(1))
				Expect(conflicts[0].String()).To(Equal("spec.template.spec.domain.resources.limits.cpu"))

			})
		})
		Context("instancetype.Spec.Memory", func() {
			BeforeEach(func() {
				instancetypeSpec = &instancetypev1alpha2.VirtualMachineInstancetypeSpec{
					Memory: instancetypev1alpha2.MemoryInstancetype{
						Guest: resource.MustParse("512M"),
						Hugepages: &v1.Hugepages{
							PageSize: "1Gi",
						},
					},
				}
			})

			It("should apply to VMI", func() {

				conflicts := instancetypeMethods.ApplyToVmi(field, instancetypeSpec, preferenceSpec, &vmi.Spec)
				Expect(conflicts).To(BeEmpty())

				Expect(*vmi.Spec.Domain.Memory.Guest).To(Equal(instancetypeSpec.Memory.Guest))
				Expect(*vmi.Spec.Domain.Memory.Hugepages).To(Equal(*instancetypeSpec.Memory.Hugepages))

			})

			It("should detect memory conflict", func() {

				vmiMemGuest := resource.MustParse("512M")
				vmi.Spec.Domain.Memory = &v1.Memory{
					Guest: &vmiMemGuest,
				}

				conflicts := instancetypeMethods.ApplyToVmi(field, instancetypeSpec, preferenceSpec, &vmi.Spec)
				Expect(conflicts).To(HaveLen(1))
				Expect(conflicts[0].String()).To(Equal("spec.template.spec.domain.memory"))

			})

			It("should return a conflict if vmi.Spec.Domain.Resources.Requests[k8sv1.ResourceMemory] already defined", func() {

				instancetypeSpec = &instancetypev1alpha2.VirtualMachineInstancetypeSpec{
					Memory: instancetypev1alpha2.MemoryInstancetype{
						Guest: resource.MustParse("512M"),
					},
				}

				vmi.Spec.Domain.Resources = v1.ResourceRequirements{
					Requests: k8sv1.ResourceList{
						k8sv1.ResourceMemory: resource.MustParse("128Mi"),
					},
				}

				conflicts := instancetypeMethods.ApplyToVmi(field, instancetypeSpec, preferenceSpec, &vmi.Spec)
				Expect(conflicts).To(HaveLen(1))
				Expect(conflicts[0].String()).To(Equal("spec.template.spec.domain.resources.requests.memory"))

			})

			It("should return a conflict if vmi.Spec.Domain.Resources.Limits[k8sv1.ResourceMemory] already defined", func() {

				instancetypeSpec = &instancetypev1alpha2.VirtualMachineInstancetypeSpec{
					Memory: instancetypev1alpha2.MemoryInstancetype{
						Guest: resource.MustParse("512M"),
					},
				}

				vmi.Spec.Domain.Resources = v1.ResourceRequirements{
					Limits: k8sv1.ResourceList{
						k8sv1.ResourceMemory: resource.MustParse("128Mi"),
					},
				}

				conflicts := instancetypeMethods.ApplyToVmi(field, instancetypeSpec, preferenceSpec, &vmi.Spec)
				Expect(conflicts).To(HaveLen(1))
				Expect(conflicts[0].String()).To(Equal("spec.template.spec.domain.resources.limits.memory"))

			})
		})
		Context("instancetype.Spec.ioThreadsPolicy", func() {

			var instancetypePolicy v1.IOThreadsPolicy

			BeforeEach(func() {
				instancetypePolicy = v1.IOThreadsPolicyShared
				instancetypeSpec = &instancetypev1alpha2.VirtualMachineInstancetypeSpec{
					IOThreadsPolicy: &instancetypePolicy,
				}
			})

			It("should apply to VMI", func() {
				conflicts := instancetypeMethods.ApplyToVmi(field, instancetypeSpec, preferenceSpec, &vmi.Spec)
				Expect(conflicts).To(HaveLen(0))

				Expect(*vmi.Spec.Domain.IOThreadsPolicy).To(Equal(*instancetypeSpec.IOThreadsPolicy))
			})

			It("should detect IOThreadsPolicy conflict", func() {
				vmi.Spec.Domain.IOThreadsPolicy = &instancetypePolicy

				conflicts := instancetypeMethods.ApplyToVmi(field, instancetypeSpec, preferenceSpec, &vmi.Spec)
				Expect(conflicts).To(HaveLen(1))
				Expect(conflicts[0].String()).To(Equal("spec.template.spec.domain.ioThreadsPolicy"))
			})
		})

		Context("instancetype.Spec.LaunchSecurity", func() {

			BeforeEach(func() {
				instancetypeSpec = &instancetypev1alpha2.VirtualMachineInstancetypeSpec{
					LaunchSecurity: &v1.LaunchSecurity{
						SEV: &v1.SEV{},
					},
				}
			})

			It("should apply to VMI", func() {
				conflicts := instancetypeMethods.ApplyToVmi(field, instancetypeSpec, preferenceSpec, &vmi.Spec)
				Expect(conflicts).To(HaveLen(0))

				Expect(*vmi.Spec.Domain.LaunchSecurity).To(Equal(*instancetypeSpec.LaunchSecurity))
			})

			It("should detect LaunchSecurity conflict", func() {
				vmi.Spec.Domain.LaunchSecurity = &v1.LaunchSecurity{
					SEV: &v1.SEV{},
				}

				conflicts := instancetypeMethods.ApplyToVmi(field, instancetypeSpec, preferenceSpec, &vmi.Spec)
				Expect(conflicts).To(HaveLen(1))
				Expect(conflicts[0].String()).To(Equal("spec.template.spec.domain.launchSecurity"))
			})
		})

		Context("instancetype.Spec.GPUs", func() {

			BeforeEach(func() {
				instancetypeSpec = &instancetypev1alpha2.VirtualMachineInstancetypeSpec{
					GPUs: []v1.GPU{
						v1.GPU{
							Name:       "barfoo",
							DeviceName: "vendor.com/gpu_name",
						},
					},
				}
			})

			It("should apply to VMI", func() {

				conflicts := instancetypeMethods.ApplyToVmi(field, instancetypeSpec, preferenceSpec, &vmi.Spec)
				Expect(conflicts).To(HaveLen(0))

				Expect(vmi.Spec.Domain.Devices.GPUs).To(Equal(instancetypeSpec.GPUs))

			})

			It("should detect GPU conflict", func() {

				vmi.Spec.Domain.Devices.GPUs = []v1.GPU{
					v1.GPU{
						Name:       "foobar",
						DeviceName: "vendor.com/gpu_name",
					},
				}

				conflicts := instancetypeMethods.ApplyToVmi(field, instancetypeSpec, preferenceSpec, &vmi.Spec)
				Expect(conflicts).To(HaveLen(1))
				Expect(conflicts[0].String()).To(Equal("spec.template.spec.domain.devices.gpus"))

			})
		})

		Context("instancetype.Spec.HostDevices", func() {

			BeforeEach(func() {
				instancetypeSpec = &instancetypev1alpha2.VirtualMachineInstancetypeSpec{
					HostDevices: []v1.HostDevice{
						v1.HostDevice{
							Name:       "foobar",
							DeviceName: "vendor.com/device_name",
						},
					},
				}
			})

			It("should apply to VMI", func() {

				conflicts := instancetypeMethods.ApplyToVmi(field, instancetypeSpec, preferenceSpec, &vmi.Spec)
				Expect(conflicts).To(HaveLen(0))

				Expect(vmi.Spec.Domain.Devices.HostDevices).To(Equal(instancetypeSpec.HostDevices))

			})

			It("should detect HostDevice conflict", func() {

				vmi.Spec.Domain.Devices.HostDevices = []v1.HostDevice{
					v1.HostDevice{
						Name:       "foobar",
						DeviceName: "vendor.com/device_name",
					},
				}

				conflicts := instancetypeMethods.ApplyToVmi(field, instancetypeSpec, preferenceSpec, &vmi.Spec)
				Expect(conflicts).To(HaveLen(1))
				Expect(conflicts[0].String()).To(Equal("spec.template.spec.domain.devices.hostDevices"))

			})
		})

		// TODO - break this up into smaller more targeted tests
		Context("Preference.Devices", func() {

			var userDefinedBlockSize *v1.BlockSize

			BeforeEach(func() {

				userDefinedBlockSize = &v1.BlockSize{
					Custom: &v1.CustomBlockSize{
						Logical:  512,
						Physical: 512,
					},
				}
				vmi.Spec.Domain.Devices.AutoattachGraphicsDevice = pointer.Bool(false)
				vmi.Spec.Domain.Devices.AutoattachMemBalloon = pointer.Bool(false)
				vmi.Spec.Domain.Devices.Disks = []v1.Disk{
					v1.Disk{
						Cache:             v1.CacheWriteBack,
						IO:                v1.IODefault,
						DedicatedIOThread: pointer.Bool(false),
						BlockSize:         userDefinedBlockSize,
						DiskDevice: v1.DiskDevice{
							Disk: &v1.DiskTarget{
								Bus: v1.DiskBusSCSI,
							},
						},
					},
					v1.Disk{
						DiskDevice: v1.DiskDevice{
							Disk: &v1.DiskTarget{},
						},
					},
					v1.Disk{
						DiskDevice: v1.DiskDevice{
							CDRom: &v1.CDRomTarget{
								Bus: v1.DiskBusSATA,
							},
						},
					},
					v1.Disk{
						DiskDevice: v1.DiskDevice{
							CDRom: &v1.CDRomTarget{},
						},
					},
					v1.Disk{
						DiskDevice: v1.DiskDevice{
							LUN: &v1.LunTarget{
								Bus: v1.DiskBusSATA,
							},
						},
					},
					v1.Disk{
						DiskDevice: v1.DiskDevice{
							LUN: &v1.LunTarget{},
						},
					},
				}
				vmi.Spec.Domain.Devices.Inputs = []v1.Input{
					v1.Input{
						Bus:  "usb",
						Type: "tablet",
					},
					v1.Input{},
				}
				vmi.Spec.Domain.Devices.Interfaces = []v1.Interface{
					v1.Interface{
						Model: "e1000",
					},
					v1.Interface{},
				}
				vmi.Spec.Domain.Devices.Sound = &v1.SoundDevice{}

				preferenceSpec = &instancetypev1alpha2.VirtualMachinePreferenceSpec{
					Devices: &instancetypev1alpha2.DevicePreferences{
						PreferredAutoattachGraphicsDevice:   pointer.Bool(true),
						PreferredAutoattachMemBalloon:       pointer.Bool(true),
						PreferredAutoattachPodInterface:     pointer.Bool(true),
						PreferredAutoattachSerialConsole:    pointer.Bool(true),
						PreferredAutoattachInputDevice:      pointer.Bool(true),
						PreferredDiskDedicatedIoThread:      pointer.Bool(true),
						PreferredDisableHotplug:             pointer.Bool(true),
						PreferredUseVirtioTransitional:      pointer.Bool(true),
						PreferredNetworkInterfaceMultiQueue: pointer.Bool(true),
						PreferredBlockMultiQueue:            pointer.Bool(true),
						PreferredDiskBlockSize: &v1.BlockSize{
							Custom: &v1.CustomBlockSize{
								Logical:  4096,
								Physical: 4096,
							},
						},
						PreferredDiskCache:      v1.CacheWriteThrough,
						PreferredDiskIO:         v1.IONative,
						PreferredDiskBus:        v1.DiskBusVirtio,
						PreferredCdromBus:       v1.DiskBusSCSI,
						PreferredLunBus:         v1.DiskBusSATA,
						PreferredInputBus:       v1.InputBusVirtio,
						PreferredInputType:      v1.InputTypeTablet,
						PreferredInterfaceModel: v1.VirtIO,
						PreferredSoundModel:     "ac97",
						PreferredRng:            &v1.Rng{},
						PreferredTPM:            &v1.TPMDevice{},
					},
				}

			})

			It("should apply to VMI", func() {

				conflicts := instancetypeMethods.ApplyToVmi(field, instancetypeSpec, preferenceSpec, &vmi.Spec)
				Expect(conflicts).To(HaveLen(0))

				Expect(*vmi.Spec.Domain.Devices.AutoattachGraphicsDevice).To(BeFalse())
				Expect(*vmi.Spec.Domain.Devices.AutoattachMemBalloon).To(BeFalse())
				Expect(*vmi.Spec.Domain.Devices.AutoattachInputDevice).To(BeTrue())
				Expect(vmi.Spec.Domain.Devices.Disks[0].Cache).To(Equal(v1.CacheWriteBack))
				Expect(vmi.Spec.Domain.Devices.Disks[0].IO).To(Equal(v1.IODefault))
				Expect(*vmi.Spec.Domain.Devices.Disks[0].DedicatedIOThread).To(BeFalse())
				Expect(*vmi.Spec.Domain.Devices.Disks[0].BlockSize).To(Equal(*userDefinedBlockSize))
				Expect(vmi.Spec.Domain.Devices.Disks[0].DiskDevice.Disk.Bus).To(Equal(v1.DiskBusSCSI))
				Expect(vmi.Spec.Domain.Devices.Disks[2].DiskDevice.CDRom.Bus).To(Equal(v1.DiskBusSATA))
				Expect(vmi.Spec.Domain.Devices.Disks[4].DiskDevice.LUN.Bus).To(Equal(v1.DiskBusSATA))
				Expect(vmi.Spec.Domain.Devices.Inputs[0].Bus).To(Equal(v1.InputBusUSB))
				Expect(vmi.Spec.Domain.Devices.Inputs[0].Type).To(Equal(v1.InputTypeTablet))
				Expect(vmi.Spec.Domain.Devices.Interfaces[0].Model).To(Equal("e1000"))

				// Assert that everything that isn't defined in the VM/VMI should use Preferences
				Expect(*vmi.Spec.Domain.Devices.AutoattachPodInterface).To(Equal(*preferenceSpec.Devices.PreferredAutoattachPodInterface))
				Expect(*vmi.Spec.Domain.Devices.AutoattachSerialConsole).To(Equal(*preferenceSpec.Devices.PreferredAutoattachSerialConsole))
				Expect(vmi.Spec.Domain.Devices.DisableHotplug).To(Equal(*preferenceSpec.Devices.PreferredDisableHotplug))
				Expect(*vmi.Spec.Domain.Devices.UseVirtioTransitional).To(Equal(*preferenceSpec.Devices.PreferredUseVirtioTransitional))
				Expect(vmi.Spec.Domain.Devices.Disks[1].Cache).To(Equal(preferenceSpec.Devices.PreferredDiskCache))
				Expect(vmi.Spec.Domain.Devices.Disks[1].IO).To(Equal(preferenceSpec.Devices.PreferredDiskIO))
				Expect(*vmi.Spec.Domain.Devices.Disks[1].DedicatedIOThread).To(Equal(*preferenceSpec.Devices.PreferredDiskDedicatedIoThread))
				Expect(*vmi.Spec.Domain.Devices.Disks[1].BlockSize).To(Equal(*preferenceSpec.Devices.PreferredDiskBlockSize))
				Expect(vmi.Spec.Domain.Devices.Disks[1].DiskDevice.Disk.Bus).To(Equal(preferenceSpec.Devices.PreferredDiskBus))
				Expect(vmi.Spec.Domain.Devices.Disks[3].DiskDevice.CDRom.Bus).To(Equal(preferenceSpec.Devices.PreferredCdromBus))
				Expect(vmi.Spec.Domain.Devices.Disks[5].DiskDevice.LUN.Bus).To(Equal(preferenceSpec.Devices.PreferredLunBus))
				Expect(vmi.Spec.Domain.Devices.Inputs[1].Bus).To(Equal(preferenceSpec.Devices.PreferredInputBus))
				Expect(vmi.Spec.Domain.Devices.Inputs[1].Type).To(Equal(preferenceSpec.Devices.PreferredInputType))
				Expect(vmi.Spec.Domain.Devices.Interfaces[1].Model).To(Equal(preferenceSpec.Devices.PreferredInterfaceModel))
				Expect(vmi.Spec.Domain.Devices.Sound.Model).To(Equal(preferenceSpec.Devices.PreferredSoundModel))
				Expect(*vmi.Spec.Domain.Devices.Rng).To(Equal(*preferenceSpec.Devices.PreferredRng))
				Expect(*vmi.Spec.Domain.Devices.NetworkInterfaceMultiQueue).To(Equal(*preferenceSpec.Devices.PreferredNetworkInterfaceMultiQueue))
				Expect(*vmi.Spec.Domain.Devices.BlockMultiQueue).To(Equal(*preferenceSpec.Devices.PreferredBlockMultiQueue))
				Expect(*vmi.Spec.Domain.Devices.TPM).To(Equal(*preferenceSpec.Devices.PreferredTPM))

			})

			It("Should apply when a VMI disk doesn't have a DiskDevice target defined", func() {

				vmi.Spec.Domain.Devices.Disks[1].DiskDevice.Disk = nil

				conflicts := instancetypeMethods.ApplyToVmi(field, instancetypeSpec, preferenceSpec, &vmi.Spec)
				Expect(conflicts).To(HaveLen(0))

				Expect(vmi.Spec.Domain.Devices.Disks[1].DiskDevice.Disk.Bus).To(Equal(preferenceSpec.Devices.PreferredDiskBus))

			})
		})

		Context("Preference.Features", func() {

			BeforeEach(func() {
				spinLockRetries := uint32(32)
				preferenceSpec = &instancetypev1alpha2.VirtualMachinePreferenceSpec{
					Features: &instancetypev1alpha2.FeaturePreferences{
						PreferredAcpi: &v1.FeatureState{},
						PreferredApic: &v1.FeatureAPIC{
							Enabled:        pointer.Bool(true),
							EndOfInterrupt: false,
						},
						PreferredHyperv: &v1.FeatureHyperv{
							Relaxed: &v1.FeatureState{},
							VAPIC:   &v1.FeatureState{},
							Spinlocks: &v1.FeatureSpinlocks{
								Enabled: pointer.Bool(true),
								Retries: &spinLockRetries,
							},
							VPIndex: &v1.FeatureState{},
							Runtime: &v1.FeatureState{},
							SyNIC:   &v1.FeatureState{},
							SyNICTimer: &v1.SyNICTimer{
								Enabled: pointer.Bool(true),
								Direct:  &v1.FeatureState{},
							},
							Reset: &v1.FeatureState{},
							VendorID: &v1.FeatureVendorID{
								Enabled:  pointer.Bool(true),
								VendorID: "1234",
							},
							Frequencies:     &v1.FeatureState{},
							Reenlightenment: &v1.FeatureState{},
							TLBFlush:        &v1.FeatureState{},
							IPI:             &v1.FeatureState{},
							EVMCS:           &v1.FeatureState{},
						},
						PreferredKvm: &v1.FeatureKVM{
							Hidden: true,
						},
						PreferredPvspinlock: &v1.FeatureState{},
						PreferredSmm:        &v1.FeatureState{},
					},
				}
			})

			It("should apply to VMI", func() {
				conflicts := instancetypeMethods.ApplyToVmi(field, instancetypeSpec, preferenceSpec, &vmi.Spec)
				Expect(conflicts).To(HaveLen(0))

				Expect(vmi.Spec.Domain.Features.ACPI).To(Equal(*preferenceSpec.Features.PreferredAcpi))
				Expect(*vmi.Spec.Domain.Features.APIC).To(Equal(*preferenceSpec.Features.PreferredApic))
				Expect(*vmi.Spec.Domain.Features.Hyperv).To(Equal(*preferenceSpec.Features.PreferredHyperv))
				Expect(*vmi.Spec.Domain.Features.KVM).To(Equal(*preferenceSpec.Features.PreferredKvm))
				Expect(*vmi.Spec.Domain.Features.Pvspinlock).To(Equal(*preferenceSpec.Features.PreferredPvspinlock))
				Expect(*vmi.Spec.Domain.Features.SMM).To(Equal(*preferenceSpec.Features.PreferredSmm))
			})

			It("should apply when some HyperV features already defined in the VMI", func() {

				vmi.Spec.Domain.Features = &v1.Features{
					Hyperv: &v1.FeatureHyperv{
						EVMCS: &v1.FeatureState{
							Enabled: pointer.Bool(false),
						},
					},
				}

				conflicts := instancetypeMethods.ApplyToVmi(field, instancetypeSpec, preferenceSpec, &vmi.Spec)
				Expect(conflicts).To(HaveLen(0))

				Expect(*vmi.Spec.Domain.Features.Hyperv.EVMCS.Enabled).To(BeFalse())

			})
		})

		Context("Preference.Firmware", func() {

			It("should apply BIOS preferences full to VMI", func() {
				preferenceSpec = &instancetypev1alpha2.VirtualMachinePreferenceSpec{
					Firmware: &instancetypev1alpha2.FirmwarePreferences{
						PreferredUseBios:       pointer.Bool(true),
						PreferredUseBiosSerial: pointer.Bool(true),
						PreferredUseEfi:        pointer.Bool(false),
						PreferredUseSecureBoot: pointer.Bool(false),
					},
				}

				conflicts := instancetypeMethods.ApplyToVmi(field, instancetypeSpec, preferenceSpec, &vmi.Spec)
				Expect(conflicts).To(HaveLen(0))

				Expect(*vmi.Spec.Domain.Firmware.Bootloader.BIOS.UseSerial).To(Equal(*preferenceSpec.Firmware.PreferredUseBiosSerial))
			})

			It("should apply SecureBoot preferences full to VMI", func() {
				preferenceSpec = &instancetypev1alpha2.VirtualMachinePreferenceSpec{
					Firmware: &instancetypev1alpha2.FirmwarePreferences{
						PreferredUseBios:       pointer.Bool(false),
						PreferredUseBiosSerial: pointer.Bool(false),
						PreferredUseEfi:        pointer.Bool(true),
						PreferredUseSecureBoot: pointer.Bool(true),
					},
				}

				conflicts := instancetypeMethods.ApplyToVmi(field, instancetypeSpec, preferenceSpec, &vmi.Spec)
				Expect(conflicts).To(HaveLen(0))

				Expect(*vmi.Spec.Domain.Firmware.Bootloader.EFI.SecureBoot).To(Equal(*preferenceSpec.Firmware.PreferredUseSecureBoot))
			})
		})

		Context("Preference.Machine", func() {

			It("should apply to VMI", func() {
				preferenceSpec = &instancetypev1alpha2.VirtualMachinePreferenceSpec{
					Machine: &instancetypev1alpha2.MachinePreferences{
						PreferredMachineType: "q35-rhel-8.0",
					},
				}

				conflicts := instancetypeMethods.ApplyToVmi(field, instancetypeSpec, preferenceSpec, &vmi.Spec)
				Expect(conflicts).To(HaveLen(0))

				Expect(vmi.Spec.Domain.Machine.Type).To(Equal(preferenceSpec.Machine.PreferredMachineType))
			})
		})
		Context("Preference.Clock", func() {

			It("should apply to VMI", func() {
				preferenceSpec = &instancetypev1alpha2.VirtualMachinePreferenceSpec{
					Clock: &instancetypev1alpha2.ClockPreferences{
						PreferredClockOffset: &v1.ClockOffset{
							UTC: &v1.ClockOffsetUTC{
								OffsetSeconds: pointer.Int(30),
							},
						},
						PreferredTimer: &v1.Timer{
							Hyperv: &v1.HypervTimer{},
						},
					},
				}

				conflicts := instancetypeMethods.ApplyToVmi(field, instancetypeSpec, preferenceSpec, &vmi.Spec)
				Expect(conflicts).To(HaveLen(0))

				Expect(*&vmi.Spec.Domain.Clock.ClockOffset).To(Equal(*preferenceSpec.Clock.PreferredClockOffset))
				Expect(*vmi.Spec.Domain.Clock.Timer).To(Equal(*preferenceSpec.Clock.PreferredTimer))
			})
		})
	})
})
