package operands

import (
	"context"
	"maps"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	appsv1 "k8s.io/api/apps/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"

	hcov1beta1 "github.com/kubevirt/hyperconverged-cluster-operator/api/v1beta1"
	"github.com/kubevirt/hyperconverged-cluster-operator/controllers/common"
	"github.com/kubevirt/hyperconverged-cluster-operator/controllers/commontestutils"
)

var _ = Describe("Deployment Handler", func() {
	Context("update or recreate the Deployment as required", func() {
		var hco *hcov1beta1.HyperConverged
		var req *common.HcoRequest
		var expectedDeployment *appsv1.Deployment

		BeforeEach(func() {
			hco = commontestutils.NewHco()
			req = commontestutils.NewReq(hco)

			expectedDeployment = NewExpectedDeployment(hco)
		})

		It("should recreate the Deployment as LabelSelector has changed", func() {
			modifiedDeployment := &appsv1.Deployment{}
			expectedDeployment.DeepCopyInto(modifiedDeployment)
			// modify the LabelSelector
			modifiedDeployment.Spec.Selector = &metav1.LabelSelector{
				MatchLabels: map[string]string{"key2": "value2"},
			}
			modifiedDeployment.UID = "oldObjectUID"

			// let's initialize the fake client with a modified object
			cl := commontestutils.InitClient([]client.Object{modifiedDeployment})

			foundResource := &appsv1.Deployment{}
			Expect(
				cl.Get(context.TODO(),
					types.NamespacedName{Namespace: modifiedDeployment.GetNamespace(), Name: modifiedDeployment.GetName()},
					foundResource),
			).ToNot(HaveOccurred())
			Expect(foundResource.GetUID()).To(Equal(types.UID("oldObjectUID")))

			// let's ensure the handler properly reconcile it back to the expected state
			handler := newDeploymentHandler(cl, commontestutils.GetScheme(), NewExpectedDeployment, hco)
			res := handler.ensure(req)
			Expect(res.Updated).To(BeTrue())
			Expect(res.Err).ToNot(HaveOccurred())

			foundResource = &appsv1.Deployment{}
			Expect(
				cl.Get(context.TODO(),
					types.NamespacedName{Namespace: modifiedDeployment.GetNamespace(), Name: modifiedDeployment.GetName()},
					foundResource),
			).ToNot(HaveOccurred())

			Expect(foundResource.Spec.Selector).To(Equal(expectedDeployment.Spec.Selector))
			// let's check the object UID to ensure that the object get really deleted and recreated
			Expect(foundResource.ObjectMeta.UID).ToNot(Equal(modifiedDeployment.UID))
		})

		It("should only update, not recreate, the Deployment since LabelSelector hasn't changed", func() {
			modifiedDeployment := &appsv1.Deployment{}
			expectedDeployment.DeepCopyInto(modifiedDeployment)
			// modify only the labels
			gotLabels := modifiedDeployment.GetLabels()
			gotLabels["key1"] = "wrongValue1"
			gotLabels["key2"] = "newValue2"
			modifiedDeployment.SetLabels(gotLabels)
			modifiedDeployment.UID = "oldObjectUID"

			// let's initialize the fake client with a modified object
			cl := commontestutils.InitClient([]client.Object{modifiedDeployment})
			foundResource := &appsv1.Deployment{}
			Expect(
				cl.Get(context.TODO(),
					types.NamespacedName{Namespace: modifiedDeployment.GetNamespace(), Name: modifiedDeployment.GetName()},
					foundResource),
			).ToNot(HaveOccurred())
			Expect(foundResource.GetUID()).To(Equal(types.UID("oldObjectUID")))

			// let's ensure the handler properly reconcile it back to the expected state
			handler := newDeploymentHandler(cl, commontestutils.GetScheme(), NewExpectedDeployment, hco)
			res := handler.ensure(req)
			Expect(res.Updated).To(BeTrue())
			Expect(res.Err).ToNot(HaveOccurred())

			foundResource = &appsv1.Deployment{}
			Expect(
				cl.Get(context.TODO(),
					types.NamespacedName{Namespace: modifiedDeployment.GetNamespace(), Name: modifiedDeployment.GetName()},
					foundResource),
			).ToNot(HaveOccurred())

			Expect(foundResource.Spec.Selector).To(Equal(expectedDeployment.Spec.Selector))
			// let's check the object UID to ensure that the object get updated and not deleted and recreated
			Expect(foundResource.GetUID()).To(Equal(types.UID("oldObjectUID")))
		})

		It("should reconcile managed labels to default without touching user added ones", func() {
			const userLabelKey = "userLabelKey"
			const userLabelValue = "userLabelValue"
			outdatedResource := NewExpectedDeployment(hco)
			expectedLabels := maps.Clone(outdatedResource.Labels)

			for k, v := range expectedLabels {
				outdatedResource.Labels[k] = "wrong_" + v
			}
			outdatedResource.Labels[userLabelKey] = userLabelValue

			cl := commontestutils.InitClient([]client.Object{hco, outdatedResource})
			handler := newDeploymentHandler(cl, commontestutils.GetScheme(), NewExpectedDeployment, hco)

			res := handler.ensure(req)
			Expect(res.UpgradeDone).To(BeFalse())
			Expect(res.Updated).To(BeTrue())
			Expect(res.Err).ToNot(HaveOccurred())

			foundResource := &appsv1.Deployment{}
			Expect(
				cl.Get(context.TODO(),
					types.NamespacedName{Name: outdatedResource.Name, Namespace: outdatedResource.Namespace},
					foundResource),
			).ToNot(HaveOccurred())

			for k, v := range expectedLabels {
				Expect(foundResource.Labels).To(HaveKeyWithValue(k, v))
			}
			Expect(foundResource.Labels).To(HaveKeyWithValue(userLabelKey, userLabelValue))
		})

		It("should reconcile managed labels to default on label deletion without touching user added ones", func() {
			const userLabelKey = "userLabelKey"
			const userLabelValue = "userLabelValue"
			outdatedResource := NewExpectedDeployment(hco)
			expectedLabels := maps.Clone(outdatedResource.Labels)

			removed := false
			for k := range outdatedResource.Labels {
				if !removed {
					delete(outdatedResource.Labels, k)
					removed = true
				}
			}
			outdatedResource.Labels[userLabelKey] = userLabelValue

			cl := commontestutils.InitClient([]client.Object{hco, outdatedResource})
			handler := newDeploymentHandler(cl, commontestutils.GetScheme(), NewExpectedDeployment, hco)

			res := handler.ensure(req)
			Expect(res.UpgradeDone).To(BeFalse())
			Expect(res.Updated).To(BeTrue())
			Expect(res.Err).ToNot(HaveOccurred())

			foundResource := &appsv1.Deployment{}
			Expect(
				cl.Get(context.TODO(),
					types.NamespacedName{Name: outdatedResource.Name, Namespace: outdatedResource.Namespace},
					foundResource),
			).ToNot(HaveOccurred())

			for k, v := range expectedLabels {
				Expect(foundResource.Labels).To(HaveKeyWithValue(k, v))
			}
			Expect(foundResource.Labels).To(HaveKeyWithValue(userLabelKey, userLabelValue))
		})

	})

})

func NewExpectedDeployment(_ *hcov1beta1.HyperConverged) *appsv1.Deployment {
	return &appsv1.Deployment{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Deployment",
			APIVersion: "apps/v1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:   "modifiedDeployment",
			Labels: map[string]string{"key1": "value1"},
		},
		Spec: appsv1.DeploymentSpec{
			Selector: &metav1.LabelSelector{
				MatchLabels: map[string]string{"key1": "value1"},
			},
		},
	}
}
