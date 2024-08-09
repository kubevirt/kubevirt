package validating_webhooks

import (
	"bytes"
	"context"
	"net/http"
	"testing"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"kubevirt.io/client-go/testutils"
)

var _ = Describe("Test timeout in admitters", func() {
	It("should use timeout from the context", func() {
		oneSecondsFromNow := time.Now().Add(time.Second)
		ctx, cancel := context.WithTimeout(context.Background(), time.Second)
		defer cancel()

		req, err := getTestReq(ctx, "/test")
		Expect(err).ToNot(HaveOccurred())

		ctxFromReq, cancelFromReq, err := getContextFromRequest(req)
		Expect(err).ToNot(HaveOccurred())
		Expect(cancelFromReq).To(BeNil())

		Expect(ctxFromReq).To(Equal(ctx))
		deadline, ok := ctxFromReq.Deadline()
		Expect(ok).To(BeTrue())
		Expect(deadline).To(BeTemporally("~", oneSecondsFromNow))
	})

	It("should use timeout from the query", func() {
		fifteenSecondsFromNow := time.Now().Add(15 * time.Second)
		req, err := getTestReq(context.Background(), "/test?timeout=15s")
		Expect(err).ToNot(HaveOccurred())

		ctxFromReq, cancel, err := getContextFromRequest(req)
		Expect(err).ToNot(HaveOccurred())
		Expect(cancel).ToNot(BeNil())
		defer cancel()

		deadline, ok := ctxFromReq.Deadline()
		Expect(ok).To(BeTrue())
		Expect(deadline).To(BeTemporally("~", fifteenSecondsFromNow))
	})

	It("should use default timeout if no context, nor timeout query param in the query", func() {
		tenSecondsFromNow := time.Now().Add(10 * time.Second)
		req, err := getTestReq(context.Background(), "/test")
		Expect(err).ToNot(HaveOccurred())

		ctxFromReq, cancel, err := getContextFromRequest(req)
		Expect(err).ToNot(HaveOccurred())
		Expect(cancel).ToNot(BeNil())
		defer cancel()

		deadline, ok := ctxFromReq.Deadline()
		Expect(ok).To(BeTrue())
		Expect(deadline).To(BeTemporally("~", tenSecondsFromNow))
	})

	It("should ignore the query timeout, if the context use timeout", func() {
		// context timeout is shorter than admitter. addmiter shorter than the timeout query param
		fiveSecondsFromNow := time.Now().Add(5 * time.Second)
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		req, err := getTestReq(ctx, "/test?timeout=15s")
		Expect(err).ToNot(HaveOccurred())

		ctxFromReq, cancelFromReq, err := getContextFromRequest(req)
		Expect(err).ToNot(HaveOccurred())
		Expect(cancelFromReq).To(BeNil())

		deadline, ok := ctxFromReq.Deadline()
		Expect(ok).To(BeTrue())
		Expect(deadline).To(BeTemporally("~", fiveSecondsFromNow))
	})

	It("should return error if the time duration is invalid", func() {
		req, err := getTestReq(context.Background(), "/test?timeout=15notvalid")
		Expect(err).ToNot(HaveOccurred())

		ctxFromReq, cancelFromReq, err := getContextFromRequest(req)
		Expect(err).To(HaveOccurred())
		Expect(ctxFromReq).To(BeNil())
		Expect(cancelFromReq).To(BeNil())
	})
})

func getTestReq(ctx context.Context, url string) (*http.Request, error) {
	body := bytes.NewReader([]byte(`{"request":{}}`))
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, body)
	if err != nil {
		return nil, err
	}

	req.Header.Add("Content-Type", "application/json")

	return req, nil
}

func TestValidatingWebhook(t *testing.T) {
	testutils.KubeVirtTestSuiteSetup(t)
}
