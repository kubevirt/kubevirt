//nolint:dupl,lll
package instancetype_test

import (
	"context"
	"fmt"

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
	"k8s.io/apimachinery/pkg/util/json"
	"k8s.io/client-go/testing"
	"k8s.io/client-go/tools/cache"
	"k8s.io/utils/pointer"

	k8sfield "k8s.io/apimachinery/pkg/util/validation/field"
	k8sfake "k8s.io/client-go/kubernetes/fake"

	"kubevirt.io/client-go/api"
	"kubevirt.io/client-go/kubecli"

	"kubevirt.io/kubevirt/pkg/instancetype"
	"kubevirt.io/kubevirt/pkg/testutils"

	v1 "kubevirt.io/api/core/v1"
	apiinstancetype "kubevirt.io/api/instancetype"
	instancetypev1alpha1 "kubevirt.io/api/instancetype/v1alpha1"
	instancetypev1beta1 "kubevirt.io/api/instancetype/v1beta1"
	fakeclientset "kubevirt.io/client-go/generated/kubevirt/clientset/versioned/fake"
	instancetypeclientv1beta1 "kubevirt.io/client-go/generated/kubevirt/clientset/versioned/typed/instancetype/v1beta1"
)

const resourceUID types.UID = "9160e5de-2540-476a-86d9-af0081aee68a"
const resourceGeneration int64 = 1
const nonExistingResourceName = "non-existing-resource"

var _ = Describe("Instancetype and Preferences", func() {
	var (
		ctrl                             *gomock.Controller
		instancetypeMethods              instancetype.Methods
		vm                               *v1.VirtualMachine
		vmi                              *v1.VirtualMachineInstance
		virtClient                       *kubecli.MockKubevirtClient
		vmInterface                      *kubecli.MockVirtualMachineInterface
		fakeInstancetypeClients          instancetypeclientv1beta1.InstancetypeV1beta1Interface
		k8sClient                        *k8sfake.Clientset
		instancetypeInformerStore        cache.Store
		clusterInstancetypeInformerStore cache.Store
		preferenceInformerStore          cache.Store
		clusterPreferenceInformerStore   cache.Store
		controllerrevisionInformerStore  cache.Store
	)

	expectControllerRevisionCreation := func(expectedControllerRevision *appsv1.ControllerRevision) {
		k8sClient.Fake.PrependReactor("create", "controllerrevisions", func(action testing.Action) (handled bool, obj runtime.Object, err error) {
			created, ok := action.(testing.CreateAction)
			Expect(ok).To(BeTrue())

			createObj := created.GetObject().(*appsv1.ControllerRevision)

			// This is already covered by the below assertion but be explicit here to ensure coverage
			Expect(createObj.Labels).To(HaveKey(apiinstancetype.ControllerRevisionObjectGenerationLabel))
			Expect(createObj.Labels).To(HaveKey(apiinstancetype.ControllerRevisionObjectKindLabel))
			Expect(createObj.Labels).To(HaveKey(apiinstancetype.ControllerRevisionObjectNameLabel))
			Expect(createObj.Labels).To(HaveKey(apiinstancetype.ControllerRevisionObjectUIDLabel))
			Expect(createObj.Labels).To(HaveKey(apiinstancetype.ControllerRevisionObjectVersionLabel))
			Expect(createObj).To(Equal(expectedControllerRevision))

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
		fakeInstancetypeClients = fakeclientset.NewSimpleClientset().InstancetypeV1beta1()

		instancetypeInformer, _ := testutils.NewFakeInformerFor(&instancetypev1beta1.VirtualMachineInstancetype{})
		instancetypeInformerStore = instancetypeInformer.GetStore()

		clusterInstancetypeInformer, _ := testutils.NewFakeInformerFor(&instancetypev1beta1.VirtualMachineClusterInstancetype{})
		clusterInstancetypeInformerStore = clusterInstancetypeInformer.GetStore()

		preferenceInformer, _ := testutils.NewFakeInformerFor(&instancetypev1beta1.VirtualMachinePreference{})
		preferenceInformerStore = preferenceInformer.GetStore()

		clusterPreferenceInformer, _ := testutils.NewFakeInformerFor(&instancetypev1beta1.VirtualMachineClusterPreference{})
		clusterPreferenceInformerStore = clusterPreferenceInformer.GetStore()

		controllerrevisionInformer, _ := testutils.NewFakeInformerFor(&appsv1.ControllerRevision{})
		controllerrevisionInformerStore = controllerrevisionInformer.GetStore()

		instancetypeMethods = &instancetype.InstancetypeMethods{
			InstancetypeStore:        instancetypeInformerStore,
			ClusterInstancetypeStore: clusterInstancetypeInformerStore,
			PreferenceStore:          preferenceInformerStore,
			ClusterPreferenceStore:   clusterInstancetypeInformerStore,
			ControllerRevisionStore:  controllerrevisionInformerStore,
			Clientset:                virtClient,
		}

		vm = kubecli.NewMinimalVM("testvm")
		vm.Spec.Template = &v1.VirtualMachineInstanceTemplateSpec{
			Spec: v1.VirtualMachineInstanceSpec{
				Domain: v1.DomainSpec{},
			},
		}
		vm.Namespace = k8sv1.NamespaceDefault
	})

	Context("Find and store instancetype", func() {

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
			var clusterInstancetype *instancetypev1beta1.VirtualMachineClusterInstancetype
			var fakeClusterInstancetypeClient instancetypeclientv1beta1.VirtualMachineClusterInstancetypeInterface

			BeforeEach(func() {
				fakeClusterInstancetypeClient = fakeInstancetypeClients.VirtualMachineClusterInstancetypes()
				virtClient.EXPECT().VirtualMachineClusterInstancetype().Return(fakeClusterInstancetypeClient).AnyTimes()

				clusterInstancetype = &instancetypev1beta1.VirtualMachineClusterInstancetype{
					TypeMeta: metav1.TypeMeta{
						Kind:       "VirtualMachineClusterInstancetype",
						APIVersion: instancetypev1beta1.SchemeGroupVersion.String(),
					},
					ObjectMeta: metav1.ObjectMeta{
						Name:       "test-cluster-instancetype",
						UID:        resourceUID,
						Generation: resourceGeneration,
					},
					Spec: instancetypev1beta1.VirtualMachineInstancetypeSpec{
						CPU: instancetypev1beta1.CPUInstancetype{
							Guest: uint32(2),
						},
						Memory: instancetypev1beta1.MemoryInstancetype{
							Guest: resource.MustParse("128Mi"),
						},
					},
				}

				_, err := virtClient.VirtualMachineClusterInstancetype().Create(context.Background(), clusterInstancetype, metav1.CreateOptions{})
				Expect(err).ToNot(HaveOccurred())

				err = clusterInstancetypeInformerStore.Add(clusterInstancetype)
				Expect(err).ToNot(HaveOccurred())

				vm.Spec.Instancetype = &v1.InstancetypeMatcher{
					Name: clusterInstancetype.Name,
					Kind: apiinstancetype.ClusterSingularResourceName,
				}
			})

			It("returns expected instancetype", func() {
				instancetypeSpec, err := instancetypeMethods.FindInstancetypeSpec(vm)
				Expect(err).ToNot(HaveOccurred())
				Expect(*instancetypeSpec).To(Equal(clusterInstancetype.Spec))
			})

			It("uses client when instancetype not found within informer", func() {
				err := clusterInstancetypeInformerStore.Delete(clusterInstancetype)
				Expect(err).ToNot(HaveOccurred())

				instancetypeSpec, err := instancetypeMethods.FindInstancetypeSpec(vm)
				Expect(err).ToNot(HaveOccurred())
				Expect(*instancetypeSpec).To(Equal(clusterInstancetype.Spec))
			})

			It("returns expected instancetype using only the client", func() {
				instancetypeMethods = &instancetype.InstancetypeMethods{Clientset: virtClient}
				instancetypeSpec, err := instancetypeMethods.FindInstancetypeSpec(vm)
				Expect(err).ToNot(HaveOccurred())
				Expect(*instancetypeSpec).To(Equal(clusterInstancetype.Spec))
			})

			It("find fails when instancetype does not exist", func() {
				vm.Spec.Instancetype.Name = nonExistingResourceName
				_, err := instancetypeMethods.FindInstancetypeSpec(vm)
				Expect(err).To(HaveOccurred())
				Expect(errors.IsNotFound(err)).To(BeTrue())
			})

			It("store VirtualMachineClusterInstancetype ControllerRevision", func() {
				clusterInstancetypeControllerRevision, err := instancetype.CreateControllerRevision(vm, clusterInstancetype)
				Expect(err).ToNot(HaveOccurred())

				expectedRevisionNamePatch, err := instancetype.GenerateRevisionNamePatch(clusterInstancetypeControllerRevision, nil)
				Expect(err).ToNot(HaveOccurred())

				vmInterface.EXPECT().Patch(context.Background(), vm.Name, types.JSONPatchType, expectedRevisionNamePatch, &metav1.PatchOptions{})

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
				vm.Spec.Instancetype.Name = nonExistingResourceName
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

				vmInterface.EXPECT().Patch(context.Background(), vm.Name, types.JSONPatchType, expectedRevisionNamePatch, &metav1.PatchOptions{})

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

			It("find successfully decodes v1alpha1 VirtualMachineInstancetypeSpecRevision ControllerRevision without APIVersion set - bug #9261", func() {
				clusterInstancetype.Spec.CPU = instancetypev1beta1.CPUInstancetype{
					Guest: uint32(2),
					// Set the following values to be compatible with objects converted from v1alpha1
					Model:                 pointer.String(""),
					DedicatedCPUPlacement: pointer.Bool(false),
					IsolateEmulatorThread: pointer.Bool(false),
				}

				specData, err := json.Marshal(clusterInstancetype.Spec)
				Expect(err).ToNot(HaveOccurred())

				// Do not set APIVersion as part of VirtualMachineInstancetypeSpecRevision in order to trigger bug #9261
				specRevision := instancetypev1alpha1.VirtualMachineInstancetypeSpecRevision{
					Spec: specData,
				}
				specRevisionData, err := json.Marshal(specRevision)
				Expect(err).ToNot(HaveOccurred())

				controllerRevision := &appsv1.ControllerRevision{
					ObjectMeta: metav1.ObjectMeta{
						Name:            "crName",
						Namespace:       vm.Namespace,
						OwnerReferences: []metav1.OwnerReference{*metav1.NewControllerRef(vm, v1.VirtualMachineGroupVersionKind)},
					},
					Data: runtime.RawExtension{
						Raw: specRevisionData,
					},
				}

				_, err = virtClient.AppsV1().ControllerRevisions(vm.Namespace).Create(context.Background(), controllerRevision, metav1.CreateOptions{})
				Expect(err).ToNot(HaveOccurred())

				vm.Spec.Instancetype = &v1.InstancetypeMatcher{
					Name:         clusterInstancetype.Name,
					RevisionName: controllerRevision.Name,
					Kind:         apiinstancetype.ClusterSingularResourceName,
				}

				foundInstancetypeSpec, err := instancetypeMethods.FindInstancetypeSpec(vm)
				Expect(err).ToNot(HaveOccurred())
				Expect(*foundInstancetypeSpec).To(Equal(clusterInstancetype.Spec))
			})
		})

		Context("Using namespaced Instancetype", func() {
			var fakeInstancetype *instancetypev1beta1.VirtualMachineInstancetype
			var fakeInstancetypeClient instancetypeclientv1beta1.VirtualMachineInstancetypeInterface

			BeforeEach(func() {
				fakeInstancetypeClient = fakeInstancetypeClients.VirtualMachineInstancetypes(vm.Namespace)
				virtClient.EXPECT().VirtualMachineInstancetype(gomock.Any()).Return(fakeInstancetypeClient).AnyTimes()

				fakeInstancetype = &instancetypev1beta1.VirtualMachineInstancetype{
					TypeMeta: metav1.TypeMeta{
						Kind:       "VirtualMachineInstancetype",
						APIVersion: instancetypev1beta1.SchemeGroupVersion.String(),
					},
					ObjectMeta: metav1.ObjectMeta{
						Name:       "test-instancetype",
						Namespace:  vm.Namespace,
						UID:        resourceUID,
						Generation: resourceGeneration,
					},
					Spec: instancetypev1beta1.VirtualMachineInstancetypeSpec{
						CPU: instancetypev1beta1.CPUInstancetype{
							Guest: uint32(2),
						},
						Memory: instancetypev1beta1.MemoryInstancetype{
							Guest: resource.MustParse("128Mi"),
						},
					},
				}

				_, err := virtClient.VirtualMachineInstancetype(vm.Namespace).Create(context.Background(), fakeInstancetype, metav1.CreateOptions{})
				Expect(err).ToNot(HaveOccurred())

				err = instancetypeInformerStore.Add(fakeInstancetype)
				Expect(err).ToNot(HaveOccurred())

				vm.Spec.Instancetype = &v1.InstancetypeMatcher{
					Name: fakeInstancetype.Name,
					Kind: apiinstancetype.SingularResourceName,
				}
			})

			It("find returns expected instancetype", func() {
				instancetypeSpec, err := instancetypeMethods.FindInstancetypeSpec(vm)
				Expect(err).ToNot(HaveOccurred())
				Expect(*instancetypeSpec).To(Equal(fakeInstancetype.Spec))
			})

			It("uses client when instancetype not found within informer", func() {
				err := clusterInstancetypeInformerStore.Delete(fakeInstancetype)
				Expect(err).ToNot(HaveOccurred())
				instancetypeSpec, err := instancetypeMethods.FindInstancetypeSpec(vm)
				Expect(err).ToNot(HaveOccurred())
				Expect(*instancetypeSpec).To(Equal(fakeInstancetype.Spec))
			})

			It("returns expected instancetype using only the client", func() {
				instancetypeMethods = &instancetype.InstancetypeMethods{Clientset: virtClient}
				instancetypeSpec, err := instancetypeMethods.FindInstancetypeSpec(vm)
				Expect(err).ToNot(HaveOccurred())
				Expect(*instancetypeSpec).To(Equal(fakeInstancetype.Spec))
			})

			It("find fails when instancetype does not exist", func() {
				vm.Spec.Instancetype.Name = nonExistingResourceName
				_, err := instancetypeMethods.FindInstancetypeSpec(vm)
				Expect(err).To(HaveOccurred())
				Expect(errors.IsNotFound(err)).To(BeTrue())
			})

			It("store VirtualMachineInstancetype ControllerRevision", func() {
				instancetypeControllerRevision, err := instancetype.CreateControllerRevision(vm, fakeInstancetype)
				Expect(err).ToNot(HaveOccurred())

				expectedRevisionNamePatch, err := instancetype.GenerateRevisionNamePatch(instancetypeControllerRevision, nil)
				Expect(err).ToNot(HaveOccurred())

				vmInterface.EXPECT().Patch(context.Background(), vm.Name, types.JSONPatchType, expectedRevisionNamePatch, &metav1.PatchOptions{})

				expectControllerRevisionCreation(instancetypeControllerRevision)

				Expect(instancetypeMethods.StoreControllerRevisions(vm)).To(Succeed())
				Expect(vm.Spec.Instancetype.RevisionName).To(Equal(instancetypeControllerRevision.Name))
			})

			It("store fails when instancetype does not exist", func() {
				vm.Spec.Instancetype.Name = nonExistingResourceName
				err := instancetypeMethods.StoreControllerRevisions(vm)
				Expect(err).To(HaveOccurred())
				Expect(errors.IsNotFound(err)).To(BeTrue())
			})

			It("store returns a nil revision when RevisionName already populated", func() {
				instancetypeControllerRevision, err := instancetype.CreateControllerRevision(vm, fakeInstancetype)
				Expect(err).ToNot(HaveOccurred())

				_, err = virtClient.AppsV1().ControllerRevisions(vm.Namespace).Create(context.Background(), instancetypeControllerRevision, metav1.CreateOptions{})
				Expect(err).ToNot(HaveOccurred())

				vm.Spec.Instancetype = &v1.InstancetypeMatcher{
					Name:         fakeInstancetype.Name,
					RevisionName: instancetypeControllerRevision.Name,
					Kind:         apiinstancetype.SingularResourceName,
				}

				Expect(instancetypeMethods.StoreControllerRevisions(vm)).To(Succeed())
				Expect(vm.Spec.Instancetype.RevisionName).To(Equal(instancetypeControllerRevision.Name))
			})

			It("store ControllerRevision succeeds if a revision exists with expected data", func() {
				instancetypeControllerRevision, err := instancetype.CreateControllerRevision(vm, fakeInstancetype)
				Expect(err).ToNot(HaveOccurred())

				_, err = virtClient.AppsV1().ControllerRevisions(vm.Namespace).Create(context.Background(), instancetypeControllerRevision, metav1.CreateOptions{})
				Expect(err).ToNot(HaveOccurred())

				expectedRevisionNamePatch, err := instancetype.GenerateRevisionNamePatch(instancetypeControllerRevision, nil)
				Expect(err).ToNot(HaveOccurred())

				vmInterface.EXPECT().Patch(context.Background(), vm.Name, types.JSONPatchType, expectedRevisionNamePatch, &metav1.PatchOptions{})

				Expect(instancetypeMethods.StoreControllerRevisions(vm)).To(Succeed())
				Expect(vm.Spec.Instancetype.RevisionName).To(Equal(instancetypeControllerRevision.Name))

			})

			It("store ControllerRevision fails if a revision exists with unexpected data", func() {
				unexpectedInstancetype := fakeInstancetype.DeepCopy()
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

			It("find successfully decodes v1alpha1 VirtualMachineInstancetypeSpecRevision ControllerRevision without APIVersion set - bug #9261", func() {
				fakeInstancetype.Spec.CPU = instancetypev1beta1.CPUInstancetype{
					Guest: uint32(2),
					// Set the following values to be compatible with objects converted from v1alpha1
					Model:                 pointer.String(""),
					DedicatedCPUPlacement: pointer.Bool(false),
					IsolateEmulatorThread: pointer.Bool(false),
				}

				specData, err := json.Marshal(fakeInstancetype.Spec)
				Expect(err).ToNot(HaveOccurred())

				// Do not set APIVersion as part of VirtualMachineInstancetypeSpecRevision in order to trigger bug #9261
				specRevision := instancetypev1alpha1.VirtualMachineInstancetypeSpecRevision{
					Spec: specData,
				}
				specRevisionData, err := json.Marshal(specRevision)
				Expect(err).ToNot(HaveOccurred())

				controllerRevision := &appsv1.ControllerRevision{
					ObjectMeta: metav1.ObjectMeta{
						Name:            "crName",
						Namespace:       vm.Namespace,
						OwnerReferences: []metav1.OwnerReference{*metav1.NewControllerRef(vm, v1.VirtualMachineGroupVersionKind)},
					},
					Data: runtime.RawExtension{
						Raw: specRevisionData,
					},
				}

				_, err = virtClient.AppsV1().ControllerRevisions(vm.Namespace).Create(context.Background(), controllerRevision, metav1.CreateOptions{})
				Expect(err).ToNot(HaveOccurred())

				vm.Spec.Instancetype = &v1.InstancetypeMatcher{
					Name:         fakeInstancetype.Name,
					RevisionName: controllerRevision.Name,
					Kind:         apiinstancetype.SingularResourceName,
				}

				foundInstancetypeSpec, err := instancetypeMethods.FindInstancetypeSpec(vm)
				Expect(err).ToNot(HaveOccurred())
				Expect(*foundInstancetypeSpec).To(Equal(fakeInstancetype.Spec))
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

	Context("Find and store preference", func() {

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
			var clusterPreference *instancetypev1beta1.VirtualMachineClusterPreference
			var fakeClusterPreferenceClient instancetypeclientv1beta1.VirtualMachineClusterPreferenceInterface

			BeforeEach(func() {
				fakeClusterPreferenceClient = fakeInstancetypeClients.VirtualMachineClusterPreferences()
				virtClient.EXPECT().VirtualMachineClusterPreference().Return(fakeClusterPreferenceClient).AnyTimes()

				preferredCPUTopology := instancetypev1beta1.PreferCores
				clusterPreference = &instancetypev1beta1.VirtualMachineClusterPreference{
					TypeMeta: metav1.TypeMeta{
						Kind:       "VirtualMachineClusterPreference",
						APIVersion: instancetypev1beta1.SchemeGroupVersion.String(),
					},
					ObjectMeta: metav1.ObjectMeta{
						Name:       "test-cluster-preference",
						UID:        resourceUID,
						Generation: resourceGeneration,
					},
					Spec: instancetypev1beta1.VirtualMachinePreferenceSpec{
						CPU: &instancetypev1beta1.CPUPreferences{
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
				s, err := instancetypeMethods.FindPreferenceSpec(vm)
				Expect(err).ToNot(HaveOccurred())
				Expect(*s).To(Equal(clusterPreference.Spec))
			})

			It("uses client when preference not found within informer", func() {
				err := clusterPreferenceInformerStore.Delete(clusterPreference)
				Expect(err).ToNot(HaveOccurred())
				preferenceSpec, err := instancetypeMethods.FindPreferenceSpec(vm)
				Expect(err).ToNot(HaveOccurred())
				Expect(*preferenceSpec).To(Equal(clusterPreference.Spec))
			})

			It("returns expected preference using only the client", func() {
				instancetypeMethods = &instancetype.InstancetypeMethods{Clientset: virtClient}
				preferenceSpec, err := instancetypeMethods.FindPreferenceSpec(vm)
				Expect(err).ToNot(HaveOccurred())
				Expect(*preferenceSpec).To(Equal(clusterPreference.Spec))
			})

			It("find fails when preference does not exist", func() {
				vm.Spec.Preference.Name = nonExistingResourceName
				_, err := instancetypeMethods.FindPreferenceSpec(vm)
				Expect(err).To(HaveOccurred())
				Expect(errors.IsNotFound(err)).To(BeTrue())
			})

			It("store VirtualMachineClusterPreference ControllerRevision", func() {
				clusterPreferenceControllerRevision, err := instancetype.CreateControllerRevision(vm, clusterPreference)
				Expect(err).ToNot(HaveOccurred())

				expectedRevisionNamePatch, err := instancetype.GenerateRevisionNamePatch(nil, clusterPreferenceControllerRevision)
				Expect(err).ToNot(HaveOccurred())

				vmInterface.EXPECT().Patch(context.Background(), vm.Name, types.JSONPatchType, expectedRevisionNamePatch, &metav1.PatchOptions{})

				expectControllerRevisionCreation(clusterPreferenceControllerRevision)

				Expect(instancetypeMethods.StoreControllerRevisions(vm)).To(Succeed())
				Expect(vm.Spec.Preference.RevisionName).To(Equal(clusterPreferenceControllerRevision.Name))
			})

			It("store fails when VirtualMachineClusterPreference doesn't exist", func() {
				vm.Spec.Preference.Name = nonExistingResourceName
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

				vmInterface.EXPECT().Patch(context.Background(), vm.Name, types.JSONPatchType, expectedRevisionNamePatch, &metav1.PatchOptions{})

				Expect(instancetypeMethods.StoreControllerRevisions(vm)).To(Succeed())
				Expect(vm.Spec.Preference.RevisionName).To(Equal(clusterPreferenceControllerRevision.Name))

			})

			It("store ControllerRevision fails if a revision exists with unexpected data", func() {
				unexpectedPreference := clusterPreference.DeepCopy()
				preferredCPUTopology := instancetypev1beta1.PreferThreads
				unexpectedPreference.Spec.CPU.PreferredCPUTopology = &preferredCPUTopology

				clusterPreferenceControllerRevision, err := instancetype.CreateControllerRevision(vm, unexpectedPreference)
				Expect(err).ToNot(HaveOccurred())

				_, err = virtClient.AppsV1().ControllerRevisions(vm.Namespace).Create(context.Background(), clusterPreferenceControllerRevision, metav1.CreateOptions{})
				Expect(err).ToNot(HaveOccurred())

				Expect(instancetypeMethods.StoreControllerRevisions(vm)).To(MatchError(ContainSubstring("found existing ControllerRevision with unexpected data")))
			})
		})

		Context("Using namespaced Preference", func() {
			var preference *instancetypev1beta1.VirtualMachinePreference
			var fakePreferenceClient instancetypeclientv1beta1.VirtualMachinePreferenceInterface

			BeforeEach(func() {
				fakePreferenceClient = fakeInstancetypeClients.VirtualMachinePreferences(vm.Namespace)
				virtClient.EXPECT().VirtualMachinePreference(gomock.Any()).Return(fakePreferenceClient).AnyTimes()

				preferredCPUTopology := instancetypev1beta1.PreferCores
				preference = &instancetypev1beta1.VirtualMachinePreference{
					TypeMeta: metav1.TypeMeta{
						Kind:       "VirtualMachinePreference",
						APIVersion: instancetypev1beta1.SchemeGroupVersion.String(),
					},
					ObjectMeta: metav1.ObjectMeta{
						Name:       "test-preference",
						Namespace:  vm.Namespace,
						UID:        resourceUID,
						Generation: resourceGeneration,
					},
					Spec: instancetypev1beta1.VirtualMachinePreferenceSpec{
						CPU: &instancetypev1beta1.CPUPreferences{
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
				preferenceSpec, err := instancetypeMethods.FindPreferenceSpec(vm)
				Expect(err).ToNot(HaveOccurred())
				Expect(*preferenceSpec).To(Equal(preference.Spec))
			})

			It("uses client when preference not found within informer", func() {
				err := preferenceInformerStore.Delete(preference)
				Expect(err).ToNot(HaveOccurred())

				preferenceSpec, err := instancetypeMethods.FindPreferenceSpec(vm)
				Expect(err).ToNot(HaveOccurred())
				Expect(*preferenceSpec).To(Equal(preference.Spec))
			})

			It("returns expected preference using only the client", func() {
				instancetypeMethods = &instancetype.InstancetypeMethods{Clientset: virtClient}
				preferenceSpec, err := instancetypeMethods.FindPreferenceSpec(vm)
				Expect(err).ToNot(HaveOccurred())
				Expect(*preferenceSpec).To(Equal(preference.Spec))
			})

			It("find fails when preference does not exist", func() {
				vm.Spec.Preference.Name = nonExistingResourceName
				_, err := instancetypeMethods.FindPreferenceSpec(vm)
				Expect(err).To(HaveOccurred())
				Expect(errors.IsNotFound(err)).To(BeTrue())
			})

			It("store VirtualMachinePreference ControllerRevision", func() {
				preferenceControllerRevision, err := instancetype.CreateControllerRevision(vm, preference)
				Expect(err).ToNot(HaveOccurred())

				expectedRevisionNamePatch, err := instancetype.GenerateRevisionNamePatch(nil, preferenceControllerRevision)
				Expect(err).ToNot(HaveOccurred())

				vmInterface.EXPECT().Patch(context.Background(), vm.Name, types.JSONPatchType, expectedRevisionNamePatch, &metav1.PatchOptions{})

				expectControllerRevisionCreation(preferenceControllerRevision)

				Expect(instancetypeMethods.StoreControllerRevisions(vm)).To(Succeed())
				Expect(vm.Spec.Preference.RevisionName).To(Equal(preferenceControllerRevision.Name))

			})

			It("store fails when VirtualMachinePreference doesn't exist", func() {
				vm.Spec.Preference.Name = nonExistingResourceName
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

				vmInterface.EXPECT().Patch(context.Background(), vm.Name, types.JSONPatchType, expectedRevisionNamePatch, &metav1.PatchOptions{})

				Expect(instancetypeMethods.StoreControllerRevisions(vm)).To(Succeed())
				Expect(vm.Spec.Preference.RevisionName).To(Equal(preferenceControllerRevision.Name))

			})

			It("store ControllerRevision fails if a revision exists with unexpected data", func() {
				unexpectedPreference := preference.DeepCopy()
				preferredCPUTopology := instancetypev1beta1.PreferThreads
				unexpectedPreference.Spec.CPU.PreferredCPUTopology = &preferredCPUTopology

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
			instancetypeSpec *instancetypev1beta1.VirtualMachineInstancetypeSpec
			preferenceSpec   *instancetypev1beta1.VirtualMachinePreferenceSpec
			field            *k8sfield.Path
		)

		BeforeEach(func() {
			vmi = api.NewMinimalVMI("testvmi")

			vmi.Spec = v1.VirtualMachineInstanceSpec{
				Domain: v1.DomainSpec{},
			}
			field = k8sfield.NewPath("spec", "template", "spec")
		})

		Context("instancetype.spec.NodeSelector", func() {
			It("should apply to VMI", func() {
				instancetypeSpec = &instancetypev1beta1.VirtualMachineInstancetypeSpec{
					NodeSelector: map[string]string{"key": "value"},
				}

				conflicts := instancetypeMethods.ApplyToVmi(field, instancetypeSpec, preferenceSpec, &vmi.Spec, &vmi.ObjectMeta)
				Expect(conflicts).To(BeEmpty())
				Expect(vmi.Spec.NodeSelector).To(Equal(instancetypeSpec.NodeSelector))
			})

			It("should be no-op if vmi.Spec.NodeSelector is already set but instancetype.NodeSelector is empty", func() {
				instancetypeSpec = &instancetypev1beta1.VirtualMachineInstancetypeSpec{}
				vmi.Spec.NodeSelector = map[string]string{"key": "value"}

				conflicts := instancetypeMethods.ApplyToVmi(field, instancetypeSpec, preferenceSpec, &vmi.Spec, &vmi.ObjectMeta)
				Expect(conflicts).To(BeEmpty())
				Expect(vmi.Spec.NodeSelector).To(Equal(map[string]string{"key": "value"}))
			})

			It("should return a conflict if vmi.Spec.NodeSelector is already set and instancetype.NodeSelector is defined", func() {
				instancetypeSpec = &instancetypev1beta1.VirtualMachineInstancetypeSpec{
					NodeSelector: map[string]string{"key": "value"},
				}
				vmi.Spec.NodeSelector = map[string]string{"key": "value"}

				conflicts := instancetypeMethods.ApplyToVmi(field, instancetypeSpec, preferenceSpec, &vmi.Spec, &vmi.ObjectMeta)
				Expect(conflicts).To(HaveLen(1))
				Expect(conflicts[0].String()).To(Equal("spec.template.spec.nodeSelector"))
			})
		})

		Context("instancetype.spec.SchedulerName", func() {
			It("should apply to VMI", func() {
				instancetypeSpec = &instancetypev1beta1.VirtualMachineInstancetypeSpec{
					SchedulerName: "ultra-scheduler",
				}

				conflicts := instancetypeMethods.ApplyToVmi(field, instancetypeSpec, preferenceSpec, &vmi.Spec, &vmi.ObjectMeta)
				Expect(conflicts).To(BeEmpty())
				Expect(vmi.Spec.SchedulerName).To(Equal(instancetypeSpec.SchedulerName))
			})

			It("should be no-op if vmi.Spec.SchedulerName is already set but instancetype.SchedulerName is empty", func() {
				instancetypeSpec = &instancetypev1beta1.VirtualMachineInstancetypeSpec{}
				vmi.Spec.SchedulerName = "super-fast-scheduler"

				conflicts := instancetypeMethods.ApplyToVmi(field, instancetypeSpec, preferenceSpec, &vmi.Spec, &vmi.ObjectMeta)
				Expect(conflicts).To(BeEmpty())
				Expect(vmi.Spec.SchedulerName).To(Equal("super-fast-scheduler"))
			})

			It("should return a conflict if vmi.Spec.SchedulerName is already set and instancetype.SchedulerName is defined", func() {
				instancetypeSpec = &instancetypev1beta1.VirtualMachineInstancetypeSpec{
					SchedulerName: "ultra-fast-scheduler",
				}
				vmi.Spec.SchedulerName = "slow-scheduler"

				conflicts := instancetypeMethods.ApplyToVmi(field, instancetypeSpec, preferenceSpec, &vmi.Spec, &vmi.ObjectMeta)
				Expect(conflicts).To(HaveLen(1))
				Expect(conflicts[0].String()).To(Equal("spec.template.spec.schedulerName"))
			})
		})

		Context("instancetype.spec.CPU and preference.spec.CPU", func() {

			BeforeEach(func() {
				instancetypeSpec = &instancetypev1beta1.VirtualMachineInstancetypeSpec{
					CPU: instancetypev1beta1.CPUInstancetype{
						Guest:                 uint32(2),
						Model:                 pointer.String("host-passthrough"),
						DedicatedCPUPlacement: pointer.Bool(true),
						IsolateEmulatorThread: pointer.Bool(true),
						NUMA: &v1.NUMA{
							GuestMappingPassthrough: &v1.NUMAGuestMappingPassthrough{},
						},
						Realtime: &v1.Realtime{
							Mask: "0-3,^1",
						},
					},
				}
				preferenceSpec = &instancetypev1beta1.VirtualMachinePreferenceSpec{
					CPU: &instancetypev1beta1.CPUPreferences{},
				}
			})

			It("should default to PreferSockets", func() {
				conflicts := instancetypeMethods.ApplyToVmi(field, instancetypeSpec, preferenceSpec, &vmi.Spec, &vmi.ObjectMeta)
				Expect(conflicts).To(BeEmpty())

				Expect(vmi.Spec.Domain.CPU.Sockets).To(Equal(instancetypeSpec.CPU.Guest))
				Expect(vmi.Spec.Domain.CPU.Cores).To(Equal(uint32(1)))
				Expect(vmi.Spec.Domain.CPU.Threads).To(Equal(uint32(1)))
				Expect(vmi.Spec.Domain.CPU.Model).To(Equal(*instancetypeSpec.CPU.Model))
				Expect(vmi.Spec.Domain.CPU.DedicatedCPUPlacement).To(Equal(*instancetypeSpec.CPU.DedicatedCPUPlacement))
				Expect(vmi.Spec.Domain.CPU.IsolateEmulatorThread).To(Equal(*instancetypeSpec.CPU.IsolateEmulatorThread))
				Expect(*vmi.Spec.Domain.CPU.NUMA).To(Equal(*instancetypeSpec.CPU.NUMA))
				Expect(*vmi.Spec.Domain.CPU.Realtime).To(Equal(*instancetypeSpec.CPU.Realtime))
			})

			It("should default to Sockets, when instancetype is used with PreferAny", func() {
				preferredCPUTopology := instancetypev1beta1.PreferAny
				preferenceSpec.CPU.PreferredCPUTopology = &preferredCPUTopology

				conflicts := instancetypeMethods.ApplyToVmi(field, instancetypeSpec, preferenceSpec, &vmi.Spec, &vmi.ObjectMeta)
				Expect(conflicts).To(BeEmpty())
				Expect(vmi.Spec.Domain.CPU.Sockets).To(Equal(instancetypeSpec.CPU.Guest))
				Expect(vmi.Spec.Domain.CPU.Cores).To(Equal(uint32(1)))
				Expect(vmi.Spec.Domain.CPU.Threads).To(Equal(uint32(1)))
			})

			It("should apply in full with PreferCores selected", func() {
				preferredCPUTopology := instancetypev1beta1.PreferCores
				preferenceSpec.CPU.PreferredCPUTopology = &preferredCPUTopology

				conflicts := instancetypeMethods.ApplyToVmi(field, instancetypeSpec, preferenceSpec, &vmi.Spec, &vmi.ObjectMeta)
				Expect(conflicts).To(BeEmpty())

				Expect(vmi.Spec.Domain.CPU.Sockets).To(Equal(uint32(1)))
				Expect(vmi.Spec.Domain.CPU.Cores).To(Equal(instancetypeSpec.CPU.Guest))
				Expect(vmi.Spec.Domain.CPU.Threads).To(Equal(uint32(1)))
				Expect(vmi.Spec.Domain.CPU.Model).To(Equal(*instancetypeSpec.CPU.Model))
				Expect(vmi.Spec.Domain.CPU.DedicatedCPUPlacement).To(Equal(*instancetypeSpec.CPU.DedicatedCPUPlacement))
				Expect(vmi.Spec.Domain.CPU.IsolateEmulatorThread).To(Equal(*instancetypeSpec.CPU.IsolateEmulatorThread))
				Expect(*vmi.Spec.Domain.CPU.NUMA).To(Equal(*instancetypeSpec.CPU.NUMA))
				Expect(*vmi.Spec.Domain.CPU.Realtime).To(Equal(*instancetypeSpec.CPU.Realtime))
			})

			It("should apply in full with PreferThreads selected", func() {
				preferredCPUTopology := instancetypev1beta1.PreferThreads
				preferenceSpec.CPU.PreferredCPUTopology = &preferredCPUTopology

				conflicts := instancetypeMethods.ApplyToVmi(field, instancetypeSpec, preferenceSpec, &vmi.Spec, &vmi.ObjectMeta)
				Expect(conflicts).To(BeEmpty())

				Expect(vmi.Spec.Domain.CPU.Sockets).To(Equal(uint32(1)))
				Expect(vmi.Spec.Domain.CPU.Cores).To(Equal(uint32(1)))
				Expect(vmi.Spec.Domain.CPU.Threads).To(Equal(instancetypeSpec.CPU.Guest))
				Expect(vmi.Spec.Domain.CPU.Model).To(Equal(*instancetypeSpec.CPU.Model))
				Expect(vmi.Spec.Domain.CPU.DedicatedCPUPlacement).To(Equal(*instancetypeSpec.CPU.DedicatedCPUPlacement))
				Expect(vmi.Spec.Domain.CPU.IsolateEmulatorThread).To(Equal(*instancetypeSpec.CPU.IsolateEmulatorThread))
				Expect(*vmi.Spec.Domain.CPU.NUMA).To(Equal(*instancetypeSpec.CPU.NUMA))
				Expect(*vmi.Spec.Domain.CPU.Realtime).To(Equal(*instancetypeSpec.CPU.Realtime))
			})

			It("should apply in full with PreferSockets selected", func() {
				preferredCPUTopology := instancetypev1beta1.PreferSockets
				preferenceSpec.CPU.PreferredCPUTopology = &preferredCPUTopology

				conflicts := instancetypeMethods.ApplyToVmi(field, instancetypeSpec, preferenceSpec, &vmi.Spec, &vmi.ObjectMeta)
				Expect(conflicts).To(BeEmpty())

				Expect(vmi.Spec.Domain.CPU.Sockets).To(Equal(instancetypeSpec.CPU.Guest))
				Expect(vmi.Spec.Domain.CPU.Cores).To(Equal(uint32(1)))
				Expect(vmi.Spec.Domain.CPU.Threads).To(Equal(uint32(1)))
				Expect(vmi.Spec.Domain.CPU.Model).To(Equal(*instancetypeSpec.CPU.Model))
				Expect(vmi.Spec.Domain.CPU.DedicatedCPUPlacement).To(Equal(*instancetypeSpec.CPU.DedicatedCPUPlacement))
				Expect(vmi.Spec.Domain.CPU.IsolateEmulatorThread).To(Equal(*instancetypeSpec.CPU.IsolateEmulatorThread))
				Expect(*vmi.Spec.Domain.CPU.NUMA).To(Equal(*instancetypeSpec.CPU.NUMA))
				Expect(*vmi.Spec.Domain.CPU.Realtime).To(Equal(*instancetypeSpec.CPU.Realtime))
			})

			DescribeTable("should spread between cores and sockets with PreferSpread selected", func(CPU, expectedCores, expectedSockets, spreadRatio int) {
				preferredCPUTopology := instancetypev1beta1.PreferSpread
				preferenceSpec.CPU.PreferredCPUTopology = &preferredCPUTopology
				preferenceSpec.PreferSpreadSocketToCoreRatio = uint32(spreadRatio)
				instancetypeSpec.CPU.Guest = uint32(CPU)

				conflicts := instancetypeMethods.ApplyToVmi(field, instancetypeSpec, preferenceSpec, &vmi.Spec, &vmi.ObjectMeta)
				Expect(conflicts).To(BeEmpty())

				Expect(vmi.Spec.Domain.CPU.Cores).To(Equal(uint32(expectedCores)))
				Expect(vmi.Spec.Domain.CPU.Sockets).To(Equal(uint32(expectedSockets)))
				Expect(vmi.Spec.Domain.CPU.Threads).To(Equal(uint32(1)))
			},
				Entry("with default PreferSpreadSocketToCoreRatio", 4, 2, 2, 0),
				Entry("with even PreferSpreadSocketToCoreRatio set", 8, 4, 2, 4),
				Entry("with odd PreferSpreadSocketToCoreRatio set", 9, 3, 3, 3),
			)

			It("should return a conflict if vmi.Spec.Domain.CPU already defined", func() {
				instancetypeSpec = &instancetypev1beta1.VirtualMachineInstancetypeSpec{
					CPU: instancetypev1beta1.CPUInstancetype{
						Guest: uint32(2),
					},
				}

				vmi.Spec.Domain.CPU = &v1.CPU{
					Cores:   4,
					Sockets: 1,
					Threads: 1,
				}

				conflicts := instancetypeMethods.ApplyToVmi(field, instancetypeSpec, preferenceSpec, &vmi.Spec, &vmi.ObjectMeta)
				Expect(conflicts).To(HaveLen(3))
				Expect(conflicts[0].String()).To(Equal("spec.template.spec.domain.cpu.sockets"))
				Expect(conflicts[1].String()).To(Equal("spec.template.spec.domain.cpu.cores"))
				Expect(conflicts[2].String()).To(Equal("spec.template.spec.domain.cpu.threads"))
			})

			It("should return a conflict if vmi.Spec.Domain.Resources.Requests[k8sv1.ResourceCPU] already defined", func() {
				instancetypeSpec = &instancetypev1beta1.VirtualMachineInstancetypeSpec{
					CPU: instancetypev1beta1.CPUInstancetype{
						Guest: uint32(2),
					},
				}

				vmi.Spec.Domain.Resources = v1.ResourceRequirements{
					Requests: k8sv1.ResourceList{
						k8sv1.ResourceCPU: resource.MustParse("1"),
					},
				}

				conflicts := instancetypeMethods.ApplyToVmi(field, instancetypeSpec, preferenceSpec, &vmi.Spec, &vmi.ObjectMeta)
				Expect(conflicts).To(HaveLen(1))
				Expect(conflicts[0].String()).To(Equal("spec.template.spec.domain.resources.requests.cpu"))
			})

			It("should return a conflict if vmi.Spec.Domain.Resources.Limits[k8sv1.ResourceCPU] already defined", func() {
				instancetypeSpec = &instancetypev1beta1.VirtualMachineInstancetypeSpec{
					CPU: instancetypev1beta1.CPUInstancetype{
						Guest: uint32(2),
					},
				}

				vmi.Spec.Domain.Resources = v1.ResourceRequirements{
					Limits: k8sv1.ResourceList{
						k8sv1.ResourceCPU: resource.MustParse("1"),
					},
				}

				conflicts := instancetypeMethods.ApplyToVmi(field, instancetypeSpec, preferenceSpec, &vmi.Spec, &vmi.ObjectMeta)
				Expect(conflicts).To(HaveLen(1))
				Expect(conflicts[0].String()).To(Equal("spec.template.spec.domain.resources.limits.cpu"))
			})

			It("should apply PreferredCPUFeatures", func() {
				preferenceSpec = &instancetypev1beta1.VirtualMachinePreferenceSpec{
					CPU: &instancetypev1beta1.CPUPreferences{
						PreferredCPUFeatures: []v1.CPUFeature{
							{
								Name:   "foo",
								Policy: "require",
							},
							{
								Name:   "bar",
								Policy: "force",
							},
						},
					},
				}
				vmi.Spec.Domain.CPU = &v1.CPU{
					Features: []v1.CPUFeature{
						{
							Name:   "bar",
							Policy: "optional",
						},
					},
				}
				Expect(instancetypeMethods.ApplyToVmi(field, instancetypeSpec, preferenceSpec, &vmi.Spec, &vmi.ObjectMeta)).To(BeEmpty())
				Expect(vmi.Spec.Domain.CPU.Features).To(HaveLen(2))
				Expect(vmi.Spec.Domain.CPU.Features).To(ContainElements([]v1.CPUFeature{
					{
						Name:   "foo",
						Policy: "require",
					},
					{
						Name:   "bar",
						Policy: "optional",
					},
				}))
			})
		})

		Context("instancetype.Spec.Memory", func() {
			BeforeEach(func() {
				instancetypeSpec = &instancetypev1beta1.VirtualMachineInstancetypeSpec{
					Memory: instancetypev1beta1.MemoryInstancetype{
						Guest: resource.MustParse("512M"),
						Hugepages: &v1.Hugepages{
							PageSize: "1Gi",
						},
					},
				}
			})

			It("should apply to VMI", func() {
				conflicts := instancetypeMethods.ApplyToVmi(field, instancetypeSpec, preferenceSpec, &vmi.Spec, &vmi.ObjectMeta)
				Expect(conflicts).To(BeEmpty())

				Expect(*vmi.Spec.Domain.Memory.Guest).To(Equal(instancetypeSpec.Memory.Guest))
				Expect(*vmi.Spec.Domain.Memory.Hugepages).To(Equal(*instancetypeSpec.Memory.Hugepages))
			})

			It("should apply memory overcommit correctly to VMI", func() {
				instancetypeSpec.Memory.Hugepages = nil
				instancetypeSpec.Memory.OvercommitPercent = 15

				expectedOverhead := int64(float32(instancetypeSpec.Memory.Guest.Value()) * (1 - float32(instancetypeSpec.Memory.OvercommitPercent)/100))
				Expect(expectedOverhead).ToNot(Equal(instancetypeSpec.Memory.Guest.Value()))

				conflicts := instancetypeMethods.ApplyToVmi(field, instancetypeSpec, preferenceSpec, &vmi.Spec, &vmi.ObjectMeta)
				Expect(conflicts).To(BeEmpty())
				memRequest := vmi.Spec.Domain.Resources.Requests[k8sv1.ResourceMemory]
				Expect(memRequest.Value()).To(Equal(expectedOverhead))
			})

			It("should detect memory conflict", func() {
				vmiMemGuest := resource.MustParse("512M")
				vmi.Spec.Domain.Memory = &v1.Memory{
					Guest: &vmiMemGuest,
				}

				conflicts := instancetypeMethods.ApplyToVmi(field, instancetypeSpec, preferenceSpec, &vmi.Spec, &vmi.ObjectMeta)
				Expect(conflicts).To(HaveLen(1))
				Expect(conflicts[0].String()).To(Equal("spec.template.spec.domain.memory"))
			})

			It("should return a conflict if vmi.Spec.Domain.Resources.Requests[k8sv1.ResourceMemory] already defined", func() {
				instancetypeSpec = &instancetypev1beta1.VirtualMachineInstancetypeSpec{
					Memory: instancetypev1beta1.MemoryInstancetype{
						Guest: resource.MustParse("512M"),
					},
				}

				vmi.Spec.Domain.Resources = v1.ResourceRequirements{
					Requests: k8sv1.ResourceList{
						k8sv1.ResourceMemory: resource.MustParse("128Mi"),
					},
				}

				conflicts := instancetypeMethods.ApplyToVmi(field, instancetypeSpec, preferenceSpec, &vmi.Spec, &vmi.ObjectMeta)
				Expect(conflicts).To(HaveLen(1))
				Expect(conflicts[0].String()).To(Equal("spec.template.spec.domain.resources.requests.memory"))
			})

			It("should return a conflict if vmi.Spec.Domain.Resources.Limits[k8sv1.ResourceMemory] already defined", func() {
				instancetypeSpec = &instancetypev1beta1.VirtualMachineInstancetypeSpec{
					Memory: instancetypev1beta1.MemoryInstancetype{
						Guest: resource.MustParse("512M"),
					},
				}

				vmi.Spec.Domain.Resources = v1.ResourceRequirements{
					Limits: k8sv1.ResourceList{
						k8sv1.ResourceMemory: resource.MustParse("128Mi"),
					},
				}

				conflicts := instancetypeMethods.ApplyToVmi(field, instancetypeSpec, preferenceSpec, &vmi.Spec, &vmi.ObjectMeta)
				Expect(conflicts).To(HaveLen(1))
				Expect(conflicts[0].String()).To(Equal("spec.template.spec.domain.resources.limits.memory"))
			})
		})
		Context("instancetype.Spec.ioThreadsPolicy", func() {

			var instancetypePolicy v1.IOThreadsPolicy

			BeforeEach(func() {
				instancetypePolicy = v1.IOThreadsPolicyShared
				instancetypeSpec = &instancetypev1beta1.VirtualMachineInstancetypeSpec{
					IOThreadsPolicy: &instancetypePolicy,
				}
			})

			It("should apply to VMI", func() {
				conflicts := instancetypeMethods.ApplyToVmi(field, instancetypeSpec, preferenceSpec, &vmi.Spec, &vmi.ObjectMeta)
				Expect(conflicts).To(BeEmpty())

				Expect(*vmi.Spec.Domain.IOThreadsPolicy).To(Equal(*instancetypeSpec.IOThreadsPolicy))
			})

			It("should detect IOThreadsPolicy conflict", func() {
				vmi.Spec.Domain.IOThreadsPolicy = &instancetypePolicy

				conflicts := instancetypeMethods.ApplyToVmi(field, instancetypeSpec, preferenceSpec, &vmi.Spec, &vmi.ObjectMeta)
				Expect(conflicts).To(HaveLen(1))
				Expect(conflicts[0].String()).To(Equal("spec.template.spec.domain.ioThreadsPolicy"))
			})
		})

		Context("instancetype.Spec.LaunchSecurity", func() {

			BeforeEach(func() {
				instancetypeSpec = &instancetypev1beta1.VirtualMachineInstancetypeSpec{
					LaunchSecurity: &v1.LaunchSecurity{
						SEV: &v1.SEV{},
					},
				}
			})

			It("should apply to VMI", func() {
				conflicts := instancetypeMethods.ApplyToVmi(field, instancetypeSpec, preferenceSpec, &vmi.Spec, &vmi.ObjectMeta)
				Expect(conflicts).To(BeEmpty())

				Expect(*vmi.Spec.Domain.LaunchSecurity).To(Equal(*instancetypeSpec.LaunchSecurity))
			})

			It("should detect LaunchSecurity conflict", func() {
				vmi.Spec.Domain.LaunchSecurity = &v1.LaunchSecurity{
					SEV: &v1.SEV{},
				}

				conflicts := instancetypeMethods.ApplyToVmi(field, instancetypeSpec, preferenceSpec, &vmi.Spec, &vmi.ObjectMeta)
				Expect(conflicts).To(HaveLen(1))
				Expect(conflicts[0].String()).To(Equal("spec.template.spec.domain.launchSecurity"))
			})
		})

		Context("instancetype.Spec.GPUs", func() {

			BeforeEach(func() {
				instancetypeSpec = &instancetypev1beta1.VirtualMachineInstancetypeSpec{
					GPUs: []v1.GPU{
						{
							Name:       "barfoo",
							DeviceName: "vendor.com/gpu_name",
						},
					},
				}
			})

			It("should apply to VMI", func() {
				conflicts := instancetypeMethods.ApplyToVmi(field, instancetypeSpec, preferenceSpec, &vmi.Spec, &vmi.ObjectMeta)
				Expect(conflicts).To(BeEmpty())

				Expect(vmi.Spec.Domain.Devices.GPUs).To(Equal(instancetypeSpec.GPUs))
			})

			It("should detect GPU conflict", func() {
				vmi.Spec.Domain.Devices.GPUs = []v1.GPU{
					{
						Name:       "foobar",
						DeviceName: "vendor.com/gpu_name",
					},
				}

				conflicts := instancetypeMethods.ApplyToVmi(field, instancetypeSpec, preferenceSpec, &vmi.Spec, &vmi.ObjectMeta)
				Expect(conflicts).To(HaveLen(1))
				Expect(conflicts[0].String()).To(Equal("spec.template.spec.domain.devices.gpus"))
			})
		})

		Context("instancetype.Spec.HostDevices", func() {

			BeforeEach(func() {
				instancetypeSpec = &instancetypev1beta1.VirtualMachineInstancetypeSpec{
					HostDevices: []v1.HostDevice{
						{
							Name:       "foobar",
							DeviceName: "vendor.com/device_name",
						},
					},
				}
			})

			It("should apply to VMI", func() {
				conflicts := instancetypeMethods.ApplyToVmi(field, instancetypeSpec, preferenceSpec, &vmi.Spec, &vmi.ObjectMeta)
				Expect(conflicts).To(BeEmpty())

				Expect(vmi.Spec.Domain.Devices.HostDevices).To(Equal(instancetypeSpec.HostDevices))
			})

			It("should detect HostDevice conflict", func() {
				vmi.Spec.Domain.Devices.HostDevices = []v1.HostDevice{
					{
						Name:       "foobar",
						DeviceName: "vendor.com/device_name",
					},
				}

				conflicts := instancetypeMethods.ApplyToVmi(field, instancetypeSpec, preferenceSpec, &vmi.Spec, &vmi.ObjectMeta)
				Expect(conflicts).To(HaveLen(1))
				Expect(conflicts[0].String()).To(Equal("spec.template.spec.domain.devices.hostDevices"))
			})
		})

		Context("Instancetype.Spec.Annotations", func() {

			var multipleAnnotations map[string]string

			BeforeEach(func() {
				multipleAnnotations = map[string]string{
					"annotation-1": "1",
					"annotation-2": "2",
				}

				instancetypeSpec = &instancetypev1beta1.VirtualMachineInstancetypeSpec{
					Annotations: make(map[string]string),
				}
			})

			It("should apply to VMI", func() {
				instancetypeSpec.Annotations = multipleAnnotations

				conflicts := instancetypeMethods.ApplyToVmi(field, instancetypeSpec, nil, &vmi.Spec, &vmi.ObjectMeta)
				Expect(conflicts).To(BeEmpty())
				Expect(vmi.Annotations).To(Equal(instancetypeSpec.Annotations))
			})

			It("should not detect conflict when annotation with the same value already exists", func() {
				instancetypeSpec.Annotations = multipleAnnotations
				vmi.Annotations = multipleAnnotations

				conflicts := instancetypeMethods.ApplyToVmi(field, instancetypeSpec, nil, &vmi.Spec, &vmi.ObjectMeta)
				Expect(conflicts).To(BeEmpty())
				Expect(vmi.Annotations).To(Equal(instancetypeSpec.Annotations))
			})

			It("should detect conflict when annotation with different value already exists", func() {
				instancetypeSpec.Annotations = multipleAnnotations
				vmi.Annotations = map[string]string{
					"annotation-1": "conflict",
				}

				conflicts := instancetypeMethods.ApplyToVmi(field, instancetypeSpec, nil, &vmi.Spec, &vmi.ObjectMeta)
				Expect(conflicts).To(HaveLen(1))
				Expect(conflicts[0].String()).To(Equal("annotations.annotation-1"))
			})
		})

		Context("Preference.Spec.Annotations", func() {

			var multipleAnnotations map[string]string

			BeforeEach(func() {
				multipleAnnotations = map[string]string{
					"annotation-1": "1",
					"annotation-2": "2",
				}

				preferenceSpec = &instancetypev1beta1.VirtualMachinePreferenceSpec{
					Annotations: make(map[string]string),
				}
			})

			It("should apply to VMI", func() {
				preferenceSpec.Annotations = multipleAnnotations

				conflicts := instancetypeMethods.ApplyToVmi(field, nil, preferenceSpec, &vmi.Spec, &vmi.ObjectMeta)
				Expect(conflicts).To(BeEmpty())
				Expect(vmi.Annotations).To(Equal(preferenceSpec.Annotations))
			})

			It("should not overwrite already existing values", func() {
				preferenceSpec.Annotations = multipleAnnotations
				vmiAnnotations := map[string]string{
					"annotation-1": "dont-overwrite",
					"annotation-2": "dont-overwrite",
					"annotation-3": "dont-overwrite",
				}
				vmi.Annotations = vmiAnnotations

				conflicts := instancetypeMethods.ApplyToVmi(field, nil, preferenceSpec, &vmi.Spec, &vmi.ObjectMeta)
				Expect(conflicts).To(BeEmpty())
				Expect(vmi.Annotations).To(HaveLen(3))
				Expect(vmi.Annotations).To(Equal(vmiAnnotations))
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
					{
						Cache:             v1.CacheWriteBack,
						IO:                v1.IONative,
						DedicatedIOThread: pointer.Bool(false),
						BlockSize:         userDefinedBlockSize,
						DiskDevice: v1.DiskDevice{
							Disk: &v1.DiskTarget{
								Bus: v1.DiskBusSCSI,
							},
						},
					},
					{
						DiskDevice: v1.DiskDevice{
							Disk: &v1.DiskTarget{},
						},
					},
					{
						DiskDevice: v1.DiskDevice{
							CDRom: &v1.CDRomTarget{
								Bus: v1.DiskBusSATA,
							},
						},
					},
					{
						DiskDevice: v1.DiskDevice{
							CDRom: &v1.CDRomTarget{},
						},
					},
					{
						DiskDevice: v1.DiskDevice{
							LUN: &v1.LunTarget{
								Bus: v1.DiskBusSATA,
							},
						},
					},
					{
						DiskDevice: v1.DiskDevice{
							LUN: &v1.LunTarget{},
						},
					},
				}
				vmi.Spec.Domain.Devices.Inputs = []v1.Input{
					{
						Bus:  "usb",
						Type: "tablet",
					},
					{},
				}
				vmi.Spec.Domain.Devices.Interfaces = []v1.Interface{
					{
						Name:  "primary",
						Model: "e1000",
					},
					{
						Name: "secondary",
					},
				}
				vmi.Spec.Domain.Devices.Sound = &v1.SoundDevice{}

				preferenceSpec = &instancetypev1beta1.VirtualMachinePreferenceSpec{
					Devices: &instancetypev1beta1.DevicePreferences{
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
						PreferredDiskCache:           v1.CacheWriteThrough,
						PreferredDiskIO:              v1.IONative,
						PreferredDiskBus:             v1.DiskBusVirtio,
						PreferredCdromBus:            v1.DiskBusSCSI,
						PreferredLunBus:              v1.DiskBusSATA,
						PreferredInputBus:            v1.InputBusVirtio,
						PreferredInputType:           v1.InputTypeTablet,
						PreferredInterfaceModel:      v1.VirtIO,
						PreferredSoundModel:          "ac97",
						PreferredRng:                 &v1.Rng{},
						PreferredTPM:                 &v1.TPMDevice{},
						PreferredInterfaceMasquerade: &v1.InterfaceMasquerade{},
					},
				}
			})

			Context("PreferredInterfaceMasquerade", func() {
				It("should be applied to interface on Pod network", func() {
					vmi.Spec.Networks = []v1.Network{{
						Name: vmi.Spec.Domain.Devices.Interfaces[0].Name,
						NetworkSource: v1.NetworkSource{
							Pod: &v1.PodNetwork{},
						},
					}}
					Expect(instancetypeMethods.ApplyToVmi(field, instancetypeSpec, preferenceSpec, &vmi.Spec, &vmi.ObjectMeta)).To(BeNil())
					Expect(vmi.Spec.Domain.Devices.Interfaces[0].Masquerade).ToNot(BeNil())
					Expect(vmi.Spec.Domain.Devices.Interfaces[1].Masquerade).To(BeNil())
				})
				It("should not be applied on interface that has another binding set", func() {
					vmi.Spec.Domain.Devices.Interfaces[0].SRIOV = &v1.InterfaceSRIOV{}
					Expect(instancetypeMethods.ApplyToVmi(field, instancetypeSpec, preferenceSpec, &vmi.Spec, &vmi.ObjectMeta)).To(BeNil())
					Expect(vmi.Spec.Domain.Devices.Interfaces[0].Masquerade).To(BeNil())
					Expect(vmi.Spec.Domain.Devices.Interfaces[0].SRIOV).ToNot(BeNil())
				})
				It("should not be applied on interface that is not on Pod network", func() {
					vmi.Spec.Networks = []v1.Network{{
						Name: vmi.Spec.Domain.Devices.Interfaces[0].Name,
					}}
					Expect(instancetypeMethods.ApplyToVmi(field, instancetypeSpec, preferenceSpec, &vmi.Spec, &vmi.ObjectMeta)).To(BeNil())
					Expect(vmi.Spec.Domain.Devices.Interfaces[0].Masquerade).To(BeNil())
				})
			})

			It("should apply to VMI", func() {
				conflicts := instancetypeMethods.ApplyToVmi(field, instancetypeSpec, preferenceSpec, &vmi.Spec, &vmi.ObjectMeta)
				Expect(conflicts).To(BeEmpty())

				Expect(*vmi.Spec.Domain.Devices.AutoattachGraphicsDevice).To(BeFalse())
				Expect(*vmi.Spec.Domain.Devices.AutoattachMemBalloon).To(BeFalse())
				Expect(*vmi.Spec.Domain.Devices.AutoattachInputDevice).To(BeTrue())
				Expect(vmi.Spec.Domain.Devices.Disks[0].Cache).To(Equal(v1.CacheWriteBack))
				Expect(vmi.Spec.Domain.Devices.Disks[0].IO).To(Equal(v1.IONative))
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

				conflicts := instancetypeMethods.ApplyToVmi(field, instancetypeSpec, preferenceSpec, &vmi.Spec, &vmi.ObjectMeta)
				Expect(conflicts).To(BeEmpty())

				Expect(vmi.Spec.Domain.Devices.Disks[1].DiskDevice.Disk.Bus).To(Equal(preferenceSpec.Devices.PreferredDiskBus))
			})

			It("[test_id:CNV-9817] Should ignore preference when a VMI disk have a DiskDevice defined", func() {
				diskTypeForTest := v1.DiskBusSCSI

				vmi.Spec.Domain.Devices.Disks[1].DiskDevice.Disk.Bus = diskTypeForTest
				conflicts := instancetypeMethods.ApplyToVmi(field, instancetypeSpec, preferenceSpec, &vmi.Spec, &vmi.ObjectMeta)
				Expect(conflicts).To(BeEmpty())

				Expect(vmi.Spec.Domain.Devices.Disks[1].DiskDevice.Disk.Bus).To(Equal(diskTypeForTest))
			})
		})

		Context("Preference.Features", func() {

			BeforeEach(func() {
				spinLockRetries := uint32(32)
				preferenceSpec = &instancetypev1beta1.VirtualMachinePreferenceSpec{
					Features: &instancetypev1beta1.FeaturePreferences{
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
				conflicts := instancetypeMethods.ApplyToVmi(field, instancetypeSpec, preferenceSpec, &vmi.Spec, &vmi.ObjectMeta)
				Expect(conflicts).To(BeEmpty())

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

				conflicts := instancetypeMethods.ApplyToVmi(field, instancetypeSpec, preferenceSpec, &vmi.Spec, &vmi.ObjectMeta)
				Expect(conflicts).To(BeEmpty())

				Expect(*vmi.Spec.Domain.Features.Hyperv.EVMCS.Enabled).To(BeFalse())
			})
		})

		Context("Preference.Firmware", func() {

			It("should apply BIOS preferences full to VMI", func() {
				preferenceSpec = &instancetypev1beta1.VirtualMachinePreferenceSpec{
					Firmware: &instancetypev1beta1.FirmwarePreferences{
						PreferredUseBios:       pointer.Bool(true),
						PreferredUseBiosSerial: pointer.Bool(true),
						PreferredUseEfi:        pointer.Bool(false),
						PreferredUseSecureBoot: pointer.Bool(false),
					},
				}

				conflicts := instancetypeMethods.ApplyToVmi(field, instancetypeSpec, preferenceSpec, &vmi.Spec, &vmi.ObjectMeta)
				Expect(conflicts).To(BeEmpty())

				Expect(*vmi.Spec.Domain.Firmware.Bootloader.BIOS.UseSerial).To(Equal(*preferenceSpec.Firmware.PreferredUseBiosSerial))
			})

			It("should apply SecureBoot preferences full to VMI", func() {
				preferenceSpec = &instancetypev1beta1.VirtualMachinePreferenceSpec{
					Firmware: &instancetypev1beta1.FirmwarePreferences{
						PreferredUseBios:       pointer.Bool(false),
						PreferredUseBiosSerial: pointer.Bool(false),
						PreferredUseEfi:        pointer.Bool(true),
						PreferredUseSecureBoot: pointer.Bool(true),
					},
				}

				conflicts := instancetypeMethods.ApplyToVmi(field, instancetypeSpec, preferenceSpec, &vmi.Spec, &vmi.ObjectMeta)
				Expect(conflicts).To(BeEmpty())

				Expect(*vmi.Spec.Domain.Firmware.Bootloader.EFI.SecureBoot).To(Equal(*preferenceSpec.Firmware.PreferredUseSecureBoot))
			})

			It("should not overwrite user defined Bootloader.BIOS with PreferredUseEfi - bug #10313", func() {
				preferenceSpec = &instancetypev1beta1.VirtualMachinePreferenceSpec{
					Firmware: &instancetypev1beta1.FirmwarePreferences{
						PreferredUseEfi:        pointer.Bool(true),
						PreferredUseSecureBoot: pointer.Bool(true),
					},
				}
				vmi.Spec.Domain.Firmware = &v1.Firmware{
					Bootloader: &v1.Bootloader{
						BIOS: &v1.BIOS{
							UseSerial: pointer.Bool(false),
						},
					},
				}
				Expect(instancetypeMethods.ApplyToVmi(field, instancetypeSpec, preferenceSpec, &vmi.Spec, &vmi.ObjectMeta)).To(BeEmpty())
				Expect(vmi.Spec.Domain.Firmware.Bootloader.EFI).To(BeNil())
				Expect(*vmi.Spec.Domain.Firmware.Bootloader.BIOS.UseSerial).To(BeFalse())
			})

			It("should not overwrite user defined value with PreferredUseBiosSerial - bug #10313", func() {
				preferenceSpec = &instancetypev1beta1.VirtualMachinePreferenceSpec{
					Firmware: &instancetypev1beta1.FirmwarePreferences{
						PreferredUseBios:       pointer.Bool(true),
						PreferredUseBiosSerial: pointer.Bool(true),
					},
				}
				vmi.Spec.Domain.Firmware = &v1.Firmware{
					Bootloader: &v1.Bootloader{
						BIOS: &v1.BIOS{
							UseSerial: pointer.Bool(false),
						},
					},
				}
				Expect(instancetypeMethods.ApplyToVmi(field, instancetypeSpec, preferenceSpec, &vmi.Spec, &vmi.ObjectMeta)).To(BeEmpty())
				Expect(*vmi.Spec.Domain.Firmware.Bootloader.BIOS.UseSerial).To(BeFalse())
			})

			It("should not overwrite user defined Bootloader.EFI with PreferredUseBios - bug #10313", func() {
				preferenceSpec = &instancetypev1beta1.VirtualMachinePreferenceSpec{
					Firmware: &instancetypev1beta1.FirmwarePreferences{
						PreferredUseBios:       pointer.Bool(true),
						PreferredUseBiosSerial: pointer.Bool(true),
					},
				}
				vmi.Spec.Domain.Firmware = &v1.Firmware{
					Bootloader: &v1.Bootloader{
						EFI: &v1.EFI{
							SecureBoot: pointer.Bool(false),
						},
					},
				}
				Expect(instancetypeMethods.ApplyToVmi(field, instancetypeSpec, preferenceSpec, &vmi.Spec, &vmi.ObjectMeta)).To(BeEmpty())
				Expect(vmi.Spec.Domain.Firmware.Bootloader.BIOS).To(BeNil())
				Expect(*vmi.Spec.Domain.Firmware.Bootloader.EFI.SecureBoot).To(BeFalse())
			})

			It("should not overwrite user defined value with PreferredUseSecureBoot - bug #10313", func() {
				preferenceSpec = &instancetypev1beta1.VirtualMachinePreferenceSpec{
					Firmware: &instancetypev1beta1.FirmwarePreferences{
						PreferredUseEfi:        pointer.Bool(true),
						PreferredUseSecureBoot: pointer.Bool(true),
					},
				}
				vmi.Spec.Domain.Firmware = &v1.Firmware{
					Bootloader: &v1.Bootloader{
						EFI: &v1.EFI{
							SecureBoot: pointer.Bool(false),
						},
					},
				}
				Expect(instancetypeMethods.ApplyToVmi(field, instancetypeSpec, preferenceSpec, &vmi.Spec, &vmi.ObjectMeta)).To(BeEmpty())
				Expect(*vmi.Spec.Domain.Firmware.Bootloader.EFI.SecureBoot).To(BeFalse())
			})
		})

		Context("Preference.Machine", func() {

			It("should apply to VMI", func() {
				preferenceSpec = &instancetypev1beta1.VirtualMachinePreferenceSpec{
					Machine: &instancetypev1beta1.MachinePreferences{
						PreferredMachineType: "q35-rhel-8.0",
					},
				}

				conflicts := instancetypeMethods.ApplyToVmi(field, instancetypeSpec, preferenceSpec, &vmi.Spec, &vmi.ObjectMeta)
				Expect(conflicts).To(BeEmpty())

				Expect(vmi.Spec.Domain.Machine.Type).To(Equal(preferenceSpec.Machine.PreferredMachineType))
			})
		})
		Context("Preference.Clock", func() {

			It("should apply to VMI", func() {
				preferenceSpec = &instancetypev1beta1.VirtualMachinePreferenceSpec{
					Clock: &instancetypev1beta1.ClockPreferences{
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

				conflicts := instancetypeMethods.ApplyToVmi(field, instancetypeSpec, preferenceSpec, &vmi.Spec, &vmi.ObjectMeta)
				Expect(conflicts).To(BeEmpty())

				Expect(vmi.Spec.Domain.Clock.ClockOffset).To(Equal(*preferenceSpec.Clock.PreferredClockOffset))
				Expect(*vmi.Spec.Domain.Clock.Timer).To(Equal(*preferenceSpec.Clock.PreferredTimer))
			})
		})

		Context("Preference.PreferredSubdomain", func() {
			It("should apply to VMI", func() {
				preferenceSpec = &instancetypev1beta1.VirtualMachinePreferenceSpec{
					PreferredSubdomain: pointer.String("kubevirt.io"),
				}
				Expect(instancetypeMethods.ApplyToVmi(field, instancetypeSpec, preferenceSpec, &vmi.Spec, &vmi.ObjectMeta)).To(BeEmpty())
				Expect(vmi.Spec.Subdomain).To(Equal(*preferenceSpec.PreferredSubdomain))
			})

			It("should not overwrite user defined value", func() {
				const userDefinedValue = "foo.com"
				vmi.Spec.Subdomain = userDefinedValue
				preferenceSpec = &instancetypev1beta1.VirtualMachinePreferenceSpec{
					PreferredSubdomain: pointer.String("kubevirt.io"),
				}
				Expect(instancetypeMethods.ApplyToVmi(field, instancetypeSpec, preferenceSpec, &vmi.Spec, &vmi.ObjectMeta)).To(BeEmpty())
				Expect(vmi.Spec.Subdomain).To(Equal(userDefinedValue))
			})
		})

		Context("Preference.PreferredTerminationGracePeriodSeconds", func() {
			It("should apply to VMI", func() {
				preferenceSpec = &instancetypev1beta1.VirtualMachinePreferenceSpec{
					PreferredTerminationGracePeriodSeconds: pointer.Int64(180),
				}
				Expect(instancetypeMethods.ApplyToVmi(field, instancetypeSpec, preferenceSpec, &vmi.Spec, &vmi.ObjectMeta)).To(BeEmpty())
				Expect(*vmi.Spec.TerminationGracePeriodSeconds).To(Equal(*preferenceSpec.PreferredTerminationGracePeriodSeconds))
			})

			It("should not overwrite user defined value", func() {
				const userDefinedValue = int64(100)
				vmi.Spec.TerminationGracePeriodSeconds = pointer.Int64(userDefinedValue)
				preferenceSpec = &instancetypev1beta1.VirtualMachinePreferenceSpec{
					PreferredTerminationGracePeriodSeconds: pointer.Int64(180),
				}
				Expect(instancetypeMethods.ApplyToVmi(field, instancetypeSpec, preferenceSpec, &vmi.Spec, &vmi.ObjectMeta)).To(BeEmpty())
				Expect(*vmi.Spec.TerminationGracePeriodSeconds).To(Equal(userDefinedValue))
			})
		})
	})

	Context("preference requirements check", func() {
		var (
			preferCores   = instancetypev1beta1.PreferCores
			preferSockets = instancetypev1beta1.PreferSockets
			preferThreads = instancetypev1beta1.PreferThreads
			preferSpread  = instancetypev1beta1.PreferSpread
			preferAny     = instancetypev1beta1.PreferAny
		)

		DescribeTable("should pass when sufficient resources are provided", func(instancetypeSpec *instancetypev1beta1.VirtualMachineInstancetypeSpec, preferenceSpec *instancetypev1beta1.VirtualMachinePreferenceSpec, vmiSpec *v1.VirtualMachineInstanceSpec) {
			path, err := instancetypeMethods.CheckPreferenceRequirements(instancetypeSpec, preferenceSpec, vmiSpec)
			Expect(path).To(BeNil())
			Expect(err).ToNot(HaveOccurred())
		},
			Entry("by an instance type for vCPUs",
				&instancetypev1beta1.VirtualMachineInstancetypeSpec{
					CPU: instancetypev1beta1.CPUInstancetype{
						Guest: uint32(2),
					},
				},
				&instancetypev1beta1.VirtualMachinePreferenceSpec{
					Requirements: &instancetypev1beta1.PreferenceRequirements{
						CPU: &instancetypev1beta1.CPUPreferenceRequirement{
							Guest: uint32(2),
						},
					},
				},
				nil,
			),
			Entry("by an instance type for Memory",
				&instancetypev1beta1.VirtualMachineInstancetypeSpec{
					Memory: instancetypev1beta1.MemoryInstancetype{
						Guest: resource.MustParse("1Gi"),
					},
				},
				&instancetypev1beta1.VirtualMachinePreferenceSpec{
					Requirements: &instancetypev1beta1.PreferenceRequirements{
						Memory: &instancetypev1beta1.MemoryPreferenceRequirement{
							Guest: resource.MustParse("1Gi"),
						},
					},
				},
				nil,
			),
			Entry("by a VM for vCPUs using PreferSockets (default)",
				nil,
				&instancetypev1beta1.VirtualMachinePreferenceSpec{
					CPU: &instancetypev1beta1.CPUPreferences{
						PreferredCPUTopology: &preferSockets,
					},
					Requirements: &instancetypev1beta1.PreferenceRequirements{
						CPU: &instancetypev1beta1.CPUPreferenceRequirement{
							Guest: uint32(2),
						},
					},
				},
				&v1.VirtualMachineInstanceSpec{
					Domain: v1.DomainSpec{
						CPU: &v1.CPU{
							Sockets: uint32(2),
						},
					},
				},
			),
			Entry("by a VM for vCPUs using PreferCores",
				nil,
				&instancetypev1beta1.VirtualMachinePreferenceSpec{
					CPU: &instancetypev1beta1.CPUPreferences{
						PreferredCPUTopology: &preferCores,
					},
					Requirements: &instancetypev1beta1.PreferenceRequirements{
						CPU: &instancetypev1beta1.CPUPreferenceRequirement{
							Guest: uint32(2),
						},
					},
				},
				&v1.VirtualMachineInstanceSpec{
					Domain: v1.DomainSpec{
						CPU: &v1.CPU{
							Cores: uint32(2),
						},
					},
				},
			),
			Entry("by a VM for vCPUs using PreferThreads",
				nil,
				&instancetypev1beta1.VirtualMachinePreferenceSpec{
					CPU: &instancetypev1beta1.CPUPreferences{
						PreferredCPUTopology: &preferThreads,
					},
					Requirements: &instancetypev1beta1.PreferenceRequirements{
						CPU: &instancetypev1beta1.CPUPreferenceRequirement{
							Guest: uint32(2),
						},
					},
				},
				&v1.VirtualMachineInstanceSpec{
					Domain: v1.DomainSpec{
						CPU: &v1.CPU{
							Threads: uint32(2),
						},
					},
				},
			),
			Entry("by a VM for vCPUs using PreferSpread",
				nil,
				&instancetypev1beta1.VirtualMachinePreferenceSpec{
					CPU: &instancetypev1beta1.CPUPreferences{
						PreferredCPUTopology: &preferSpread,
					},
					Requirements: &instancetypev1beta1.PreferenceRequirements{
						CPU: &instancetypev1beta1.CPUPreferenceRequirement{
							Guest: uint32(6),
						},
					},
				},
				&v1.VirtualMachineInstanceSpec{
					Domain: v1.DomainSpec{
						CPU: &v1.CPU{
							Cores:   uint32(2),
							Sockets: uint32(3),
						},
					},
				},
			),
			Entry("by a VM for vCPUs using PreferAny",
				nil,
				&instancetypev1beta1.VirtualMachinePreferenceSpec{
					CPU: &instancetypev1beta1.CPUPreferences{
						PreferredCPUTopology: &preferAny,
					},
					Requirements: &instancetypev1beta1.PreferenceRequirements{
						CPU: &instancetypev1beta1.CPUPreferenceRequirement{
							Guest: uint32(4),
						},
					},
				},
				&v1.VirtualMachineInstanceSpec{
					Domain: v1.DomainSpec{
						CPU: &v1.CPU{
							Cores:   uint32(2),
							Sockets: uint32(2),
							Threads: uint32(1),
						},
					},
				},
			),
			Entry("by a VM for Memory",
				nil,
				&instancetypev1beta1.VirtualMachinePreferenceSpec{
					Requirements: &instancetypev1beta1.PreferenceRequirements{
						Memory: &instancetypev1beta1.MemoryPreferenceRequirement{
							Guest: resource.MustParse("1Gi"),
						},
					},
				},
				&v1.VirtualMachineInstanceSpec{
					Domain: v1.DomainSpec{
						Memory: &v1.Memory{
							Guest: resource.NewQuantity(1024*1024*1024, resource.BinarySI),
						},
					},
				},
			),
		)

		DescribeTable("should be rejected when insufficient resources are provided", func(instancetypeSpec *instancetypev1beta1.VirtualMachineInstancetypeSpec, preferenceSpec *instancetypev1beta1.VirtualMachinePreferenceSpec, vmiSpec *v1.VirtualMachineInstanceSpec, expectedConflict instancetype.Conflicts, errSubString string) {
			conflicts, err := instancetypeMethods.CheckPreferenceRequirements(instancetypeSpec, preferenceSpec, vmiSpec)
			Expect(conflicts).To(Equal(expectedConflict))
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring(errSubString))
		},
			Entry("by an instance type for vCPUs",
				&instancetypev1beta1.VirtualMachineInstancetypeSpec{
					CPU: instancetypev1beta1.CPUInstancetype{
						Guest: uint32(1),
					},
				},
				&instancetypev1beta1.VirtualMachinePreferenceSpec{
					Requirements: &instancetypev1beta1.PreferenceRequirements{
						CPU: &instancetypev1beta1.CPUPreferenceRequirement{
							Guest: uint32(2),
						},
					},
				},
				nil,
				instancetype.Conflicts{k8sfield.NewPath("spec", "instancetype")},
				fmt.Sprintf(instancetype.InsufficientInstanceTypeCPUResourcesErrorFmt, uint32(1), uint32(2)),
			),
			Entry("by an instance type for Memory",
				&instancetypev1beta1.VirtualMachineInstancetypeSpec{
					Memory: instancetypev1beta1.MemoryInstancetype{
						Guest: resource.MustParse("1Gi"),
					},
				},
				&instancetypev1beta1.VirtualMachinePreferenceSpec{
					Requirements: &instancetypev1beta1.PreferenceRequirements{
						Memory: &instancetypev1beta1.MemoryPreferenceRequirement{
							Guest: resource.MustParse("2Gi"),
						},
					},
				},
				nil,
				instancetype.Conflicts{k8sfield.NewPath("spec", "instancetype")},
				fmt.Sprintf(instancetype.InsufficientInstanceTypeMemoryResourcesErrorFmt, "1Gi", "2Gi"),
			),
			Entry("by a VM for vCPUs using PreferSockets (default)",
				nil,
				&instancetypev1beta1.VirtualMachinePreferenceSpec{
					CPU: &instancetypev1beta1.CPUPreferences{
						PreferredCPUTopology: &preferSockets,
					},
					Requirements: &instancetypev1beta1.PreferenceRequirements{
						CPU: &instancetypev1beta1.CPUPreferenceRequirement{
							Guest: uint32(2),
						},
					},
				},
				&v1.VirtualMachineInstanceSpec{
					Domain: v1.DomainSpec{
						CPU: &v1.CPU{
							Sockets: uint32(1),
						},
					},
				},
				instancetype.Conflicts{k8sfield.NewPath("spec", "template", "spec", "domain", "cpu", "sockets")},
				fmt.Sprintf(instancetype.InsufficientVMCPUResourcesErrorFmt, uint32(1), uint32(2), "sockets"),
			),
			Entry("by a VM for vCPUs using PreferCores",
				nil,
				&instancetypev1beta1.VirtualMachinePreferenceSpec{
					CPU: &instancetypev1beta1.CPUPreferences{
						PreferredCPUTopology: &preferCores,
					},
					Requirements: &instancetypev1beta1.PreferenceRequirements{
						CPU: &instancetypev1beta1.CPUPreferenceRequirement{
							Guest: uint32(2),
						},
					},
				},
				&v1.VirtualMachineInstanceSpec{
					Domain: v1.DomainSpec{
						CPU: &v1.CPU{
							Cores: uint32(1),
						},
					},
				},
				instancetype.Conflicts{k8sfield.NewPath("spec", "template", "spec", "domain", "cpu", "cores")},
				fmt.Sprintf(instancetype.InsufficientVMCPUResourcesErrorFmt, uint32(1), uint32(2), "cores"),
			),
			Entry("by a VM for vCPUs using PreferThreads",
				nil,
				&instancetypev1beta1.VirtualMachinePreferenceSpec{
					CPU: &instancetypev1beta1.CPUPreferences{
						PreferredCPUTopology: &preferThreads,
					},
					Requirements: &instancetypev1beta1.PreferenceRequirements{
						CPU: &instancetypev1beta1.CPUPreferenceRequirement{
							Guest: uint32(2),
						},
					},
				},
				&v1.VirtualMachineInstanceSpec{
					Domain: v1.DomainSpec{
						CPU: &v1.CPU{
							Threads: uint32(1),
						},
					},
				},
				instancetype.Conflicts{k8sfield.NewPath("spec", "template", "spec", "domain", "cpu", "threads")},
				fmt.Sprintf(instancetype.InsufficientVMCPUResourcesErrorFmt, uint32(1), uint32(2), "threads"),
			),
			Entry("by a VM for vCPUs using PreferSpread",
				nil,
				&instancetypev1beta1.VirtualMachinePreferenceSpec{
					CPU: &instancetypev1beta1.CPUPreferences{
						PreferredCPUTopology: &preferSpread,
					},
					Requirements: &instancetypev1beta1.PreferenceRequirements{
						CPU: &instancetypev1beta1.CPUPreferenceRequirement{
							Guest: uint32(4),
						},
					},
				},
				&v1.VirtualMachineInstanceSpec{
					Domain: v1.DomainSpec{
						CPU: &v1.CPU{
							Cores:   uint32(1),
							Sockets: uint32(1),
						},
					},
				},
				instancetype.Conflicts{k8sfield.NewPath("spec", "template", "spec", "domain", "cpu", "cores"), k8sfield.NewPath("spec", "template", "spec", "domain", "cpu", "sockets")},
				fmt.Sprintf(instancetype.InsufficientVMCPUResourcesErrorFmt, uint32(1), uint32(4), "cores and sockets"),
			),
			Entry("by a VM for vCPUs using PreferAny",
				nil,
				&instancetypev1beta1.VirtualMachinePreferenceSpec{
					CPU: &instancetypev1beta1.CPUPreferences{
						PreferredCPUTopology: &preferAny,
					},
					Requirements: &instancetypev1beta1.PreferenceRequirements{
						CPU: &instancetypev1beta1.CPUPreferenceRequirement{
							Guest: uint32(4),
						},
					},
				},
				&v1.VirtualMachineInstanceSpec{
					Domain: v1.DomainSpec{
						CPU: &v1.CPU{
							Sockets: uint32(2),
							Cores:   uint32(1),
							Threads: uint32(1),
						},
					},
				},
				instancetype.Conflicts{
					k8sfield.NewPath("spec", "template", "spec", "domain", "cpu", "cores"),
					k8sfield.NewPath("spec", "template", "spec", "domain", "cpu", "sockets"),
					k8sfield.NewPath("spec", "template", "spec", "domain", "cpu", "threads"),
				},
				fmt.Sprintf(instancetype.InsufficientVMCPUResourcesErrorFmt, uint32(2), uint32(4), "cores, sockets and threads"),
			),
			Entry("by a VM for Memory",
				nil,
				&instancetypev1beta1.VirtualMachinePreferenceSpec{
					Requirements: &instancetypev1beta1.PreferenceRequirements{
						Memory: &instancetypev1beta1.MemoryPreferenceRequirement{
							Guest: resource.MustParse("2Gi"),
						},
					},
				},
				&v1.VirtualMachineInstanceSpec{
					Domain: v1.DomainSpec{
						Memory: &v1.Memory{
							Guest: resource.NewQuantity(1024*1024*1024, resource.BinarySI),
						},
					},
				},
				instancetype.Conflicts{k8sfield.NewPath("spec", "template", "spec", "domain", "memory")},
				fmt.Sprintf(instancetype.InsufficientVMMemoryResourcesErrorFmt, "1Gi", "2Gi"),
			))

	})
})
