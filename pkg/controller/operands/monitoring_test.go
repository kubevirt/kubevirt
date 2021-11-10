package operands

import (
	"context"
	"fmt"

	. "github.com/onsi/ginkgo"
	monitoringv1 "github.com/prometheus-operator/prometheus-operator/pkg/apis/monitoring/v1"

	hcov1beta1 "github.com/kubevirt/hyperconverged-cluster-operator/pkg/apis/hco/v1beta1"
	"github.com/kubevirt/hyperconverged-cluster-operator/pkg/controller/common"
	"github.com/kubevirt/hyperconverged-cluster-operator/pkg/controller/commonTestUtils"
	hcoutil "github.com/kubevirt/hyperconverged-cluster-operator/pkg/util"

	. "github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/tools/reference"
)

var _ = Describe("Monitoring Operand", func() {
	Context("Metrics Service", func() {

		var hco *hcov1beta1.HyperConverged
		var req *common.HcoRequest

		BeforeEach(func() {
			hco = commonTestUtils.NewHco()
			req = commonTestUtils.NewReq(hco)
		})

		It("should create if not present", func() {
			expectedResource := NewMetricsService(hco, commonTestUtils.Namespace)
			cl := commonTestUtils.InitClient([]runtime.Object{})
			handler := (*genericOperand)(newMetricsServiceHandler(cl, commonTestUtils.GetScheme()))
			res := handler.ensure(req)
			Expect(res.Created).To(BeTrue())
			Expect(res.Updated).To(BeFalse())
			Expect(res.Overwritten).To(BeFalse())
			Expect(res.UpgradeDone).To(BeFalse())
			Expect(res.Err).To(BeNil())

			foundResource := &corev1.Service{}
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
			expectedResource := NewMetricsService(hco, commonTestUtils.Namespace)
			cl := commonTestUtils.InitClient([]runtime.Object{expectedResource})
			handler := (*genericOperand)(newMetricsServiceHandler(cl, commonTestUtils.GetScheme()))
			res := handler.ensure(req)
			Expect(res.Created).To(BeFalse())
			Expect(res.Updated).To(BeFalse())
			Expect(res.Overwritten).To(BeFalse())
			Expect(res.UpgradeDone).To(BeFalse())
			Expect(res.Err).To(BeNil())

			objectRef, err := reference.GetReference(handler.Scheme, expectedResource)
			Expect(err).To(BeNil())
			Expect(hco.Status.RelatedObjects).To(ContainElement(*objectRef))
		})

		It("should reconcile to default", func() {
			existingResource := NewMetricsService(hco, commonTestUtils.Namespace)
			existingResource.ObjectMeta.SelfLink = fmt.Sprintf("/apis/v1/namespaces/%s/dummies/%s", existingResource.Namespace, existingResource.Name)

			existingResource.Spec.Ports[0].Name = "Non default value"
			existingResource.Spec.Ports[0].Port = 0
			req.HCOTriggered = false

			cl := commonTestUtils.InitClient([]runtime.Object{hco, existingResource})
			handler := (*genericOperand)(newMetricsServiceHandler(cl, commonTestUtils.GetScheme()))
			res := handler.ensure(req)
			Expect(res.Created).To(BeFalse())
			Expect(res.Updated).To(BeTrue())
			Expect(res.Overwritten).To(BeTrue())
			Expect(res.UpgradeDone).To(BeFalse())
			Expect(res.Err).To(BeNil())

			foundResource := &corev1.Service{}
			Expect(
				cl.Get(context.TODO(),
					types.NamespacedName{Name: existingResource.Name, Namespace: existingResource.Namespace},
					foundResource),
			).To(BeNil())
			Expect(foundResource.Spec.Ports[0].Name).To(BeIdenticalTo(operatorPortName))
			Expect(foundResource.Spec.Ports[0].Port).To(BeIdenticalTo(hcoutil.MetricsPort))

			// ObjectReference should have been updated
			Expect(hco.Status.RelatedObjects).To(Not(BeNil()))
			objectRefOutdated, err := reference.GetReference(handler.Scheme, existingResource)
			Expect(err).To(BeNil())
			objectRefFound, err := reference.GetReference(handler.Scheme, foundResource)
			Expect(err).To(BeNil())
			Expect(hco.Status.RelatedObjects).To(Not(ContainElement(*objectRefOutdated)))
			Expect(hco.Status.RelatedObjects).To(ContainElement(*objectRefFound))

		})

	})

	Context("Service Monitor", func() {

		var hco *hcov1beta1.HyperConverged
		var req *common.HcoRequest

		BeforeEach(func() {
			hco = commonTestUtils.NewHco()
			req = commonTestUtils.NewReq(hco)
		})

		It("should create if not present", func() {
			expectedResource := NewServiceMonitor(hco, commonTestUtils.Namespace)
			cl := commonTestUtils.InitClient([]runtime.Object{})
			handler := (*genericOperand)(newMetricsServiceMonitorHandler(cl, commonTestUtils.GetScheme()))
			res := handler.ensure(req)
			Expect(res.Created).To(BeTrue())
			Expect(res.Updated).To(BeFalse())
			Expect(res.Overwritten).To(BeFalse())
			Expect(res.UpgradeDone).To(BeFalse())
			Expect(res.Err).To(BeNil())

			foundResource := &monitoringv1.ServiceMonitor{}
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
			expectedResource := NewServiceMonitor(hco, commonTestUtils.Namespace)
			cl := commonTestUtils.InitClient([]runtime.Object{expectedResource})
			handler := (*genericOperand)(newMetricsServiceMonitorHandler(cl, commonTestUtils.GetScheme()))
			res := handler.ensure(req)
			Expect(res.Created).To(BeFalse())
			Expect(res.Updated).To(BeFalse())
			Expect(res.Overwritten).To(BeFalse())
			Expect(res.UpgradeDone).To(BeFalse())
			Expect(res.Err).To(BeNil())

			objectRef, err := reference.GetReference(handler.Scheme, expectedResource)
			Expect(err).To(BeNil())
			Expect(hco.Status.RelatedObjects).To(ContainElement(*objectRef))
		})

		It("should reconcile to default", func() {
			existingResource := NewServiceMonitor(hco, commonTestUtils.Namespace)
			existingResource.ObjectMeta.SelfLink = fmt.Sprintf("/apis/v1/namespaces/%s/dummies/%s", existingResource.Namespace, existingResource.Name)

			existingResource.Spec.Endpoints[0].Port = "Non default value"
			req.HCOTriggered = false

			cl := commonTestUtils.InitClient([]runtime.Object{hco, existingResource})
			handler := (*genericOperand)(newMetricsServiceMonitorHandler(cl, commonTestUtils.GetScheme()))
			res := handler.ensure(req)
			Expect(res.Created).To(BeFalse())
			Expect(res.Updated).To(BeTrue())
			Expect(res.Overwritten).To(BeTrue())
			Expect(res.UpgradeDone).To(BeFalse())
			Expect(res.Err).To(BeNil())

			foundResource := &monitoringv1.ServiceMonitor{}
			Expect(
				cl.Get(context.TODO(),
					types.NamespacedName{Name: existingResource.Name, Namespace: existingResource.Namespace},
					foundResource),
			).To(BeNil())
			Expect(foundResource.Spec.Endpoints[0].Port).To(BeIdenticalTo(operatorPortName))

			// ObjectReference should have been updated
			Expect(hco.Status.RelatedObjects).To(Not(BeNil()))
			objectRefOutdated, err := reference.GetReference(handler.Scheme, existingResource)
			Expect(err).To(BeNil())
			objectRefFound, err := reference.GetReference(handler.Scheme, foundResource)
			Expect(err).To(BeNil())
			Expect(hco.Status.RelatedObjects).To(Not(ContainElement(*objectRefOutdated)))
			Expect(hco.Status.RelatedObjects).To(ContainElement(*objectRefFound))
		})

	})

	Context("Prometheus rule", func() {

		var hco *hcov1beta1.HyperConverged
		var req *common.HcoRequest

		BeforeEach(func() {
			hco = commonTestUtils.NewHco()
			req = commonTestUtils.NewReq(hco)
		})

		It("should create if not present", func() {
			expectedResource := NewPrometheusRule(hco, commonTestUtils.Namespace)
			cl := commonTestUtils.InitClient([]runtime.Object{})
			handler := (*genericOperand)(newMonitoringPrometheusRuleHandler(cl, commonTestUtils.GetScheme()))
			res := handler.ensure(req)
			Expect(res.Created).To(BeTrue())
			Expect(res.Updated).To(BeFalse())
			Expect(res.Overwritten).To(BeFalse())
			Expect(res.UpgradeDone).To(BeFalse())
			Expect(res.Err).To(BeNil())

			foundResource := &monitoringv1.PrometheusRule{}
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
			expectedResource := NewPrometheusRule(hco, commonTestUtils.Namespace)
			cl := commonTestUtils.InitClient([]runtime.Object{expectedResource})
			handler := (*genericOperand)(newMonitoringPrometheusRuleHandler(cl, commonTestUtils.GetScheme()))
			res := handler.ensure(req)
			Expect(res.Created).To(BeFalse())
			Expect(res.Updated).To(BeFalse())
			Expect(res.Overwritten).To(BeFalse())
			Expect(res.UpgradeDone).To(BeFalse())
			Expect(res.Err).To(BeNil())

			objectRef, err := reference.GetReference(handler.Scheme, expectedResource)
			Expect(err).To(BeNil())
			Expect(hco.Status.RelatedObjects).To(ContainElement(*objectRef))
		})

		It("should reconcile to default", func() {
			existingResource := NewPrometheusRule(hco, commonTestUtils.Namespace)
			existingResource.ObjectMeta.SelfLink = fmt.Sprintf("/apis/v1/namespaces/%s/dummies/%s", existingResource.Namespace, existingResource.Name)

			existingResource.Spec.Groups[0].Name = "Non default value"
			existingResource.Spec.Groups[0].Rules[0].Alert = "Non default value"
			req.HCOTriggered = false

			cl := commonTestUtils.InitClient([]runtime.Object{hco, existingResource})
			handler := (*genericOperand)(newMonitoringPrometheusRuleHandler(cl, commonTestUtils.GetScheme()))
			res := handler.ensure(req)
			Expect(res.Created).To(BeFalse())
			Expect(res.Updated).To(BeTrue())
			Expect(res.Overwritten).To(BeTrue())
			Expect(res.UpgradeDone).To(BeFalse())
			Expect(res.Err).To(BeNil())

			foundResource := &monitoringv1.PrometheusRule{}
			Expect(
				cl.Get(context.TODO(),
					types.NamespacedName{Name: existingResource.Name, Namespace: existingResource.Namespace},
					foundResource),
			).To(BeNil())
			Expect(foundResource.Spec.Groups[0].Name).To(BeIdenticalTo(alertRuleGroup))
			Expect(foundResource.Spec.Groups[0].Rules[0].Alert).To(BeIdenticalTo(outOfBandUpdateAlert))

			// ObjectReference should have been updated
			Expect(hco.Status.RelatedObjects).To(Not(BeNil()))
			objectRefOutdated, err := reference.GetReference(handler.Scheme, existingResource)
			Expect(err).To(BeNil())
			objectRefFound, err := reference.GetReference(handler.Scheme, foundResource)
			Expect(err).To(BeNil())
			Expect(hco.Status.RelatedObjects).To(Not(ContainElement(*objectRefOutdated)))
			Expect(hco.Status.RelatedObjects).To(ContainElement(*objectRefFound))
		})

	})

})
