FROM centos

ENV container docker

RUN yum install -y psmisc

COPY vm-killer.sh /vm-killer.sh
RUN chmod a+x /vm-killer.sh
