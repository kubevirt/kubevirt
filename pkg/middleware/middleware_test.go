package middleware

import (
	"errors"
	"github.com/go-kit/kit/log"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"golang.org/x/net/context"
	"kubevirt.io/kubevirt/pkg/precond"
)

var _ = Describe("Middleware", func() {
	Describe("Call", func() {
		Context("with precond.PreconditionError panic", func() {
			It("returns application level PreconditionError", func() {
				data, err := InternalErrorMiddleware(log.NewLogfmtLogger(GinkgoWriter))(
					func(ctx context.Context, request interface{}) (interface{}, error) {
						precond.MustNotBeEmpty("")
						return nil, nil
					})(nil, nil)
				Expect(err).ShouldNot(HaveOccurred())
				_, ok := data.(*PreconditionError)
				Expect(ok).Should(BeTrue())
			})
		})
		Context("with generic panic", func() {
			It("returns application level InternalServerError", func() {
				data, err := InternalErrorMiddleware(log.NewLogfmtLogger(GinkgoWriter))(
					func(ctx context.Context, request interface{}) (interface{}, error) {
						panic("generic one")
					})(nil, nil)
				Expect(err).ShouldNot(HaveOccurred())
				_, ok := data.(*InternalServerError)
				Expect(ok).Should(BeTrue())
			})
		})
		Context("without panic", func() {
			It("returns the normal endpoint results", func() {
				d := "everything"
				e := errors.New("is fine")
				data, err := InternalErrorMiddleware(log.NewLogfmtLogger(GinkgoWriter))(
					func(ctx context.Context, request interface{}) (interface{}, error) {
						return d, e
					})(nil, nil)
				Expect(err).To(BeIdenticalTo(e))
				Expect(data).To(BeIdenticalTo(d))
			})
		})
	})
})
