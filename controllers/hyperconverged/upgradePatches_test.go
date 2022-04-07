package hyperconverged

import (
	"os"
	"path"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	hcov1beta1 "github.com/kubevirt/hyperconverged-cluster-operator/api/v1beta1"
	"github.com/kubevirt/hyperconverged-cluster-operator/controllers/common"
	"github.com/kubevirt/hyperconverged-cluster-operator/controllers/commonTestUtils"
)

var origFile string

var _ = Describe("upgradePatches", func() {

	BeforeEach(func() {
		wd, _ := os.Getwd()
		origFile = path.Join(wd, "upgradePatches.json")
		err := commonTestUtils.CopyFile(origFile+".orig", origFile)
		Expect(err).ToNot(HaveOccurred())
		hcoUpgradeChangesRead = false
	})

	AfterEach(func() {
		err := os.Remove(origFile + ".orig")
		Expect(err).ToNot(HaveOccurred())
		hcoUpgradeChangesRead = false
	})

	Context("readUpgradeChangesFromFile", func() {

		var hco *hcov1beta1.HyperConverged
		var req *common.HcoRequest

		BeforeEach(func() {
			hco = commonTestUtils.NewHco()
			req = commonTestUtils.NewReq(hco)
		})

		AfterEach(func() {
			err := commonTestUtils.CopyFile(origFile, origFile+".orig")
			Expect(err).ToNot(HaveOccurred())
		})

		It("should correctly parse and validate actual upgradePatches.json", func() {
			err := validateUpgradePatches(req)
			Expect(err).ToNot(HaveOccurred())
		})

		It("should correctly parse and validate empty upgradePatches", func() {
			err := copyTestFile("empty.json")
			Expect(err).ToNot(HaveOccurred())

			err = validateUpgradePatches(req)
			Expect(err).ToNot(HaveOccurred())
		})

		It("should fail parsing upgradePatches with bad json", func() {
			err := copyTestFile("badJson.json")
			Expect(err).ToNot(HaveOccurred())

			err = validateUpgradePatches(req)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).Should(HavePrefix("invalid character"))
		})

		Context("hcoCRPatchList", func() {

			It("should fail validating upgradePatches with bad semver ranges", func() {
				err := copyTestFile("badSemverRange.json")
				Expect(err).ToNot(HaveOccurred())

				err = validateUpgradePatches(req)
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).Should(HavePrefix("Could not get version from string:"))
			})

			DescribeTable(
				"should fail validating upgradePatches with bad patches",
				func(filename, message string) {
					err := copyTestFile(filename)
					Expect(err).ToNot(HaveOccurred())

					err = validateUpgradePatches(req)
					Expect(err).To(HaveOccurred())
					Expect(err.Error()).Should(HavePrefix(message))
				},
				Entry(
					"bad operation kind",
					"badPatches1.json",
					"Unexpected kind:",
				),
				Entry(
					"not on spec",
					"badPatches2.json",
					"can only modify spec fields",
				),
				Entry(
					"unexisting path",
					"badPatches3.json",
					"replace operation does not apply: doc is missing path:",
				),
			)

		})

		Context("objectsToBeRemoved", func() {

			It("should fail validating upgradePatches with bad semver ranges", func() {
				err := copyTestFile("badSemverRangeOR.json")
				Expect(err).ToNot(HaveOccurred())

				err = validateUpgradePatches(req)
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).Should(HavePrefix("Could not get version from string:"))
			})

			DescribeTable(
				"should fail validating upgradePatches with bad patches",
				func(filename, message string) {
					err := copyTestFile(filename)
					Expect(err).ToNot(HaveOccurred())

					err = validateUpgradePatches(req)
					Expect(err).To(HaveOccurred())
					Expect(err.Error()).Should(HavePrefix(message))
				},
				Entry(
					"empty object kind",
					"badObject1.json",
					"missing object kind",
				),
				Entry(
					"missing object kind",
					"badObject1m.json",
					"missing object kind",
				),
				Entry(
					"empty object API version",
					"badObject2.json",
					"missing object API version",
				),
				Entry(
					"missing object API version",
					"badObject2m.json",
					"missing object API version",
				),
				Entry(
					"empty object name",
					"badObject3.json",
					"missing object name",
				),
				Entry(
					"missing object name",
					"badObject3m.json",
					"missing object name",
				),
			)

		})

	})

})

func copyTestFile(filename string) error {
	testFilesLocation := getTestFilesLocation() + "/upgradePatches"
	err := commonTestUtils.CopyFile(origFile, path.Join(testFilesLocation, filename))
	return err
}
