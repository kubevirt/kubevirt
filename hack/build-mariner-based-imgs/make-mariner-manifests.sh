#!/bin/bash

SCRIPT_DIR=$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")" &>/dev/null && pwd)

cd $SCRIPT_DIR/../../

export KUBEVIRT_ONLY_USE_TAGS=true
export FEATURE_GATES="Root,CPUManager,DataVolumes,ExpandDisks,HostDevices,NUMA"

make manifests

cd -
