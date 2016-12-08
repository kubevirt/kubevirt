# -*- mode: ruby -*-
# vi: set ft=ruby :


$use_nfs = ENV['VAGRANT_USE_NFS'] == 'true'
$use_rng = ENV['VAGRANT_USE_RNG'] == 'true'
$cache_docker = ENV['VAGRANT_CACHE_DOCKER'] == 'true'
$cache_rpm = ENV['VAGRANT_CACHE_RPM'] == 'true'
$master_ip = ENV['MASTER_IP'].nil? ? "192.168.200.2" : ENV['MASTER_IP']
$network_provider = ENV['NETWORK_PROVIDER'].nil? ? "weave" : ENV['NETWORK_PROVIDER']

Vagrant.configure(2) do |config|
  config.vm.box = "centos7"
  config.vm.box_url = "http://cloud.centos.org/centos/7/vagrant/x86_64/images/CentOS-7-x86_64-Vagrant-1608_01.LibVirt.box"

  if Vagrant.has_plugin?("vagrant-cachier") and $cache_rpm then
      config.cache.scope = :machine
      config.cache.auto_detect = false
      config.cache.enable :yum
  end

  config.vm.provider :libvirt do |domain|
      domain.cpus = 2
      domain.nested = true  # enable nested virtualization
      domain.cpu_mode = "host-model"

      if $use_rng then
          # Will be part of vagrant-libvirt 0.0.36:
          #     https://github.com/vagrant-libvirt/vagrant-libvirt/pull/654
          libvirt.random :model => 'random' # give ovirt-engine some random data for SSO
      end
  end

  if $use_nfs then
    config.vm.synced_folder "./", "/vagrant", type: "nfs"
  else
    config.vm.synced_folder "./", "/vagrant", type: "rsync", rsync__exclude: [ "cluster/.kubectl", "cluster/.kubeconfig" ]
  end

  config.vm.provision "shell", inline: <<-SHELL
    #!/bin/bash
    set -xe
    sed -i -e "s/PasswordAuthentication no/PasswordAuthentication yes/" /etc/ssh/sshd_config
    systemctl restart sshd
  SHELL

  config.vm.define "master" do |master|
      master.vm.hostname = "master"
      master.vm.network "private_network", ip: "#{$master_ip}"
      master.vm.provider :libvirt do |domain|
          domain.memory = 3000
          if $cache_docker then
              domain.storage :file, :size => '10G', :path => 'kubevirt_master_docker.img', :allow_existing => true
          end
      end

      master.vm.provision "shell", inline: <<-SHELL
        #!/bin/bash
        set -xe
        export WITH_LOCAL_NFS=true
        export KUBERNETES_MASTER=true
        export VM_IP=#{$master_ip}
        export MASTER_IP=#{$master_ip}
        export NETWORK_PROVIDER=#{$network_provider}
        cd /vagrant/cluster
        bash setup_kubernetes_master.sh
        set +x
        echo -e "\033[0;32m Deployment was successful!"
        echo -e "Cockpit is accessible at https://192.168.200.2:9090."
        echo -e "Credentials for Cockpit are 'root:vagrant'.\033[0m"
      SHELL
  end
  config.vm.define "node" do |node|
      node.vm.hostname = "node"
      node.vm.provider :libvirt do |domain|
          domain.memory = 2048
          if $cache_docker then
              domain.storage :file, :size => '10G', :path => 'kubevirt_node_docker.img', :allow_existing => true
          end
      end

      node.vm.provision "shell", inline: <<-SHELL
        #!/bin/bash
        set -xe
        export MASTER_IP=#{$master_ip}
        cd /vagrant/cluster
        bash setup_kubernetes_node.sh
        set +x
        echo -e "\033[0;32m Deployment was successful!\033[0m"
      SHELL
  end
end
