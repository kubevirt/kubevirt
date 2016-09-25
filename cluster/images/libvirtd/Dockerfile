FROM centos:7

MAINTAINER "Fabian Deutsch" <fabiand@redhat.com>
ENV container docker

RUN yum install -y \
  libvirt-daemon-kvm \
  libvirt-daemon-qemu \
  libvirt-client \
  selinux-policy selinux-policy-targeted \
  augeas

COPY augconf /augconf
RUN augtool -f /augconf

COPY libvirtd.sh /libvirtd.sh
RUN chmod a+x /libvirtd.sh
#EXPOSE 16509

CMD ["/libvirtd.sh"]
