# !!! Nested KVM (i.e. Nested VMX feature) is not enabled, shutting down!!! #
I got the above error message while trying to start vrnetlab/juniper_vjunos-router:25.4R1.12 on a PC with AMD CPU.
Said log entry came up as i tried to understand why my instance went into "exited" state in containerlab

```
sudo containerlab inspect -t test-topology.yml
```
What follows is the amazing journey how to overcome that error.

## Root cause - vrouter appears to expect intel CPU ##
First I checked whether my CPU support nested virtualisation as this is vRouter image wants to run the PFE as a nested VM

```
cat /sys/module/kvm_amd/parameters/nested
1
```
Ok, that's good - so what gives?

As the log entry was clearly thrown by something inside the vrouter image i figured i might as well take a look..

### Mounting the image for investigation ###
```
sudo modprobe nbd max_part=8
sudo qemu-nbd --connect=/dev/nbd0 vJunos-router-25.4R1.12.qcow2
sudo mount /dev/nbd0p2 /mnt
```

Above will make an attempt to expose the file as a block device (/dev/nbd0), similar result could likely be achieved with loop.

### Finding the culprit ###
I was both lazy and stupid so I looked around for the cause like this:

```
cd /mnt
grep -iR "Nested KVM (i.e. Nested VMX feature)" *  2>/dev/null
home/pfe/junos/start-junos.sh:    echo "!!! Nested KVM (i.e. Nested VMX feature) is not enabled, shutting down!!!"  | tee -a "${QEMU_MONITOR_LOG}"
```

### The fix? ###
Modified the file home/pfe/junos/start-junos.sh as follows:
```
#CPU_FLAG=$(cat /proc/cpuinfo | grep -ci vmx) 
CPU_FLAG=$(cat /proc/cpuinfo | grep -Eci "vmx|svm")

if [ "${CPU_FLAG}" -gt 0 ] ; then
    CACHE_MODE="writeback"
else
    echo "!!! Nested KVM (i.e. Nested VMX feature) is not enabled, shutting down!!!"  | tee -a "${QEMU_MONITOR_LOG}"
    /sbin/shutdown -h now
    exit 0
fi
```

Modification simply tells the thing to accept svm in addition to vmx.
Unmount the partition and disconnect

```
umount /mnt
qemu-nbd --disconnect /dev/nbd0
```
Now the image is "prepared" and only thing left to do is to rebuild the docker image.
```
cd ~/vrnetlab/juniper/vjunosrouter
make
```

After the above I was able to start the instances and eventually also pfe came up with the following containerlab file

```
name: test-lab
mgmt:
  bridge: virbr0
  ipv4-subnet: 192.168.122.0/24
topology:
  kinds:
    juniper_vjunosrouter:
      image: vrnetlab/juniper_vjunos-router:25.4R1.12
  nodes:
    spine1:
            kind: juniper_vjunosrouter
            mgmt-ipv4: 192.168.122.101
    spine2:
            kind: juniper_vjunosrouter
            mgmt-ipv4: 192.168.122.102
    leaf1:
            kind: juniper_vjunosrouter
            mgmt-ipv4: 192.168.122.11
    leaf2:
            kind: juniper_vjunosrouter
            mgmt-ipv4: 192.168.122.12
  links:
          - endpoints: ["leaf1:eth1", "spine1:eth1"]
          - endpoints: ["leaf1:eth2", "spine2:eth1"]
          - endpoints: ["leaf2:eth1", "spine1:eth2"]
          - endpoints: ["leaf2:eth2", "spine2:eth2"]
```

### Things to keep in mind ###
I have no idea why Juniper thinks I deserved no vrouter on AMD, maybe there is some technical reason for this which will screw up lab?
For sure, this procedure has to be done every time there is a new version of vrouter, which is both annoying and frustrating.
