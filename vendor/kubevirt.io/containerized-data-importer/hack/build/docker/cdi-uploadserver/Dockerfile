FROM fedora:29

COPY fedora.repo /tmp/fedora_ci.dnf.conf

RUN mv /etc/dnf/dnf.conf /etc/dnf/dnf.conf.org && \
    mv /tmp/fedora_ci.dnf.conf /etc/dnf/dnf.conf && \
	dnf install -y qemu-img && dnf clean all && \
	mv /etc/dnf/dnf.conf /tmp/fedora_ci.dnf.conf && \
	mv /etc/dnf/dnf.conf.org /etc/dnf/dnf.conf

RUN mkdir /data

COPY ./cdi-uploadserver /usr/bin/cdi-uploadserver

ENTRYPOINT [ "/usr/bin/cdi-uploadserver", "-alsologtostderr"]
