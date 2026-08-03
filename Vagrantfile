# -*- mode: ruby -*-
# vi: set ft=ruby :
#
# 4 x Ubuntu 26.04 nodes, SSH forwarded on host ports 12221-12224.
#
# Each node gets three users (mirrors Dockerfile.test):
#   user1:Password1  -> no sudo
#   user2:Password2  -> sudo with password
#   user3:Password3  -> sudo without password
#
# Host-architecture aware: the box `bento/ubuntu-26.04` publishes separate
# arm64 and amd64 builds for every common provider (virtualbox, libvirt,
# parallels, qemu, utm, vmware_desktop). Vagrant automatically downloads
# the build that matches the host CPU, so an Apple Silicon Mac boots ARM64
# VMs and an x86_64 host boots AMD64 VMs — no separate Vagrantfile needed.

require "rbconfig"

HOST_ARCH = case RbConfig::CONFIG["host_cpu"]
            when /arm64|aarch64/ then "arm64"
            when /x86_64|amd64/  then "amd64"
            else
              # Fallback: ask the OS.
              `uname -m`.strip
            end

BOX        = "bento/ubuntu-26.04"
BOX_ARCH   = HOST_ARCH
NODES = {
  "node1" => 12221,
  "node2" => 12222,
  "node3" => 12223,
  "node4" => 12224,
}

PROVISION_SCRIPT = <<~SHELL
  set -eux
  export DEBIAN_FRONTEND=noninteractive

  apt-get update
  apt-get install -y openssh-server sudo curl

  # --- user1: no sudo ---------------------------------------------------
  useradd -m -s /bin/bash user1
  echo "user1:Password1" | chpasswd

  # --- user2: sudo with password ----------------------------------------
  useradd -m -s /bin/bash user2
  echo "user2:Password2" | chpasswd
  usermod -aG sudo user2

  # --- user3: sudo without password ------------------------------------
  useradd -m -s /bin/bash user3
  echo "user3:Password3" | chpasswd
  usermod -aG sudo user3
  echo "user3 ALL=(ALL) NOPASSWD: ALL" > /etc/sudoers.d/nopasswd-users
  chmod 440 /etc/sudoers.d/nopasswd-users

  # Allow password SSH auth for the test users (bento ships it disabled).
  echo "PasswordAuthentication yes" > /etc/ssh/sshd_config.d/99-vagrant-test.conf
  sed -i 's/^#\\?PasswordAuthentication.*/PasswordAuthentication yes/' /etc/ssh/sshd_config || true
  systemctl restart ssh 2>/dev/null || systemctl restart sshd 2>/dev/null || true
SHELL

Vagrant.configure("2") do |config|
  config.vm.box = BOX
  config.vm.box_check_update = false

  # We don't need the project dir shared into the guest; disabling it avoids
  # the vagrant-qemu plugin's interactive SMB credential prompt on macOS.
  config.vm.synced_folder ".", "/vagrant", disabled: true

  NODES.each do |name, host_port|
    config.vm.define name do |node|
      node.vm.hostname = name

      # Override the default "ssh" forwarded port so `vagrant ssh` and the
      # test harness both use 1222x on the host side.
      node.vm.network "forwarded_port",
                      guest: 22,
                      host:  host_port,
                      id:    "ssh",
                      auto_correct: false

      node.vm.provider "virtualbox" do |vb|
        vb.name   = "ubuntu26-#{name}"
        vb.memory = 1024
        vb.cpus   = 1
      end

      node.vm.provider "libvirt" do |lv|
        lv.memory = 1024
        lv.cpus   = 1
      end

      node.vm.provider "parallels" do |pr|
        pr.name   = "ubuntu26-#{name}"
        pr.memory = 1024
        pr.cpus   = 1
      end

      node.vm.provider "qemu" do |q|
        # The vagrant-qemu plugin drives its own SSH hostfwd and ignores the
        # Vagrantfile's forwarded_port host for id: "ssh"; point it at our
        # desired host port explicitly.
        q.ssh_port = host_port
        q.memory   = 1024
        q.cpus     = 1
      end

      node.vm.provision "shell", inline: PROVISION_SCRIPT
    end
  end

  # Banner so the user can see what arch was detected.
  puts "-> Vagrantfile: host arch=#{HOST_ARCH}, box=#{BOX} (#{BOX_ARCH})"
end
