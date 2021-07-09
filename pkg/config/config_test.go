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
 * Copyright 2018 Red Hat, Inc.
 *
 */
package config

import (
	"io/ioutil"
	"os"
	"path/filepath"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

func mockCreateISOImage(output string, _ string, _ []string) error {
	_, err := os.Create(output)
	if err != nil {
		panic(err)
	}
	return nil
}

var _ = BeforeSuite(func() {
	setIsoCreationFunction(mockCreateISOImage)
})

var _ = Describe("Creating config images", func() {

	Context("With creating file system layout", func() {
		var tempConfDir string
		var tempISODir string
		var expectedLayout []string

		BeforeEach(func() {
			var err error
			tempConfDir, err = ioutil.TempDir("", "config-dir")
			Expect(err).NotTo(HaveOccurred())
			tempISODir, err = ioutil.TempDir("", "iso-dir")
			Expect(err).NotTo(HaveOccurred())
			expectedLayout = []string{"test-dir=" + tempConfDir + "/test-dir", "test-file2=" + tempConfDir + "/test-file2"}

			os.Mkdir(filepath.Join(tempConfDir, "test-dir"), 0755)
			os.OpenFile(filepath.Join(tempConfDir, "test-dir", "test-file1"), os.O_RDONLY|os.O_CREATE, 0666)
			os.OpenFile(filepath.Join(tempConfDir, "test-file2"), os.O_RDONLY|os.O_CREATE, 0666)

		})

		AfterEach(func() {
			os.RemoveAll(tempConfDir)
			os.RemoveAll(tempISODir)
		})

		It("Should create an appropriate file system layout for iso image", func() {
			fsLayout, err := getFilesLayout(tempConfDir)
			Expect(err).NotTo(HaveOccurred())
			Expect(fsLayout).To(Equal(expectedLayout))
		})

		It("Should create an iso image", func() {
			imgPath := filepath.Join(tempISODir, "volume1.iso")
			err := createIsoConfigImage(imgPath, "", expectedLayout)
			Expect(err).NotTo(HaveOccurred())
			_, err = os.Stat(imgPath)
			Expect(err).NotTo(HaveOccurred())
		})
	})
})
