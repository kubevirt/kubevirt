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

package leaderelectionconfig

import (
	"testing"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/spf13/pflag"
	"k8s.io/client-go/tools/leaderelection/resourcelock"
)

func TestLeaderElectionConfig(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "LeaderElectionConfig Suite")
}

var _ = Describe("LeaderElectionConfiguration", func() {
	var originalFlags *pflag.FlagSet

	BeforeEach(func() {
		// Save the original flags to restore them later (avoids side effects)
		originalFlags = pflag.CommandLine
	})

	AfterEach(func() {
		// Restore the original flags
		pflag.CommandLine = originalFlags
	})

	Context("Defaults", func() {
		It("should return correct default values", func() {
			config := DefaultLeaderElectionConfiguration()

			Expect(config.LeaseDuration.Duration).To(Equal(15 * time.Second))
			Expect(config.RenewDeadline.Duration).To(Equal(10 * time.Second))
			Expect(config.RetryPeriod.Duration).To(Equal(2 * time.Second))
			Expect(config.ResourceLock).To(Equal(resourcelock.LeasesResourceLock))
		})
	})

	Context("Flags", func() {
		It("should bind flags and parse values correctly", func() {
			config := DefaultLeaderElectionConfiguration()
			
			// Use a clean FlagSet for testing
			pflag.CommandLine = pflag.NewFlagSet("test", pflag.ContinueOnError)
			
			BindFlags(&config)

			// Simulate passing custom arguments
			err := pflag.CommandLine.Parse([]string{
				"--leader-elect-lease-duration=30s",
				"--leader-elect-renew-deadline=20s",
				"--leader-elect-retry-period=5s",
				"--leader-elect-resource-lock=configmap",
			})
			Expect(err).ToNot(HaveOccurred())

			// Assert that the configuration struct was actually updated
			Expect(config.LeaseDuration.Duration).To(Equal(30 * time.Second))
			Expect(config.RenewDeadline.Duration).To(Equal(20 * time.Second))
			Expect(config.RetryPeriod.Duration).To(Equal(5 * time.Second))
			Expect(config.ResourceLock).To(Equal("configmap"))
		})
	})
})