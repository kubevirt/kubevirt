FROM fedora:28

RUN yum -y install \
  qemu \
  && dnf -y clean all && \
  test $(id -u qemu) = 107 # make sure that the qemu user really is 107

COPY share_disk.sh /

ENTRYPOINT [ "/share_disk.sh" ]
