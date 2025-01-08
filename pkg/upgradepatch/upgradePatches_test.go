package upgradepatch

import (
	"os"
	"path"
	"slices"
	"strings"
	"sync"
	"testing"

	"github.com/blang/semver/v4"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/types"
	"k8s.io/utils/ptr"

	"github.com/kubevirt/hyperconverged-cluster-operator/controllers/commontestutils"
	"github.com/kubevirt/hyperconverged-cluster-operator/pkg/components"
)

const (
	pkgDirectory = "pkg/upgradepatch"
	testFilesLoc = "test-files"
)

var (
	origFile      string
	hcCRBytesOrig []byte
)

func resetOnce() {
	once = &sync.Once{}
}

var _ = Describe("upgradePatches", func() {

	BeforeEach(func() {
		wd, _ := os.Getwd()
		origFile = path.Join(wd, "upgradePatches.json")
		Expect(commontestutils.CopyFile(origFile+".orig", origFile)).To(Succeed())
		resetOnce()
		hcCRBytes = slices.Clone(hcCRBytesOrig)
	})

	AfterEach(func() {
		Expect(os.Remove(origFile + ".orig")).To(Succeed())
		resetOnce()
	})

	Context("readUpgradeChangesFromFile", func() {
		AfterEach(func() {
			Expect(commontestutils.CopyFile(origFile, origFile+".orig")).To(Succeed())
			resetOnce()
			Expect(Init(GinkgoLogr)).To(Succeed())
		})

		It("should correctly parse and validate actual upgradePatches.json", func() {
			Expect(Init(GinkgoLogr)).To(Succeed())
		})

		It("should correctly parse and validate empty upgradePatches", func() {
			Expect(copyTestFile("empty.json")).To(Succeed())
			Expect(Init(GinkgoLogr)).To(Succeed())
		})

		It("should fail parsing upgradePatches with bad json", func() {
			Expect(copyTestFile("badJson.json")).To(Succeed())

			err := Init(GinkgoLogr)
			Expect(err).To(MatchError(HavePrefix("invalid character")))
		})

		Context("hcoCRPatchList", func() {

			It("should fail validating upgradePatches with bad semver ranges", func() {
				Expect(copyTestFile("badSemverRange.json")).To(Succeed())

				err := Init(GinkgoLogr)
				Expect(err).To(MatchError(HavePrefix("Could not get version from string:")))
			})

			DescribeTable(
				"should fail validating upgradePatches with bad patches",
				func(filename, message string) {
					Expect(copyTestFile(filename)).To(Succeed())
					Expect(Init(GinkgoLogr)).To(MatchError(HavePrefix(message)))
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

			DescribeTable(
				"should handle MissingPathOnRemove according to jsonPatchApplyOptions",
				func(filename string, expectedErr bool, message string) {
					Expect(copyTestFile(filename)).To(Succeed())

					err := Init(GinkgoLogr)
					if expectedErr {
						Expect(err).To(MatchError(HavePrefix(message)))
					} else {
						Expect(err).ToNot(HaveOccurred())
					}
				},
				Entry(
					"without jsonPatchApplyOptions",
					"badPatches4.json",
					true,
					"remove operation does not apply: doc is missing path: ",
				),
				Entry(
					"with AllowMissingPathOnRemove on jsonPatchApplyOptions",
					"badPatches5.json",
					false,
					"",
				),
				Entry(
					"without jsonPatchApplyOptions",
					"badPatches6.json",
					true,
					"add operation does not apply: doc is missing path: ",
				),
				Entry(
					"with EnsurePathExistsOnAdd on jsonPatchApplyOptions",
					"badPatches7.json",
					false,
					"",
				),
			)

		})

		Context("objectsToBeRemoved", func() {

			It("should fail validating upgradePatches with bad semver ranges", func() {
				Expect(copyTestFile("badSemverRangeOR.json")).To(Succeed())
				Expect(Init(GinkgoLogr)).To(MatchError(HavePrefix("Could not get version from string:")))
			})

			DescribeTable(
				"should fail validating upgradePatches with bad patches",
				func(filename, message string) {
					Expect(copyTestFile(filename)).To(Succeed())
					Expect(Init(GinkgoLogr)).To(MatchError(HavePrefix(message)))
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

	Context("check semverRange type", func() {
		DescribeTable("check isAffectedRange", func(verRange, ver string, m types.GomegaMatcher) {
			vr, err := newSemverRange(verRange)
			Expect(err).NotTo(HaveOccurred())

			v, err := semver.Parse(ver)
			Expect(err).NotTo(HaveOccurred())
			Expect(vr.isAffectedRange(v)).To(m)
		},
			Entry("", ">4.16.0", "4.16.9", BeTrue()),
			Entry("", ">4.16.0", "4.17.2", BeTrue()),
			Entry("", ">4.16.0", "4.16.0", BeFalse()),
			Entry("", ">4.16.0", "4.15.2", BeFalse()),

			Entry("", "<=4.17.0", "4.16.4", BeTrue()),
			Entry("", "<=4.17.0", "4.17.2", BeFalse()),
			Entry("", "<4.17.3", "4.17.2", BeTrue()),
			Entry("", "<4.17.3", "4.17.3", BeFalse()),
			Entry("", "<4.17.3", "4.17.4", BeFalse()),
			Entry("", "<4.17.3", "4.16.9", BeTrue()),
		)
	})

	//nolint:staticcheck // ignore SA1019 for old code
	Context("check patches", func() {
		It("should apply changes as defined in the upgradePatches.json file", func() {
			hc := components.GetOperatorCR()
			hc.Spec.FeatureGates.DeployKubevirtIpamController = ptr.To(false)
			hc.Spec.FeatureGates.EnableManagedTenantQuota = ptr.To(false)
			hc.Spec.FeatureGates.EnableManagedTenantQuota = ptr.To(false)
			hc.Spec.FeatureGates.NonRoot = ptr.To(false)
			hc.Spec.FeatureGates.WithHostPassthroughCPU = ptr.To(false)
			hc.Spec.FeatureGates.PrimaryUserDefinedNetworkBinding = ptr.To(false)

			ver, err := semver.Parse("1.13.9")
			Expect(err).NotTo(HaveOccurred())

			newHc, err := ApplyUpgradePatch(GinkgoLogr, hc, ver)
			Expect(err).NotTo(HaveOccurred())

			Expect(newHc.Spec.FeatureGates.DeployKubevirtIpamController).To(BeNil())
			Expect(newHc.Spec.FeatureGates.EnableManagedTenantQuota).To(BeNil())
			Expect(newHc.Spec.FeatureGates.EnableManagedTenantQuota).To(BeNil())
			Expect(newHc.Spec.FeatureGates.NonRoot).To(BeNil())
			Expect(newHc.Spec.FeatureGates.WithHostPassthroughCPU).To(BeNil())
			Expect(newHc.Spec.FeatureGates.PrimaryUserDefinedNetworkBinding).To(BeNil())
		})
	})
})

func copyTestFile(filename string) error {
	return commontestutils.CopyFile(origFile, path.Join(getTestFilesLocation(), filename))
}

func getTestFilesLocation() string {
	wd, err := os.Getwd()
	Expect(err).ToNot(HaveOccurred())
	if strings.HasSuffix(wd, pkgDirectory) {
		return testFilesLoc
	}
	return path.Join(pkgDirectory, testFilesLoc)
}

func TestUpgradePatch(t *testing.T) {
	RegisterFailHandler(Fail)

	var (
		destFile string
	)

	BeforeSuite(func() {
		wd, _ := os.Getwd()
		destFile = path.Join(wd, "upgradePatches.json")
		Expect(commontestutils.CopyFile(destFile, path.Join(getTestFilesLocation(), "upgradePatches.json"))).To(Succeed())
		hcCRBytesOrig = slices.Clone(hcCRBytes)
	})

	AfterSuite(func() {
		Expect(os.Remove(destFile)).To(Succeed())
		hcCRBytes = hcCRBytesOrig
	})

	RunSpecs(t, "Upgrade Patches Suite")
}
