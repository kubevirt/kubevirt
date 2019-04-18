FROM fedora:29

RUN mkdir -p /tmp/shared /tmp/source

COPY fedora.repo /tmp/fedora_ci.dnf.conf

RUN mv /etc/dnf/dnf.conf /etc/dnf/dnf.conf.org && \
    mv /tmp/fedora_ci.dnf.conf /etc/dnf/dnf.conf && \
	dnf install -y qemu-img qemu-block-curl && dnf clean all && \
	mv /etc/dnf/dnf.conf /tmp/fedora_ci.dnf.conf && \
	mv /etc/dnf/dnf.conf.org /etc/dnf/dnf.conf

COPY cdi-func-test-file-host-init /usr/bin/

RUN chmod u+x /usr/bin/cdi-func-test-file-host-init

COPY tinyCore.iso /tmp/source/tinyCore.iso

ENTRYPOINT ["cdi-func-test-file-host-init", "-alsologtostderr"]
