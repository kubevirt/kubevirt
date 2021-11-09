FROM fedora:34

MAINTAINER "The KubeVirt Project" <kubevirt-dev@googlegroups.com>

RUN echo "diskspacecheck=0" >> /etc/dnf/dnf.conf && dnf update -y && dnf install qemu xz gzip git python3-pip gcc autoconf automake libtool python jq wget -y && dnf clean all

RUN pip3 install j2cli && pip3 install operator-courier

ENV GO_VERSION=1.17.3 \
    KUBEBUILDER_VERSION="3.2.0" \
    ARCH="amd64" \
    GOPATH="/go" \
    PATH=$PATH:${GOPATH}/go/bin \
    GO111MODULE=on

RUN \
    mkdir -p ${GOPATH} && \
    wget https://golang.org/dl/go${GO_VERSION}.linux-amd64.tar.gz && \
    tar -C ${GOPATH}/../ -xzf go${GO_VERSION}.linux-amd64.tar.gz && \
    rm -rf go${GO_VERSION}.linux-amd64.tar.gz && \
    export PATH=${GOPATH}/bin:$PATH && \
    eval $(go env) && \
    go get github.com/onsi/ginkgo/ginkgo && \
    go get github.com/onsi/gomega && \
    go get golang.org/x/tools/cmd/goimports && \
    go get -u golang.org/x/lint/golint&& \
    go get -u github.com/rmohr/go-swagger-utils/swagger-doc && \
    go get github.com/mattn/goveralls && \
    go get mvdan.cc/sh/v3/cmd/shfmt && \
    go get -u -d k8s.io/code-generator/cmd/deepcopy-gen && \
    go get -u -d k8s.io/kube-openapi/cmd/openapi-gen && \
    go get -u -d github.com/Masterminds/glide && \
    (curl -L -O "https://github.com/kubernetes-sigs/kubebuilder/releases/download/v${KUBEBUILDER_VERSION}/kubebuilder_${KUBEBUILDER_VERSION}_linux_${ARCH}" && \
     mv kubebuilder_${KUBEBUILDER_VERSION}_linux_${ARCH} /usr/local/kubebuilder )

ADD entrypoint.sh /entrypoint.sh

ENTRYPOINT ["/entrypoint.sh"]
