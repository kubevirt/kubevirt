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
	RunSpecs(t, "HyperConvergedStatus Suite")
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
	})
})
