# Makefile for fake-iommu kernel module
#
# Build:    make
# Clean:    make clean
# Load:     sudo make load    (load fake-iommu BEFORE fake-pci)
# Unload:   sudo make unload  (unload fake-pci BEFORE fake-iommu)

MODULE_NAME := fake-iommu
obj-m := $(MODULE_NAME).o

KDIR ?= /lib/modules/$(shell uname -r)/build
PWD  := $(shell pwd)

all: modules

modules:
	$(MAKE) -C $(KDIR) M=$(PWD) modules

clean:
	$(MAKE) -C $(KDIR) M=$(PWD) clean
	rm -f *.o *.ko *.mod.c *.mod.o *.order *.symvers .*.cmd

install: modules
	$(MAKE) -C $(KDIR) M=$(PWD) modules_install
	depmod -a

load: modules
	@echo "Loading $(MODULE_NAME) module..."
	-rmmod $(MODULE_NAME) 2>/dev/null || true
	insmod $(MODULE_NAME).ko
	@echo "Module loaded. Now load fake-pci."

unload:
	@echo "Unloading $(MODULE_NAME) module..."
	-rmmod $(MODULE_NAME) 2>/dev/null || true
	@echo "Module unloaded."

status:
	@echo "=== Module Status ==="
	@lsmod | grep $(subst -,_,$(MODULE_NAME)) || echo "Module not loaded"
	@echo ""
	@echo "=== /sys/class/iommu ==="
	@ls -la /sys/class/iommu/ 2>/dev/null || echo "  none"

.PHONY: all modules clean install load unload status
