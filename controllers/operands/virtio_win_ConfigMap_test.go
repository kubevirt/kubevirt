package operands

import (
	"context"
	"fmt"
	"reflect"

	rbacv1 "k8s.io/api/rbac/v1"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/tools/reference"

	hcoutil "github.com/kubevirt/hyperconverged-cluster-operator/pkg/util"

	"os"

	hcov1beta1 "github.com/kubevirt/hyperconverged-cluster-operator/api/v1beta1"
	"github.com/kubevirt/hyperconverged-cluster-operator/controllers/common"
	"github.com/kubevirt/hyperconverged-cluster-operator/controllers/commonTestUtils"
)

var _ = Describe("VirtioWin", func() {
	Context("Virtio-Win ConfigMap", func() {

		var hco *hcov1beta1.HyperConverged
		var req *common.HcoRequest

		BeforeEach(func() {
			os.Setenv("VIRTIOWIN_CONTAINER", "new-virtiowin-container-value")
			hco = commonTestUtils.NewHco()
			req = commonTestUtils.NewReq(hco)
		})

		It("should error if VIRTIOWIN_CONTAINER environment var not specified", func() {
			os.Unsetenv("VIRTIOWIN_CONTAINER")

			cl := commonTestUtils.InitClient([]runtime.Object{})
			handler, err := newVirtioWinCmHandler(logger, cl, commonTestUtils.GetScheme(), hco)

			Expect(err).ToNot(BeNil())
			Expect(handler).To(BeNil())
		})

		It("should create if not present", func() {
			expectedResource, err := NewVirtioWinCm(hco)
			Expect(err).ToNot(HaveOccurred())
			cl := commonTestUtils.InitClient([]runtime.Object{})
			handler, _ := newVirtioWinCmHandler(logger, cl, commonTestUtils.GetScheme(), hco)
			res := handler[0].ensure(req)
			Expect(res.UpgradeDone).To(BeFalse())
			Expect(res.Err).To(BeNil())

			foundResource := &corev1.ConfigMap{}
			Expect(
				cl.Get(context.TODO(),
					types.NamespacedName{Name: expectedResource.Name, Namespace: expectedResource.Namespace},
					foundResource),
			).To(BeNil())
			Expect(foundResource.Name).To(Equal(expectedResource.Name))
			Expect(foundResource.Labels).Should(HaveKeyWithValue(hcoutil.AppLabel, commonTestUtils.Name))
			Expect(foundResource.Namespace).To(Equal(expectedResource.Namespace))
		})

		It("should find if present", func() {
			expectedResource, err := NewVirtioWinCm(hco)
			Expect(err).ToNot(HaveOccurred())

			expectedResource.ObjectMeta.SelfLink = fmt.Sprintf("/apis/v1/namespaces/%s/dummies/%s", expectedResource.Namespace, expectedResource.Name)
			cl := commonTestUtils.InitClient([]runtime.Object{hco, expectedResource})
			handler, _ := newVirtioWinCmHandler(logger, cl, commonTestUtils.GetScheme(), hco)
			res := handler[0].ensure(req)
			Expect(res.UpgradeDone).To(BeFalse())
			Expect(res.Err).To(BeNil())

			// Check HCO's status
			Expect(hco.Status.RelatedObjects).To(Not(BeNil()))
			objectRef, err := reference.GetReference(commonTestUtils.GetScheme(), expectedResource)
			Expect(err).To(BeNil())
			// ObjectReference should have been added
			Expect(hco.Status.RelatedObjects).To(ContainElement(*objectRef))
		})

		It("should reconcile according to env values and HCO CR", func() {
			virtiowink := "virtio-win-image"
			updatableKeys := [...]string{virtiowink}
			toBeRemovedKey := "toberemoved"

			expectedResource, err := NewVirtioWinCm(hco)
			Expect(err).ToNot(HaveOccurred())

			expectedResource.ObjectMeta.SelfLink = fmt.Sprintf("/apis/v1/namespaces/%s/dummies/%s", expectedResource.Namespace, expectedResource.Name)
			outdatedResource, err := NewVirtioWinCm(hco)
			Expect(err).ToNot(HaveOccurred())

			outdatedResource.ObjectMeta.SelfLink = fmt.Sprintf("/apis/v1/namespaces/%s/dummies/%s", outdatedResource.Namespace, outdatedResource.Name)
			// values we should update
			outdatedResource.Data[virtiowink] = "old-virtiowin-container-value-we-have-to-update"

			// add values we should remove
			outdatedResource.Data[toBeRemovedKey] = "value-we-should-remove"

			cl := commonTestUtils.InitClient([]runtime.Object{hco, outdatedResource})
			handler, _ := newVirtioWinCmHandler(logger, cl, commonTestUtils.GetScheme(), hco)
			res := handler[0].ensure(req)
			Expect(res.UpgradeDone).To(BeFalse())
			Expect(res.Updated).To(BeTrue())
			Expect(res.Err).To(BeNil())

			foundResource := &corev1.ConfigMap{}
			Expect(
				cl.Get(context.TODO(),
					types.NamespacedName{Name: expectedResource.Name, Namespace: expectedResource.Namespace},
					foundResource),
			).To(BeNil())

			for _, k := range updatableKeys {
				Expect(foundResource.Data[k]).To(Not(Equal(outdatedResource.Data[k])))
				Expect(foundResource.Data[k]).To(Equal(expectedResource.Data[k]))
			}

			Expect(foundResource.Data).To(Not(HaveKey(toBeRemovedKey)))

			// ObjectReference should have been updated
			Expect(hco.Status.RelatedObjects).To(Not(BeNil()))
			objectRefOutdated, err := reference.GetReference(commonTestUtils.GetScheme(), outdatedResource)
			Expect(err).To(BeNil())
			objectRefFound, err := reference.GetReference(commonTestUtils.GetScheme(), foundResource)
			Expect(err).To(BeNil())
			Expect(hco.Status.RelatedObjects).To(Not(ContainElement(*objectRefOutdated)))
			Expect(hco.Status.RelatedObjects).To(ContainElement(*objectRefFound))
		})

	})

	Context("ConfigMap Reader Role", func() {
		var hco *hcov1beta1.HyperConverged
		var req *common.HcoRequest

		BeforeEach(func() {
			os.Setenv("VIRTIOWIN_CONTAINER", "new-virtiowin-container-value")
			hco = commonTestUtils.NewHco()
			req = commonTestUtils.NewReq(hco)
		})
		It("should do nothing if exists", func() {
			expectedRole := NewVirtioWinCmReaderRole(hco)
			cl := commonTestUtils.InitClient([]runtime.Object{hco, expectedRole})

			handler, _ := newVirtioWinCmReaderRoleHandler(logger, cl, commonTestUtils.GetScheme(), hco)
			res := handler[0].ensure(req)
			Expect(res.Err).ToNot(HaveOccurred())

			foundRole := &rbacv1.Role{}
			Expect(
				cl.Get(context.TODO(),
					types.NamespacedName{Name: expectedRole.Name, Namespace: expectedRole.Namespace},
					foundRole),
			).ToNot(HaveOccurred())

			Expect(expectedRole.ObjectMeta).Should(Equal(foundRole.ObjectMeta))
			Expect(expectedRole.Rules).Should(Equal(foundRole.Rules))
		})

		It("should update if labels are missing", func() {
			expectedRole := NewVirtioWinCmReaderRole(hco)
			expectedLabels := expectedRole.Labels
			expectedRole.Labels = nil

			cl := commonTestUtils.InitClient([]runtime.Object{hco, expectedRole})

			handler, _ := newVirtioWinCmReaderRoleHandler(logger, cl, commonTestUtils.GetScheme(), hco)
			res := handler[0].ensure(req)
			Expect(res.Err).ToNot(HaveOccurred())

			foundRole := &rbacv1.Role{}
			Expect(
				cl.Get(context.TODO(),
					types.NamespacedName{Name: expectedRole.Name, Namespace: expectedRole.Namespace},
					foundRole),
			).ToNot(HaveOccurred())

			Expect(reflect.DeepEqual(expectedLabels, foundRole.Labels)).To(BeTrue())
		})
	})

	Context("ConfigMap Reader Role Binding", func() {
		var hco *hcov1beta1.HyperConverged
		var req *common.HcoRequest

		BeforeEach(func() {
			os.Setenv("VIRTIOWIN_CONTAINER", "new-virtiowin-container-value")
			hco = commonTestUtils.NewHco()
			req = commonTestUtils.NewReq(hco)
		})
		It("should do nothing if exists", func() {
			expectedRoleBinding := NewVirtioWinCmReaderRoleBinding(hco)

			cl := commonTestUtils.InitClient([]runtime.Object{hco, expectedRoleBinding})

			handler, _ := newVirtioWinCmReaderRoleBindingHandler(logger, cl, commonTestUtils.GetScheme(), hco)
			res := handler[0].ensure(req)
			Expect(res.Err).ToNot(HaveOccurred())

			foundRoleBinding := &rbacv1.RoleBinding{}
			Expect(
				cl.Get(context.TODO(),
					types.NamespacedName{Name: expectedRoleBinding.Name, Namespace: expectedRoleBinding.Namespace},
					foundRoleBinding),
			).ToNot(HaveOccurred())

			Expect(reflect.DeepEqual(expectedRoleBinding.Labels, foundRoleBinding.Labels)).To(BeTrue())
		})

		It("should update if labels are missing", func() {
			expectedRoleBinding := NewVirtioWinCmReaderRoleBinding(hco)
			expectedLabels := expectedRoleBinding.Labels
			expectedRoleBinding.Labels = nil

			cl := commonTestUtils.InitClient([]runtime.Object{hco, expectedRoleBinding})

			handler, _ := newVirtioWinCmReaderRoleBindingHandler(logger, cl, commonTestUtils.GetScheme(), hco)
			res := handler[0].ensure(req)
			Expect(res.Err).ToNot(HaveOccurred())

			foundRoleBinding := &rbacv1.RoleBinding{}
			Expect(
				cl.Get(context.TODO(),
					types.NamespacedName{Name: expectedRoleBinding.Name, Namespace: expectedRoleBinding.Namespace},
					foundRoleBinding),
			).ToNot(HaveOccurred())

			Expect(reflect.DeepEqual(expectedLabels, foundRoleBinding.Labels)).To(BeTrue())
		})
	})
})
