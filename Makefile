export GO15VENDOREXPERIMENT := 1

all: build

build: sync fmt vet
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
	rm -f ./contrib/manifest/*.yaml

sync:
	govendor sync

docker: build
	./hack/build-docker.sh build ${WHAT}

contrib:
	./hack/build-contrib.sh

.PHONY: build fmt test clean distclean sync docker contrib vet
