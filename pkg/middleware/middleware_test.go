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

package middleware

import (
	"errors"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"golang.org/x/net/context"

	"kubevirt.io/client-go/precond"
)

var _ = Describe("Middleware", func() {

	Describe("Call", func() {
		Context("with precond.PreconditionError panic", func() {
			It("returns application level PreconditionError", func() {
				data, err := InternalErrorMiddleware()(
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
				data, err := InternalErrorMiddleware()(
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
				data, err := InternalErrorMiddleware()(
					func(ctx context.Context, request interface{}) (interface{}, error) {
						return d, e
					})(nil, nil)
				Expect(err).To(BeIdenticalTo(e))
				Expect(data).To(BeIdenticalTo(d))
			})
		})
	})
})
