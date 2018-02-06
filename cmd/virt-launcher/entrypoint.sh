#!/bin/bash
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
			echo "qemu pid exited after after SIGTERM"
			exit $rc
		fi
		echo "waiting for qemu pid to exit"
		sleep 1
	done

	# if we got here, the pid never exitted gracefully.
	echo "timed out waiting for qemu pid to exit"
fi

exit $rc
