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

