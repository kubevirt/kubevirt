#!/usr/bin/env bash
set -exu

IMAGE_TAG=$1
NEW_IMAGE_TAG=$2

docker tag "$IMAGE_TAG" "$NEW_IMAGE_TAG"
docker push "$NEW_IMAGE_TAG"
