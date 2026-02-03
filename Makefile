export GO15VENDOREXPERIMENT := 1

ifeq (${CI}, true)
  TIMESTAMP=1
endif

ifeq (${TIMESTAMP}, 1)
  SHELL = ./hack/timestamps.sh
endif

# =============================================================================
# Native Go Build
# =============================================================================

all: format go-build manifests

build: go-build

test: go-test

build-functests: go-build-functests

rpm-deps: go-rpm-deps

build-images: go-build-images

go-all: go-build manifests

# Native image build
go-build-images: go-build
	./hack/build-images.sh --registry ${DOCKER_PREFIX} --tag ${DOCKER_TAG} \
		virt-launcher virt-handler virt-api virt-controller virt-operator \
		virt-exportserver virt-exportproxy sidecar-shim libguestfs-tools pr-helper

push: go-build-images
	./hack/build-images.sh --push --registry ${DOCKER_PREFIX} --tag ${DOCKER_TAG} \
		virt-launcher virt-handler virt-api virt-controller virt-operator \
		virt-exportserver virt-exportproxy sidecar-shim libguestfs-tools pr-helper
	BUILD_ARCH=${BUILD_ARCH} DOCKER_PREFIX=${DOCKER_PREFIX} DOCKER_TAG=${DOCKER_TAG} hack/push-container-manifest.sh

gen-proto:
	hack/dockerized "DOCKER_PREFIX=${DOCKER_PREFIX} DOCKER_TAG=${DOCKER_TAG} IMAGE_PULL_POLICY=${IMAGE_PULL_POLICY} VERBOSITY=${VERBOSITY} ./hack/gen-proto.sh"

generate:
	hack/dockerized hack/build-ginkgo.sh
	hack/dockerized "DOCKER_PREFIX=${DOCKER_PREFIX} DOCKER_TAG=${DOCKER_TAG} IMAGE_PULL_POLICY=${IMAGE_PULL_POLICY} VERBOSITY=${VERBOSITY} ./hack/generate.sh"
	hack/dockerized hack/sync-kubevirtci.sh
	hack/dockerized hack/common-instancetypes/sync.sh
	./hack/update-generated-api-testdata.sh

generate-verify: generate
	./hack/verify-generate.sh
	./hack/check-for-binaries.sh

apidocs:
	hack/dockerized "./hack/gen-swagger-doc/gen-swagger-docs.sh v1 html"

client-python:
	hack/dockerized "DOCKER_TAG=${DOCKER_TAG} ./hack/gen-client-python/generate.sh"

go-build:
	hack/dockerized "export KUBEVIRT_VERSION=${KUBEVIRT_VERSION} && KUBEVIRT_GO_BUILD_TAGS=${KUBEVIRT_GO_BUILD_TAGS} KUBEVIRT_RELEASE=${KUBEVIRT_RELEASE} ./hack/build-go.sh install ${WHAT}" && ./hack/build-copy-artifacts.sh ${WHAT}

go-build-functests:
	hack/dockerized "KUBEVIRT_GO_BUILD_TAGS=${KUBEVIRT_GO_BUILD_TAGS} ./hack/go-build-functests.sh"

gosec:
	hack/dockerized "GOSEC=${GOSEC} ARTIFACTS=${ARTIFACTS} ./hack/gosec.sh"

coverage:
	hack/dockerized "./hack/coverage.sh ${WHAT}"

goveralls:
	SYNC_OUT=false hack/dockerized "COVERALLS_TOKEN_FILE=${COVERALLS_TOKEN_FILE} COVERALLS_TOKEN=${COVERALLS_TOKEN} CI_NAME=prow CI_BRANCH=${PULL_BASE_REF} CI_PR_NUMBER=${PULL_NUMBER} GIT_ID=${PULL_PULL_SHA} PROW_JOB_ID=${PROW_JOB_ID} ./hack/goveralls.sh"

go-test: go-build
	SYNC_OUT=false hack/dockerized "export KUBEVIRT_GO_BUILD_TAGS=${KUBEVIRT_GO_BUILD_TAGS} && ./hack/build-go.sh test ${WHAT}"

fuzz:
	hack/dockerized "./hack/fuzz.sh"

integ-test:
	hack/integration-test.sh

functest: build-functests
	hack/functests.sh

dump: go-build
	hack/dump.sh

functest-image-build: manifests build-functests
	hack/func-tests-image.sh build

functest-image-push: functest-image-build
	hack/func-tests-image.sh push

conformance:
	hack/dockerized "export KUBEVIRT_PROVIDER=${KUBEVIRT_PROVIDER} SKIP_OUTSIDE_CONN_TESTS=${SKIP_OUTSIDE_CONN_TESTS} RUN_ON_ARM64_INFRA=${RUN_ON_ARM64_INFRA} SKIP_BLOCK_STORAGE_TESTS=${SKIP_BLOCK_STORAGE_TESTS} SKIP_SNAPSHOT_STORAGE_TESTS=${SKIP_SNAPSHOT_STORAGE_TESTS} KUBEVIRT_E2E_FOCUS=${KUBEVIRT_E2E_FOCUS} DOCKER_PREFIX=${DOCKER_PREFIX} DOCKER_TAG=${DOCKER_TAG} && hack/conformance.sh"

perftest: build-functests
	hack/perftests.sh

kwok-perftest: build-functests
	hack/kwok-perftests.sh

realtime-perftest: build-functests
	hack/realtime-perftests.sh

clean:
	hack/dockerized "./hack/build-go.sh clean ${WHAT} && rm _out/* -rf"
	rm -f tools/openapispec/openapispec tools/resource-generator/resource-generator tools/manifest-templator/manifest-templator tools/vms-generator/vms-generator

distclean: clean
	hack/dockerized "rm -rf vendor/ && rm -f go.sum && GO111MODULE=on go clean -modcache"
	rm -rf vendor/

deps-update-patch:
	SYNC_VENDOR=true hack/dockerized "./hack/dep-update.sh -- -u=patch"

deps-update:
	SYNC_VENDOR=true hack/dockerized "./hack/dep-update.sh"

deps-sync:
	SYNC_VENDOR=true hack/dockerized "./hack/dep-update.sh --sync-only"

# Native RPM freeze (generates JSON lock files)
go-rpm-deps:
	hack/dockerized "./hack/rpm-freeze-all.sh"

verify-rpm-deps:
	hack/dockerized "./hack/verify-rpm-deps.sh"

build-verify:
	hack/build-verify.sh

manifests:
	hack/manifests.sh

cluster-up:
	./hack/cluster-up.sh

cluster-down:
	./kubevirtci/cluster-up/down.sh

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
	./hack/dockerized "hack/gofumpt.sh"

fmt: format

lint:
	hack/dockerized "hack/lint-test-cleanup-label.sh"
	hack/dockerized "hack/golangci-lint.sh"
	hack/dockerized "monitoringlinter ./pkg/..."
	hack/dockerized "hack/license-header-check.sh"

lint-metrics:
	hack/dockerized "./hack/prom-metric-linter/metrics_collector.sh > metrics.json"
	./hack/prom-metric-linter/metric_name_linter.sh --operator-name="kubevirt" --sub-operator-name="kubevirt" --metrics-file=metrics.json
	rm metrics.json

gofumpt:
	./hack/dockerized "hack/gofumpt.sh"

update-generated-api-testdata:
	./hack/update-generated-api-testdata.sh

.PHONY: \
	all \
	build \
	build-verify \
	conformance \
	go-build \
	go-test \
	go-all \
	go-build-images \
	functest-image-build \
	functest-image-push \
	test \
	clean \
	distclean \
	deps-sync \
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
	lint-metrics \
	update-generated-api-testdata \
	$(NULL)
