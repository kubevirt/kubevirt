FROM fedora:28

MAINTAINER "The KubeVirt Project" <kubevirt-dev@googlegroups.com>

ENV container docker

RUN dnf -y update && dnf -y install nginx && dnf clean all -y

ARG IMAGE_DIR=/usr/share/nginx/html/images

RUN mkdir -p $IMAGE_DIR/priv

RUN mkdir -p $IMAGE_DIR

RUN rm -f /etc/nginx/nginx.conf

COPY nginx.conf /etc/nginx/

COPY htpasswd /etc/nginx/

EXPOSE 80

EXPOSE 81

ENTRYPOINT nginx
