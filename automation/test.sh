#!/bin/bash -xe

make cluster-down
make cluster-up
make cluster-sync
CMD='cluster/kubectl.sh' make functest
