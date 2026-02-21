FROM quay.io/centos/centos:stream9

RUN dnf config-manager --set-enabled crb
RUN dnf update -y && dnf builddep -y libvirt qemu-kvm

RUN dnf install -y \
    rsync \
    createrepo_c \
    jq \
    libpmem-devel \
    libepoxy-devel \
    mesa-libgbm-devel \
    libdrm-devel \
    rpmdevtools \
    rpm-build

# Basic rsync config file that configures the module "build" at path "/root"
RUN cat <<EOF > /etc/rsyncd.conf
read only = no
uid = root
gid = root
[build] 
    path = /root
    comment = Libvirt/QEMU source
EOF