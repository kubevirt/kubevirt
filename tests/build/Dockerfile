FROM fedora:30

MAINTAINER "The KubeVirt Project" <kubevirt-dev@googlegroups.com>

RUN echo "diskspacecheck=0" >> /etc/dnf/dnf.conf && dnf update -y && dnf install qemu xz gzip git python3-pip gradle gcc autoconf automake libtool python jq -y && dnf clean all

RUN pip3 install j2cli && pip3 install operator-courier

ENV GIMME_GO_VERSION=1.16.4 \
    KUBEBUILDER_VERSION="2.3.1" \
    ARCH="amd64" \
    GOPATH="/go" \
    PATH=$PATH:${GOPATH}/go/bin \
    GO111MODULE=on

RUN mkdir -p /gimme && \
    curl -sL https://raw.githubusercontent.com/travis-ci/gimme/master/gimme | HOME=/gimme bash >> /etc/profile.d/gimme.sh

RUN \
    mkdir -p ${GOPATH} && \
    source /etc/profile.d/gimme.sh && \
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
    GO111MODULE=off go get -u -d github.com/Masterminds/glide && \
    (cd $GOPATH/src/github.com/Masterminds/glide && \
    git checkout v0.13.2 -b build-v0.13.2 && \
    GO111MODULE=off go install ./...) && \
    (curl -L -O "https://github.com/kubernetes-sigs/kubebuilder/releases/download/v${KUBEBUILDER_VERSION}/kubebuilder_${KUBEBUILDER_VERSION}_linux_${ARCH}.tar.gz" && \
     tar -zxvf kubebuilder_${KUBEBUILDER_VERSION}_linux_${ARCH}.tar.gz && \
     mv kubebuilder_${KUBEBUILDER_VERSION}_linux_${ARCH} /usr/local/kubebuilder && \
     rm kubebuilder_${KUBEBUILDER_VERSION}_linux_${ARCH}.tar.gz )

ADD entrypoint.sh /entrypoint.sh

ENTRYPOINT ["/entrypoint.sh"]
