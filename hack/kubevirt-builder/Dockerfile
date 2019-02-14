FROM kubevirt/builder@sha256:e835f34f53f09e73ba06f15b876ca7182b61cec0a1dcd8313c385450a0a1c736

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
