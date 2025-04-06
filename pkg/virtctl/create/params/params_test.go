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
 * Copyright The KubeVirt Authors.
 *
 */
package params_test

import (
	"errors"
	"fmt"
	"os"
	"strings"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	. "github.com/onsi/gomega/gstruct"

	"k8s.io/apimachinery/pkg/api/resource"
	"k8s.io/apimachinery/pkg/util/rand"

	"kubevirt.io/kubevirt/pkg/pointer"
	"kubevirt.io/kubevirt/pkg/virtctl/create/params"
)

var _ = Describe("params", func() {
	const (
		testParam    = "testParam"
		flagName     = "testFlag"
		parseFlagErr = "failed to parse \"--testFlag\" flag: "
	)

	Context("NotFoundError", func() {
		It("should have correct message", func() {
			err := params.NotFoundError{Name: testParam}
			Expect(err.Error()).To(Equal(testParam + " must be specified"))
		})

		DescribeTable("should detect equal errors when passed as", func(target error) {
			err := params.NotFoundError{Name: testParam}
			Expect(err.Is(target)).To(BeTrue())
		},
			Entry("value", params.NotFoundError{Name: testParam}),
			Entry("pointer", pointer.P(params.NotFoundError{Name: testParam})),
		)

		It("should detect unequal errors", func() {
			err := params.NotFoundError{Name: testParam}
			Expect(err.Is(os.ErrNotExist)).To(BeFalse())
		})
	})

	Context("FlagErr", func() {
		const (
			format = "My wrapped test error: %s"
			value  = "testvalue"
		)

		It("should have correct message", func() {
			err := params.FlagErr(flagName, format, value)
			Expect(err).To(MatchError(parseFlagErr + fmt.Sprintf(format, value)))
		})

		It("should produce wrapped errors", func() {
			err := params.FlagErr("testFlag", format, value)
			Expect(errors.Unwrap(err)).To(MatchError(fmt.Sprintf(format, value)))
		})
	})

	Context("Supported", func() {
		It("should panic on unsupported object", func() {
			testFn := func() { params.Supported("something") }
			Expect(testFn).To(PanicWith(MatchError("passed in interface needs to be a struct")))
		})

		It("should detect fields with supported types and param tag", func() {
			testStruct := struct {
				Param1 string             `param:"param1"`
				Param2 *uint              `param:"param2"`
				Param3 *resource.Quantity `param:"param3"`
				Param4 []string           `param:"param4"`
			}{}
			Expect(params.Supported(testStruct)).To(Equal("param1:string,param2:uint,param3:resource.Quantity,param4:[]string"))
		})

		It("should ignore fields without param tag", func() {
			testStruct := struct {
				Param                    string `param:"param"`
				ParamWithSupportedType   *uint
				ParamWithUnsupportedType bool
			}{}
			Expect(params.Supported(testStruct)).To(Equal("param:string"))
		})

		It("should panic on fields with unsupported types", func() {
			testStruct := struct {
				Param bool `param:"param"`
			}{}
			testFn := func() { params.Supported(testStruct) }
			Expect(testFn).To(PanicWith(MatchError("unsupported struct field \"Param\" with kind \"bool\"")))
		})
	})

	Context("Map", func() {
		DescribeTable("should fail on invalid params", func(paramsStr, errMsg string) {
			err := params.Map(flagName, paramsStr, "")
			Expect(err).To(MatchError(parseFlagErr + errMsg))
		},
			Entry("empty params", "", "params may not be empty"),
			Entry("param without colon", "nocolon", "params need to have at least one colon: nocolon"),
		)

		DescribeTable("should fail on invalid objects", func(obj interface{}, errMsg string) {
			err := params.Map(flagName, "param:value", obj)
			Expect(err).To(MatchError(parseFlagErr + errMsg))
		},
			Entry("not a pointer", "something", "passed in interface needs to be a pointer"),
			Entry("not a struct", pointer.P("something"), "passed in pointer needs to point to a struct"),
		)

		It("should map supported parameters to supported fields with param tag", func() {
			param1 := rand.String(10)
			param2 := rand.Intn(10)
			param3 := fmt.Sprintf("%dGi", rand.Intn(10))
			var param4 []string
			for range rand.IntnRange(1, 10) {
				param4 = append(param4, rand.String(10))
			}
			paramsStr := fmt.Sprintf("param1:%s,param2:%d,param3:%s,param4:%s", param1, param2, param3, strings.Join(param4, ";"))
			testStruct := &struct {
				Param1 string             `param:"param1"`
				Param2 *uint              `param:"param2"`
				Param3 *resource.Quantity `param:"param3"`
				Param4 []string           `param:"param4"`
			}{}
			Expect(params.Map(flagName, paramsStr, testStruct)).To(Succeed())
			Expect(testStruct.Param1).To(Equal(param1))
			Expect(testStruct.Param2).To(PointTo(Equal(uint(param2))))
			Expect(testStruct.Param3).To(PointTo(Equal(resource.MustParse(param3))))
			Expect(testStruct.Param4).To(Equal(param4))
		})

		It("should ignore fields without param tag", func() {
			const paramsStr = "param:test"
			testStruct := &struct {
				Param                    string `param:"param"`
				ParamWithSupportedType   *uint
				ParamWithUnsupportedType bool
			}{}
			Expect(params.Map(flagName, paramsStr, testStruct)).To(Succeed())
			Expect(testStruct.Param).To(Equal("test"))
			Expect(testStruct.ParamWithSupportedType).To(BeNil())
			Expect(testStruct.ParamWithUnsupportedType).To(BeFalse())
		})

		DescribeTable("should fail on params with invalid values", func(paramsStr, errMsg string) {
			testStruct := &struct {
				Uint     *uint              `param:"uint"`
				Quantity *resource.Quantity `param:"quantity"`
			}{}
			err := params.Map(flagName, paramsStr, testStruct)
			Expect(err).To(MatchError(parseFlagErr + errMsg))
		},
			Entry("uint negative", "uint:-1,quantity:1Gi",
				"failed to parse param \"uint\": strconv.ParseUint: parsing \"-1\": invalid syntax"),
			Entry("uint too large", "uint:9999999999999,quantity:1Gi",
				"failed to parse param \"uint\": strconv.ParseUint: parsing \"9999999999999\": value out of range"),
			Entry("quantity suffix invalid", "uint:8,quantity:1Gu",
				"failed to parse param \"quantity\": unable to parse quantity's suffix"),
		)

		It("should fail on unsupported fields with param tag", func() {
			const paramsStr = "param:true"
			testStruct := &struct {
				Param bool `param:"param"`
			}{}
			err := params.Map(flagName, paramsStr, testStruct)
			Expect(err).To(MatchError(parseFlagErr + "unsupported struct field \"Param\" with kind \"bool\""))
		})

		It("should fail on single unknown param", func() {
			const paramsStr = "param:test,unknown:test"
			testStruct := &struct {
				Param string `param:"param"`
			}{}
			err := params.Map(flagName, paramsStr, testStruct)
			Expect(err).To(MatchError(parseFlagErr + "unknown param(s): unknown:test"))
			// testStruct is modified anyway
			Expect(testStruct.Param).To(Equal("test"))
		})

		It("should fail on multiple unknown params", func() {
			const paramsStr = "param:test,unknown1:test,unknown2:test"
			testStruct := &struct {
				Param string `param:"param"`
			}{}
			err := params.Map(flagName, paramsStr, testStruct)
			Expect(err).To(MatchError(ContainSubstring(parseFlagErr + "unknown param(s): ")))
			Expect(err).To(MatchError(ContainSubstring("unknown1:test")))
			Expect(err).To(MatchError(ContainSubstring("unknown2:test")))
			// testStruct is modified anyway
			Expect(testStruct.Param).To(Equal("test"))
		})
	})

	Context("SplitPrefixedName", func() {
		DescribeTable("should split valid strings", func(prefixedName, expectedpPrefix, expectedName string) {
			prefix, name, err := params.SplitPrefixedName(prefixedName)
			Expect(err).ToNot(HaveOccurred())
			Expect(prefix).To(Equal(expectedpPrefix))
			Expect(name).To(Equal(expectedName))
		},
			Entry("with prefix", "testname", "", "testname"),
			Entry("without prefix", "testns/testname", "testns", "testname"),
		)

		DescribeTable("should fail on invalid strings", func(prefixedName, errMsg string) {
			prefix, name, err := params.SplitPrefixedName(prefixedName)
			Expect(err).To(MatchError(errMsg))
			Expect(prefix).To(BeEmpty())
			Expect(name).To(BeEmpty())
		},
			Entry("two slashes", "testns/testname/test", "invalid count 2 of slashes in prefix/name"),
			Entry("three slashes", "testns/testname/test/test2", "invalid count 3 of slashes in prefix/name"),
			Entry("empty string", "", "name cannot be empty"),
			Entry("empty name after slash", "testns/", "name cannot be empty"),
		)
	})

	Context("GetParamByName", func() {
		DescribeTable("should fail on invalid params", func(paramsStr, errMsg string) {
			value, err := params.GetParamByName(testParam, paramsStr)
			Expect(err).To(MatchError(errMsg))
			Expect(value).To(BeEmpty())
		},
			Entry("empty params", "", "params may not be empty"),
			Entry("param without colon", "nocolon", "params need to have at least one colon: nocolon"),
		)

		DescribeTable("should get param value from params", func(paramsStrBase string) {
			expectedValue := rand.String(10)
			value, err := params.GetParamByName(testParam, paramsStrBase+expectedValue)
			Expect(err).To(Not(HaveOccurred()))
			Expect(value).To(Equal(expectedValue))
		},
			Entry("with single param", "testParam:"),
			Entry("with multiple params", "anotherParam:test,testParam:"),
		)

		DescribeTable("should fail if param is not found", func(paramsStr string) {
			value, err := params.GetParamByName(testParam, paramsStr)
			Expect(err).To(MatchError("testParam must be specified"))
			notFoundError := params.NotFoundError{Name: testParam}
			Expect(notFoundError.Is(err)).To(BeTrue())
			Expect(value).To(BeEmpty())
		},
			Entry("with single param", "anotherParam:test:"),
			Entry("with multiple params", "anotherParam:test,someParam:test"),
		)
	})
})
