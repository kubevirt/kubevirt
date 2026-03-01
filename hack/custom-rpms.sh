#!/usr/bin/env bash
# Builds QEMU and libvirt RPMs from local source directories
# See docs/custom-rpms.md for usage

set +x

source $(dirname "$0")/common.sh

KUBEVIRT_CRI=$(determine_cri_bin)
fail_if_cri_bin_missing
SCRIPT_DIR=${KUBEVIRT_DIR}/hack/custom-rpms

dockerized() {
  if [[ -z "$SRCS" ]]; then
    echo "The required variable 'SRCS' is not defined."
    exit 1
  fi

  default_BUILDER_IMAGE="rpm-build-image"

  BUILDER_IMAGE=${BUILDER_IMAGE:-"${default_BUILDER_IMAGE}"}

  RPM_BUILD_VOL=${RPM_BUILD_VOL:-custom-rpms}

  # Create the persistent container volume
  if [ -z "$($KUBEVIRT_CRI volume list | grep ${RPM_BUILD_VOL})" ]; then
      $KUBEVIRT_CRI volume create ${RPM_BUILD_VOL}
  fi

  selinux_bind_options=",z"
  # Using Podman and MacOS and 'z' bind option may not work correctly.
  # See: https://github.com/containers/podman/issues/13631
  if [[ $KUBEVIRT_CRI = podman* ]] && [[ "$(uname -s)" == "Darwin" ]]; then
      selinux_bind_options=""
  fi

  # Make sure that the output directory exists on both sides
  if [[ ! -z "$OUT_DIR" ]]; then
      OUT_DIR_SRC=$(echo "$OUT_DIR" | cut -d ":" -f 1)
      OUT_DIR_DST=$(echo "$OUT_DIR" | cut -d ":" -f 2)
      $KUBEVIRT_CRI run -v "${RPM_BUILD_VOL}:/root:rw${selinux_bind_options}" --security-opt "label=disable" --rm ${BUILDER_IMAGE} mkdir -p /root/${OUT_DIR_SRC}
      mkdir -p ${OUT_DIR_DST}
  fi

  # Start an rsyncd instance and make sure it gets stopped after the script exits
  RSYNC_CID=$($KUBEVIRT_CRI run -d -v "${RPM_BUILD_VOL}:/root:rw${selinux_bind_options}" --security-opt "label=disable" --cap-add SYS_CHROOT --expose 873 -P ${BUILDER_IMAGE} /usr/bin/rsync --no-detach --daemon --verbose)

  function finish() {
      $KUBEVIRT_CRI stop --time 1 ${RSYNC_CID} >/dev/null 2>&1
      $KUBEVIRT_CRI rm -f ${RSYNC_CID} >/dev/null 2>&1
  }
  trap finish RETURN

  RSYNCD_PORT=$($KUBEVIRT_CRI port $RSYNC_CID 873 | cut -d':' -f2)

  rsynch_fail_count=0

  while ! rsync ${KUBEVIRT_DIR}/${RSYNCTEMP} "rsync://root@127.0.0.1:${RSYNCD_PORT}/build/${RSYNCTEMP}" &>/dev/null; do
      if [[ "$rsynch_fail_count" -eq 0 ]]; then
          printf "Waiting for rsyncd to be ready"
          sleep .1
      elif [[ "$rsynch_fail_count" -lt 30 ]]; then
          printf "."
          sleep 1
      else
          printf "failed"
          exit 1
      fi
      rsynch_fail_count=$((rsynch_fail_count + 1))
  done

  printf "\n"

  rsynch_fail_count=0

  _rsync() {
      # Preserve permissions, but change ownership to destination
      rsync -rlpt "$@"
  }

  IFS="," read -r -a sources <<< "$SRCS"

  for item in "${sources[@]}"; do
      if [[ ! -z "$item" ]]; then
          # Copy source into the persistent container volume
          _rsync \
              --delete \
              $item \
              "rsync://root@127.0.0.1:${RSYNCD_PORT}/build/"
      fi
  done

  volumes="${EXTRA_VOLS}"

  # Add build volume mount (directory root for source in the container)
  volumes="$volumes -v ${RPM_BUILD_VOL}:/build:rw${selinux_bind_options}"

  # append .docker secrets directory as volume
  mkdir -p "${HOME}/.docker/secrets"
  volumes="$volumes -v ${HOME}/.docker/secrets:/root/.docker/secrets:ro${selinux_bind_options}"

  # Use a bind-mount to expose docker/podman auth file to the container
  if [[ $KUBEVIRT_CRI = podman* ]] && [[ -f "${XDG_RUNTIME_DIR}/containers/auth.json" ]]; then
      volumes="$volumes --mount type=bind,source=${XDG_RUNTIME_DIR}/containers/auth.json,target=/root/.docker/config.json,readonly"
  elif [[ -f "${HOME}/.docker/config.json" && "$(cat ${HOME}/.docker/config.json | jq 'has("credHelpers")')" != "true" ]]; then
      volumes="$volumes --mount type=bind,source=${HOME}/.docker/config.json,target=/root/.docker/config.json,readonly"
  fi

  # add custom docker certs, if needed
  if [ -n "$DOCKER_CA_CERT_FILE" ] && [ -f "$DOCKER_CA_CERT_FILE" ]; then
      volumes="$volumes -v ${DOCKER_CA_CERT_FILE}:${DOCKERIZED_CUSTOM_CA_PATH}:ro${selinux_bind_options}"
  fi

  BUILD_CID=$($KUBEVIRT_CRI run -dit ${volumes} -w /build ${BUILDER_IMAGE})
  function finish() {
      $KUBEVIRT_CRI stop --time 1 ${RSYNC_CID} >/dev/null 2>&1
      $KUBEVIRT_CRI rm -f ${RSYNC_CID} >/dev/null 2>&1
      $KUBEVIRT_CRI stop --time 1 ${BUILD_CID} >/dev/null 2>&1
      $KUBEVIRT_CRI rm -f ${BUILD_CID} >/dev/null 2>&1
  }

  function copy_out() {
      if [[ ! -z "$OUT_DIR" ]]; then
          echo "Copying output to ${OUT_DIR_DST}"
          _rsync --delete "rsync://root@127.0.0.1:${RSYNCD_PORT}/build/${OUT_DIR_SRC}/" ${OUT_DIR_DST}
      fi
  }

  # Run the command
  test -t 1 && USE_TTY="-t"
  if ! $KUBEVIRT_CRI exec ${CONTAINER_ENV} ${USE_TTY} ${BUILD_CID} $1; then
      # Copy the build output out of the container, make sure that _out exactly matches the build result
      copy_out
      exit 1
  fi

  copy_out
}

(
  # Shutdown the old rpm server before building (ignore failures)
  ${KUBEVIRT_CRI} kill rpms-http-server &> /dev/null
  ${KUBEVIRT_CRI} rm -f rpms-http-server &> /dev/null

  set -e

  # Pulled from custom-rpms.md (creates a shared docker volume for built RPMs)
  ${KUBEVIRT_CRI} volume create rpms
  # Build the CentOS Stream 9 image with some extra dependencies
  ${KUBEVIRT_CRI} build -t rpm-build-image -f ${SCRIPT_DIR}/Dockerfile . 
  # Mount the rpm volume to the default build output directory for libvirt RPMs
  EXTRA_VOLS="-v rpms:/root/rpmbuild/RPMS"

  RPM_VERSION_STRING=""

  if [[ -z "$LIBVIRT_ONLY" ]]; then
    if [[ -z "$QEMU_DIR" ]]; then
        echo "The variable 'QEMU_DIR' is not defined."
        echo "Defined the value with the QEMU source directory or disable QEMU builds with `export LIBVIRT_ONLY=1`"
        exit 1
    fi

    if [[ -z "$QEMU_KVM_DIR" ]]; then
        echo "The variable 'QEMU_KVM_DIR' is not defined."
        echo "Define the value with the qemu-kvm source directory or disable QEMU builds with `export LIBVIRT_ONLY=1`"
        exit 1
    fi
    # This directory will capture the version file from the containerized RPM build
    OUT_DIR="output:${QEMU_DIR}/build"

    SRCS="${QEMU_DIR},${QEMU_KVM_DIR},${SCRIPT_DIR}/build-qemu.bash"
    # Pass the container-side path to the script (it will be rsynced to the cwd in the container)
    dockerized ./build-qemu.bash

    # Get the qemu version number from build data (Format is "EPOCH:VERSION-RELEASE")
    RPM_VERSION_STRING="${RPM_VERSION_STRING} QEMU_VERSION=${QEMU_VERSION:-$(cat ${QEMU_DIR}/build/version.txt).el9}"
  fi

  if [[ -z "$QEMU_ONLY" ]]; then
    if [[ -z "$LIBVIRT_DIR" ]]; then
        echo "The variable 'LIBVIRT_DIR' is not defined."
        echo "Define the value with the libvirt source directory or disable libvirt builds with `export QEMU_ONLY=1`"
        exit 1
    fi

    OUT_DIR="libvirt/build:${LIBVIRT_DIR}/build"

    SRCS="${LIBVIRT_DIR},${SCRIPT_DIR}/build-libvirt.bash"

    # Pass the container-side path to the script (it will be rsynced to the cwd in the container)
    dockerized ./build-libvirt.bash

    # Get the libvirt version number from meson introspection data (Format is "EPOCH:VERSION-RELEASE")
    RPM_VERSION_STRING="${RPM_VERSION_STRING} LIBVIRT_VERSION=${LIBVIRT_VERSION:-0:$(cat ${LIBVIRT_DIR}/build/version.txt)-1.el9}"
  fi


  # Mount the same rpm volume to an httpd container, and output it's IP to the "${KUBEVIRT_DIR}/hack/custom-rpms/generated/custom-repo.yaml" file
  # This file MUST be in the kubevirt source directory in a non-ignored location (see rsync commands in ${KUBEVIRT_DIR}/hack/dockerized), 
  # otherwise it will not be copied into the build container.
  ${KUBEVIRT_CRI} run -dit --name rpms-http-server -p 80 -v rpms:/usr/local/apache2/htdocs/ httpd:latest
  DOCKER_URL=$(${KUBEVIRT_CRI} inspect -f '{{range.NetworkSettings.Networks}}{{.IPAddress}}{{end}}' rpms-http-server)
  mkdir -p ${KUBEVIRT_DIR}/hack/custom-rpms/generated
  sed "s|DOCKER_URL|$DOCKER_URL|g" ${SCRIPT_DIR}/custom-repo.yaml > ${KUBEVIRT_DIR}/hack/custom-rpms/generated/custom-repo.yaml 

  # Run the `make rpm-deps` command, adding our rpm host as a repo and setting other args as described in the building-libvirt doc 
  pushd ${KUBEVIRT_DIR}
  make CUSTOM_REPO=hack/custom-rpms/generated/custom-repo.yaml ${RPM_VERSION_STRING} SINGLE_ARCH="x86_64" rpm-deps
  popd
)