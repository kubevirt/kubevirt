FROM quay.io/openshift/origin-operator-registry:latest

WORKDIR /registry

COPY deploy/olm-catalog .

USER root

# Initialize the database
RUN ["initializer", "--manifests", "/registry/kubevirt-hyperconverged", "--output", "bundles.db"]

RUN ["chown", "1001", "bundles.db"]

USER 1001

# There are multiple binaries in the origin-operator-registry
# We want the registry-server
ENTRYPOINT ["registry-server"]
CMD ["--database", "bundles.db"]
