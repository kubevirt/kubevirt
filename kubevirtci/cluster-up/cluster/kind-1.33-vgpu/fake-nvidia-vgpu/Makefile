# Makefile for fake-nvidia-vgpu kernel module
#
# Build against running kernel:
#   make
#
# Build against specific kernel:
#   make KDIR=/path/to/kernel/build
#
# Install to custom location:
#   make INSTALL_MOD_PATH=/path install
#
# Clean:
#   make clean

MODULE_NAME := fake-nvidia-vgpu
obj-m := $(MODULE_NAME).o

# Kernel build directory
KDIR ?= /lib/modules/$(shell uname -r)/build

# Current directory
PWD := $(shell pwd)

# Default target
all: modules

modules:
	$(MAKE) -C $(KDIR) M=$(PWD) modules

clean:
	$(MAKE) -C $(KDIR) M=$(PWD) clean
	rm -f *.o *.ko *.mod.c *.mod.o *.order *.symvers .*.cmd

install: modules
	$(MAKE) -C $(KDIR) M=$(PWD) modules_install
	depmod -a

# Load module
load: modules
	@echo "Loading $(MODULE_NAME) module..."
	-rmmod $(MODULE_NAME) 2>/dev/null || true
	insmod $(MODULE_NAME).ko
	@echo "Module loaded. Check dmesg for details."
	@echo "mdev bus should now be available at /sys/class/mdev_bus/"

# Unload module
unload:
	@echo "Unloading $(MODULE_NAME) module..."
	-rmmod $(MODULE_NAME) 2>/dev/null || true
	@echo "Module unloaded."

# Show mdev devices
status:
	@echo "=== Module Status ==="
	@lsmod | grep $(MODULE_NAME) || echo "Module not loaded"
	@echo ""
	@echo "=== mdev_bus contents ==="
	@ls -la /sys/class/mdev_bus/ 2>/dev/null || echo "No mdev_bus found"
	@echo ""
	@echo "=== Supported mdev types ==="
	@find /sys/class/mdev_bus/*/mdev_supported_types -maxdepth 1 -type d 2>/dev/null || echo "No mdev types found"

# Create an mdev instance (example)
# Usage: make create-mdev UUID=<uuid>
create-mdev:
ifndef UUID
	$(error UUID is not set. Usage: make create-mdev UUID=<uuid>)
endif
	@echo "Creating mdev instance with UUID $(UUID)..."
	@PARENT=$$(ls /sys/class/mdev_bus/ | head -1); \
	if [ -z "$$PARENT" ]; then echo "No mdev parent found"; exit 1; fi; \
	TYPE_PATH="/sys/class/mdev_bus/$$PARENT/mdev_supported_types/nvidia-222/create"; \
	echo "Writing to $$TYPE_PATH"; \
	echo "$(UUID)" > $$TYPE_PATH
	@echo "mdev instance created"

# Remove an mdev instance
# Usage: make remove-mdev UUID=<uuid>
remove-mdev:
ifndef UUID
	$(error UUID is not set. Usage: make remove-mdev UUID=<uuid>)
endif
	@echo "Removing mdev instance $(UUID)..."
	@echo 1 > /sys/bus/mdev/devices/$(UUID)/remove
	@echo "mdev instance removed"

.PHONY: all modules clean install load unload status create-mdev remove-mdev
