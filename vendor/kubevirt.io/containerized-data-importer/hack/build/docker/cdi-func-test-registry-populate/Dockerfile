FROM fedora:28

RUN mkdir -p /tmp/shared /tmp/source
RUN dnf install -y buildah

ADD populate-registry.sh /
RUN chmod u+x /populate-registry.sh

ENTRYPOINT ["./populate-registry.sh"]
