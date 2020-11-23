#!/usr/bin/env bash

source hack/common.sh

mkdir -p $ARTIFACTS

echo "Run go sec in pkg"
cd $KUBEVIRT_DIR/pkg

if [ -z $GOSEC ]; then
    gosec -sort -quiet -out=${ARTIFACTS}/junit-gosec.xml -exclude-dir=testutils -fmt=junit-xml ./...
else
    echo "Running subset"
    echo $GOSEC
    gosec -include=$GOSEC -sort -quiet -out=${ARTIFACTS}/junit-gosec.xml -exclude-dir=testutils -fmt=junit-xml ./...
fi
