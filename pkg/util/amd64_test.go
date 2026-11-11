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

package util

import (
	"reflect"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	v1 "kubevirt.io/api/core/v1"
)

var _ = Describe("LaunchSecurityTypes", func() {
	It("should report no types for a nil launch security", func() {
		Expect(LaunchSecurityTypes(nil)).To(BeEmpty())
	})

	It("should report no types when none are set", func() {
		Expect(LaunchSecurityTypes(&v1.LaunchSecurity{})).To(BeEmpty())
	})

	DescribeTable("should report the selected type", func(launchSec *v1.LaunchSecurity, expected []string) {
		Expect(LaunchSecurityTypes(launchSec)).To(Equal(expected))
	},
		Entry("SEV", &v1.LaunchSecurity{SEV: &v1.SEV{}}, []string{"SEV"}),
		Entry("SNP", &v1.LaunchSecurity{SNP: &v1.SEVSNP{}}, []string{"SNP"}),
		Entry("TDX", &v1.LaunchSecurity{TDX: &v1.TDX{}}, []string{"TDX"}),
		Entry("SEV and TDX",
			&v1.LaunchSecurity{SEV: &v1.SEV{}, TDX: &v1.TDX{}}, []string{"SEV", "TDX"}),
		Entry("all of them",
			&v1.LaunchSecurity{SEV: &v1.SEV{}, SNP: &v1.SEVSNP{}, TDX: &v1.TDX{}},
			[]string{"SEV", "SNP", "TDX"}),
	)

	// A type missing from the list would silently stop being mutually exclusive
	It("should report every type defined in the API", func() {
		launchSecurityType := reflect.TypeOf(v1.LaunchSecurity{})

		launchSec := &v1.LaunchSecurity{}
		value := reflect.ValueOf(launchSec).Elem()
		var fieldNames []string
		for i := range launchSecurityType.NumField() {
			field := launchSecurityType.Field(i)

			Expect(field.Type.Kind()).To(Equal(reflect.Pointer),
				"v1.LaunchSecurity.%s must be a pointer: LaunchSecurityTypes reports a "+
					"type per non-nil field", field.Name)

			value.Field(i).Set(reflect.New(field.Type.Elem()))
			fieldNames = append(fieldNames, field.Name)
		}

		Expect(LaunchSecurityTypes(launchSec)).To(ConsistOf(fieldNames),
			"LaunchSecurityTypes does not cover every v1.LaunchSecurity field; add the "+
				"missing type so admission and domain conversion keep rejecting conflicts")
	})
})
