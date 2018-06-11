/*
 * This file is part of the kubevirt project
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

package watchdog

import (
	"io/ioutil"
	"os"
	"time"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	v1 "kubevirt.io/kubevirt/pkg/api/v1"
	"kubevirt.io/kubevirt/pkg/precond"
)

var _ = Describe("Watchdog", func() {

	Context("When watching files in a directory", func() {

		var tmpVirtShareDir string
		var tmpWatchdogDir string
		var err error

		BeforeEach(func() {

			tmpVirtShareDir, err = ioutil.TempDir("", "kubevirt")
			Expect(err).ToNot(HaveOccurred())

			tmpWatchdogDir = WatchdogFileDirectory(tmpVirtShareDir)
			err = os.MkdirAll(tmpWatchdogDir, 0755)
			Expect(err).ToNot(HaveOccurred())
		})

		It("should detect expired watchdog files", func() {

			fileName := tmpWatchdogDir + "/default_expiredvmi"
			Expect(os.Create(fileName)).ToNot(BeNil())

			domains, err := GetExpiredDomains(1, tmpVirtShareDir)
			Expect(err).ToNot(HaveOccurred())
			Expect(len(domains)).To(Equal(0))

			time.Sleep(time.Second * 3)

			domains, err = GetExpiredDomains(1, tmpVirtShareDir)
			Expect(err).ToNot(HaveOccurred())
			Expect(len(domains)).To(Equal(1))

			Expect(os.Create(fileName)).ToNot(BeNil())
			domains, err = GetExpiredDomains(1, tmpVirtShareDir)
			Expect(err).ToNot(HaveOccurred())
			Expect(len(domains)).To(Equal(0))
		})

		It("should successfully remove watchdog file", func() {

			vmi := v1.NewMinimalVMI("tvmi")
			namespace := precond.MustNotBeEmpty(vmi.GetObjectMeta().GetNamespace())
			domain := precond.MustNotBeEmpty(vmi.GetObjectMeta().GetName())

			fileName := WatchdogFileFromNamespaceName(tmpVirtShareDir, namespace, domain)
			Expect(os.Create(fileName)).ToNot(BeNil())
			domains, err := GetExpiredDomains(1, tmpVirtShareDir)
			Expect(err).ToNot(HaveOccurred())
			Expect(len(domains)).To(Equal(0))

			expired, err := WatchdogFileIsExpired(1, tmpVirtShareDir, vmi)
			Expect(err).ToNot(HaveOccurred())
			Expect(expired).To(Equal(false))

			time.Sleep(time.Second * 3)

			domains, err = GetExpiredDomains(1, tmpVirtShareDir)
			Expect(err).ToNot(HaveOccurred())
			Expect(len(domains)).To(Equal(1))

			expired, err = WatchdogFileIsExpired(1, tmpVirtShareDir, vmi)
			Expect(err).ToNot(HaveOccurred())
			Expect(expired).To(Equal(true))

			exists, err := WatchdogFileExists(tmpVirtShareDir, vmi)
			Expect(err).ToNot(HaveOccurred())
			Expect(exists).To(Equal(true))

			err = WatchdogFileRemove(tmpVirtShareDir, vmi)
			Expect(err).ToNot(HaveOccurred())

			domains, err = GetExpiredDomains(1, tmpVirtShareDir)
			Expect(err).ToNot(HaveOccurred())
			Expect(len(domains)).To(Equal(0))

			exists, err = WatchdogFileExists(tmpVirtShareDir, vmi)
			Expect(err).ToNot(HaveOccurred())
			Expect(exists).To(Equal(false))
		})

		It("should not expire updated files", func() {
			fileName := tmpVirtShareDir + "/default_expiredvmi"
			Expect(os.Create(fileName)).ToNot(BeNil())

			for i := 0; i < 4; i++ {
				WatchdogFileUpdate(fileName)
				time.Sleep(time.Second * 1)
				domains, err := GetExpiredDomains(2, tmpVirtShareDir)
				Expect(err).ToNot(HaveOccurred())
				Expect(len(domains)).To(Equal(0))
			}
		})

		It("should provide file in watchdog subdirectory", func() {
			dir := WatchdogFileDirectory(tmpVirtShareDir)
			Expect(dir).To(Equal(tmpVirtShareDir + "/watchdog-files"))

			dir = WatchdogFileFromNamespaceName(tmpVirtShareDir, "tnamespace", "tvmi")
			Expect(dir).To(Equal(tmpVirtShareDir + "/watchdog-files/tnamespace_tvmi"))
		})

		AfterEach(func() {
			os.RemoveAll(tmpVirtShareDir)
		})

	})
})
