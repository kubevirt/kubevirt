FROM fedora:28

RUN mkdir -p /tmp/shared /tmp/source

RUN yum install -y qemu-img qemu-block-curl && dnf clean all

COPY cdi-func-test-file-host-init /usr/bin/

RUN chmod u+x /usr/bin/cdi-func-test-file-host-init

COPY tinyCore.iso /tmp/source/tinyCore.iso

ENTRYPOINT ["cdi-func-test-file-host-init", "-alsologtostderr"]
