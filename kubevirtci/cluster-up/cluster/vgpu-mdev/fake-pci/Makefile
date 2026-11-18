# Makefile for fake-pci kernel module
#
# Build against running kernel:
#   make
#
# Build against specific kernel:
#   make KDIR=/path/to/kernel/build
#
# Install:
#   sudo make install
#
# Clean:
#   make clean

MODULE_NAME := fake-pci
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
	@echo "Module loaded. Check dmesg for details."
	@echo "Look for devices under /sys/bus/pci/devices/ (domain default 0xfaca)."

unload:
	@echo "Unloading $(MODULE_NAME) module..."
	@module_name=$$(echo "$(MODULE_NAME)" | tr '-' '_'); \
	if lsmod | grep -q "^$$module_name "; then \
		rmmod $(MODULE_NAME) || exit $$?; \
		echo "Module unloaded."; \
	else \
		echo "Module not loaded."; \
	fi

status:
	@echo "=== Module Status ==="
	@lsmod | grep $(subst -,_,$(MODULE_NAME)) || echo "Module not loaded"
	@echo ""
	@echo "=== Fake PCI devices ==="
	@vendor_id=$$(cat /sys/module/$(subst -,_,$(MODULE_NAME))/parameters/vendor_id 2>/dev/null); \
	for d in /sys/bus/pci/devices/*; do \
	  v=$$(cat $$d/vendor 2>/dev/null); \
	  if [ -n "$$vendor_id" ] && [ "$$v" = "$$(printf '0x%04x' "$$vendor_id")" ]; then \
	    echo "  $$(basename $$d) vendor=$$v device=$$(cat $$d/device)"; \
	  fi; \
	done

.PHONY: all modules clean install load unload status
