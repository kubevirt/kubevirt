ARG ARCH
FROM ${ARCH}/fedora:30

COPY qemu-ppc64le-static /usr/bin/qemu-ppc64le-static

ENV LIBVIRT_VERSION 5.1.0

ENV BAZEL_VERSION 1.2.1

# Install packages
RUN dnf install -y dnf-plugins-core && \
    dnf copr enable -y @virtmaint-sig/virt-preview && \
    dnf copr enable -y vbatts/bazel && \
    dnf -y install \
    libvirt-devel-${LIBVIRT_VERSION} \
    bazel-${BAZEL_VERSION} \
    cpio \
    patch \
    make \
    git \
    mercurial \
    sudo \
    gcc \
    gcc-c++ \
    glibc-static \
    libstdc++-static \
    glibc-devel \
    findutils \
    gradle \
    rsync-daemon \
    rsync \
    qemu-img \
    protobuf-compiler \
    python3-devel \
    redhat-rpm-config \
    jq && \
    dnf -y clean all

ENV GIMME_GO_VERSION=1.13.14

RUN mkdir -p /gimme && curl -sL \
    https://raw.githubusercontent.com/travis-ci/gimme/master/gimme | \
    HOME=/gimme bash >> /etc/profile.d/gimme.sh

ENV GOPATH="/go" GOBIN="/usr/bin" GO111MODULE="on"

# Install persistent go packages
RUN set -x && \
    mkdir -p /go && \
    source /etc/profile.d/gimme.sh && \
    go get -v golang.org/x/tools/cmd/goimports@d5fe738 && \
    go get -v mvdan.cc/sh/v3/cmd/shfmt@v3.1.1 && \
    go get -v k8s.io/code-generator/cmd/deepcopy-gen@v0.18.2 && \
    go get -v k8s.io/code-generator/cmd/defaulter-gen@v0.15.12  && \
    go get -v k8s.io/kube-openapi/cmd/openapi-gen@30be4d1 && \
    go get -v github.com/golang/protobuf/protoc-gen-go@1643683 && \
    go get -v k8s.io/code-generator/cmd/client-gen@v0.16.4 && \
    go get -v sigs.k8s.io/controller-tools/cmd/controller-gen@v0.3.0 && \
    go clean -cache -modcache

RUN set -x && \
    source /etc/profile.d/gimme.sh && \
    go get -v -u github.com/mattn/goveralls@v0.0.5 && \
    go get -v -u github.com/golang/mock/gomock@v1.4.3 && \
    go get -v -u github.com/rmohr/mock/mockgen@v0.0.0-20170327080805-5980f5d9a70d && \
    go get -v -u github.com/rmohr/go-swagger-utils/swagger-doc@v0.0.0-20181110145135-878f7deb1d17 && \
    go get -v -u github.com/onsi/ginkgo/ginkgo@v1.12.0 && \
    go clean -cache -modcache

RUN set -x && \
    cd / && \
    source /etc/profile.d/gimme.sh && \
    git clone https://github.com/kubernetes/test-infra.git && \
    cd /test-infra && \
    git checkout fd0699b906b0593a33ba2bddd3b1ae8822f42dd8 && \
    cd /test-infra/robots/pr-creator && \
    go install && \
    cd /test-infra/robots/issue-creator && \
    go install && \
    cd /test-infra/robots/pr-labeler && \
    go install && \
    go clean -cache -modcache && \
    rm -rf /test-infra && \
    rm -rf /go && mkdir /go

RUN pip3 install j2cli operator-courier==1.3.0

COPY rsyncd.conf /etc/rsyncd.conf

COPY entrypoint.sh /entrypoint.sh

ENTRYPOINT [ "/entrypoint.sh" ]
