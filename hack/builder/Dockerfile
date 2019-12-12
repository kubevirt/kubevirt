FROM fedora:30

ENV LIBVIRT_VERSION 5.1.0

# Install packages
RUN dnf install -y dnf-plugins-core && \
    dnf copr enable -y @virtmaint-sig/virt-preview && \
    dnf copr enable -y vbatts/bazel && \
    dnf -y install \
        libvirt-devel-${LIBVIRT_VERSION} \
        bazel \
        cpio \
        patch \
        make \
        git \
        mercurial \
        sudo \
        gcc \
        gcc-c++ \
        glibc-devel \
        findutils \
        gradle \
        rsync-daemon \
        rsync \
        qemu-img \
        protobuf-compiler \
        python3-devel \
        redhat-rpm-config \
        jq \
        selinux-policy-devel \
        rpm-build && \
    dnf -y clean all

ENV GIMME_GO_VERSION=1.12.8

RUN mkdir -p /gimme && curl -sL https://raw.githubusercontent.com/travis-ci/gimme/master/gimme | HOME=/gimme bash >> /etc/profile.d/gimme.sh

ENV GOPATH="/go" GOBIN="/usr/bin"

# Install persisten go packages
RUN \
    mkdir -p /go && \
    source /etc/profile.d/gimme.sh && \
    # Install goimports
    go get -d golang.org/x/tools/cmd/goimports && \
    cd $GOPATH/src/golang.org/x/tools/cmd/goimports && \
    git checkout release-branch.go1.12 && \
    go install && \
    # Install mvdan/sh
    git clone https://github.com/mvdan/sh.git $GOPATH/src/mvdan.cc/sh && \
    cd $GOPATH/src/mvdan.cc/sh/cmd/shfmt && \
    git checkout v2.5.0 && \
    go get mvdan.cc/sh/cmd/shfmt && \
    go install && \
    go get -u -d k8s.io/code-generator/cmd/deepcopy-gen && \
    go get -u -d k8s.io/code-generator/cmd/defaulter-gen && \
    go get -u -d k8s.io/kube-openapi/cmd/openapi-gen && \
    go get -u -d github.com/golang/protobuf/protoc-gen-go && \
    # Install deepcopy-gen
    cd $GOPATH/src/k8s.io/code-generator/cmd/deepcopy-gen && \
    git checkout v0.16.4 && \
    go install && \
    # Install defaulter-gen
    cd $GOPATH/src/k8s.io/code-generator/cmd/defaulter-gen && \
    git checkout v0.16.4 && \
    go install && \
     # Install client-gen
    cd $GOPATH/src/k8s.io/code-generator/cmd/client-gen && \
    git checkout v0.16.4 && \
    go install && \
    # Install client-gen
    cd $GOPATH/src/k8s.io/code-generator/cmd/client-gen && \
    git checkout v0.16.4 && \
    go install && \
    # Install openapi-gen
    cd $GOPATH/src/k8s.io/kube-openapi/cmd/openapi-gen && \
    git checkout 30be4d16710ac61bce31eb28a01054596fe6a9f1 && \
    go install && \
    # Install protoc-gen-go
    cd $GOPATH/src/github.com/golang/protobuf/protoc-gen-go && \
    git checkout 1643683e1b54a9e88ad26d98f81400c8c9d9f4f9 && \
    go install

COPY entrypoint.sh /entrypoint.sh
