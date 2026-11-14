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

package device_manager

import (
	"os"
	"path/filepath"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/fsnotify/fsnotify"
)

var _ = Describe("deviceWatcher", func() {
	const quietPeriod = 100 * time.Millisecond

	It ("does not emit an event for a write to a watched file", func() {
		dir := GinkgoT().TempDir()
		file := filepath.Join(dir, "device")
		Expect(os.WriteFile(file, []byte("initial"), 0666)).To(Succeed())

		w, err := newDeviceWatcher()
		Expect(err).ToNot(HaveOccurred())
		defer func() { Expect(w.Close()).To(Succeed()) }()

		Expect(w.Add(dir)).To(Succeed())

		// Give the kernel a moment to register the watch before writing.
		time.Sleep(quietPeriod)

		Expect(os.WriteFile(file, []byte("a longer payload that changes size"), 0666)).To(Succeed())

		Consistently(w.Events, quietPeriod, 10*time.Millisecond).
			ShouldNot(Receive(), "a plain write must not wake the watcher (IN_MODIFY must not be subscribed)")
	})

	It ("emits a Create event when a file appears in a watched directory", func() {
		dir := GinkgoT().TempDir()

		w, err := newDeviceWatcher()
		Expect(err).ToNot(HaveOccurred())
		defer func() { Expect(w.Close()).To(Succeed()) }()

		Expect(w.Add(dir)).To(Succeed())
		time.Sleep(quietPeriod)

		file := filepath.Join(dir, "device")
		Expect(os.WriteFile(file, []byte("x"), 0666)).To(Succeed())

		var event fsnotify.Event
		Eventually(w.Events, quietPeriod, 10*time.Millisecond).Should(Receive(&event))
		Expect(event.Name).To(Equal(file))
		Expect(event.Op).To(Equal(fsnotify.Create))
	})

	It ("emits a Remove event when a watched file is deleted", func() {
		dir := GinkgoT().TempDir()
		file := filepath.Join(dir, "device")
		Expect(os.WriteFile(file, []byte("x"), 0666)).To(Succeed())

		w, err := newDeviceWatcher()
		Expect(err).ToNot(HaveOccurred())
		defer func() { Expect(w.Close()).To(Succeed()) }()

		Expect(w.Add(dir)).To(Succeed())
		time.Sleep(quietPeriod)

		Expect(os.Remove(file)).To(Succeed())

		var event fsnotify.Event
		Eventually(w.Events, quietPeriod, 10*time.Millisecond).Should(Receive(&event))
		Expect(event.Name).To(Equal(file))
		Expect(event.Op & fsnotify.Remove).ToNot(BeZero())
	})
})
