FROM kubevirt/builder@sha256:78b911299cabbdb5a18c414e9a6baa15e63c24c0b0eb43f0756ab36b00722a09

ENV GIMME_GO_VERSION=1.11.5
ENV GOPATH="/go" GOBIN="/usr/bin"

RUN \
    mkdir -p /go && \
    source /etc/profile.d/gimme.sh && \
    go get github.com/mattn/goveralls && \
    go get -u github.com/Masterminds/glide && \
    go get -u github.com/golang/mock/gomock && \
    go get -u github.com/rmohr/mock/mockgen && \
    go get -u github.com/rmohr/go-swagger-utils/swagger-doc && \
    go get -u github.com/onsi/ginkgo/ginkgo

RUN pip install j2cli

COPY rsyncd.conf /etc/rsyncd.conf

ENTRYPOINT [ "/entrypoint.sh" ]
