FROM debian:sid

MAINTAINER "Fabian Deutsch" <fabiand@redhat.com>
ENV container docker

RUN apt-get update && apt-get install -y \
  tgt bash curl bzip2

ADD run-tgt.sh /
RUN chmod a+x /run-tgt.sh

#EXPOSE 3260

CMD ["/run-tgt.sh"]
