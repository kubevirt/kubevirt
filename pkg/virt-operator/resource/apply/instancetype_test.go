package apply

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/golang/mock/gomock"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/testing"
	v1 "kubevirt.io/api/core/v1"
	apiinstancetype "kubevirt.io/api/instancetype"
	instancetypev1beta1 "kubevirt.io/api/instancetype/v1beta1"
	"kubevirt.io/client-go/generated/kubevirt/clientset/versioned/fake"
	"kubevirt.io/client-go/kubecli"
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

		reconciler = &Reconciler{
			kv:        &v1.KubeVirt{},
			clientset: virtClient,
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

			imageTag, imageRegistry, id := getTargetVersionRegistryID(reconciler.kv)
			injectOperatorMetadata(reconciler.kv, &instancetype.ObjectMeta, imageTag, imageRegistry, id, true)
		})

		It("should create instancetype", func() {
			getCalled := expectGetReturnNotFound(fakeClient, apiinstancetype.ClusterPluralResourceName)
			createCalled := expectCreate(fakeClient, apiinstancetype.ClusterPluralResourceName, instancetype)
			Expect(reconciler.createOrUpdateInstancetype(instancetype)).To(Succeed())
			Expect(*getCalled).To(BeTrue())
			Expect(*createCalled).To(BeTrue())
		})

		It("should not update instancetypes on sync when they are equal", func() {
			getCalled := expectGet(fakeClient, apiinstancetype.ClusterPluralResourceName, instancetype)
			Expect(reconciler.createOrUpdateInstancetype(instancetype)).To(Succeed())
			Expect(*getCalled).To(BeTrue())
		})

		DescribeTable("should update instancetypes on sync when they are not equal", func(modifyFn func(*instancetypev1beta1.VirtualMachineClusterInstancetype)) {
			modifiedInstancetype := instancetype.DeepCopy()
			modifyFn(modifiedInstancetype)
			getCalled := expectGet(fakeClient, apiinstancetype.ClusterPluralResourceName, modifiedInstancetype)
			updateCalled := expectUpdate(fakeClient, apiinstancetype.ClusterPluralResourceName, instancetype)
			Expect(reconciler.createOrUpdateInstancetype(instancetype)).To(Succeed())
			Expect(*getCalled).To(BeTrue())
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

		It("should delete all instancetypes managed by virt-operator", func() {
			deleteCollectionCalled := expectDeleteCollection(fakeClient, apiinstancetype.ClusterPluralResourceName)
			Expect(reconciler.deleteInstancetypes()).To(Succeed())
			Expect(*deleteCollectionCalled).To(BeTrue())
		})
	})

	Context("Preferences", func() {
		var preference *instancetypev1beta1.VirtualMachineClusterPreference

		BeforeEach(func() {
			clusterPreferenceClient := fakeClient.InstancetypeV1beta1().VirtualMachineClusterPreferences()
			virtClient.EXPECT().VirtualMachineClusterPreference().Return(clusterPreferenceClient).AnyTimes()

			preferredTopology := instancetypev1beta1.PreferCores
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

			imageTag, imageRegistry, id := getTargetVersionRegistryID(reconciler.kv)
			injectOperatorMetadata(reconciler.kv, &preference.ObjectMeta, imageTag, imageRegistry, id, true)
		})

		It("should create preference", func() {
			getCalled := expectGetReturnNotFound(fakeClient, apiinstancetype.ClusterPluralPreferenceResourceName)
			createCalled := expectCreate(fakeClient, apiinstancetype.ClusterPluralPreferenceResourceName, preference)
			Expect(reconciler.createOrUpdatePreference(preference)).To(Succeed())
			Expect(*getCalled).To(BeTrue())
			Expect(*createCalled).To(BeTrue())
		})

		It("should not update preferences on sync when they are equal", func() {
			getCalled := expectGet(fakeClient, apiinstancetype.ClusterPluralPreferenceResourceName, preference)
			Expect(reconciler.createOrUpdatePreference(preference)).To(Succeed())
			Expect(*getCalled).To(BeTrue())
		})

		DescribeTable("should update preferences on sync when they are not equal", func(modifyFn func(*instancetypev1beta1.VirtualMachineClusterPreference)) {
			modifiedPreference := preference.DeepCopy()
			modifyFn(modifiedPreference)
			getCalled := expectGet(fakeClient, apiinstancetype.ClusterPluralPreferenceResourceName, modifiedPreference)
			updateCalled := expectUpdate(fakeClient, apiinstancetype.ClusterPluralPreferenceResourceName, preference)
			Expect(reconciler.createOrUpdatePreference(preference)).To(Succeed())
			Expect(*getCalled).To(BeTrue())
			Expect(*updateCalled).To(BeTrue())
		},
			Entry("on modified annotations", func(preference *instancetypev1beta1.VirtualMachineClusterPreference) {
				preference.Annotations["test"] = "modified"
			}),
			Entry("on modified labels", func(preference *instancetypev1beta1.VirtualMachineClusterPreference) {
				preference.Labels["test"] = "modified"
			}),
			Entry("on modified spec", func(preference *instancetypev1beta1.VirtualMachineClusterPreference) {
				preferredTopology := instancetypev1beta1.PreferThreads
				preference.Spec.CPU.PreferredCPUTopology = &preferredTopology
			}),
		)

		It("should delete all preferences managed by virt-operator", func() {
			deleteCollectionCalled := expectDeleteCollection(fakeClient, apiinstancetype.ClusterPluralPreferenceResourceName)
			Expect(reconciler.deletePreferences()).To(Succeed())
			Expect(*deleteCollectionCalled).To(BeTrue())
		})
	})

})

func expectGet(fakeClient *fake.Clientset, resource string, object runtime.Object) *bool {
	called := false
	fakeClient.Fake.PrependReactor("get", resource, func(action testing.Action) (bool, runtime.Object, error) {
		_, ok := action.(testing.GetAction)
		Expect(ok).To(BeTrue())
		called = true
		return true, object, nil
	})
	return &called
}

func expectGetReturnNotFound(fakeClient *fake.Clientset, resource string) *bool {
	called := false
	fakeClient.Fake.PrependReactor("get", resource, func(action testing.Action) (bool, runtime.Object, error) {
		get, ok := action.(testing.GetAction)
		Expect(ok).To(BeTrue())
		called = true
		return true, nil, k8serrors.NewNotFound(get.GetResource().GroupResource(), get.GetName())
	})
	return &called
}

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

func expectDeleteCollection(fakeClient *fake.Clientset, resource string) *bool {
	called := false
	fakeClient.Fake.PrependReactor("delete-collection", resource, func(action testing.Action) (bool, runtime.Object, error) {
		deleteCollection, ok := action.(testing.DeleteCollectionAction)
		Expect(ok).To(BeTrue())
		ls := labels.Set{
			v1.AppComponentLabel: v1.AppComponent,
			v1.ManagedByLabel:    v1.ManagedByLabelOperatorValue,
		}
		Expect(deleteCollection.GetListRestrictions().Labels).To(Equal(ls.AsSelector()))
		called = true
		return true, nil, nil
	})
	return &called
}
