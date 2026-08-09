# -*- mode: ruby -*-
# vi: set ft=ruby :
#
# 4 x Ubuntu 26.04 nodes, SSH forwarded on host ports 12221-12224.
#
# Each node also gets a second NIC on a static private network
# (192.168.56.11-14) so the nodes can talk to each other at
# deterministic addresses.  /etc/hosts entries for all four nodes are
# written by the provision script (node1 .. node4).
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
  "node1" => { ssh_port: 12221, private_ip: "192.168.56.11" },
  "node2" => { ssh_port: 12222, private_ip: "192.168.56.12" },
  "node3" => { ssh_port: 12223, private_ip: "192.168.56.13" },
  "node4" => { ssh_port: 12224, private_ip: "192.168.56.14" },
}

# The bento/ubuntu-26.04 box is UEFI-only: GPT disk with an EFI System
# Partition and no BIOS boot partition.  libvirt must boot it with OVMF
# (or AAVMF on arm64) instead of the default SeaBIOS, otherwise the
# firmware can't find a bootloader and the vCPU spins at 100% forever.
# Search the common install paths across distros.
UEFI_CODE_CANDIDATES = {
  "amd64" => %w[
    /usr/share/edk2/x64/OVMF_CODE.4m.fd
    /usr/share/edk2/ovmf/OVMF_CODE.fd
    /usr/share/OVMF/OVMF_CODE.fd
    /usr/share/OVMF/OVMF_CODE_4M.fd
  ],
  "arm64" => %w[
    /usr/share/AAVMF/AAVMF_CODE.fd
    /usr/share/AAVMF/AAVMF_CODE_4M.fd
    /usr/share/edk2/aarch64/QEMU_EFI.fd
    /usr/share/edk2/aarch64/QEMU_EFI_4M.fd
  ],
}
UEFI_CODE = (UEFI_CODE_CANDIDATES[HOST_ARCH] || []).find { |p| File.exist?(p) }

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

  # --- deterministic inter-node name resolution ---------------------------
  # Point our own hostname at our private-network IP (bento maps it to
  # 127.0.1.1, which breaks anything that advertises its own hostname),
  # then add /etc/hosts entries for every other node.
  idx="$(hostname | sed 's/^node//')"
  sed -i "s/^127\\.0\\.1\\.1 .*/192.168.56.1${idx} $(hostname)/" /etc/hosts
  for i in 1 2 3 4; do
    if [ "$i" != "$idx" ]; then
      grep -qE "[[:space:]]node${i}$" /etc/hosts ||
        echo "192.168.56.1${i} node${i}" >> /etc/hosts
    fi
  done
SHELL

Vagrant.configure("2") do |config|
  config.vm.box = BOX
  config.vm.box_check_update = false

  # We don't need the project dir shared into the guest; disabling it avoids
  # the vagrant-qemu plugin's interactive SMB credential prompt on macOS.
  config.vm.synced_folder ".", "/vagrant", disabled: true

  NODES.each do |name, cfg|
    config.vm.define name do |node|
      node.vm.hostname = name

      # Override the default "ssh" forwarded port so `vagrant ssh` and the
      # test harness both use 1222x on the host side.
      node.vm.network "forwarded_port",
                      guest: 22,
                      host:  cfg[:ssh_port],
                      id:    "ssh",
                      auto_correct: false

      # Static private network so the nodes can talk to each other at
      # deterministic IPs (works on libvirt, virtualbox and parallels; the
      # qemu provider only supports forwarded ports).
      node.vm.network "private_network", ip: cfg[:private_ip]

      node.vm.provider "virtualbox" do |vb|
        vb.name   = "ubuntu26-#{name}"
        vb.memory = 1024
        vb.cpus   = 1
      end

      node.vm.provider "libvirt" do |lv|
        # Force KVM when the host has it; the bento box ships `driver = "qemu"`
        # in its own Vagrantfile so it runs in CI/containers without /dev/kvm.
        # On a real Linux host that drops us into TCG (pure software emulation),
        # pinning every QEMU process at 100% and making the boot never finish.
        # Override it here so KVM is used wherever /dev/kvm is available.
        if File.exist?("/dev/kvm")
          lv.driver = "kvm"
        end
        # Boot via UEFI (OVMF/AAVMF) — see UEFI_CODE above.  libvirt
        # auto-creates the per-domain NVRAM store from the firmware
        # metadata's nvram-template on first start.
        if UEFI_CODE
          lv.loader = UEFI_CODE
          lv.nvram  = "/var/lib/libvirt/qemu/nvram/#{name}_VARS.fd"
        end
        # vagrant-libvirt reaches the guest via its 192.168.121.x IP for
        # `vagrant ssh`, so it skips forwarding the `id: "ssh"` port by
        # default (see forward_ports.rb).  That breaks the localhost:1222x
        # contract the rest of the harness (and macOS/other providers)
        # rely on.  Opt back in so libvirt matches the localhost+port model.
        lv.forward_ssh_port = true
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
        q.ssh_port = cfg[:ssh_port]
        q.memory   = 1024
        q.cpus     = 1
      end

      node.vm.provision "shell", inline: PROVISION_SCRIPT
    end
  end

  # Banner so the user can see what arch was detected.
  puts "-> Vagrantfile: host arch=#{HOST_ARCH}, box=#{BOX} (#{BOX_ARCH})"
end
