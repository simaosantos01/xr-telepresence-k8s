Vagrant.configure("2") do |config|
  config.vm.box = "ubuntu/jammy64"

  config.vm.provider "virtualbox" do |vb|
    vb.memory = 2048
    vb.cpus = 2
  end

  config.vm.provision "shell", name: "install-dependencies", path: "vagrant-scripts/install-dependencies.sh"

  config.vm.define "cloud" do |cloud|
    cloud.vm.hostname = "cloud"
    cloud.vm.network "private_network", ip: "192.168.56.10"
  end

  config.vm.define "edge1" do |edge1|
    edge1.vm.hostname = "edge1"
    edge1.vm.network "private_network", ip: "192.168.56.11"
  end

  config.vm.define "edge2" do |edge2|
    edge2.vm.hostname = "edge2"
    edge2.vm.network "private_network", ip: "192.168.56.12"
  end
end