#!/bin/bash
unset binaries docker_images docker_tag docker_tag_alt image_prefix image_prefix_alt manifest_templates \
    namespace image_pull_policy verbosity \
    csv_version package_name

# shellcheck source=hack/config-default.sh
source ${KUBEVIRT_PATH}hack/config-default.sh
# shellcheck source=cluster-up/hack/config.sh
source ${KUBEVIRT_PATH}cluster-up/hack/config.sh

export binaries docker_images docker_tag docker_tag_alt image_prefix image_prefix_alt manifest_templates \
    namespace image_pull_policy verbosity \
    csv_version package_name
