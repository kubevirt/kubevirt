# Makefile

# Define variables
BINARY_NAME=libvirt_helper
PKG=./pkg/virt-launcher/virtwrap/util

# Go related variables
GO=go
GOFMT=gofmt
GOCMD=$(GO)
GOBUILD=$(GO) build
GOTEST=$(GO) test
GOINSTALL=$(GO) install
GOMOD=go.mod

# Bazel related variables
BAZEL=bazel

# Output directories
BIN_DIR=bin
BUILD_DIR=build

.PHONY: all build test generate clean

all: build

# Build the Go binary
build:
	@echo "Building..."
	mkdir -p $(BIN_DIR)
	$(GOBUILD) -o $(BIN_DIR)/$(BINARY_NAME) $(PKG)

# Run tests
test:
	@echo "Running tests..."
	$(GOTEST) -v ./...

# Generate code (e.g., mock generation, protobufs, etc.)
generate:
	@echo "Generating code..."
	# Add your code generation commands here
	# Example for Mock generation using gomock:
	mockgen -source=$(PKG)/libvirt_helper.go -destination=$(PKG)/mock_libvirt_helper.go -package=util
	# If using Bazel for generation:
	# $(BAZEL) run //:generate

# Format the code
fmt:
	@echo "Formatting code..."
	$(GOFMT) -w $(PKG)

# Clean build artifacts
clean:
	@echo "Cleaning..."
	rm -rf $(BIN_DIR)
	rm -rf $(BUILD_DIR)
	# Add other clean-up commands as needed

# Bazel specific targets (optional)
bazel-build:
	@echo "Building with Bazel..."
	$(BAZEL) build //...

bazel-test:
	@echo "Testing with Bazel..."
	$(BAZEL) test //...

bazel-clean:
	@echo "Cleaning Bazel build artifacts..."
	$(BAZEL) clean
