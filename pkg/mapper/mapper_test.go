/*
 * This file is part of the KubeVirt project
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 *
 * Copyright 2017 Red Hat, Inc.
 *
 */

package mapper

import (
	"github.com/jeevatkm/go-model"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
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
