#!/bin/bash -xe

DIR=/usr/local/lib/guestfs
LIBGUEST_APPLIANCE=${DIR}/downloaded
tar -Jxf ${LIBGUEST_APPLIANCE} -C ${DIR}

/bin/bash
