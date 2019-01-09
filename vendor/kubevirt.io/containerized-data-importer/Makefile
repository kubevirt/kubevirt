#Copyright 2018 The CDI Authors.
#
#Licensed under the Apache License, Version 2.0 (the "License");
#you may not use this file except in compliance with the License.
#You may obtain a copy of the License at
#
#    http://www.apache.org/licenses/LICENSE-2.0
#
#Unless required by applicable law or agreed to in writing, software
#distributed under the License is distributed on an "AS IS" BASIS,
#WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
#See the License for the specific language governing permissions and
#limitations under the License.

.PHONY: build build-controller build-importer build-cloner build-apiserver build-uploadproxy build-uploadserver build-functest-image-init build-functest-image-http build-functest \
		docker docker-controller docker-cloner docker-importer docker-apiserver docker-uploadproxy docker-uploadserver docker-functest-image-init docker-functest-image-http\
		cluster-sync cluster-sync-controller cluster-sync-cloner cluster-sync-importer cluster-sync-apiserver cluster-sync-uploadproxy cluster-sync-uploadserver \
		test test-functional test-unit test-lint \
		publish \
		vet \
		format \
		manifests \
		goveralls \
		release-description

DOCKER=1
ifeq (${DOCKER}, 1)
DO=./hack/build/in-docker.sh
else
DO=eval
endif

all: docker

clean:
	${DO} "./hack/build/build-go.sh clean; rm -rf bin/* _out/* manifests/generated/* .coverprofile release-announcement"

generate:
	${DO} "./hack/update-codegen.sh"

generate-verify:
	${DO} "./hack/verify-codegen.sh"

apidocs:
	${DO} "./hack/update-codegen.sh && ./hack/gen-swagger-doc/gen-swagger-docs.sh v1alpha1 html"

build:
	${DO} "./hack/build/build-go.sh clean && ./hack/build/build-go.sh build ${WHAT} && ./hack/build/build-cdi-func-test-file-host.sh && DOCKER_REPO=${DOCKER_REPO} DOCKER_TAG=${DOCKER_TAG} VERBOSITY=${VERBOSITY} PULL_POLICY=${PULL_POLICY} ./hack/build/build-manifests.sh ${WHAT} && ./hack/build/build-copy-artifacts.sh ${WHAT}"

build-controller: WHAT = cmd/cdi-controller
build-controller: build
build-importer: WHAT = cmd/cdi-importer
build-importer: build
build-apiserver: WHAT = cmd/cdi-apiserver
build-apiserver: build
build-uploadproxy: WHAT = cmd/cdi-uploadproxy
build-uploadproxy: build
build-uploadserver: WHAT = cmd/cdi-uploadserver
build-uploadserver: build
build-cloner: WHAT = cmd/cdi-cloner
build-cloner: build
build-functest-image-init: WHAT = tools/cdi-func-test-file-host-init
build-functest-image-init:
build-functest:
	${DO} ./hack/build/build-functest.sh

# WHAT must match go tool style package paths for test targets (e.g. ./path/to/my/package/...)
test: test-unit test-functional test-lint

test-unit: WHAT = ./pkg/... ./cmd/...
test-unit:
	${DO} "./hack/build/run-tests.sh ${WHAT}"

test-functional:  WHAT = ./tests/...
test-functional:
	./hack/build/run-functional-tests.sh ${WHAT} "${TEST_ARGS}"

test-functional-ci: build-functest test-functional

# test-lint runs gofmt and golint tests against src files
test-lint:
	${DO} "./hack/build/run-lint-checks.sh"

docker: build
	./hack/build/build-docker.sh build ${WHAT}

docker-controller: WHAT = cmd/cdi-controller
docker-controller: docker
docker-importer: WHAT = cmd/cdi-importer
docker-importer: docker
docker-cloner: WHAT = cmd/cdi-cloner
docker-cloner: docker
docker-apiserver: WHAT = cmd/cdi-apiserver
docker-apiserver: docker
docker-uploadproxy: WHAT = cmd/cdi-uploadproxy
docker-uploadproxy: docker
docker-uploadserver: WHAT = cmd/cdi-uploadserver
docker-uploadserver: docker

docker-functest-image: docker-functest-image-http docker-functest-image-init
docker-functest-image-init: WHAT = tools/cdi-func-test-file-host-init
docker-functest-image-init: docker
docker-functest-image-http: WHAT = tools/cdi-func-test-file-host-http
docker-functest-image-http: # no code to compile, just build image
	./hack/build/build-cdi-func-test-file-host.sh && ./hack/build/build-docker.sh build ${WHAT}

push: docker
	./hack/build/build-docker.sh push ${WHAT}

push-controller: WHAT = cmd/cdi-controller
push-controller: push
push-importer: WHAT = cmd/cdi-importer
push-importer: push
push-cloner: WHAT = cdm/cdi-cloner
push-cloner: push
push-apiserver: WHAT = cmd/cdi-apiserver
push-apiserver: push
push-uploadproxy: WHAT = cmd/cdi-uploadproxy
push-uploadproxy: push
push-uploadserver: WHAT = cmd/cdi-uploadserver
push-uploadserver: push

publish: docker
	./hack/build/build-docker.sh publish ${WHAT}

vet:
	${DO} "./hack/build/build-go.sh vet ${WHAT}"

format:
	${DO} "./hack/build/format.sh"

manifests:
	${DO} "DOCKER_REPO=${DOCKER_REPO} DOCKER_TAG=${DOCKER_TAG} VERBOSITY=${VERBOSITY} PULL_POLICY=${PULL_POLICY} ./hack/build/build-manifests.sh"

goveralls:
	${DO} "TRAVIS_JOB_ID=${TRAVIS_JOB_ID} TRAVIS_PULL_REQUEST=${TRAVIS_PULL_REQUEST} TRAVIS_BRANCH=${TRAVIS_BRANCH} ./hack/build/goveralls.sh"

release-description:
	./hack/build/release-description.sh ${RELREF} ${PREREF}

cluster-up:
	./cluster/up.sh

cluster-down:
	./cluster/down.sh

cluster-clean:
	./cluster/clean.sh

cluster-sync: cluster-clean build ${WHAT}
	./cluster/sync.sh ${WHAT}

cluster-sync-controller: WHAT = cmd/cdi-controller
cluster-sync-controller: cluster-sync
cluster-sync-importer: WHAT = cmd/cdi-importer
cluster-sync-importer: cluster-sync
cluster-sync-cloner: WHAT = cmd/cdi-cloner
cluster-sync-cloner: cluster-sync
cluster-sync-apiserver: WHAT = cmd/cdi-apiserver
cluster-sync-apiserver: cluster-sync
cluster-sync-uploadproxy: WHAT = cmd/cdi-uploadproxy
cluster-sync-uploadproxy: cluster-sync
cluster-sync-uploadserver: WHAT = cmd/cdi-uploadserver
cluster-sync-uploadserver: cluster-sync

functest:
	./hack/build/functests.sh

functest-image-host: WHAT=tools/cdi-func-test-file-host-init
functest-image-host:  manifests build
	${DO} ./hack/build/build-cdi-func-test-file-host.sh && ./hack/build/build-docker.sh "tools/cdi-func-test-file-host-init tools/cdi-func-test-file-host-http"
