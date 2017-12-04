unset binaries docker_images docker_prefix docker_tag manifest_templates \
      master_ip master_port network_provider

source ${KUBEVIRT_PATH}hack/config-default.sh

# Let devs override any default variables, to avoid needing
# to change the version controlled config-default.sh file
test -f "hack/config-local.sh" && source hack/config-local.sh

export binaries docker_images docker_prefix docker_tag manifest_templates \
       master_ip master_port network_provider
