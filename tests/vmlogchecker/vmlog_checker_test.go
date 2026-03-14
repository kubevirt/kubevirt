package vmlogchecker

import (
	"regexp"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("ClassifyLogLine", func() {
	It("should return NotAnError for empty lines", func() {
		Expect(ClassifyLogLine("")).To(Equal(NotAnError))
	})

	DescribeTable("should return NotAnError when no error keyword is present",
		func(line string) {
			Expect(ClassifyLogLine(line)).To(Equal(NotAnError))
		},
		Entry("info level", `{"level":"info","msg":"Starting VM","pos":"manager.go:42"}`),
		Entry("warning level", `{"level":"warning","msg":"Something happened"}`),
		Entry("plain text", `plain text log with no keywords`),
	)

	DescribeTable("should return UnexpectedError for unrecognized errors",
		func(line string) {
			Expect(ClassifyLogLine(line)).To(Equal(UnexpectedError))
		},
		Entry("failed keyword", `{"level":"error","msg":"something totally unexpected failed"}`),
		Entry("fatal keyword", `{"level":"error","msg":"fatal crash in component"}`),
		Entry("panic keyword", `{"level":"error","msg":"panic in goroutine"}`),
	)

	It("should return AllowlistedError when line matches an allowlist pattern", func() {
		original := VirtLauncherErrorAllowlist
		VirtLauncherErrorAllowlist = []*regexp.Regexp{
			regexp.MustCompile(`known error pattern`),
		}
		defer func() { VirtLauncherErrorAllowlist = original }()

		Expect(ClassifyLogLine(`{"level":"error","msg":"known error pattern occurred"}`)).To(Equal(AllowlistedError))
	})
})

var _ = Describe("IsErrorLevel", func() {
	DescribeTable("should detect error-level JSON log lines",
		func(line string, expected bool) {
			Expect(IsErrorLevel(line)).To(Equal(expected))
		},
		Entry("error level", `{"level":"error","msg":"something failed"}`, true),
		Entry("info level", `{"level":"info","msg":"all good"}`, false),
		Entry("warning level", `{"level":"warning","msg":"be careful"}`, false),
		Entry("plain text", `just a plain line with no JSON`, false),
		Entry("empty", ``, false),
	)
})

var _ = Describe("containsErrorKeyword", func() {
	DescribeTable("should detect error keywords case-insensitively",
		func(line string, expected bool) {
			Expect(containsErrorKeyword(line)).To(Equal(expected))
		},
		Entry("Error", `something Error happened`, true),
		Entry("FAILED", `operation FAILED`, true),
		Entry("PANIC", `PANIC in handler`, true),
		Entry("Fatal", `Fatal shutdown`, true),
		Entry("no keyword", `everything is fine`, false),
		Entry("no keyword info", `info: started successfully`, false),
	)
})
