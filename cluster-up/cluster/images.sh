#!/usr/bin/env bash

set -e

declare -A IMAGES
IMAGES[gocli]="gocli@sha256:220f55f6b1bcb3975d535948d335bd0e6b6297149a3eba1a4c14cad9ac80f80d"
if [ -z $KUBEVIRTCI_PROVISION_CHECK ]; then
    IMAGES[k8s-fedora-1.17.0]="k8s-fedora-1.17.0@sha256:5fc78a20fae562ce78618fc25d0a15acd6de384b27adc3b6cd54f54f6c9d4fdf"
    IMAGES[k8s-1.17]="k8s-1.17@sha256:427d69d702043cbaf3e8f7bedb7cc1444a1d90695e3fd206799865cf5a1016f4"
    IMAGES[k8s-1.16]="k8s-1.16@sha256:3559c7d83baa16d1bb641c38f24afee82a24023f9cc03bf4cffc9b54435d35ab"
    IMAGES[k8s-1.15]="k8s-1.15@sha256:bfa0b87f7a561d15ed8bdba1506f34daf024c48d70677a02920e02494e40354b"
    IMAGES[k8s-1.14]="k8s-1.14@sha256:410468892ed51308b0e71c755d2b3a65b060a22302c7cfdbc213b5566de0e661"
    IMAGES[k8s-multus-1.13.3]="k8s-multus-1.13.3@sha256:c0bcf0d2e992e5b4d96a7bcbf988b98b64c4f5aef2f2c4d1c291e90b85529738"
    IMAGES[okd-4.1]="okd-4.1@sha256:e7e3a03bb144eb8c0be4dcd700592934856fb623d51a2b53871d69267ca51c86"
    IMAGES[okd-4.2]="okd-4.2@sha256:a830064ca7bf5c5c2f15df180f816534e669a9a038fef4919116d61eb33e84c5"
    IMAGES[okd-4.3]="okd-4.3@sha256:63abc3884002a615712dfac5f42785be864ea62006892bf8a086ccdbca8b3d38"
    IMAGES[ocp-4.3]="ocp-4.3@sha256:d293f0bca338136ed136b08851de780d710c9e40e2a1d18e5a5595491dbdd1ea"
    IMAGES[ocp-4.4]="ocp-4.4@sha256:42497f3a848c2847e3caeff6fbb7f4bb28ee48b692c0541ec7099392067a0387"
fi
export IMAGES

IMAGE_SUFFIX=""
if [[ $KUBEVIRT_PROVIDER =~ (ocp|okd).* ]]; then
    IMAGE_SUFFIX="-provision"
fi

image="${IMAGES[$KUBEVIRT_PROVIDER]:-${KUBEVIRT_PROVIDER}${IMAGE_SUFFIX}:latest}"
export image
