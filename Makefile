QUAY_USERNAME      ?=
QUAY_PASSWORD      ?=
SOURCE_DIRS        = cmd pkg
SOURCES            := $(shell find . -name '*.go' -not -path "*/vendor/*")
IMAGE_REGISTRY     ?= quay.io
IMAGE_TAG          ?= latest
OPERATOR_IMAGE     ?= kubevirt/hyperconverged-cluster-operator
REGISTRY_NAMESPACE ?=

build: $(SOURCES) ## Build binary from source
	go build -i -ldflags="-s -w" -o _out/hyperconverged-cluster-operator ./cmd/hyperconverged-cluster-operator
	go build -i -ldflags="-s -w" -o _out/csv-merger tools/csv-merger/csv-merger.go

install:
	go install ./cmd/...

clean: ## Clean up the working environment
	@rm -rf _out/

start:
	./hack/deploy.sh

quay-token:
	@./tools/token.sh $(QUAY_USERNAME) $(QUAY_PASSWORD)

bundle-push: container-build-operator-courier
	@QUAY_USERNAME=$(QUAY_USERNAME) QUAY_PASSWORD=$(QUAY_PASSWORD) ./tools/operator-courier/push.sh

hack-clean: ## Run ./hack/clean.sh
	./hack/clean.sh

container-build: container-build-operator container-build-operator-courier

container-build-operator:
	docker build -f build/Dockerfile -t $(IMAGE_REGISTRY)/$(OPERATOR_IMAGE):$(IMAGE_TAG) .

container-build-operator-courier:
	docker build -f tools/operator-courier/Dockerfile -t hco-courier .

container-push: container-push-operator

container-push-operator:
	docker push $(IMAGE_REGISTRY)/$(OPERATOR_IMAGE):$(IMAGE_TAG)

cluster-up:
	./cluster-up/up.sh

cluster-down:
	./cluster-up/down.sh

cluster-sync:
	./cluster-up/sync.sh

cluster-clean:
	CMD="./cluster-up/kubectl.sh" ./hack/clean.sh

functest:
	./hack/functest.sh

stageRegistry:
	@APP_REGISTRY_NAMESPACE=redhat-operators-stage PACKAGE=kubevirt-hyperconverged ./tools/quay-registry.sh $(QUAY_USERNAME) $(QUAY_PASSWORD)

bundleRegistry:
	REGISTRY_NAMESPACE=$(REGISTRY_NAMESPACE) IMAGE_REGISTRY=$(IMAGE_REGISTRY) ./hack/build-registry-bundle.sh

build-push-all: container-build-operator container-push-operator container-build-operator-courier bundle-push

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
		container-build \
		container-build-operator \
		container-push \
		container-push-operator \
		container-build-operator-courier \
		cluster-up \
		cluster-down \
		cluster-sync \
		cluster-clean \
		stageRegistry \
		functest \
		quay-token \
		bundle-push \
		build-push-all
