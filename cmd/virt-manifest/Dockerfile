FROM centos:7

# The jmliger-virt7-upstream-epel-7 has updated libvirt binaries. Since
# the statically linked libvirt-go code requires a runtime version of libvirt
# no-lower than what was used when the binary was built.
RUN yum -y install wget && \
    wget -P /etc/yum.repos.d https://copr.fedorainfracloud.org/coprs/jmliger/virt7-upstream/repo/epel-7/jmliger-virt7-upstream-epel-7.repo  \
    && yum -y install libvirt-client \
    && yum -y clean all

COPY virt-manifest /virt-manifest

ENTRYPOINT [ "/virt-manifest" ]
