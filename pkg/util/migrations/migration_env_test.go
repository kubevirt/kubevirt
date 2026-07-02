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

	"kubevirt.io/kubevirt/pkg/pointer"
	"kubevirt.io/kubevirt/pkg/util/migrations"
)

var _ = Describe("LookupOptionsOverrides", func() {
	AfterEach(func() {
		Expect(os.Unsetenv(migrations.EnvParallelMigrationThreads)).To(Succeed())
		Expect(os.Unsetenv(migrations.EnvBandwidthPerMigration)).To(Succeed())
	})

	It("should leave parallel migration threads unset when env is absent", func() {
		overrides := migrations.LookupOptionsOverrides()
		Expect(overrides.ParallelMigrationThreadsConfigured).To(BeFalse())
	})

	It("should disable parallel migration when env is zero", func() {
		Expect(os.Setenv(migrations.EnvParallelMigrationThreads, "0")).To(Succeed())
		overrides := migrations.LookupOptionsOverrides()
		Expect(overrides.ParallelMigrationThreadsConfigured).To(BeTrue())
		Expect(overrides.ParallelMigrationThreads).To(BeNil())
	})

	It("should override when env is set to a positive value", func() {
		Expect(os.Setenv(migrations.EnvParallelMigrationThreads, "4")).To(Succeed())
		overrides := migrations.LookupOptionsOverrides()
		Expect(overrides.ParallelMigrationThreadsConfigured).To(BeTrue())
		Expect(overrides.ParallelMigrationThreads).To(Equal(pointer.P(uint(4))))
	})

	DescribeTable("should accept valid Kubernetes quantity strings for bandwidth",
		func(value string, expected string) {
			Expect(os.Setenv(migrations.EnvBandwidthPerMigration, value)).To(Succeed())
			overrides := migrations.LookupOptionsOverrides()
			Expect(overrides.Bandwidth).ToNot(BeNil())
			Expect(overrides.Bandwidth.String()).To(Equal(expected))
		},
		Entry("binary SI suffix", "15Mi", "15Mi"),
		Entry("decimal SI suffix", "10M", "10M"),
		Entry("milli suffix", "300m", "300m"),
		Entry("decimal exponent", "12e2", "1200"),
	)

	It("should ignore invalid bandwidth env values", func() {
		Expect(os.Setenv(migrations.EnvBandwidthPerMigration, "not-a-quantity")).To(Succeed())
		overrides := migrations.LookupOptionsOverrides()
		Expect(overrides.Bandwidth).To(BeNil())
	})

	It("should ignore negative bandwidth env values", func() {
		Expect(os.Setenv(migrations.EnvBandwidthPerMigration, "-15Mi")).To(Succeed())
		overrides := migrations.LookupOptionsOverrides()
		Expect(overrides.Bandwidth).To(BeNil())
	})
})

var _ = Describe("ApplyEnvOverrides", func() {
	AfterEach(func() {
		Expect(os.Unsetenv(migrations.EnvStallMargin)).To(Succeed())
		Expect(os.Unsetenv(migrations.EnvStallProgressTimeout)).To(Succeed())
	})

	It("should overlay explicitly set stall detector env vars on top of base options", func() {
		Expect(os.Setenv(migrations.EnvStallMargin, "0.08")).To(Succeed())
		Expect(os.Setenv(migrations.EnvStallProgressTimeout, "10")).To(Succeed())

		base := migrations.StallDetectorOptions{
			StallMargin:             0.04,
			StallProgressTimeout:    25,
			SwitchoverTimeout:       42,
			EwmaAlpha:               0.25,
			PrecopyPossibleFactor:   2.0,
			SearchLocalMinima:       false,
			CompletionTimeoutFactor: 2,
		}

		Expect(migrations.ApplyEnvOverrides(base)).To(Equal(migrations.StallDetectorOptions{
			StallMargin:             0.08,
			StallProgressTimeout:    10,
			SwitchoverTimeout:       42,
			EwmaAlpha:               0.25,
			PrecopyPossibleFactor:   2.0,
			SearchLocalMinima:       false,
			CompletionTimeoutFactor: 2,
		}))
	})
})
