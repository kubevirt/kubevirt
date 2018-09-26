FROM fedora:28

RUN dnf install -y qemu-img && dnf clean all

RUN mkdir /data

COPY ./cdi-uploadserver /cdi-uploadserver

ENTRYPOINT [ "/cdi-uploadserver", "-alsologtostderr"]
