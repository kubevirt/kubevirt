#!/bin/bash
set +e

_term() { 
  echo "caught signal"
  kill -TERM "$virt_launcher_pid" 2>/dev/null
}

trap _term SIGTERM SIGINT SIGQUIT

# HACK
# Try to create /dev/kvm if not present
# /dev/kvm will be present if DevicePlugins were used.
# otherwise assume the proper privileges are in place to allow this
if [ ! -e /dev/kvm ]; then
   mknod /dev/kvm c 10 $(grep '\<kvm\>' /proc/misc | cut -f 1 -d' ')
fi

# FIXME: The plugin framework doesn't appear to (currently) have a means
# to specify device ownership. This needs to be re-visited if that changes
chown :qemu /dev/kvm
chmod 660 /dev/kvm

# HACK
# Try to create /dev/tun if not present
# /dev/tun will be present if DevicePlugins were used.
# otherwise assume the proper privileges are in place to allow this
if [ ! -e /dev/tun ]; then
   mknod /dev/tun c 10 $(grep '\<tun\>' /proc/misc | cut -f 1 -d' ')
fi

# Cockpit/OCP hack to all shoing the vm terminal
mv /usr/bin/sh /usr/bin/sh.orig
mv /sh.sh /usr/bin/sh
chmod +x /usr/bin/sh

./virt-launcher $@ &
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
