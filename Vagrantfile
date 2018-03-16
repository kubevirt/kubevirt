# -*- mode: ruby -*-
# vi: set ft=ruby :

if ARGV.first == "up" && ENV['USING_KUBE_SCRIPTS'] != 'true'
  raise Vagrant::Errors::VagrantError.new, <<END
Calling 'vagrant up' directly is not supported.  Instead, please run the following:

  export PROVIDER=vagrant-kubernetes
  make cluster-up
END
end

$provider = ENV['PROVIDER'] || "vagrant-kubernetes"
$use_nfs = ENV['VAGRANT_USE_NFS'] == 'true'
$use_rng = ENV['VAGRANT_USE_RNG'] == 'true'
$cache_docker = ENV['VAGRANT_CACHE_DOCKER'] == 'true'
$cache_rpm = ENV['VAGRANT_CACHE_RPM'] == 'true'
$nodes = (ENV['VAGRANT_NUM_NODES'] || 0).to_i
$vagrant_pool = (ENV['VAGRANT_POOL'] unless
                  (ENV['VAGRANT_POOL'].nil? or ENV['VAGRANT_POOL'].empty?))
# Used for matrix builds to similar setups on the same node without vagrant
# machine name clashes.
$libvirt_prefix = ENV['TARGET'] || "kubevirt"

$config = Hash[*File.read('hack/config-default.sh').split(/=|\n/)]
if File.file?('hack/config-local.sh') then
  $localConfig = Hash[*File.read('hack/config-local.sh').split(/=|\n/)]
  $config = $config.merge($localConfig)
end

$master_ip = $config["master_ip"]
$network_provider = $config["network_provider"]

$common_setup = <<SCRIPT
#!/bin/bash
set -xe
sed -i -e "s/PasswordAuthentication no/PasswordAuthentication yes/" /etc/ssh/sshd_config
systemctl restart sshd
# FIXME, sometimes eth1 does not come up on Vagrant on latest fc26
sudo ifup eth1
SCRIPT

Vagrant.configure(2) do |config|
  config.vm.box = "centos/7"
  config.vm.box_url = "http://cloud.centos.org/centos/7/vagrant/x86_64/images/CentOS-7-x86_64-Vagrant-1802_01.Libvirt.box"

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
      if $vagrant_pool then
          domain.storage_pool_name = $vagrant_pool
      end
      domain.default_prefix = $libvirt_prefix
  end

  if $use_nfs then
    config.vm.synced_folder "./", "/vagrant", type: "nfs"
  else
    # Vagrant seems to insist on using NFS sometimes even when explicitly
    # configured to use `rsync`, this prevents that
    config.nfs.functional = false
    config.vm.synced_folder "./", "/vagrant", type: "rsync",
      rsync__exclude: [
        "cluster/vagrant/.kubectl", "cluster/vagrant/.kubeconfig", ".vagrant",
        "vendor", ".git"
      ],
      rsync__args: ["--archive", "--delete"]
  end

  config.vm.provision "shell", inline: $common_setup

  config.vm.define "master" do |master|
      master.vm.hostname = "master"
      master.vm.network "private_network", ip: "#{$master_ip}", libvirt__network_name: $libvirt_prefix + "0"
      master.vm.provider :libvirt do |domain|
          domain.memory = 3000
          if $cache_docker then
            domain.storage :file, :size => '10G', :path => $libvirt_prefix.to_s + '_master_docker.img', :allow_existing => true
          end
      end

      master.vm.provision "shell" do |s|
        s.path = "cluster/#{$provider}/setup_master.sh"
        s.args = ["#{$master_ip}", "#{$nodes}", "#{$network_provider}"]
      end
  end

  (0..($nodes-1)).each do |suffix|
    config.vm.define "node" + suffix.to_s do |node|
      node.vm.hostname = "node" + suffix.to_s
      node_ip = $master_ip[0..-2] + ($master_ip[-1].to_i + 1 + suffix).to_s
      node.vm.network "private_network", ip: node_ip, libvirt__network_name: $libvirt_prefix + "0" 

      node.vm.provider :libvirt do |domain|
        domain.memory = 2048
        if $cache_docker then
          domain.storage :file, :size => '10G', :path => $libvirt_prefix.to_s + '_node_docker' + suffix.to_s + '.img', :allow_existing => true
        end
      end

      node.vm.provision "shell" do |s|
        s.path = "cluster/#{$provider}/setup_node.sh"
        s.args = ["#{$master_ip}", "#{$nodes}"]
      end
    end
  end
end
