FROM centos:7

RUN yum -y install libvirt-client \
    && yum -y clean all

COPY virt-handler /virt-handler

ENTRYPOINT [ "/virt-handler" ]
