# Trivial MPLS network #

Idea with this lab is to build a simple MPLS network in order to test different EVPN data centre interconnection options across a service provider -type network.

In order to keep things simple I opted for the below implementation specs for the SP network:

* Unnumered IS-IS as IGP
* IBGP with route-reflector on one of the P routers
* LDP instead of RSVP for initial setup
* L2vpn interconnection between sites
* L3vpn interconnection between sites
* Possibly service vrf inside SP network that could be made available in the L3 DCI vrf

## Building blocks ##

I used the below "equipment":

* vmx 21.4R1.12 for all SP gear

## Configuration ##

### P routers ###

Starting with the interface configuration:

```
root@ISP-2-P> show configuration interfaces 
ge-0/0/0 {
    description "Connection to ISP-1-P";
    mtu 9500;
    unit 0 {
        family inet {
            unnumbered-address lo0.0;
        }
        family iso;
        family mpls;
    }
}
ge-0/0/1 {
    description "Connection to ISP-3-P";
    mtu 9500;
    unit 0 {
        family inet {
            unnumbered-address lo0.0;
        }
        family iso;
        family mpls;
    }
}
ge-0/0/9 {
    description "Connection to ISP-PE-2";
    mtu 9500;
    unit 0 {
        family inet {
            unnumbered-address lo0.0;
        }
        family iso;
        family mpls;
    }
}
lo0 {
    unit 0 {
        family inet {
            address 10.0.0.2/32;
        }
        family iso {
            address 49.0002.0010.0000.0000.0002.00;
        }
    }
}

```

In the above config I've enabled the required protocols on all of the SP gear facing interfaces.

Next, we have the required protocols section

```
root@ISP-2-P> show configuration protocols 
router-advertisement {
    interface fxp0.0;
}
isis {
    interface ge-0/0/0.0 {
        point-to-point;
    }
    interface ge-0/0/1.0 {
        point-to-point;
    }
    interface ge-0/0/9.0 {
        point-to-point;
    }
    interface lo0.0 {
        passive;
    }
}
ldp {
    interface ge-0/0/0.0;
    interface ge-0/0/1.0;
    interface ge-0/0/9.0;
}
mpls {
    interface ge-0/0/0.0;               
    interface ge-0/0/1.0;
    interface ge-0/0/9.0;
}
lldp {
    interface all;
}

```

I opted not to run BGP on the P routers except obviously on the route-reflector. More on this later.

*Notice* that mpls has to be enabled on both interface and protocol level.

In this sort of setup the P router config is pretty much there, not much to configure really.

#### Route reflector ####

Route reflector device has similar concept as the other P routers except that it has BGP configured:

```
root@ISP-4-P> show configuration protocols bgp   
group IBGP {
    type internal;
    passive;
    family inet {
        unicast;
    }
    family inet-vpn {
        unicast;
    }
    family l2vpn {
        signaling;
    }
    cluster 10.0.0.4;
    local-as 650000;
    allow [ 10.0.0.0/24 10.1.0.0/24 ];
}

```

Here, we add the required address-families and the route-reflector -enabling-statement "cluster".

I decided to accept connections from both the P and PE routers and didn't bother to explicitly define the neighbors, hence the "allow" and "passive" statements.

### PE routers ###

There is a bit more stuff to configure on the PE routers as this is where the "magic" of MPLS is sort of put to use:

* customer facing interfaces
* vrf instances

Again, starting with the interface config:

```
root@ISP-PE-1> show configuration interfaces 
ge-0/0/0 {
    description "Connection to ISP-1-P";
    mtu 9500;
    unit 0 {
        family inet {
            unnumbered-address lo0.0;
        }
        family iso;
        family mpls;
    }
}
ge-0/0/9 {
    description "Customer site 1";
    encapsulation ethernet-ccc;
    unit 0 {
        family ccc;
    }
}
lo0 {
    unit 0 {
        family inet {
            address 10.1.0.1/32;
        }
        family iso {                    
            address 49.0002.0010.0001.0000.0001.00;
        }
    }
    unit 100 {
        family inet {
            address 192.168.255.1/32;
        }
    }
}
```
Here, I am also running IS-IS and MPLS on the core facing interface ge-0/0/0. However, customer facing interface ge-0/0/9 is bit more interesting:

```
ge-0/0/9 {
    description "Customer site 1";
    encapsulation ethernet-ccc;
    unit 0 {
        family ccc;
    }
}
```
I've set the encapsulation to ethernet-ccc, this indicates the the said interface has to deal with untagged ethernet.
If vlan tagging were required I likely would have used vlan-ccc and for more exotic things maybe extended-vlan-ccc.

The family ccc stands for circuit cross-connect which is also what I've set for the L2vpn routing instance which allows for complete separation of SP and customer networks at the protocol level.
To put all of the above to other words: ge-0/0/9 is one end of a pipe that blindly transports ethernet traffic and does not care what is going on inside the said pipe.

Loopback lo0.100 was used to initially test the L3vpn. 

Next we have the protocols section:

```
root@ISP-PE-1> show configuration protocols 
router-advertisement {
    interface fxp0.0;
}
bgp {
    group IBGP {
        type internal;
        family inet {
            unicast;
        }
        family inet-vpn {
            unicast;
        }
        family l2vpn {
            signaling;
        }
        local-as 650000;
        neighbor 10.0.0.4;
    }
}
isis {
    interface ge-0/0/0.0 {
        point-to-point;
    }
    interface lo0.0 {                   
        passive;
    }
}
ldp {
    interface ge-0/0/0.0;
}
mpls {
    interface ge-0/0/0.0;
}
lldp {
    interface all;
}

root@ISP-PE-1> 
```

Here I am telling the PE to talk BGP to our route-reflector 10.0.0.4 and we've also defined the required address-families.
Similarly to P routers, isis, ldp and mpls are configured on the core facing interfaces.

Next, we have the routing instances and few policies and policy components:

#### Routing-instances: L3vpn ####

```
root@ISP-PE-1> show configuration routing-instances 
DC-CLIENT-100 {
    instance-type vrf;
    interface lo0.100;
    route-distinguisher 10.1.0.1:100;
    vrf-import VRF-IMPORT-DC-CLIENT-100;
    vrf-export VRF-EXPORT-DC-CLIENT-100;
}
DC-CLIENT-100-L2-A {
    instance-type l2vpn;
    protocols {
        l2vpn {
            site CE-1 {
                interface ge-0/0/9.0 {
                    remote-site-id 2;
                }
                site-identifier 1;
            }
            encapsulation-type ethernet;
        }
    }
    interface ge-0/0/9.0;
    route-distinguisher 10.1.0.1:1002;
    vrf-import VRF-IMPORT-DC-CLIENT-100-L2-A;
    vrf-export VRF-EXPORT-DC-CLIENT-100-L2-A;
}
```

Starting with the L3vpn DC-CLIENT-100, I've included only one interface, lo0.100 for now but customer facing L3 interfaces would need to be similarly included in the routing-instance.
I set the route-distinguisher(RD) to match to the ipv4 address on the loopback and imagined a customer-id of 100 to complete the RD.

RD is used to keep the possibly overlapping prefixes separate inside the SP network. Same concept is also used in EVPN control plane for the exact same purpose.
Next, I've opted to use specific vrf import/export policies:

```
show configuration policy-options policy-statement VRF-IMPORT-DC-CLIENT-100       
term DC-CLIENT-100 {
    from community DC-CLIENT-100;
    then accept;
}
term DEFAULT {
    then reject;
}

show configuration policy-options policy-statement VRF-EXPORT-DC-CLIENT-100    
term DC-CLIENT-100 {
    then {
        community add DC-CLIENT-100;
        accept;
    }
}

show configuration policy-options community DC-CLIENT-100   
members target:650000L:100;

root@ISP-PE-1> 
 
```
What these policies do is that they cause the VRF to import all routes that match on a route-target(RT) target:650000L:100 (the L is required for long AS numbers).
Export policy slaps the said extended community to all routes exported to BGP so that other PE routers with the same VRF can import the said routes.

That sums it up for the L3vpn for now.

#### Routing-instances: L2vpn ####

```
root@ISP-PE-1> show configuration routing-instances DC-CLIENT-100-L2-A    
instance-type l2vpn;
protocols {
    l2vpn {
        site CE-1 {
            interface ge-0/0/9.0 {
                remote-site-id 2;
            }
            site-identifier 1;
        }
        encapsulation-type ethernet;
    }
}
interface ge-0/0/9.0;
route-distinguisher 10.1.0.1:1002;
vrf-import VRF-IMPORT-DC-CLIENT-100-L2-A;
vrf-export VRF-EXPORT-DC-CLIENT-100-L2-A;

root@ISP-PE-1> 
```
The only real differences here to the L3vpn are the instance-type l2vpn and the protocols section.
We need to define the "sites" for our site-to-site pipe and the interface of whose other end is on the other "site" and the encapsulation.
ISP-PE-1 is on "site" CE-1 and ISP-PE-2 is on "site" CE-2, both have their ge-0/0/9 facing the customer.

Policy elements are basically identical to the L3vpn except we have a different RT:
```
root@ISP-PE-1> show configuration policy-options                    
...
policy-statement VRF-EXPORT-DC-CLIENT-100-L2-A {
    then {
        community add DC-CLIENT-100-L2-A;
        accept;
    }
}
...
policy-statement VRF-IMPORT-DC-CLIENT-100-L2-A {
    term DC-CLIENT-100-L2-A {
        from community DC-CLIENT-100-L2-A;
        then accept;
    }
    term DEFAULT {
        then reject;
    }
}
...
community DC-CLIENT-100-L2-A members target:650000L:1002;
```

### Verification ###

To test the L3vpn I can try to check the routes and ping the loopbacks of the lo0.100 interfaces amongst the L3vpn instances:

```
root@ISP-PE-2> show configuration interfaces lo0.100 
family inet {
    address 192.168.255.2/32;
}

root@ISP-PE-2> 

root@ISP-PE-2> show route table DC-CLIENT-100 192.168.255.1 

DC-CLIENT-100.inet.0: 2 destinations, 2 routes (2 active, 0 holddown, 0 hidden)
+ = Active Route, - = Last Active, * = Both

192.168.255.1/32   *[BGP/170] 01:01:18, localpref 100, from 10.0.0.4
                      AS path: I, validation-state: unverified
                    >  to 10.0.0.2 via ge-0/0/0.0, Push 339408, Push 349136(top)

root@ISP-PE-2> 

root@ISP-PE-2> show route table DC-CLIENT-100 192.168.255.1 detail | grep "label|next op|Distin|Source"
                Route Distinguisher: 10.1.0.1:100
                Source: 10.0.0.4
                Label operation: Push 339408, Push 349136(top)
                Label TTL action: prop-ttl, prop-ttl(top)
                Load balance label: Label 339408: None; Label 349136: None; 
                Label element ptr: 0x7bf5970
                Label parent element ptr: 0x7bf52e0
                Label element references: 1
                Label element child references: 0
                Label element lsp id: 0
                Label operation: Push 339408
                Label TTL action: prop-ttl
                Load balance label: Label 339408: None; 
                VPN Label: 339408
```
So it would seem that We've learned the route to the other loopback on the same L3vpn, ping should maybe work:

```
root@ISP-PE-2> ping 192.168.255.1 source 192.168.255.2 routing-instance DC-CLIENT-100 count 1
PING 192.168.255.1 (192.168.255.1): 56 data bytes
64 bytes from 192.168.255.1: icmp_seq=0 ttl=64 time=3.951 ms

--- 192.168.255.1 ping statistics ---
1 packets transmitted, 1 packets received, 0% packet loss
round-trip min/avg/max/stddev = 3.951/3.951/3.951/0.000 ms

```

Seems to work as expected.

It might be worth mentioning that the since the said VRF is only configured on the PEs 1 and 2, the rest of the routers are completely oblivious to the said routes, with the exception of the route-reflector ofcourse.

To verify the L2vpn we can also look at the routing table:

```
root@ISP-PE-1> show route ccc ge-0/0/9.0 detail 

mpls.0: 17 destinations, 17 routes (17 active, 0 holddown, 0 hidden)
ge-0/0/9.0 (1 entry, 1 announced)
        *L2VPN  Preference: 7
                Next hop type: Indirect, Next hop index: 0
                Address: 0x7aaee6c
                Next-hop reference count: 2
                Next hop type: Router, Next hop index: 612
                Next hop: 10.0.0.1 via ge-0/0/0.0, selected
                Label operation: Push 800000, Push 363216(top) Offset: 252
                Label TTL action: no-prop-ttl, no-prop-ttl(top)
                Load balance label: Label 800000: None; Label 363216: None; 
                Label element ptr: 0x7bf58f8
                Label parent element ptr: 0x7bf5538
                Label element references: 1
                Label element child references: 0
                Label element lsp id: 0
                Session Id: 141
                Protocol next hop: 10.1.0.2
                Label operation: Push 800000 Offset: 252
                Label TTL action: no-prop-ttl
                Load balance label: Label 800000: None; 
                Indirect next hop: 0x712995c 1048575 INH Session ID: 330
                State: <Active Int>     
                Age: 1:11:01    Metric2: 1 
                Validation State: unverified 
                Task: Common L2 VC
                Announcement bits (2): 1-KRT 2-Common L2 VC 
                AS path: I 
                Communities: target:650000L:1002 Layer2-info: encaps: ETHERNET, control flags:[0x2] Control-Word, mtu: 0, site preference: 100
                Thread: junos-main 
```

From the above we can see for instance the protocol next-hop of 10.1.0.2 that is the loopback of the ISP-PE-2. This is reachable via next hop 10.0.0.1.


