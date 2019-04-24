FROM fedora:29

MAINTAINER "The KubeVirt Project" <kubevirt-dev@googlegroups.com>

ENV container docker

COPY fedora.repo /tmp/fedora_ci.dnf.conf

RUN mv /etc/dnf/dnf.conf /etc/dnf/dnf.conf.org && \
    mv /tmp/fedora_ci.dnf.conf /etc/dnf/dnf.conf && \
	dnf -y update && dnf -y install nginx && dnf clean all -y && \
	mv /etc/dnf/dnf.conf /tmp/fedora_ci.dnf.conf && \
	mv /etc/dnf/dnf.conf.org /etc/dnf/dnf.conf

ARG IMAGE_DIR=/usr/share/nginx/html/images

RUN mkdir -p $IMAGE_DIR/priv

RUN mkdir -p $IMAGE_DIR

RUN rm -f /etc/nginx/nginx.conf

COPY nginx.conf /etc/nginx/

COPY htpasswd /etc/nginx/

EXPOSE 80

EXPOSE 81

EXPOSE 82

ENTRYPOINT nginx
