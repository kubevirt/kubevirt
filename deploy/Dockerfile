FROM quay.io/openshift/origin-operator-registry:latest

COPY deploy/olm-catalog /registry

# Initialize the database
RUN initializer --manifests /registry/kubevirt-hyperconverged --output bundles.db

# There are multiple binaries in the origin-operator-registry
# We want the registry-server
ENTRYPOINT ["registry-server"]
CMD ["--database", "bundles.db"]
