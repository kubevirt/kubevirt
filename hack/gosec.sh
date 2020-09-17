source hack/common.sh
export ARTIFACTS=$KUBEVIRT_DIR/${ARTIFACTS:-_out/artifacts}
if ${GENERATE:-"false"} == "true"; then

    mkdir -p $ARTIFACTS

    echo "Run go sec in pkg"
    cd $KUBEVIRT_DIR/pkg

    # -confidence=high -severity=high <- for filtering
    gosec -sort -quiet -out=${ARTIFACTS}/junit-gosec.xml -no-fail -exclude-dir=testutils -fmt=junit-xml ./...

    cp ${ARTIFACTS}/junit-gosec.xml ${KUBEVIRT_DIR}/tools/gosec
else
    cd ${KUBEVIRT_DIR}/tools/gosec
    git --no-pager diff "junit-gosec.xml"
fi
