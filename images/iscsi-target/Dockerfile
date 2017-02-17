FROM alpine

MAINTAINER "Fabian Deutsch" <fabiand@redhat.com>
ENV container docker

RUN apk update
RUN apk add targetcli

ADD run-targetcli.sh /
RUN chmod a+x /run-targetcli.sh

ADD saveconfig.json /etc/target/saveconfig.json
#EXPOSE 16509

CMD ["/run-targetcli.sh"]
