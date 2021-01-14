#!/usr/bin/env bash

source hack/common.sh

mkdir -p $ARTIFACTS

echo "Run go sec in pkg"
cd $KUBEVIRT_DIR/pkg

OUT="${ARTIFACTS}/junit-gosec.xml"
ARGS="-sort -quiet -out=$OUT -exclude-dir=testutils -fmt=junit-xml ./..."
if [ ! -z $GOSEC ]; then
    echo "Running subset"
    echo $GOSEC
    ARGS="-include=$GOSEC $ARGS"
fi
if gosec $ARGS; then
    echo "no errors detected"
    exit 0
else
    echo "report written to" $(echo "$OUT" | sed 's/.*_out/_out/')
    exit 1
fi
