unset binaries docker_images docker_prefix docker_tag manifest_templates \
    master_ip network_provider kubeconfig manifest_docker_prefix namespace image_pull_policy

KUBEVIRT_PROVIDER=${KUBEVIRT_PROVIDER:-${PROVIDER}}

# We don't have any control over the infrastructure avaialbe on an external provider.  In some cases
# like using hack/local-cluster-up.sh we can get everything we need here, but in others lik GKE we may
# not be able to add things depending on our config.  So, by default we'll skip trying to deploy the tests
# for external providers, but provide this override for folks that know what they're doing
KUBEVIRT_DEPLOY_TESTS_TO_EXTERNAL_PROVIDER=${KUBEVIRT_DEPLOY_TESTS_TO_EXTERNAL_PROVIDER:-false}

source ${KUBEVIRT_PATH}hack/config-default.sh source ${KUBEVIRT_PATH}hack/config-${KUBEVIRT_PROVIDER}.sh

# Allow different providers to override default config values
test -f "hack/config-provider-${KUBEVIRT_PROVIDER}.sh" && source hack/config-provider-${KUBEVIRT_PROVIDER}.sh

# Let devs override any default variables, to avoid needing
# to change the version controlled config-default.sh file
test -f "hack/config-local.sh" && source hack/config-local.sh

export binaries docker_images docker_prefix docker_tag manifest_templates \
    master_ip network_provider kubeconfig namespace image_pull_policy
