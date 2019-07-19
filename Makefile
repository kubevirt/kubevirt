QUAY_USERNAME      ?=
QUAY_PASSWORD      ?=
SOURCE_DIRS        = cmd pkg
SOURCES            := $(shell find . -name '*.go' -not -path "*/vendor/*")
IMAGE_REGISTRY     ?= quay.io
IMAGE_TAG          ?= latest
OPERATOR_IMAGE     ?= kubevirt/hyperconverged-cluster-operator
REGISTRY_NAMESPACE ?=

build: $(SOURCES) ## Build binary from source
	go build -i -ldflags="-s -w" -o _out/hyperconverged-cluster-operator ./cmd/manager > /dev/null

clean: ## Clean up the working environment
	@rm -rf _out/

start:
	./hack/deploy.sh

hack-clean: ## Run ./hack/clean.sh
	./hack/clean.sh

docker-build: docker-build-operator docker-build-operator-courier

docker-build-operator:
	docker build -f build/Dockerfile -t $(IMAGE_REGISTRY)/$(OPERATOR_IMAGE):$(IMAGE_TAG) .

docker-build-operator-courier:
	docker build -f hack/operator-courier/Dockerfile -t hco-courier .

docker-push: docker-push-operator

docker-push-operator:
	docker push $(IMAGE_REGISTRY)/$(OPERATOR_IMAGE):$(IMAGE_TAG)

cluster-up:
	./cluster/up.sh

cluster-down:
	./cluster/down.sh

cluster-sync:
	./cluster/sync.sh

cluster-clean:
	CMD="./cluster/kubectl.sh" ./hack/clean.sh

functest:
	./hack/functest.sh

stageRegistry:
	@APP_REGISTRY_NAMESPACE=redhat-operators-stage PACKAGE=kubevirt-hyperconverged ./tools/quay-registry.sh $(QUAY_USERNAME) $(QUAY_PASSWORD)

bundleRegistry:
	REGISTRY_NAMESPACE=$(REGISTRY_NAMESPACE) IMAGE_REGISTRY=$(IMAGE_REGISTRY) ./hack/build-registry-bundle.sh

help: ## Show this help screen
	@echo 'Usage: make <OPTIONS> ... <TARGETS>'
	@echo ''
	@echo 'Available targets are:'
	@echo ''
	@grep -E '^[ a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | \
		awk 'BEGIN {FS = ":.*?## "}; {printf "\033[36m%-20s\033[0m %s\n", $$1, $$2}'
	@echo ''

test-unit:
	./hack/unit-test.sh

test: test-unit

.PHONY: start \
		clean \
		build \
		help \
		hack-clean \
		docker-build \
		docker-build-operator \
		docker-push \
		docker-push-operator \
		cluster-up \
		cluster-down \
		cluster-sync \
		cluster-clean \
		stageRegistry \
		functest
