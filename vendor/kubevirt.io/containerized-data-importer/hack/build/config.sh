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
APISERVER="cdi-apiserver"
UPLOADPROXY="cdi-uploadproxy"
UPLOADSERVER="cdi-uploadserver"
FUNC_TEST_INIT="cdi-func-test-file-host-init"
FUNC_TEST_HTTP="cdi-func-test-file-host-http"

BINARIES="cmd/${CONTROLLER} cmd/${IMPORTER} cmd/${CLONER} cmd/${APISERVER} cmd/${UPLOADPROXY} cmd/${UPLOADSERVER} tools/${FUNC_TEST_INIT}"
CDI_PKGS="cmd/ pkg/ test/"

CONTROLLER_MAIN="cmd/${CONTROLLER}"
IMPORTER_MAIN="cmd/${IMPORTER}"
CLONER_MAIN="cmd/${CLONER}"
APISERVER_MAIN="cmd/${APISERVER}"
UPLOADPROXY_MAIN="cmd/${UPLOADPROXY}"
UPLOADSERVER_MAIN="cmd/${UPLOADSERVER}"

DOCKER_IMAGES="cmd/${CONTROLLER} cmd/${IMPORTER} cmd/${CLONER} cmd/${APISERVER} cmd/${UPLOADPROXY} cmd/${UPLOADSERVER} tools/${FUNC_TEST_INIT} tools/${FUNC_TEST_HTTP}"
DOCKER_REPO=${DOCKER_REPO:-kubevirt}
CONTROLLER_IMAGE_NAME=${CONTROLLER_IMAGE_NAME:-cdi-controller}
IMPORTER_IMAGE_NAME=${IMPORTER_IMAGE_NAME:-cdi-importer}
CLONER_IMAGE_NAME=${CLONER_IMAGE_NAME:-cdi-cloner}
APISERVER_IMAGE_NAME=${APISERVER_IMAGE_NAME:-cdi-apiserver}
UPLOADPROXY_IMAGE_NAME=${UPLOADPROXY_IMAGE_NAME:-cdi-uploadproxy}
UPLOADSERVER_IMAGE_NAME=${UPLOADSERVER_IMAGE_NAME:-cdi-uploadserver}
DOCKER_TAG=${DOCKER_TAG:-latest}
VERBOSITY=${VERBOSITY:-1}
PULL_POLICY=${PULL_POLICY:-IfNotPresent}
NAMESPACE=${NAMESPACE:-kube-system}

KUBERNETES_IMAGE="k8s-1.11.0@sha256:3412f158ecad53543c9b0aa8468db84dd043f01832a66f0db90327b7dc36a8e8"
OPENSHIFT_IMAGE="os-3.11.0-crio@sha256:ede83c50c2676e919ef3914e847450f2caf1d6771cb8183eb511bc08660d6fe1"


KUBEVIRT_PROVIDER=${KUBEVIRT_PROVIDER:-k8s-1.11.0}

function allPkgs() {
    ret=$(sed "s,kubevirt.io/containerized-data-importer,${CDI_DIR},g" <(go list ./... | grep -v "pkg/client" | sort -u))
    echo "$ret"
}

function parseTestOpts() {
    pkgs=""
    test_args=""
    while [[ $# -gt 0 ]] && [[ $1 != "" ]]; do
        case "${1}" in
        --test-args=*)
            test_args="${1#*=}"
            shift 1
            ;;
        ./*...)
            pkgs="${pkgs} ${1}"
            shift 1
            ;;
        *)
            echo "ABORT: Unrecognized option \"$1\""
            exit 1
            ;;
        esac
    done
}

function getClusterType() {
    local image
    case "${KUBEVIRT_PROVIDER}" in
    "k8s-1.11.0")
        image=$KUBERNETES_IMAGE
        ;;
    "os-3.11.0")
        image=$OPENSHIFT_IMAGE
        ;;
    esac
    echo "$image"
}
