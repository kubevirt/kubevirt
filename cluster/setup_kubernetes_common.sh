#/bin/bash -xe

export DOCKER_PRIV_REG='10.35.37.2:5000'

setenforce 0
sed -i "s/^SELINUX=.*/SELINUX=permissive/" /etc/selinux/config

systemctl stop firewalld NetworkManager || :
systemctl disable firewalld NetworkManager || :
# Make sure the firewall is never enabled again
# Enabling the firewall destroys the iptable rules
yum -y remove NetworkManager firewalld

yum install -y bridge-utils

# Install epel
yum -y install https://dl.fedoraproject.org/pub/epel/epel-release-latest-7.noarch.rpm
yum -y install jq

# if there is a second disk, use it for docker
if ls /dev/*db ; then
# We use the loopback docker dm support, and not a VG for now
  mkdir -p /var/lib/docker/
  restorecon -r /var/lib/docker
  mount LABEL=dockerdata /var/lib/docker/ || {
    mkfs.xfs -L dockerdata -f /dev/?db
  }
  # FAILS because of vdsms multpoath stuff
  #echo -e "\nLABEL=dockerdata /var/lib/docker/ xfs defaults 0 0" >> /etc/fstab
  mkdir -p /etc/systemd/system/docker.service.d/
  cat > /etc/systemd/system/docker.service.d/mount.conf <<EOT
[Service]
ExecStartPre=/usr/bin/sleep 5
ExecStartPre=-/usr/bin/mount LABEL=dockerdata /var/lib/docker
MountFlags=shared
EOT
  mount LABEL=dockerdata /var/lib/docker/
fi

# To get the qemu user and libvirt
yum install -y qemu-common qemu-kvm qemu-system-x86 libvirt || :

yum install -y docker || :

sed -i "s@OPTIONS='--selinux-enabled --log-driver=journald'@OPTIONS='--selinux-enabled --log-driver=journald --insecure-registry ${DOCKER_PRIV_REG} -G qemu'@" /etc/sysconfig/docker

systemctl restart docker
systemctl enable docker

cat <<EOT > /etc/systemd/system/kubelet.service
[Unit]
Description=Kubernetes Kubelet
Documentation=https://github.com/kubernetes/kubernetes
Wants=vdsmd.service
After=vdsmd.service

[Service]
ExecStart=/usr/bin/kubelet \
  --api-servers=http://192.168.200.2:8080 \
  --register-node=true \
  --allow-privileged=true \
  --config=/etc/kubernetes/manifests \
  --v=2
Restart=on-failure
RestartSec=5

[Install]
WantedBy=multi-user.target
EOT

yum -y install kubernetes-node kubernetes-client

# Disble sasl for libvirt. VDSM configured that
sed -i '/^auth_unix_rw/c\auth_unix_rw="none"' /etc/libvirt/libvirtd.conf
systemctl restart libvirtd
systemctl enable libvirtd

# Allow qemu passwordless sudo
cat >> /etc/sudoers.d/55-kubevirt <<EOF
# hack to allow sudo without tty and password
Defaults !requiretty
Defaults closefrom_override
%wheel	ALL=(ALL)	NOPASSWD: ALL
EOF

# Now add qemu to wheel
usermod -G wheel -a qemu

# Kubelet deployment path
mkdir -p /etc/kubernetes/manifests

# Install qemu hack
ln -s /vagrant/pkg/virt-launcher/qemu-kube /usr/local/bin/qemu-x86_64
