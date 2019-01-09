FROM fedora:28

RUN dnf install -y qemu xz gzip git gradle && dnf clean all

ENV GIMME_GO_VERSION=1.10 GOPATH="/go"

RUN mkdir -p /gimme && curl -sL https://raw.githubusercontent.com/travis-ci/gimme/master/gimme | HOME=/gimme bash >> /etc/profile.d/gimme.sh

RUN \
    mkdir -p ${GOPATH} && \
    source /etc/profile.d/gimme.sh && \
    eval $(go env) && \
    (go get -u github.com/onsi/ginkgo/ginkgo && \
     cd $GOPATH/src/github.com/onsi/ginkgo/ginkgo && \
     go install ./... ) && \
    go get github.com/onsi/gomega && \
    go get golang.org/x/tools/cmd/goimports && \
    ( go get -d mvdan.cc/sh/cmd/shfmt || echo "**** Expecting error \"cannot find package mvdan.cc/sh/v2/fileutil\"" ) && \
    ( cd $GOPATH/src/golang.org/x/tools/cmd/goimports && \
    go install ./... ) && \
    ( cd $GOPATH/src/mvdan.cc/sh/cmd/shfmt && \
    git checkout v2.5.0 -b build-v2.5.0 && \
    go install ./... ) && \
    ( go get -d github.com/mattn/goveralls && \
    cd $GOPATH/src/github.com/mattn/goveralls && \
    go install ./... ) && \
    ( go get -u golang.org/x/lint/golint ) && \
    ( go get -u github.com/rmohr/go-swagger-utils/swagger-doc ) && \
    ( go get -u -d k8s.io/code-generator/cmd/deepcopy-gen && \
    go get -u -d k8s.io/kube-openapi/cmd/openapi-gen ) && \
    ( cd $GOPATH/src/k8s.io/code-generator/cmd/deepcopy-gen && \
    git checkout kubernetes-1.11.2 && \
    go install ./... ) && \
    ( cd $GOPATH/src/k8s.io/kube-openapi/cmd/openapi-gen && \
    git checkout c59034cc13d587f5ef4e85ca0ade0c1866ae8e1d && \
    go install ./... )


ADD entrypoint.sh /entrypoint.sh

ENTRYPOINT ["/entrypoint.sh"]
