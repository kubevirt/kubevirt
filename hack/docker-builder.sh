#!/bin/bash
#
# This file is part of the KubeVirt project
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#     http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.
#
# Copyright 2017 Red Hat, Inc.
#

set -e

source hack/config.sh

if [ -z "$1" ]; then
    target="build"
else
    target=$1
    shift
fi

image_name="${docker_prefix}/builder:${docker_tag}"
if [[ $target == "clean" ]]; then
    container=$(docker ps -a --format '{{ .Image }}\t{{ .ID }}' | grep ^$image_name | cut -f2)
    if [[ -n $container ]]; then
	      docker rm $container
    fi
    if [[ -n $(docker image ls -q kubevirt/builder:latest) ]]; then
	      docker image rm $image_name
    fi
elif [[ $target == "install" ]]; then
    echo "Creating a container for the build"
    docker build --build-arg userid=$UID -t $image_name hack/
    echo "Running build in a docker container"
    docker run -v $(go env GOPATH):/builder -e KUBEVIRT_BUILD_TYPE=plain $image_name hack/build-go.sh $target $@
else
    echo "Running $0 with invalid target: $target" 1>&2
fi
