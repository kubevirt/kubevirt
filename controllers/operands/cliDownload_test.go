package operands

import (
	"context"
	"reflect"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/ginkgo/extensions/table"
	. "github.com/onsi/gomega"
	consolev1 "github.com/openshift/api/console/v1"
	routev1 "github.com/openshift/api/route/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/intstr"
	"k8s.io/client-go/tools/reference"
	"sigs.k8s.io/controller-runtime/pkg/client"

	hcov1beta1 "github.com/kubevirt/hyperconverged-cluster-operator/api/v1beta1"
	"github.com/kubevirt/hyperconverged-cluster-operator/controllers/common"
	"github.com/kubevirt/hyperconverged-cluster-operator/controllers/commonTestUtils"
	hcoutil "github.com/kubevirt/hyperconverged-cluster-operator/pkg/util"
)

var _ = Describe("CLI Download", func() {
	Context("ConsoleCLIDownload", func() {

		var hco *hcov1beta1.HyperConverged
		var req *common.HcoRequest

		BeforeEach(func() {
			hco = commonTestUtils.NewHco()
			req = commonTestUtils.NewReq(hco)
		})

		It("should create if not present", func() {
			expectedResource := NewConsoleCLIDownload(hco)
			cl := commonTestUtils.InitClient([]runtime.Object{})
			handler := (*genericOperand)(newCliDownloadHandler(cl, commonTestUtils.GetScheme()))
			res := handler.ensure(req)
			Expect(res.Err).To(BeNil())

			key := client.ObjectKeyFromObject(expectedResource)
			foundResource := &consolev1.ConsoleCLIDownload{}
			Expect(cl.Get(context.TODO(), key, foundResource)).To(BeNil())
			Expect(foundResource.Name).To(Equal(expectedResource.Name))
			Expect(foundResource.Labels).Should(HaveKeyWithValue(hcoutil.AppLabel, commonTestUtils.Name))
			Expect(foundResource.Spec.Links).Should(HaveLen(3))
			Expect(foundResource.Namespace).To(Equal(expectedResource.Namespace))
		})

		It("should find if present", func() {
			expectedResource := NewConsoleCLIDownload(hco)
			cl := commonTestUtils.InitClient([]runtime.Object{hco, expectedResource})
			handler := (*genericOperand)(newCliDownloadHandler(cl, commonTestUtils.GetScheme()))
			res := handler.ensure(req)
			Expect(res.Err).To(BeNil())

			// Check HCO's status
			Expect(hco.Status.RelatedObjects).To(Not(BeNil()))
			objectRef, err := reference.GetReference(handler.Scheme, expectedResource)
			Expect(err).To(BeNil())
			// ObjectReference should have been added
			Expect(hco.Status.RelatedObjects).To(ContainElement(*objectRef))
		})

		DescribeTable("should update if something changed", func(modify func(resource *consolev1.ConsoleCLIDownload)) {
			expectedResource := NewConsoleCLIDownload(hco)
			modifiedResource := &consolev1.ConsoleCLIDownload{}
			expectedResource.DeepCopyInto(modifiedResource)
			modify(modifiedResource)

			cl := commonTestUtils.InitClient([]runtime.Object{modifiedResource})
			handler := (*genericOperand)(newCliDownloadHandler(cl, commonTestUtils.GetScheme()))
			res := handler.ensure(req)
			Expect(res.Err).To(BeNil())

			key := client.ObjectKeyFromObject(expectedResource)
			foundResource := &consolev1.ConsoleCLIDownload{}
			Expect(cl.Get(context.TODO(), key, foundResource))
			Expect(reflect.DeepEqual(expectedResource.Spec, foundResource.Spec)).To(BeTrue())

			// ObjectReference should have been updated
			Expect(hco.Status.RelatedObjects).To(Not(BeNil()))
			objectRefOutdated, err := reference.GetReference(handler.Scheme, modifiedResource)
			Expect(err).To(BeNil())
			objectRefFound, err := reference.GetReference(handler.Scheme, foundResource)
			Expect(err).To(BeNil())
			Expect(hco.Status.RelatedObjects).To(Not(ContainElement(*objectRefOutdated)))
			Expect(hco.Status.RelatedObjects).To(ContainElement(*objectRefFound))
		},
			Entry("with modified description", func(resource *consolev1.ConsoleCLIDownload) {
				resource.Spec.Description = "different text"
			}),
			Entry("with modified display name", func(resource *consolev1.ConsoleCLIDownload) {
				resource.Spec.DisplayName = "different text"
			}),
			Entry("with modified links", func(resource *consolev1.ConsoleCLIDownload) {
				resource.Spec.Links = []consolev1.CLIDownloadLink{{Text: "text", Href: "href"}}
			}),
			Entry("with modified labels", func(resource *consolev1.ConsoleCLIDownload) {
				resource.Labels = map[string]string{"key": "value"}
			}),
		)

	})
})

var _ = Describe("Downloads Service", func() {
	Context("Downloads Service", func() {

		var hco *hcov1beta1.HyperConverged
		var req *common.HcoRequest

		BeforeEach(func() {
			hco = commonTestUtils.NewHco()
			req = commonTestUtils.NewReq(hco)
		})

		It("should create if not present", func() {
			expectedResource := NewCliDownloadsService(hco)
			cl := commonTestUtils.InitClient([]runtime.Object{})
			handler := (*genericOperand)(newCliDownloadsServiceHandler(cl, commonTestUtils.GetScheme()))
			res := handler.ensure(req)
			Expect(res.Err).To(BeNil())

			key := client.ObjectKeyFromObject(expectedResource)
			foundResource := &corev1.Service{}
			Expect(cl.Get(context.TODO(), key, foundResource)).To(BeNil())
			Expect(foundResource.Name).To(Equal(expectedResource.Name))
			Expect(foundResource.Labels).Should(HaveKeyWithValue(hcoutil.AppLabel, commonTestUtils.Name))
			Expect(foundResource.Namespace).To(Equal(expectedResource.Namespace))
		})

		It("should find if present", func() {
			expectedResource := NewCliDownloadsService(hco)
			cl := commonTestUtils.InitClient([]runtime.Object{hco, expectedResource})
			handler := (*genericOperand)(newCliDownloadsServiceHandler(cl, commonTestUtils.GetScheme()))
			res := handler.ensure(req)
			Expect(res.Err).To(BeNil())

			// Check HCO's status
			Expect(hco.Status.RelatedObjects).To(Not(BeNil()))
			objectRef, err := reference.GetReference(handler.Scheme, expectedResource)
			Expect(err).To(BeNil())
			// ObjectReference should have been added
			Expect(hco.Status.RelatedObjects).To(ContainElement(*objectRef))
		})

		DescribeTable("should update if something changed", func(modify func(resource *corev1.Service)) {
			expectedResource := NewCliDownloadsService(hco)
			modifiedResource := &corev1.Service{}
			expectedResource.DeepCopyInto(modifiedResource)
			modify(modifiedResource)

			cl := commonTestUtils.InitClient([]runtime.Object{modifiedResource})
			handler := (*genericOperand)(newCliDownloadsServiceHandler(cl, commonTestUtils.GetScheme()))
			res := handler.ensure(req)
			Expect(res.Err).To(BeNil())

			key := client.ObjectKeyFromObject(expectedResource)
			foundResource := &corev1.Service{}
			Expect(cl.Get(context.TODO(), key, foundResource))
			Expect(hasServiceRightFields(foundResource, expectedResource)).To(BeTrue())

			// ObjectReference should have been updated
			Expect(hco.Status.RelatedObjects).To(Not(BeNil()))
			objectRefOutdated, err := reference.GetReference(handler.Scheme, modifiedResource)
			Expect(err).To(BeNil())
			objectRefFound, err := reference.GetReference(handler.Scheme, foundResource)
			Expect(err).To(BeNil())
			Expect(hco.Status.RelatedObjects).To(Not(ContainElement(*objectRefOutdated)))
			Expect(hco.Status.RelatedObjects).To(ContainElement(*objectRefFound))
		},
			Entry("with modified selector", func(resource *corev1.Service) {
				resource.Spec.Selector = map[string]string{"key": "value"}
			}),
			Entry("with modified labels", func(resource *corev1.Service) {
				resource.Labels = map[string]string{"key": "value"}
			}),
			Entry("with modified ports", func(resource *corev1.Service) {
				resource.Spec.Ports = []corev1.ServicePort{{Port: 1111, Protocol: corev1.ProtocolUDP}}
			}),
		)

	})
})

var _ = Describe("Cli Downloads Route", func() {
	Context("Cli Downloads Route", func() {

		var hco *hcov1beta1.HyperConverged
		var req *common.HcoRequest

		BeforeEach(func() {
			hco = commonTestUtils.NewHco()
			req = commonTestUtils.NewReq(hco)
		})

		It("should create if not present", func() {
			expectedResource := NewCliDownloadsRoute(hco)
			cl := commonTestUtils.InitClient([]runtime.Object{})
			handler := (*genericOperand)(newCliDownloadsRouteHandler(cl, commonTestUtils.GetScheme()))
			res := handler.ensure(req)
			Expect(res.Err).To(BeNil())

			key := client.ObjectKeyFromObject(expectedResource)
			foundResource := &routev1.Route{}
			Expect(cl.Get(context.TODO(), key, foundResource)).To(BeNil())
			Expect(foundResource.Name).To(Equal(expectedResource.Name))
			Expect(foundResource.Labels).Should(HaveKeyWithValue(hcoutil.AppLabel, commonTestUtils.Name))
			Expect(foundResource.Namespace).To(Equal(expectedResource.Namespace))
		})

		It("should find if present", func() {
			expectedResource := NewCliDownloadsRoute(hco)
			cl := commonTestUtils.InitClient([]runtime.Object{hco, expectedResource})
			handler := (*genericOperand)(newCliDownloadsRouteHandler(cl, commonTestUtils.GetScheme()))
			res := handler.ensure(req)
			Expect(res.Err).To(BeNil())

			// Check HCO's status
			Expect(hco.Status.RelatedObjects).To(Not(BeNil()))
			objectRef, err := reference.GetReference(handler.Scheme, expectedResource)
			Expect(err).To(BeNil())
			// ObjectReference should have been added
			Expect(hco.Status.RelatedObjects).To(ContainElement(*objectRef))
		})

		DescribeTable("should update if something changed", func(modify func(resource *routev1.Route)) {
			expectedResource := NewCliDownloadsRoute(hco)
			modifiedResource := &routev1.Route{}
			expectedResource.DeepCopyInto(modifiedResource)
			modify(modifiedResource)

			cl := commonTestUtils.InitClient([]runtime.Object{modifiedResource})
			handler := (*genericOperand)(newCliDownloadsRouteHandler(cl, commonTestUtils.GetScheme()))
			res := handler.ensure(req)
			Expect(res.Err).To(BeNil())

			key := client.ObjectKeyFromObject(expectedResource)
			foundResource := &routev1.Route{}
			Expect(cl.Get(context.TODO(), key, foundResource))
			Expect(hasRouteRightFields(foundResource, expectedResource)).To(BeTrue())

			// ObjectReference should have been updated
			Expect(hco.Status.RelatedObjects).To(Not(BeNil()))
			objectRefOutdated, err := reference.GetReference(handler.Scheme, modifiedResource)
			Expect(err).To(BeNil())
			objectRefFound, err := reference.GetReference(handler.Scheme, foundResource)
			Expect(err).To(BeNil())
			Expect(hco.Status.RelatedObjects).To(Not(ContainElement(*objectRefOutdated)))
			Expect(hco.Status.RelatedObjects).To(ContainElement(*objectRefFound))
		},
			Entry("with modified labels", func(resource *routev1.Route) {
				resource.Labels = map[string]string{"key": "value"}
			}),
			Entry("with modified port", func(resource *routev1.Route) {
				resource.Spec.Port = &routev1.RoutePort{
					TargetPort: intstr.IntOrString{IntVal: 1111},
				}
			}),
			Entry("with modified tls", func(resource *routev1.Route) {
				resource.Spec.TLS = &routev1.TLSConfig{
					Termination: routev1.TLSTerminationReencrypt,
				}
			}),
			Entry("with modified target reference", func(resource *routev1.Route) {
				resource.Spec.To = routev1.RouteTargetReference{
					Kind: "Service",
					Name: "test-service",
				}
			}),
		)

	})
})
