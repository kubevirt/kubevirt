#!/bin/bash

set -euxo pipefail

# The NFS grace period is set to 90 seconds and it stalls the clients
# trying to access the share right after the server start. This may affect
# the tests and lead to timeouts so disable the setting.
sed -i"" \
    -e "s#Grace_Period = 90#Graceless = true#g" \
    /opt/start_nfs.sh

exec /opt/start_nfs.sh
