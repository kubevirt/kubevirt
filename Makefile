export GO15VENDOREXPERIMENT := 1

HASH := md5sum

all: build manifests

generate:
	find pkg/ -name "*generated*.go" -exec rm {} -f \;
	./hack/build-go.sh generate ${WHAT}
	goimports -w -local kubevirt.io cmd/ pkg/ tests/
	./hack/bootstrap-ginkgo.sh

build: checksync fmt vet compile

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

generate-openapi-spec: build
	echo -e "apiVersion: v1\nclusters:\n- cluster:\n    server: https://127.0.0.1:6443\nkind: Config\n" > .test.kubeconfig
	./cmd/virt-api/virt-api \
		--kubeconfig .test.kubeconfig \
		--dump-api-spec --dump-api-spec-path api/openapi-spec/swagger.json
	rm -f .test.kubeconfig

clean:
	./hack/build-go.sh clean ${WHAT}
	rm ./bin -rf

distclean: clean
	rm -rf vendor/
	rm -f manifest/*.yaml
	rm -f .Gopkg.*.hash

checksync:
	if [ ! -e .Gopkg.toml.hash ] || [ "`${HASH} Gopkg.toml`" != "`cat .Gopkg.toml.hash`" ]; then \
		dep ensure; \
		${HASH} Gopkg.toml > .Gopkg.toml.hash; \
		${HASH} Gopkg.lock > .Gopkg.lock.hash; \
	elif [ ! -e .Gopkg.lock.hash ] || [ "`${HASH} Gopkg.lock`" != "`cat .Gopkg.lock.hash`" ]; then \
		make sync; \
	fi

sync:
	dep ensure -vendor-only
	${HASH} Gopkg.lock > .Gopkg.lock.hash;

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

vagrant-sync-optional:
	./cluster/vagrant/sync_build.sh 'build optional'

vagrant-deploy: vagrant-sync-config vagrant-sync-build
	export KUBECTL="cluster/kubectl.sh --core" && ./cluster/deploy.sh

.release-functest:
	make functest > .release-functest 2>&1

release-announce: .release-functest
	./hack/release-announce.sh $(RELREF) $(PREREF)

.PHONY: build fmt test clean distclean checksync sync docker manifests vet publish vagrant-sync-config vagrant-sync-build vagrant-deploy functest release-announce
