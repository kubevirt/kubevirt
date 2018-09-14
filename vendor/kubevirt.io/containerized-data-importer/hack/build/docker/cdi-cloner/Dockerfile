FROM fedora:28
ADD cloner_startup.sh /usr/bin/cloner_startup.sh
RUN chmod +x /usr/bin/cloner_startup.sh

ENTRYPOINT [ "/usr/bin/cloner_startup.sh" ]
