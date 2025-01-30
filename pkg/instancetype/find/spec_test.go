//nolint:dupl
package find_test

import (
	"context"
	"encoding/json"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	appsv1 "k8s.io/api/apps/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	k8sfake "k8s.io/client-go/kubernetes/fake"
	"k8s.io/client-go/testing"
	"k8s.io/client-go/tools/cache"

	"github.com/golang/mock/gomock"

	v1 "kubevirt.io/api/core/v1"
	apiinstancetype "kubevirt.io/api/instancetype"
	"kubevirt.io/api/instancetype/v1alpha1"
	"kubevirt.io/api/instancetype/v1beta1"
	"kubevirt.io/client-go/kubecli"
	"kubevirt.io/client-go/kubevirt/fake"

	"kubevirt.io/kubevirt/pkg/instancetype/find"
	"kubevirt.io/kubevirt/pkg/libvmi"
	"kubevirt.io/kubevirt/pkg/pointer"
	"kubevirt.io/kubevirt/pkg/testutils"
)

var _ = Describe("Instance Type SpecFinder", func() {
	const (
		nonExistingResourceName = "non-existing-resource"
	)

	type instancetypeSpecFinder interface {
		Find(vm *v1.VirtualMachine) (*v1beta1.VirtualMachineInstancetypeSpec, error)
	}

	var (
		finder instancetypeSpecFinder
		vm     *v1.VirtualMachine

		virtClient                       *kubecli.MockKubevirtClient
		fakeClientset                    *fake.Clientset
		instancetypeInformerStore        cache.Store
		clusterInstancetypeInformerStore cache.Store
		controllerRevisionInformerStore  cache.Store
	)

	BeforeEach(func() {
		ctrl := gomock.NewController(GinkgoT())
		virtClient = kubecli.NewMockKubevirtClient(ctrl)

		virtClient.EXPECT().AppsV1().Return(k8sfake.NewSimpleClientset().AppsV1()).AnyTimes()

		fakeClientset = fake.NewSimpleClientset()

		virtClient.EXPECT().VirtualMachine(metav1.NamespaceDefault).Return(
			fakeClientset.KubevirtV1().VirtualMachines(metav1.NamespaceDefault)).AnyTimes()

		virtClient.EXPECT().VirtualMachineInstancetype(metav1.NamespaceDefault).Return(
			fakeClientset.InstancetypeV1beta1().VirtualMachineInstancetypes(metav1.NamespaceDefault)).AnyTimes()

		virtClient.EXPECT().VirtualMachineClusterInstancetype().Return(
			fakeClientset.InstancetypeV1beta1().VirtualMachineClusterInstancetypes()).AnyTimes()

		instancetypeInformer, _ := testutils.NewFakeInformerFor(&v1beta1.VirtualMachineInstancetype{})
		instancetypeInformerStore = instancetypeInformer.GetStore()

		clusterInstancetypeInformer, _ := testutils.NewFakeInformerFor(&v1beta1.VirtualMachineClusterInstancetype{})
		clusterInstancetypeInformerStore = clusterInstancetypeInformer.GetStore()

		controllerRevisionInformer, _ := testutils.NewFakeInformerFor(&appsv1.ControllerRevision{})
		controllerRevisionInformerStore = controllerRevisionInformer.GetStore()

		finder = find.NewSpecFinder(
			instancetypeInformerStore,
			clusterInstancetypeInformerStore,
			controllerRevisionInformerStore,
			virtClient,
		)
	})

	It("find returns nil when no instancetype is specified", func() {
		vm = libvmi.NewVirtualMachine(libvmi.New())
		spec, err := finder.Find(vm)
		Expect(err).ToNot(HaveOccurred())
		Expect(spec).To(BeNil())
	})

	It("find returns error when invalid Instancetype Kind is specified", func() {
		vm = libvmi.NewVirtualMachine(libvmi.New())
		vm.Spec.Instancetype = &v1.InstancetypeMatcher{
			Name: "foo",
			Kind: "bar",
		}
		spec, err := finder.Find(vm)
		Expect(err).To(MatchError(ContainSubstring("got unexpected kind in InstancetypeMatcher")))
		Expect(spec).To(BeNil())
	})

	Context("Using global ClusterInstancetype", func() {
		var clusterInstancetype *v1beta1.VirtualMachineClusterInstancetype

		BeforeEach(func() {
			clusterInstancetype = &v1beta1.VirtualMachineClusterInstancetype{
				ObjectMeta: metav1.ObjectMeta{
					Name: "test-cluster-instancetype",
				},
				Spec: v1beta1.VirtualMachineInstancetypeSpec{
					CPU: v1beta1.CPUInstancetype{
						Guest: uint32(2),
					},
					Memory: v1beta1.MemoryInstancetype{
						Guest: resource.MustParse("128Mi"),
					},
				},
			}

			_, err := virtClient.VirtualMachineClusterInstancetype().Create(context.Background(), clusterInstancetype, metav1.CreateOptions{})
			Expect(err).ToNot(HaveOccurred())

			err = clusterInstancetypeInformerStore.Add(clusterInstancetype)
			Expect(err).ToNot(HaveOccurred())

			vm = libvmi.NewVirtualMachine(
				libvmi.New(libvmi.WithNamespace(metav1.NamespaceDefault)),
				libvmi.WithClusterInstancetype(clusterInstancetype.Name),
			)
		})

		It("returns expected instancetype", func() {
			instancetypeSpec, err := finder.Find(vm)
			Expect(err).ToNot(HaveOccurred())
			Expect(instancetypeSpec).To(HaveValue(Equal(clusterInstancetype.Spec)))
		})

		It("find returns expected instancetype spec with no kind provided", func() {
			vm.Spec.Instancetype.Kind = ""
			instancetypeSpec, err := finder.Find(vm)
			Expect(err).ToNot(HaveOccurred())
			Expect(instancetypeSpec).To(HaveValue(Equal(clusterInstancetype.Spec)))
		})

		It("uses client when instancetype not found within informer", func() {
			err := clusterInstancetypeInformerStore.Delete(clusterInstancetype)
			Expect(err).ToNot(HaveOccurred())

			instancetypeSpec, err := finder.Find(vm)
			Expect(err).ToNot(HaveOccurred())
			Expect(instancetypeSpec).To(HaveValue(Equal(clusterInstancetype.Spec)))
			Expect(fakeClientset.Actions()).To(
				ContainElement(
					testing.NewGetAction(
						v1beta1.SchemeGroupVersion.WithResource(apiinstancetype.ClusterPluralResourceName),
						"",
						vm.Spec.Instancetype.Name,
					),
				),
			)
		})

		It("returns expected instancetype using only the client", func() {
			finder = find.NewSpecFinder(nil, nil, nil, virtClient)
			instancetypeSpec, err := finder.Find(vm)
			Expect(err).ToNot(HaveOccurred())
			Expect(instancetypeSpec).To(HaveValue(Equal(clusterInstancetype.Spec)))
			Expect(fakeClientset.Actions()).To(
				ContainElement(
					testing.NewGetAction(
						v1beta1.SchemeGroupVersion.WithResource(apiinstancetype.ClusterPluralResourceName),
						"",
						vm.Spec.Instancetype.Name,
					),
				),
			)
		})

		It("find fails when instancetype does not exist", func() {
			vm = libvmi.NewVirtualMachine(libvmi.New(), libvmi.WithClusterInstancetype(nonExistingResourceName))
			_, err := finder.Find(vm)
			Expect(err).To(MatchError(errors.IsNotFound, "IsNotFound"))
		})

		It("find successfully decodes v1alpha1 SpecRevision ControllerRevision without APIVersion set - bug #9261", func() {
			clusterInstancetype.Spec.CPU = v1beta1.CPUInstancetype{
				Guest: uint32(2),
				// Set the following values to be compatible with objects converted from v1alpha1
				Model:                 pointer.P(""),
				DedicatedCPUPlacement: pointer.P(false),
				IsolateEmulatorThread: pointer.P(false),
			}

			specData, err := json.Marshal(clusterInstancetype.Spec)
			Expect(err).ToNot(HaveOccurred())

			// Do not set APIVersion as part of VirtualMachineInstancetypeSpecRevision in order to trigger bug #9261
			specRevision := v1alpha1.VirtualMachineInstancetypeSpecRevision{
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

			_, err = virtClient.AppsV1().ControllerRevisions(vm.Namespace).Create(
				context.Background(), controllerRevision, metav1.CreateOptions{})
			Expect(err).ToNot(HaveOccurred())

			vm.Spec.Instancetype.RevisionName = controllerRevision.Name

			foundInstancetypeSpec, err := finder.Find(vm)
			Expect(err).ToNot(HaveOccurred())
			Expect(foundInstancetypeSpec).To(HaveValue(Equal(clusterInstancetype.Spec)))
		})
	})

	Context("Using namespaced Instancetype", func() {
		var fakeInstancetype *v1beta1.VirtualMachineInstancetype

		BeforeEach(func() {
			fakeInstancetype = &v1beta1.VirtualMachineInstancetype{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-instancetype",
					Namespace: metav1.NamespaceDefault,
				},
				Spec: v1beta1.VirtualMachineInstancetypeSpec{
					CPU: v1beta1.CPUInstancetype{
						Guest: uint32(2),
					},
					Memory: v1beta1.MemoryInstancetype{
						Guest: resource.MustParse("128Mi"),
					},
				},
			}

			_, err := virtClient.VirtualMachineInstancetype(metav1.NamespaceDefault).Create(
				context.Background(), fakeInstancetype, metav1.CreateOptions{})
			Expect(err).ToNot(HaveOccurred())

			err = instancetypeInformerStore.Add(fakeInstancetype)
			Expect(err).ToNot(HaveOccurred())

			vm = libvmi.NewVirtualMachine(libvmi.New(
				libvmi.WithNamespace(metav1.NamespaceDefault)),
				libvmi.WithInstancetype(fakeInstancetype.Name),
			)
		})

		It("find returns expected instancetype", func() {
			instancetypeSpec, err := finder.Find(vm)
			Expect(err).ToNot(HaveOccurred())
			Expect(instancetypeSpec).To(HaveValue(Equal(fakeInstancetype.Spec)))
		})

		It("uses client when instancetype not found within informer", func() {
			err := instancetypeInformerStore.Delete(fakeInstancetype)
			Expect(err).ToNot(HaveOccurred())
			instancetypeSpec, err := finder.Find(vm)
			Expect(err).ToNot(HaveOccurred())
			Expect(instancetypeSpec).To(HaveValue(Equal(fakeInstancetype.Spec)))
			Expect(fakeClientset.Actions()).To(
				ContainElement(
					testing.NewGetAction(
						v1beta1.SchemeGroupVersion.WithResource(apiinstancetype.PluralResourceName),
						vm.Namespace,
						vm.Spec.Instancetype.Name,
					),
				),
			)
		})

		It("returns expected instancetype using only the client", func() {
			finder = find.NewSpecFinder(nil, nil, nil, virtClient)
			instancetypeSpec, err := finder.Find(vm)
			Expect(err).ToNot(HaveOccurred())
			Expect(instancetypeSpec).To(HaveValue(Equal(fakeInstancetype.Spec)))
			Expect(fakeClientset.Actions()).To(
				ContainElement(
					testing.NewGetAction(
						v1beta1.SchemeGroupVersion.WithResource(apiinstancetype.PluralResourceName),
						vm.Namespace,
						vm.Spec.Instancetype.Name,
					),
				),
			)
		})

		It("find fails when instancetype does not exist", func() {
			libvmi.NewVirtualMachine(libvmi.New(libvmi.WithNamespace(metav1.NamespaceDefault)), libvmi.WithInstancetype(nonExistingResourceName))
			vm.Spec.Instancetype.Name = nonExistingResourceName
			_, err := finder.Find(vm)
			Expect(err).To(MatchError(errors.IsNotFound, "IsNotFound"))
		})

		It("find successfully decodes v1alpha1 SpecRevision ControllerRevision without APIVersion set - bug #9261", func() {
			fakeInstancetype.Spec.CPU = v1beta1.CPUInstancetype{
				Guest: uint32(2),
				// Set the following values to be compatible with objects converted from v1alpha1
				Model:                 pointer.P(""),
				DedicatedCPUPlacement: pointer.P(false),
				IsolateEmulatorThread: pointer.P(false),
			}

			specData, err := json.Marshal(fakeInstancetype.Spec)
			Expect(err).ToNot(HaveOccurred())

			// Do not set APIVersion as part of VirtualMachineInstancetypeSpecRevision in order to trigger bug #9261
			specRevision := v1alpha1.VirtualMachineInstancetypeSpecRevision{
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

			vm.Spec.Instancetype.RevisionName = controllerRevision.Name

			foundInstancetypeSpec, err := finder.Find(vm)
			Expect(err).ToNot(HaveOccurred())
			Expect(foundInstancetypeSpec).To(HaveValue(Equal(fakeInstancetype.Spec)))
		})
	})
})
