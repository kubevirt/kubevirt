package operands

import (
	"context"
	"fmt"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	consolev1alpha1 "github.com/openshift/api/console/v1alpha1"
	operatorv1 "github.com/openshift/api/operator/v1"
	appsv1 "k8s.io/api/apps/v1"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/tools/reference"

	hcov1beta1 "github.com/kubevirt/hyperconverged-cluster-operator/api/v1beta1"
	"github.com/kubevirt/hyperconverged-cluster-operator/controllers/common"
	"github.com/kubevirt/hyperconverged-cluster-operator/controllers/commonTestUtils"
	hcoutil "github.com/kubevirt/hyperconverged-cluster-operator/pkg/util"
)

var _ = Describe("Kubevirt Console Plugin", func() {
	Context("Console Plugin CR", func() {
		var hco *hcov1beta1.HyperConverged
		var req *common.HcoRequest

		var expectedConsoleConfig = &operatorv1.Console{
			TypeMeta: metav1.TypeMeta{
				Kind:       "Console",
				APIVersion: "operator.openshift.io/v1",
			},
			ObjectMeta: metav1.ObjectMeta{
				Name: "cluster",
			},
		}

		BeforeEach(func() {
			hco = commonTestUtils.NewHco()
			req = commonTestUtils.NewReq(hco)
		})

		It("should create plugin CR if not present", func() {
			expectedResource := NewKvConsolePlugin(hco)
			cl := commonTestUtils.InitClient([]runtime.Object{})
			handler, err := newKvUiPluginCRHandler(logger, cl, commonTestUtils.GetScheme(), hco)
			Expect(err).ToNot(HaveOccurred())

			res := handler[0].ensure(req)
			Expect(res.UpgradeDone).To(BeFalse())
			Expect(res.Err).ToNot(HaveOccurred())

			foundResource := &consolev1alpha1.ConsolePlugin{}
			Expect(
				cl.Get(context.TODO(),
					types.NamespacedName{Name: expectedResource.Name, Namespace: expectedResource.Namespace},
					foundResource),
			).ToNot(HaveOccurred())
			Expect(foundResource.Name).To(Equal(expectedResource.Name))
			Expect(foundResource.Labels).Should(HaveKeyWithValue(hcoutil.AppLabel, commonTestUtils.Name))
			Expect(foundResource.Namespace).To(Equal(expectedResource.Namespace))
		})

		It("should find plugin CR if present", func() {
			expectedResource := NewKvConsolePlugin(hco)

			expectedResource.ObjectMeta.SelfLink = fmt.Sprintf("/apis/v1/namespaces/%s/dummies/%s", expectedResource.Namespace, expectedResource.Name)
			cl := commonTestUtils.InitClient([]runtime.Object{hco, expectedResource, expectedConsoleConfig})
			handler, err := newKvUiPluginCRHandler(logger, cl, commonTestUtils.GetScheme(), hco)
			Expect(err).ToNot(HaveOccurred())

			res := handler[0].ensure(req)
			Expect(res.UpgradeDone).To(BeFalse())
			Expect(res.Err).ToNot(HaveOccurred())

			// Check HCO's status
			Expect(hco.Status.RelatedObjects).To(Not(BeNil()))
			objectRef, err := reference.GetReference(commonTestUtils.GetScheme(), expectedResource)
			Expect(err).ToNot(HaveOccurred())
			// ObjectReference should have been added
			Expect(hco.Status.RelatedObjects).To(ContainElement(*objectRef))
		})

		It("should reconcile plugin to default if changed", func() {
			expectedResource := NewKvConsolePlugin(hco)
			outdatedResource := NewKvConsolePlugin(hco)

			outdatedResource.Spec.Service.Port = int32(6666)
			outdatedResource.Spec.Service.BasePath = "/fakepath"
			outdatedResource.Spec.DisplayName = "fake plugin name"

			cl := commonTestUtils.InitClient([]runtime.Object{hco, outdatedResource, expectedConsoleConfig})
			handler, err := newKvUiPluginCRHandler(logger, cl, commonTestUtils.GetScheme(), hco)
			Expect(err).ToNot(HaveOccurred())

			res := handler[0].ensure(req)
			Expect(res.UpgradeDone).To(BeFalse())
			Expect(res.Updated).To(BeTrue())
			Expect(res.Err).ToNot(HaveOccurred())

			foundResource := &consolev1alpha1.ConsolePlugin{}
			Expect(
				cl.Get(context.TODO(),
					types.NamespacedName{Name: expectedResource.Name, Namespace: expectedResource.Namespace},
					foundResource),
			).ToNot(HaveOccurred())

			Expect(foundResource.Spec.Service.Port).ToNot(Equal(outdatedResource.Spec.Service.Port))
			Expect(foundResource.Spec.Service.Port).To(Equal(int32(hcoutil.UiPluginServerPort)))
			Expect(foundResource.Spec.Service.BasePath).ToNot(Equal(outdatedResource.Spec.Service.BasePath))
			Expect(foundResource.Spec.Service.BasePath).To(Equal(expectedResource.Spec.Service.BasePath))
			Expect(foundResource.Spec.DisplayName).ToNot(Equal(outdatedResource.Spec.DisplayName))
			Expect(foundResource.Spec.DisplayName).To(Equal(expectedResource.Spec.DisplayName))

			// ObjectReference should have been updated
			Expect(hco.Status.RelatedObjects).To(Not(BeNil()))
			objectRefOutdated, err := reference.GetReference(commonTestUtils.GetScheme(), outdatedResource)
			Expect(err).ToNot(HaveOccurred())
			objectRefFound, err := reference.GetReference(commonTestUtils.GetScheme(), foundResource)
			Expect(err).ToNot(HaveOccurred())
			Expect(hco.Status.RelatedObjects).To(Not(ContainElement(*objectRefOutdated)))
			Expect(hco.Status.RelatedObjects).To(ContainElement(*objectRefFound))
		})
	})

	Context("Kubevirt Plugin Deployment", func() {
		var hco *hcov1beta1.HyperConverged
		var req *common.HcoRequest

		BeforeEach(func() {
			hco = commonTestUtils.NewHco()
			req = commonTestUtils.NewReq(hco)
		})

		It("should create if not present", func() {
			expectedResource, err := NewKvUiPluginDeplymnt(hco)
			Expect(err).ToNot(HaveOccurred())

			cl := commonTestUtils.InitClient([]runtime.Object{})
			handler, err := newKvUiPluginDplymntHandler(logger, cl, commonTestUtils.GetScheme(), hco)
			Expect(err).ToNot(HaveOccurred())

			res := handler[0].ensure(req)
			Expect(res.UpgradeDone).To(BeFalse())
			Expect(res.Err).ToNot(HaveOccurred())

			foundResource := &appsv1.Deployment{}
			Expect(
				cl.Get(context.TODO(),
					types.NamespacedName{Name: expectedResource.Name, Namespace: expectedResource.Namespace},
					foundResource),
			).ToNot(HaveOccurred())
			Expect(foundResource.Name).To(Equal(expectedResource.Name))
			Expect(foundResource.Labels).Should(HaveKeyWithValue(hcoutil.AppLabel, commonTestUtils.Name))
			Expect(foundResource.Namespace).To(Equal(expectedResource.Namespace))
		})

		It("should find plugin deployment if present", func() {
			expectedResource, err := NewKvUiPluginDeplymnt(hco)
			Expect(err).ToNot(HaveOccurred())

			expectedResource.ObjectMeta.SelfLink = fmt.Sprintf("/apis/v1/namespaces/%s/dummies/%s", expectedResource.Namespace, expectedResource.Name)
			cl := commonTestUtils.InitClient([]runtime.Object{hco, expectedResource})
			handler, err := newKvUiPluginDplymntHandler(logger, cl, commonTestUtils.GetScheme(), hco)
			Expect(err).ToNot(HaveOccurred())

			res := handler[0].ensure(req)
			Expect(res.UpgradeDone).To(BeFalse())
			Expect(res.Err).ToNot(HaveOccurred())

			foundResource := &appsv1.Deployment{}
			Expect(
				cl.Get(context.TODO(),
					types.NamespacedName{Name: expectedResource.Name, Namespace: expectedResource.Namespace},
					foundResource),
			).ToNot(HaveOccurred())

			// Check HCO's status
			Expect(hco.Status.RelatedObjects).To(Not(BeNil()))
			objectRef, err := reference.GetReference(commonTestUtils.GetScheme(), foundResource)
			Expect(err).ToNot(HaveOccurred())
			// ObjectReference should have been added
			Expect(hco.Status.RelatedObjects).To(ContainElement(*objectRef))
		})

		It("should reconcile deployment to default if changed", func() {
			expectedResource, _ := NewKvUiPluginDeplymnt(hco)
			outdatedResource, _ := NewKvUiPluginDeplymnt(hco)

			outdatedResource.ObjectMeta.Labels[hcoutil.AppLabel] = "wrong label"
			outdatedResource.Spec.Template.Spec.Containers[0].Image = "quay.io/fake/image:latest"

			cl := commonTestUtils.InitClient([]runtime.Object{hco, outdatedResource})
			handler, err := newKvUiPluginDplymntHandler(logger, cl, commonTestUtils.GetScheme(), hco)
			Expect(err).ToNot(HaveOccurred())
			res := handler[0].ensure(req)
			Expect(res.UpgradeDone).To(BeFalse())
			Expect(res.Updated).To(BeTrue())
			Expect(res.Err).ToNot(HaveOccurred())

			foundResource := &appsv1.Deployment{}
			Expect(
				cl.Get(context.TODO(),
					types.NamespacedName{Name: expectedResource.Name, Namespace: expectedResource.Namespace},
					foundResource),
			).ToNot(HaveOccurred())

			Expect(foundResource.ObjectMeta.Labels).ToNot(Equal(outdatedResource.ObjectMeta.Labels))
			Expect(foundResource.ObjectMeta.Labels).To(Equal(expectedResource.ObjectMeta.Labels))
			Expect(foundResource.Spec.Template.Spec.Containers[0].Image).ToNot(Equal(outdatedResource.Spec.Template.Spec.Containers[0].Image))
			Expect(foundResource.Spec.Template.Spec.Containers[0].Image).To(Equal(expectedResource.Spec.Template.Spec.Containers[0].Image))

			// ObjectReference should have been updated
			Expect(hco.Status.RelatedObjects).To(Not(BeNil()))
			objectRefOutdated, err := reference.GetReference(commonTestUtils.GetScheme(), outdatedResource)
			Expect(err).ToNot(HaveOccurred())
			objectRefFound, err := reference.GetReference(commonTestUtils.GetScheme(), foundResource)
			Expect(err).ToNot(HaveOccurred())
			Expect(hco.Status.RelatedObjects).To(Not(ContainElement(*objectRefOutdated)))
			Expect(hco.Status.RelatedObjects).To(ContainElement(*objectRefFound))
		})
	})

	Context("Kubevirt Plugin Service", func() {
		var hco *hcov1beta1.HyperConverged
		var req *common.HcoRequest

		BeforeEach(func() {
			hco = commonTestUtils.NewHco()
			req = commonTestUtils.NewReq(hco)
		})

		It("should create plugin service if not present", func() {
			expectedResource := NewKvUiPluginSvc(hco)
			cl := commonTestUtils.InitClient([]runtime.Object{})
			handler := (*genericOperand)(newServiceHandler(cl, commonTestUtils.GetScheme(), NewKvUiPluginSvc))

			res := handler.ensure(req)
			Expect(res.UpgradeDone).To(BeFalse())
			Expect(res.Err).ToNot(HaveOccurred())

			foundResource := &v1.Service{}
			Expect(
				cl.Get(context.TODO(),
					types.NamespacedName{Name: expectedResource.Name, Namespace: expectedResource.Namespace},
					foundResource),
			).ToNot(HaveOccurred())
			Expect(foundResource.Name).To(Equal(expectedResource.Name))
			Expect(foundResource.Labels).Should(HaveKeyWithValue(hcoutil.AppLabel, commonTestUtils.Name))
			Expect(foundResource.Namespace).To(Equal(expectedResource.Namespace))
		})

		It("should find plugin service if present", func() {
			expectedResource := NewKvUiPluginSvc(hco)

			expectedResource.ObjectMeta.SelfLink = fmt.Sprintf("/apis/v1/namespaces/%s/dummies/%s", expectedResource.Namespace, expectedResource.Name)
			cl := commonTestUtils.InitClient([]runtime.Object{hco, expectedResource})
			handler := (*genericOperand)(newServiceHandler(cl, commonTestUtils.GetScheme(), NewKvUiPluginSvc))

			res := handler.ensure(req)
			Expect(res.UpgradeDone).To(BeFalse())
			Expect(res.Err).ToNot(HaveOccurred())

			foundResource := &v1.Service{}
			Expect(
				cl.Get(context.TODO(),
					types.NamespacedName{Name: expectedResource.Name, Namespace: expectedResource.Namespace},
					foundResource),
			).ToNot(HaveOccurred())

			// Check HCO's status
			Expect(hco.Status.RelatedObjects).To(Not(BeNil()))
			objectRef, err := reference.GetReference(commonTestUtils.GetScheme(), foundResource)
			Expect(err).ToNot(HaveOccurred())
			// ObjectReference should have been added
			Expect(hco.Status.RelatedObjects).To(ContainElement(*objectRef))
		})

		It("should reconcile service to default if changed", func() {
			expectedResource := NewKvUiPluginSvc(hco)
			outdatedResource := NewKvUiPluginSvc(hco)

			outdatedResource.ObjectMeta.Labels[hcoutil.AppLabel] = "wrong label"
			outdatedResource.Spec.Ports[0].Port = 6666

			cl := commonTestUtils.InitClient([]runtime.Object{hco, outdatedResource})
			handler := (*genericOperand)(newServiceHandler(cl, commonTestUtils.GetScheme(), NewKvUiPluginSvc))
			res := handler.ensure(req)
			Expect(res.UpgradeDone).To(BeFalse())
			Expect(res.Updated).To(BeTrue())
			Expect(res.Err).ToNot(HaveOccurred())

			foundResource := &v1.Service{}
			Expect(
				cl.Get(context.TODO(),
					types.NamespacedName{Name: expectedResource.Name, Namespace: expectedResource.Namespace},
					foundResource),
			).ToNot(HaveOccurred())

			Expect(foundResource.ObjectMeta.Labels).ToNot(Equal(outdatedResource.ObjectMeta.Labels))
			Expect(foundResource.ObjectMeta.Labels).To(Equal(expectedResource.ObjectMeta.Labels))
			Expect(foundResource.Spec.Ports).ToNot(Equal(outdatedResource.Spec.Ports))
			Expect(foundResource.Spec.Ports).To(Equal(expectedResource.Spec.Ports))

			// ObjectReference should have been updated
			Expect(hco.Status.RelatedObjects).To(Not(BeNil()))
			objectRefOutdated, err := reference.GetReference(commonTestUtils.GetScheme(), outdatedResource)
			Expect(err).ToNot(HaveOccurred())
			objectRefFound, err := reference.GetReference(commonTestUtils.GetScheme(), foundResource)
			Expect(err).ToNot(HaveOccurred())
			Expect(hco.Status.RelatedObjects).To(Not(ContainElement(*objectRefOutdated)))
			Expect(hco.Status.RelatedObjects).To(ContainElement(*objectRefFound))
		})
	})

})
