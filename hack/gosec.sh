#!/usr/bin/env bash

source hack/common.sh

mkdir -p $ARTIFACTS

echo "Run go sec in pkg"
cd $KUBEVIRT_DIR/pkg

ARGS="-sort -quiet -out=${ARTIFACTS}/junit-gosec.xml -exclude-dir=testutils -fmt=junit-xml ./..."
if [ ! -z $GOSEC ]; then
    echo "Running subset"
    echo $GOSEC
    ARGS="-include=$GOSEC $ARGS"
fi
gosec $ARGS
