export GO15VENDOREXPERIMENT := 1

all: build manifests

generate:
	./hack/build-go.sh generate ${WHAT}

build: sync fmt vet compile

compile:
	./hack/build-go.sh install ${WHAT}

vet:
	./hack/build-go.sh vet ${WHAT}

fmt:
	goimports -w -local kubevirt.io cmd/ pkg/ tests/

test: build
	./hack/build-go.sh test ${WHAT}

clean:
	./hack/build-go.sh clean ${WHAT}
	rm ./bin -rf

distclean: clean
	find vendor/ -maxdepth 1 -mindepth 1 -not -name vendor.json -exec rm {} -rf \;
	rm -f manifest/*.yaml

sync:
	govendor sync

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

.PHONY: build fmt test clean distclean sync docker manifests vet publish
