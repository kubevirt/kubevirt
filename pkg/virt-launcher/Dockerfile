FROM frolvlad/alpine-glibc

COPY virt-launcher /virt-launcher

ENTRYPOINT [ "/virt-launcher" ]

CMD [ "--domain-path=/domain.xml", "--libvirt-uri='qemu:///system'"  ]
