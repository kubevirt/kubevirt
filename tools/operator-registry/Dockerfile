FROM quay.io/openshift/origin-operator-registry

COPY deploy/olm-catalog /registry

# Initialize the database
RUN initializer --manifests /registry --output bundles.db

# There are multiple binaries in the origin-operator-registry
# We want the registry-server
ENTRYPOINT ["registry-server"]
CMD ["--database", "bundles.db"]
