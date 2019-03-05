load("@io_bazel_rules_go//go:def.bzl", "go_test")

go_test(
    name = "go_default_test",
    srcs = [
        "jsoniter_any_array_test.go",
        "jsoniter_any_bool_test.go",
        "jsoniter_any_float_test.go",
        "jsoniter_any_int_test.go",
        "jsoniter_any_map_test.go",
        "jsoniter_any_null_test.go",
        "jsoniter_any_object_test.go",
        "jsoniter_any_string_test.go",
        "jsoniter_must_be_valid_test.go",
        "jsoniter_wrap_test.go",
    ],
    deps = [
        "//vendor/github.com/json-iterator/go:go_default_library",
        "//vendor/github.com/stretchr/testify/require:go_default_library",
    ],
)
