FROM fedora:26

MAINTAINER "The KubeVirt Project" <kubevirt-dev@googlegroups.com>

EXPOSE 3128

RUN dnf -y install squid \
    && dnf -y clean all

RUN sed -i -e "s/http_access deny CONNECT !SSL_ports/http_access deny CONNECT !Safe_ports/" /etc/squid/squid.conf
RUN echo "pid_filename /home/proxy/run/squid.pid" >> /etc/squid/squid.conf

RUN useradd --create-home -s /bin/bash proxy
RUN chown -R proxy /var/log/squid && chown -R proxy /var/spool/squid
RUN cp /etc/squid/squid.conf /home/proxy && chown proxy /home/proxy/squid.conf
WORKDIR /home/proxy
USER proxy
RUN mkdir /home/proxy/run

CMD squid -NCd1 -f /home/proxy/squid.conf
