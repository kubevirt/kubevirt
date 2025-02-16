package reqresolver_test

import (
	"os"
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	"github.com/kubevirt/hyperconverged-cluster-operator/controllers/reqresolver"
	hcoutil "github.com/kubevirt/hyperconverged-cluster-operator/pkg/util"
)

const (
	namespace = "test-ns"
)

func TestReqResolver(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "ReqResolver Suite")
}

var (
	nsBefore string

	_ = BeforeSuite(func() {
		nsBefore = hcoutil.GetOperatorNamespaceFromEnv()
		Expect(os.Setenv(hcoutil.OperatorNamespaceEnv, namespace)).To(Succeed())
		reqresolver.GeneratePlaceHolders()
	})

	_ = AfterSuite(func() {
		Expect(os.Setenv(hcoutil.OperatorNamespaceEnv, nsBefore)).To(Succeed())
		reqresolver.GeneratePlaceHolders()
	})
)

var _ = Describe("test ResolveReconcileRequest", func() {
	It("should return original req and true for request triggered by HC", func() {
		req := reconcile.Request{
			NamespacedName: types.NamespacedName{
				Namespace: namespace,
				Name:      hcoutil.HyperConvergedName,
			},
		}
		requestAfter, triggeredByHC := reqresolver.ResolveReconcileRequest(GinkgoLogr, req)
		Expect(requestAfter).To(Equal(req))
		Expect(triggeredByHC).To(BeTrue())
	})

	It("should return original req and true for request triggered by unknown source", func() {
		req := reconcile.Request{
			NamespacedName: types.NamespacedName{
				Namespace: namespace,
				Name:      "unknown",
			},
		}
		requestAfter, triggeredByHC := reqresolver.ResolveReconcileRequest(GinkgoLogr, req)
		Expect(requestAfter).To(Equal(req))
		Expect(triggeredByHC).To(BeTrueBecause("should recognized as triggered by the HyperConverged CR"))
	})

	It("should return HC req and true for request triggered by API-Server", func() {
		expected := reconcile.Request{
			NamespacedName: reqresolver.GetHyperConvergedNamespacedName(),
		}
		requestAfter, triggeredByHC := reqresolver.ResolveReconcileRequest(GinkgoLogr, reqresolver.GetAPIServerCRRequest())
		Expect(requestAfter).To(Equal(expected))
		Expect(triggeredByHC).To(BeTrueBecause("should recognized as triggered by the HyperConverged CR"))
	})

	It("should return HC req and true for request triggered by Ingress", func() {
		expected := reconcile.Request{
			NamespacedName: reqresolver.GetHyperConvergedNamespacedName(),
		}
		requestAfter, triggeredByHC := reqresolver.ResolveReconcileRequest(GinkgoLogr, reqresolver.GetIngressCRResource())
		Expect(requestAfter).To(Equal(expected))
		Expect(triggeredByHC).To(BeTrueBecause("should recognized as triggered by the HyperConverged CR"))
	})

	It("should return HC req and false for request triggered by Secondary resource", func() {
		expected := reconcile.Request{
			NamespacedName: reqresolver.GetHyperConvergedNamespacedName(),
		}
		requestAfter, triggeredByHC := reqresolver.ResolveReconcileRequest(GinkgoLogr, reqresolver.GetSecondaryCRRequest())
		Expect(requestAfter).To(Equal(expected))
		Expect(triggeredByHC).To(BeFalseBecause("should recognized as triggered by a secondary resource"))
	})
})

var _ = Describe("Check generated requests", func() {
	It("test GetHyperConvergedNamespacedName", func() {
		n := reqresolver.GetHyperConvergedNamespacedName()
		Expect(n).To(Equal(types.NamespacedName{Name: hcoutil.HyperConvergedName, Namespace: namespace}))
		Expect(reqresolver.IsTriggeredByHyperConverged(n)).To(BeTrueBecause("should recognized as triggered by HyperConverged CR"))
		Expect(reqresolver.IsTriggeredByAPIServerCR(reconcile.Request{NamespacedName: n})).To(BeFalseBecause("should not be recognized as triggered by APIServer CR"))
	})

	It("test GetAPIServerCRRequest", func() {
		req := reqresolver.GetAPIServerCRRequest()
		Expect(req.NamespacedName.Namespace).To(Equal(namespace))
		Expect(req.NamespacedName.Name).To(HavePrefix("api-server-cr-"))
		Expect(reqresolver.IsTriggeredByHyperConverged(req.NamespacedName)).To(BeFalseBecause("should not be recognized as triggered by HyperConverged CR"))
		Expect(reqresolver.IsTriggeredByAPIServerCR(req)).To(BeTrueBecause("should be recognized as triggered by APIServer CR"))
	})

	It("test GetSecondaryCRRequest", func() {
		req := reqresolver.GetSecondaryCRRequest()
		Expect(req.NamespacedName.Namespace).To(Equal(namespace))
		Expect(req.NamespacedName.Name).To(HavePrefix("hco-controlled-cr-"))
		Expect(reqresolver.IsTriggeredByHyperConverged(req.NamespacedName)).To(BeFalseBecause("should not be recognized as triggered by HyperConverged CR"))
		Expect(reqresolver.IsTriggeredByAPIServerCR(req)).To(BeFalseBecause("should not be recognized as triggered by APIServer CR"))
	})

	It("test GetIngressCRResource", func() {
		req := reqresolver.GetIngressCRResource()
		Expect(req.NamespacedName.Namespace).To(Equal(namespace))
		Expect(req.NamespacedName.Name).To(HavePrefix("ingress-cr-"))
		Expect(reqresolver.IsTriggeredByHyperConverged(req.NamespacedName)).To(BeFalseBecause("should not be recognized as triggered by HyperConverged CR"))
		Expect(reqresolver.IsTriggeredByAPIServerCR(req)).To(BeFalseBecause("should not be recognized as triggered by APIServer CR"))
	})
})
