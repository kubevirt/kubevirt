package cache

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("time defined cache", func() {

	getMockCalcFunc := func() func() (int, error) {
		mockValue := 0
		return func() (int, error) {
			mockValue++
			return mockValue, nil
		}
	}

	It("should get the same value if the refresh duration has not passed", func() {
		cache := NewTimeDefinedCache(123, false, getMockCalcFunc())
		value, err := cache.Get()
		Expect(err).ToNot(HaveOccurred())
		Expect(value).To(Equal(1))

		value, err = cache.Get()
		Expect(err).ToNot(HaveOccurred())
		Expect(value).To(Equal(1))
	})

	It("should get a new value if the refresh duration has passed", func() {
		cache := NewTimeDefinedCache(0, true, getMockCalcFunc())
		value, err := cache.Get()
		Expect(err).ToNot(HaveOccurred())
		Expect(value).To(Equal(1))

		value, err = cache.Get()
		Expect(err).ToNot(HaveOccurred())
		Expect(value).To(Equal(2))
	})

	It("should return an error if the re-calculation function is not set", func() {
		cache := &TimeDefinedCache[int]{}
		_, err := cache.Get()
		Expect(err).To(HaveOccurred())
	})

})
