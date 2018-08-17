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

.PHONY: build build-controller build-importer \
		docker docker-controller docker-cloner docker-importer \
		cluster-sync cluster-sync-controller cluster-sync-cloner cluster-sync-importer \
		test test-functional test-unit \
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

build:
	${DO} "./hack/build/build-go.sh clean && ./hack/build/build-go.sh build ${WHAT} && DOCKER_REPO=${DOCKER_REPO} DOCKER_TAG=${DOCKER_TAG} VERBOSITY=${VERBOSITY} PULL_POLICY=${PULL_POLICY} ./hack/build/build-manifests.sh ${WHAT} && ./hack/build/build-copy-artifacts.sh ${WHAT}"


build-controller: WHAT = cmd/cdi-controller
build-controller: build
build-importer: WHAT = cmd/cdi-importer
build-importer: build
# Note, the cloner is a bash script and has nothing to build

test:
	 ${DO} "./hack/build/build-go.sh test ${WHAT}"

test-unit: WHAT = pkg/
test-unit: test
test-functional: WHAT = test/
test-functional: test

docker: build
	./hack/build/build-docker.sh build ${WHAT}

docker-controller: WHAT = cmd/cdi-controller
docker-controller: docker
docker-importer: WHAT = cmd/cdi-importer
docker-importer: docker
docker-cloner: WHAT = cmd/cdi-cloner
docker-cloner: docker

push: docker
	./hack/build/build-docker.sh push ${WHAT}

push-controller: WHAT = cmd/cdi-controller
push-controller: push
push-importer: WHAT = cmd/cdi-importer
push-importer: push
push-cloner: WHAT = cdm/cdi-cloner
push-cloner: push

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

cluster-sync: build ${WHAT}
	./cluster/sync.sh ${WHAT}

cluster-sync-controller: WHAT = cmd/cdi-controller
cluster-sync-controller: cluster-sync
cluster-sync-importer: WHAT = cmd/cdi-importer
cluster-sync-importer: cluster-sync
cluster-sync-cloner: WHAT = cmd/cdi-cloner
cluster-sync-cloner: cluster-sync

functest: .functest-image-host
	./hack/build/functests.sh

.functest-image-host: WHAT=tools/cdi-func-test-file-host-init
.functest-image-host:  manifests build
	./hack/build/build-cdi-func-test-file-host.sh
