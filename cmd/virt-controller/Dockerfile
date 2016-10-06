FROM centos:7.2.1511

RUN yum -y install kubernetes-client \
    && yum -y clean all

COPY virt-controller /virt-controller

COPY templates /templates

ENTRYPOINT [ "/virt-controller" ]
