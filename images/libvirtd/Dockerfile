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

# Faking qemu-kube
# qemu-kube originally requires docker and sudo - also
# access to docker service (which is running on the host)
# modifying this image would be possible, but it's not
# essential for the demo.
# It should be obsoleted anyway once the virt-handler
# populates the cgroup.
# Another step is to see how qemu can be moved in a namespace.
# For reference - this was considered by upstream libvirtd (
# supporting namespaces, but had security implications, caused
# by race conditions)
RUN ln -s /usr/bin/qemu-system-x86_64 /usr/local/bin/qemu-x86_64

COPY libvirtd.sh /libvirtd.sh
RUN chmod a+x /libvirtd.sh
#EXPOSE 16509

CMD ["/libvirtd.sh"]
