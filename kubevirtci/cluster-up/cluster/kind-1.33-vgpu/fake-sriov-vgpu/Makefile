# Makefile for fake-sriov-vgpu kernel module
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

MODULE_NAME := fake-sriov-vgpu
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

# Unload module
unload:
	@echo "Unloading $(MODULE_NAME) module..."
	-rmmod $(MODULE_NAME) 2>/dev/null || true
	@echo "Module unloaded."

# Show status
status:
	@echo "=== Module Status ==="
	@lsmod | grep $(MODULE_NAME) || echo "Module not loaded"
	@echo ""
	@echo "=== Control Interface ==="
	@ls -la /sys/class/fake-sriov-vgpu/control/ 2>/dev/null || echo "Not available"
	@echo ""
	@echo "=== Created VFs ==="
	@cat /sys/class/fake-sriov-vgpu/control/list 2>/dev/null || echo "None"

.PHONY: all modules clean install load unload status
