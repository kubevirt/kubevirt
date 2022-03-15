package hyperconverged

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	hcov1beta1 "github.com/kubevirt/hyperconverged-cluster-operator/api/v1beta1"
)

const (
	testName       = "aName"
	testVersion    = "aVersion"
	testOldVersion = "anOldVersion"
)

var _ = Describe("Test utilities for HCO versions", func() {
	Describe("HyperConvergedStatus.UpdateVersion", func() {
		Context("Should be able to add a new version to a nil version array", func() {
			hcs := &hcov1beta1.HyperConvergedStatus{
				Conditions:     []metav1.Condition{},
				RelatedObjects: []corev1.ObjectReference{},
			}

			UpdateVersion(hcs, testName, testVersion)

			It("Versions array should be with one element", func() {
				Expect(hcs.Versions).Should(HaveLen(1))
			})

			It(`The version name should be "aName"`, func() {
				Expect(hcs.Versions[0].Name).Should(Equal(testName))
			})

			It(`The version should be "aVersion"`, func() {
				Expect(hcs.Versions[0].Version).Should(Equal(testVersion))
			})
		})

		Context("Should be able to add a new version to an empty version array", func() {
			hcs := &hcov1beta1.HyperConvergedStatus{
				Conditions:     []metav1.Condition{},
				RelatedObjects: []corev1.ObjectReference{},
				Versions:       []hcov1beta1.Version{},
			}

			UpdateVersion(hcs, testName, testVersion)

			It("Versions array should be with one element", func() {
				Expect(hcs.Versions).Should(HaveLen(1))
			})

			It(`The version name should be "aName"`, func() {
				Expect(hcs.Versions[0].Name).Should(Equal(testName))
			})

			It(`The version should be "aVersion"`, func() {
				Expect(hcs.Versions[0].Version).Should(Equal(testVersion))
			})
		})

		Context("Should be able to add a new version to an existing version array", func() {
			hcs := &hcov1beta1.HyperConvergedStatus{
				Conditions:     []metav1.Condition{},
				RelatedObjects: []corev1.ObjectReference{},
				Versions: []hcov1beta1.Version{
					{Name: "aaa", Version: "1.2.3"},
					{Name: "bbb", Version: "4.5.6"},
					{Name: "ccc", Version: "7.8.9"},
				},
			}

			UpdateVersion(hcs, testName, testVersion)

			It("Versions array should be with four elements", func() {
				Expect(hcs.Versions).Should(HaveLen(4))
			})

			It(`The version name should be "aName"`, func() {
				Expect(hcs.Versions[3].Name).Should(Equal(testName))
			})

			It(`The version should be "aVersion"`, func() {
				Expect(hcs.Versions[3].Version).Should(Equal(testVersion))
			})
		})

		Context("Should be able to update a new version in an existing version array (first element)", func() {
			hcs := &hcov1beta1.HyperConvergedStatus{
				Conditions:     []metav1.Condition{},
				RelatedObjects: []corev1.ObjectReference{},
				Versions: []hcov1beta1.Version{
					{Name: testName, Version: testOldVersion},
					{Name: "bbb", Version: "4.5.6"},
					{Name: "ccc", Version: "7.8.9"},
				},
			}

			UpdateVersion(hcs, testName, testVersion)

			It("Versions array should be with three elements", func() {
				Expect(hcs.Versions).Should(HaveLen(3))
			})

			It(`The version name should be "aName"`, func() {
				Expect(hcs.Versions[0].Name).Should(Equal(testName))
			})

			It(`The version should be "aVersion"`, func() {
				Expect(hcs.Versions[0].Version).Should(Equal(testVersion))
			})
		})

		Context("Should be able to update a new version in an existing version array (middle element)", func() {
			hcs := &hcov1beta1.HyperConvergedStatus{
				Conditions:     []metav1.Condition{},
				RelatedObjects: []corev1.ObjectReference{},
				Versions: []hcov1beta1.Version{
					{Name: "aaa", Version: "1.2.3"},
					{Name: testName, Version: testOldVersion},
					{Name: "ccc", Version: "7.8.9"},
				},
			}

			UpdateVersion(hcs, testName, testVersion)

			It("Versions array should be with three elements", func() {
				Expect(hcs.Versions).Should(HaveLen(3))
			})

			It(`The version name should be "aName"`, func() {
				Expect(hcs.Versions[1].Name).Should(Equal(testName))
			})

			It(`The version should be "aVersion"`, func() {
				Expect(hcs.Versions[1].Version).Should(Equal(testVersion))
			})
		})

		Context("Should be able to update a new version in an existing version array (last element)", func() {
			hcs := &hcov1beta1.HyperConvergedStatus{
				Conditions:     []metav1.Condition{},
				RelatedObjects: []corev1.ObjectReference{},
				Versions: []hcov1beta1.Version{
					{Name: "aaa", Version: "1.2.3"},
					{Name: "bbb", Version: "4.5.6"},
					{Name: testName, Version: testOldVersion},
				},
			}

			UpdateVersion(hcs, testName, testVersion)

			It("Versions array should be with three elements", func() {
				Expect(hcs.Versions).Should(HaveLen(3))
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
			hcs := &hcov1beta1.HyperConvergedStatus{
				Conditions:     []metav1.Condition{},
				RelatedObjects: []corev1.ObjectReference{},
			}

			ver, ok := GetVersion(hcs, testName)

			It("should not find the version", func() {
				Expect(ok).To(BeFalse())
			})
			It("the version should be empty", func() {
				Expect(ver).To(BeEmpty())
			})
		})

		Context("should return empty response for empty array", func() {
			hcs := &hcov1beta1.HyperConvergedStatus{
				Conditions:     []metav1.Condition{},
				RelatedObjects: []corev1.ObjectReference{},
				Versions:       []hcov1beta1.Version{},
			}

			ver, ok := GetVersion(hcs, testName)

			It("should not find the version", func() {
				Expect(ok).To(BeFalse())
			})

			It("the version should be empty", func() {
				Expect(ver).To(BeEmpty())
			})
		})

		Context("should return empty response if the version is not in the versions array", func() {
			hcs := &hcov1beta1.HyperConvergedStatus{
				Conditions:     []metav1.Condition{},
				RelatedObjects: []corev1.ObjectReference{},
				Versions: []hcov1beta1.Version{
					{Name: "aaa", Version: "1.2.3"},
					{Name: "bbb", Version: "4.5.6"},
					{Name: "ccc", Version: "7.8.9"},
				},
			}

			ver, ok := GetVersion(hcs, testName)

			It("should not find the version", func() {
				Expect(ok).To(BeFalse())
			})

			It("the version should be empty", func() {
				Expect(ver).To(BeEmpty())
			})
		})

		Context("should return a valid response if the version is in the versions array (first element)", func() {
			hcs := &hcov1beta1.HyperConvergedStatus{
				Conditions:     []metav1.Condition{},
				RelatedObjects: []corev1.ObjectReference{},
				Versions: []hcov1beta1.Version{
					{Name: testName, Version: testVersion},
					{Name: "bbb", Version: "4.5.6"},
					{Name: "ccc", Version: "7.8.9"},
				},
			}

			ver, ok := GetVersion(hcs, testName)

			It("should not find the version", func() {
				Expect(ok).To(BeTrue())
			})

			It("the version should be empty", func() {
				Expect(ver).Should(Equal(testVersion))
			})
		})

		Context("should return a valid response if the version is in the versions array (middle element)", func() {
			hcs := &hcov1beta1.HyperConvergedStatus{
				Conditions:     []metav1.Condition{},
				RelatedObjects: []corev1.ObjectReference{},
				Versions: []hcov1beta1.Version{
					{Name: "aaa", Version: "1.2.3"},
					{Name: testName, Version: testVersion},
					{Name: "ccc", Version: "7.8.9"},
				},
			}

			ver, ok := GetVersion(hcs, testName)

			It("should not find the version", func() {
				Expect(ok).To(BeTrue())
			})

			It("the version should be empty", func() {
				Expect(ver).Should(Equal(testVersion))
			})
		})

		Context("should return a valid response if the version is in the versions array (last element)", func() {
			hcs := &hcov1beta1.HyperConvergedStatus{
				Conditions:     []metav1.Condition{},
				RelatedObjects: []corev1.ObjectReference{},
				Versions: []hcov1beta1.Version{
					{Name: "aaa", Version: "1.2.3"},
					{Name: "bbb", Version: "4.5.6"},
					{Name: testName, Version: testVersion},
				},
			}

			ver, ok := GetVersion(hcs, testName)

			It("should not find the version", func() {
				Expect(ok).To(BeTrue())
			})

			It("the version should be empty", func() {
				Expect(ver).Should(Equal(testVersion))
			})
		})

		// TODO: add tests on nodeselectors and tolerations

	})

})
