#!/usr/bin/env bash

set -e

declare -A IMAGES
IMAGES[gocli]="gocli@sha256:c9abd3eb50abc339214a95db6e48828c562db8b3801854b8b4ed9fddbf87b7f3"
if [ -z $KUBEVIRTCI_PROVISION_CHECK ]; then
    IMAGES[k8s-fedora-1.17.0]="k8s-fedora-1.17.0@sha256:aebf67b8b1b499c721f4d98a7ab9542c680553a14cbc144d1fa701fe611f3c0d"
    IMAGES[k8s-1.17]="k8s-1.17@sha256:c56cece694d21370de10322780c358898f65bf5ffca9fce0837e9d807dd395b0"
    IMAGES[k8s-1.16]="k8s-1.16@sha256:31b404aecda635b561bf681508edea2316388ddacaaa10db5c789f4b58f155ce"
    IMAGES[k8s-1.15]="k8s-1.15@sha256:3df97b7bfba57c0f159b4b3c50a5638e534b6200bf20781bfd6c5d53aff41d23"
    IMAGES[k8s-1.14]="k8s-1.14@sha256:cb6b37bffffa3e2e2a351d972a09445f1a9f4dab72d3cd5fcdd533f81961e0bb"
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
