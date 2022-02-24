FROM quay.io/centos/centos:stream8

ARG SONOBUOY_ARCH
ARG BAZEL_ARCH

ENV BAZEL_VERSION 4.2.1

ENV KUBEVIRT_CREATE_BAZELRCS false

# Install packages
RUN dnf install -y dnf-plugins-core && \
    dnf config-manager --enable powertools && \
    dnf -y install \
        java-11-openjdk-devel \
        libvirt-devel \
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
        rsync-daemon \
        rsync \
        qemu-img \
        protobuf-compiler \
        python38 \
        python38-devel \
        python38-pip \
        redhat-rpm-config \
        jq \
        wget \
        diffutils && \
    dnf -y clean all

# Avoids the need to install sssd-client by disabling lookups
COPY nsswitch.conf /etc/nsswitch.conf

# Necessary for Bazel to find Python inside the container
#
# https://github.com/bazelbuild/bazel/issues/8665
# https://github.com/bazelbuild/bazel/issues/11554
RUN ln -s /usr/bin/python3 /usr/bin/python

# install gradle (required for swagger)
RUN wget https://services.gradle.org/distributions/gradle-6.6-bin.zip && \
    mkdir /opt/gradle && \
    unzip -d /opt/gradle gradle-6.6-bin.zip

ENV PATH=$PATH:/opt/gradle/gradle-6.6/bin \
    JAVA_HOME=/usr/lib/jvm/java-11

ENV GIMME_GO_VERSION=1.16.6

# reference to master is for an external repo and can't yet be changed
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
    go get -v k8s.io/code-generator/cmd/deepcopy-gen@v0.20.2 && \
    go get -v k8s.io/code-generator/cmd/defaulter-gen@v0.20.2  && \
    go get -v k8s.io/kube-openapi/cmd/openapi-gen@30be4d1 && \
    go get -v github.com/golang/protobuf/protoc-gen-go@1643683 && \
    go get -v k8s.io/code-generator/cmd/client-gen@v0.20.2 && \
    go get -v github.com/securego/gosec/v2/cmd/gosec@0ce48a5 && \
    go get -v sigs.k8s.io/controller-tools/cmd/controller-gen@v0.4.0 && \
    go clean -cache -modcache

RUN set -x && \
    source /etc/profile.d/gimme.sh && \
    go get -v -u github.com/mattn/goveralls@21feffdfd && \
    go get -v -u github.com/golang/mock/gomock@v1.4.3 && \
    go get -v -u github.com/rmohr/mock/mockgen@v0.0.0-20170327080805-5980f5d9a70d && \
    go get -v -u github.com/rmohr/go-swagger-utils/swagger-doc@v0.0.0-20181110145135-878f7deb1d17 && \
    go get -v -u github.com/onsi/ginkgo/ginkgo@v1.12.0 && \
    go clean -cache -modcache

RUN set -x && \
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
    go clean -cache -modcache -r && \
    rm -rf /test-infra && \
    rm -rf /go && mkdir /go

RUN pip3 install --upgrade operator-courier==2.1.11

ENV SONOBUOY_VERSION=0.19.0
RUN set -x && \
    wget https://github.com/vmware-tanzu/sonobuoy/releases/download/v${SONOBUOY_VERSION}/sonobuoy_${SONOBUOY_VERSION}_linux_${SONOBUOY_ARCH}.tar.gz && \
    tar xvf sonobuoy_${SONOBUOY_VERSION}_linux_${SONOBUOY_ARCH}.tar.gz && \
    chmod +x sonobuoy && \
    mv sonobuoy /usr/bin

COPY rsyncd.conf /etc/rsyncd.conf

COPY entrypoint.sh /entrypoint.sh

COPY create_bazel_cache_rcs.sh /create_bazel_cache_rcs.sh

RUN curl -L -o /usr/bin/bazel https://github.com/bazelbuild/bazel/releases/download/${BAZEL_VERSION}/bazel-${BAZEL_VERSION}-linux-${BAZEL_ARCH} && chmod u+x /usr/bin/bazel

ENTRYPOINT [ "/entrypoint.sh" ]
