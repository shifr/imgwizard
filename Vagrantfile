# -*- mode: ruby -*-
# vi: set ft=ruby :

VAGRANTFILE_API_VERSION = "2"

Vagrant.configure(VAGRANTFILE_API_VERSION) do |config|
  config.vm.provider "virtualbox" do |v|
    v.memory = 512
    v.cpus = 2
  end

  config.vm.define "imgwizard", primary: true do |img|
      img.vm.box = "ubuntu/trusty64"
      img.vm.network "forwarded_port", guest: 8070, host: 8070
      img.vm.hostname = "imgwizard"
      img.vm.synced_folder "./", "/home/vagrant/imgwizard"
  end

  config.vm.provision "ansible" do |ansible|
    ansible.playbook = "ansible/provision.yml"
  end

end
