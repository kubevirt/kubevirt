package util

import (
	"context"
	"fmt"
	"os"

	corev1 "k8s.io/api/core/v1"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/ginkgo/extensions/table"
	. "github.com/onsi/gomega"
	openshiftconfigv1 "github.com/openshift/api/config/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
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

		infrastructure = &openshiftconfigv1.Infrastructure{
			ObjectMeta: metav1.ObjectMeta{
				Name: "cluster",
			},
			Status: openshiftconfigv1.InfrastructureStatus{
				ControlPlaneTopology:   openshiftconfigv1.HighlyAvailableTopologyMode,
				InfrastructureTopology: openshiftconfigv1.HighlyAvailableTopologyMode,
				PlatformStatus: &openshiftconfigv1.PlatformStatus{
					Type: "mocked",
				},
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
			WithRuntimeObjects(clusterVersion, infrastructure, ingress).
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
			WithRuntimeObjects(clusterVersion, infrastructure, ingress).
			Build()
		err := GetClusterInfo().Init(context.TODO(), cl, logger)
		Expect(err).ToNot(HaveOccurred())

		Expect(GetClusterInfo().IsOpenshift()).To(BeTrue(), "should return true for IsOpenshift()")
		Expect(GetClusterInfo().IsManagedByOLM()).To(BeFalse(), "should return false for IsManagedByOLM()")
	})

	DescribeTable(
		"check Init on openshift, with OLM, infrastructure topology ...",
		func(controlPlaneTopology, infrastructureTopology openshiftconfigv1.TopologyMode, expectedIsControlPlaneHighlyAvailable, expectedIsInfrastructureHighlyAvailable bool) {

			testInfrastructure := &openshiftconfigv1.Infrastructure{
				ObjectMeta: metav1.ObjectMeta{
					Name: "cluster",
				},
				Status: openshiftconfigv1.InfrastructureStatus{
					ControlPlaneTopology:   controlPlaneTopology,
					InfrastructureTopology: infrastructureTopology,
					PlatformStatus: &openshiftconfigv1.PlatformStatus{
						Type: "mocked",
					},
				},
			}

			os.Setenv(OperatorConditionNameEnvVar, "aValue")
			cl := fake.NewClientBuilder().
				WithScheme(testScheme).
				WithRuntimeObjects(clusterVersion, testInfrastructure, ingress).
				Build()
			err := GetClusterInfo().Init(context.TODO(), cl, logger)
			Expect(err).ToNot(HaveOccurred())

			Expect(GetClusterInfo().IsOpenshift()).To(BeTrue(), "should return true for IsOpenshift()")
			Expect(GetClusterInfo().IsManagedByOLM()).To(BeTrue(), "should return true for IsManagedByOLM()")
			Expect(GetClusterInfo().IsControlPlaneHighlyAvailable()).To(Equal(expectedIsControlPlaneHighlyAvailable), "should return true for HighlyAvailable ControlPlane")
			Expect(GetClusterInfo().IsInfrastructureHighlyAvailable()).To(Equal(expectedIsInfrastructureHighlyAvailable), "should return true for HighlyAvailable Infrastructure")
		},
		Entry(
			"HighlyAvailable ControlPlane and Infrastructure",
			openshiftconfigv1.HighlyAvailableTopologyMode,
			openshiftconfigv1.HighlyAvailableTopologyMode,
			true,
			true,
		),
		Entry(
			"HighlyAvailable ControlPlane, SingleReplica Infrastructure",
			openshiftconfigv1.HighlyAvailableTopologyMode,
			openshiftconfigv1.SingleReplicaTopologyMode,
			true,
			false,
		),
		Entry(
			"SingleReplica ControlPlane, HighlyAvailable Infrastructure",
			openshiftconfigv1.SingleReplicaTopologyMode,
			openshiftconfigv1.HighlyAvailableTopologyMode,
			false,
			true,
		),
		Entry(
			"SingleReplica ControlPlane and Infrastructure",
			openshiftconfigv1.SingleReplicaTopologyMode,
			openshiftconfigv1.SingleReplicaTopologyMode,
			false,
			false,
		),
	)

	DescribeTable(
		"check Init on k8s, infrastructure topology ...",
		func(numMasterNodes, numWorkerNodes int, expectedIsControlPlaneHighlyAvailable, expectedIsInfrastructureHighlyAvailable bool) {

			var nodesArray []runtime.Object
			for i := 0; i < numMasterNodes; i++ {
				masterNode := &corev1.Node{
					ObjectMeta: metav1.ObjectMeta{
						Name: "master" + fmt.Sprint(i),
						Labels: map[string]string{
							"node-role.kubernetes.io/master": "",
						},
					},
				}
				nodesArray = append(nodesArray, masterNode)
			}
			for i := 0; i < numWorkerNodes; i++ {
				workerNode := &corev1.Node{
					ObjectMeta: metav1.ObjectMeta{
						Name: "worker" + fmt.Sprint(i),
						Labels: map[string]string{
							"node-role.kubernetes.io/worker": "",
						},
					},
				}
				nodesArray = append(nodesArray, workerNode)
			}
			os.Unsetenv(OperatorConditionNameEnvVar)
			cl := fake.NewClientBuilder().
				WithScheme(testScheme).
				WithRuntimeObjects(nodesArray...).
				Build()

			err := GetClusterInfo().Init(context.TODO(), cl, logger)
			Expect(err).ToNot(HaveOccurred())

			Expect(GetClusterInfo().IsOpenshift()).To(BeFalse(), "should return false for IsOpenshift()")
			Expect(GetClusterInfo().IsManagedByOLM()).To(BeFalse(), "should return false for IsManagedByOLM()")
			Expect(GetClusterInfo().IsControlPlaneHighlyAvailable()).To(Equal(expectedIsControlPlaneHighlyAvailable), "should return true for HighlyAvailable ControlPlane")
			Expect(GetClusterInfo().IsInfrastructureHighlyAvailable()).To(Equal(expectedIsInfrastructureHighlyAvailable), "should return true for HighlyAvailable Infrastructure")
		},
		Entry(
			"3 master nodes, 3 worker nodes",
			3,
			3,
			true,
			true,
		),
		Entry(
			"2 master nodes, 2 worker nodes",
			2,
			2,
			false,
			true,
		),
		Entry(
			"3 master nodes, 1 worker node",
			3,
			1,
			true,
			false,
		),
		Entry(
			"1 master mode, 1 worker node",
			1,
			1,
			false,
			false,
		),
	)

})
