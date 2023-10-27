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
 * Copyright the KubeVirt Authors.
 *
 */

package watchdog

import (
	"os"
	"path/filepath"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	v1 "kubevirt.io/client-go/api"

	"k8s.io/apimachinery/pkg/types"

	"kubevirt.io/client-go/precond"
)

var _ = Describe("Watchdog", func() {

	Context("When watching files in a directory", func() {

		var tmpVirtShareDir string
		var tmpWatchdogDir string
		var err error

		BeforeEach(func() {

			tmpVirtShareDir, err = os.MkdirTemp("", "kubevirt")
			Expect(err).ToNot(HaveOccurred())

			tmpWatchdogDir = WatchdogFileDirectory(tmpVirtShareDir)
			err = os.MkdirAll(tmpWatchdogDir, 0755)
			Expect(err).ToNot(HaveOccurred())
		})

		It("should detect expired watchdog files", func() {

			fileName := filepath.Join(tmpWatchdogDir, "default_expiredvmi")
			Expect(os.Create(fileName)).ToNot(BeNil())

			now := time.Now()
			domains, err := getExpiredDomains(1, tmpVirtShareDir, now)
			Expect(err).ToNot(HaveOccurred())
			Expect(domains).To(BeEmpty())

			now = now.Add(time.Second * 3)
			domains, err = getExpiredDomains(1, tmpVirtShareDir, now)
			Expect(err).ToNot(HaveOccurred())
			Expect(domains).To(HaveLen(1))

			Expect(os.Create(fileName)).ToNot(BeNil())
			now = time.Now()
			domains, err = getExpiredDomains(1, tmpVirtShareDir, now)
			Expect(err).ToNot(HaveOccurred())
			Expect(domains).To(BeEmpty())
		})

		It("should successfully remove watchdog file", func() {

			vmi := v1.NewMinimalVMI("tvmi")
			namespace := precond.MustNotBeEmpty(vmi.GetObjectMeta().GetNamespace())
			domain := precond.MustNotBeEmpty(vmi.GetObjectMeta().GetName())

			now := time.Now()
			fileName := WatchdogFileFromNamespaceName(tmpVirtShareDir, namespace, domain)
			Expect(os.Create(fileName)).ToNot(BeNil())
			domains, err := getExpiredDomains(1, tmpVirtShareDir, now)
			Expect(err).ToNot(HaveOccurred())
			Expect(domains).To(BeEmpty())

			expired, err := watchdogFileIsExpired(1, tmpVirtShareDir, vmi, now)
			Expect(err).ToNot(HaveOccurred())
			Expect(expired).To(BeFalse())

			now = now.Add(time.Second * 3)
			domains, err = getExpiredDomains(1, tmpVirtShareDir, now)
			Expect(err).ToNot(HaveOccurred())
			Expect(domains).To(HaveLen(1))

			expired, err = watchdogFileIsExpired(1, tmpVirtShareDir, vmi, now)
			Expect(err).ToNot(HaveOccurred())
			Expect(expired).To(BeTrue())

			exists, err := WatchdogFileExists(tmpVirtShareDir, vmi)
			Expect(err).ToNot(HaveOccurred())
			Expect(exists).To(BeTrue())

			err = WatchdogFileRemove(tmpVirtShareDir, vmi)
			Expect(err).ToNot(HaveOccurred())

			domains, err = getExpiredDomains(1, tmpVirtShareDir, now)
			Expect(err).ToNot(HaveOccurred())
			Expect(domains).To(BeEmpty())

			exists, err = WatchdogFileExists(tmpVirtShareDir, vmi)
			Expect(err).ToNot(HaveOccurred())
			Expect(exists).To(BeFalse())
		})

		It("should not expire updated files", func() {
			fileName := filepath.Join(tmpVirtShareDir, "default_expiredvmi")
			Expect(os.Create(fileName)).ToNot(BeNil())
			now := time.Now()

			for i := 0; i < 4; i++ {
				Expect(WatchdogFileUpdate(fileName, "somestring")).To(Succeed())
				now = now.Add(time.Second * 1)
				domains, err := getExpiredDomains(2, tmpVirtShareDir, now)
				Expect(err).ToNot(HaveOccurred())
				Expect(domains).To(BeEmpty())
			}
		})

		It("should be able to get uid from watchdog", func() {
			vmi := v1.NewMinimalVMI("testvmi")
			vmi.UID = types.UID("1234")

			fileName := filepath.Join(tmpVirtShareDir, "watchdog-files", vmi.Namespace+"_"+vmi.Name)
			Expect(WatchdogFileUpdate(fileName, string(vmi.UID))).To(Succeed())

			uid := WatchdogFileGetUID(tmpVirtShareDir, vmi)
			Expect(uid).To(Equal(string(vmi.UID)))
		})

		It("should provide file in watchdog subdirectory", func() {
			dir := WatchdogFileDirectory(tmpVirtShareDir)
			Expect(dir).To(Equal(filepath.Join(tmpVirtShareDir, "watchdog-files")))

			dir = WatchdogFileFromNamespaceName(tmpVirtShareDir, "tnamespace", "tvmi")
			Expect(dir).To(Equal(filepath.Join(tmpVirtShareDir, "watchdog-files/tnamespace_tvmi")))
		})

		AfterEach(func() {
			Expect(os.RemoveAll(tmpVirtShareDir)).To(Succeed())
		})

	})
})
