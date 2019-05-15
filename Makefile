QUAY_USERNAME ?=
QUAY_PASSWORD ?=
SOURCE_DIRS   = cmd pkg
SOURCES       := $(shell find . -name '*.go' -not -path "*/vendor/*")

build: $(SOURCES) ## Build binary from source
	go build -i -ldflags="-s -w" -o _out/hyperconverged-cluster-operator ./cmd/manager > /dev/null

clean: ## Clean up the working environment
	@rm -rf _out/

# TODO: maybe we don't need make targets for stuff already in shell scripts?
hack-start: ## Run .hack/deploy.sh
	./hack/deploy.sh

hack-clean: ## Run ./hack/clean.sh
	./hack/clean.sh

stageRegistry:
	@REGISTRY_NAMESPACE=redhat-operators-stage ./hack/quay-registry.sh $(QUAY_USERNAME) $(QUAY_PASSWORD)

help: ## Show this help screen
	@echo 'Usage: make <OPTIONS> ... <TARGETS>'
	@echo ''
	@echo 'Available targets are:'
	@echo ''
	@grep -E '^[ a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | \
		awk 'BEGIN {FS = ":.*?## "}; {printf "\033[36m%-20s\033[0m %s\n", $$1, $$2}'
	@echo ''

.PHONY: build clean hack-start hack-clean
