REPO_ROOT=$(abspath $(dir $(lastword $(MAKEFILE_LIST))))

# Basenames
CONTROLLER=controller
IMPORTER=importer
F_TEST=datastream-test

# Binary Path
BIN=$(REPO_ROOT)/bin
CONTROLLER_BIN=import-controller
IMPORTER_BIN=importer
F_TEST_BIN=$(BIN)/$(F_TEST)

# Source dirs
CMD_DIR=$(REPO_ROOT)/cmd
PKG_DIR=$(REPO_ROOT)/pkg
CONTROLLER_CMD=$(CMD_DIR)/$(CONTROLLER)
IMPORTER_CMD=$(CMD_DIR)/$(IMPORTER)
LIB_PKG_DIR=$(PKG_DIR)/lib
LIB_SIZE_DIR=$(LIB_PKG_DIR)/size
F_TEST_DIR=$(REPO_ROOT)/test/functional/importer
F_IMG_DIR=$(REPO_ROOT)/test/images/tinyCore.iso
BUILD_CMD=GOOS=$(GOOS) GOARCH=$(ARCH) CGO_ENABLED=$(CGO_ENABLED) go build -a -ldflags $(LDFLAGS)
DOCKER_BUILD_CMD=docker run -it --rm -v $(REPO_ROOT):$(WORK_DIR):Z -w $(WORK_DIR) -e GOOS=$(GOOS) -e GOARCH=$(ARCH) -e CGO_ENABLED=$(CGO_ENABLED) $(BUILD_IMAGE) go build

# Build Dirs
BUILD_DIR=$(REPO_ROOT)/build
CONTROLLER_BUILD=$(BUILD_DIR)/$(CONTROLLER)
IMPORTER_BUILD=$(BUILD_DIR)/$(IMPORTER)

# DOCKER TAG VARS
DEV_REGISTRY=jcoperh
RELEASE_REGISTRY=kubevirt
RELEASE_TAG=$(shell git describe --tags --abbrev=0 HEAD)
CTRL_IMG_NAME=cdi-$(CONTROLLER)
IMPT_IMG_NAME=cdi-$(IMPORTER)
GIT_USER=$(shell git config --get user.email | sed 's/@.*//')
TAG=$(GIT_USER)-latest

# Preflight Check Defaults
USE_DOCKER=1
RUNNING_DOCKER=$(shell docker ps > /dev/null 2>&1; echo $$?)

.PHONY: controller importer controller-bin importer-bin controller-image importer-image push-controller push-controller-release push-importer-release push-importer lib clean test
all: clean test controller importer lib
controller: controller-bin controller-image
importer: importer-bin importer-image
push: push-importer push-controller
test: functional-test unit-test
functional-test: func-test-bin func-test-image func-test-run
lib: lib-size

BUILD_IMAGE=golang:1.10.2
WORK_DIR=/go/src/kubevirt.io/containerized-data-importer
GOOS?=linux
ARCH?=amd64
CGO_ENABLED=0
LDFLAGS='-extldflags "-static"'

# Compile controller binary
controller-bin:
	@echo '********'
	@echo 'Compiling controller binary'
	@if [ '$(RUNNING_DOCKER)' -eq 0 ] && [ $(USE_DOCKER) -eq 1 ]; then \
		echo 'building with docker'; \
		$(DOCKER_BUILD_CMD) -o $(WORK_DIR)/bin/$(CONTROLLER_BIN) $(WORK_DIR)/cmd/controller/controller.go; \
	else \
		echo 'building without docker'; \
		$(BUILD_CMD) -o $(REPO_ROOT)/bin/$(CONTROLLER_BIN) $(CONTROLLER_CMD)/controller.go; \
	fi

# Compile importer binary
importer-bin:
	@echo '********'
	@echo 'Compiling importer binary'
	@if [ $(RUNNING_DOCKER) -eq 0 ] && [ $(USE_DOCKER) -eq 1 ]; then \
		echo 'building with docker'; \
		$(DOCKER_BUILD_CMD) -o $(WORK_DIR)/bin/$(IMPORTER_BIN) $(WORK_DIR)/cmd/importer/importer.go; \
	else \
		echo 'building without docker'; \
		$(BUILD_CMD) -o $(REPO_ROOT)/bin/$(IMPORTER_BIN) $(IMPORTER_CMD)/importer.go; \
	fi

# Compile datastream functional test binary
func-test-bin:
	@echo '********'
	@echo 'Compiling functional test binary'
	-rm -f $(F_TEST_BIN)
	GOOS=$(GOOS) GOARCH=$(ARCH) CGO_ENABLED=$(CGO_ENABLED) go test -a -c -ldflags $(LDFLAGS) -o $(F_TEST_BIN) $(F_TEST_DIR)/*.go

# build the controller image
controller-image: $(CONTROLLER_BUILD)/Dockerfile
ifeq ($(RUNNING_DOCKER), 1)
	@echo 'Docker daemon not running, skipping image build.'
else ifeq ($(USE_DOCKER), 1)
	@echo '********'
	@echo 'Building controller image'
	$(eval TEMP_BUILD_DIR=$(CONTROLLER_BUILD)/tmp)
	mkdir -p $(TEMP_BUILD_DIR)
	cp $(BIN)/$(CONTROLLER_BIN) $(TEMP_BUILD_DIR)
	cp $(CONTROLLER_BUILD)/Dockerfile $(TEMP_BUILD_DIR)
	docker build -t $(CTRL_IMG_NAME) $(TEMP_BUILD_DIR)
	-rm -rf $(TEMP_BUILD_DIR)
endif

# build the importer image
importer-image: $(IMPORTER_BUILD)/Dockerfile
ifeq ($(RUNNING_DOCKER), 1)
	@echo 'Docker daemon not running, skipping image build.'
else ifeq ($(USE_DOCKER), 1)
	@echo '********'
	@echo 'Building importer image'
	$(eval TEMP_BUILD_DIR=$(IMPORTER_BUILD)/tmp)
	mkdir -p $(TEMP_BUILD_DIR)
	cp $(BIN)/$(IMPORTER_BIN) $(TEMP_BUILD_DIR)
	cp $(IMPORTER_BUILD)/Dockerfile $(TEMP_BUILD_DIR)
	docker build --build-arg entrypoint=$(IMPORTER) -t $(IMPT_IMG_NAME) $(TEMP_BUILD_DIR)
	-rm -rf $(TEMP_BUILD_DIR)
endif

# build the functional test image.  The importer image is used to provide consistency between test
# and run environments.
func-test-image: $(IMPORTER_BUILD)/Dockerfile
	@echo '********'
	@echo 'Building functional test image'
	$(eval TEMP_BUILD_DIR=$(IMPORTER_BUILD)/tmp)
	mkdir -p $(TEMP_BUILD_DIR)
	cp $(F_TEST_BIN) $(TEMP_BUILD_DIR)
	cp $(F_IMG_DIR) $(TEMP_BUILD_DIR)
	cp $(IMPORTER_BUILD)/Dockerfile $(TEMP_BUILD_DIR)
	docker build --build-arg entrypoint=$(F_TEST) --build-arg runArgs='-ginkgo.v' --build-arg depFile=tinyCore.iso -t $(F_TEST) $(TEMP_BUILD_DIR)
	-rm -rf $(TEMP_BUILD_DIR)


func-test-run:
	@echo '********'
	@echo 'Running functional tests'
	docker ps -qa && docker run --rm $(F_TEST) || echo 'Docker service not detected, skipping functional tests'

push-controller:
	@echo '********'
	@echo 'Pushing controller image'
	docker tag $(CTRL_IMG_NAME) $(DEV_REGISTRY)/$(CTRL_IMG_NAME):$(TAG)
	docker push $(DEV_REGISTRY)/$(CTRL_IMG_NAME):$(TAG)

push-importer:
	@echo '********'
	@echo 'Pushing importer image'
	docker tag $(IMPT_IMG_NAME) $(DEV_REGISTRY)/$(IMPT_IMG_NAME):$(TAG)
	docker push $(DEV_REGISTRY)/$(IMPT_IMG_NAME):$(TAG)

unit-test:
	@echo '********'
	@echo 'Running unit tests'
	CGO_ENABLED=$(CGO_ENABLED) go test -v -tags=unit_test ./...

lib-size:
	# compile size "library" package consumed by external repos
	@if [ $(RUNNING_DOCKER) -eq 1 ] && [ $(USE_DOCKER) -eq 1 ]; then \
		echo 'building with docker'; \
		$(DOCKER_BUILD_CMD) -o /tmp/size $(WORK_DIR)/pkg/lib/size/size.go; \
	else \
		echo 'building without docker'; \
		$(BUILD_CMD) -o /tmp/size $(REPO_ROOT)/pkg/lib/size/size.go; \
	fi

clean:
	@echo '********'
	@echo 'Cleaning build artifacts'
	-rm -rf $(BIN)/*
	-rm -rf $(CONTROLLER_BUILD)/tmp
	-rm -rf $(IMPORTER_BUILD)/tmp

# push cdi-importer and cdi-controller images to kubevirt repo for general use. Intended to release stable image built from master branch.
release:
	@echo '********'
	@echo 'Releasing CDI images'
	docker tag $(IMPT_IMG_NAME) $(RELEASE_REGISTRY)/$(IMPT_IMG_NAME):$(RELEASE_TAG)
	docker push $(RELEASE_REGISTRY)/$(IMPT_IMG_NAME):$(RELEASE_TAG)
	docker tag $(CTRL_IMG_NAME) $(RELEASE_REGISTRY)/$(CTRL_IMG_NAME):$(RELEASE_TAG)
	docker push $(RELEASE_REGISTRY)/$(CTRL_IMG_NAME):$(RELEASE_TAG)

set-version:
	@echo '********'
	@[ -n "$(VERSION)" ] || (echo "Must provide VERSION=<version> on command line" && exit 1)
	@echo 'Setting new version.'
	$(REPO_ROOT)/hack/version/set-version.sh $(VERSION)
	@echo "Version change complete (=> $(VERSION))"
	@echo "To finalize this update, push to these changes to the upstream reposoitory with"
	@echo "    $ make release"
	@echo "    $ git push <upstream> master &&  git push <upstream> --tags"
	@echo "To undo local changes without pushing, rollback to the previous commit"
	@echo "    $ git reset HEAD~1"

.PHONY: build-and-deploy
build-and-deploy: importer controller deploy-controller patch-controller

.PHONY: deploy-controller
deploy-controller: $(REPO_ROOT)/manifests/controller/cdi-controller-deployment.yaml
	sed -E -e 's#kubevirt/cdi-controller.*#cdi-controller#g' -e 's#imagePullPolicy:.*#imagePullPolicy: Never#g' $(REPO_ROOT)/manifests/controller/cdi-controller-deployment.yaml | kubectl apply -f -

.PHONY: patch-controller
patch-controller:
	kubectl patch deployment cdi-deployment --type='json' -p='[{"op": "add", "path": "/spec/template/spec/containers/0/env", "value": [{"name": "IMPORTER_IMAGE", "value": "cdi-importer"}]}]'

.PHONY: my-golden-pvc.yaml
my-golden-pvc.yaml: manifests/example/golden-pvc.yaml
	sed "s,endpoint:.*,endpoint: \"$(URI)\"," $< > $@
