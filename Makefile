export GO15VENDOREXPERIMENT := 1

all: build manifests

generate:
	find pkg/ -name "*generated*.go" -exec rm {} -f \;
	./hack/build-go.sh generate ${WHAT}
	goimports -w -local kubevirt.io cmd/ pkg/ tests/

build: sync fmt vet compile

compile:
	./hack/build-go.sh install ${WHAT}

vet:
	./hack/build-go.sh vet ${WHAT}

fmt:
	goimports -w -local kubevirt.io cmd/ pkg/ tests/

test: build
	./hack/build-go.sh test ${WHAT}

functest:
	./hack/build-go.sh functest ${WHAT}

clean:
	./hack/build-go.sh clean ${WHAT}
	rm ./bin -rf

distclean: clean
	find vendor/ -maxdepth 1 -mindepth 1 -not -name vendor.json -exec rm {} -rf \;
	rm -f manifest/*.yaml

sync:
	glide install

docker: build
	./hack/build-docker.sh build ${WHAT}

publish: docker
	./hack/build-docker.sh push ${WHAT}

manifests: $(wildcard manifests/*.in)
	./hack/build-manifests.sh

check: check-bash vet
	test -z "`./hack/build-go.sh fmt`"

check-bash:
	find . -name \*.sh -exec bash -n \{\} \;

vagrant-sync-config:
	./cluster/vagrant/sync_config.sh

vagrant-sync-build: build
	./cluster/vagrant/sync_build.sh

vagrant-deploy: vagrant-sync-config vagrant-sync-build
	export KUBECTL="cluster/kubectl.sh --core" && ./cluster/deploy.sh

.PHONY: build fmt test clean distclean sync docker manifests vet publish vagrant-sync-config vagrant-sync-build vagrant-deploy functest
