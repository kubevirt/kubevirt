FROM centos:7

RUN yum -y install wget && \
    wget -P /etc/yum.repos.d https://copr.fedorainfracloud.org/coprs/jmliger/virt7-upstream/repo/epel-7/jmliger-virt7-upstream-epel-7.repo  \
    && yum -y install libvirt-client \
    && yum -y clean all

COPY virt-handler /virt-handler

ENTRYPOINT [ "/virt-handler" ]
