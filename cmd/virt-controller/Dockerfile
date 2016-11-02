FROM centos:7.2.1511

COPY kubernetes.repo /etc/yum.repos.d/kubernetes.repo

RUN yum -y install kubectl \
    && yum -y clean all

COPY virt-controller /virt-controller

COPY templates /templates

ENTRYPOINT [ "/virt-controller" ]
