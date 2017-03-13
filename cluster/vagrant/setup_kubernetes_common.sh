#/bin/bash -xe

setenforce 0
sed -i "s/^SELINUX=.*/SELINUX=permissive/" /etc/selinux/config

systemctl stop firewalld NetworkManager || :
systemctl disable firewalld NetworkManager || :
# Make sure the firewall is never enabled again
# Enabling the firewall destroys the iptable rules
yum -y remove NetworkManager firewalld

# Needed for kubernetes service routing and dns
# https://github.com/kubernetes/kubernetes/issues/33798#issuecomment-250962627
sysctl -w net.bridge.bridge-nf-call-iptables=1
sysctl -w net.bridge.bridge-nf-call-ip6tables=1

# Install epel
yum -y install https://dl.fedoraproject.org/pub/epel/epel-release-latest-7.noarch.rpm
yum -y install jq

yum -y install bind-utils net-tools

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

cat <<EOF > /etc/yum.repos.d/kubernetes.repo
[kubernetes]
name=Kubernetes
baseurl=http://yum.kubernetes.io/repos/kubernetes-el7-x86_64
enabled=1
gpgcheck=1
repo_gpgcheck=1
gpgkey=https://packages.cloud.google.com/yum/doc/yum-key.gpg
       https://packages.cloud.google.com/yum/doc/rpm-package-key.gpg
EOF
yum install --nogpgcheck -y docker kubelet kubeadm kubectl kubernetes-cni

# To get the qemu user and libvirt
yum install -y qemu-common qemu-kvm qemu-system-x86 libcgroup-tools libvirt || :

systemctl enable docker && systemctl start docker
systemctl enable kubelet && systemctl start kubelet

# Disable libvirt cgroup management
echo "cgroup_controllers = [ ]" >> /etc/libvirt/qemu.conf

# Let libvirt listen on TCP for migrations
echo 'LIBVIRTD_ARGS="--listen"' >> /etc/sysconfig/libvirtd

cat << EOT >>/etc/libvirt/libvirtd.conf
listen_tcp = 1
tcp_port = "16509"
auth_tcp = "none"
listen_addr = "0.0.0.0"
listen_tls = 0
EOT

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

# Install qemu hack
ln -s /vagrant/cmd/virt-launcher/qemu-kube /usr/local/bin/qemu-x86_64

# Create log location for qemu hack
mkdir -p /var/log/kubevirt/
touch /var/log/kubevirt/qemu-kube.log
chown qemu:qemu /var/log/kubevirt/qemu-kube.log
