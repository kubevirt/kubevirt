FROM registry.svc.ci.openshift.org/openshift/origin-v4.0:base

RUN useradd marketplace-operator
USER marketplace-operator

ADD build/_output/bin/operator-marketplace /usr/local/bin/marketplace-operator
