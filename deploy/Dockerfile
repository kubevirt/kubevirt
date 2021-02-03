FROM quay.io/openshift/origin-operator-registry:latest

WORKDIR /registry

COPY deploy/olm-catalog .

USER root

# Initialize the database
RUN ["initializer", "--manifests", "/registry/community-kubevirt-hyperconverged", "--output", "bundles.db"]
RUN ["chown", "1001", "bundles.db"]
RUN ["chmod", "-R", "g+rwx", "."]

USER 1001

# There are multiple binaries in the origin-operator-registry
# We want the registry-server
ENTRYPOINT ["registry-server"]
CMD ["--database", "bundles.db"]
