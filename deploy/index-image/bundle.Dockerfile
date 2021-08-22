FROM scratch

ARG VERSION=1.6.0

LABEL operators.operatorframework.io.bundle.mediatype.v1=registry+v1
LABEL operators.operatorframework.io.bundle.manifests.v1=manifests/
LABEL operators.operatorframework.io.bundle.metadata.v1=metadata/
LABEL operators.operatorframework.io.bundle.package.v1=community-kubevirt-hyperconverged
LABEL operators.operatorframework.io.bundle.channels.v1=${VERSION}
LABEL operators.operatorframework.io.bundle.channel.default.v1=${VERSION}

COPY community-kubevirt-hyperconverged/${VERSION}/manifests/ /manifests/
COPY community-kubevirt-hyperconverged/${VERSION}/metadata /metadata/
