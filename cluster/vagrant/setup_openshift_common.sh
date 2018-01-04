yum install -y centos-release-openshift-origin
yum install -y wget git net-tools bind-utils iptables-services bridge-utils bash-completion kexec-tools sos psacct docker
systemctl start docker
systemctl enable docker
yum -y update
yum --enablerepo=centos-openshift-origin-testing install -y atomic-openshift-utils
