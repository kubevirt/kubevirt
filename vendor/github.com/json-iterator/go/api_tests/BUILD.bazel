load("@io_bazel_rules_go//go:def.bzl", "go_test")

go_test(
    name = "go_default_test",
    srcs = [
        "config_test.go",
        "decoder_test.go",
        "encoder_18_test.go",
        "encoder_test.go",
        "marshal_indent_test.go",
    ],
    deps = [
        "//vendor/github.com/json-iterator/go:go_default_library",
        "//vendor/github.com/stretchr/testify/require:go_default_library",
    ],
)
