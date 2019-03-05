load("@io_bazel_rules_go//go:def.bzl", "go_test")

go_test(
    name = "go_default_test",
    srcs = [
        "encode_string_test.go",
        "jsoniter_large_file_test.go",
    ],
    deps = ["//vendor/github.com/json-iterator/go:go_default_library"],
)
