load("@io_bazel_rules_go//go:def.bzl", "go_test")

go_test(
    name = "go_default_test",
    srcs = [
        "array_test.go",
        "float64_test.go",
        "jsoniter_skip_test.go",
        "skip_test.go",
        "string_test.go",
        "struct_test.go",
    ],
    deps = [
        "//vendor/github.com/json-iterator/go:go_default_library",
        "//vendor/github.com/stretchr/testify/require:go_default_library",
    ],
)
