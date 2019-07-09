FROM centos:7.6.1810

COPY tests/kubernetes.repo /etc/yum.repos.d/kubernetes.repo

RUN yum install -y kubectl --disableexcludes=kubernetes && \ 
    yum clean all && \
    rm -rf /var/cache/yum

COPY tests/test.sh /test.sh
COPY deploy/deploy.sh /deploy.sh

ENTRYPOINT /test.sh
