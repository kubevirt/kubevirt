CURDIR = $(shell pwd)

export GOPATH = $(CURDIR)/.gopath
export GOBIN = $(CURDIR)/bin
export GO15VENDOREXPERIMENT = 1
export PATH := ${PATH}:$(CURDIR)/bin

all: build contrib

build: gopath sync fmt vet
	./hack/build-go.sh install ${WHAT}

vet:
	./hack/build-go.sh vet ${WHAT}

fmt:
	./hack/build-go.sh fmt ${WHAT}

test: build
	./hack/build-go.sh test ${WHAT}

clean:
	./hack/build-go.sh clean ${WHAT}
	rm ./bin -rf

distclean: clean
	find vendor/ -maxdepth 1 -mindepth 1 -not -name vendor.json -exec rm {} -rf \;
	rm -f ./custer/manifest/*.yaml

sync:
	cd $(GOPATH) && govendor sync -v

docker: build
	./hack/build-docker.sh build ${WHAT}

publish: docker
	./hack/build-docker.sh push ${WHAT}

contrib: manifests

manifests: $(wildcard cluster/manifest/*.in)
	./hack/build-manifests.sh

# Sets up the gopath / build environment
gopath:
	mkdir -p .gopath/src/kubevirt.io/ bin
	ln -sf "$(CURDIR)" .gopath/src/kubevirt.io/kubevirt
	env | sort | grep GO

.PHONY: build fmt test clean distclean sync docker contrib vet publish
