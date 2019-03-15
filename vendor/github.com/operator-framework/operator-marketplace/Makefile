# OpenShift Marketplace - Build and Test

SHELL := /bin/bash
PKG := github.com/operator-framework/operator-marketplace/pkg
MOCKS_DIR := ./pkg/mocks
CONTROLLER_RUNTIME_PKG := sigs.k8s.io/controller-runtime/pkg
OPERATORSOURCE_MOCK_PKG := operatorsource_mocks

# If the GOBIN environment variable is set, 'go install' will install the 
# commands to the directory it names, otherwise it will default of $GOPATH/bin.
# GOBIN must be an absolute path.
ifeq ($(GOBIN),)
mockgen := $(GOPATH)/bin/mockgen
else
mockgen := $(GOBIN)/mockgen
endif

all: osbs-build

osbs-build:
	# hack/build.sh
	./build/build.sh

unit: generate-mocks unit-test

unit-test:
	go test -v ./pkg/...

generate-mocks:
	# Build mockgen from the same version used by gomock. This ensures that 
	# gomock and mockgen are never out of sync.
	go install ./vendor/github.com/golang/mock/mockgen
	
	@echo making sure directory for mocks exists
	mkdir -p $(MOCKS_DIR)

	$(mockgen) -destination=$(MOCKS_DIR)/$(OPERATORSOURCE_MOCK_PKG)/mock_datastore.go -package=$(OPERATORSOURCE_MOCK_PKG) -mock_names=Reader=DatastoreReader,Writer=DatastoreWriter $(PKG)/datastore Reader,Writer
	$(mockgen) -destination=$(MOCKS_DIR)/$(OPERATORSOURCE_MOCK_PKG)/mock_phase_reconciler.go -package=$(OPERATORSOURCE_MOCK_PKG) -mock_names=Reconciler=PhaseReconciler $(PKG)/operatorsource Reconciler
	$(mockgen) -destination=$(MOCKS_DIR)/$(OPERATORSOURCE_MOCK_PKG)/mock_phase_tansitioner.go -package=$(OPERATORSOURCE_MOCK_PKG) -mock_names=Transitioner=PhaseTransitioner $(PKG)/phase Transitioner
	$(mockgen) -destination=$(MOCKS_DIR)/$(OPERATORSOURCE_MOCK_PKG)/mock_kubeclient.go -package=$(OPERATORSOURCE_MOCK_PKG) -mock_names=Client=KubeClient $(CONTROLLER_RUNTIME_PKG)/client Client
	$(mockgen) -destination=$(MOCKS_DIR)/$(OPERATORSOURCE_MOCK_PKG)/mock_phase_reconciler_strategy.go -package=$(OPERATORSOURCE_MOCK_PKG) $(PKG)/operatorsource PhaseReconcilerFactory
	$(mockgen) -destination=$(MOCKS_DIR)/$(OPERATORSOURCE_MOCK_PKG)/mock_appregistry.go -package=$(OPERATORSOURCE_MOCK_PKG) -mock_names=ClientFactory=AppRegistryClientFactory,Client=AppRegistryClient $(PKG)/appregistry ClientFactory,Client
	$(mockgen) -destination=$(MOCKS_DIR)/$(OPERATORSOURCE_MOCK_PKG)/mock_syncer.go -package=$(OPERATORSOURCE_MOCK_PKG) -mock_names=PackageRefreshNotificationSender=SyncerPackageRefreshNotificationSender $(PKG)/operatorsource PackageRefreshNotificationSender

clean-mocks:
	@echo cleaning mock folder
	rm -rf $(MOCKS_DIR)

clean: clean-mocks

e2e-test:
	./scripts/e2e-tests.sh

e2e-job:
	./scripts/run-e2e-job.sh
