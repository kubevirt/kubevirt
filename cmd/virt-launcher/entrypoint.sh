#!/bin/bash
set +e

# HACK
# Try to create /dev/kvm if not present
if [ ! -e /dev/kvm ]; then
   mknod /dev/kvm c 10 $(grep '\<kvm\>' /proc/misc | cut -f 1 -d' ')
fi

chown :qemu /dev/kvm
chmod 660 /dev/kvm


# Cockpit/OCP hack to all shoing the vm terminal
mv /usr/bin/sh /usr/bin/sh.orig
mv /sh.sh /usr/bin/sh
chmod +x /usr/bin/sh

./virt-launcher $@
rc=$?

echo "virt-launcher exited with code $rc"

# if the qemu pid outlives virt-launcher because virt-launcher
# segfaulted/panicked/etc... then make sure we perform a sane
# shutdown of the qemu process before exitting. 
qemu_pid=$(pgrep -u qemu)
if [ -n "$qemu_pid" ]; then
	echo "qemu pid outlived virt-launcher process. Sending SIGTERM"
	kill -SIGTERM $qemu_pid

	# give the pid 10 seconds to exit. 
	for x in $(seq 1 10); do
		if ! [ -d /proc/$qemu_pid ]; then
			echo "qemu pid [$qemu_pid] exited after after SIGTERM"
			exit $rc
		fi
		echo "waiting for qemu pid [$qemu_pid] to exit"
		sleep 1
	done

	# if we got here, the pid never exitted gracefully.
	echo "timed out waiting for qemu pid [$qemu_pid] to exit"
fi

exit $rc
