package hyperconverged

import (
	"os"
	"path"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/ginkgo/extensions/table"
	. "github.com/onsi/gomega"

	hcov1beta1 "github.com/kubevirt/hyperconverged-cluster-operator/pkg/apis/hco/v1beta1"
	"github.com/kubevirt/hyperconverged-cluster-operator/pkg/controller/common"
	"github.com/kubevirt/hyperconverged-cluster-operator/pkg/controller/commonTestUtils"
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

})

func copyTestFile(filename string) error {
	testFilesLocation := getTestFilesLocation() + "/upgradePatches"
	err := commonTestUtils.CopyFile(origFile, path.Join(testFilesLocation, filename))
	return err
}
