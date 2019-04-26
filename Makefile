QUAY_USERNAME ?=
QUAY_PASSWORD ?=

start:
	./hack/deploy.sh

clean:
	./hack/clean.sh

stageRegistry:
	@REGISTRY_NAMESPACE=redhat-operators-stage ./hack/quay-registry.sh $(QUAY_USERNAME) $(QUAY_PASSWORD)

.PHONY: start clean
