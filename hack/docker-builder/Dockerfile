FROM fedora:28

ENV LIBVIRT_VERSION 4.2.0

COPY fedora-virt-preview.repo /etc/yum.repos.d/fedora-virt-preview.repo

RUN dnf -y install libvirt-devel-${LIBVIRT_VERSION} make git mercurial sudo gcc findutils gradle rsync-daemon rsync qemu-img protobuf-compiler && \
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
    go get -u golang.org/x/tools/cmd/goimports && \
    git clone https://github.com/mvdan/sh.git $GOPATH/src/mvdan.cc/sh && \
    cd /go/src/mvdan.cc/sh/cmd/shfmt && \
    git checkout v2.5.0 && \
    go get mvdan.cc/sh/cmd/shfmt && \
    go install && \
    go get -u github.com/golang/mock/gomock && \
    go get -u github.com/rmohr/mock/mockgen && \
    go get -u github.com/rmohr/go-swagger-utils/swagger-doc && \
    go get -u github.com/onsi/ginkgo/ginkgo && \
    go get -u -d k8s.io/code-generator/cmd/deepcopy-gen && \
    go get -u -d k8s.io/code-generator/cmd/defaulter-gen && \
    go get -u -d k8s.io/code-generator/cmd/openapi-gen && \
    cd /go/src/k8s.io/code-generator/cmd/deepcopy-gen && \
    git checkout kubernetes-1.11.2 && \
    go install && \
    cd /go/src/k8s.io/code-generator/cmd/defaulter-gen && \
    git checkout kubernetes-1.11.2 && \
    go install && \
    cd /go/src/k8s.io/code-generator/cmd/openapi-gen && \
    git checkout kubernetes-1.11.2 && \
    go install && \
    go get -u -d github.com/golang/protobuf/protoc-gen-go && \
    cd /go/src/github.com/golang/protobuf/protoc-gen-go && \
    git checkout 1643683e1b54a9e88ad26d98f81400c8c9d9f4f9 && \
    go install

RUN pip install j2cli

ADD entrypoint.sh /entrypoint.sh

ENTRYPOINT [ "/entrypoint.sh" ]
