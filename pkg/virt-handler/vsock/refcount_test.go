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

package vsock_test

import (
	"errors"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"kubevirt.io/kubevirt/pkg/virt-handler/vsock"
)

var _ = Describe("RefCounter", func() {
	const testValue = "test-value"

	var rc *vsock.RefCounter[int, string]

	BeforeEach(func() {
		rc = vsock.NewRefCounter[int, string]()
	})

	It("should create object on first Get", func() {
		createCount := 0
		createFn := func() (string, func(), error) {
			createCount++
			return testValue, nil, nil
		}

		obj, _, err := rc.Get(1, createFn)
		Expect(err).ToNot(HaveOccurred())
		Expect(obj).To(Equal(testValue))
		Expect(createCount).To(Equal(1))
	})

	It("should reuse object on second Get with same key", func() {
		createCount := 0
		createFn := func() (string, func(), error) {
			createCount++
			return testValue, nil, nil
		}

		obj1, _, err := rc.Get(1, createFn)
		Expect(err).ToNot(HaveOccurred())
		Expect(obj1).To(Equal(testValue))
		Expect(createCount).To(Equal(1))

		obj2, _, err := rc.Get(1, createFn)
		Expect(err).ToNot(HaveOccurred())
		Expect(obj2).To(Equal(testValue))
		Expect(createCount).To(Equal(1))
	})

	It("should call destroyFn when last reference is released", func() {
		destroyCount := 0
		createFn := func() (string, func(), error) {
			return testValue, func() { destroyCount++ }, nil
		}

		_, release1, err := rc.Get(1, createFn)
		Expect(err).ToNot(HaveOccurred())

		_, release2, err := rc.Get(1, createFn)
		Expect(err).ToNot(HaveOccurred())

		release1()
		Expect(destroyCount).To(Equal(0))

		release2()
		Expect(destroyCount).To(Equal(1))
	})

	It("should handle nil destroyFn", func() {
		createFn := func() (string, func(), error) {
			return testValue, nil, nil
		}

		_, release, err := rc.Get(1, createFn)
		Expect(err).ToNot(HaveOccurred())
		Expect(release).ToNot(Panic())
	})

	It("should recreate object after all references are released", func() {
		createCount := 0
		createFn := func() (string, func(), error) {
			createCount++
			return testValue, nil, nil
		}

		_, release1, err := rc.Get(1, createFn)
		Expect(err).ToNot(HaveOccurred())
		Expect(createCount).To(Equal(1))
		release1()

		_, release2, err := rc.Get(1, createFn)
		Expect(err).ToNot(HaveOccurred())
		Expect(createCount).To(Equal(2))
		release2()
	})

	It("should isolate different keys", func() {
		const object1 = "object-1"
		obj1, _, err := rc.Get(1, func() (string, func(), error) {
			return object1, nil, nil
		})
		Expect(err).ToNot(HaveOccurred())

		const object2 = "object-2"
		obj2, _, err := rc.Get(2, func() (string, func(), error) {
			return object2, nil, nil
		})
		Expect(err).ToNot(HaveOccurred())

		Expect(obj1).To(Equal(object1))
		Expect(obj2).To(Equal(object2))
	})

	It("should independently track references for different keys", func() {
		destroy1, destroy2 := 0, 0

		_, release1, err := rc.Get(1, func() (string, func(), error) {
			return "obj-1", func() { destroy1++ }, nil
		})
		Expect(err).ToNot(HaveOccurred())

		_, release2, err := rc.Get(2, func() (string, func(), error) {
			return "obj-2", func() { destroy2++ }, nil
		})
		Expect(err).ToNot(HaveOccurred())

		release1()
		Expect(destroy1).To(Equal(1))
		Expect(destroy2).To(Equal(0))

		release2()
		Expect(destroy1).To(Equal(1))
		Expect(destroy2).To(Equal(1))
	})

	It("should return error when createFn fails", func() {
		createFn := func() (string, func(), error) {
			return "", nil, errors.New("creation failed")
		}

		obj, release, err := rc.Get(1, createFn)

		Expect(err).To(MatchError("creation failed"))
		Expect(obj).To(BeZero())
		Expect(release).To(BeNil())
	})

	It("should not store failed object in cache", func() {
		createCount := 0

		failFn := func() (string, func(), error) {
			return "", nil, errors.New("fail")
		}
		_, _, err := rc.Get(1, failFn)
		Expect(err).To(HaveOccurred())

		const value = "success"
		successFn := func() (string, func(), error) {
			createCount++
			return value, nil, nil
		}
		obj, _, err := rc.Get(1, successFn)
		Expect(err).ToNot(HaveOccurred())
		Expect(obj).To(Equal(value))
		Expect(createCount).To(Equal(1))
	})
})
