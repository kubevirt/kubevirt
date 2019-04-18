FROM fedora:29

RUN mkdir -p /tmp/shared /tmp/source

COPY fedora.repo /tmp/fedora_ci.dnf.conf

RUN mv /etc/dnf/dnf.conf /etc/dnf/dnf.conf.org && \
    mv /tmp/fedora_ci.dnf.conf /etc/dnf/dnf.conf && \
	dnf install -y buildah && dnf clean all && \
	mv /etc/dnf/dnf.conf /tmp/fedora_ci.dnf.conf && \
	mv /etc/dnf/dnf.conf.org /etc/dnf/dnf.conf

ADD populate-registry.sh /
RUN chmod u+x /populate-registry.sh

ENTRYPOINT ["./populate-registry.sh"]
