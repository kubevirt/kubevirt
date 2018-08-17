FROM fedora:27

RUN dnf install -y qemu-img

RUN mkdir /data

COPY ./cdi-importer /usr/bin/cdi-importer

ENTRYPOINT ["/usr/bin/cdi-importer", "-alsologtostderr"]
