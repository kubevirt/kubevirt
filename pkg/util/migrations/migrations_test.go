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

package migrations_test

import (
	"reflect"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	v1 "kubevirt.io/api/core/v1"

	"kubevirt.io/kubevirt/pkg/testutils"
	"kubevirt.io/kubevirt/pkg/util/migrations"
)

var _ = Describe("ToVMIMConfigurationOptions", func() {
	// Guards against silently dropping cluster defaults due to a missing field
	// assignment in ToVMIMConfigurationOptions. Fields are discovered via reflection
	// so the test catches new fields automatically. Assumes matching fields share
	// the same JSON tag name; update this test to explicitly acknowledge any
	// intentional divergence.
	It("covers all MigrationConfiguration fields", func() {
		src := testutils.WithAllFieldsSet(reflect.TypeOf(v1.MigrationConfiguration{})).(*v1.MigrationConfiguration)
		oracle := testutils.CopyByJSONTag(src, reflect.TypeOf(v1.VMIMConfigurationOptions{})).(*v1.VMIMConfigurationOptions)

		got := migrations.ToVMIMConfigurationOptions(src)

		Expect(got).To(Equal(oracle))
	})

	It("panics when config is nil", func() {
		Expect(func() {
			migrations.ToVMIMConfigurationOptions(nil)
		}).To(Panic())
	})
})
