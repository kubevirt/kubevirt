FROM kubevirt/libvirt

MAINTAINER "Stu Gott" <sgott@redhat.com>
ENV container docker

# util-linux listed here explicity as nsenter must be present in this image
RUN yum install -y \
  util-linux \
  libcgroup-tools \
  ethtool \
  sudo \
  docker && yum -y clean all

COPY qemu-kube /usr/local/bin/qemu-x86_64
RUN chmod a+x /usr/local/bin/qemu-x86_64

COPY kubevirt-sudo /etc/sudoers.d/kubevirt
RUN chmod 0640 /etc/sudoers.d/kubevirt

# libvirtd.sh in this image differs from upstream
RUN rm -f /libvirtd.sh
COPY libvirtd.sh /libvirtd.sh
RUN chmod a+x /libvirtd.sh

CMD ["/libvirtd.sh"]
