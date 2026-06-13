package rest

import (
	"errors"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("RefCounter", func() {
	const testValue = "test-value"

	var rc RefCounter[int, string]

	BeforeEach(func() {
		rc = NewRefCounter[int, string]()
	})

	It("should create object on first Get", func() {
		createCount := 0
		createFn := func() (string, func(), error) {
			createCount++
			return testValue, nil, nil
		}

		obj, release, err := rc.Get(1, createFn)
		Expect(err).ToNot(HaveOccurred())
		defer release()

		Expect(obj).To(Equal(testValue))
		Expect(createCount).To(Equal(1))
	})

	It("should reuse object on second Get with same key", func() {
		createCount := 0
		createFn := func() (string, func(), error) {
			createCount++
			return testValue, nil, nil
		}

		obj1, release1, err := rc.Get(1, createFn)
		Expect(err).ToNot(HaveOccurred())
		defer release1()
		Expect(obj1).To(Equal(testValue))

		obj2, release2, err := rc.Get(1, createFn)
		Expect(err).ToNot(HaveOccurred())
		defer release2()
		Expect(obj2).To(Equal(testValue))

		Expect(createCount).To(Equal(1))
	})

	It("should call destroyFn when last reference is released", func() {
		destroyCount := 0
		createFn := func() (string, func(), error) {
			return testValue, func() { destroyCount++ }, nil
		}

		_, release1, _ := rc.Get(1, createFn)
		_, release2, _ := rc.Get(1, createFn)

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

		Expect(func() { release() }).ToNot(Panic())
	})

	It("should recreate object after all references are released", func() {
		createCount := 0
		createFn := func() (string, func(), error) {
			createCount++
			return testValue, nil, nil
		}

		_, release1, err := rc.Get(1, createFn)
		Expect(err).ToNot(HaveOccurred())
		release1()

		obj, release2, err := rc.Get(1, createFn)
		Expect(err).ToNot(HaveOccurred())
		Expect(obj).To(Equal(testValue))
		Expect(createCount).To(Equal(2))
		release2()
	})

	It("should isolate different keys", func() {
		const object1 = "object-1"
		obj1, release1, _ := rc.Get(1, func() (string, func(), error) {
			return object1, nil, nil
		})

		const object2 = "object-2"
		obj2, release2, _ := rc.Get(2, func() (string, func(), error) {
			return object2, nil, nil
		})

		Expect(obj1).To(Equal(object1))
		Expect(obj2).To(Equal(object2))

		release1()
		release2()
	})

	It("should independently track references for different keys", func() {
		destroy1, destroy2 := 0, 0

		_, release1, _ := rc.Get(1, func() (string, func(), error) {
			return "obj-1", func() { destroy1++ }, nil
		})

		_, release2, _ := rc.Get(2, func() (string, func(), error) {
			return "obj-2", func() { destroy2++ }, nil
		})

		release1()
		Expect(destroy1).To(Equal(1))
		Expect(destroy2).To(Equal(0))

		release2()
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
		obj, release, err := rc.Get(1, successFn)
		Expect(err).ToNot(HaveOccurred())
		Expect(obj).To(Equal(value))
		Expect(createCount).To(Equal(1))
		defer release()
	})

	It("should panic when releasing non-existent reference", func() {
		createFn := func() (string, func(), error) {
			return testValue, nil, nil
		}

		_, release, _ := rc.Get(1, createFn)
		release()

		Expect(func() { release() }).To(PanicWith("Tried to release non-existing object"))
	})
})
