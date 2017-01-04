FROM fedora

MAINTAINER "Stu Gott" <sgott@redhat.com>
ENV container docker

RUN yum install -y \
  libvirt-daemon-kvm \
  libvirt-daemon-qemu \
  libvirt-client \
  selinux-policy selinux-policy-targeted \
  augeas

COPY augconf /augconf
RUN augtool -f /augconf

RUN echo -n shibboleth | saslpasswd2 -p -a libvirt vdsm@ovirt

COPY libvirtd.sh /libvirtd.sh
RUN chmod a+x /libvirtd.sh
#EXPOSE 16509

CMD ["/libvirtd.sh"]
