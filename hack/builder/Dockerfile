FROM quay.io/centos/centos:stream9

ARG ARCH
ARG SONOBUOY_ARCH
ARG BAZEL_ARCH

ENV BAZEL_VERSION=6.5.0
ENV GIMME_GO_VERSION=1.24.7
ENV OPERATOR_COURIER_VERSION=2.1.11
ENV SONOBUOY_VERSION=0.56.9
ENV GOLANGCI_LINT_VERSION=v2.2.2

ENV KUBEVIRT_CREATE_BAZELRCS=false

# Install packages
RUN dnf install -y dnf-plugins-core && \
    dnf config-manager --enable crb && \
    dnf install -y --setopt=install_weak_deps=False \
        java-11-openjdk-devel \
        libvirt-devel \
        cpio \
        patch \
        make \
        git \
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
        python3 \
        python3-devel \
        python3-pip \
        python3-setuptools \
        redhat-rpm-config \
        jq \
        wget \
        rubygems \
        diffutils \
        skopeo && \
    dnf clean -y all

# Avoids the need to install sssd-client by disabling lookups
COPY nsswitch.conf /etc/nsswitch.conf

# Necessary for generation of HTML-formatted API docs (.adoc)
RUN gem install asciidoctor

# Generates Asciidoc files from swagger.json
ADD https://storage.googleapis.com/builddeps/swagger2markup-cli-1.3.3.jar /opt/swagger2markup-cli/swagger2markup-cli-1.3.3.jar

# Necessary for Bazel to find Python inside the container
#
# https://github.com/bazelbuild/bazel/issues/8665
# https://github.com/bazelbuild/bazel/issues/11554
RUN ln -s /usr/bin/python3 /usr/bin/python

ENV JAVA_HOME=/usr/lib/jvm/java-11

# reference to master is for an external repo and can't yet be changed
RUN mkdir -p /gimme && curl -sL \
    https://raw.githubusercontent.com/travis-ci/gimme/master/gimme | \
    HOME=/gimme bash >> /etc/profile.d/gimme.sh

ENV GOPATH="/go" GOBIN="/usr/bin" GO111MODULE="on"

# Install persistent go packages
RUN set -x && \
    mkdir -p /go && \
    source /etc/profile.d/gimme.sh && \
    go install -v golang.org/x/tools/cmd/goimports@d5fe738 && \
    go install -v mvdan.cc/sh/v3/cmd/shfmt@v3.1.1 && \
    go install -v k8s.io/code-generator/cmd/conversion-gen@v0.31.1  && \
    go install -v k8s.io/code-generator/cmd/deepcopy-gen@v0.31.1 && \
    go install -v k8s.io/code-generator/cmd/defaulter-gen@v0.31.1  && \
    go install -v k8s.io/kube-openapi/cmd/openapi-gen@f7e401e && \
    go install -v github.com/golang/protobuf/protoc-gen-go@1643683 && \
    go install -v k8s.io/code-generator/cmd/client-gen@v0.31.1 && \
    go install -v github.com/securego/gosec/v2/cmd/gosec@0ce48a5 && \
    go install -v sigs.k8s.io/controller-tools/cmd/controller-gen@v0.16.3 && \
    go install -v github.com/kubevirt/monitoring/monitoringlinter/cmd/monitoringlinter@e2be790 && \
    go clean -cache -modcache

RUN set -x && \
    source /etc/profile.d/gimme.sh && \
    go install -v github.com/mattn/goveralls@21feffdfd && \
    go install -v go.uber.org/mock/mockgen@v0.5.1 && \
    go install -v github.com/rmohr/go-swagger-utils/swagger-doc@v0.0.0-20181110145135-878f7deb1d17 && \
    go clean -cache -modcache

RUN set -x && \
    source /etc/profile.d/gimme.sh && \
    git clone https://github.com/kubernetes/test-infra.git && \
    cd /test-infra && \
    git checkout f2693aba912dd40c974304caca999d45ee8dce33 && \
    cd /test-infra/robots/pr-creator && \
    go install && \
    cd /test-infra/robots/issue-creator && \
    go install && \
    cd /test-infra/robots/pr-labeler && \
    go install && \
    go clean -cache -modcache -r && \
    rm -rf /test-infra && \
    rm -rf /go && mkdir /go

RUN set -x && \
    source /etc/profile.d/gimme.sh && \
    go install -v mvdan.cc/gofumpt@v0.8.0 && \
    curl -sSfL https://raw.githubusercontent.com/golangci/golangci-lint/master/install.sh | sh -s -- -b "$(go env GOROOT)"/bin $GOLANGCI_LINT_VERSION

RUN pip3 install --upgrade operator-courier==${OPERATOR_COURIER_VERSION}

RUN set -x && \
    wget https://github.com/vmware-tanzu/sonobuoy/releases/download/v${SONOBUOY_VERSION}/sonobuoy_${SONOBUOY_VERSION}_linux_${SONOBUOY_ARCH}.tar.gz && \
    tar xvf sonobuoy_${SONOBUOY_VERSION}_linux_${SONOBUOY_ARCH}.tar.gz && \
    chmod +x sonobuoy && \
    mv sonobuoy /usr/bin && \
    rm sonobuoy_${SONOBUOY_VERSION}_linux_${SONOBUOY_ARCH}.tar.gz

COPY rsyncd.conf /etc/rsyncd.conf

COPY entrypoint.sh /entrypoint.sh

COPY create_bazel_cache_rcs.sh /create_bazel_cache_rcs.sh

RUN if test "${ARCH}" != "s390x"; then \
       curl -L -o /usr/bin/bazel https://github.com/bazelbuild/bazel/releases/download/${BAZEL_VERSION}/bazel-${BAZEL_VERSION}-linux-${BAZEL_ARCH} && \
       chmod u+x /usr/bin/bazel; \
    fi

# Add /root/go/src/kubevirt.io/kubevirt and mark it as a safe directory for git
RUN mkdir -p /root/go/src/kubevirt.io/kubevirt && git config --global --add safe.directory /root/go/src/kubevirt.io/kubevirt

ENTRYPOINT [ "/entrypoint.sh" ]
