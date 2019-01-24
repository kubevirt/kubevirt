FROM fedora:28

RUN mkdir -p /tmp/shared /tmp/source

RUN dnf install -y qemu-img qemu-block-curl && dnf clean all

COPY cdi-func-test-registry-init /usr/bin/

RUN chmod u+x /usr/bin/cdi-func-test-registry-init

COPY tinyCore.iso /tmp/source/tinyCore.iso


RUN mkdir -p /tmp/source/certs
RUN dnf install -y openssl
RUN  openssl req \
	  -newkey rsa:4096 -nodes -sha256 -keyout /tmp/source/certs/domain.key \
	  -x509 -days 365  \
          -subj "/C=GB/ST=TLV/L=TLV/O=RedHat/OU=CDI/CN=cdi-docker-registry-host.cdi" \
	  -out /tmp/source/certs/domain.crt

ENTRYPOINT ["cdi-func-test-registry-init", "-alsologtostderr"]

