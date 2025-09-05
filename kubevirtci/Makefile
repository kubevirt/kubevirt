
export KUBEVIRTCI_TAG ?= $(shell curl -L -Ss https://storage.googleapis.com/kubevirt-prow/release/kubevirt/kubevirtci/latest)

cluster-up:
	./cluster-up/check.sh
	./cluster-up/up.sh

cluster-down:
	./cluster-up/down.sh

.PHONY: \
	cluster-up \
	cluster-down \
	bump
