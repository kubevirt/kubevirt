FROM centos:7

MAINTAINER "Fabian Deutsch" <fabiand@redhat.com>
ENV container docker

RUN yum install -y \
  libvirt-daemon-kvm \
  libvirt-daemon-qemu \
  libvirt-client \
  augeas

COPY augconf /augconf
RUN augtool -f /augconf

#EXPOSE 16509

CMD ["/usr/sbin/libvirtd"]
