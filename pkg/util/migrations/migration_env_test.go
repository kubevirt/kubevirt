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
	"os"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"k8s.io/apimachinery/pkg/api/resource"

	"kubevirt.io/kubevirt/pkg/util/migrations"
)

var _ = Describe("ShouldDisableMultifd", func() {
	AfterEach(func() {
		Expect(os.Unsetenv(migrations.EnvDisableMultifd)).To(Succeed())
	})

	DescribeTable("should report whether multifd should be disabled", func(value string, setEnv, expected bool) {
		if setEnv {
			Expect(os.Setenv(migrations.EnvDisableMultifd, value)).To(Succeed())
		}
		Expect(migrations.ShouldDisableMultifd()).To(Equal(expected))
	},
		Entry("absent", "", false, false),
		Entry("true", "true", true, true),
		Entry("false", "false", true, false),
		Entry("invalid", "not-bool", true, false),
	)
})

var _ = Describe("ApplyEnvOverrides", func() {
	AfterEach(func() {
		Expect(os.Unsetenv(migrations.EnvStallMargin)).To(Succeed())
		Expect(os.Unsetenv(migrations.EnvStallProgressTimeout)).To(Succeed())
	})

	It("should overlay explicitly set stall detector env vars on top of base options", func() {
		Expect(os.Setenv(migrations.EnvStallMargin, "8")).To(Succeed())
		Expect(os.Setenv(migrations.EnvStallProgressTimeout, "10")).To(Succeed())

		base := migrations.StallDetectorOptions{
			StallMargin:             4,
			StallProgressTimeout:    25,
			SwitchoverTimeout:       42,
			EwmaAlpha:               resource.MustParse("0.25"),
			PrecopyPossibleFactor:   resource.MustParse("2.0"),
			SearchLocalMinima:       false,
			CompletionTimeoutFactor: resource.MustParse("2.0"),
		}

		Expect(migrations.ApplyEnvOverrides(base)).To(Equal(migrations.StallDetectorOptions{
			StallMargin:             8,
			StallProgressTimeout:    10,
			SwitchoverTimeout:       42,
			EwmaAlpha:               resource.MustParse("0.25"),
			PrecopyPossibleFactor:   resource.MustParse("2.0"),
			SearchLocalMinima:       false,
			CompletionTimeoutFactor: resource.MustParse("2.0"),
		}))
	})
})
