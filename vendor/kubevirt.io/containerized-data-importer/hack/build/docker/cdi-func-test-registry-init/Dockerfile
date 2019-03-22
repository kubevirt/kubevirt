FROM fedora:28

RUN mkdir -p /tmp/shared /tmp/source

RUN dnf install -y qemu-img qemu-block-curl && dnf clean all

COPY cdi-func-test-registry-init /usr/bin/

RUN chmod u+x /usr/bin/cdi-func-test-registry-init

COPY tinyCore.iso /tmp/source/tinyCore.iso

ENTRYPOINT ["cdi-func-test-registry-init", "-alsologtostderr"]

