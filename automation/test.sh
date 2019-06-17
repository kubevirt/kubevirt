#!/bin/bash -xe

make cluster-down
make cluster-up
make cluster-sync
make functest
make docker-push
