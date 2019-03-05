load("@io_bazel_rules_go//go:def.bzl", "go_test")

go_test(
    name = "go_default_test",
    srcs = [
        "array_test.go",
        "bool_test.go",
        "eface_test.go",
        "error_test.go",
        "float_test.go",
        "iface_test.go",
        "int_test.go",
        "invalid_test.go",
        "map_test.go",
        "marshaler_test.go",
        "number_test.go",
        "ptr_test.go",
        "raw_message_test.go",
        "slice_test.go",
        "string_test.go",
        "struct_test.go",
        "value_test.go",
    ],
    deps = [
        "//vendor/github.com/json-iterator/go:go_default_library",
        "//vendor/github.com/modern-go/reflect2:go_default_library",
        "//vendor/github.com/stretchr/testify/assert:go_default_library",
        "//vendor/github.com/stretchr/testify/require:go_default_library",
    ],
)
