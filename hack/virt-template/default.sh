virt_template_version=${VIRT_TEMPLATE_VERSION:-"v0.1.2"}

# Image digests per architecture
virt_template_apiserver_digest_amd64=${VIRT_TEMPLATE_APISERVER_DIGEST_AMD64:-"sha256:9b5d7fde015a467cf11cf84a63639a5457897fc77bbc7324fc6632e08124ae12"}
virt_template_apiserver_digest_arm64=${VIRT_TEMPLATE_APISERVER_DIGEST_ARM64:-"sha256:1933e0f78b0a58bb195fb450b629fe5bd12cad50a0fa18eb702ef69872a07462"}
virt_template_apiserver_digest_s390x=${VIRT_TEMPLATE_APISERVER_DIGEST_S390X:-"sha256:135695e94e90145e2fa986982b3098ac720f73c0cc5376f44f11e3645127ebbc"}

virt_template_controller_digest_amd64=${VIRT_TEMPLATE_CONTROLLER_DIGEST_AMD64:-"sha256:910a8fffa74571e299075a1436db553f529f9b717ff979a3200e8cfcef92009f"}
virt_template_controller_digest_arm64=${VIRT_TEMPLATE_CONTROLLER_DIGEST_ARM64:-"sha256:1e2c35d07089bc5ddd14e3457410d9817fe0745cbf9321b017ebfd36e748c0dc"}
virt_template_controller_digest_s390x=${VIRT_TEMPLATE_CONTROLLER_DIGEST_S390X:-"sha256:304a1df0a1671e2d1ecc7b9970910dafcce7f530f94c822cf50c52316ac6382a"}

virt_template_targets="virt-template-apiserver virt-template-controller"
function is_virt_template_target() {
    local target=$1
    for vt_target in ${virt_template_targets}; do
        if [[ "${target}" == "${vt_target}" ]]; then
            return 0
        fi
    done
    return 1
}
