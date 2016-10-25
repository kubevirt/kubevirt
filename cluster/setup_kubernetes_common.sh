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
yum install -y qemu-common qemu-kvm qemu-system-x86 libcgroup-tools libvirt || :

yum install -y docker || :

sed -i "s@^OPTIONS=.*@OPTIONS='--selinux-enabled --log-driver=journald --insecure-registry ${DOCKER_PRIV_REG} -G qemu'@" /etc/sysconfig/docker

cat <<EOT > /etc/systemd/system/kubelet.service
[Unit]
Description=Kubernetes Kubelet
Documentation=https://github.com/kubernetes/kubernetes
Wants=docker.service
After=docker.service

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

yum -y install flannel
echo 'FLANNEL_OPTIONS="-iface=eth1"' >> /etc/sysconfig/flanneld
sed -i /etc/sysconfig/flanneld  -e  "s@FLANNEL_ETCD=.*@FLANNEL_ETCD=\"http://${MASTER_IP}:2379\"@"

if ${KUBERNETES_MASTER:-false}; then
yum -y install etcd

sed -i /etc/etcd/etcd.conf -e "s@.*ETCD_ADVERTISE_CLIENT_URLS=.*@@"
sed -i /etc/etcd/etcd.conf -e "s@.*ETCD_LISTEN_CLIENT_URLS=.*@ETCD_LISTEN_CLIENT_URLS=\"http://localhost:2379,http://${MASTER_IP}:2379\"@"

systemctl enable etcd
systemctl start etcd

sleep 2
cat flannel.conf | etcdctl set /atomic.io/network/config
fi

yum -y install kubernetes-node kubernetes-client

systemctl enable flanneld
systemctl start flanneld # this will block until it can read a network config
systemctl restart docker
systemctl enable docker

# Disable libvirt cgroup management
echo "cgroup_controllers = [ ]" >> /etc/libvirt/qemu.conf

# Disble sasl for libvirt. VDSM configured that
sed -i '/^auth_unix_rw/c\auth_unix_rw="none"' /etc/libvirt/libvirtd.conf
systemctl restart libvirtd
systemctl enable libvirtd

# Define macvtap network interface for libvirt
virsh net-define libvirt_network.xml
virsh net-autostart kubevirt-net
virsh net-start kubevirt-net

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
ln -s /vagrant/cmd/virt-launcher/qemu-kube /usr/local/bin/qemu-x86_64

# Create log location for qemu hack
mkdir -p /var/log/kubevirt/
touch /var/log/kubevirt/qemu-kube.log
chown qemu:qemu /var/log/kubevirt/qemu-kube.log
