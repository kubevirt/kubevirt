package main

import (
	"errors"
	"io"
	"os"
	"path/filepath"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("Conformance", func() {
	var (
		tmpDir              string
		originalResultsDir  string
		originalExecuteFunc func() error
		originalOutput      io.Writer
	)

	BeforeEach(func() {
		tmpDir = GinkgoT().TempDir()
		originalResultsDir = resultsDir
		originalExecuteFunc = executeFunc
		originalOutput = output
		resultsDir = tmpDir
		output = io.Discard
	})

	AfterEach(func() {
		resultsDir = originalResultsDir
		executeFunc = originalExecuteFunc
		output = originalOutput
	})

	Describe("writeDoneFile", func() {
		It("should create done file with results dir path", func() {
			writeDoneFile()

			content, err := os.ReadFile(filepath.Join(tmpDir, "done"))
			Expect(err).ToNot(HaveOccurred())
			Expect(string(content)).To(Equal(tmpDir))
		})
	})

	Describe("run", func() {
		It("should return 0 on success and create done file", func() {
			executeFunc = func() error { return nil }

			exitCode := run()

			Expect(exitCode).To(Equal(0))
			Expect(filepath.Join(tmpDir, "done")).To(BeAnExistingFile())
		})

		It("should return 1 on failure and create done file", func() {
			executeFunc = func() error { return errors.New("test failure") }

			exitCode := run()

			Expect(exitCode).To(Equal(1))
			Expect(filepath.Join(tmpDir, "done")).To(BeAnExistingFile())
		})

		It("should create done file even on panic", func() {
			executeFunc = func() error { panic("test panic") }

			Expect(func() { run() }).To(Panic())
			Expect(filepath.Join(tmpDir, "done")).To(BeAnExistingFile())
		})
	})
})
