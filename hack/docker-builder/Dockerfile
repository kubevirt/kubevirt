FROM fedora:28

ENV LIBVIRT_VERSION 4.10.0

RUN dnf install -y dnf-plugins-core && \
    dnf copr enable -y @virtmaint-sig/virt-preview && \
    dnf -y install libvirt-devel-${LIBVIRT_VERSION} make git mercurial sudo gcc findutils gradle rsync-daemon rsync qemu-img protobuf-compiler && \
    dnf -y clean all

ENV GIMME_GO_VERSION=1.11.2

RUN mkdir -p /gimme && curl -sL https://raw.githubusercontent.com/travis-ci/gimme/master/gimme | HOME=/gimme bash >> /etc/profile.d/gimme.sh

ENV GOPATH="/go" GOBIN="/usr/bin"

RUN \
    mkdir -p /go && \
    source /etc/profile.d/gimme.sh && \
    go get github.com/mattn/goveralls && \
    go get -u github.com/Masterminds/glide && \
    go get -d golang.org/x/tools/cmd/goimports && \
    cd /go/src/golang.org/x/tools/cmd/goimports && \
    git checkout release-branch.go1.11 && \
    go install && \
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
    go get -u -d k8s.io/kube-openapi/cmd/openapi-gen && \
    cd /go/src/k8s.io/code-generator/cmd/deepcopy-gen && \
    git checkout kubernetes-1.11.2 && \
    go install && \
    cd /go/src/k8s.io/code-generator/cmd/defaulter-gen && \
    git checkout kubernetes-1.11.2 && \
    go install && \
    cd /go/src/k8s.io/kube-openapi/cmd/openapi-gen && \
    git checkout c59034cc13d587f5ef4e85ca0ade0c1866ae8e1d && \
    go install && \
    go get -u -d github.com/golang/protobuf/protoc-gen-go && \
    cd /go/src/github.com/golang/protobuf/protoc-gen-go && \
    git checkout 1643683e1b54a9e88ad26d98f81400c8c9d9f4f9 && \
    go install

RUN pip install j2cli

ENTRYPOINT [ "/entrypoint.sh" ]

ADD rsyncd.conf /etc/rsyncd.conf

ADD entrypoint.sh /entrypoint.sh
