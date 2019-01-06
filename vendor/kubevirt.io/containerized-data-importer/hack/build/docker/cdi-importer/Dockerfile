FROM fedora:28

RUN dnf install -y qemu-img qemu-block-curl skopeo && dnf clean all

RUN mkdir /data

COPY ./cdi-importer /usr/bin/cdi-importer

ENTRYPOINT ["/usr/bin/cdi-importer", "-alsologtostderr"]
