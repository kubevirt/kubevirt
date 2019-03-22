load("@io_bazel_rules_go//go:def.bzl", "go_test")

go_test(
    name = "go_default_test",
    srcs = [
        "jsoniter_array_test.go",
        "jsoniter_bool_test.go",
        "jsoniter_float_test.go",
        "jsoniter_int_test.go",
        "jsoniter_interface_test.go",
        "jsoniter_iterator_test.go",
        "jsoniter_map_test.go",
        "jsoniter_nested_test.go",
        "jsoniter_null_test.go",
        "jsoniter_object_test.go",
        "jsoniter_raw_message_test.go",
    ],
    deps = [
        "//vendor/github.com/json-iterator/go:go_default_library",
        "//vendor/github.com/stretchr/testify/require:go_default_library",
    ],
)
