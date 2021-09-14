package hyperconverged

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"github.com/kubevirt/hyperconverged-cluster-operator/pkg/controller/commonTestUtils"
)

var _ = Describe("test data import schedule", func() {
	const schedule = "42 */12 * * *"

	It("should update the status and the variable if both are empty", func() {
		hco := commonTestUtils.NewHco()
		req := commonTestUtils.NewReq(hco)

		applyDataImportSchedule(req)

		Expect(dataImportSchedule).Should(MatchRegexp(`\d+ \*/12 \* \* \*`))
		Expect(hco.Status.DataImportSchedule).Should(Equal(dataImportSchedule))
		Expect(req.StatusDirty).Should(BeTrue())
	})

	It("should update the status if the variable is set", func() {
		hco := commonTestUtils.NewHco()
		req := commonTestUtils.NewReq(hco)

		dataImportSchedule = schedule

		applyDataImportSchedule(req)

		Expect(dataImportSchedule).Should(Equal(schedule))
		Expect(hco.Status.DataImportSchedule).Should(Equal(schedule))
		Expect(req.StatusDirty).Should(BeTrue())
	})

	It("should update the variable if it empty and the status is set", func() {
		hco := commonTestUtils.NewHco()
		hco.Status.DataImportSchedule = schedule
		req := commonTestUtils.NewReq(hco)

		dataImportSchedule = ""

		applyDataImportSchedule(req)

		Expect(dataImportSchedule).Should(Equal(schedule))
		Expect(hco.Status.DataImportSchedule).Should(Equal(schedule))
		Expect(req.StatusDirty).Should(BeFalse())
	})

	It("should update the variable if it different than the status", func() {
		hco := commonTestUtils.NewHco()
		hco.Status.DataImportSchedule = schedule
		req := commonTestUtils.NewReq(hco)

		dataImportSchedule = "24 */12 * * *"

		applyDataImportSchedule(req)

		Expect(dataImportSchedule).Should(Equal(schedule))
		Expect(hco.Status.DataImportSchedule).Should(Equal(schedule))
		Expect(req.StatusDirty).Should(BeFalse())
	})
})
