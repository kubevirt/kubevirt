package hyperconverged

import (
	"regexp"
	"strconv"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"github.com/kubevirt/hyperconverged-cluster-operator/pkg/controller/commonTestUtils"
)

var _ = Describe("test data import schedule", func() {
	const schedule = "42 5/12 * * *"

	It("should update the status and the variable if both are empty", func() {
		regex := `(\d+) (\d+)/12 \* \* \*`

		for i := 0; i < 1000; i++ { // testing random number - need some statistic confidence, so running this 1000 times
			dataImportSchedule = ""
			hco := commonTestUtils.NewHco()
			req := commonTestUtils.NewReq(hco)

			applyDataImportSchedule(req)

			Expect(dataImportSchedule).Should(MatchRegexp(regex))

			rx := regexp.MustCompile(regex)
			groups := rx.FindStringSubmatch(dataImportSchedule)
			Expect(groups).To(HaveLen(3))
			minute, err := strconv.Atoi(groups[1])

			Expect(err).ToNot(HaveOccurred())
			Expect(minute).Should(BeNumerically(">=", 0), "minute should be grater than or equal to 0; cron expression is: %s", dataImportSchedule)
			Expect(minute).Should(BeNumerically("<", 60), "minute should br less than 60; cron expression is: %s", dataImportSchedule)

			hour, err := strconv.Atoi(groups[2])
			Expect(err).ToNot(HaveOccurred())
			Expect(hour).Should(BeNumerically(">=", 0), "hour should be grater than or equal to 0; cron expression is: %s", dataImportSchedule)
			Expect(hour).Should(BeNumerically("<", 12), "hour should br less than 12; cron expression is: %s", dataImportSchedule)

			Expect(hco.Status.DataImportSchedule).Should(Equal(dataImportSchedule))
			Expect(req.StatusDirty).Should(BeTrue())
		}
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

		dataImportSchedule = "24 2/12 * * *"

		applyDataImportSchedule(req)

		Expect(dataImportSchedule).Should(Equal(schedule))
		Expect(hco.Status.DataImportSchedule).Should(Equal(schedule))
		Expect(req.StatusDirty).Should(BeFalse())
	})
})
