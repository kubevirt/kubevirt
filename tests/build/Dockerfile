FROM fedora:35

MAINTAINER "The KubeVirt Project" <kubevirt-dev@googlegroups.com>


RUN echo "diskspacecheck=0" >> /etc/dnf/dnf.conf && dnf update -y && dnf install qemu xz gzip git python3-pip gcc autoconf automake libtool python jq wget -y && dnf clean all

RUN pip3 install j2cli && pip3 install operator-courier

ENV GO_VERSION=1.18.1 \
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
    go install github.com/onsi/ginkgo/v2/ginkgo@latest && \
    go install golang.org/x/tools/cmd/goimports@latest && \
    go install golang.org/x/lint/golint@latest && \
    go install github.com/rmohr/go-swagger-utils/swagger-doc@latest && \
    go install github.com/mattn/goveralls@latest && \
    go install mvdan.cc/sh/v3/cmd/shfmt@latest && \
    go install k8s.io/code-generator/cmd/deepcopy-gen@latest && \
    go install k8s.io/kube-openapi/cmd/openapi-gen@latest && \
    (curl -L -O "https://github.com/kubernetes-sigs/kubebuilder/releases/download/v${KUBEBUILDER_VERSION}/kubebuilder_${KUBEBUILDER_VERSION}_linux_${ARCH}" && \
     mv kubebuilder_${KUBEBUILDER_VERSION}_linux_${ARCH} /usr/local/kubebuilder )

ADD entrypoint.sh /entrypoint.sh

ENTRYPOINT ["/entrypoint.sh"]
