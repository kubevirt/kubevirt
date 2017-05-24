FROM centos

ENV container docker

RUN yum install -y http://resources.ovirt.org/releases/ovirt-release/ovirt-release-master.rpm
RUN yum install -y python-cpopen python-yaml 
RUN yum install -y vdsm-client && yum -y clean all
