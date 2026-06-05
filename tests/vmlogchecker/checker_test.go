package vmlogchecker_test

import (
	"regexp"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"kubevirt.io/kubevirt/tests/vmlogchecker"
)

var _ = Describe("ClassifyLogLine", func() {
	It("should return NotAnError for empty lines", func() {
		Expect(vmlogchecker.ClassifyLogLine("")).To(Equal(vmlogchecker.NotAnError))
	})

	DescribeTable("should return NotAnError when no error keyword is present",
		func(line string) {
			Expect(vmlogchecker.ClassifyLogLine(line)).To(Equal(vmlogchecker.NotAnError))
		},
		Entry("info level", `{"level":"info","msg":"Starting VM","pos":"manager.go:42"}`),
		Entry("warning level", `{"level":"warning","msg":"Something happened"}`),
		Entry("plain text", `plain text log with no keywords`),
	)

	DescribeTable("should return UnexpectedError for unrecognized errors",
		func(line string) {
			Expect(vmlogchecker.ClassifyLogLine(line)).To(Equal(vmlogchecker.UnexpectedError))
		},
		Entry("failed keyword", `{"level":"error","msg":"something totally unexpected failed"}`),
		Entry("fatal keyword", `{"level":"error","msg":"fatal crash in component"}`),
		Entry("panic keyword", `{"level":"error","msg":"panic in goroutine"}`),
	)

	It("should return AllowlistedError when line matches an allowlist pattern", func() {
		original := vmlogchecker.VirtLauncherErrorAllowlist
		vmlogchecker.VirtLauncherErrorAllowlist = []vmlogchecker.AllowlistEntry{
			{ID: 1, Regex: regexp.MustCompile(`known error pattern`)},
		}
		defer func() { vmlogchecker.VirtLauncherErrorAllowlist = original }()

		Expect(vmlogchecker.ClassifyLogLine(`{"level":"error","msg":"known error pattern occurred"}`)).To(Equal(vmlogchecker.AllowlistedError))
	})
})

var _ = Describe("IsErrorLevel", func() {
	DescribeTable("should detect error-level JSON log lines",
		func(line string, expected bool) {
			Expect(vmlogchecker.IsErrorLevel(line)).To(Equal(expected))
		},
		Entry("error level", `{"level":"error","msg":"something failed"}`, true),
		Entry("info level", `{"level":"info","msg":"all good"}`, false),
		Entry("warning level", `{"level":"warning","msg":"be careful"}`, false),
		Entry("plain text", `just a plain line with no JSON`, false),
		Entry("empty", ``, false),
	)
})
