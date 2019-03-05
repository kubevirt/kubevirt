load("@io_bazel_rules_go//go:def.bzl", "go_test")

go_test(
    name = "go_default_test",
    srcs = [
        "array_test.go",
        "builtin_test.go",
        "map_key_test.go",
        "map_test.go",
        "marshaler_string_test.go",
        "marshaler_struct_test.go",
        "slice_test.go",
        "struct_embedded_test.go",
        "struct_field_case_test.go",
        "struct_tags_test.go",
        "struct_test.go",
        "text_marshaler_string_test.go",
        "text_marshaler_struct_test.go",
        "type_test.go",
    ],
    deps = [
        "//vendor/github.com/davecgh/go-spew/spew:go_default_library",
        "//vendor/github.com/google/gofuzz:go_default_library",
        "//vendor/github.com/json-iterator/go:go_default_library",
    ],
)
