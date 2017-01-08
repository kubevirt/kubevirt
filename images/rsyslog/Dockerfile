FROM alpine:3.5
MAINTAINER "Roman Mohr" <rmohr@redhat.com>

RUN  apk update \
  && apk add rsyslog \
  && rm -rf /var/cache/apk/*

EXPOSE 514 514/udp

ENTRYPOINT [ "rsyslogd", "-n" ]

COPY rsyslog.conf /
