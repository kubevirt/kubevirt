FROM centos:7.2.1511

RUN yum -y localinstall http://resources.ovirt.org/pub/yum-repo/ovirt-release40.rpm \
    && yum -y localinstall https://dl.fedoraproject.org/pub/epel/epel-release-latest-7.noarch.rpm \
    && yum -y install libvirt-client \
    && yum -y clean all

COPY virt-launcher /virt-launcher

ENTRYPOINT [ "/virt-launcher" ]

CMD [ "--domain-path=/domain.xml", "--libvirt-uri=qemu:///system"  ]
