FROM registry:2

ADD registry-config.yml /etc/docker/registry/

ADD start-registry.sh /
RUN chmod u+x /start-registry.sh

EXPOSE 443

ENTRYPOINT ["/start-registry.sh"]
