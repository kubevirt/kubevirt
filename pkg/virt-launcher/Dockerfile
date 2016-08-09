FROM alpine

RUN apk --no-cache add libvirt-client

COPY virt-launcher /virt-launcher

ENTRYPOINT [ "/virt-launcher" ]

CMD [ "--domain-path=/domain.xml", "--libvirt-uri=qemu:///system"  ]
