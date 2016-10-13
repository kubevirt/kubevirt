package util

import (
	"fmt"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("Http", func() {
	Describe("String supplied to ExtractEmbeddedmap", func() {
		Context("with valid multiple key/value pairs", func() {
			It("should be extracted into a map", func() {
				val := "key1= val1;key2 =val2 "
				mymap, err := ExtractEmbeddedMap(val)
				Expect(err).To(BeNil())
				Expect(mymap).Should(HaveKeyWithValue("key1", "val1"))
				Expect(mymap).Should(HaveKeyWithValue("key2", "val2"))
			})
		})
		Context("with a single key/value pair", func() {
			It("should be extracted into a map", func() {
				val := "key1= val1 "
				mymap, err := ExtractEmbeddedMap(val)
				Expect(err).To(BeNil())
				Expect(mymap).Should(HaveKeyWithValue("key1", "val1"))
			})
		})
		Context("with empty value", func() {
			It("should produce an error ", func() {
				val := "key1="
				mymap, err := ExtractEmbeddedMap(val)
				Expect(err).To(Not(BeNil()))
				Expect(mymap).To(BeNil())
			})
		})
		Context("with key only", func() {
			It("should produce an error", func() {
				val := "key1"
				mymap, err := ExtractEmbeddedMap(val)
				Expect(err).To(Not(BeNil()))
				Expect(mymap).To(BeNil())
			})
		})
		Context("which is empty", func() {
			It("should return an empty read only map", func() {
				val := ""
				mymap, err := ExtractEmbeddedMap(val)
				Expect(err).To(BeNil())
				Expect(mymap).To(BeEmpty())
			})
		})
	})
})

func ExampleExtractEmbeddedMap() {
	val := "key1=val1;key2=val2"
	mymap, _ := ExtractEmbeddedMap(val)
	fmt.Print(mymap["key1"])
	fmt.Print(" ")
	fmt.Print(mymap["key2"])
	// Output: val1 val2
}
