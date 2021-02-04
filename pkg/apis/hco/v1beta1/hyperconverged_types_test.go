package v1beta1

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	conditionsv1 "github.com/openshift/custom-resource-status/conditions/v1"
	corev1 "k8s.io/api/core/v1"
	"testing"
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

	Context("HyperConvergedFeatureGates", func() {

		Context("Test IsHotplugVolumesEnabled", func() {
			It("Should return false if HyperConvergedFeatureGates is nil", func() {
				var fgs *HyperConvergedFeatureGates = nil
				Expect(fgs.IsHotplugVolumesEnabled()).To(BeFalse())
			})

			It("Should return false if HotplugVolumes does not exist", func() {
				fgs := &HyperConvergedFeatureGates{}
				Expect(fgs.IsHotplugVolumesEnabled()).To(BeFalse())
			})

			It("Should return false if HotplugVolumes is false", func() {
				disabled := false
				fgs := &HyperConvergedFeatureGates{
					HotplugVolumes: &disabled,
				}
				Expect(fgs.IsHotplugVolumesEnabled()).To(BeFalse())
			})

			It("Should return false if HotplugVolumes is true", func() {
				enabled := true
				fgs := &HyperConvergedFeatureGates{
					HotplugVolumes: &enabled,
				}
				Expect(fgs.IsHotplugVolumesEnabled()).To(BeTrue())
			})
		})

		Context("Test IsSRIOVLiveMigrationEnabled", func() {
			It("Should return false if HyperConvergedFeatureGates is nil", func() {
				var fgs *HyperConvergedFeatureGates = nil
				Expect(fgs.IsSRIOVLiveMigrationEnabled()).To(BeFalse())
			})

			It("Should return false if IsSRIOVLiveMigrationEnabled does not exist", func() {
				fgs := &HyperConvergedFeatureGates{}
				Expect(fgs.IsSRIOVLiveMigrationEnabled()).To(BeFalse())
			})

			It("Should return false if IsSRIOVLiveMigrationEnabled is false", func() {
				disabled := false
				fgs := &HyperConvergedFeatureGates{
					SRIOVLiveMigration: &disabled,
				}
				Expect(fgs.IsSRIOVLiveMigrationEnabled()).To(BeFalse())
			})

			It("Should return false if IsSRIOVLiveMigrationEnabled is true", func() {
				enabled := true
				fgs := &HyperConvergedFeatureGates{
					SRIOVLiveMigration: &enabled,
				}
				Expect(fgs.IsSRIOVLiveMigrationEnabled()).To(BeTrue())
			})
		})

		Context("Test IsWithHostPassthroughCPUEnabled", func() {
			It("Should return false if HyperConvergedFeatureGates is nil", func() {
				var fgs *HyperConvergedFeatureGates = nil
				Expect(fgs.IsWithHostPassthroughCPUEnabled()).To(BeFalse())
			})

			It("Should return false if WithHostPassthroughCPU does not exist", func() {
				fgs := &HyperConvergedFeatureGates{}
				Expect(fgs.IsWithHostPassthroughCPUEnabled()).To(BeFalse())
			})

			It("Should return false if WithHostPassthroughCPU is false", func() {
				disabled := false
				fgs := &HyperConvergedFeatureGates{
					WithHostPassthroughCPU: &disabled,
				}
				Expect(fgs.IsWithHostPassthroughCPUEnabled()).To(BeFalse())
			})

			It("Should return false if WithHostPassthroughCPU is true", func() {
				enabled := true
				fgs := &HyperConvergedFeatureGates{
					WithHostPassthroughCPU: &enabled,
				}
				Expect(fgs.IsWithHostPassthroughCPUEnabled()).To(BeTrue())
			})
		})

		Context("Test IsWithHostModelCPUEnabled", func() {
			It("Should return false if HyperConvergedFeatureGates is nil", func() {
				var fgs *HyperConvergedFeatureGates = nil
				Expect(fgs.IsWithHostModelCPUEnabled()).To(BeFalse())
			})

			It("Should return false if WithHostModelCPU does not exist", func() {
				fgs := &HyperConvergedFeatureGates{}
				Expect(fgs.IsWithHostModelCPUEnabled()).To(BeFalse())
			})

			It("Should return false if WithHostModelCPU is false", func() {
				disabled := false
				fgs := &HyperConvergedFeatureGates{
					WithHostModelCPU: &disabled,
				}
				Expect(fgs.IsWithHostModelCPUEnabled()).To(BeFalse())
			})

			It("Should return false if WithHostModelCPU is true", func() {
				enabled := true
				fgs := &HyperConvergedFeatureGates{
					WithHostModelCPU: &enabled,
				}
				Expect(fgs.IsWithHostModelCPUEnabled()).To(BeTrue())
			})
		})
	})
})
