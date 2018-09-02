FROM fedora:27

RUN dnf install -y qemu-img

RUN mkdir /data

COPY ./cdi-importer /cdi-importer

ENTRYPOINT ["./cdi-importer", "-alsologtostderr"]
