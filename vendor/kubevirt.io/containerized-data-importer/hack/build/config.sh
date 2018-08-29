#Copyright 2018 The CDI Authors.
#
#Licensed under the Apache License, Version 2.0 (the "License");
#you may not use this file except in compliance with the License.
#You may obtain a copy of the License at
#
#    http://www.apache.org/licenses/LICENSE-2.0
#
#Unless required by applicable law or agreed to in writing, software
#distributed under the License is distributed on an "AS IS" BASIS,
#WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
#See the License for the specific language governing permissions and
#limitations under the License.

CONTROLLER="cdi-controller"
IMPORTER="cdi-importer"
CLONER="cdi-cloner"

BINARIES="cmd/${CONTROLLER} cmd/${IMPORTER}"
CDI_PKGS="cmd/ pkg/ test/"

CONTROLLER_MAIN="cmd/${CONTROLLER}"
IMPORTER_MAIN="cmd/${IMPORTER}"
CLONER_MAIN="cmd/${CLONER}"

DOCKER_IMAGES="cmd/${CONTROLLER} cmd/${IMPORTER} cmd/${CLONER}"
DOCKER_REPO=${DOCKER_REPO:-kubevirt}
DOCKER_TAG=${DOCKER_TAG:-latest}
VERBOSITY=${VERBOSITY:-1}
PULL_POLICY=${PULL_POLICY:-IfNotPresent}

function allPkgs {
    ret=$(sed "s,kubevirt.io/containerized-data-importer,${CDI_DIR},g" <(go list ./... | grep -v -E "vendor|pkg/client" | sort -u ))
    echo "$ret"
}
