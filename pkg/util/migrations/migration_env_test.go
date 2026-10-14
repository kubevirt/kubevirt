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

var stallDetectorEnvVars = []string{
	migrations.EnvStallMargin,
	migrations.EnvStallProgressTimeout,
	migrations.EnvSwitchoverTimeout,
	migrations.EnvEwmaAlpha,
	migrations.EnvPrecopyPossibleFactor,
	migrations.EnvPatienceWindowDecayFactor,
	migrations.EnvSearchLocalMinima,
	migrations.EnvCompletionTimeoutFactor,
}

func baseStallDetectorOptions() migrations.StallDetectorOptions {
	return migrations.StallDetectorOptions{
		StallMargin:               4,
		StallProgressTimeout:      25,
		SwitchoverTimeout:         42,
		EwmaAlpha:                 resource.MustParse("0.25"),
		PrecopyPossibleFactor:     resource.MustParse("2.0"),
		PatienceWindowDecayFactor: resource.MustParse("0.75"),
		SearchLocalMinima:         false,
		CompletionTimeoutFactor:   resource.MustParse("2.0"),
	}
}

var _ = Describe("ApplyEnvOverrides", func() {
	AfterEach(func() {
		for _, key := range stallDetectorEnvVars {
			Expect(os.Unsetenv(key)).To(Succeed())
		}
	})

	It("should overlay every explicitly set stall detector env var on top of base options", func() {
		Expect(os.Setenv(migrations.EnvStallMargin, "8")).To(Succeed())
		Expect(os.Setenv(migrations.EnvStallProgressTimeout, "10")).To(Succeed())
		Expect(os.Setenv(migrations.EnvSwitchoverTimeout, "15")).To(Succeed())
		Expect(os.Setenv(migrations.EnvEwmaAlpha, "0.4")).To(Succeed())
		Expect(os.Setenv(migrations.EnvPrecopyPossibleFactor, "1.5")).To(Succeed())
		Expect(os.Setenv(migrations.EnvPatienceWindowDecayFactor, "0.5")).To(Succeed())
		Expect(os.Setenv(migrations.EnvSearchLocalMinima, "true")).To(Succeed())
		Expect(os.Setenv(migrations.EnvCompletionTimeoutFactor, "3")).To(Succeed())

		Expect(migrations.ApplyEnvOverrides(baseStallDetectorOptions())).To(Equal(migrations.StallDetectorOptions{
			StallMargin:               8,
			StallProgressTimeout:      10,
			SwitchoverTimeout:         15,
			EwmaAlpha:                 resource.MustParse("0.4"),
			PrecopyPossibleFactor:     resource.MustParse("1.5"),
			PatienceWindowDecayFactor: resource.MustParse("0.5"),
			SearchLocalMinima:         true,
			CompletionTimeoutFactor:   resource.MustParse("3"),
		}))
	})

	It("should leave unset fields at their base values", func() {
		Expect(os.Setenv(migrations.EnvStallMargin, "8")).To(Succeed())
		Expect(os.Setenv(migrations.EnvStallProgressTimeout, "10")).To(Succeed())

		base := baseStallDetectorOptions()
		Expect(migrations.ApplyEnvOverrides(base)).To(Equal(migrations.StallDetectorOptions{
			StallMargin:               8,
			StallProgressTimeout:      10,
			SwitchoverTimeout:         base.SwitchoverTimeout,
			EwmaAlpha:                 base.EwmaAlpha,
			PrecopyPossibleFactor:     base.PrecopyPossibleFactor,
			PatienceWindowDecayFactor: base.PatienceWindowDecayFactor,
			SearchLocalMinima:         base.SearchLocalMinima,
			CompletionTimeoutFactor:   base.CompletionTimeoutFactor,
		}))
	})

	It("should keep base options when an override does not parse", func() {
		Expect(os.Setenv(migrations.EnvStallMargin, "0.07")).To(Succeed())
		Expect(os.Setenv(migrations.EnvEwmaAlpha, "not-a-quantity")).To(Succeed())
		Expect(os.Setenv(migrations.EnvSearchLocalMinima, "not-bool")).To(Succeed())

		base := baseStallDetectorOptions()
		Expect(migrations.ApplyEnvOverrides(base)).To(Equal(base))
	})
})
