FROM fedora:28

RUN dnf install -y qemu-img && dnf clean all

RUN mkdir /data

COPY ./cdi-uploadserver /usr/bin/cdi-uploadserver

ENTRYPOINT [ "/usr/bin/cdi-uploadserver", "-alsologtostderr"]
