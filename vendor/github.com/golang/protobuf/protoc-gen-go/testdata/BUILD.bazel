load("@io_bazel_rules_go//go:def.bzl", "go_test")

go_test(
    name = "go_default_test",
    srcs = [
        "extension_test.go",
        "import_public_test.go",
        "main_test.go",
    ],
    deps = [
        "//vendor/github.com/golang/protobuf/proto:go_default_library",
        "//vendor/github.com/golang/protobuf/protoc-gen-go/testdata/extension_base:go_default_library",
        "//vendor/github.com/golang/protobuf/protoc-gen-go/testdata/extension_user:go_default_library",
        "//vendor/github.com/golang/protobuf/protoc-gen-go/testdata/import_public:go_default_library",
        "//vendor/github.com/golang/protobuf/protoc-gen-go/testdata/import_public/sub:go_default_library",
        "//vendor/github.com/golang/protobuf/protoc-gen-go/testdata/imports:go_default_library",
        "//vendor/github.com/golang/protobuf/protoc-gen-go/testdata/multi:go_default_library",
        "//vendor/github.com/golang/protobuf/protoc-gen-go/testdata/my_test:go_default_library",
    ],
)
