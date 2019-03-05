load("@io_bazel_rules_go//go:def.bzl", "go_test")

go_test(
    name = "go_default_test",
    srcs = [
        "decoder_test.go",
        "extension_test.go",
    ],
    deps = [
        "//vendor/github.com/json-iterator/go:go_default_library",
        "//vendor/github.com/modern-go/reflect2:go_default_library",
        "//vendor/github.com/stretchr/testify/require:go_default_library",
    ],
)
