export GO15VENDOREXPERIMENT := 1

ifeq (${TIMESTAMP}, 1)
  $(info "Timestamp is enabled")
  SHELL = ./hack/timestamps.sh
endif

all: format bazel-build manifests

go-all: go-build manifests-no-bazel

bazel-generate:
	SYNC_VENDOR=true hack/dockerized "./hack/bazel-generate.sh"

bazel-build:
	hack/dockerized "export BUILD_ARCH=${BUILD_ARCH} && export DOCKER_TAG=${DOCKER_TAG} && export CI=${CI} && export KUBEVIRT_RELEASE=${KUBEVIRT_RELEASE} && hack/bazel-fmt.sh && ./hack/multi-arch.sh build"

bazel-build-functests:
	hack/dockerized "hack/bazel-fmt.sh && hack/bazel-build-functests.sh"

build-functests: bazel-build-functests

bazel-build-image-bundle:
	hack/dockerized "export BUILD_ARCH=${BUILD_ARCH} && hack/bazel-fmt.sh && DOCKER_PREFIX=${DOCKER_PREFIX} DOCKER_TAG=${DOCKER_TAG} IMAGE_PREFIX=${IMAGE_PREFIX} hack/multi-arch.sh build-image-bundle"

bazel-build-verify: bazel-build
	./hack/dockerized "hack/bazel-fmt.sh"
	./hack/verify-generate.sh
	./hack/build-verify.sh
	./hack/dockerized "hack/bazel-test.sh"

bazel-build-images:
	hack/dockerized "export BUILD_ARCH=${BUILD_ARCH} && DOCKER_PREFIX=${DOCKER_PREFIX} DOCKER_TAG=${DOCKER_TAG} DOCKER_TAG_ALT=${DOCKER_TAG_ALT} IMAGE_PREFIX=${IMAGE_PREFIX} IMAGE_PREFIX_ALT=${IMAGE_PREFIX_ALT} ./hack/multi-arch.sh build-images"

bazel-push-images:
	hack/dockerized "export BUILD_ARCH=${BUILD_ARCH} && hack/bazel-fmt.sh && DOCKER_PREFIX=${DOCKER_PREFIX} DOCKER_TAG=${DOCKER_TAG} DOCKER_TAG_ALT=${DOCKER_TAG_ALT} IMAGE_PREFIX=${IMAGE_PREFIX} IMAGE_PREFIX_ALT=${IMAGE_PREFIX_ALT} KUBEVIRT_PROVIDER=${KUBEVIRT_PROVIDER} PUSH_TARGETS='${PUSH_TARGETS}' ./hack/multi-arch.sh push-images"
	BUILD_ARCH=${BUILD_ARCH} DOCKER_PREFIX=${DOCKER_PREFIX} DOCKER_TAG=${DOCKER_TAG} hack/push-container-manifest.sh

push: bazel-push-images

bazel-test:
	hack/dockerized "hack/bazel-fmt.sh && CI=${CI} ARTIFACTS=${ARTIFACTS} WHAT=${WHAT}  hack/bazel-test.sh"

gen-proto:
	hack/dockerized "DOCKER_PREFIX=${DOCKER_PREFIX} DOCKER_TAG=${DOCKER_TAG} IMAGE_PULL_POLICY=${IMAGE_PULL_POLICY} VERBOSITY=${VERBOSITY} ./hack/gen-proto.sh"

generate:
	hack/dockerized "DOCKER_PREFIX=${DOCKER_PREFIX} DOCKER_TAG=${DOCKER_TAG} IMAGE_PULL_POLICY=${IMAGE_PULL_POLICY} VERBOSITY=${VERBOSITY} ./hack/generate.sh"
	SYNC_VENDOR=true hack/dockerized "./hack/bazel-generate.sh && hack/bazel-fmt.sh"
	hack/dockerized hack/sync-kubevirtci.sh
	hack/dockerized hack/sync-common-instancetypes.sh

generate-verify: generate
	./hack/verify-generate.sh
	./hack/check-for-binaries.sh

apidocs:
	hack/dockerized "./hack/gen-swagger-doc/gen-swagger-docs.sh v1 html"

client-python:
	hack/dockerized "DOCKER_TAG=${DOCKER_TAG} ./hack/gen-client-python/generate.sh"

go-build:
	hack/dockerized "export KUBEVIRT_NO_BAZEL=true && KUBEVIRT_VERSION=${KUBEVIRT_VERSION} KUBEVIRT_GO_BUILD_TAGS=${KUBEVIRT_GO_BUILD_TAGS} KUBEVIRT_RELEASE=${KUBEVIRT_RELEASE} ./hack/build-go.sh install ${WHAT}" && ./hack/build-copy-artifacts.sh ${WHAT}

go-build-functests:
	hack/dockerized "export KUBEVIRT_NO_BAZEL=true && KUBEVIRT_GO_BUILD_TAGS=${KUBEVIRT_GO_BUILD_TAGS} ./hack/go-build-functests.sh"

gosec:
	hack/dockerized "GOSEC=${GOSEC} ARTIFACTS=${ARTIFACTS} ./hack/gosec.sh"

coverage:
	hack/dockerized "./hack/coverage.sh ${WHAT}"

goveralls:
	SYNC_OUT=false hack/dockerized "COVERALLS_TOKEN_FILE=${COVERALLS_TOKEN_FILE} COVERALLS_TOKEN=${COVERALLS_TOKEN} CI_NAME=prow CI_BRANCH=${PULL_BASE_REF} CI_PR_NUMBER=${PULL_NUMBER} GIT_ID=${PULL_PULL_SHA} PROW_JOB_ID=${PROW_JOB_ID} ./hack/bazel-goveralls.sh"

go-test: go-build
	SYNC_OUT=false hack/dockerized "export KUBEVIRT_NO_BAZEL=true && KUBEVIRT_GO_BUILD_TAGS=${KUBEVIRT_GO_BUILD_TAGS} ./hack/build-go.sh test ${WHAT}"

test: bazel-test

fuzz:
	hack/dockerized "./hack/fuzz.sh"

integ-test:
	hack/integration-test.sh

functest: build-functests
	hack/functests.sh

dump: bazel-build
	hack/dump.sh

functest-image-build: manifests build-functests
	hack/func-tests-image.sh build

functest-image-push: functest-image-build
	hack/func-tests-image.sh push

conformance:
	hack/dockerized "export KUBEVIRT_PROVIDER=${KUBEVIRT_PROVIDER} SKIP_OUTSIDE_CONN_TESTS=${SKIP_OUTSIDE_CONN_TESTS} RUN_ON_ARM64_INFRA=${RUN_ON_ARM64_INFRA} KUBEVIRT_E2E_FOCUS=${KUBEVIRT_E2E_FOCUS} DOCKER_PREFIX=${DOCKER_PREFIX} DOCKER_TAG=${DOCKER_TAG} && hack/conformance.sh"

perftest: build-functests
	hack/perftests.sh

realtime-perftest: build-functests
	hack/realtime-perftests.sh

clean:
	hack/dockerized "./hack/build-go.sh clean ${WHAT} && rm _out/* -rf"
	hack/dockerized "bazel clean --expunge"
	rm -f tools/openapispec/openapispec tools/resource-generator/resource-generator tools/manifest-templator/manifest-templator tools/vms-generator/vms-generator

distclean: clean
	hack/dockerized "rm -rf vendor/ && rm -f go.sum && GO111MODULE=on go clean -modcache"
	rm -rf vendor/

cluster-patch:
	hack/dockerized "export BUILD_ARCH=${BUILD_ARCH} && hack/bazel-fmt.sh && DOCKER_PREFIX=${DOCKER_PREFIX} DOCKER_TAG=${DOCKER_TAG} DOCKER_TAG_ALT=${DOCKER_TAG_ALT} IMAGE_PREFIX=${IMAGE_PREFIX} IMAGE_PREFIX_ALT=${IMAGE_PREFIX_ALT} KUBEVIRT_PROVIDER=${KUBEVIRT_PROVIDER} PUSH_TARGETS='virt-api virt-controller virt-handler virt-launcher' ./hack/bazel-push-images.sh"
	hack/cluster-patch.sh

deps-update-patch:
	SYNC_VENDOR=true hack/dockerized " ./hack/dep-update.sh -- -u=patch && ./hack/dep-prune.sh && ./hack/bazel-generate.sh"

deps-update:
	SYNC_VENDOR=true hack/dockerized " ./hack/dep-update.sh && ./hack/dep-prune.sh && ./hack/bazel-generate.sh"

deps-sync:
	SYNC_VENDOR=true hack/dockerized " ./hack/dep-update.sh --sync-only && ./hack/dep-prune.sh && ./hack/bazel-generate.sh"

rpm-deps:
	SYNC_VENDOR=true hack/dockerized "CUSTOM_REPO=${CUSTOM_REPO} SINGLE_ARCH=${SINGLE_ARCH} BASESYSTEM=${BASESYSTEM} LIBVIRT_VERSION=${LIBVIRT_VERSION} QEMU_VERSION=${QEMU_VERSION} SEABIOS_VERSION=${SEABIOS_VERSION} EDK2_VERSION=${EDK2_VERSION} LIBGUESTFS_VERSION=${LIBGUESTFS_VERSION} GUESTFSTOOLS_VERSION=${GUESTFSTOOLS_VERSION} PASST_VERSION=${PASST_VERSION} VIRTIOFSD_VERSION=${VIRTIOFSD_VERSION} SWTPM_VERSION=${SWTPM_VERSION} ./hack/rpm-deps.sh"

bump-images:
	hack/dockerized "./hack/rpm-deps.sh && ./hack/bump-distroless.sh"

verify-rpm-deps:
	SYNC_VENDOR=true hack/dockerized " ./hack/verify-rpm-deps.sh"

build-verify:
	hack/build-verify.sh

manifests:
	hack/manifests.sh

manifests-no-bazel:
	KUBEVIRT_NO_BAZEL=true hack/manifests.sh

cluster-up:
	./hack/cluster-up.sh

cluster-down:
	./cluster-up/down.sh

cluster-build:
	./hack/cluster-build.sh

cluster-clean:
	./hack/cluster-clean.sh

cluster-deploy: cluster-clean
	./hack/cluster-deploy.sh

cluster-sync:
	./hack/cluster-sync.sh

builder-build:
	./hack/builder/build.sh

builder-publish:
	./hack/builder/publish.sh

olm-verify:
	hack/dockerized "./hack/olm.sh verify"

current-dir := $(realpath .)
rule-spec-dumper-executable := "rule-spec-dumper"

build-prom-spec-dumper:
	hack/dockerized "go build -o ${rule-spec-dumper-executable} ./hack/prom-rule-ci/rule-spec-dumper.go"

clean-prom-spec-dumper:
	rm -f ${rule-spec-dumper-executable}

prom-rules-verify: build-prom-spec-dumper
	./hack/prom-rule-ci/verify-rules.sh \
		"${current-dir}/${rule-spec-dumper-executable}" \
		"${current-dir}/hack/prom-rule-ci/prom-rules-tests.yaml"
	rm ${rule-spec-dumper-executable}

olm-push:
	hack/dockerized "DOCKER_TAG=${DOCKER_TAG} CSV_VERSION=${CSV_VERSION} QUAY_USERNAME=${QUAY_USERNAME} \
	    QUAY_PASSWORD=${QUAY_PASSWORD} QUAY_REPOSITORY=${QUAY_REPOSITORY} PACKAGE_NAME=${PACKAGE_NAME} ./hack/olm.sh push"

bump-kubevirtci:
	./hack/bump-kubevirtci.sh

fossa:
	hack/dockerized "FOSSA_TOKEN_FILE=${FOSSA_TOKEN_FILE} PULL_BASE_REF=${PULL_BASE_REF} CI=${CI} ./hack/fossa.sh"

format:
	./hack/dockerized "hack/bazel-fmt.sh"

fmt: format

lint:
	if [ $$(wc -l < tests/utils.go) -gt 2813 ]; then echo >&2 "do not make tests/utils longer"; exit 1; fi

	hack/dockerized "golangci-lint run --timeout 20m --verbose \
	  pkg/instancetype/... \
	  pkg/network/namescheme/... \
	  pkg/network/domainspec/... \
	  pkg/network/sriov/... \
	  tests/console/... \
	  tests/libnet/... \
	  tests/libvmi/... \
	"

lint-metrics:
	hack/dockerized "./hack/prom-metric-linter/metrics_collector.sh > metrics.json"
	./hack/prom-metric-linter/metric_name_linter.sh --operator-name="kubevirt" --sub-operator-name="kubevirt" --metrics-file=metrics.json
	rm metrics.json

.PHONY: \
	build-verify \
	conformance \
	go-build \
	go-test \
	go-all \
	bazel-generate \
	bazel-build \
	bazel-build-image-bundle \
	bazel-build-images \
	bazel-push-images \
	bazel-test \
	functest-image-build \
	functest-image-push \
	test \
	clean \
	distclean \
	deps-sync \
	sync \
	manifests \
	functest \
	cluster-up \
	cluster-down \
	cluster-clean \
	cluster-deploy \
	cluster-sync \
	olm-verify \
	olm-push \
	coverage \
	goveralls \
	build-functests \
	fossa \
	realtime-perftest \
	format \
	fmt \
	lint \
	lint-metrics\
	$(NULL)
