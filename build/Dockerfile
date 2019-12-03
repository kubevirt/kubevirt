FROM golang:1.11 AS builder
ENV GOPATH=/go
WORKDIR /go/src/github.com/kubevirt/hyperconverged-cluster-operator/
COPY . .
RUN make build

FROM registry.access.redhat.com/ubi8/ubi-minimal
ENV OPERATOR=/usr/local/bin/hyperconverged-cluster-operator \
    CSV_MERGER=/usr/local/bin/csv-merger \
    USER_UID=1001 \
    USER_NAME=hyperconverged-cluster-operator \
    CSV_TEMPLATE=/var/hco-csv-template.yaml.in 


COPY --from=builder /go/src/github.com/kubevirt/hyperconverged-cluster-operator/build/bin/ /usr/local/bin
RUN /usr/local/bin/user_setup
COPY --from=builder /go/src/github.com/kubevirt/hyperconverged-cluster-operator/_out/hyperconverged-cluster-operator $OPERATOR
COPY --from=builder /go/src/github.com/kubevirt/hyperconverged-cluster-operator/_out/csv-merger $CSV_MERGER
COPY --from=builder /go/src/github.com/kubevirt/hyperconverged-cluster-operator/templates/olm-catalog/kubevirt-hyperconverged/VERSION/kubevirt-hyperconverged-operator.VERSION.clusterserviceversion_merger.yaml.in $CSV_TEMPLATE
ENTRYPOINT ["/usr/local/bin/entrypoint"]
USER ${USER_UID}
