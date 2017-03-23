curl -O https://raw.githubusercontent.com/vagrant-libvirt/vagrant-libvirt/master/tools/create_box.sh
touch Vagrantfile
sudo bash create_box.sh /var/lib/libvirt/images/kubevirt_master.img kubevirt_master.box
sudo bash create_box.sh /var/lib/libvirt/images/kubevirt_node0.img kubevirt_node0.box
