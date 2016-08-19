FROM centos:7.2.1511

RUN yum -y install libvirt-client && yum -y clean all

COPY virt-launcher /virt-launcher

ENTRYPOINT [ "/virt-launcher" ]

CMD [ "--domain-path=/domain.xml", "--libvirt-uri=qemu:///system"  ]
