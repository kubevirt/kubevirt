#!/bin/bash -xe
# Patch/PR testing hook script
#

main() {
    trap 'cleanup' SIGTERM SIGINT SIGQUIT EXIT
    setup
    run_tests
}

setup() {
    # Some scripts like things Jenkins so it expects a WORKSPACE
    # variable
    setup_go_dirs
    setup_vagrant_storage
    setup_vagrant_plugins
    setup_vagrant_env
}

setup_vagrant_storage() {
    # We assume that if we see /var/host_cache we are running in a container or
    # a chroot, and this is a mount of /var/cache/ from the underlying host
    if [[ -d /var/host_cache ]]; then
        # Store VAGRANT_HOME on the host if possible
        if mkdir -p /var/host_cache/vagrant/$UID; then
            export VAGRANT_HOME=/var/host_cache/vagrant/$UID
        fi
        # Use a libvirt pool stored in the persistent cache so backing stores
        # stick around between runs
        if mkdir -p /var/host_cache/vagrant/pool_$UID; then
            export VAGRANT_POOL=vagrant_pool_$UID
            if ! virsh pool-info "$VAGRANT_POOL"; then
                virsh pool-create-as "$VAGRANT_POOL" \
                    dir --target /var/cache/vagrant/pool_$UID
            fi
            virsh pool-refresh "$VAGRANT_POOL"
        fi
        # Put the .vagrant file on the host too so we can keep the VMs between
        # runs
        local ws_sha null
        read ws_sha null < <(sha256sum <<<"${WORKSPACE?-$PWD}")
        if mkdir -p "/var/host_cache/vagrant/dotfiles/$ws_sha"; then
            VAGRANT_DOTFILE_PATH="/var/host_cache/vagrant/dotfiles/$ws_sha"
            export VAGRANT_DOTFILE_PATH
            echo ".vagrant file will be stored in: $VAGRANT_DOTFILE_PATH"
        fi
    fi
}

setup_vagrant_plugins() {
    local PLUGINS=(vagrant-libvirt vagrant-cachier)

    for plugin in "${PLUGINS[@]}"; do
        if ! vagrant plugin list | grep -q "^$plugin"; then
            vagrant plugin install "$plugin"
        fi
    done
}


setup_vagrant_env() {
    # TODO: Find a way to make this work with some userspace NFS server or 9p
    # export VAGRANT_USE_NFS=true
    export VAGRANT_USE_NFS=false
    export VAGRANT_CACHE_RPM=true
    export VAGRANT_CACHE_DOCKER=true
    export VAGRANT_NUM_NODES=1
}

setup_go_dirs() {
    # Try to set WORKSPACE so that GOPATH could be $WORKSPACE/go
    WORKSPACE="${PWD%/go/src/kubevirt.io/kubevirt}"
    if [[ "$WORKSPACE" == "$PWD" ]]; then
        WORKSPACE="${PWD%/*}"
    fi
    export WORKSPACE
    export GOPATH="$WORKSPACE/go"

    local src_base="$GOPATH/src/kubevirt.io"
    local src_path="$src_base/kubevirt"
    # Make symlinks so that source is where the go compiler likes it to be
    if [[ "$PWD" != "$src_path" ]]; then
        mkdir -p "$src_base"
        ln -s "$PWD" "$src_path"
        cd "$src_path"
    fi
}

run_tests() {
    /bin/bash automation/test.sh
}

cleanup() {
    # We need to destroy the VMs so they won't be found in 'virsh list'.
    # Otherwise the environment cleanup code will clean them up as well as the
    # Docker data disks that we want to keep around.
    vagrant destroy
}

if [[ "${BASH_SOURCE[0]}" == "$0" ]]; then
    main "$@"
fi
