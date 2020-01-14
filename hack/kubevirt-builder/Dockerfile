FROM kubevirt/builder@sha256:6eb3245a6f3a17e37c08d737b91fb7d4c8460cf0407b1bcb532e28327dfd1933

ENV GIMME_GO_VERSION=1.12.8
ENV GOPATH="/go" GOBIN="/usr/bin"

RUN \
    mkdir -p /go && \
    source /etc/profile.d/gimme.sh && \
    go get github.com/mattn/goveralls && \
    go get -u github.com/golang/mock/gomock && \
    go get -u github.com/rmohr/mock/mockgen && \
    go get -u github.com/rmohr/go-swagger-utils/swagger-doc && \
    go get -u github.com/onsi/ginkgo/ginkgo

RUN pip install j2cli && pip3 install operator-courier==1.3.0

COPY rsyncd.conf /etc/rsyncd.conf

ENTRYPOINT [ "/entrypoint.sh" ]
