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

package migration

import (
	"reflect"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	v1 "kubevirt.io/api/core/v1"
	migrationsv1 "kubevirt.io/api/migrations/v1alpha1"

	"kubevirt.io/kubevirt/pkg/testutils"
)

var _ = Describe("applyMigrationPolicySpec", func() {
	// Guards against silently dropping policy fields due to a missing setIfNotNil
	// call in applyMigrationPolicySpec. Fields are discovered via reflection so the
	// test catches new fields automatically. Assumes matching fields share the same
	// JSON tag name; update this test to explicitly acknowledge any divergence.
	It("maps every MigrationPolicySpec field without mutating base", func() {
		src := testutils.WithAllFieldsSet(reflect.TypeOf(migrationsv1.MigrationPolicySpec{})).(*migrationsv1.MigrationPolicySpec)
		oracle := testutils.CopyByJSONTag(src, reflect.TypeOf(v1.VMIMConfigurationOptions{})).(*v1.VMIMConfigurationOptions)

		base := &v1.VMIMConfigurationOptions{}
		baseBefore := *base
		got := applyMigrationPolicySpec(base, src)

		Expect(got).To(Equal(oracle))
		Expect(base).To(Equal(&baseBefore))
	})

	DescribeTable("backward compatibility shim for AllowWorkloadDisruption", func(allowPostCopy bool, wantDisrupt bool) {
		spec := &migrationsv1.MigrationPolicySpec{
			AllowPostCopy: new(allowPostCopy),
		}
		got := applyMigrationPolicySpec(&v1.VMIMConfigurationOptions{}, spec)

		Expect(got.AllowWorkloadDisruption).ToNot(BeNil())
		Expect(*got.AllowWorkloadDisruption).To(Equal(wantDisrupt))
	},
		Entry("AllowPostCopy true implies AllowWorkloadDisruption true", true, true),
		Entry("AllowPostCopy false implies AllowWorkloadDisruption false", false, false),
	)
})
