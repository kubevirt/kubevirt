package util

import (
	"context"
	"os"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	openshiftconfigv1 "github.com/openshift/api/config/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes/scheme"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"
)

var _ = Describe("test clusterInfo", func() {
	var (
		origIsVarSet   bool
		origVar        string
		clusterVersion = &openshiftconfigv1.ClusterVersion{
			ObjectMeta: metav1.ObjectMeta{
				Name: "version",
			},
			Spec: openshiftconfigv1.ClusterVersionSpec{
				ClusterID: "clusterId",
			},
		}

		ingress = &openshiftconfigv1.Ingress{
			ObjectMeta: metav1.ObjectMeta{
				Name: "cluster",
			},
			Spec: openshiftconfigv1.IngressSpec{
				Domain: "domain",
			},
		}
	)

	testScheme := scheme.Scheme
	Expect(openshiftconfigv1.Install(testScheme)).ToNot(HaveOccurred())

	BeforeSuite(func() {
		origVar, origIsVarSet = os.LookupEnv(OperatorConditionNameEnvVar)
	})

	AfterSuite(func() {
		if origIsVarSet {
			os.Setenv(OperatorConditionNameEnvVar, origVar)
		} else {
			os.Unsetenv(OperatorConditionNameEnvVar)
		}
	})

	logger := zap.New(zap.WriteTo(GinkgoWriter), zap.UseDevMode(true)).WithName("clusterInfo_test")

	It("check Init on kubernetes, without OLM", func() {
		os.Unsetenv(OperatorConditionNameEnvVar)
		cl := fake.NewClientBuilder().
			WithScheme(testScheme).
			Build()
		err := GetClusterInfo().Init(context.TODO(), cl, logger)
		Expect(err).ToNot(HaveOccurred())

		Expect(GetClusterInfo().IsOpenshift()).To(BeFalse(), "should return false for IsOpenshift()")
		Expect(GetClusterInfo().IsManagedByOLM()).To(BeFalse(), "should return false for IsManagedByOLM()")
	})

	It("check Init on kubernetes, with OLM", func() {
		os.Setenv(OperatorConditionNameEnvVar, "aValue")
		cl := fake.NewClientBuilder().
			WithScheme(testScheme).
			Build()
		err := GetClusterInfo().Init(context.TODO(), cl, logger)
		Expect(err).ToNot(HaveOccurred())

		Expect(GetClusterInfo().IsOpenshift()).To(BeFalse(), "should return false for IsOpenshift()")
		Expect(GetClusterInfo().IsManagedByOLM()).To(BeTrue(), "should return true for IsManagedByOLM()")
	})

	It("check Init on openshift, with OLM", func() {
		os.Setenv(OperatorConditionNameEnvVar, "aValue")
		cl := fake.NewClientBuilder().
			WithScheme(testScheme).
			WithRuntimeObjects(clusterVersion, ingress).
			Build()
		err := GetClusterInfo().Init(context.TODO(), cl, logger)
		Expect(err).ToNot(HaveOccurred())

		Expect(GetClusterInfo().IsOpenshift()).To(BeTrue(), "should return true for IsOpenshift()")
		Expect(GetClusterInfo().IsManagedByOLM()).To(BeTrue(), "should return true for IsManagedByOLM()")
	})

	It("check Init on openshift, without OLM", func() {
		os.Unsetenv(OperatorConditionNameEnvVar)

		cl := fake.NewClientBuilder().
			WithScheme(testScheme).
			WithRuntimeObjects(clusterVersion, ingress).
			Build()
		err := GetClusterInfo().Init(context.TODO(), cl, logger)
		Expect(err).ToNot(HaveOccurred())

		Expect(GetClusterInfo().IsOpenshift()).To(BeTrue(), "should return true for IsOpenshift()")
		Expect(GetClusterInfo().IsManagedByOLM()).To(BeFalse(), "should return false for IsManagedByOLM()")
	})
})
