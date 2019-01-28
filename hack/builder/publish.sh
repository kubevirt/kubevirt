#!/bin/bash

docker tag kubevirt/builder:28-5.0.0 docker.io/kubevirt/builder:28-5.0.0
docker push docker.io/kubevirt/builder:28-5.0.0
