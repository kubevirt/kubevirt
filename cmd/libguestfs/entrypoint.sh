#!/bin/bash -xe

LIBGUESTFS_PATH=${LIBGUESTFS_PATH:=/tmp/guestfs}
LIBGUEST_APPLIANCE=/usr/local/lib/guestfs/downloaded
mkdir -p ${LIBGUESTFS_PATH}
tar -Jxf ${LIBGUEST_APPLIANCE} -C ${LIBGUESTFS_PATH} --strip-components=1

touch ${LIBGUESTFS_PATH}/done

/bin/bash
