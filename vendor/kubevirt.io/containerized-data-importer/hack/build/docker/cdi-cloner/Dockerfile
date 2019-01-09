FROM fedora:28
ADD cloner_startup.sh /usr/bin/cloner_startup.sh
RUN chmod +x /usr/bin/cloner_startup.sh

COPY ./cdi-cloner /usr/bin/cdi-cloner

ENTRYPOINT [ "/usr/bin/cloner_startup.sh" ]
