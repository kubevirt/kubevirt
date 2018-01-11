export GO15VENDOREXPERIMENT := 1

UNAME := $(shell uname)
ifeq ($(UNAME), Darwin)
    HASH := md5
else
    HASH := md5sum
endif

all: build manifests

generate: sync
	find pkg/ -name "*generated*.go" -exec rm {} -f \;
	./hack/build-go.sh generate ${WHAT}
	goimports -w -local kubevirt.io cmd/ pkg/ tests/
	./hack/bootstrap-ginkgo.sh
	(cd tools/openapispec/ && go build)
	tools/openapispec/openapispec --dump-api-spec-path api/openapi-spec/swagger.json

apidocs: generate
	./hack/gen-swagger-doc/gen-swagger-docs.sh v1 html

build: checksync fmt vet compile

goveralls:
	./hack/goveralls.sh

compile:
	./hack/build-go.sh install ${WHAT}

vet:
	./hack/build-go.sh vet ${WHAT}

fmt:
	goimports -w -local kubevirt.io cmd/ pkg/ tests/

fmt-bash:
	shfmt -i 4 -w cluster/ hack/ images/

test: build
	./hack/build-go.sh test ${WHAT}

functest:
	./hack/functests.sh

clean:
	./hack/build-go.sh clean ${WHAT}
	rm _out/ -rf
	rm tools/openapispec/openapispec -rf

distclean: clean
	find manifests -name "*.yaml" -type f -delete
	rm -rf vendor/
	rm -f .glide.*.hash
	glide cc

checksync:
	@test -f .glide.yaml.hash || ${HASH} glide.yaml > .glide.yaml.hash
	@if [ "`${HASH} glide.yaml`" != "`cat .glide.yaml.hash`" ]; then \
		glide cc; \
		glide update --strip-vendor; \
		${HASH} glide.yaml > .glide.yaml.hash; \
		${HASH} glide.lock > .glide.lock.hash; \
	elif [ "`${HASH} glide.lock`" != "`cat .glide.lock.hash`" ]; then \
 		make sync; \
 	fi
 
sync:
	glide install --strip-vendor
	${HASH} glide.lock > .glide.lock.hash

docker:
	hack/dockerized make build
	./hack/build-docker.sh build ${WHAT}

publish: docker
	./hack/build-docker.sh push ${WHAT}

manifests:
	./hack/build-manifests.sh

check: check-bash vet
	test -z "`./hack/build-go.sh fmt`"

check-bash:
	find . -name \*.sh -exec bash -n \{\} \;

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

cluster-deploy:
	./cluster/deploy.sh

cluster-sync: cluster-build cluster-deploy

.PHONY: build fmt test clean distclean checksync sync docker manifests vet publish functest release-announce fmt-bash cluster-up cluster-down cluster-deploy cluster-sync
