package util

import (
	"errors"
	"os"
	"strings"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("test file", func() {
	Context("Test UnmarshalYamlFileToObject", func() {
		type nestedObj struct {
			NestedStr string `json:"nestedStr,omitempty"`
			NestedInt int    `json:"nestedInt,omitempty"`
		}

		type testType struct {
			FieldStr string `json:"fieldStr,omitempty"`
			FieldInt int    `json:"fieldInt,omitempty"`
			FieldObj *nestedObj
		}

		It("should unmarshal valid input", func() {
			input := `
fieldStr: abcd
fieldInt: 123            
fieldObj:
  nestedStr: nested
  nestedInt: 456
            `
			reader := strings.NewReader(input)
			var obj testType
			err := UnmarshalYamlFileToObject(reader, &obj)
			Expect(err).ToNot(HaveOccurred())

			Expect(obj).Should(Equal(testType{FieldStr: "abcd", FieldInt: 123, FieldObj: &nestedObj{NestedStr: "nested", NestedInt: 456}}))
		})

		It("should unmarshal invalid input", func() {
			input := `
fieldStr: abcd
fieldInt: "123"            
fieldObj:
  nestedStr: nested
  nestedInt: 456
            `
			reader := strings.NewReader(input)
			var obj testType
			err := UnmarshalYamlFileToObject(reader, &obj)
			Expect(err).To(HaveOccurred())
		})

		It("should return error if failed to read", func() {
			reader := badReader{}
			var obj testType
			err := UnmarshalYamlFileToObject(reader, &obj)
			Expect(err).To(HaveOccurred())
		})
	})

	Context("test ValidateManifestDir", func() {

		It("should return nil wrapped with error, if the directory des not exist", func() {
			err := ValidateManifestDir("not-existing-dir")
			Expect(err).To(HaveOccurred())
			Expect(errors.Unwrap(err)).ToNot(HaveOccurred())
		})

		It("should return real error if trying to get the file stat (instead of a dir)", func() {
			tempDir := os.TempDir()
			defer os.RemoveAll(tempDir)

			fileName := tempDir + "/testFile.txt"
			_, err := os.Create(fileName)
			Expect(err).ToNot(HaveOccurred())

			err = ValidateManifestDir(fileName)
			Expect(err).To(HaveOccurred())
			err = errors.Unwrap(err)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(Equal(fileName + " is not a directory"))
		})

		It("should return no error for a valid dir name", func() {
			tempDir := os.TempDir()
			defer os.RemoveAll(tempDir)

			err := ValidateManifestDir(tempDir)
			Expect(err).ToNot(HaveOccurred())
		})
	})

	Context("test GetManifestDirPath", func() {
		It("should return default if the environment variable is not set", func() {
			result := GetManifestDirPath("TEST_VAR_NAME", "defaultValue")
			Expect(result).Should(Equal("defaultValue"))
		})

		It("should return value of the environment variable is it set", func() {
			os.Setenv("TEST_VAR_NAME", "non-default-value")
			defer os.Unsetenv("TEST_VAR_NAME")
			result := GetManifestDirPath("TEST_VAR_NAME", "defaultValue")
			Expect(result).Should(Equal("non-default-value"))
		})
	})
})

type badReader struct{}

func (badReader) Read(_ []byte) (int, error) {
	return 0, errors.New("fake error")
}
