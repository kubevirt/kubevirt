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
 * Copyright 2021 Red Hat, Inc.
 *
 */

package apply

import (
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	v1 "kubevirt.io/api/core/v1"
)

var _ = Describe("Certificates", func() {
	oneDay := &metav1.Duration{Duration: 24 * time.Hour}
	twoDays := &metav1.Duration{Duration: 48 * time.Hour}
	threeDays := &metav1.Duration{Duration: 72 * time.Hour}
	fiveDays := &metav1.Duration{Duration: 120 * time.Hour}
	week := &metav1.Duration{Duration: 168 * time.Hour}

	var config *v1.KubeVirtSelfSignConfiguration
	BeforeEach(func() {
		config = &v1.KubeVirtSelfSignConfiguration{}
		config.CA = &v1.CertConfig{}
		config.Server = &v1.CertConfig{}
	})
	Context("With deprecated fields defined", func() {
		It("should use CARotateInterval if defined", func() {
			config.CA.Duration = fiveDays

			By("Using only CA.Duration")
			result := GetCADuration(config)
			Expect(result).To(Equal(fiveDays))

			By("Defining CARotateInterval")
			config.CARotateInterval = week

			result = GetCADuration(config)
			Expect(result).To(Equal(week))
		})

		It("should use CAOverlapInterval if defined", func() {
			config.CA.Duration = week
			config.CA.RenewBefore = threeDays

			result := GetCARenewBefore(config)
			Expect(result).To(Equal(threeDays))

			By("Defining CAOverlapInterval")
			config.CAOverlapInterval = twoDays
			result = GetCARenewBefore(config)
			Expect(result).To(Equal(twoDays))
		})

		It("should use CertRotateInterval if defined", func() {
			config.Server.Duration = twoDays

			By("Using only Server.Duration")
			result := GetCertDuration(config)
			Expect(result).To(Equal(twoDays))

			By("Defining CertRotateInterval")
			config.CertRotateInterval = threeDays

			result = GetCertDuration(config)
			Expect(result).To(Equal(threeDays))
		})
	})

	Context("defaults", func() {
		It("should return default CA RenewBefore", func() {
			result := GetCARenewBefore(config)
			// Default renewal period is 20% of default duration of one week
			reference := &metav1.Duration{Duration: time.Duration(168 * float64(time.Hour) * 0.2)}
			Expect(result).To(Equal(reference))

			By("Defining CA.RenewBefore")
			config.CA.RenewBefore = fiveDays

			result = GetCARenewBefore(config)
			Expect(result).To(Equal(fiveDays))
		})

		It("should return default Cert RenewBefore", func() {
			result := GetCertRenewBefore(config)
			// Default renew before is 80% of default duration of one day
			reference := &metav1.Duration{Duration: time.Duration(24 * float64(time.Hour) * 0.2)}
			Expect(result).To(Equal(reference))

			By("Defining Server.RenewBefore")
			config.Server.RenewBefore = oneDay

			result = GetCertRenewBefore(config)
			Expect(result).To(Equal(oneDay))
		})
	})
})
