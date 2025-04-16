package util

import (
	"context"
	"fmt"
	"os"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	openshiftconfigv1 "github.com/openshift/api/config/v1"
	operatorv1 "github.com/openshift/api/operator/v1"
	deschedulerv1 "github.com/openshift/cluster-kube-descheduler-operator/pkg/apis/descheduler/v1"
	corev1 "k8s.io/api/core/v1"
	apiextensionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes/scheme"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"
)

var _ = Describe("test clusterInfo", func() {
	BeforeEach(func() {
		origVar, origIsVarSet := os.LookupEnv(OperatorConditionNameEnvVar)

		DeferCleanup(func() {
			if origIsVarSet {
				Expect(os.Setenv(OperatorConditionNameEnvVar, origVar)).To(Succeed())
			} else {
				Expect(os.Unsetenv(OperatorConditionNameEnvVar)).To(Succeed())
			}
		})
	})

	const baseDomain = "basedomain"
	var (
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

		apiServer = &openshiftconfigv1.APIServer{
			ObjectMeta: metav1.ObjectMeta{
				Name: "cluster",
			},
			Spec: openshiftconfigv1.APIServerSpec{
				TLSSecurityProfile: &openshiftconfigv1.TLSSecurityProfile{
					Type:   openshiftconfigv1.TLSProfileModernType,
					Modern: &openshiftconfigv1.ModernTLSProfile{},
				},
			},
		}

		dns = &openshiftconfigv1.DNS{
			ObjectMeta: metav1.ObjectMeta{
				Name: "cluster",
			},
			Spec: openshiftconfigv1.DNSSpec{
				BaseDomain: baseDomain,
			},
		}

		ipv4network = &openshiftconfigv1.Network{
			ObjectMeta: metav1.ObjectMeta{
				Name: "cluster",
			},
			Status: openshiftconfigv1.NetworkStatus{
				ClusterNetwork: []openshiftconfigv1.ClusterNetworkEntry{
					{
						CIDR: "10.128.0.0/14",
					},
				},
			},
		}

		ipv6network = &openshiftconfigv1.Network{
			ObjectMeta: metav1.ObjectMeta{
				Name: "cluster",
			},
			Status: openshiftconfigv1.NetworkStatus{
				ClusterNetwork: []openshiftconfigv1.ClusterNetworkEntry{
					{
						CIDR: "fd01::/48",
					},
				},
			},
		}

		dualStackNetwork = &openshiftconfigv1.Network{
			ObjectMeta: metav1.ObjectMeta{
				Name: "cluster",
			},
			Status: openshiftconfigv1.NetworkStatus{
				ClusterNetwork: []openshiftconfigv1.ClusterNetworkEntry{
					{
						CIDR: "fd01::/48",
					},
					{
						CIDR: "10.128.0.0/14",
					},
				},
			},
		}

		deschedulerCRD = &apiextensionsv1.CustomResourceDefinition{
			ObjectMeta: metav1.ObjectMeta{
				Name: DeschedulerCRDName,
			},
		}

		deschedulerNamespace = &corev1.Namespace{
			ObjectMeta: metav1.ObjectMeta{
				Name: DeschedulerNamespace,
				Annotations: map[string]string{
					OpenshiftNodeSelectorAnn: "",
				},
			},
		}
	)

	testScheme := scheme.Scheme
	Expect(openshiftconfigv1.Install(testScheme)).To(Succeed())
	Expect(deschedulerv1.AddToScheme(testScheme)).To(Succeed())
	Expect(apiextensionsv1.AddToScheme(testScheme)).To(Succeed())

	logger := zap.New(zap.WriteTo(GinkgoWriter), zap.UseDevMode(true)).WithName("clusterInfo_test")

	It("check Init on kubernetes, without OLM", func() {
		Expect(os.Unsetenv(OperatorConditionNameEnvVar)).To(Succeed())
		cl := fake.NewClientBuilder().
			WithScheme(testScheme).
			Build()
		Expect(GetClusterInfo().Init(context.TODO(), cl, logger)).To(Succeed())

		Expect(GetClusterInfo().IsOpenshift()).To(BeFalse(), "should return false for IsOpenshift()")
		Expect(GetClusterInfo().IsManagedByOLM()).To(BeFalse(), "should return false for IsManagedByOLM()")
	})

	It("check Init on kubernetes, with OLM", func() {
		Expect(os.Setenv(OperatorConditionNameEnvVar, "aValue")).To(Succeed())
		cl := fake.NewClientBuilder().
			WithScheme(testScheme).
			Build()
		Expect(GetClusterInfo().Init(context.TODO(), cl, logger)).To(Succeed())

		Expect(GetClusterInfo().IsOpenshift()).To(BeFalse(), "should return false for IsOpenshift()")
		Expect(GetClusterInfo().IsManagedByOLM()).To(BeTrue(), "should return true for IsManagedByOLM()")
	})

	It("check Init on kubernetes, with KubeDescheduler CRD without any CR for it", func() {
		cl := fake.NewClientBuilder().
			WithScheme(testScheme).
			WithObjects(deschedulerCRD).
			WithStatusSubresource(deschedulerCRD).
			Build()
		Expect(GetClusterInfo().Init(context.TODO(), cl, logger)).To(Succeed())

		Expect(GetClusterInfo().IsOpenshift()).To(BeFalse(), "should return false for IsOpenshift()")
		Expect(GetClusterInfo().IsDeschedulerAvailable()).To(BeTrue(), "should return true for IsDeschedulerAvailable()")
		Expect(GetClusterInfo().IsDeschedulerCRDDeployed(context.TODO(), cl)).To(BeTrue(), "should return true for IsDeschedulerCRDDeployed(...)")
	})

	It("check Init on kubernetes, with KubeDescheduler CRD with a CR for it with default values", func() {
		cl := fake.NewClientBuilder().
			WithScheme(testScheme).
			WithObjects(deschedulerCRD, deschedulerNamespace).
			WithStatusSubresource(deschedulerCRD, deschedulerNamespace).
			Build()
		Expect(GetClusterInfo().Init(context.TODO(), cl, logger)).To(Succeed())

		Expect(GetClusterInfo().IsOpenshift()).To(BeFalse(), "should return false for IsOpenshift()")
		Expect(GetClusterInfo().IsDeschedulerAvailable()).To(BeTrue(), "should return true for IsDeschedulerAvailable()")
		Expect(GetClusterInfo().IsDeschedulerCRDDeployed(context.TODO(), cl)).To(BeTrue(), "should return true for IsDeschedulerCRDDeployed(...)")
	})

	It("check Init on openshift, with KubeDescheduler CRD without any CR for it", func() {
		cl := fake.NewClientBuilder().
			WithScheme(testScheme).
			WithObjects(clusterVersion, infrastructure, ingress, apiServer, dns, ipv4network, deschedulerCRD).
			WithStatusSubresource(clusterVersion, infrastructure, ingress, apiServer, dns, ipv4network, deschedulerCRD).
			Build()
		Expect(GetClusterInfo().Init(context.TODO(), cl, logger)).To(Succeed())

		Expect(GetClusterInfo().IsOpenshift()).To(BeTrue(), "should return true for IsOpenshift()")
		Expect(GetClusterInfo().IsDeschedulerAvailable()).To(BeTrue(), "should return true for IsDeschedulerAvailable()")
		Expect(GetClusterInfo().IsDeschedulerCRDDeployed(context.TODO(), cl)).To(BeTrue(), "should return true for IsDeschedulerCRDDeployed(...)")
	})

	DescribeTable(
		"check Init on openshift, with KubeDescheduler CRD with a CR for it ...",
		func(deschedulerCR *deschedulerv1.KubeDescheduler, expectedIsDeschedulerMisconfigured bool) {
			cl := fake.NewClientBuilder().
				WithScheme(testScheme).
				WithObjects(clusterVersion, infrastructure, ingress, apiServer, dns, ipv4network, deschedulerCRD, deschedulerNamespace, deschedulerCR).
				WithStatusSubresource(clusterVersion, infrastructure, ingress, apiServer, dns, ipv4network, deschedulerCRD, deschedulerNamespace, deschedulerCR).
				Build()
			Expect(GetClusterInfo().Init(context.TODO(), cl, logger)).To(Succeed())

			Expect(GetClusterInfo().IsOpenshift()).To(BeTrue(), "should return true for IsOpenshift()")
			Expect(GetClusterInfo().IsDeschedulerAvailable()).To(BeTrue(), "should return true for IsDeschedulerAvailable()")
			Expect(GetClusterInfo().IsDeschedulerCRDDeployed(context.TODO(), cl)).To(BeTrue(), "should return true for IsDeschedulerCRDDeployed(...)")
		},
		Entry(
			"with default configuration",
			&deschedulerv1.KubeDescheduler{
				ObjectMeta: metav1.ObjectMeta{
					Name:      DeschedulerCRName,
					Namespace: DeschedulerNamespace,
				},
				Spec: deschedulerv1.KubeDeschedulerSpec{},
			},
			true,
		),
		Entry(
			"with KubeVirt specific profile",
			&deschedulerv1.KubeDescheduler{
				ObjectMeta: metav1.ObjectMeta{
					Name:      DeschedulerCRName,
					Namespace: DeschedulerNamespace,
				},
				Spec: deschedulerv1.KubeDeschedulerSpec{
					Profiles: []deschedulerv1.DeschedulerProfile{
						deschedulerv1.RelieveAndMigrate,
					},
					ProfileCustomizations: &deschedulerv1.ProfileCustomizations{
						DevDeviationThresholds:      &deschedulerv1.AsymmetricLowDeviationThreshold,
						DevEnableSoftTainter:        true,
						DevActualUtilizationProfile: deschedulerv1.PrometheusCPUCombinedProfile,
					},
				},
			},
			false,
		),
		Entry(
			"with obsolete configuration",
			&deschedulerv1.KubeDescheduler{
				ObjectMeta: metav1.ObjectMeta{
					Name:      DeschedulerCRName,
					Namespace: DeschedulerNamespace,
				},
				Spec: deschedulerv1.KubeDeschedulerSpec{
					Profiles: []deschedulerv1.DeschedulerProfile{
						deschedulerv1.LifecycleAndUtilization,
					},
					ProfileCustomizations: &deschedulerv1.ProfileCustomizations{
						DevEnableEvictionsInBackground: true,
					},
				},
			},
			true,
		),
		Entry(
			"with wrong configuration 1",
			&deschedulerv1.KubeDescheduler{
				ObjectMeta: metav1.ObjectMeta{
					Name:      DeschedulerCRName,
					Namespace: DeschedulerNamespace,
				},
				Spec: deschedulerv1.KubeDeschedulerSpec{
					ProfileCustomizations: &deschedulerv1.ProfileCustomizations{
						DevEnableEvictionsInBackground: false,
					},
				},
			},
			true,
		),
		Entry(
			"with wrong configuration 2",
			&deschedulerv1.KubeDescheduler{
				ObjectMeta: metav1.ObjectMeta{
					Name:      DeschedulerCRName,
					Namespace: DeschedulerNamespace,
				},
				Spec: deschedulerv1.KubeDeschedulerSpec{
					ProfileCustomizations: &deschedulerv1.ProfileCustomizations{
						ThresholdPriorityClassName:     "testvalue",
						Namespaces:                     deschedulerv1.Namespaces{},
						DevEnableEvictionsInBackground: false,
					},
				},
			},
			true,
		),
		Entry(
			"with wrong configuration 3",
			&deschedulerv1.KubeDescheduler{
				ObjectMeta: metav1.ObjectMeta{
					Name:      DeschedulerCRName,
					Namespace: DeschedulerNamespace,
				},
				Spec: deschedulerv1.KubeDeschedulerSpec{
					OperatorSpec: operatorv1.OperatorSpec{
						ManagementState:  "testvalue",
						LogLevel:         "testvalue",
						OperatorLogLevel: "testvalue",
					},
					Profiles: []deschedulerv1.DeschedulerProfile{"test1", "test2"},
					Mode:     "testvalue",
				},
			},
			true,
		),
		Entry(
			"with configuration tuned for KubeVirt but with a wrong name",
			&deschedulerv1.KubeDescheduler{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test",
					Namespace: DeschedulerNamespace,
				},
				Spec: deschedulerv1.KubeDeschedulerSpec{
					ProfileCustomizations: &deschedulerv1.ProfileCustomizations{
						DevEnableEvictionsInBackground: true,
					},
				},
			},
			false,
		),
		Entry(
			"with configuration tuned for KubeVirt but in the wrong namespace",
			&deschedulerv1.KubeDescheduler{
				ObjectMeta: metav1.ObjectMeta{
					Name:      DeschedulerCRName,
					Namespace: "test",
				},
				Spec: deschedulerv1.KubeDeschedulerSpec{
					ProfileCustomizations: &deschedulerv1.ProfileCustomizations{
						DevEnableEvictionsInBackground: true,
					},
				},
			},
			false,
		),
	)

	It("check Init on openshift, with OLM", func() {
		Expect(os.Setenv(OperatorConditionNameEnvVar, "aValue")).To(Succeed())
		cl := fake.NewClientBuilder().
			WithScheme(testScheme).
			WithObjects(clusterVersion, infrastructure, ingress, apiServer, dns, ipv4network).
			WithStatusSubresource(clusterVersion, infrastructure, ingress, apiServer, dns, ipv4network).
			Build()
		Expect(GetClusterInfo().Init(context.TODO(), cl, logger)).To(Succeed())

		Expect(GetClusterInfo().IsOpenshift()).To(BeTrue(), "should return true for IsOpenshift()")
		Expect(GetClusterInfo().IsManagedByOLM()).To(BeTrue(), "should return true for IsManagedByOLM()")

		By("Check clusterInfo additional fields (for openshift)", func() {
			Expect(GetClusterInfo().GetBaseDomain()).To(Equal(baseDomain), "should return expected base domain")
		})
	})

	It("check Init on openshift, without OLM", func() {
		Expect(os.Unsetenv(OperatorConditionNameEnvVar)).To(Succeed())

		cl := fake.NewClientBuilder().
			WithScheme(testScheme).
			WithObjects(clusterVersion, infrastructure, ingress, apiServer, dns, ipv4network).
			WithStatusSubresource(clusterVersion, infrastructure, ingress, apiServer, dns, ipv4network).
			Build()
		Expect(GetClusterInfo().Init(context.TODO(), cl, logger)).To(Succeed())

		Expect(GetClusterInfo().IsOpenshift()).To(BeTrue(), "should return true for IsOpenshift()")
		Expect(GetClusterInfo().IsManagedByOLM()).To(BeFalse(), "should return false for IsManagedByOLM()")
	})

	It("check init on OpenShift, with single-stack IPv6 network", func() {
		cl := fake.NewClientBuilder().
			WithScheme(testScheme).
			WithObjects(clusterVersion, infrastructure, ingress, apiServer, dns, ipv6network).
			WithStatusSubresource(clusterVersion, infrastructure, ingress, apiServer, dns, ipv6network).
			Build()
		Expect(GetClusterInfo().Init(context.TODO(), cl, logger)).To(Succeed())

		Expect(GetClusterInfo().IsOpenshift()).To(BeTrue())
		Expect(GetClusterInfo().IsSingleStackIPv6()).To(BeTrue())
	})

	It("checks init on OpenShift with dual stack ipv4/ipv6 network configuration", func() {
		cl := fake.NewClientBuilder().
			WithScheme(testScheme).
			WithObjects(clusterVersion, infrastructure, ingress, apiServer, dns, dualStackNetwork).
			WithStatusSubresource(clusterVersion, infrastructure, ingress, apiServer, dns, dualStackNetwork).
			Build()
		Expect(GetClusterInfo().Init(context.TODO(), cl, logger)).To(Succeed())

		Expect(GetClusterInfo().IsOpenshift()).To(BeTrue())
		Expect(GetClusterInfo().IsSingleStackIPv6()).To(BeFalse())
	})

	DescribeTable(
		"check Init on openshift, with OLM, infrastructure topology ...",
		func(controlPlaneTopology, infrastructureTopology openshiftconfigv1.TopologyMode, numMasterNodes, numWorkerNodes int, expectedIsControlPlaneHighlyAvailable, expectedIsInfrastructureHighlyAvailable bool) {

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
			var nodesArray []client.Object
			for i := range numMasterNodes {
				masterNode := &corev1.Node{
					ObjectMeta: metav1.ObjectMeta{
						Name: fmt.Sprintf("master%d", i),
						Labels: map[string]string{
							"node-role.kubernetes.io/control-plane": "",
						},
					},
				}
				nodesArray = append(nodesArray, masterNode)
			}
			for i := range numWorkerNodes {
				workerNode := &corev1.Node{
					ObjectMeta: metav1.ObjectMeta{
						Name: fmt.Sprintf("worker%d", i),
						Labels: map[string]string{
							"node-role.kubernetes.io/worker": "",
						},
					},
				}
				nodesArray = append(nodesArray, workerNode)
			}

			Expect(os.Setenv(OperatorConditionNameEnvVar, "aValue")).To(Succeed())
			cl := fake.NewClientBuilder().
				WithScheme(testScheme).
				WithObjects(clusterVersion, testInfrastructure, ingress, apiServer, dns, ipv4network).
				WithObjects(nodesArray...).
				WithStatusSubresource(clusterVersion, testInfrastructure, ingress, apiServer, dns, ipv4network).
				Build()
			Expect(GetClusterInfo().Init(context.TODO(), cl, logger)).To(Succeed())

			Expect(GetClusterInfo().IsOpenshift()).To(BeTrue(), "should return true for IsOpenshift()")
			Expect(GetClusterInfo().IsManagedByOLM()).To(BeTrue(), "should return true for IsManagedByOLM()")
			Expect(GetClusterInfo().IsControlPlaneHighlyAvailable()).To(Equal(expectedIsControlPlaneHighlyAvailable), "should return true for HighlyAvailable ControlPlane")
			Expect(GetClusterInfo().IsInfrastructureHighlyAvailable()).To(Equal(expectedIsInfrastructureHighlyAvailable), "should return true for HighlyAvailable Infrastructure")
		},
		Entry(
			"HighlyAvailable ControlPlane and Infrastructure",
			openshiftconfigv1.HighlyAvailableTopologyMode,
			openshiftconfigv1.HighlyAvailableTopologyMode,
			3,
			2,
			true,
			true,
		),
		Entry(
			"HighlyAvailable ControlPlane, SingleReplica Infrastructure",
			openshiftconfigv1.HighlyAvailableTopologyMode,
			openshiftconfigv1.SingleReplicaTopologyMode,
			3,
			1,
			true,
			false,
		),
		Entry(
			"SingleReplica ControlPlane, HighlyAvailable Infrastructure",
			openshiftconfigv1.SingleReplicaTopologyMode,
			openshiftconfigv1.HighlyAvailableTopologyMode,
			1,
			2,
			false,
			true,
		),
		Entry(
			"SingleReplica ControlPlane and Infrastructure",
			openshiftconfigv1.SingleReplicaTopologyMode,
			openshiftconfigv1.SingleReplicaTopologyMode,
			1,
			1,
			false,
			false,
		),
	)

	DescribeTable(
		"check Init on k8s, infrastructure topology ...",
		func(numMasterNodes, numWorkerNodes int, expectedIsControlPlaneHighlyAvailable, expectedIsInfrastructureHighlyAvailable bool) {

			var nodesArray []client.Object
			for i := range numMasterNodes {
				masterNode := &corev1.Node{
					ObjectMeta: metav1.ObjectMeta{
						Name: fmt.Sprintf("master%d", i),
						Labels: map[string]string{
							"node-role.kubernetes.io/master": "",
						},
					},
				}
				nodesArray = append(nodesArray, masterNode)
			}
			for i := range numWorkerNodes {
				workerNode := &corev1.Node{
					ObjectMeta: metav1.ObjectMeta{
						Name: fmt.Sprintf("worker%d", i),
						Labels: map[string]string{
							"node-role.kubernetes.io/worker": "",
						},
					},
				}
				nodesArray = append(nodesArray, workerNode)
			}
			Expect(os.Unsetenv(OperatorConditionNameEnvVar)).To(Succeed())
			cl := fake.NewClientBuilder().
				WithScheme(testScheme).
				WithObjects(nodesArray...).
				WithStatusSubresource(nodesArray...).
				Build()

			Expect(GetClusterInfo().Init(context.TODO(), cl, logger)).To(Succeed())

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

	Context("TLSSecurityProfile", func() {

		DescribeTable(
			"check TLSSecurityProfile on different configurations ...",
			func(isOnOpenshift bool, clusterTLSSecurityProfile *openshiftconfigv1.TLSSecurityProfile, hcoTLSSecurityProfile *openshiftconfigv1.TLSSecurityProfile, expectedTLSSecurityProfile *openshiftconfigv1.TLSSecurityProfile) {

				testAPIServer := &openshiftconfigv1.APIServer{
					ObjectMeta: metav1.ObjectMeta{
						Name: "cluster",
					},
					Spec: openshiftconfigv1.APIServerSpec{
						TLSSecurityProfile: clusterTLSSecurityProfile,
					},
				}

				cl := fake.NewClientBuilder().
					WithScheme(testScheme).
					WithRuntimeObjects().
					Build()
				if isOnOpenshift {
					Expect(os.Setenv(OperatorConditionNameEnvVar, "aValue")).To(Succeed())
					cl = fake.NewClientBuilder().
						WithScheme(testScheme).
						WithRuntimeObjects(clusterVersion, infrastructure, ingress, testAPIServer, dns, ipv4network).
						Build()
				}
				Expect(GetClusterInfo().Init(context.TODO(), cl, logger)).To(Succeed())

				Expect(GetClusterInfo().IsOpenshift()).To(Equal(isOnOpenshift), "should return true for IsOpenshift()")
				Expect(GetClusterInfo().GetTLSSecurityProfile(hcoTLSSecurityProfile)).To(Equal(expectedTLSSecurityProfile), "should return the expected TLSSecurityProfile")

			},
			Entry(
				"on Openshift, TLSSecurityProfile unset on HCO, should return cluster wide TLSSecurityProfile",
				true,
				&openshiftconfigv1.TLSSecurityProfile{
					Type:   openshiftconfigv1.TLSProfileModernType,
					Modern: &openshiftconfigv1.ModernTLSProfile{},
				},
				nil,
				&openshiftconfigv1.TLSSecurityProfile{
					Type:   openshiftconfigv1.TLSProfileModernType,
					Modern: &openshiftconfigv1.ModernTLSProfile{},
				},
			),
			Entry(
				"on Openshift with wrong values, TLSSecurityProfile unset on HCO, should return sanitized cluster wide TLSSecurityProfile - 1",
				true,
				&openshiftconfigv1.TLSSecurityProfile{
					Type:   openshiftconfigv1.TLSProfileCustomType,
					Modern: &openshiftconfigv1.ModernTLSProfile{},
				},
				nil,
				&openshiftconfigv1.TLSSecurityProfile{
					Type: openshiftconfigv1.TLSProfileCustomType,
					Custom: &openshiftconfigv1.CustomTLSProfile{
						TLSProfileSpec: openshiftconfigv1.TLSProfileSpec{
							Ciphers:       openshiftconfigv1.TLSProfiles[openshiftconfigv1.TLSProfileIntermediateType].Ciphers,
							MinTLSVersion: openshiftconfigv1.TLSProfiles[openshiftconfigv1.TLSProfileIntermediateType].MinTLSVersion,
						},
					},
				},
			),
			Entry(
				"on Openshift with wrong values, TLSSecurityProfile unset on HCO, should return sanitized cluster wide TLSSecurityProfile - 2",
				true,
				&openshiftconfigv1.TLSSecurityProfile{
					Type: openshiftconfigv1.TLSProfileCustomType,
				},
				nil,
				&openshiftconfigv1.TLSSecurityProfile{
					Type: openshiftconfigv1.TLSProfileCustomType,
					Custom: &openshiftconfigv1.CustomTLSProfile{
						TLSProfileSpec: openshiftconfigv1.TLSProfileSpec{
							Ciphers:       openshiftconfigv1.TLSProfiles[openshiftconfigv1.TLSProfileIntermediateType].Ciphers,
							MinTLSVersion: openshiftconfigv1.TLSProfiles[openshiftconfigv1.TLSProfileIntermediateType].MinTLSVersion,
						},
					},
				},
			),
			Entry(
				"on Openshift with wrong values, TLSSecurityProfile unset on HCO, should return sanitized cluster wide TLSSecurityProfile - 3",
				true,
				&openshiftconfigv1.TLSSecurityProfile{
					Type: openshiftconfigv1.TLSProfileCustomType,
					Custom: &openshiftconfigv1.CustomTLSProfile{
						TLSProfileSpec: openshiftconfigv1.TLSProfileSpec{
							Ciphers: []string{
								"wrongname1",
								"TLS_AES_128_GCM_SHA256",
								"TLS_AES_256_GCM_SHA384",
								"TLS_CHACHA20_POLY1305_SHA256",
								"ECDHE-ECDSA-AES128-GCM-SHA256",
								"ECDHE-RSA-AES128-GCM-SHA256",
								"ECDHE-ECDSA-AES256-GCM-SHA384",
								"ECDHE-RSA-AES256-GCM-SHA384",
								"ECDHE-ECDSA-CHACHA20-POLY1305",
								"ECDHE-RSA-CHACHA20-POLY1305",
								"wrongname2",
								"DHE-RSA-AES128-GCM-SHA256",
								"DHE-RSA-AES256-GCM-SHA384",
								"wrongname3",
							},
							MinTLSVersion: openshiftconfigv1.TLSProfiles[openshiftconfigv1.TLSProfileIntermediateType].MinTLSVersion,
						},
					},
				},
				nil,
				&openshiftconfigv1.TLSSecurityProfile{
					Type: openshiftconfigv1.TLSProfileCustomType,
					Custom: &openshiftconfigv1.CustomTLSProfile{
						TLSProfileSpec: openshiftconfigv1.TLSProfileSpec{
							Ciphers: []string{
								"TLS_AES_128_GCM_SHA256",
								"TLS_AES_256_GCM_SHA384",
								"TLS_CHACHA20_POLY1305_SHA256",
								"ECDHE-ECDSA-AES128-GCM-SHA256",
								"ECDHE-RSA-AES128-GCM-SHA256",
								"ECDHE-ECDSA-AES256-GCM-SHA384",
								"ECDHE-RSA-AES256-GCM-SHA384",
								"ECDHE-ECDSA-CHACHA20-POLY1305",
								"ECDHE-RSA-CHACHA20-POLY1305",
								"DHE-RSA-AES128-GCM-SHA256",
								"DHE-RSA-AES256-GCM-SHA384",
							},
							MinTLSVersion: openshiftconfigv1.TLSProfiles[openshiftconfigv1.TLSProfileIntermediateType].MinTLSVersion,
						},
					},
				},
			),
			Entry(
				"on Openshift, TLSSecurityProfile set on HCO, should return HCO specific TLSSecurityProfile",
				true,
				&openshiftconfigv1.TLSSecurityProfile{
					Type:   openshiftconfigv1.TLSProfileModernType,
					Modern: &openshiftconfigv1.ModernTLSProfile{},
				},
				&openshiftconfigv1.TLSSecurityProfile{
					Type: openshiftconfigv1.TLSProfileOldType,
					Old:  &openshiftconfigv1.OldTLSProfile{},
				},
				&openshiftconfigv1.TLSSecurityProfile{
					Type: openshiftconfigv1.TLSProfileOldType,
					Old:  &openshiftconfigv1.OldTLSProfile{},
				},
			),
			Entry(
				"on k8s, TLSSecurityProfile unset on HCO, should return a default value (Intermediate TLSSecurityProfile)",
				false,
				nil,
				nil,
				&openshiftconfigv1.TLSSecurityProfile{
					Type:         openshiftconfigv1.TLSProfileIntermediateType,
					Intermediate: &openshiftconfigv1.IntermediateTLSProfile{},
				},
			),
			Entry(
				"on k8s, TLSSecurityProfile unset on HCO, should return HCO specific TLSSecurityProfile)",
				false,
				nil,
				&openshiftconfigv1.TLSSecurityProfile{
					Type:   openshiftconfigv1.TLSProfileModernType,
					Modern: &openshiftconfigv1.ModernTLSProfile{},
				},
				&openshiftconfigv1.TLSSecurityProfile{
					Type:   openshiftconfigv1.TLSProfileModernType,
					Modern: &openshiftconfigv1.ModernTLSProfile{},
				},
			),
		)

		It("should refresh ApiServer on changes", func() {
			Expect(os.Setenv(OperatorConditionNameEnvVar, "aValue")).To(Succeed())

			initialTLSSecurityProfile := &openshiftconfigv1.TLSSecurityProfile{
				Type:         openshiftconfigv1.TLSProfileIntermediateType,
				Intermediate: &openshiftconfigv1.IntermediateTLSProfile{},
			}
			updatedTLSSecurityProfile := &openshiftconfigv1.TLSSecurityProfile{
				Type:   openshiftconfigv1.TLSProfileModernType,
				Modern: &openshiftconfigv1.ModernTLSProfile{},
			}

			testAPIServer := &openshiftconfigv1.APIServer{
				ObjectMeta: metav1.ObjectMeta{
					Name: "cluster",
				},
				Spec: openshiftconfigv1.APIServerSpec{
					TLSSecurityProfile: initialTLSSecurityProfile,
				},
			}

			cl := fake.NewClientBuilder().
				WithScheme(testScheme).
				WithRuntimeObjects(clusterVersion, infrastructure, ingress, testAPIServer, dns, ipv4network).
				Build()
			Expect(GetClusterInfo().Init(context.TODO(), cl, logger)).To(Succeed())

			Expect(GetClusterInfo().IsOpenshift()).To(BeTrue(), "should return true for IsOpenshift()")
			Expect(GetClusterInfo().IsManagedByOLM()).To(BeTrue(), "should return true for IsManagedByOLM()")

			Expect(GetClusterInfo().GetTLSSecurityProfile(nil)).To(Equal(initialTLSSecurityProfile), "should return the initial value")

			testAPIServer.Spec.TLSSecurityProfile = updatedTLSSecurityProfile
			Expect(cl.Update(context.TODO(), testAPIServer)).To(Succeed())
			Expect(GetClusterInfo().GetTLSSecurityProfile(nil)).To(Equal(initialTLSSecurityProfile), "should still return the cached value (initial value)")

			Expect(GetClusterInfo().RefreshAPIServerCR(context.TODO(), cl)).To(Succeed())

			Expect(GetClusterInfo().GetTLSSecurityProfile(nil)).To(Equal(updatedTLSSecurityProfile), "should return the updated value")
		})

		It("should detect that KubeDescheduler CRD got deployed if initially unavailable", func() {
			Expect(os.Setenv(OperatorConditionNameEnvVar, "aValue")).To(Succeed())

			testAPIServer := &openshiftconfigv1.APIServer{
				ObjectMeta: metav1.ObjectMeta{
					Name: "cluster",
				},
				Spec: openshiftconfigv1.APIServerSpec{},
			}

			cl := fake.NewClientBuilder().
				WithScheme(testScheme).
				WithRuntimeObjects(clusterVersion, infrastructure, ingress, testAPIServer, dns, ipv4network).
				Build()
			Expect(GetClusterInfo().Init(context.TODO(), cl, logger)).To(Succeed())

			Expect(GetClusterInfo().IsOpenshift()).To(BeTrue(), "should return true for IsOpenshift()")
			Expect(GetClusterInfo().IsManagedByOLM()).To(BeTrue(), "should return true for IsManagedByOLM()")
			Expect(GetClusterInfo().IsDeschedulerAvailable()).To(BeFalse(), "should initially return false for IsDeschedulerAvailable()")
			Expect(GetClusterInfo().IsDeschedulerCRDDeployed(context.TODO(), cl)).To(BeFalse(), "should initially return false for IsDeschedulerCRDDeployed(...)")

			deschedulerCRD = &apiextensionsv1.CustomResourceDefinition{
				ObjectMeta: metav1.ObjectMeta{
					Name: DeschedulerCRDName,
				},
			}

			Expect(cl.Create(context.TODO(), deschedulerCRD)).To(Succeed())
			Expect(GetClusterInfo().IsDeschedulerAvailable()).To(BeFalse(), "should still return false for IsDeschedulerAvailable() (until the operator will restart)")
			Expect(GetClusterInfo().IsDeschedulerCRDDeployed(context.TODO(), cl)).To(BeTrue(), "should now return true for IsDeschedulerCRDDeployed(...)")
		})
	})
})
