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
 */

//go:build !excludenative

package ssh

import (
	"bufio"
	"crypto/rand"
	"crypto/rsa"
	"os"
	"path/filepath"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"golang.org/x/crypto/ssh"
)

var _ = Describe("Known Hosts", func() {

	var knowHostFile string
	BeforeEach(func() {
		tmpDir := GinkgoT().TempDir()
		knowHostFile = filepath.Join(tmpDir, "knownhosts")
		f, err := os.Create(knowHostFile)
		Expect(err).ToNot(HaveOccurred())
		_ = f.Close()
	})

	It("should be added with a newline", func() {
		publicKey, err := newPublicKey()
		Expect(err).ToNot(HaveOccurred())
		Expect(addHostKey(knowHostFile, "host1", publicKey)).To(Succeed())
		Expect(addHostKey(knowHostFile, "host2", publicKey)).To(Succeed())
		Expect(addHostKey(knowHostFile, "host3", publicKey)).To(Succeed())

		Expect(numberOfLines(knowHostFile)).To(Equal(3))
	})
})

func newPublicKey() (ssh.PublicKey, error) {
	privateKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		return nil, err
	}
	pub, err := ssh.NewPublicKey(&privateKey.PublicKey)
	if err != nil {
		return nil, err
	}
	return pub, nil
}

func numberOfLines(knowHostFile string) (int, error) {
	f, err := os.Open(knowHostFile)
	if err != nil {
		return -1, err
	}
	scanner := bufio.NewScanner(f)

	lineCount := 0
	for {
		if !scanner.Scan() {
			break
		}
		lineCount++
	}
	return lineCount, nil
}
