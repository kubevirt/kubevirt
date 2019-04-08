
FROM registry.svc.ci.openshift.org/openshift/release:golang-1.10 AS builder
WORKDIR /go/src/github.com/operator-framework/operator-marketplace
COPY . .
RUN make osbs-build

FROM registry.svc.ci.openshift.org/openshift/origin-v4.0:base
RUN useradd marketplace-operator
USER marketplace-operator
COPY --from=builder /go/src/github.com/operator-framework/operator-marketplace/build/_output/bin/marketplace-operator /usr/bin
ADD manifests /manifests

LABEL io.k8s.display-name="OpenShift Marketplace Operator" \
      io.k8s.description="This is a component of OpenShift Container Platform and manages the OpenShift Marketplace." \
      io.openshift.tags="openshift,marketplace" \
      io.openshift.release.operator=true \
      maintainer="AOS Marketplace <aos-marketplace@redhat.com>"

# entrypoint specified in operator.yaml as `marketplace-operator`
CMD ["/usr/bin/marketplace-operator"]
