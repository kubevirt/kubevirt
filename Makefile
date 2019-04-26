export GO15VENDOREXPERIMENT := 1


all:
	hack/dockerized "DOCKER_PREFIX=${DOCKER_PREFIX} DOCKER_TAG=${DOCKER_TAG} IMAGE_PULL_POLICY=${IMAGE_PULL_POLICY} VERBOSITY=${VERBOSITY} ./hack/build-manifests.sh && \
	    hack/bazel-fmt.sh && hack/bazel-build.sh"

go-all:
	hack/dockerized "KUBEVIRT_VERSION=${KUBEVIRT_VERSION} ./hack/build-go.sh install ${WHAT} && ./hack/build-copy-artifacts.sh ${WHAT} && DOCKER_PREFIX=${DOCKER_PREFIX} DOCKER_TAG=${DOCKER_TAG} IMAGE_PULL_POLICY=${IMAGE_PULL_POLICY} VERBOSITY=${VERBOSITY} ./hack/build-manifests.sh"

bazel-generate:
	SYNC_VENDOR=true hack/dockerized "./hack/bazel-generate.sh"

bazel-build:
	hack/dockerized "hack/bazel-fmt.sh && hack/bazel-build.sh"

bazel-build-images:
	hack/dockerized "DOCKER_PREFIX=${DOCKER_PREFIX} DOCKER_TAG=${DOCKER_TAG} DOCKER_TAG_ALT=${DOCKER_TAG_ALT} ./hack/bazel-build-images.sh"

bazel-push-images:
	hack/dockerized "hack/bazel-fmt.sh && DOCKER_PREFIX=${DOCKER_PREFIX} DOCKER_TAG=${DOCKER_TAG} DOCKER_TAG_ALT=${DOCKER_TAG_ALT} ./hack/bazel-push-images.sh"

push: bazel-push-images

bazel-tests:
	hack/dockerized "hack/bazel-fmt.sh && bazel test \
		--platforms=@io_bazel_rules_go//go/toolchain:linux_amd64 \
		--workspace_status_command=./hack/print-workspace-status.sh \
        --test_output=errors -- //pkg/..."

generate:
	hack/dockerized "DOCKER_PREFIX=${DOCKER_PREFIX} DOCKER_TAG=${DOCKER_TAG} IMAGE_PULL_POLICY=${IMAGE_PULL_POLICY} VERBOSITY=${VERBOSITY} ./hack/generate.sh"
	SYNC_VENDOR=true hack/dockerized "./hack/bazel-generate.sh && hack/bazel-fmt.sh"

apidocs:
	hack/dockerized "./hack/generate.sh && ./hack/gen-swagger-doc/gen-swagger-docs.sh v1 html"

client-python:
	hack/dockerized "./hack/generate.sh && TRAVIS_TAG=${TRAVIS_TAG} ./hack/gen-client-python/generate.sh"

go-build:
	hack/dockerized "KUBEVIRT_VERSION=${KUBEVIRT_VERSION} ./hack/build-go.sh install ${WHAT}" && ./hack/build-copy-artifacts.sh ${WHAT}

coverage:
	hack/dockerized "./hack/coverage.sh ${WHAT}"

goveralls: go-build
	SYNC_OUT=false hack/dockerized "TRAVIS_JOB_ID=${TRAVIS_JOB_ID} TRAVIS_PULL_REQUEST=${TRAVIS_PULL_REQUEST} TRAVIS_BRANCH=${TRAVIS_BRANCH} ./hack/goveralls.sh"

go-test: go-build
	SYNC_OUT=false hack/dockerized "./hack/build-go.sh test ${WHAT}"

test: go-test

functest:
	hack/dockerized "hack/build-func-tests.sh"
	hack/functests.sh

clean:
	hack/dockerized "./hack/build-go.sh clean ${WHAT} && rm _out/* -rf"
	hack/dockerized "bazel clean --expunge"
	rm -f tools/openapispec/openapispec tools/resource-generator/resource-generator tools/manifest-templator/manifest-templator tools/vms-generator/vms-generator tools/marketplace/marketplace

distclean: clean
	hack/dockerized "rm -rf vendor/ && rm -f .glide.*.hash && glide cc"
	rm -rf vendor/

deps-install:
	SYNC_VENDOR=true hack/dockerized "glide install --strip-vendor && hack/dep-prune.sh && ./hack/bazel-generate.sh"

deps-update:
	SYNC_VENDOR=true hack/dockerized "glide cc && glide update --strip-vendor && hack/dep-prune.sh && ./hack/bazel-generate.sh"

check:
	hack/dockerized "./hack/check.sh"

docker: build
	hack/build-docker.sh build ${WHAT}

push-cache: docker verify-build
	hack/build-docker.sh push-cache ${WHAT}

pull-cache:
	hack/build-docker.sh pull-cache ${WHAT}

publish: docker verify-build
	hack/build-docker.sh push ${WHAT}

verify-build:
	hack/verify-build.sh

manifests:
	hack/dockerized "CSV_VERSION=${CSV_VERSION} QUAY_REPOSITORY=${QUAY_REPOSITORY} \
	  DOCKER_PREFIX=${DOCKER_PREFIX} DOCKER_TAG=${DOCKER_TAG} \
	  IMAGE_PULL_POLICY=${IMAGE_PULL_POLICY} VERBOSITY=${VERBOSITY} PACKAGE_NAME=${PACKAGE_NAME} ./hack/build-manifests.sh"

.release-functest:
	make functest > .release-functest 2>&1

release-announce: .release-functest
	./hack/release-announce.sh $(RELREF) $(PREREF)

cluster-up:
	./cluster/up.sh

cluster-down:
	./cluster/down.sh

cluster-build:
	./cluster/build.sh

cluster-clean:
	./cluster/clean.sh

cluster-deploy: cluster-clean
	./cluster/deploy.sh

cluster-sync: cluster-build cluster-deploy

builder-build:
	./hack/builder/build.sh

builder-publish:
	./hack/builder/publish.sh

olm-verify:
	hack/dockerized "./hack/olm.sh verify"

olm-push:
	hack/dockerized "DOCKER_TAG=${DOCKER_TAG} CSV_VERSION=${CSV_VERSION} QUAY_USERNAME=${QUAY_USERNAME} \
	    QUAY_PASSWORD=${QUAY_PASSWORD} QUAY_REPOSITORY=${QUAY_REPOSITORY} PACKAGE_NAME=${PACKAGE_NAME} ./hack/olm.sh push"

.PHONY: \
	go-build \
	go-test \
	go-all \
	bazel-generate \
	bazel-build \
	bazel-build-images \
	bazel-push-images \
	bazel-tests \
	build \
	test \
	clean \
	distclean \
	checksync \
	sync \
	docker \
	manifests \
	publish \
	functest \
	release-announce \
	cluster-up \
	cluster-down \
	cluster-clean \
	cluster-deploy \
	cluster-sync \
	olm-verify \
	olm-push
