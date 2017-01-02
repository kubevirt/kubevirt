package util

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/jeevatkm/go-model"
)

var _ = Describe("Mapper", func() {

	type E struct {
		X string
	}

	type F struct {
		X string
	}

	type A struct {
		X E
	}

	type B struct {
		X F
	}

	type C struct {
		X *E
	}

	type D struct {
		X *F
	}

	BeforeSuite(func() {
		AddConversion(&E{}, &F{})
		AddPtrConversion((**E)(nil), (**F)(nil))
	})

	Describe("converting objects", func() {
		Context("with concrete types", func() {
			It("should succeed from srt to dest", func() {

				a := &A{X: E{X: "test"}}
				b := &B{}
				errs := model.Copy(b, a)
				Expect(errs).To(BeEmpty())
				Expect(b.X.X).To(Equal("test"))
			})
			It("should succeed from dest to src", func() {

				b := &B{X: F{X: "test"}}
				a := &A{}
				errs := model.Copy(a, b)
				Expect(errs).To(BeEmpty())
				Expect(a.X.X).To(Equal("test"))
			})
		})
		Context("with pointer types", func() {
			It("should succeed from src to dest", func() {

				c := &C{X: &E{X: "test"}}
				d := &D{}
				errs := model.Copy(d, c)
				Expect(errs).To(BeEmpty())
				Expect(d.X.X).To(Equal("test"))
			})
			It("should succeed from dest to src", func() {

				d := &D{X: &F{X: "test"}}
				c := &C{}
				errs := model.Copy(c, d)
				Expect(errs).To(BeEmpty())
				Expect(c.X.X).To(Equal("test"))
			})
		})
	})

	AfterSuite(func() {

	})
})
