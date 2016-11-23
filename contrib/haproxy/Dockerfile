FROM haproxy:1.6-alpine
COPY haproxy.cfg /usr/local/etc/haproxy/haproxy.cfg

RUN cp /docker-entrypoint.sh /docker-entrypoint-orig.sh

COPY docker-entrypoint.sh /docker-entrypoint.sh
