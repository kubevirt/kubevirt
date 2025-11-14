package apply

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"go.uber.org/mock/gomock"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/testing"
	v1 "kubevirt.io/api/core/v1"
	apiinstancetype "kubevirt.io/api/instancetype"
	instancetypev1beta1 "kubevirt.io/api/instancetype/v1beta1"
	"kubevirt.io/client-go/kubecli"
	"kubevirt.io/client-go/kubevirt/fake"

	"kubevirt.io/kubevirt/pkg/pointer"
	"kubevirt.io/kubevirt/pkg/testutils"
	fake2 "kubevirt.io/kubevirt/pkg/virt-operator/resource/apply/fake"
	"kubevirt.io/kubevirt/pkg/virt-operator/util"
)

var _ = Describe("Apply Instancetypes", func() {
	var virtClient *kubecli.MockKubevirtClient
	var fakeClient *fake.Clientset
	var reconciler *Reconciler

	BeforeEach(func() {
		ctrl := gomock.NewController(GinkgoT())
		virtClient = kubecli.NewMockKubevirtClient(ctrl)

		fakeClient = fake.NewSimpleClientset()
		// Make sure that any unexpected call to the client will fail
		fakeClient.Fake.PrependReactor("*", "*", func(action testing.Action) (bool, runtime.Object, error) {
			Expect(action).To(BeNil())
			return true, nil, nil
		})

		clusterInstancetypeInformer, _ := testutils.NewFakeInformerFor(&instancetypev1beta1.VirtualMachineClusterInstancetype{})
		clusterPreferenceInformer, _ := testutils.NewFakeInformerFor(&instancetypev1beta1.VirtualMachineClusterPreference{})

		reconciler = &Reconciler{
			kv:            &v1.KubeVirt{},
			virtClientset: virtClient,
			stores: util.Stores{
				ClusterInstancetype: clusterInstancetypeInformer.GetStore(),
				ClusterPreference:   clusterPreferenceInformer.GetStore(),
			},
		}
	})

	Context("Instancetypes", func() {
		var instancetype *instancetypev1beta1.VirtualMachineClusterInstancetype

		BeforeEach(func() {
			clusterInstancetypeClient := fakeClient.InstancetypeV1beta1().VirtualMachineClusterInstancetypes()
			virtClient.EXPECT().VirtualMachineClusterInstancetype().Return(clusterInstancetypeClient).AnyTimes()

			instancetype = &instancetypev1beta1.VirtualMachineClusterInstancetype{
				ObjectMeta: metav1.ObjectMeta{
					Name: "clusterinstancetype",
				},
				Spec: instancetypev1beta1.VirtualMachineInstancetypeSpec{
					CPU: instancetypev1beta1.CPUInstancetype{
						Guest: 1,
					},
				},
			}

			reconciler.targetStrategy = &fake2.FakeStrategy{
				FakeInstancetypes: []*instancetypev1beta1.VirtualMachineClusterInstancetype{instancetype},
			}

			imageTag, imageRegistry, id := getTargetVersionRegistryID(reconciler.kv)
			injectOperatorMetadata(reconciler.kv, &instancetype.ObjectMeta, imageTag, imageRegistry, id, true)
		})

		It("should create instancetype", func() {
			createCalled := expectCreate(fakeClient, apiinstancetype.ClusterPluralResourceName, instancetype)
			Expect(reconciler.createOrUpdateInstancetype(instancetype)).To(Succeed())
			Expect(*createCalled).To(BeTrue())
		})

		It("should not update instancetypes on sync when they are equal", func() {
			reconciler.stores.ClusterInstancetype.Add(instancetype)
			Expect(reconciler.createOrUpdateInstancetype(instancetype)).To(Succeed())
		})

		DescribeTable("should update instancetypes on sync when they are not equal", func(modifyFn func(*instancetypev1beta1.VirtualMachineClusterInstancetype)) {
			modifiedInstancetype := instancetype.DeepCopy()
			modifyFn(modifiedInstancetype)
			reconciler.stores.ClusterInstancetype.Add(modifiedInstancetype)
			updateCalled := expectUpdate(fakeClient, apiinstancetype.ClusterPluralResourceName, instancetype)
			Expect(reconciler.createOrUpdateInstancetype(instancetype)).To(Succeed())
			Expect(*updateCalled).To(BeTrue())
		},
			Entry("on modified annotations", func(instancetype *instancetypev1beta1.VirtualMachineClusterInstancetype) {
				instancetype.Annotations["test"] = "modified"
			}),
			Entry("on modified labels", func(instancetype *instancetypev1beta1.VirtualMachineClusterInstancetype) {
				instancetype.Labels["test"] = "modified"
			}),
			Entry("on modified spec", func(instancetype *instancetypev1beta1.VirtualMachineClusterInstancetype) {
				instancetype.Spec.CPU.Guest = 2
			}),
		)

		It("should delete all deployed instance types managed by virt-operator", func() {
			reconciler.stores.ClusterInstancetype.Add(instancetype)
			deleteCollectionCalled := expectDeleteCollection(fakeClient, reconciler.kv, apiinstancetype.ClusterPluralResourceName)
			Expect(reconciler.deleteInstancetypes()).To(Succeed())
			Expect(*deleteCollectionCalled).To(BeTrue())
		})

		It("should not call delete if no instance types have been deployed by virt-operator", func() {
			Expect(reconciler.deleteInstancetypes()).To(Succeed())
		})
	})

	Context("Preferences", func() {
		var preference *instancetypev1beta1.VirtualMachineClusterPreference

		BeforeEach(func() {
			clusterPreferenceClient := fakeClient.InstancetypeV1beta1().VirtualMachineClusterPreferences()
			virtClient.EXPECT().VirtualMachineClusterPreference().Return(clusterPreferenceClient).AnyTimes()

			preferredTopology := instancetypev1beta1.Cores
			preference = &instancetypev1beta1.VirtualMachineClusterPreference{
				ObjectMeta: metav1.ObjectMeta{
					Name: "clusterpreference",
				},
				Spec: instancetypev1beta1.VirtualMachinePreferenceSpec{
					CPU: &instancetypev1beta1.CPUPreferences{
						PreferredCPUTopology: &preferredTopology,
					},
				},
			}

			reconciler.targetStrategy = &fake2.FakeStrategy{
				FakePreferences: []*instancetypev1beta1.VirtualMachineClusterPreference{preference},
			}

			imageTag, imageRegistry, id := getTargetVersionRegistryID(reconciler.kv)
			injectOperatorMetadata(reconciler.kv, &preference.ObjectMeta, imageTag, imageRegistry, id, true)
		})

		It("should create preference", func() {
			createCalled := expectCreate(fakeClient, apiinstancetype.ClusterPluralPreferenceResourceName, preference)
			Expect(reconciler.createOrUpdatePreference(preference)).To(Succeed())
			Expect(*createCalled).To(BeTrue())
		})

		It("should not update preferences on sync when they are equal", func() {
			reconciler.stores.ClusterPreference.Add(preference)
			Expect(reconciler.createOrUpdatePreference(preference)).To(Succeed())
		})

		DescribeTable("should update preferences on sync when they are not equal", func(modifyFn func(*instancetypev1beta1.VirtualMachineClusterPreference)) {
			modifiedPreference := preference.DeepCopy()
			modifyFn(modifiedPreference)
			reconciler.stores.ClusterPreference.Add(modifiedPreference)
			updateCalled := expectUpdate(fakeClient, apiinstancetype.ClusterPluralPreferenceResourceName, preference)
			Expect(reconciler.createOrUpdatePreference(preference)).To(Succeed())
			Expect(*updateCalled).To(BeTrue())
		},
			Entry("on modified annotations", func(preference *instancetypev1beta1.VirtualMachineClusterPreference) {
				preference.Annotations["test"] = "modified"
			}),
			Entry("on modified labels", func(preference *instancetypev1beta1.VirtualMachineClusterPreference) {
				preference.Labels["test"] = "modified"
			}),
			Entry("on modified spec", func(preference *instancetypev1beta1.VirtualMachineClusterPreference) {
				preference.Spec.CPU.PreferredCPUTopology = pointer.P(instancetypev1beta1.Spread)
			}),
		)

		It("should delete all deployed preferences managed by virt-operator", func() {
			reconciler.stores.ClusterPreference.Add(preference)
			deleteCollectionCalled := expectDeleteCollection(fakeClient, reconciler.kv, apiinstancetype.ClusterPluralPreferenceResourceName)
			Expect(reconciler.deletePreferences()).To(Succeed())
			Expect(*deleteCollectionCalled).To(BeTrue())
		})

		It("should not call delete if no preferences have been deployed by virt-operator", func() {
			Expect(reconciler.deletePreferences()).To(Succeed())
		})
	})

})

func expectCreate(fakeClient *fake.Clientset, resource string, object runtime.Object) *bool {
	called := false
	fakeClient.Fake.PrependReactor("create", resource, func(action testing.Action) (bool, runtime.Object, error) {
		create, ok := action.(testing.CreateAction)
		Expect(ok).To(BeTrue())
		Expect(create.GetObject()).To(Equal(object))
		called = true
		return true, object, nil
	})
	return &called
}

func expectUpdate(fakeClient *fake.Clientset, resource string, object runtime.Object) *bool {
	called := false
	fakeClient.Fake.PrependReactor("update", resource, func(action testing.Action) (bool, runtime.Object, error) {
		update, ok := action.(testing.UpdateAction)
		Expect(ok).To(BeTrue())
		Expect(update.GetObject()).To(Equal(object))
		called = true
		return true, object, nil
	})
	return &called
}

func expectDeleteCollection(fakeClient *fake.Clientset, kv *v1.KubeVirt, resource string) *bool {
	called := false
	fakeClient.Fake.PrependReactor("delete-collection", resource, func(action testing.Action) (bool, runtime.Object, error) {
		deleteCollection, ok := action.(testing.DeleteCollectionAction)
		Expect(ok).To(BeTrue())
		ls := labels.Set{
			v1.AppComponentLabel: GetAppComponent(kv),
			v1.ManagedByLabel:    v1.ManagedByLabelOperatorValue,
		}
		Expect(deleteCollection.GetListRestrictions().Labels).To(Equal(ls.AsSelector()))
		called = true
		return true, nil, nil
	})
	return &called
}
