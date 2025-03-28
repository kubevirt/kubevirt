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
