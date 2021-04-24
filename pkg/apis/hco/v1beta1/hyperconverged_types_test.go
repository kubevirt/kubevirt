package v1beta1

import (
	"testing"
	"time"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	conditionsv1 "github.com/openshift/custom-resource-status/conditions/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	sdkapi "kubevirt.io/controller-lifecycle-operator-sdk/pkg/sdk/api"
)

const (
	testName       = "aName"
	testVersion    = "aVersion"
	testOldVersion = "anOldVersion"
)

func TestHyperConvergedStatus(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "v1beta1.HyperConvergedStatus Suite")
}

var _ = Describe("HyperconvergedTypes", func() {
	Describe("HyperConvergedStatus.UpdateVersion", func() {
		Context("Should be able to add a new version to a nil version array", func() {
			hcs := &HyperConvergedStatus{
				Conditions:     []conditionsv1.Condition{},
				RelatedObjects: []corev1.ObjectReference{},
			}

			hcs.UpdateVersion(testName, testVersion)

			It("Versions array should be with one element", func() {
				Expect(len(hcs.Versions)).Should(Equal(1))
			})

			It(`The version name should be "aName"`, func() {
				Expect(hcs.Versions[0].Name).Should(Equal(testName))
			})

			It(`The version should be "aVersion"`, func() {
				Expect(hcs.Versions[0].Version).Should(Equal(testVersion))
			})
		})

		Context("Should be able to add a new version to an empty version array", func() {
			hcs := &HyperConvergedStatus{
				Conditions:     []conditionsv1.Condition{},
				RelatedObjects: []corev1.ObjectReference{},
				Versions:       Versions{},
			}

			hcs.UpdateVersion(testName, testVersion)

			It("Versions array should be with one element", func() {
				Expect(len(hcs.Versions)).Should(Equal(1))
			})

			It(`The version name should be "aName"`, func() {
				Expect(hcs.Versions[0].Name).Should(Equal(testName))
			})

			It(`The version should be "aVersion"`, func() {
				Expect(hcs.Versions[0].Version).Should(Equal(testVersion))
			})
		})

		Context("Should be able to add a new version to an existing version array", func() {
			hcs := &HyperConvergedStatus{
				Conditions:     []conditionsv1.Condition{},
				RelatedObjects: []corev1.ObjectReference{},
				Versions: Versions{
					{Name: "aaa", Version: "1.2.3"},
					{Name: "bbb", Version: "4.5.6"},
					{Name: "ccc", Version: "7.8.9"},
				},
			}

			hcs.UpdateVersion(testName, testVersion)

			It("Versions array should be with four elements", func() {
				Expect(len(hcs.Versions)).Should(Equal(4))
			})

			It(`The version name should be "aName"`, func() {
				Expect(hcs.Versions[3].Name).Should(Equal(testName))
			})

			It(`The version should be "aVersion"`, func() {
				Expect(hcs.Versions[3].Version).Should(Equal(testVersion))
			})
		})

		Context("Should be able to update a new version in an existing version array (first element)", func() {
			hcs := &HyperConvergedStatus{
				Conditions:     []conditionsv1.Condition{},
				RelatedObjects: []corev1.ObjectReference{},
				Versions: Versions{
					{Name: testName, Version: testOldVersion},
					{Name: "bbb", Version: "4.5.6"},
					{Name: "ccc", Version: "7.8.9"},
				},
			}

			hcs.UpdateVersion(testName, testVersion)

			It("Versions array should be with three elements", func() {
				Expect(len(hcs.Versions)).Should(Equal(3))
			})

			It(`The version name should be "aName"`, func() {
				Expect(hcs.Versions[0].Name).Should(Equal(testName))
			})

			It(`The version should be "aVersion"`, func() {
				Expect(hcs.Versions[0].Version).Should(Equal(testVersion))
			})
		})

		Context("Should be able to update a new version in an existing version array (middle element)", func() {
			hcs := &HyperConvergedStatus{
				Conditions:     []conditionsv1.Condition{},
				RelatedObjects: []corev1.ObjectReference{},
				Versions: Versions{
					{Name: "aaa", Version: "1.2.3"},
					{Name: testName, Version: testOldVersion},
					{Name: "ccc", Version: "7.8.9"},
				},
			}

			hcs.UpdateVersion(testName, testVersion)

			It("Versions array should be with three elements", func() {
				Expect(len(hcs.Versions)).Should(Equal(3))
			})

			It(`The version name should be "aName"`, func() {
				Expect(hcs.Versions[1].Name).Should(Equal(testName))
			})

			It(`The version should be "aVersion"`, func() {
				Expect(hcs.Versions[1].Version).Should(Equal(testVersion))
			})
		})

		Context("Should be able to update a new version in an existing version array (last element)", func() {
			hcs := &HyperConvergedStatus{
				Conditions:     []conditionsv1.Condition{},
				RelatedObjects: []corev1.ObjectReference{},
				Versions: Versions{
					{Name: "aaa", Version: "1.2.3"},
					{Name: "bbb", Version: "4.5.6"},
					{Name: testName, Version: testOldVersion},
				},
			}

			hcs.UpdateVersion(testName, testVersion)

			It("Versions array should be with three elements", func() {
				Expect(len(hcs.Versions)).Should(Equal(3))
			})

			It(`The version name should be "aName"`, func() {
				Expect(hcs.Versions[2].Name).Should(Equal(testName))
			})

			It(`The version should be "aVersion"`, func() {
				Expect(hcs.Versions[2].Version).Should(Equal(testVersion))
			})
		})

	})

	Describe("HyperConvergedStatus.GetVersion", func() {
		Context("should return empty response for nil array", func() {
			hcs := &HyperConvergedStatus{
				Conditions:     []conditionsv1.Condition{},
				RelatedObjects: []corev1.ObjectReference{},
			}

			ver, ok := hcs.GetVersion(testName)

			It("should not find the version", func() {
				Expect(ok).To(BeFalse())
			})
			It("the version should be empty", func() {
				Expect(ver).To(BeEmpty())
			})
		})

		Context("should return empty response for empty array", func() {
			hcs := &HyperConvergedStatus{
				Conditions:     []conditionsv1.Condition{},
				RelatedObjects: []corev1.ObjectReference{},
				Versions:       Versions{},
			}

			ver, ok := hcs.GetVersion(testName)

			It("should not find the version", func() {
				Expect(ok).To(BeFalse())
			})

			It("the version should be empty", func() {
				Expect(ver).To(BeEmpty())
			})
		})

		Context("should return empty response if the version is not in the versions array", func() {
			hcs := &HyperConvergedStatus{
				Conditions:     []conditionsv1.Condition{},
				RelatedObjects: []corev1.ObjectReference{},
				Versions: Versions{
					{Name: "aaa", Version: "1.2.3"},
					{Name: "bbb", Version: "4.5.6"},
					{Name: "ccc", Version: "7.8.9"},
				},
			}

			ver, ok := hcs.GetVersion(testName)

			It("should not find the version", func() {
				Expect(ok).To(BeFalse())
			})

			It("the version should be empty", func() {
				Expect(ver).To(BeEmpty())
			})
		})

		Context("should return a valid response if the version is in the versions array (first element)", func() {
			hcs := &HyperConvergedStatus{
				Conditions:     []conditionsv1.Condition{},
				RelatedObjects: []corev1.ObjectReference{},
				Versions: Versions{
					{Name: testName, Version: testVersion},
					{Name: "bbb", Version: "4.5.6"},
					{Name: "ccc", Version: "7.8.9"},
				},
			}

			ver, ok := hcs.GetVersion(testName)

			It("should not find the version", func() {
				Expect(ok).To(BeTrue())
			})

			It("the version should be empty", func() {
				Expect(ver).Should(Equal(testVersion))
			})
		})

		Context("should return a valid response if the version is in the versions array (middle element)", func() {
			hcs := &HyperConvergedStatus{
				Conditions:     []conditionsv1.Condition{},
				RelatedObjects: []corev1.ObjectReference{},
				Versions: Versions{
					{Name: "aaa", Version: "1.2.3"},
					{Name: testName, Version: testVersion},
					{Name: "ccc", Version: "7.8.9"},
				},
			}

			ver, ok := hcs.GetVersion(testName)

			It("should not find the version", func() {
				Expect(ok).To(BeTrue())
			})

			It("the version should be empty", func() {
				Expect(ver).Should(Equal(testVersion))
			})
		})

		Context("should return a valid response if the version is in the versions array (last element)", func() {
			hcs := &HyperConvergedStatus{
				Conditions:     []conditionsv1.Condition{},
				RelatedObjects: []corev1.ObjectReference{},
				Versions: Versions{
					{Name: "aaa", Version: "1.2.3"},
					{Name: "bbb", Version: "4.5.6"},
					{Name: testName, Version: testVersion},
				},
			}

			ver, ok := hcs.GetVersion(testName)

			It("should not find the version", func() {
				Expect(ok).To(BeTrue())
			})

			It("the version should be empty", func() {
				Expect(ver).Should(Equal(testVersion))
			})
		})

		// TODO: add tests on nodeselectors and tolerations

	})

	Context("Test Auto generated code", func() {
		bandwidthPerMigration := "64Mi"
		completionTimeoutPerGiB := int64(800)
		parallelMigrationsPerCluster := uint32(5)
		parallelOutboundMigrationsPerNode := uint32(2)
		progressTimeout := int64(150)
		ScratchSpaceStorageClass := "ScratchSpaceStorageClassValue"

		hco := HyperConverged{
			TypeMeta: metav1.TypeMeta{
				Kind: "Hyperconverged",
			},
			ObjectMeta: metav1.ObjectMeta{
				Name:      "Hyperconverged",
				Namespace: "namespace",
			},
			Spec: HyperConvergedSpec{
				LocalStorageClassName: "LocalStorageClassName",
				Infra: HyperConvergedConfig{
					NodePlacement: &sdkapi.NodePlacement{
						NodeSelector: map[string]string{"key1": "value1", "key2": "value2"},
					},
				},
				Workloads: HyperConvergedConfig{
					NodePlacement: &sdkapi.NodePlacement{
						Tolerations: []corev1.Toleration{
							{
								Key:      "key1",
								Operator: corev1.TolerationOpExists,
								Effect:   corev1.TaintEffectNoSchedule,
							},
						},
					},
				},
				FeatureGates: HyperConvergedFeatureGates{
					WithHostPassthroughCPU: true,
				},
				LiveMigrationConfig: LiveMigrationConfigurations{
					BandwidthPerMigration:             &bandwidthPerMigration,
					CompletionTimeoutPerGiB:           &completionTimeoutPerGiB,
					ParallelMigrationsPerCluster:      &parallelMigrationsPerCluster,
					ParallelOutboundMigrationsPerNode: &parallelOutboundMigrationsPerNode,
					ProgressTimeout:                   &progressTimeout,
				},
				PermittedHostDevices: &PermittedHostDevices{
					PciHostDevices: []PciHostDevice{
						{
							PCIVendorSelector:        "PCIVendorSelector",
							ResourceName:             "ResourceName",
							ExternalResourceProvider: true,
						},
					},
					MediatedDevices: []MediatedHostDevice{
						{
							MDEVNameSelector:         "MDEVNameSelector",
							ResourceName:             "ResourceName",
							ExternalResourceProvider: false,
						},
					},
				},
				CertConfig: HyperConvergedCertConfig{
					CA: CertRotateConfigCA{
						Duration: metav1.Duration{
							Duration: time.Hour * 24 * 365,
						},
						RenewBefore: metav1.Duration{
							Duration: time.Hour * 24,
						},
					},
					Server: CertRotateConfigServer{
						Duration: metav1.Duration{
							Duration: time.Hour * 24 * 365,
						},
						RenewBefore: metav1.Duration{
							Duration: time.Hour * 24,
						},
					},
				},
				ResourceRequirements: &OperandResourceRequirements{
					StorageWorkloads: &corev1.ResourceRequirements{
						Limits: corev1.ResourceList{
							corev1.ResourceCPU: resource.MustParse("300m"),
						},
						Requests: corev1.ResourceList{
							corev1.ResourceCPU: resource.MustParse("250m"),
						},
					},
				},
				ScratchSpaceStorageClass: &ScratchSpaceStorageClass,
				StorageImport:            &StorageImportConfig{InsecureRegistries: []string{"aaa", "bbb", "ccc"}},
				Version:                  "v1.2.3",
			},
			Status: HyperConvergedStatus{
				Conditions: []conditionsv1.Condition{
					{
						Type:   "a condition type",
						Status: "True",
					},
				},
				RelatedObjects: []corev1.ObjectReference{
					{
						Kind:      "kind",
						Name:      "a name",
						Namespace: "a namespace",
					},
				},
				Versions: Versions{
					{
						Name:    "HCO",
						Version: "version",
					},
				},
			},
		}

		It("Should copy the HC type", func() {
			aCopy := hco.DeepCopy()

			Expect(aCopy.Kind).Should(Equal("Hyperconverged"))
			Expect(aCopy.Name).Should(Equal("Hyperconverged"))
			Expect(aCopy.Namespace).Should(Equal("namespace"))
			Expect(aCopy.Spec.LocalStorageClassName).Should(Equal("LocalStorageClassName"))
			Expect(aCopy.Spec.Infra.NodePlacement).Should(Equal(hco.Spec.Infra.NodePlacement))
			Expect(aCopy.Spec.Workloads.NodePlacement).Should(Equal(hco.Spec.Workloads.NodePlacement))
			Expect(aCopy.Spec.FeatureGates.WithHostPassthroughCPU).Should(BeTrue())
		})

		It("Should fail to compare if modified", func() {
			aCopy := hco.DeepCopy()
			aCopy.Spec.Infra.NodePlacement.NodeSelector["key1"] = "otherValue"
			Expect(aCopy.Spec).ShouldNot(Equal(hco.Spec))
			Expect(aCopy.Spec.Infra.NodePlacement).ShouldNot(Equal(hco.Spec.Infra.NodePlacement))
		})
	})
})
