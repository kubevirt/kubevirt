all: build

build:
	cd pkg/ && go fmt ./... && go install -v ./...
deps-update:
	glide cc && glide update --strip-vendor
	hack/dep-prune.sh

test:
	cd pkg/ && go test -v ./...

.PHONY: build deps-update test
