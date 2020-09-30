#!/usr/bin/env bash

source hack/common.sh
export ARTIFACTS=${ARTIFACTS:-$KUBEVIRT_DIR/_out/artifacts}

mkdir -p $ARTIFACTS

echo "Run go sec in pkg"
cd $KUBEVIRT_DIR/pkg

gosec -sort -quiet -out=${ARTIFACTS}/junit-gosec.xml -exclude-dir=testutils -fmt=junit-xml ./...
