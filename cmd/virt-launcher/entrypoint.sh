#!/bin/bash
set +e

_term() { 
  echo "caught signal"
  kill -TERM "$virt_launcher_pid" 2>/dev/null
}

trap _term SIGTERM SIGINT SIGQUIT

# FIXME: The plugin framework doesn't appear to (currently) have a means
# to specify device ownership. This needs to be re-visited if that changes
chown :qemu /dev/kvm
chmod 660 /dev/kvm

virt-launcher $@ &
virt_launcher_pid=$!
while true; do
	if ! [ -d /proc/$virt_launcher_pid ]; then
		break;
	fi
	sleep 1
done
# call wait after we know the pid has exited in order
# to get the return code. If we call wait before the pid
# exits, wait will actually return early when we forward
# the trapped signal in _trap(). We don't want that.
wait -n $virt_launcher_pid
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
