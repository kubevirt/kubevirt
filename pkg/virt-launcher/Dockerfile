FROM centos:7.2.1511

RUN yum -y localinstall http://resources.ovirt.org/pub/yum-repo/ovirt-release40.rpm \
    && yum -y install qemu-img-ev qemu-kvm-ev qemu-kvm-tools-ev \
    && yum -y install libvirt-client

COPY virt-launcher /virt-launcher

ENTRYPOINT [ "/virt-launcher" ]

CMD [ "--domain-path=/domain.xml", "--libvirt-uri=qemu:///system"  ]
