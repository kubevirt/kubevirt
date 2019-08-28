# we need to pin the 4.2.0 image because https://github.com/operator-framework/operator-registry/issues/70
FROM quay.io/openshift/origin-operator-registry:4.2.0

COPY olm-catalog /registry

# Initialize the database
RUN initializer --manifests /registry/kubevirt-hyperconverged --output bundles.db

# There are multiple binaries in the origin-operator-registry
# We want the registry-server
ENTRYPOINT ["registry-server"]
CMD ["--database", "bundles.db"]
