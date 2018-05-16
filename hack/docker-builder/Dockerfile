FROM fedora:27

ENV LIBVIRT_VERSION 3.7.0

RUN dnf -y install libvirt-devel-${LIBVIRT_VERSION} make git mercurial sudo gcc findutils gradle rsync-daemon rsync qemu-img && \
    dnf -y clean all

ENV GIMME_GO_VERSION=1.10

RUN mkdir -p /gimme && curl -sL https://raw.githubusercontent.com/travis-ci/gimme/master/gimme | HOME=/gimme bash >> /etc/profile.d/gimme.sh

ENV GOPATH="/go" GOBIN="/usr/bin"

ADD rsyncd.conf /etc/rsyncd.conf

RUN \
    mkdir -p /go && \
    source /etc/profile.d/gimme.sh && \
    go get github.com/mattn/goveralls && \
    go get -u github.com/Masterminds/glide && \
    go get golang.org/x/tools/cmd/goimports && \
    go get -u mvdan.cc/sh/cmd/shfmt && \
    cd /go/src/mvdan.cc/sh/cmd/shfmt && \
    git checkout v2.3.0 && \
    go install && \
    go get -u github.com/golang/mock/gomock && \
    go get -u github.com/rmohr/mock/mockgen && \
    go get -u github.com/rmohr/go-swagger-utils/swagger-doc && \
    go get -u github.com/onsi/ginkgo/ginkgo && \
    go get -u -d k8s.io/code-generator/cmd/deepcopy-gen && \
    go get -u -d k8s.io/code-generator/cmd/defaulter-gen && \
    go get -u -d k8s.io/code-generator/cmd/openapi-gen && \
    cd /go/src/k8s.io/code-generator/cmd/deepcopy-gen && \
    git checkout release-1.9 && \
    go install && \
    cd /go/src/k8s.io/code-generator/cmd/defaulter-gen && \
    git checkout release-1.9 && \
    go install && \
    cd /go/src/k8s.io/code-generator/cmd/openapi-gen && \
    git checkout release-1.9 && \
    go install

ADD entrypoint.sh /entrypoint.sh

ENTRYPOINT [ "/entrypoint.sh" ]
