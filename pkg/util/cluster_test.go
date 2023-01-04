package util

import (
	"context"
	"fmt"
	"os"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	openshiftconfigv1 "github.com/openshift/api/config/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes/scheme"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"
)

var _ = Describe("test clusterInfo", func() {
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
	)

	testScheme := scheme.Scheme
	Expect(openshiftconfigv1.Install(testScheme)).ToNot(HaveOccurred())

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
			WithRuntimeObjects(clusterVersion, infrastructure, ingress, apiServer, dns).
			Build()
		err := GetClusterInfo().Init(context.TODO(), cl, logger)
		Expect(err).ToNot(HaveOccurred())

		Expect(GetClusterInfo().IsOpenshift()).To(BeTrue(), "should return true for IsOpenshift()")
		Expect(GetClusterInfo().IsManagedByOLM()).To(BeTrue(), "should return true for IsManagedByOLM()")

		By("Check clusterInfo additional fields (for openshift)", func() {
			Expect(GetClusterInfo().GetBaseDomain()).To(Equal(baseDomain), "should return expected base domain")
		})
	})

	It("check Init on openshift, without OLM", func() {
		os.Unsetenv(OperatorConditionNameEnvVar)

		cl := fake.NewClientBuilder().
			WithScheme(testScheme).
			WithRuntimeObjects(clusterVersion, infrastructure, ingress, apiServer, dns).
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
				WithRuntimeObjects(clusterVersion, testInfrastructure, ingress, apiServer, dns).
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

	Context("TLSSecurityProfile", func() {

		DescribeTable(
			"check TLSSecurityProfile on different configurations ...",
			func(isOnOpenshift bool, clusterTLSSecurityProfile *openshiftconfigv1.TLSSecurityProfile, hcoTLSSecurityProfile *openshiftconfigv1.TLSSecurityProfile, expectedTLSSecurityProfile *openshiftconfigv1.TLSSecurityProfile) {

				testApiServer := &openshiftconfigv1.APIServer{
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
					os.Setenv(OperatorConditionNameEnvVar, "aValue")
					cl = fake.NewClientBuilder().
						WithScheme(testScheme).
						WithRuntimeObjects(clusterVersion, infrastructure, ingress, testApiServer, dns).
						Build()
				}
				err := GetClusterInfo().Init(context.TODO(), cl, logger)
				Expect(err).ToNot(HaveOccurred())

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
			os.Setenv(OperatorConditionNameEnvVar, "aValue")

			initialTLSSecurityProfile := &openshiftconfigv1.TLSSecurityProfile{
				Type:         openshiftconfigv1.TLSProfileIntermediateType,
				Intermediate: &openshiftconfigv1.IntermediateTLSProfile{},
			}
			updatedTLSSecurityProfile := &openshiftconfigv1.TLSSecurityProfile{
				Type:   openshiftconfigv1.TLSProfileModernType,
				Modern: &openshiftconfigv1.ModernTLSProfile{},
			}

			testApiServer := &openshiftconfigv1.APIServer{
				ObjectMeta: metav1.ObjectMeta{
					Name: "cluster",
				},
				Spec: openshiftconfigv1.APIServerSpec{
					TLSSecurityProfile: initialTLSSecurityProfile,
				},
			}

			cl := fake.NewClientBuilder().
				WithScheme(testScheme).
				WithRuntimeObjects(clusterVersion, infrastructure, ingress, testApiServer, dns).
				Build()
			err := GetClusterInfo().Init(context.TODO(), cl, logger)
			Expect(err).ToNot(HaveOccurred())

			Expect(GetClusterInfo().IsOpenshift()).To(BeTrue(), "should return true for IsOpenshift()")
			Expect(GetClusterInfo().IsManagedByOLM()).To(BeTrue(), "should return true for IsManagedByOLM()")

			Expect(GetClusterInfo().GetTLSSecurityProfile(nil)).To(Equal(initialTLSSecurityProfile), "should return the initial value")

			testApiServer.Spec.TLSSecurityProfile = updatedTLSSecurityProfile
			err = cl.Update(context.TODO(), testApiServer)
			Expect(err).ToNot(HaveOccurred())
			Expect(GetClusterInfo().GetTLSSecurityProfile(nil)).To(Equal(initialTLSSecurityProfile), "should still return the cached value (initial value)")

			err = GetClusterInfo().RefreshAPIServerCR(context.TODO(), cl)
			Expect(err).ToNot(HaveOccurred())

			Expect(GetClusterInfo().GetTLSSecurityProfile(nil)).To(Equal(updatedTLSSecurityProfile), "should return the updated value")

		})

	})

})
