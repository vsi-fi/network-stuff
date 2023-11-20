# Access port configurations and ESI #

This lab has two types of access interfaces:

* Single homed, untagged
* Dual homed, untagged, active-active

## Single homed access port ##

Single homed access port is configured similarly to classic approach:

    root@SITE-1-SERVICE-1-A> show configuration interfaces ge-0/0/9  
    unit 0 {
        family ethernet-switching {
            vlan {
                members VLAN100;
            }
        }
    }

We take a physical port ge-0/0/0 and configure it as an access port on VLAN100.

VLAN100 is defined under the mac-vrf routing instance:

    root@SITE-1-SERVICE-1-A> show configuration routing-instances SITE-1-L2 vlans
    VLAN100 {
        vlan-id 100;
        l3-interface irb.100;
        ##
        ## Warning: requires 'vxlan' license
        ##
        vxlan {
            vni 100100;
        }
    }

Notice that in addition to different place in the config hierarchy we're also defining the vxlan vni.

The L3 interface, irb.100, is also defined being part of a VRF but the addressing etc. is still done in the usual place:

    root@SITE-1-SERVICE-1-A> show configuration routing-instances SITE-1-SERVICE-1
    instance-type vrf;
    protocols {
        evpn {
            ip-prefix-routes {
                advertise direct-nexthop;
                encapsulation vxlan;
                vni 3100;
            }
        }
    }
    interface irb.100;
    route-distinguisher 10.1.1.3:3100;
    vrf-import VRF-IMPORT-SITE-1-SERVICE-1;
    vrf-target target:65001:3100;

As to the irb.100 itself:

    root@SITE-1-SERVICE-1-A> show configuration interfaces irb.100                
    virtual-gateway-accept-data;
    family inet {
        address 192.168.100.11/24 {
            virtual-gateway-address 192.168.100.1;
        }
    }

Here the only interesting bits are:

* **virtual-gateway-address**: This is the next-hop for the device connected to the 192.168.100.0/24 network on this device.
    * This address could be the same on all the devices producing access services on the said instance.
    * It could be considered bit similar in concept to VRRP address except that it is active on all the devices at the same time.
* **virtual-gateway-accept-data**: Means that the device will process packets destined to the virtual address.
    * This can be important if the address should respond to icmp pings etc.

All in all, single-homed port is not that spectacularly different to the classic approach.

## Active-active multi homing ##

EVPN has a concept of Ethernet segment identifier (ESI) that can be exploited for active-active multi homing.
ESI is a 10 octet integer value that identifies the said Ethernet segment.

ESI value can be auto derived and could look something like this: 01:00:00:00:00:00:01:00:0a:00
ESI needs to match on all the interfaces taking part in the aggregate.

Physical interface itself can be configured quite similarly as a single homed one:

    root@SITE-1-LEAF-1-B> show configuration interfaces ae9 
    apply-groups ESI;
    unit 0 {
        family ethernet-switching {
            vlan {
                members VLAN10;
            }
        }
    }

Trickery takes place in the ESI apply-group:

    root@SITE-1-LEAF-1-B> show configuration groups ESI 
    interfaces {
        <*> {
            esi {
                auto-derive {
                    lacp-pe-system-id-and-admin-key;
                }
                all-active;
            }
            aggregated-ether-options {
                lacp {
                    active;
                    periodic fast;
                    system-id 00:00:00:00:00:01;
                }
            }
        }
    }

Things worth pointing out:

* **lacp-pe-system-id-and-admin-key** will derive the ESI based on the LACP attributes system-id **00:00:00:00:00:01** and admin key.
* **all-active** pretty much what you'd expect, all interfaces are active. To my knowledge, with MPLS it is possible to have active-passive as well, but not with the setup in this lab.
* **periodic fast** Tells LACP to sent/receive pdus once per second. This acts as a keep alive as well.
* **sytem-id**: This has to match amongst all the participating network devices as this is component to what allows the system to make the downstream device think it is talking to a single device.


### Verifying active-active multi homing ###

    root@SITE-1-LEAF-1-B> show lacp interfaces ae9 extensive 
    Aggregated interface: ae9
    LACP state:       Role   Exp   Def  Dist  Col  Syn  Aggr  Timeout  Activity
      ge-0/0/9       Actor    No    No   Yes  Yes  Yes   Yes     Fast    Active
      ge-0/0/9     Partner    No    No   Yes  Yes  Yes   Yes     Slow    Active
    LACP protocol:        Receive State  Transmit State          Mux State 
      ge-0/0/9                  Current   Slow periodic Collecting distributing
    LACP info:        Role     System             System       Port     Port    Port 
                             priority         identifier   priority   number     key 
      ge-0/0/9       Actor        127  00:00:00:00:00:01        127        1      10
      ge-0/0/9     Partner      65535  0c:ea:18:b8:00:01        255        2       9

As we can see, the ESI multi homed interface takes the same show command as the classical one.
Note the configured system id being visible.

You can view the auto-derived ESI value:

    root@SITE-1-LEAF-1-B> show interfaces ae9 |grep Segment 
      Ethernet segment value: 01:00:00:00:00:00:01:00:0a:00, Mode: all-active

You can see what other devices are taking part in the same ESI:

    root@SITE-1-LEAF-1-A> show route table bgp.evpn.0 evpn-esi-value 01:00:00:00:00:00:01:00:0a:00 brief
    ...
    1:10.1.10.2:100::01000000000001000a00::0/192 AD/EVI        
                   *[BGP/170] 04:02:44, localpref 100
                      AS path: I, validation-state: unverified
                    >  to 10.1.1.1 via ge-0/0/0.0
                       to 10.1.1.2 via ge-0/0/1.0
                    [BGP/170] 04:03:59, localpref 100, from 10.1.1.2
                      AS path: I, validation-state: unverified
                    >  to 10.1.1.1 via ge-0/0/0.0
                       to 10.1.1.2 via ge-0/0/1.0

Here we can see that LEAF-1-B (based on the 10.1.10.2) is also actively taking part in this.

Mac addresses are still learned also locally as you might expect.

### Few words about host-side config ###

In this lab I've used Alpine Linux on which the LACP trunk is configured as follows:

File: /etc/network/interfaces

    auto bond0
    iface bond0 inet static
            address 192.168.10.254
            netmask 255.255.255.0
            gateway 192.168.10.1
            bond-slaves eth1 eth2
            bond-mode 802.3ad

The above config basically tells the Kernel to use 802.3ad trunking on physical interfaces eth1 and eth2.
IP addressing etc. is configured on the resulting bond0 interface.

**To verify**  how this looks from the host perspective:

// Notice the Partner Mac address, system mac address as those match to the above Juniper config

    TENANT-1-ESI:~# cat /proc/net/bonding/bond0
    Ethernet Channel Bonding Driver: v6.1.53-0-lts
 
    Bonding Mode: IEEE 802.3ad Dynamic link aggregation
    Transmit Hash Policy: layer2 (0)
    MII Status: up
    MII Polling Interval (ms): 100
    Up Delay (ms): 0
    Down Delay (ms): 0
    Peer Notification Delay (ms): 0
 
    802.3ad info
    LACP active: on
    LACP rate: slow
    Min links: 0
    Aggregator selection policy (ad_select): stable
    System priority: 65535
    System MAC address: 0c:ea:18:b8:00:01
    Active Aggregator Info:
            Aggregator ID: 2
            Number of ports: 2
            Actor Key: 9
            Partner Key: 10
            Partner Mac Address: 00:00:00:00:00:01
 
    Slave Interface: eth1
    MII Status: up
    Speed: 1000 Mbps
    Duplex: full
    Link Failure Count: 7
    Permanent HW addr: 0c:ea:18:b8:00:01
    Slave queue ID: 0
    Aggregator ID: 2
    Actor Churn State: none
    Partner Churn State: none
    Actor Churned Count: 1
    Partner Churned Count: 1
    details actor lacp pdu:
        system priority: 65535
        system mac address: 0c:ea:18:b8:00:01
        port key: 9
        port priority: 255
        port number: 1
        port state: 61
    details partner lacp pdu:
        system priority: 127
        system mac address: 00:00:00:00:00:01
        oper key: 10
        port priority: 127
        port number: 1
        port state: 63
 
    Slave Interface: eth2
    MII Status: up
    Speed: 1000 Mbps
    Duplex: full
    Link Failure Count: 2
    Permanent HW addr: 0c:ea:18:b8:00:02
    Slave queue ID: 0
    Aggregator ID: 2
    Actor Churn State: none
    Partner Churn State: none
    Actor Churned Count: 0
    Partner Churned Count: 1
    details actor lacp pdu:
        system priority: 65535
        system mac address: 0c:ea:18:b8:00:01
        port key: 9
        port priority: 255
        port number: 2
        port state: 61
    details partner lacp pdu:
        system priority: 127
        system mac address: 00:00:00:00:00:01
        oper key: 10
        port priority: 127
        port number: 1
        port state: 63
    TENANT-1-ESI:~#


